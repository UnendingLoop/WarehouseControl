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

	"github.com/UnendingLoop/EventBooker/internal/cleaner"
	"github.com/UnendingLoop/EventBooker/internal/mwauthlog"
	"github.com/UnendingLoop/EventBooker/internal/repository"
	"github.com/UnendingLoop/EventBooker/internal/service"
	"github.com/UnendingLoop/EventBooker/internal/transport"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/dbpg"

	"github.com/wb-go/wbf/ginext"
)

func main() {
	log.Println("Starting EventBook application...")
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
	jwtMngr := mwauthlog.NewJWTManager([]byte(appConfig.GetString("SECRET")), time.Hour, "EventBook app")
	// service
	svc := service.NewEBService(repo, dbConn, jwtMngr)
	// handlers
	handlers := transport.NewEBHandlers(svc)
	// конфиг сервера
	mode := appConfig.GetString("GIN_MODE")
	engine := ginext.New(mode)
	engine.Use(
		mwauthlog.RequestID()) // вставка уникального UID в каждый реквест

	events := engine.Group("/events", mwauthlog.RequireAuth([]byte(appConfig.GetString("SECRET"))))
	books := engine.Group("/bookings", mwauthlog.RequireAuth([]byte(appConfig.GetString("SECRET"))))
	auth := engine.Group("/auth")

	engine.GET("/ping", handlers.SimplePinger)
	engine.Static("/ui", "./internal/web") // UI админа/юзера - функциональность и контент зависит от роли

	auth.POST("/signup", handlers.SignUpUser) // регистрация пользователя
	auth.POST("/login", handlers.LoginUser)   // авторизация

	events.POST("", mwauthlog.RequireRoles("admin"), handlers.CreateEvent)       // создание ивента - только админ
	events.GET("", handlers.GetEvents)                                           // список всех ивентов
	events.DELETE("/:id", mwauthlog.RequireRoles("admin"), handlers.DeleteEvent) // удаление ивента - только админ

	books.POST("", handlers.BookEvent)               // создание бронирования
	books.POST("/:id/confirm", handlers.ConfirmBook) // подтверждение бронирования
	books.GET("/my", handlers.GetUserBooks)          // все брони по одному пользователю
	books.DELETE("/:id", handlers.CancelBook)        // отмена брони

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

	// cleaner
	clb := cleaner.NewBookCleaner(svc)
	clb.StartBookCleaner(ctx, 30)

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
