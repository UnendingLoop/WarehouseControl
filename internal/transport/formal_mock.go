package transport

import (
	"context"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
)

type ServiceMock struct {
	CreateItemFn     func(ctx context.Context, item *model.Item, role string) error
	GetItemByIDFn    func(ctx context.Context, id int, role string) (*model.Item, error)
	UpdateItemByIDFn func(ctx context.Context, item *model.ItemUpdate, role string) error
	DeleteItemByIDFn func(ctx context.Context, id int, role, username string) error

	CreateUserFn func(ctx context.Context, user *model.User) (string, error)
	LoginUserFn  func(ctx context.Context, username string, password string, role string) (string, *model.User, error)

	GetItemsListFn       func(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error)
	GetItemHistoryByIDFn func(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error)
	GetItemHistoryAllFn  func(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error)
}

func (sm *ServiceMock) CreateItem(ctx context.Context, item *model.Item, role string) error {
	return sm.CreateItemFn(ctx, item, role)
}

func (sm *ServiceMock) GetItemByID(ctx context.Context, id int, role string) (*model.Item, error) {
	return sm.GetItemByIDFn(ctx, id, role)
}

func (sm *ServiceMock) UpdateItemByID(ctx context.Context, item *model.ItemUpdate, role string) error {
	return sm.UpdateItemByIDFn(ctx, item, role)
}

func (sm *ServiceMock) DeleteItemByID(ctx context.Context, id int, role, username string) error {
	return sm.DeleteItemByIDFn(ctx, id, role, username)
}

func (sm *ServiceMock) CreateUser(ctx context.Context, user *model.User) (string, error) {
	return sm.CreateUserFn(ctx, user)
}

func (sm *ServiceMock) LoginUser(ctx context.Context, username string, password string, role string) (string, *model.User, error) {
	return sm.LoginUserFn(ctx, username, password, role)
}

func (sm *ServiceMock) GetItemsList(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
	return sm.GetItemsListFn(ctx, rpi, role)
}

func (sm *ServiceMock) GetItemHistoryByID(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
	return sm.GetItemHistoryByIDFn(ctx, rph, id, role)
}

func (sm *ServiceMock) GetItemHistoryAll(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
	return sm.GetItemHistoryAllFn(ctx, rph, role)
}
