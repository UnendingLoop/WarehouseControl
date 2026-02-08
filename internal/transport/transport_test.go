package transport_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/UnendingLoop/WarehouseControl/internal/engine"
	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"github.com/UnendingLoop/WarehouseControl/internal/transport"
	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/ginext"
)

func TestSimplePinger(t *testing.T) {
	h := transport.NewWHCHandlers(&transport.ServiceMock{})
	r := newTestServer(h)

	cases := []struct {
		name     string
		method   string
		target   string
		wantCode int
	}{
		{
			name:     "Positive - correct GET - 200",
			method:   http.MethodGet,
			target:   "/ping",
			wantCode: http.StatusOK,
		},
		{
			name:     "Negative - invalid POST - 404",
			method:   http.MethodPost,
			target:   "/ping",
			wantCode: http.StatusNotFound, // хотя в теории должен быть 405 StatusMethodNotAllowed
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestSignUpUser(t *testing.T) {
	cases := []struct {
		name       string
		user       *model.User
		method     string
		target     string
		mockSvc    *transport.ServiceMock
		wantCode   int
		wantCookie *string
	}{
		{
			name: "Positive - user create success",
			user: &model.User{
				UserName: "someName",
				Role:     "someRole",
				PassHash: "somePass",
			},
			method: http.MethodPost,
			target: "/auth/signup",
			mockSvc: &transport.ServiceMock{CreateUserFn: func(ctx context.Context, user *model.User) (string, error) {
				return "jwt-token", nil
			}},
			wantCode:   http.StatusCreated,
			wantCookie: ptrMaker("jwt-token"),
		},
		{
			name:       "Negative - nil user info",
			user:       nil,
			method:     http.MethodPost,
			target:     "/auth/signup",
			mockSvc:    &transport.ServiceMock{},
			wantCode:   http.StatusBadRequest,
			wantCookie: nil,
		},
		{
			name: "Negative - DB error - 500",
			user: &model.User{
				UserName: "someName",
				Role:     "someRole",
				PassHash: "somePass",
			},
			method: http.MethodPost,
			target: "/auth/signup",
			mockSvc: &transport.ServiceMock{CreateUserFn: func(ctx context.Context, user *model.User) (string, error) {
				return "", model.ErrCommon500
			}},
			wantCode:   http.StatusInternalServerError,
			wantCookie: nil,
		},
		{
			name: "Negative - user already exists",
			user: &model.User{
				UserName: "someName",
				Role:     "someRole",
				PassHash: "somePass",
			},
			method: http.MethodPost,
			target: "/auth/signup",
			mockSvc: &transport.ServiceMock{CreateUserFn: func(ctx context.Context, user *model.User) (string, error) {
				return "", model.ErrUserAlreadyExists
			}},
			wantCode:   http.StatusConflict,
			wantCookie: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.user)
			require.NoError(t, err)

			req := httptest.NewRequest(tt.method, tt.target, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
			if tt.wantCookie != nil {
				found := false
				resp := rec.Result()
				for _, v := range resp.Cookies() {
					if v.Name == "access_token" {
						found = true
						require.Equal(t, "jwt-token", v.Value)
						break
					}
				}
				require.True(t, found, "access_token cookie not found")
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	cases := []struct {
		name       string
		user       *model.User
		mockSvc    *transport.ServiceMock
		wantCode   int
		wantCookie *string
	}{
		{
			name: "Positive - user login success",
			user: &model.User{
				UserName: "someName",
				Role:     "someRole",
				PassHash: "somePass",
			},
			mockSvc: &transport.ServiceMock{LoginUserFn: func(ctx context.Context, username string, password string, role string) (string, *model.User, error) {
				return "jwt-token", &model.User{
					UserName: "someName",
					Role:     "someRole",
					PassHash: "somePass",
				}, nil
			}},
			wantCode:   http.StatusOK,
			wantCookie: ptrMaker("jwt-token"),
		},
		{
			name: "Negative - incorrect role",
			user: &model.User{
				UserName: "someName",
				Role:     "someRole",
				PassHash: "somePass",
			},
			mockSvc: &transport.ServiceMock{LoginUserFn: func(ctx context.Context, username string, password string, role string) (string, *model.User, error) {
				return "", nil, model.ErrIncorrectUserRole
			}},
			wantCode:   http.StatusBadRequest,
			wantCookie: nil,
		},
		{
			name: "Negative - incorrect role",
			user: &model.User{
				UserName: "someName",
				Role:     "someRole",
				PassHash: "somePass",
			},
			mockSvc: &transport.ServiceMock{LoginUserFn: func(ctx context.Context, username string, password string, role string) (string, *model.User, error) {
				return "", nil, model.ErrCommon500
			}},
			wantCode:   http.StatusInternalServerError,
			wantCookie: nil,
		},
		{
			name: "Negative - empty credentials",
			user: &model.User{
				UserName: "someName",
				Role:     "someRole",
			},
			mockSvc:    &transport.ServiceMock{},
			wantCode:   http.StatusBadRequest,
			wantCookie: nil,
		},
		{
			name: "Negative - user not found",
			user: &model.User{
				UserName: "someName",
				Role:     "someRole",
				PassHash: "somePass",
			},
			mockSvc: &transport.ServiceMock{LoginUserFn: func(ctx context.Context, username string, password string, role string) (string, *model.User, error) {
				return "", nil, model.ErrUserNotFound
			}},
			wantCode:   http.StatusNotFound,
			wantCookie: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.user)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
			if tt.wantCookie != nil {
				found := false
				resp := rec.Result()
				for _, v := range resp.Cookies() {
					if v.Name == "access_token" {
						found = true
						require.Equal(t, "jwt-token", v.Value)
						break
					}
				}
				require.True(t, found, "access_token cookie not found")
			}
		})
	}
}

func TestCreateItem(t *testing.T) {
	cases := []struct {
		name     string
		item     *model.Item
		mockSvc  *transport.ServiceMock
		wantCode int
	}{
		{
			name: "Positive - item create success",
			item: &model.Item{
				Title:           "someTitle",
				Price:           300,
				Visible:         true,
				AvailableAmount: 100500,
			},
			mockSvc: &transport.ServiceMock{CreateItemFn: func(ctx context.Context, item *model.Item, role string) error {
				return nil
			}},
			wantCode: http.StatusCreated,
		},
		{
			name:     "Negative - nil item",
			item:     nil,
			mockSvc:  nil,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Negative - DB error",
			item: &model.Item{
				Title:           "someTitle",
				Price:           300,
				Visible:         true,
				AvailableAmount: 100500,
			},
			mockSvc: &transport.ServiceMock{CreateItemFn: func(ctx context.Context, item *model.Item, role string) error {
				return model.ErrCommon500
			}},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.item)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestGetItemByID(t *testing.T) {
	cases := []struct {
		name     string
		mockSvc  *transport.ServiceMock
		wantCode int
	}{
		{
			name: "Positive - item found",
			mockSvc: &transport.ServiceMock{GetItemByIDFn: func(ctx context.Context, id int, role string) (*model.Item, error) {
				return &model.Item{}, nil
			}},
			wantCode: http.StatusOK,
		},
		{
			name: "Negative - item not found",
			mockSvc: &transport.ServiceMock{GetItemByIDFn: func(ctx context.Context, id int, role string) (*model.Item, error) {
				return nil, model.ErrItemNotFound
			}},
			wantCode: http.StatusNotFound,
		},
		{
			name: "Negative - no access to see items",
			mockSvc: &transport.ServiceMock{GetItemByIDFn: func(ctx context.Context, id int, role string) (*model.Item, error) {
				return nil, model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
		{
			name: "Negative - DB error",
			mockSvc: &transport.ServiceMock{GetItemByIDFn: func(ctx context.Context, id int, role string) (*model.Item, error) {
				return nil, errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items/300", nil)
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestUpdateItem(t *testing.T) {
	cases := []struct {
		name     string
		item     *model.ItemUpdate
		mockSvc  *transport.ServiceMock
		wantCode int
	}{
		{
			name: "Positive - item updated",
			item: &model.ItemUpdate{},
			mockSvc: &transport.ServiceMock{UpdateItemByIDFn: func(ctx context.Context, item *model.ItemUpdate, role string) error {
				return nil
			}},
			wantCode: http.StatusNoContent,
		},
		{
			name: "Negative - nil item",
			item: nil,
			mockSvc: &transport.ServiceMock{UpdateItemByIDFn: func(ctx context.Context, item *model.ItemUpdate, role string) error {
				return model.ErrNoFieldsToUpdate
			}},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Negative - no access to update",
			item: &model.ItemUpdate{},
			mockSvc: &transport.ServiceMock{UpdateItemByIDFn: func(ctx context.Context, item *model.ItemUpdate, role string) error {
				return model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
		{
			name: "Negative - item id not found",
			item: &model.ItemUpdate{},
			mockSvc: &transport.ServiceMock{UpdateItemByIDFn: func(ctx context.Context, item *model.ItemUpdate, role string) error {
				return model.ErrItemNotFound
			}},
			wantCode: http.StatusNotFound,
		},
		{
			name: "Negative - DB error",
			item: &model.ItemUpdate{},
			mockSvc: &transport.ServiceMock{UpdateItemByIDFn: func(ctx context.Context, item *model.ItemUpdate, role string) error {
				return errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.item)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPatch, "/items/300", bytes.NewReader(body))
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestDeleteItem(t *testing.T) {
	cases := []struct {
		name     string
		mockSvc  *transport.ServiceMock
		wantCode int
	}{
		{
			name: "Positive - item updated",
			mockSvc: &transport.ServiceMock{DeleteItemByIDFn: func(ctx context.Context, id int, role string, username string) error {
				return nil
			}},
			wantCode: http.StatusNoContent,
		},
		{
			name: "Negative - no access to delete",
			mockSvc: &transport.ServiceMock{DeleteItemByIDFn: func(ctx context.Context, id int, role string, username string) error {
				return model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
		{
			name: "Negative - item id not found",
			mockSvc: &transport.ServiceMock{DeleteItemByIDFn: func(ctx context.Context, id int, role string, username string) error {
				return model.ErrItemNotFound
			}},
			wantCode: http.StatusNotFound,
		},
		{
			name: "Negative - DB error",
			mockSvc: &transport.ServiceMock{DeleteItemByIDFn: func(ctx context.Context, id int, role string, username string) error {
				return errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/items/300", nil)
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestGetItemsList(t *testing.T) {
	cases := []struct {
		name     string
		mockSvc  *transport.ServiceMock
		wantCode int
	}{
		{
			name: "Positive - items fetched",
			mockSvc: &transport.ServiceMock{GetItemsListFn: func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
				return []*model.Item{{}, {}}, nil
			}},
			wantCode: http.StatusOK,
		},
		{
			name: "Negative - DB error",
			mockSvc: &transport.ServiceMock{GetItemsListFn: func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
				return nil, errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "Negative - incorrect request params",
			mockSvc: &transport.ServiceMock{GetItemsListFn: func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
				return nil, model.ErrInvalidAscDesc
			}},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Negative - no access to get items",
			mockSvc: &transport.ServiceMock{GetItemsListFn: func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
				return nil, model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items", nil)
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestGetItemHistoryByID(t *testing.T) {
	cases := []struct {
		name     string
		mockSvc  *transport.ServiceMock
		wantCode int
		wantBody bool
	}{
		{
			name: "Positive - history fetched",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return []*model.ItemHistory{{}, {}}, nil
			}},
			wantCode: http.StatusOK,
			wantBody: true,
		},
		{
			name: "Negative - no access to see history",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
		{
			name: "Negative - DB error",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return nil, errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "Negative - item not found",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrItemNotFound
			}},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items/300/history", nil)
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
			if tt.wantBody {
				require.NotEqual(t, 0, rec.Body.Len(), "expected body, got empty")
			}
		})
	}
}

func TestGetItemsHistoryList(t *testing.T) {
	_, testHistory := generateValidItemAndHistoryArray(t)
	cases := []struct {
		name     string
		mockSvc  *transport.ServiceMock
		wantCode int
		wantBody bool
	}{
		{
			name: "Positive - history fetched",
			mockSvc: &transport.ServiceMock{GetItemHistoryAllFn: func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
				return testHistory, nil
			}},
			wantCode: http.StatusOK,
			wantBody: true,
		},
		{
			name: "Negative - Db error",
			mockSvc: &transport.ServiceMock{GetItemHistoryAllFn: func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
				return nil, errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "Negative - req params invalid",
			mockSvc: &transport.ServiceMock{GetItemHistoryAllFn: func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrInvalidAscDesc
			}},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Negative -  o access to see history",
			mockSvc: &transport.ServiceMock{GetItemHistoryAllFn: func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items/history", nil)
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			r := newTestServer(h)
			r.ServeHTTP(rec, req)

			require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

			if tt.wantBody {
				var hist []*model.ItemHistory
				err := json.Unmarshal(rec.Body.Bytes(), &hist)
				require.NoError(t, err, "failed to Unmarshal model.History from recorded body")
				require.Equal(t, len(testHistory), len(hist))
				for i := range len(testHistory) {
					require.Equal(t, *testHistory[i], *hist[i])
				}

			}

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestExportItemsHistoryCSV(t *testing.T) {
	_, testHistory := generateValidItemAndHistoryArray(t)
	cases := []struct {
		name     string
		mockSvc  *transport.ServiceMock
		wantCode int
		wantBody bool
	}{
		{
			name: "Positive - history fetched",
			mockSvc: &transport.ServiceMock{GetItemHistoryAllFn: func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
				return testHistory, nil
			}},
			wantCode: http.StatusOK,
			wantBody: true,
		},
		{
			name: "Negative - Db error",
			mockSvc: &transport.ServiceMock{GetItemHistoryAllFn: func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
				return nil, errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "Negative - req params invalid",
			mockSvc: &transport.ServiceMock{GetItemHistoryAllFn: func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrInvalidAscDesc
			}},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Negative - no access to see history",
			mockSvc: &transport.ServiceMock{GetItemHistoryAllFn: func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items/history/csv", nil)
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			newTestServer(h).ServeHTTP(rec, req)

			if tt.wantBody {
				require.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
			} else {
				require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
			}

			if tt.wantBody {
				rows := csv.NewReader(bytes.NewReader(rec.Body.Bytes()))
				records, err := rows.ReadAll()
				require.NoError(t, err, fmt.Sprintf("invalid csv: %v", err))
				require.NotEqual(t, 0, len(records), "expected at least one csv header-row")
			}

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestExportItemsCSV(t *testing.T) {
	testItem, _ := generateValidItemAndHistoryArray(t)
	cases := []struct {
		name     string
		mockSvc  *transport.ServiceMock
		wantCode int
		wantBody bool
	}{
		{
			name: "Positive - items fetched",
			mockSvc: &transport.ServiceMock{GetItemsListFn: func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
				return []*model.Item{testItem, testItem}, nil
			}},
			wantCode: http.StatusOK,
			wantBody: true,
		},
		{
			name: "Negative - Db error",
			mockSvc: &transport.ServiceMock{GetItemsListFn: func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
				return nil, errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "Negative - req params invalid",
			mockSvc: &transport.ServiceMock{GetItemsListFn: func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
				return nil, model.ErrInvalidAscDesc
			}},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Negative - no access to see items",
			mockSvc: &transport.ServiceMock{GetItemsListFn: func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
				return nil, model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items/csv", nil)
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			newTestServer(h).ServeHTTP(rec, req)

			if tt.wantBody {
				require.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
			} else {
				require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
			}

			if tt.wantBody {
				rows := csv.NewReader(bytes.NewReader(rec.Body.Bytes()))
				records, err := rows.ReadAll()
				require.NoError(t, err, fmt.Sprintf("invalid csv: %v", err))
				require.NotEqual(t, 0, len(records), "expected at least one csv header-row")
			}

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestExportItemIDHistoryCSV(t *testing.T) {
	_, testHistory := generateValidItemAndHistoryArray(t)
	cases := []struct {
		name     string
		mockSvc  *transport.ServiceMock
		wantCode int
		wantBody bool
	}{
		{
			name: "Positive - history fetched",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return testHistory, nil
			}},
			wantCode: http.StatusOK,
			wantBody: true,
		},
		{
			name: "Negative - Db error",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return nil, errors.New("test DB error")
			}},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "Negative - req params invalid",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrInvalidAscDesc
			}},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Negative - no access to see history",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrAccessDenied
			}},
			wantCode: http.StatusForbidden,
		},
		{
			name: "Negative - item ID not found",
			mockSvc: &transport.ServiceMock{GetItemHistoryByIDFn: func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
				return nil, model.ErrItemNotFound
			}},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items/300/history/csv", nil)
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: "jwt-token",
			})

			rec := httptest.NewRecorder()

			h := transport.NewWHCHandlers(tt.mockSvc)
			newTestServer(h).ServeHTTP(rec, req)

			if tt.wantBody {
				require.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
			} else {
				require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
			}

			if tt.wantBody {
				rows := csv.NewReader(bytes.NewReader(rec.Body.Bytes()))
				records, err := rows.ReadAll()
				require.NoError(t, err, fmt.Sprintf("invalid csv: %v", err))
				require.NotEqual(t, 0, len(records), "expected at least one csv header-row")
			}

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

// =================== helpers ==================

func ptrMaker[T int | string | int64 | bool](input T) *T {
	return &input
}

func newTestServer(h *transport.WHCHandlers) *ginext.Engine {
	c := config.New()
	c.SetDefault("GIN_MODE", "testMode")
	c.SetDefault("SECRET", "TEST_SECRET")
	_, r := engine.NewServerEngine(c, h, "TEST")
	return r
}

func generateValidItemAndHistoryArray(t *testing.T) (*model.Item, []*model.ItemHistory) {
	testItem := model.Item{
		ID:              300,
		Title:           "testTitle",
		Description:     "testDescr",
		Price:           100500,
		Visible:         true,
		AvailableAmount: 300,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		UpdatedBy:       "testUser",
		DeletedAt:       nil,
	}

	rawBytes, err := json.Marshal(testItem)
	require.NoError(t, err, "failed to Marshal testItem into testBytes")
	testBytes := json.RawMessage(rawBytes)

	testHistory := model.ItemHistory{
		ID:        300,
		ItemID:    300,
		Version:   1,
		Action:    "TEST",
		ChangedAt: time.Now().UTC(),
		ChangedBy: "TEST",
		OldData:   nil,
		NewData:   &testBytes,
	}

	testArray := []*model.ItemHistory{}
	testArray = append(testArray, &testHistory, &testHistory)

	return &testItem, testArray
}
