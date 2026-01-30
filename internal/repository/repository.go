// Package repository declares contract to work with DB, also provides migration and connect-with-retry methods to DB.
package repository

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"
	"time"

	"github.com/UnendingLoop/EventBooker/internal/model"
	"github.com/UnendingLoop/EventBooker/internal/repository/ebpostgres"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/dbpg"
)

type EBRepo interface {
	CreateEvent(ctx context.Context, exec ebpostgres.Executor, newEvent *model.Event) error // только для админа
	CreateBook(ctx context.Context, exec ebpostgres.Executor, newBook *model.Book) error
	CreateUser(ctx context.Context, exec ebpostgres.Executor, newUser *model.User) error

	DeleteEvent(ctx context.Context, exec ebpostgres.Executor, eventID int) error // только для админа
	DeleteBook(ctx context.Context, exec ebpostgres.Executor, bookID int) error   // эксклюзивно для воркера BookCleaner

	UpdateBookStatus(ctx context.Context, exec ebpostgres.Executor, bookID int, newStatus string) error

	GetEventByID(ctx context.Context, exec ebpostgres.Executor, eventID int) (*model.Event, error)
	GetEventsList(ctx context.Context, exec ebpostgres.Executor, role string) ([]*model.Event, error)
	GetBookByID(ctx context.Context, exec ebpostgres.Executor, bookID int) (*model.Book, error)
	GetBooksListByUser(ctx context.Context, exec ebpostgres.Executor, id int) ([]*model.Book, error)
	GetExpiredBooksList(ctx context.Context, exec ebpostgres.Executor) ([]*model.Book, error)
	GetUserByID(ctx context.Context, exec ebpostgres.Executor, userID int) (*model.User, error)
	GetUserByEmail(ctx context.Context, exec ebpostgres.Executor, email string) (*model.User, error)

	IncrementAvailSeatsByEventID(ctx context.Context, exec ebpostgres.Executor, eventID int) error
	DecrementAvailSeatsByEventID(ctx context.Context, exec ebpostgres.Executor, eventID int) error
}

func NewPostgresImageRepo(dbconn *dbpg.DB) EBRepo {
	return &ebpostgres.PostgresRepo{}
}

func ConnectWithRetries(appConfig *config.Config, retryCount int, idleTime time.Duration) *dbpg.DB {
	dbOptions := dbpg.Options{
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: 10 * time.Minute,
	}

	dbUser := appConfig.GetString("POSTGRES_USER")
	dbName := appConfig.GetString("POSTGRES_DB")
	dbPass := appConfig.GetString("POSTGRES_PASSWORD")
	dbContName := appConfig.GetString("DB_CONTAINER_NAME")
	if dbUser == "" || dbName == "" || dbPass == "" || dbContName == "" {
		log.Fatal("DB connection credentials, db name or DB container name are not set in env")
	}
	dsn := "postgresql://" + dbUser + ":" + dbPass + "@" + dbContName + ":5432/" + dbName + "?sslmode=disable"

	var dbConn *dbpg.DB
	var err error

	for range retryCount {
		dbConn, err = dbpg.New(dsn, nil, &dbOptions)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to PGDB: %s\nWaiting %v before next retry...", err, idleTime)
		time.Sleep(idleTime)
	}

	if err != nil {
		log.Fatal("Failed to connect to DB. Exiting the app...")
	}

	return dbConn
}

func MigrateWithRetries(db *sql.DB, migrationsPath string, retries int, idle time.Duration) {
	for i := range retries {
		log.Printf("Migration try #%d...", i)
		err := migrateOnce(db, migrationsPath)
		if err == nil {
			break
		}
		switch i {
		case retries:
			log.Fatalln("Out of retries. Exiting...")
		default:
			log.Printf("Migration try #%d was unsuccessful: %v\nWaiting %v before next try...", i, err, idle)
			time.Sleep(idle)
		}
	}
}

func migrateOnce(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return err
	}

	sourceURL := "file://" + absPath
	log.Println("Running migrations from:", sourceURL)

	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Println("Database migrations applied successfully")
	return nil
}
