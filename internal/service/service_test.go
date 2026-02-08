package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// =============== METHODS TESTS ================
func TestCreateItem(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		policy  policyMock
		repoErr error
		item    *model.Item
		wantErr error
	}{
		{
			name:    "Negative - access denied",
			policy:  policyMock{canCreate: false},
			item:    &model.Item{Title: "x"},
			wantErr: model.ErrAccessDenied,
		},
		{
			name:    "Negative - validation error",
			policy:  policyMock{canCreate: true},
			item:    &model.Item{Title: ""},
			wantErr: model.ErrEmptyTitle,
		},
		{
			name:    "Negative - repo error",
			policy:  policyMock{canCreate: true},
			item:    &model.Item{Title: "ok"},
			repoErr: errors.New("db error"),
			wantErr: model.ErrCommon500,
		},
		{
			name:    "Positive - success",
			policy:  policyMock{canCreate: true},
			item:    &model.Item{Title: "ok"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &repoMock{
				CreateItemFn: func(ctx context.Context, item *model.Item) error {
					return tt.repoErr
				},
			}

			svc := WHCService{
				repo:   repo,
				policy: tt.policy,
			}

			err := svc.CreateItem(ctx, tt.item, "admin")
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestUpdateItemByID(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		item    *model.ItemUpdate
		repo    *repoMock
		policy  policyMock
		role    string
		wantErr error
	}{
		{
			name: "Positive - update success",
			item: &model.ItemUpdate{
				ID:        1,
				Title:     ptrMaker("new"),
				UpdatedBy: "someone",
			},
			repo: &repoMock{
				UpdateItemFn: func(ctx context.Context, item *model.ItemUpdate, seeDeleted bool) error { return nil },
			},
			policy:  policyMock{canUpdate: true},
			role:    "some role",
			wantErr: nil,
		},
		{
			name: "Negative - item not found",
			item: &model.ItemUpdate{
				ID:        1,
				Title:     ptrMaker("new"),
				UpdatedBy: "someone",
			},
			repo: &repoMock{
				UpdateItemFn: func(ctx context.Context, item *model.ItemUpdate, seeDeleted bool) error { return model.ErrItemNotFound },
			},
			policy:  policyMock{canUpdate: true},
			role:    "some role",
			wantErr: model.ErrItemNotFound,
		},
		{
			name: "Negative - DB error",
			item: &model.ItemUpdate{
				ID:        1,
				Title:     ptrMaker("new"),
				UpdatedBy: "someone",
			},
			repo: &repoMock{
				UpdateItemFn: func(ctx context.Context, item *model.ItemUpdate, seeDeleted bool) error {
					return errors.New("test DB error")
				},
			},
			policy:  policyMock{canUpdate: true},
			role:    "some role",
			wantErr: model.ErrCommon500,
		},
		{
			name: "Negative - nothing to update",
			item: &model.ItemUpdate{
				ID:        1,
				UpdatedBy: "someone",
			},
			repo:    nil,
			policy:  policyMock{canUpdate: true},
			role:    "some role",
			wantErr: model.ErrNoFieldsToUpdate,
		},
		{
			name: "Negative - incorrect item ID",
			item: &model.ItemUpdate{
				ID:        -100,
				UpdatedBy: "someone",
			},
			repo:    nil,
			policy:  policyMock{canUpdate: true},
			role:    "some role",
			wantErr: model.ErrIncorrectItemID,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := WHCService{
				repo:   tt.repo,
				policy: tt.policy,
			}

			err := svc.UpdateItemByID(ctx, tt.item, tt.role)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestGetItemByID(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		itemID  int
		repo    *repoMock
		policy  policyMock
		role    string
		wantErr error
	}{
		{
			name:   "Positive - item ID found",
			itemID: 5,
			policy: policyMock{canGetItems: true, canSeeDeleted: true},
			repo: &repoMock{GetItemByIDFn: func(ctx context.Context, id int, seeDeleted bool) (*model.Item, error) {
				return &model.Item{ID: 5}, nil
			}},
			role:    "some role",
			wantErr: nil,
		},
		{
			name:    "Negative - item ID incorrect",
			itemID:  -5,
			policy:  policyMock{canGetItems: true, canSeeDeleted: true},
			repo:    nil,
			role:    "some role",
			wantErr: model.ErrIncorrectItemID,
		},
		{
			name:    "Negative - unknown role - access denied",
			itemID:  5,
			policy:  policyMock{canGetItems: false},
			repo:    nil,
			role:    "some role",
			wantErr: model.ErrAccessDenied,
		},
		{
			name:   "Negative - no access to see deleted",
			itemID: 5,
			policy: policyMock{canGetItems: true, canSeeDeleted: false},
			repo: &repoMock{GetItemByIDFn: func(ctx context.Context, id int, seeDeleted bool) (*model.Item, error) {
				return nil, nil
			}},
			role:    "some role",
			wantErr: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := WHCService{
				repo:   tt.repo,
				policy: tt.policy,
			}

			_, err := svc.GetItemByID(ctx, tt.itemID, tt.role)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestDeleteItemByID(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		itemID   int
		repo     *repoMock
		policy   policyMock
		role     string
		username string
		wantErr  error
	}{
		{
			name:   "Positive - delete success",
			itemID: 1,
			repo: &repoMock{
				DeleteItemFn: func(ctx context.Context, id int, username string) error { return nil },
			},
			policy:   policyMock{canDelete: true},
			role:     "some role",
			username: "someName",
			wantErr:  nil,
		},
		{
			name:   "Negative - user not found",
			itemID: 1,
			repo: &repoMock{
				DeleteItemFn: func(ctx context.Context, id int, username string) error { return model.ErrItemNotFound },
			},
			policy:   policyMock{canDelete: true},
			role:     "some role",
			username: "someName",
			wantErr:  model.ErrItemNotFound,
		},
		{
			name:   "Negative - DB error",
			itemID: 1,
			repo: &repoMock{
				DeleteItemFn: func(ctx context.Context, id int, username string) error {
					return errors.New("test DB error")
				},
			},
			policy:   policyMock{canDelete: true},
			role:     "some role",
			username: "someName",
			wantErr:  model.ErrCommon500,
		},
		{
			name:     "Negative - incorrect item ID",
			itemID:   -300,
			repo:     nil,
			policy:   policyMock{canDelete: true},
			role:     "some role",
			username: "someName",
			wantErr:  model.ErrIncorrectItemID,
		},
		{
			name:     "Negative - no access to delete",
			itemID:   1,
			repo:     nil,
			policy:   policyMock{canDelete: false},
			role:     "some role",
			username: "someName",
			wantErr:  model.ErrAccessDenied,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := WHCService{
				repo:   tt.repo,
				policy: tt.policy,
			}

			err := svc.DeleteItemByID(ctx, tt.itemID, tt.role, tt.username)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCreateUser(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		user    *model.User
		repo    *repoMock
		jwt     *jwtMock
		policy  policyMock
		wantErr error
	}{
		{
			name: "Positive - user create success",
			user: &model.User{
				UserName: "string",
				Role:     "string",
			},
			repo:    &repoMock{CreateUserFn: func(ctx context.Context, u *model.User) error { return nil }},
			jwt:     &jwtMock{token: "jwt-token"},
			policy:  policyMock{correctRole: true},
			wantErr: nil,
		},
		{
			name: "Negative - incorrect role",
			user: &model.User{
				UserName: "string",
				Role:     "string",
			},
			repo:    nil,
			jwt:     nil,
			policy:  policyMock{correctRole: false},
			wantErr: model.ErrIncorrectUserRole,
		},
		{
			name: "Negative - some DB error",
			user: &model.User{
				UserName: "string",
				Role:     "string",
			},
			repo:    &repoMock{CreateUserFn: func(ctx context.Context, u *model.User) error { return errors.New("some db error") }},
			jwt:     nil,
			policy:  policyMock{correctRole: true},
			wantErr: model.ErrCommon500,
		},
		{
			name: "Negative - unique violation error",
			user: &model.User{
				UserName: "string",
				Role:     "string",
			},
			repo:    &repoMock{CreateUserFn: func(ctx context.Context, u *model.User) error { return errors.New("blabla unique violation blabla") }},
			jwt:     nil,
			policy:  policyMock{correctRole: true},
			wantErr: model.ErrUserAlreadyExists,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := WHCService{
				repo:       tt.repo,
				jwtManager: tt.jwt,
				policy:     tt.policy,
			}

			token, err := svc.CreateUser(ctx, tt.user)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.Equal(t, "jwt-token", token)
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	ctx := context.Background()
	testPass := "youShallNotPass!"
	testHash, err := bcrypt.GenerateFromPassword([]byte(testPass), bcrypt.DefaultCost)
	require.NoError(t, err)

	cases := []struct {
		name     string
		userName string
		password string
		role     string
		repo     *repoMock
		policy   *policyMock
		jwt      *jwtMock
		wantErr  error
	}{
		{
			name:     "Positive - user login success",
			userName: "someName",
			password: testPass,
			role:     "some role",
			policy:   &policyMock{correctRole: true},
			repo: &repoMock{GetUserByNameFn: func(ctx context.Context, username string) (*model.User, error) {
				return &model.User{
					ID:       1,
					UserName: "someName",
					Role:     "some role",
					PassHash: string(testHash),
				}, nil
			}},
			jwt:     &jwtMock{token: "jwt-token"},
			wantErr: nil,
		},
		{
			name:     "Negative - incorrect role",
			userName: "someName",
			password: testPass,
			role:     "some role",
			policy:   &policyMock{correctRole: false},
			repo: &repoMock{GetUserByNameFn: func(ctx context.Context, username string) (*model.User, error) {
				return &model.User{
					ID:       1,
					UserName: "someName",
					Role:     "some role",
					PassHash: string(testHash),
				}, nil
			}},
			jwt:     nil,
			wantErr: model.ErrIncorrectUserRole,
		},
		{
			name:     "Negative - incorrect password",
			userName: "someName",
			password: "incorrectPass",
			role:     "some role",
			policy:   &policyMock{correctRole: true},
			repo: &repoMock{GetUserByNameFn: func(ctx context.Context, username string) (*model.User, error) {
				return &model.User{
					ID:       1,
					UserName: "someName",
					Role:     "some role",
					PassHash: string(testHash),
				}, nil
			}},
			jwt:     nil,
			wantErr: model.ErrInvalidCredentials,
		},
		{
			name:     "Negative - some DB error",
			userName: "someName",
			password: "incorrectPass",
			role:     "some role",
			policy:   &policyMock{correctRole: true},
			repo: &repoMock{GetUserByNameFn: func(ctx context.Context, username string) (*model.User, error) {
				return nil, errors.New("test DB error")
			}},
			jwt:     nil,
			wantErr: model.ErrCommon500,
		},
		{
			name:     "Negative - user not found",
			userName: "someName",
			password: "incorrectPass",
			role:     "some role",
			policy:   &policyMock{correctRole: true},
			repo: &repoMock{GetUserByNameFn: func(ctx context.Context, username string) (*model.User, error) {
				return nil, model.ErrUserNotFound
			}},
			jwt:     nil,
			wantErr: model.ErrUserNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := WHCService{
				repo:       tt.repo,
				jwtManager: tt.jwt,
				policy:     tt.policy,
			}

			token, _, err := svc.LoginUser(ctx, tt.userName, tt.password, tt.role)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.Equal(t, "jwt-token", token)
			}
		})
	}
}

func TestGetItemsList(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		repo    *repoMock
		rpi     *model.RequestParam
		policy  *policyMock
		resLen  int
		wantErr error
	}{
		{
			name: "Positive - 2 items are fetched in array",
			repo: &repoMock{GetItemsListFn: func(ctx context.Context, rp *model.RequestParam, seeDeleted bool) ([]*model.Item, error) {
				return []*model.Item{{}, {}}, nil
			}},
			rpi:     &model.RequestParam{},
			resLen:  2,
			policy:  &policyMock{canGetItems: true, canSeeDeleted: true},
			wantErr: nil,
		},
		{
			name: "Negative - DB error",
			repo: &repoMock{GetItemsListFn: func(ctx context.Context, rp *model.RequestParam, seeDeleted bool) ([]*model.Item, error) {
				return nil, errors.New("some DB error")
			}},
			rpi:     &model.RequestParam{},
			policy:  &policyMock{canGetItems: true, canSeeDeleted: true},
			wantErr: model.ErrCommon500,
		},
		{
			name:    "Negative - no access to get items",
			repo:    nil,
			rpi:     &model.RequestParam{},
			policy:  &policyMock{canGetItems: false, canSeeDeleted: true},
			wantErr: model.ErrAccessDenied,
		},
		{
			name:    "Negative - nil RequestParam",
			repo:    nil,
			rpi:     nil,
			policy:  &policyMock{canGetItems: true, canSeeDeleted: true},
			wantErr: model.ErrInvalidRequestParam,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := WHCService{
				repo:   tt.repo,
				policy: tt.policy,
			}

			res, err := svc.GetItemsList(ctx, tt.rpi, "role")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.Equal(t, tt.resLen, len(res))
			}
		})
	}
}

func TestGetItemHistoryByID(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		repo    *repoMock
		policy  *policyMock
		rph     *model.RequestParam
		itemID  int
		role    string
		wantErr error
	}{
		{
			name: "Positive - history fetched",
			repo: &repoMock{GetItemHistoryByIDFn: func(ctx context.Context, rp *model.RequestParam, id int) ([]*model.ItemHistory, error) {
				return []*model.ItemHistory{{}, {}}, nil
			}},
			policy:  &policyMock{canGetHistory: true},
			rph:     &model.RequestParam{},
			itemID:  300,
			role:    "some role",
			wantErr: nil,
		},
		{
			name:    "Negative - no access to get history",
			repo:    nil,
			policy:  &policyMock{canGetHistory: false},
			rph:     &model.RequestParam{},
			itemID:  300,
			role:    "some role",
			wantErr: model.ErrAccessDenied,
		},
		{
			name: "Negative - DB error",
			repo: &repoMock{GetItemHistoryByIDFn: func(ctx context.Context, rp *model.RequestParam, id int) ([]*model.ItemHistory, error) {
				return nil, errors.New("some DB error")
			}},
			policy:  &policyMock{canGetHistory: true},
			rph:     &model.RequestParam{},
			itemID:  300,
			role:    "some role",
			wantErr: model.ErrCommon500,
		},
		{
			name:    "Negative - invalid ID",
			repo:    nil,
			policy:  &policyMock{canGetHistory: true},
			rph:     &model.RequestParam{},
			itemID:  -300,
			role:    "some role",
			wantErr: model.ErrIncorrectItemID,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := WHCService{
				repo:   tt.repo,
				policy: tt.policy,
			}

			res, err := svc.GetItemHistoryByID(ctx, tt.rph, tt.itemID, tt.role)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NotEqual(t, nil, res)
			}
		})
	}
}

func TestGetItemHistoryAll(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		repo    *repoMock
		policy  *policyMock
		rph     *model.RequestParam
		role    string
		wantErr error
	}{
		{
			name: "Positive - history fetched",
			repo: &repoMock{GetItemHistoryAllFn: func(ctx context.Context, rp *model.RequestParam) ([]*model.ItemHistory, error) {
				return []*model.ItemHistory{{}, {}}, nil
			}},
			policy:  &policyMock{canGetHistory: true},
			rph:     &model.RequestParam{},
			role:    "some role",
			wantErr: nil,
		},
		{
			name:    "Negative - no access to get history",
			repo:    nil,
			policy:  &policyMock{canGetHistory: false},
			rph:     &model.RequestParam{},
			role:    "some role",
			wantErr: model.ErrAccessDenied,
		},
		{
			name: "Negative - DB error",
			repo: &repoMock{GetItemHistoryAllFn: func(ctx context.Context, rp *model.RequestParam) ([]*model.ItemHistory, error) {
				return nil, errors.New("some DB error")
			}},
			policy:  &policyMock{canGetHistory: true},
			rph:     &model.RequestParam{},
			role:    "some role",
			wantErr: model.ErrCommon500,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := WHCService{
				repo:   tt.repo,
				policy: tt.policy,
			}

			res, err := svc.GetItemHistoryAll(ctx, tt.rph, tt.role)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NotEqual(t, nil, res)
			}
		})
	}
}

// =============== TOOLS TESTS ================

func TestValidateReqParams(t *testing.T) {
	testStart := time.Now()
	testEnd := testStart.Add(5 * time.Minute)

	cases := []struct {
		name    string
		rp      *model.RequestParam
		wantErr error
	}{
		{
			name: "Positive - full valid HISTORY reqParam",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.HistoryOrderByID),
				ASC:       true,
				DESC:      false,
				StartTime: &testStart,
				EndTime:   &testEnd,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: nil,
		},
		{
			name: "Positive - full valid ITEM reqParam",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByPrice),
				ASC:       true,
				DESC:      false,
				StartTime: &testStart,
				EndTime:   &testEnd,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: nil,
		},
		{
			name:    "Negative - nil reqParam",
			rp:      nil,
			wantErr: model.ErrInvalidRequestParam,
		},
		{
			name: "Negative - ASC=DESC",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByPrice),
				ASC:       true,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   &testEnd,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: model.ErrInvalidAscDesc,
		},
		{
			name: "Negative - incorrect OrderBy",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker("some order"),
				ASC:       false,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   &testEnd,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: model.ErrInvalidOrderBy,
		},
		{
			name: "Positive - nil OrderBy",
			rp: &model.RequestParam{
				OrderBy:   nil,
				ASC:       true,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   &testEnd,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: nil,
		},
		{
			name: "Negative - START after END",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: &testEnd,
				EndTime:   &testStart,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: model.ErrInvalidStartEndTime,
		},
		{
			name: "Positive - START ok, END nil",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   nil,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: nil,
		},
		{
			name: "Positive - START nil, END ok",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: nil,
				EndTime:   &testEnd,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: nil,
		},
		{
			name: "Positive - PAGE nil, LIMIT nil",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   &testEnd,
				Page:      nil,
				Limit:     nil,
			},
			wantErr: nil,
		},
		{
			name: "Negative - PAGE ok, LIMIT nil",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   nil,
				Page:      ptrMaker(5),
				Limit:     nil,
			},
			wantErr: model.ErrInvalidLimit,
		},
		{
			name: "Positive - PAGE nil",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   nil,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(100),
			},
			wantErr: nil,
		},
		{
			name: "Negative - LIMIT too big",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   nil,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(1001),
			},
			wantErr: model.ErrInvalidLimit,
		},
		{
			name: "Negative - negative LIMIT",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   nil,
				Page:      ptrMaker(5),
				Limit:     ptrMaker(-300),
			},
			wantErr: model.ErrInvalidLimit,
		},
		{
			name: "Negative - negative PAGE",
			rp: &model.RequestParam{
				OrderBy:   ptrMaker(model.ItemsOrderByID),
				ASC:       false,
				DESC:      true,
				StartTime: &testStart,
				EndTime:   nil,
				Page:      ptrMaker(-5),
				Limit:     ptrMaker(300),
			},
			wantErr: model.ErrInvalidPage,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := validateReqParams(tt.rp)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateItemUpdate(t *testing.T) {
	cases := []struct {
		name    string
		item    *model.ItemUpdate
		wantErr error
	}{
		{
			name: "Positive - full valid Item",
			item: &model.ItemUpdate{
				Title:           ptrMaker("title"),
				Price:           ptrMaker(int64(100500)),
				Visible:         ptrMaker(true),
				AvailableAmount: ptrMaker(500),
				UpdatedBy:       "someone",
			},
			wantErr: nil,
		},
		{
			name:    "Negative - nil Item",
			item:    nil,
			wantErr: model.ErrNoFieldsToUpdate,
		},
		{
			name: "Negative - all subfields nil",
			item: &model.ItemUpdate{
				Title:           nil,
				Price:           nil,
				Visible:         nil,
				AvailableAmount: nil,
				UpdatedBy:       "",
			},
			wantErr: model.ErrNoFieldsToUpdate,
		},
		{
			name: "Negative - Title empty string",
			item: &model.ItemUpdate{
				Title:           ptrMaker(""),
				Price:           nil,
				Visible:         nil,
				AvailableAmount: nil,
				UpdatedBy:       "",
			},
			wantErr: model.ErrEmptyTitle,
		},
		{
			name: "Negative - Price negative",
			item: &model.ItemUpdate{
				Title:           ptrMaker("title"),
				Price:           ptrMaker(int64(-100500)),
				Visible:         ptrMaker(true),
				AvailableAmount: ptrMaker(500),
				UpdatedBy:       "someone",
			},
			wantErr: model.ErrInvalidPrice,
		},
		{
			name: "Negative - AvailAmount negative",
			item: &model.ItemUpdate{
				Title:           ptrMaker("title"),
				Price:           ptrMaker(int64(100500)),
				Visible:         ptrMaker(true),
				AvailableAmount: ptrMaker(-300),
				UpdatedBy:       "someone",
			},
			wantErr: model.ErrInvalidAvail,
		},
		{
			name: "Negative - UpdatedBy empty string",
			item: &model.ItemUpdate{
				Title:           ptrMaker("title"),
				Price:           ptrMaker(int64(100500)),
				Visible:         ptrMaker(true),
				AvailableAmount: ptrMaker(300),
				UpdatedBy:       "",
			},
			wantErr: model.ErrIncorrectUserName,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := validateItemUpdate(tt.item)

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateItem(t *testing.T) {
	cases := []struct {
		name    string
		item    *model.Item
		wantErr error
	}{
		{
			name: "Positive - valid Item",
			item: &model.Item{
				Title:           "title",
				AvailableAmount: 300,
				Price:           100500,
			},
			wantErr: nil,
		},
		{
			name:    "Negative - nil Item",
			item:    nil,
			wantErr: model.ErrEmptyItemInfo,
		},
		{
			name: "Negative - Title empty string",
			item: &model.Item{
				Title:           "",
				AvailableAmount: 300,
				Price:           100500,
			},
			wantErr: model.ErrEmptyTitle,
		},
		{
			name: "Negative - Price negative",
			item: &model.Item{
				Title:           "title",
				AvailableAmount: 300,
				Price:           -100500,
			},
			wantErr: model.ErrInvalidPrice,
		},
		{
			name: "Negative - AvailAmount negative",
			item: &model.Item{
				Title:           "title",
				AvailableAmount: -300,
				Price:           100500,
			},
			wantErr: model.ErrInvalidAvail,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := validateItem(tt.item)

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateNormalizeUser(t *testing.T) {
	cases := []struct {
		name    string
		user    *model.User
		wantErr error
	}{
		{
			name: "Positive - valid user",
			user: &model.User{
				UserName: "SOMENAME",
				PassHash: "somePass",
			},
			wantErr: nil,
		},
		{
			name:    "Negative - nil user",
			user:    nil,
			wantErr: model.ErrEmptyUser,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNormalizeNewUser(tt.user)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NotEqual(t, "", tt.user.PassHash)
			}
		})
	}
}

// ============== helpers ===============
func ptrMaker[T int | string | int64 | bool](input T) *T {
	return &input
}
