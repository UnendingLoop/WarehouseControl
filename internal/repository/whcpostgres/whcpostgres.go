// Package whcpostgres provides methods for communicating with DB PostgresQL to service-layer
package whcpostgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"github.com/wb-go/wbf/dbpg"
)

type PostgresRepo struct {
	DB *dbpg.DB
}

func (pr PostgresRepo) CreateUser(ctx context.Context, newUser *model.User) error {
	query := `INSERT INTO users (id, username, role, pass_hash, created_at)
	VALUES (DEFAULT, $1, $2, $3, DEFAULT) RETURNING id, created_at`
	err := pr.DB.QueryRowContext(ctx, query,
		newUser.UserName,
		newUser.Role,
		newUser.PassHash).Scan(
		&newUser.ID,
		&newUser.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (pr PostgresRepo) GetUserByName(ctx context.Context, userName string) (*model.User, error) {
	query := `SELECT id, role, pass_hash, created_at 
	FROM users 
	WHERE username = $1`

	user := model.User{UserName: userName}

	err := pr.DB.QueryRowContext(ctx, query, userName).Scan(
		&user.ID,
		&user.Role,
		&user.PassHash,
		&user.CreatedAt)
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

func (pr PostgresRepo) CreateItem(ctx context.Context, newItem *model.Item) error {
	query := `INSERT INTO items (id, title, description, price, visible, available_amount, created_at, updated_at, updated_by)
	VALUES (DEFAULT, $1, $2, $3, $4, $5, DEFAULT,DEFAULT,$6) RETURNING id, created_at, updated_at`
	err := pr.DB.QueryRowContext(ctx, query,
		newItem.Title,
		newItem.Description,
		newItem.Price,
		newItem.Visible,
		newItem.AvailableAmount,
		newItem.UpdatedBy).Scan(&newItem.ID, &newItem.CreatedAt, &newItem.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (pr PostgresRepo) DeleteItem(ctx context.Context, itemID int) error {
	query := `UPDATE items SET deleted_at = NOW()
	WHERE id = $1`

	res, err := pr.DB.ExecContext(ctx, query, itemID)
	if err != nil {
		return err // 500
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err // 500
	}
	if rows == 0 {
		return model.ErrItemNotFound // 404
	}
	return nil
}

func (pr PostgresRepo) UpdateItem(ctx context.Context, uItem *model.ItemUpdate, canSeeDeleted bool) error {
	setClause, values, err := updateQueryBuilder(uItem)
	if err != nil {
		return err
	}

	// $1 будет id
	query := fmt.Sprintf(`UPDATE items %s WHERE id = $1`, setClause)

	// если нет доступа на просмотр удаленных - добавляем это в квери
	if !canSeeDeleted {
		query += ` AND deleted_at IS NULL`
	}

	// вставляем id первым аргументом
	args := append([]any{uItem.ID}, values...)

	// log.Printf("Update-query: %q \nArguments: %v", query, args)

	res, err := pr.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrItemNotFound
	}
	return nil
}

func (pr PostgresRepo) GetItemByID(ctx context.Context, itemID int, canSeeDeleted bool) (*model.Item, error) { // select FOR UPDATE
	query := `SELECT id, title, description, price, visible, available_amount, created_at, updated_at, deleted_at 
	FROM items 
	WHERE id = $1`

	// если нет доступа на просмотр удаленных - добавляем это в квери
	if !canSeeDeleted {
		query += ` AND deleted_at IS NULL`
	}

	var item model.Item

	err := pr.DB.QueryRowContext(ctx, query, itemID).Scan(&item.ID,
		&item.Title,
		&item.Description,
		&item.Price,
		&item.Visible,
		&item.AvailableAmount,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, model.ErrItemNotFound
		default:
			return nil, err // 500
		}
	}
	return &item, nil
}

func (pr PostgresRepo) GetItemsList(ctx context.Context, rpi *model.RequestParam, canSeeDeleted bool) ([]*model.Item, error) {
	query := `SELECT id, title, description, price, visible, available_amount, created_at, updated_at, deleted_at 
	FROM items`
	// добавляем сортировку по полю
	orderExpr, err := defineOrderExpr(rpi.OrderBy, rpi.ASC, rpi.DESC)
	if err != nil {
		return nil, err
	}

	// добавляем ограничение по времени
	periodExpr := definePeriodExpr(rpi.StartTime, rpi.EndTime, "WHERE", "created_at")
	// если нет доступа на просмотр удаленных - добавляем условие
	if !canSeeDeleted {
		switch periodExpr {
		case "":
			periodExpr = ` WHERE deleted_at IS NULL `
		default:
			periodExpr += ` AND deleted_at IS NULL `
		}
	}

	limofExpr := defineLimitOffsetExpr(rpi.Limit, rpi.Page)

	// собираем конечный квери
	query = query + periodExpr + orderExpr + limofExpr

	// выполняем запрос
	rows, err := pr.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error while closing *sql.Rows after scanning: %v", err)
		}
	}()

	items := make([]*model.Item, 0)

	for rows.Next() {
		var item model.Item
		if err := rows.Scan(&item.ID,
			&item.Title,
			&item.Description,
			&item.Price,
			&item.Visible,
			&item.AvailableAmount,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DeletedAt); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return items, nil
}

func (pr PostgresRepo) GetItemHistoryByID(ctx context.Context, rph *model.RequestParam, itemID int) ([]*model.ItemHistory, error) {
	query := `SELECT id, item_id, version, action, changed_at, changed_by, old_data, new_data 
	FROM items_history
	WHERE item_id = $1`

	// добавляем ограничение по времени
	periodExpr := definePeriodExpr(rph.StartTime, rph.EndTime, "AND", "changed_at")

	// добавляем сортировку по полю
	orderExpr, err := defineOrderExpr(rph.OrderBy, rph.ASC, rph.DESC)
	if err != nil {
		return nil, err
	}

	// применяем лимит и оффсет
	limofExpr := defineLimitOffsetExpr(rph.Limit, rph.Page)

	// собираем конечный квери
	query = query + orderExpr + periodExpr + limofExpr

	// выполняем запрос
	rows, err := pr.DB.QueryContext(ctx, query, itemID)
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

	history := make([]*model.ItemHistory, 0)

	for rows.Next() {
		var h model.ItemHistory
		if err := rows.Scan(&h.ID,
			&h.ItemID,
			&h.Version,
			&h.Action,
			&h.ChangedAt,
			&h.ChangedBy,
			&h.OldData,
			&h.NewData); err != nil {
			return nil, err
		}
		history = append(history, &h)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return history, nil
}

func (pr PostgresRepo) GetItemHistoryAll(ctx context.Context, rph *model.RequestParam) ([]*model.ItemHistory, error) {
	query := `SELECT id, item_id, version, action, changed_at, changed_by, old_data, new_data 
	FROM items_history `

	// добавляем ограничение по времени
	periodExpr := definePeriodExpr(rph.StartTime, rph.EndTime, "WHERE", "changed_at")

	// добавляем сортировку по полю
	orderExpr, err := defineOrderExpr(rph.OrderBy, rph.ASC, rph.DESC)
	if err != nil {
		return nil, err
	}

	// применяем лимит и оффсет
	limofExpr := defineLimitOffsetExpr(rph.Limit, rph.Page)

	// собираем конечный квери
	query = query + periodExpr + orderExpr + limofExpr

	// выполняем запрос
	rows, err := pr.DB.QueryContext(ctx, query)
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

	history := make([]*model.ItemHistory, 0)

	for rows.Next() {
		var h model.ItemHistory
		if err := rows.Scan(&h.ID,
			&h.ItemID,
			&h.Version,
			&h.Action,
			&h.ChangedAt,
			&h.ChangedBy,
			&h.OldData,
			&h.NewData); err != nil {
			return nil, err
		}
		history = append(history, &h)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return history, nil
}
