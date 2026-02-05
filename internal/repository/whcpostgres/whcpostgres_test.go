package whcpostgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/dbpg"
)

func newMockRepo(t *testing.T) (*PostgresRepo, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	dbConn := dbpg.DB{Master: db}

	return &PostgresRepo{DB: &dbConn}, mock
}

// ================= METHODS TESTS ====================
func TestCreateUser(t *testing.T) {
	repo, mock := newMockRepo(t)
	timeNow := time.Now()
	someErr := errors.New("some error")

	cases := []struct {
		name     string
		arg      []string
		mockRows *sqlmock.Rows
		mockErr  error
		wantErr  error
		wantUser *model.User
	}{{
		name:     "Positive case - user created",
		arg:      []string{"john", "admin", "hash"},
		mockRows: sqlmock.NewRows([]string{"id", "created_at"}).AddRow(1, timeNow),
		mockErr:  nil,
		wantErr:  nil,
		wantUser: &model.User{
			ID:        1,
			UserName:  "john",
			Role:      "admin",
			PassHash:  "hash",
			CreatedAt: timeNow,
		},
	}, {
		name:     "Negative case - some error",
		arg:      []string{"john", "admin", "hash"},
		mockRows: nil,
		mockErr:  someErr,
		wantErr:  someErr,
		wantUser: nil,
	}}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			exp := mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(tt.arg[0], tt.arg[1], tt.arg[2])

			if tt.mockRows != nil {
				exp.WillReturnRows(tt.mockRows)
			} else {
				exp.WillReturnError(tt.mockErr)
			}
			user := &model.User{
				UserName: tt.arg[0],
				Role:     tt.arg[1],
				PassHash: tt.arg[2],
			}
			err := repo.CreateUser(context.Background(), user)

			if tt.wantUser != nil {
				require.Equal(t, tt.wantUser, user)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestGetUserByName(t *testing.T) {
	repo, mock := newMockRepo(t)

	dbError := errors.New("DB error. Try later")

	timeNow := time.Now()

	cases := []struct {
		name     string
		arg      string
		mockRows *sqlmock.Rows
		mockErr  error
		wantErr  error
		wantUser *model.User
	}{
		{
			name:     "Positive case - username found",
			arg:      "john",
			mockRows: sqlmock.NewRows([]string{"id", "role", "pass_hash", "created_at"}).AddRow(1, "admin", "hash", timeNow),
			mockErr:  nil,
			wantErr:  nil,
			wantUser: &model.User{
				ID:        1,
				UserName:  "john",
				Role:      "admin",
				PassHash:  "hash",
				CreatedAt: timeNow,
			},
		},
		{
			name:     "Negative case - username NOT found",
			arg:      "alice",
			mockRows: nil,
			mockErr:  sql.ErrNoRows,
			wantErr:  model.ErrUserNotFound,
			wantUser: nil,
		},
		{
			name:     "Negative case - DB error",
			arg:      "alice",
			mockRows: nil,
			mockErr:  dbError,
			wantErr:  dbError,
			wantUser: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			exp := mock.ExpectQuery(
				`SELECT id, role, pass_hash, created_at`,
			).WithArgs(tt.arg)

			if tt.mockRows != nil {
				exp.WillReturnRows(tt.mockRows)
			} else {
				exp.WillReturnError(tt.mockErr)
			}

			user, err := repo.GetUserByName(context.Background(), tt.arg)

			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantUser, user)
		})
	}
}

func TestCreateItem(t *testing.T) {
	repo, mock := newMockRepo(t)
	timeNow := time.Now()
	someErr := errors.New("some error")

	cases := []struct {
		name     string
		arg      *model.Item
		mockRows *sqlmock.Rows
		mockErr  error
		wantErr  error
		wantItem *model.Item
	}{
		{
			name: "Positive case - item created",
			arg: &model.Item{
				Title:           "some title",
				Description:     "some description",
				Price:           100500,
				Visible:         true,
				AvailableAmount: 300,
				UpdatedBy:       "someone",
			},
			mockRows: sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, timeNow, timeNow),
			mockErr:  nil,
			wantErr:  nil,
			wantItem: &model.Item{
				ID:              1,
				Title:           "some title",
				Description:     "some description",
				Price:           100500,
				Visible:         true,
				AvailableAmount: 300,
				UpdatedBy:       "someone",
				UpdatedAt:       timeNow,
				CreatedAt:       timeNow,
			},
		},
		{
			name: "Negative case - some DB error",
			arg: &model.Item{
				Title:           "some title",
				Description:     "some description",
				Price:           100500,
				Visible:         true,
				AvailableAmount: 300,
				UpdatedBy:       "someone",
			},
			mockRows: nil,
			mockErr:  someErr,
			wantErr:  someErr,
			wantItem: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			exp := mock.ExpectQuery(`INSERT INTO items`).
				WithArgs(tt.arg.Title,
					tt.arg.Description,
					tt.arg.Price,
					tt.arg.Visible,
					tt.arg.AvailableAmount,
					tt.arg.UpdatedBy)

			if tt.mockRows != nil {
				exp.WillReturnRows(tt.mockRows)
			} else {
				exp.WillReturnError(tt.mockErr)
			}

			err := repo.CreateItem(context.Background(), tt.arg)

			if tt.wantErr == nil {
				require.Equal(t, tt.wantItem, tt.arg)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestDeleteItem(t *testing.T) {
	repo, mock := newMockRepo(t)
	someErr := errors.New("some error")

	cases := []struct {
		name         string
		arg          int
		mockErr      error
		wantErr      error
		mockAffected int
	}{
		{
			name:         "Positive case - item soft-deleted",
			arg:          5,
			mockErr:      nil,
			wantErr:      nil,
			mockAffected: 1,
		},
		{
			name:         "Negative case - item not found",
			arg:          5,
			mockErr:      nil,
			wantErr:      model.ErrItemNotFound,
			mockAffected: 0,
		},
		{
			name:         "Negative case - some DB error",
			arg:          5,
			mockErr:      someErr,
			wantErr:      someErr,
			mockAffected: 0,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			exp := mock.ExpectExec(`UPDATE items SET deleted_at`).
				WithArgs(tt.arg)

			if tt.mockErr != nil {
				exp.WillReturnError(tt.mockErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, int64(tt.mockAffected)))
			}

			err := repo.DeleteItem(context.Background(), tt.arg)

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestUpdateItem(t *testing.T) {
	repo, mock := newMockRepo(t)
	someErr := errors.New("some error")
	getStringPtr := func(input string) *string {
		return &input
	}

	cases := []struct {
		name         string
		arg          model.ItemUpdate
		mockErr      error
		permission   bool
		mockAffected int
		wantErr      error
	}{
		{
			name: "Positive case - item updated",
			arg: model.ItemUpdate{
				ID:        1,
				Title:     getStringPtr("new title"),
				UpdatedBy: "someone",
			},
			mockErr:      nil,
			permission:   true,
			mockAffected: 1,
			wantErr:      nil,
		},
		{
			name: "Negative case - nothing to update",
			arg: model.ItemUpdate{
				ID:        1,
				UpdatedBy: "someone",
			},
			mockErr:      nil,
			permission:   true,
			mockAffected: 0,
			wantErr:      model.ErrNoFieldsToUpdate,
		},
		{
			name: "Negative case - no permission to see deleted",
			arg: model.ItemUpdate{
				ID:        1,
				Title:     getStringPtr("new title"),
				UpdatedBy: "someone",
			},
			mockErr:      nil,
			permission:   false,
			mockAffected: 0,
			wantErr:      model.ErrItemNotFound,
		},
		{
			name: "Negative case - not found",
			arg: model.ItemUpdate{
				ID:        1,
				Title:     getStringPtr("new title"),
				UpdatedBy: "someone",
			},
			mockErr:      nil,
			permission:   true,
			mockAffected: 0,
			wantErr:      model.ErrItemNotFound,
		},
		{
			name: "Negative case - DB error",
			arg: model.ItemUpdate{
				ID:        1,
				Title:     getStringPtr("new title"),
				UpdatedBy: "someone",
			},
			mockErr:      someErr,
			permission:   true,
			mockAffected: 0,
			wantErr:      someErr,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr != model.ErrNoFieldsToUpdate {
				exp := mock.ExpectExec(`UPDATE items`)
				if tt.mockErr != nil {
					exp.WillReturnError(tt.mockErr)
				} else {
					exp.WillReturnResult(sqlmock.NewResult(0, int64(tt.mockAffected)))
				}
			}

			err := repo.UpdateItem(context.Background(), &tt.arg, tt.permission)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestGetItemByID(t *testing.T) {
	repo, mock := newMockRepo(t)

	dbError := errors.New("DB error. Try later")

	timeNow := time.Now()

	cases := []struct {
		name       string
		arg        int
		permission bool
		mockRows   *sqlmock.Rows
		mockErr    error
		wantErr    error
		wantItem   *model.Item
	}{
		{
			name:       "Positive case - itemID found",
			arg:        1,
			permission: true,
			mockRows: sqlmock.NewRows([]string{"id", "title", "description", "price", "visible", "available_amount", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, "title", "description", 100500, true, 300, timeNow, timeNow, nil),
			mockErr: nil,
			wantErr: nil,
			wantItem: &model.Item{
				ID:              1,
				Title:           "title",
				Description:     "description",
				Price:           100500,
				Visible:         true,
				AvailableAmount: 300,
				CreatedAt:       timeNow,
				UpdatedAt:       timeNow,
				DeletedAt:       nil,
			},
		},
		{
			name:       "Negative case - itemID NOT found",
			arg:        1,
			permission: true,
			mockRows:   nil,
			mockErr:    sql.ErrNoRows,
			wantErr:    model.ErrItemNotFound,
			wantItem:   nil,
		},
		{
			name:       "Negative case - itemID NOT found - no permission",
			arg:        1,
			permission: false,
			mockRows:   nil,
			mockErr:    sql.ErrNoRows,
			wantErr:    model.ErrItemNotFound,
			wantItem:   nil,
		},
		{
			name:     "Negative case - DB error",
			arg:      1,
			mockRows: nil,
			mockErr:  dbError,
			wantErr:  dbError,
			wantItem: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			exp := mock.ExpectQuery(
				`SELECT id, title, description, price, visible, available_amount, created_at, updated_at, deleted_at FROM items	WHERE id =`,
			).WithArgs(tt.arg)

			if tt.mockRows != nil {
				exp.WillReturnRows(tt.mockRows)
			} else {
				exp.WillReturnError(tt.mockErr)
			}

			item, err := repo.GetItemByID(context.Background(), 1, true)

			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantItem, item)
		})
	}
}

func TestGetItemsList(t *testing.T) {
	repo, mock := newMockRepo(t)

	dbError := errors.New("DB error. Try later")

	timeNow := time.Now()

	cases := []struct {
		name       string
		arg        *model.RequestParam
		permission bool
		mockRows   *sqlmock.Rows
		mockErr    error
		wantErr    error
		wantResult []*model.Item
	}{
		{
			name:       "Positive case - array of 1 item",
			arg:        &model.RequestParam{},
			permission: true,
			mockRows: sqlmock.NewRows([]string{"id", "title", "description", "price", "visible", "available_amount", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, "title", "description", 100500, true, 300, timeNow, timeNow, nil),
			mockErr: nil,
			wantErr: nil,
			wantResult: []*model.Item{{
				ID:              1,
				Title:           "title",
				Description:     "description",
				Price:           100500,
				Visible:         true,
				AvailableAmount: 300,
				CreatedAt:       timeNow,
				UpdatedAt:       timeNow,
				DeletedAt:       nil,
			}},
		},
		{
			name:       "Positive case - empty array of items",
			arg:        &model.RequestParam{},
			permission: true,
			mockRows:   nil,
			mockErr:    nil,
			wantErr:    nil,
			wantResult: nil,
		},
		{
			name:       "Positive case - empty array of items - no permission",
			arg:        &model.RequestParam{},
			permission: false,
			mockRows:   nil,
			mockErr:    nil,
			wantErr:    nil,
			wantResult: nil,
		},
		{
			name:       "Negative case - DB error",
			arg:        &model.RequestParam{},
			mockRows:   nil,
			mockErr:    dbError,
			wantErr:    dbError,
			wantResult: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			exp := mock.ExpectQuery(`SELECT id, title, description, price, visible, available_amount, created_at, updated_at, deleted_at FROM items`)

			if tt.mockRows != nil {
				exp.WillReturnRows(tt.mockRows)
			} else {
				exp.WillReturnError(tt.mockErr)
			}

			res, err := repo.GetItemsList(context.Background(), tt.arg, tt.permission)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.Equal(t, tt.wantResult, res)
			}
		})
	}
}

func TestGetItemHistoryByID(t *testing.T) {
	repo, mock := newMockRepo(t)
	jsonPtrMaker := func(input json.RawMessage) *json.RawMessage {
		return &input
	}
	dbError := errors.New("DB error. Try later")

	timeNow := time.Now()

	cases := []struct {
		name       string
		arg        *model.RequestParam
		itemID     int
		mockRows   *sqlmock.Rows
		mockErr    error
		wantErr    error
		wantResult []*model.ItemHistory
	}{
		{
			name:   "Positive case - array of 2 histories",
			arg:    &model.RequestParam{},
			itemID: 1,
			mockRows: sqlmock.NewRows([]string{"id", "item_id", "version", "action", "changed_at", "changed_by", "old_data", "new_data"}).
				AddRow(1, 1, 2, "UPDATE", timeNow, "someone", json.RawMessage("some old data"), json.RawMessage("some new data")).
				AddRow(2, 1, 3, "DELETE", timeNow, "elseone", json.RawMessage("some old data"), json.RawMessage("some new data")),
			mockErr: nil,
			wantErr: nil,
			wantResult: []*model.ItemHistory{{
				ID: 1, ItemID: 1, Version: 2, Action: "UPDATE",
				ChangedAt: timeNow, ChangedBy: "someone",
				OldData: jsonPtrMaker(json.RawMessage("some old data")),
				NewData: json.RawMessage("some new data"),
			}, {
				ID: 2, ItemID: 1, Version: 3, Action: "DELETE",
				ChangedAt: timeNow, ChangedBy: "elseone",
				OldData: jsonPtrMaker(json.RawMessage("some old data")),
				NewData: json.RawMessage("some new data"),
			}},
		},
		{
			name:       "Positive case - empty array of histories",
			arg:        &model.RequestParam{},
			itemID:     1,
			mockRows:   nil,
			mockErr:    nil,
			wantErr:    nil,
			wantResult: nil,
		},
		{
			name:       "Negative case - DB error",
			arg:        &model.RequestParam{},
			itemID:     1,
			mockRows:   nil,
			mockErr:    dbError,
			wantErr:    dbError,
			wantResult: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			exp := mock.ExpectQuery(`SELECT id, item_id, version, action, changed_at, changed_by, old_data, new_data 
			FROM items_history 
			WHERE item_id =`)

			if tt.mockRows != nil {
				exp.WillReturnRows(tt.mockRows)
			} else {
				exp.WillReturnError(tt.mockErr)
			}

			res, err := repo.GetItemHistoryByID(context.Background(), tt.arg, tt.itemID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.Equal(t, tt.wantResult, res)
			}
		})
	}
}

func TestGetItemHistoryAll(t *testing.T) {
	repo, mock := newMockRepo(t)
	jsonPtrMaker := func(input json.RawMessage) *json.RawMessage {
		return &input
	}
	dbError := errors.New("DB error. Try later")

	timeNow := time.Now()

	cases := []struct {
		name       string
		arg        *model.RequestParam
		mockRows   *sqlmock.Rows
		mockErr    error
		wantErr    error
		wantResult []*model.ItemHistory
	}{
		{
			name: "Positive case - array of 2 histories",
			arg:  &model.RequestParam{},
			mockRows: sqlmock.NewRows([]string{"id", "item_id", "version", "action", "changed_at", "changed_by", "old_data", "new_data"}).
				AddRow(1, 1, 2, "UPDATE", timeNow, "someone", json.RawMessage("some old data"), json.RawMessage("some new data")).
				AddRow(2, 2, 3, "DELETE", timeNow, "elseone", json.RawMessage("some old data"), json.RawMessage("some new data")),
			mockErr: nil,
			wantErr: nil,
			wantResult: []*model.ItemHistory{{
				ID: 1, ItemID: 1, Version: 2, Action: "UPDATE",
				ChangedAt: timeNow, ChangedBy: "someone",
				OldData: jsonPtrMaker(json.RawMessage("some old data")),
				NewData: json.RawMessage("some new data"),
			}, {
				ID: 2, ItemID: 2, Version: 3, Action: "DELETE",
				ChangedAt: timeNow, ChangedBy: "elseone",
				OldData: jsonPtrMaker(json.RawMessage("some old data")),
				NewData: json.RawMessage("some new data"),
			}},
		},
		{
			name:       "Positive case - empty array of histories",
			arg:        &model.RequestParam{},
			mockRows:   nil,
			mockErr:    nil,
			wantErr:    nil,
			wantResult: nil,
		},
		{
			name:       "Negative case - DB error",
			arg:        &model.RequestParam{},
			mockRows:   nil,
			mockErr:    dbError,
			wantErr:    dbError,
			wantResult: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			exp := mock.ExpectQuery(`SELECT id, item_id, version, action, changed_at, changed_by, old_data, new_data 
			FROM items_history`)

			if tt.mockRows != nil {
				exp.WillReturnRows(tt.mockRows)
			} else {
				exp.WillReturnError(tt.mockErr)
			}

			res, err := repo.GetItemHistoryAll(context.Background(), tt.arg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.Equal(t, tt.wantResult, res)
			}
		})
	}
}

// ==================== TOOLS TABLE TESTS ======================
func TestDefineOrderExpr(t *testing.T) {
	tests := []struct {
		inputOrderBy string
		inputAsc     bool
		inputDesc    bool
		wantErr      error
		wantString   string
	}{
		{model.ItemsOrderByTitle, true, false, nil, " ORDER BY title ASC "},
		{model.HistoryOrderByAction, false, true, nil, " ORDER BY action DESC "},
		{model.ItemsOrderByID, true, true, nil, " ORDER BY id DESC "},
		{model.HistoryOrderByActor, false, false, nil, " ORDER BY actor DESC "},
		{"foobar", true, false, model.ErrInvalidOrderBy, ""},
		{"", true, false, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.inputOrderBy, func(t *testing.T) {
			ptrOrderBy := &tt.inputOrderBy
			if tt.inputOrderBy == "" {
				ptrOrderBy = nil
			}

			res, err := defineOrderExpr(ptrOrderBy, tt.inputAsc, tt.inputDesc)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantString, res)
		})
	}
}

func TestDefineLimitOffsetExpr(t *testing.T) {
	intPtrMaker := func(n int) *int {
		return &n
	}

	cases := []struct {
		name     string
		limit    *int
		page     *int
		wantExpr string
	}{
		{
			name:     "limit and page - both nil",
			limit:    nil,
			page:     nil,
			wantExpr: "",
		},
		{
			name:     "limit = nil, page = 2",
			limit:    nil,
			page:     intPtrMaker(2),
			wantExpr: "LIMIT 20 OFFSET 20",
		},
		{
			name:     "limit = 20, page = nil",
			limit:    intPtrMaker(20),
			page:     nil,
			wantExpr: "LIMIT 20 OFFSET 0",
		},
		{
			name:     "limit = 1500, page = 5",
			limit:    intPtrMaker(1500),
			page:     intPtrMaker(5),
			wantExpr: "LIMIT 1000 OFFSET 4000",
		},
		{
			name:     "limit = -10, page = -2",
			limit:    intPtrMaker(-10),
			page:     intPtrMaker(-2),
			wantExpr: "LIMIT 20 OFFSET 0",
		},
		{
			name:     "limit = -10, page = 2",
			limit:    intPtrMaker(-10),
			page:     intPtrMaker(2),
			wantExpr: "LIMIT 20 OFFSET 20",
		},
		{
			name:     "limit = 10, page = -2",
			limit:    intPtrMaker(10),
			page:     intPtrMaker(-2),
			wantExpr: "LIMIT 10 OFFSET 0",
		},
		{
			name:     "limit = 0, page = 0",
			limit:    intPtrMaker(0),
			page:     intPtrMaker(0),
			wantExpr: "LIMIT 20 OFFSET 0",
		},
		{
			name:     "limit = 0, page = 2",
			limit:    intPtrMaker(0),
			page:     intPtrMaker(2),
			wantExpr: "LIMIT 20 OFFSET 20",
		},
		{
			name:     "limit = 10, page = 0",
			limit:    intPtrMaker(10),
			page:     intPtrMaker(0),
			wantExpr: "LIMIT 10 OFFSET 0",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := defineLimitOffsetExpr(tt.limit, tt.page)
			require.Equal(t, tt.wantExpr, res)
		})
	}
}

func TestDefinePeriodExpr(t *testing.T) {
	unitime, _ := time.Parse("2006-01-02", "2666-06-06")
	stringUnitime := unitime.Format(time.RFC3339)

	cases := []struct {
		name       string
		start      *time.Time
		end        *time.Time
		leadOp     string
		dbField    string
		wantString string
	}{
		{
			name:       "start and end - nil",
			start:      nil,
			end:        nil,
			leadOp:     "LEADOP",
			dbField:    "db_field",
			wantString: "",
		}, {
			name:       "start - nil, end - time",
			start:      nil,
			end:        &unitime,
			leadOp:     "LEADOP",
			dbField:    "db_field",
			wantString: fmt.Sprintf(" %s %s < '%s'", "LEADOP", "db_field", stringUnitime),
		}, {
			name:       "start - time, end - nil",
			start:      &unitime,
			end:        nil,
			leadOp:     "LEADOP",
			dbField:    "db_field",
			wantString: fmt.Sprintf(" %s %s > '%s'", "LEADOP", "db_field", stringUnitime),
		}, {
			name:       "start and end - correct time",
			start:      &unitime,
			end:        &unitime,
			leadOp:     "LEADOP",
			dbField:    "db_field",
			wantString: fmt.Sprintf(" %s %s BETWEEN '%s' AND '%s'", "LEADOP", "db_field", stringUnitime, stringUnitime),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := definePeriodExpr(tt.start, tt.end, tt.leadOp, tt.dbField)

			require.Equal(t, tt.wantString, res)
		})
	}
}

func TestUpdateQueryBuilder(t *testing.T) {
	title := "title"
	description := "item description"
	price := int64(100500)
	visible := true
	availamount := 100500
	updatedby := "user"

	cases := []struct {
		name        string
		itemUPD     *model.ItemUpdate
		wantString  string
		wantArgsLen int // внутри теста надо сначала проверять ниловость возвращенного слайса
		wantErr     error
	}{
		{
			name: "all pointer fields - nil - nothing to update",
			itemUPD: &model.ItemUpdate{
				Title:           nil,
				Description:     nil,
				Price:           nil,
				Visible:         nil,
				AvailableAmount: nil,
				UpdatedBy:       updatedby,
			},
			wantString:  "",
			wantArgsLen: 0,
			wantErr:     model.ErrNoFieldsToUpdate,
		}, {
			name: "title is updated",
			itemUPD: &model.ItemUpdate{
				Title:           &title,
				Description:     nil,
				Price:           nil,
				Visible:         nil,
				AvailableAmount: nil,
				UpdatedBy:       updatedby,
			},
			wantString:  "SET title = $2, updated_by = $3",
			wantArgsLen: 2,
			wantErr:     nil,
		}, {
			name: "descrition is updated",
			itemUPD: &model.ItemUpdate{
				Title:           nil,
				Description:     &description,
				Price:           nil,
				Visible:         nil,
				AvailableAmount: nil,
				UpdatedBy:       updatedby,
			},
			wantString:  "SET description = $2, updated_by = $3",
			wantArgsLen: 2,
			wantErr:     nil,
		}, {
			name: "price is updated",
			itemUPD: &model.ItemUpdate{
				Title:           nil,
				Description:     nil,
				Price:           &price,
				Visible:         nil,
				AvailableAmount: nil,
				UpdatedBy:       updatedby,
			},
			wantString:  "SET price = $2, updated_by = $3",
			wantArgsLen: 2,
			wantErr:     nil,
		}, {
			name: "visibility is updated",
			itemUPD: &model.ItemUpdate{
				Title:           nil,
				Description:     nil,
				Price:           nil,
				Visible:         &visible,
				AvailableAmount: nil,
				UpdatedBy:       updatedby,
			},
			wantString:  "SET visible = $2, updated_by = $3",
			wantArgsLen: 2,
			wantErr:     nil,
		}, {
			name: "title and description are updated",
			itemUPD: &model.ItemUpdate{
				Title:           &title,
				Description:     &description,
				Price:           nil,
				Visible:         nil,
				AvailableAmount: nil,
				UpdatedBy:       updatedby,
			},
			wantString:  "SET title = $2, description = $3, updated_by = $4",
			wantArgsLen: 3,
			wantErr:     nil,
		}, {
			name: "title, description, price are updated",
			itemUPD: &model.ItemUpdate{
				Title:           &title,
				Description:     &description,
				Price:           &price,
				Visible:         nil,
				AvailableAmount: nil,
				UpdatedBy:       updatedby,
			},
			wantString:  "SET title = $2, description = $3, price = $4, updated_by = $5",
			wantArgsLen: 4,
			wantErr:     nil,
		}, {
			name: "title, description, price, visibility are updated",
			itemUPD: &model.ItemUpdate{
				Title:           &title,
				Description:     &description,
				Price:           &price,
				Visible:         &visible,
				AvailableAmount: nil,
				UpdatedBy:       updatedby,
			},
			wantString:  "SET title = $2, description = $3, price = $4, visible = $5, updated_by = $6",
			wantArgsLen: 5,
			wantErr:     nil,
		}, {
			name: "all fields are updated",
			itemUPD: &model.ItemUpdate{
				Title:           &title,
				Description:     &description,
				Price:           &price,
				Visible:         &visible,
				AvailableAmount: &availamount,
				UpdatedBy:       updatedby,
			},
			wantString:  "SET title = $2, description = $3, price = $4, visible = $5, available_amount = $6, updated_by = $7",
			wantArgsLen: 6,
			wantErr:     nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resString, resArgs, err := updateQueryBuilder(tt.itemUPD)

			var lenArgs int
			if resArgs != nil {
				lenArgs = len(resArgs)
			}

			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantArgsLen, lenArgs)
			require.Equal(t, tt.wantString, resString)
		})
	}
}
