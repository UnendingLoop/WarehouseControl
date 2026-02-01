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

	"github.com/UnendingLoop/WarehouseControl/internal/mwauthlog"
	"github.com/UnendingLoop/WarehouseControl/internal/repository"
	"github.com/UnendingLoop/WarehouseControl/internal/service"
	"github.com/UnendingLoop/WarehouseControl/internal/transport"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/dbpg"

	"github.com/wb-go/wbf/ginext"
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
	handlers := transport.NewEBHandlers(svc)
	// конфиг сервера
	mode := appConfig.GetString("GIN_MODE")
	engine := ginext.New(mode)
	engine.Use(mwauthlog.RequestID()) // вставка уникального UID в каждый реквест
	engine.GET("/ping", handlers.SimplePinger)
	engine.Static("/ui", "./internal/web") // UI админа/юзера - функциональность и контент зависит от роли

	auth := engine.Group("/auth")
	auth.POST("/signup", handlers.SignUpUser) // регистрация пользователя
	auth.POST("/login", handlers.LoginUser)   // авторизация

	items := engine.Group("/items", mwauthlog.RequireAuth([]byte(appConfig.GetString("SECRET"))))
	items.POST("", handlers.CreateItem)                    // создание Item
	items.PATCH("/:id", handlers.GetItemByID)              // обновление Item по ID
	items.GET("/:id", handlers.GetItemByID)                // получение Item по ID
	items.GET("/:id/history", handlers.GetItemHistoryByID) // получение History товара по его ID
	items.DELETE("/:id", handlers.DeleteItem)              // удаление Item по ID
	items.GET("", handlers.GetItemsList)                   // получение всех Item

	items.GET("/history", handlers.GetItemsHistoryList) // получение History всех товаров

	items.GET("/csv", handlers.ExportItemsCSV)                     // CSV: получение всех Item
	items.GET("/:id/history/csv", handlers.ExportItemIDHistoryCSV) // CSV: получение History товара по его ID
	items.GET("/history/csv", handlers.ExportItemsHistory)         // CSV: получение History всех товаров

	srv := &http.Server{
		Addr:    ":" + appConfig.GetString("APP_PORT"),
		Handler: engine,
	}

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
