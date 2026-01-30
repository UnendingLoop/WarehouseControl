// Package ebpostgres provides methods for communicating with DB PostgresQL to service-layer
package ebpostgres

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/UnendingLoop/EventBooker/internal/model"
)

type PostgresRepo struct{}

// Executor provides a way to use both transactions and sql.DB for running queries from Service-layer
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// CreateEvent - создание ивента доступно только для админа
func (pr PostgresRepo) CreateEvent(ctx context.Context, exec Executor, newEvent *model.Event) error {
	query := `INSERT INTO events (id, title, description, status, event_date, created_at, bookwindow, total_seats, avail_seats)
	VALUES (DEFAULT, $1, $2, $3, $4, DEFAULT, $5, $6, $7) RETURNING id`
	err := exec.QueryRowContext(ctx, query, newEvent.Title, newEvent.Descr, newEvent.Status, newEvent.EventDate, newEvent.BookWindow, newEvent.TotalSeats, newEvent.AvailSeats).Scan(&newEvent.ID)
	if err != nil {
		return err
	}
	return nil
}

func (pr PostgresRepo) CreateBook(ctx context.Context, exec Executor, newBook *model.Book) error {
	query := `INSERT INTO bookings (id, event_id, user_id, status, created_at, confirm_deadline)
	VALUES (DEFAULT, $1, $2, $3, DEFAULT, $4) RETURNING id`
	err := exec.QueryRowContext(ctx, query, newBook.EventID, newBook.UserID, newBook.Status, newBook.ConfirmDeadline).Scan(&newBook.ID)
	if err != nil {
		return err
	}
	return nil
}

func (pr PostgresRepo) CreateUser(ctx context.Context, exec Executor, newUser *model.User) error {
	query := `INSERT INTO users (id, created_at, role, name, surname, tel, email, pass_hash)
	VALUES (DEFAULT, DEFAULT, $1, $2, $3, $4, $5, $6) RETURNING id`
	err := exec.QueryRowContext(ctx, query, newUser.Role, newUser.Name, newUser.Surname, newUser.Tel, newUser.UserName, newUser.PassHash).Scan(&newUser.ID)
	if err != nil {
		return err
	}
	return nil
}

// DeleteEvent - удаление ивента только для роли админа
func (pr PostgresRepo) DeleteEvent(ctx context.Context, exec Executor, eventID int) error {
	query := `DELETE FROM events
	WHERE id = $1`

	res, err := exec.ExecContext(ctx, query, eventID)
	if err != nil {
		return err // 500
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err // 500
	}
	if rows == 0 {
		return model.ErrEventNotFound // 404
	}
	return nil
}

// DeleteBook - эксклюзивно для воркера BookCleaner
func (pr PostgresRepo) DeleteBook(ctx context.Context, exec Executor, bookID int) error {
	query := `DELETE FROM bookings
	WHERE id = $1`

	row, err := exec.ExecContext(ctx, query, bookID)
	if err != nil {
		return err // 500
	}
	n, err := row.RowsAffected()
	if err != nil {
		return err // 500
	}
	if n == 0 {
		return model.ErrBookNotFound
	}
	return nil
}

func (pr PostgresRepo) UpdateBookStatus(ctx context.Context, exec Executor, bookID int, newStatus string) error {
	query := `UPDATE bookings SET status=$1 WHERE id = $2`

	res, err := exec.ExecContext(ctx, query, newStatus, bookID)
	if err != nil {
		return err // 500
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return model.ErrBookNotFound // 404
	}

	return nil
}

func (pr PostgresRepo) GetEventByID(ctx context.Context, exec Executor, id int) (*model.Event, error) { // select FOR UPDATE
	query := `SELECT id, title, description, status, event_date, created_at, bookwindow, total_seats, avail_seats 
	FROM events 
	WHERE id = $1 FOR UPDATE`

	var event model.Event

	err := exec.QueryRowContext(ctx, query, id).Scan(&event.ID,
		&event.Title,
		&event.Descr,
		&event.Status,
		&event.EventDate,
		&event.Created,
		&event.BookWindow,
		&event.TotalSeats,
		&event.AvailSeats)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, model.ErrEventNotFound
		default:
			return nil, err // 500
		}
	}
	return &event, nil
}

func (pr PostgresRepo) GetEventsList(ctx context.Context, exec Executor, role string) ([]*model.Event, error) {
	query := `SELECT id, title, description, status, event_date, created_at, bookwindow, total_seats, avail_seats 
	FROM events`
	if role != model.RoleAdmin { // пользователю - только актуальные ивенты
		query += ` WHERE event_date > now() AND status = 'actual'`
	}

	rows, err := exec.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error while closing *sql.Rows after scanning: %v", err)
		}
	}()

	events := make([]*model.Event, 0)

	for rows.Next() {
		var event model.Event
		if err := rows.Scan(&event.ID,
			&event.Title,
			&event.Descr,
			&event.Status,
			&event.EventDate,
			&event.Created,
			&event.BookWindow,
			&event.TotalSeats,
			&event.AvailSeats); err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return events, nil
}

func (pr PostgresRepo) GetBookByID(ctx context.Context, exec Executor, id int) (*model.Book, error) {
	query := `SELECT id, event_id, user_id, status, created_at, confirm_deadline 
	FROM bookings 
	WHERE id = $1 FOR UPDATE`

	var book model.Book

	err := exec.QueryRowContext(ctx, query, id).Scan(&book.ID,
		&book.EventID,
		&book.UserID,
		&book.Status,
		&book.Created,
		&book.ConfirmDeadline)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, model.ErrBookNotFound
		default:
			return nil, err // 500
		}
	}
	return &book, nil
}

func (pr PostgresRepo) GetBooksListByUser(ctx context.Context, exec Executor, id int) ([]*model.Book, error) {
	query := `SELECT id, event_id, user_id, status, created_at, confirm_deadline FROM bookings 
	WHERE user_id = $1`
	rows, err := exec.QueryContext(ctx, query, id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, model.ErrUserNotFound
		default:
			return nil, err // 500
		}
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error while closing *sql.Rows after scanning: %v", err)
		}
	}()

	books := make([]*model.Book, 0)

	for rows.Next() {
		var book model.Book
		if err := rows.Scan(&book.ID,
			&book.EventID,
			&book.UserID,
			&book.Status,
			&book.Created,
			&book.ConfirmDeadline); err != nil {
			return nil, err
		}
		books = append(books, &book)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return books, nil
}

// GetExpiredBooksList - эксклюзивно для воркера BookCleaner
func (pr PostgresRepo) GetExpiredBooksList(ctx context.Context, exec Executor) ([]*model.Book, error) {
	query := `SELECT id, event_id, user_id, status, created_at FROM bookings 
	WHERE confirm_deadline < now() AND status != $1 FOR UPDATE`
	rows, err := exec.QueryContext(ctx, query, model.BookStatusConfirmed)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error while closing *sql.Rows after scanning: %v", err)
		}
	}()

	books := make([]*model.Book, 0)

	for rows.Next() {
		var book model.Book
		if err := rows.Scan(&book.ID,
			&book.EventID,
			&book.UserID,
			&book.Status,
			&book.Created); err != nil {
			return nil, err
		}
		books = append(books, &book)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return books, nil
}

func (pr PostgresRepo) GetUserByID(ctx context.Context, exec Executor, id int) (*model.User, error) {
	query := `SELECT id, created_at, role, name, surname, tel, email, pass_hash 
	FROM users 
	WHERE id = $1`

	var user model.User

	err := exec.QueryRowContext(ctx, query, id).Scan(&user.ID,
		&user.Created,
		&user.Role,
		&user.Name,
		&user.Surname,
		&user.Tel,
		&user.UserName,
		&user.PassHash)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, model.ErrUserNotFound
		default:
			return nil, err // 500
		}
	}
	return &user, nil
}

func (pr PostgresRepo) GetUserByEmail(ctx context.Context, exec Executor, email string) (*model.User, error) {
	query := `SELECT id, created_at, role, name, surname, tel, email, pass_hash 
	FROM users 
	WHERE email = $1`

	var user model.User

	err := exec.QueryRowContext(ctx, query, email).Scan(&user.ID,
		&user.Created,
		&user.Role,
		&user.Name,
		&user.Surname,
		&user.Tel,
		&user.UserName,
		&user.PassHash)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, model.ErrUserNotFound
		default:
			return nil, err // 500
		}
	}
	return &user, nil
}

func (pr PostgresRepo) IncrementAvailSeatsByEventID(ctx context.Context, exec Executor, eventID int) error {
	query := `UPDATE events 
	SET avail_seats = avail_seats + 1 
	WHERE id = $1`

	res, err := exec.ExecContext(ctx, query, eventID)
	if err != nil {
		return err // 500
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return model.ErrEventNotFound // 404
	}

	return nil
}

func (pr PostgresRepo) DecrementAvailSeatsByEventID(ctx context.Context, exec Executor, eventID int) error {
	query := `UPDATE events 
	SET avail_seats = avail_seats - 1 
	WHERE id = $1`

	res, err := exec.ExecContext(ctx, query, eventID)
	if err != nil {
		return err // 500
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return model.ErrEventNotFound // 404
	}

	return nil
}
