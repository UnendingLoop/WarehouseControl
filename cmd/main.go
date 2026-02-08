package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/UnendingLoop/WarehouseControl/internal/engine"
	"github.com/UnendingLoop/WarehouseControl/internal/mwauthlog"
	"github.com/UnendingLoop/WarehouseControl/internal/repository"
	"github.com/UnendingLoop/WarehouseControl/internal/service"
	"github.com/UnendingLoop/WarehouseControl/internal/transport"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/dbpg"
)

func main() {
	log.Println("Starting WarehouseControl application...")
	// инициализировать конфиг/ считать энвы
	appConfig := config.New()
	appConfig.EnableEnv("")
	if err := appConfig.LoadEnvFiles("./.env"); err != nil {
		log.Fatalf("Failed to load envs: %s\nExiting app...", err)
	}

	// готовим заранее слушатель прерываний - контекст для всего приложения
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// подключитсья к базе
	dbConn := repository.ConnectWithRetries(appConfig, 5, 10*time.Second)
	// накатываем миграцию
	repository.MigrateWithRetries(dbConn.Master, "./migrations", 10, 15*time.Second)

	// repo
	repo := repository.NewPostgresImageRepo(dbConn)
	// jwt
	jwtMngr := mwauthlog.NewJWTManager([]byte(appConfig.GetString("SECRET")), time.Hour, "WarehouseControl app")
	// service
	svc := service.NewWHBService(repo, jwtMngr)
	// handlers
	handlers := transport.NewWHCHandlers(svc)
	// конфиг сервера
	srv, _ := engine.NewServerEngine(appConfig, handlers, "PROD")

	// запуск сервера
	go func() {
		log.Printf("Server running on http://localhost%s\n", srv.Addr)
		err := srv.ListenAndServe()
		if err != nil {
			switch {
			case errors.Is(err, http.ErrServerClosed):
				log.Println("Server gracefully stopping...")
			default:
				log.Printf("Server stopped: %v", err)
				stop()
			}
		}
	}()

	// слушаем контекст прерываний для запуска Graceful Shutdown
	<-ctx.Done()
	shutdown(dbConn, srv)
}

func shutdown(dbConn *dbpg.DB, srv *http.Server) {
	log.Println("Interrupt received! Starting shutdown sequence...")

	// Closing Server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("Failed to shutdown server correctly:", err)
	} else {
		log.Println("Server is closed.")
	}

	// Closing DB connection
	if err := dbConn.Master.Close(); err != nil {
		log.Println("Failed to close DB-conn correctly:", err)
	} else {
		log.Println("DBconn is closed.")
	}
}
