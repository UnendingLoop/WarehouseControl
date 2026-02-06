package service

import (
	"context"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"github.com/UnendingLoop/WarehouseControl/internal/mwauthlog"
)

type repoMock struct {
	CreateItemFn         func(ctx context.Context, item *model.Item) error
	GetItemByIDFn        func(ctx context.Context, id int, seeDeleted bool) (*model.Item, error)
	UpdateItemFn         func(ctx context.Context, item *model.ItemUpdate, seeDeleted bool) error
	DeleteItemFn         func(ctx context.Context, itemID int, username string) error
	CreateUserFn         func(ctx context.Context, user *model.User) error
	GetUserByNameFn      func(ctx context.Context, username string) (*model.User, error)
	GetItemsListFn       func(ctx context.Context, rp *model.RequestParam, seeDeleted bool) ([]*model.Item, error)
	GetItemHistoryByIDFn func(ctx context.Context, rp *model.RequestParam, id int) ([]*model.ItemHistory, error)
	GetItemHistoryAllFn  func(ctx context.Context, rp *model.RequestParam) ([]*model.ItemHistory, error)
}

func (m *repoMock) CreateItem(ctx context.Context, item *model.Item) error {
	return m.CreateItemFn(ctx, item)
}

func (m *repoMock) GetItemByID(ctx context.Context, id int, seeDeleted bool) (*model.Item, error) {
	return m.GetItemByIDFn(ctx, id, seeDeleted)
}

func (m *repoMock) UpdateItem(ctx context.Context, item *model.ItemUpdate, seeDeleted bool) error {
	return m.UpdateItemFn(ctx, item, seeDeleted)
}

func (m *repoMock) DeleteItem(ctx context.Context, itemID int, username string) error {
	return m.DeleteItemFn(ctx, itemID, username)
}

func (m *repoMock) CreateUser(ctx context.Context, user *model.User) error {
	return m.CreateUserFn(ctx, user)
}

func (m *repoMock) GetUserByName(ctx context.Context, username string) (*model.User, error) {
	return m.GetUserByNameFn(ctx, username)
}

func (m *repoMock) GetItemsList(ctx context.Context, rp *model.RequestParam, seeDeleted bool) ([]*model.Item, error) {
	return m.GetItemsListFn(ctx, rp, seeDeleted)
}

func (m *repoMock) GetItemHistoryByID(ctx context.Context, rp *model.RequestParam, id int) ([]*model.ItemHistory, error) {
	return m.GetItemHistoryByIDFn(ctx, rp, id)
}

func (m *repoMock) GetItemHistoryAll(ctx context.Context, rp *model.RequestParam) ([]*model.ItemHistory, error) {
	return m.GetItemHistoryAllFn(ctx, rp)
}

//=========================================================

type policyMock struct {
	canCreate     bool
	canUpdate     bool
	canDelete     bool
	canGetItems   bool
	canGetHistory bool
	canSeeDeleted bool
	correctRole   bool
}

func (p policyMock) AccessToCreate(string) bool     { return p.canCreate }
func (p policyMock) AccessToUpdate(string) bool     { return p.canUpdate }
func (p policyMock) AccessToDelete(string) bool     { return p.canDelete }
func (p policyMock) AccessToGetItems(string) bool   { return p.canGetItems }
func (p policyMock) AccessToGetHistory(string) bool { return p.canGetHistory }
func (p policyMock) AccessToSeeDeleted(string) bool { return p.canSeeDeleted }
func (p policyMock) IsCorrectRole(role string) bool { return p.correctRole }

//=========================================================

type jwtMock struct {
	token  string
	err    error
	claims *mwauthlog.Claims
}

func (j *jwtMock) Generate(id int, username, role string) (string, error) {
	return j.token, j.err
}

func (j *jwtMock) Parse(tokenStr string) (*mwauthlog.Claims, error) {
	return j.claims, j.err
}
