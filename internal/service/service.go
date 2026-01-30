// Package service provides all methods for app
package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/UnendingLoop/EventBooker/internal/model"
	"github.com/UnendingLoop/EventBooker/internal/mwauthlog"
	"github.com/UnendingLoop/EventBooker/internal/repository"
	"github.com/wb-go/wbf/dbpg"
	"golang.org/x/crypto/bcrypt"
)

type EBService struct {
	repo       repository.EBRepo
	db         *dbpg.DB
	jwtManager *mwauthlog.JWTManager
}

func NewEBService(ebrepo repository.EBRepo, ebdb *dbpg.DB, jwt *mwauthlog.JWTManager) *EBService {
	return &EBService{repo: ebrepo, db: ebdb, jwtManager: jwt}
}

func (eb EBService) CreateUser(ctx context.Context, user *model.User) (string, error) {
	rid := model.RequestIDFromCtx(ctx)

	if err := validateNormalizeUser(user); err != nil {
		return "", err
	}

	err := eb.repo.CreateUser(ctx, eb.db, user)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "unique_violation"):
			return "", model.ErrUserAlreadyExists
		default:
			log.Printf("RID %q Failed to put new user to DB in 'CreateUser': %q", rid, err)
			return "", model.ErrCommon500
		}
	}

	token, err := eb.jwtManager.Generate(user.ID, user.UserName, user.Role)
	if err != nil {
		return "", model.ErrCommon500
	}

	return token, nil
}

func (eb EBService) LoginUser(ctx context.Context, email string, password string) (string, *model.User, error) {
	rid := model.RequestIDFromCtx(ctx)

	user, err := eb.repo.GetUserByEmail(ctx, eb.db, email)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrUserNotFound):
			return "", nil, err
		default:
			log.Printf("RID %q Failed to get user from DB in 'LoginUser': %q", rid, err)
			return "", nil, model.ErrCommon500
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(password)); err != nil {
		return "", nil, model.ErrInvalidCredentials
	}

	token, err := eb.jwtManager.Generate(user.ID, user.UserName, user.Role)
	if err != nil {
		return "", nil, model.ErrCommon500
	}

	return token, user, nil
}

func (eb EBService) CreateEvent(ctx context.Context, event *model.Event) error {
	rid := model.RequestIDFromCtx(ctx)

	if err := validateNormalizeItem(event); err != nil {
		return err // 400
	}

	if err := eb.repo.CreateEvent(ctx, eb.db, event); err != nil {
		log.Printf("RID %q Failed to create new event in DB in 'CreateEvent': %v", rid, err)
		return model.ErrCommon500
	}

	return nil
}

func (eb EBService) BookEvent(ctx context.Context, book *model.Book) error {
	rid := model.RequestIDFromCtx(ctx)

	if book.EventID <= 0 || book.UserID <= 0 {
		return model.ErrEmptyBookInfo // 400
	}

	book.Status = model.BookStatusCreated

	// транзакция - бегин
	tx, err := eb.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("RID %q Failed to begin transaction in 'BookEvent': %v", rid, err)
		return model.ErrCommon500 // 500
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("RID %q Failed to rollback transaction in 'BookEvent': %v", rid, err)
			}
		}
	}()
	// получаем ивент и проверяем доступность мест
	event, err := eb.repo.GetEventByID(ctx, tx, book.EventID)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrEventNotFound):
			return err
		default:
			log.Printf("RID %q Failed to get event from DB in 'BookEvent': %q", rid, err)
			return model.ErrCommon500
		}
	}

	if event.Status != model.EventStatusActual {
		return model.ErrExpiredEvent // 409
	}
	if event.EventDate.Before(time.Now().UTC()) {
		return model.ErrExpiredEvent // 409
	}
	if event.AvailSeats == 0 {
		return model.ErrNoSeatsAvailable // 409
	}

	deadline := time.Now().UTC().Add(time.Duration(event.BookWindow) * time.Second)
	book.ConfirmDeadline = &deadline

	// создание записи
	if err := eb.repo.CreateBook(ctx, tx, book); err != nil {
		log.Printf("RID %q Failed to create new book in DB in 'BookEvent': %v", rid, err)
		return model.ErrCommon500 // 500
	}

	// декремент event.availSeats
	if err := eb.repo.DecrementAvailSeatsByEventID(ctx, tx, book.EventID); err != nil {
		log.Printf("RID %q Failed to decrement event avail.seats in 'BookEvent': %v", rid, err)
		return model.ErrCommon500
	}
	// коммит транзакции
	if err := tx.Commit(); err != nil {
		log.Printf("RID %q Failed to commit transaction in 'BookEvent': %v", rid, err)
		return model.ErrCommon500
	}

	committed = true

	return nil
}

func (eb EBService) ConfirmBook(ctx context.Context, bid int, uid int) error {
	rid := model.RequestIDFromCtx(ctx)

	if bid < 1 {
		return model.ErrIncorrectBookID
	}
	if uid < 1 {
		return model.ErrIncorrectUserID
	}

	// бегин транзакции
	tx, err := eb.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("RID %q Failed to begin transaction in 'ConfirmBook': %v", rid, err)
		return model.ErrCommon500
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("RID %q Failed to rollback transaction in 'CancelBook': %v", rid, err)
			}
		}
	}()

	// проверяем бронь
	book, err := eb.repo.GetBookByID(ctx, tx, bid)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrBookNotFound):
			return err
		default:
			log.Printf("RID %q Failed to get book from DB in 'ConfirmBook': %q", rid, err)
			return model.ErrCommon500
		}
	}

	if book.UserID != uid {
		return model.ErrAccessDenied
	}
	if book.Status == model.BookStatusCancelled {
		return model.ErrBookIsCancelled
	}
	if book.Status == model.BookStatusConfirmed {
		return model.ErrBookIsConfirmed
	}
	if book.ConfirmDeadline.Before(time.Now().UTC()) {
		return model.ErrExpiredBook
	}

	// апдейтим статус
	if err := eb.repo.UpdateBookStatus(ctx, tx, bid, model.BookStatusConfirmed); err != nil { // добавить обработку 404
		log.Printf("RID %q Failed to confirm book in DB in 'ConfirmBook': %v", rid, err)
		return model.ErrCommon500
	}

	// коммит транзакции
	if err := tx.Commit(); err != nil {
		log.Printf("RID %q Failed to commit transaction in 'ConfirmBook': %v", rid, err)
		return model.ErrCommon500
	}
	committed = true
	return nil
}

func (eb EBService) CancelBook(ctx context.Context, bid int, uid int) error { // не удаляет бронь, а помечает как cancelled и инкрементит availseats
	rid := model.RequestIDFromCtx(ctx)

	if bid < 1 {
		return model.ErrIncorrectBookID
	}
	if uid < 1 {
		return model.ErrIncorrectUserID
	}

	// бегин транзакции
	tx, err := eb.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("RID %q Failed to begin transaction in 'CancelBook': %v", rid, err)
		return model.ErrCommon500
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("RID %q Failed to rollback transaction in 'CancelBook': %v", rid, err)
			}
		}
	}()

	// проверяем бронь
	book, err := eb.repo.GetBookByID(ctx, tx, bid)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrBookNotFound):
			return err
		default:
			log.Printf("RID %q Failed to get book from DB in 'CancelBook': %q", rid, err)
			return model.ErrCommon500
		}
	}
	if book.Status == model.BookStatusCancelled {
		return model.ErrBookIsCancelled
	}

	// получаем ивент чтобы залочить для транзакции
	_, err = eb.repo.GetEventByID(ctx, tx, book.EventID)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrEventNotFound):
			return err
		default:
			log.Printf("RID %q Failed to get event from DB in 'CancelBook': %q", rid, err)
			return model.ErrCommon500
		}
	}

	// отменяем бронь
	if err := eb.repo.UpdateBookStatus(ctx, tx, bid, model.BookStatusCancelled); err != nil {
		log.Printf("RID %q Failed to update book status in DB in 'CancelBook': %v", rid, err)
		return model.ErrCommon500
	}

	// инкрементим event.availSeats
	if err := eb.repo.IncrementAvailSeatsByEventID(ctx, tx, book.EventID); err != nil {
		log.Printf("RID %q Failed to increment event avail.seats in 'CancelBook': %v", rid, err)
		return model.ErrCommon500
	}

	// коммит транзакции
	if err := tx.Commit(); err != nil {
		log.Printf("RID %q Failed to commit transaction in 'CancelBook': %v", rid, err)
		return model.ErrCommon500
	}
	committed = true
	return nil
}

func (eb EBService) DeleteEvent(ctx context.Context, eid int, role string) error { // добавить проверку роли пользователя
	rid := model.RequestIDFromCtx(ctx)

	if role != model.RoleAdmin {
		return model.ErrAccessDenied
	}
	if eid < 1 {
		return model.ErrIncorrectEventID
	}

	// бегин транзакции
	tx, err := eb.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("RID %q Failed to begin transaction in 'DeleteEvent': %v", rid, err)
		return model.ErrCommon500
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("RID %q Failed to rollback transaction in 'DeleteEvent': %v", rid, err)
			}
		}
	}()

	// проверяем данные ивента
	event, err := eb.repo.GetEventByID(ctx, tx, eid)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrEventNotFound):
			return err
		default:
			log.Printf("RID %q Failed to get event from DB in 'DeleteEvent': %v", rid, err)
			return model.ErrCommon500
		}
	}
	if event.TotalSeats != event.AvailSeats {
		return model.ErrEventBusy
	}

	// удаляем ивент
	if err := eb.repo.DeleteEvent(ctx, tx, eid); err != nil {
		log.Printf("RID %q Failed to delete event in DB in 'DeleteEvent': %v", rid, err)
		return model.ErrCommon500
	}

	// коммит транзакции
	if err := tx.Commit(); err != nil {
		log.Printf("RID %q Failed to commit transaction in 'DeleteEvent': %v", rid, err)
		return model.ErrCommon500
	}

	committed = true

	return nil
}

func (eb EBService) CleanExpiredBooks(ctx context.Context) error {
	// транзакция - бегин
	tx, err := eb.db.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Failed to begin transaction:", err)
		return model.ErrCommon500
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("Failed to rollback transaction in 'CleanExpiredBooks': %v", err)
			}
		}
	}()

	// запрос всех подходящих броней
	books, err := eb.repo.GetExpiredBooksList(ctx, tx)
	if err != nil {
		log.Println("Failed to fetch expired books in 'CleanExpiredBooks':", err)
		return model.ErrCommon500
	}
	if len(books) == 0 {
		return nil
	}

	// в цикле проделать декремент ивентов и удаление броней
	for _, b := range books {
		// если статус брони cancelled - availSeats уже инкрементирован
		if b.Status != model.BookStatusCancelled {
			if err := eb.repo.IncrementAvailSeatsByEventID(ctx, tx, b.EventID); err != nil {
				log.Println("Failed to decrement event avail.seats in 'CleanExpiredBooks':", err)
				return model.ErrCommon500
			}
		}

		if err = eb.repo.DeleteBook(ctx, tx, b.ID); err != nil {
			log.Println("Failed to delete expired book in 'CleanExpiredBooks':", err)
			return model.ErrCommon500
		}
	}

	// закоммитить транзакцию
	if err := tx.Commit(); err != nil {
		log.Println("Failed to commit transaction in 'CleanExpiredBooks':", err)
		return model.ErrCommon500
	}

	committed = true
	log.Printf("Cleaned %d expired bookings\n", len(books))
	return nil
}

func (eb EBService) GetBooksListByUserID(ctx context.Context, uid int) ([]*model.Book, error) {
	rid := model.RequestIDFromCtx(ctx)

	res, err := eb.repo.GetBooksListByUser(ctx, eb.db, uid)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrUserNotFound):
			return nil, err
		default:
			log.Printf("RID %q Failed to get book from DB in 'GetBooksListByUserID': %q", rid, err)
			return nil, model.ErrCommon500
		}
	}

	return res, nil
}

func (eb EBService) GetEventsList(ctx context.Context, role string) ([]*model.Event, error) {
	rid := model.RequestIDFromCtx(ctx)

	res, err := eb.repo.GetEventsList(ctx, eb.db, role)
	if err != nil {
		log.Printf("RID %q Failed to get all events from DB in 'GetEventsList': %v", rid, err)
		return nil, model.ErrCommon500
	}

	return res, nil
}
