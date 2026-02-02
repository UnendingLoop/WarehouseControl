// Package service provides all methods for app
package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func (svc WHCService) CreateItem(ctx context.Context, item *model.Item, role string) error {
	rid := model.RequestIDFromCtx(ctx)

	if !svc.policy.AccessToCreate(role) {
		return model.ErrAccessDenied
	}

	if err := validateItem(item); err != nil {
		return err // 400
	}

	if err := svc.repo.CreateItem(ctx, item); err != nil {
		log.Printf("RID %q Failed to create new item in DB in 'CreateItem': %v", rid, err)
		return model.ErrCommon500
	}

	return nil
}

func (svc WHCService) GetItemByID(ctx context.Context, id int, role string) (*model.Item, error) {
	rid := model.RequestIDFromCtx(ctx)

	if id <= 0 {
		return nil, model.ErrIncorrectItemID
	}

	if !svc.policy.AccessToGetItems(role) {
		return nil, model.ErrAccessDenied
	}

	res, err := svc.repo.GetItemByID(ctx, id, svc.policy.AccessToSeeDeleted(role))
	if err != nil {
		switch {
		case errors.Is(err, model.ErrUserNotFound):
			return nil, err
		default:
			log.Printf("RID %q Failed to get item from DB in 'GetItemByID': %q", rid, err)
			return nil, model.ErrCommon500
		}
	}

	return res, nil
}

func (svc WHCService) UpdateItemByID(ctx context.Context, item *model.ItemUpdate, role string) error {
	rid := model.RequestIDFromCtx(ctx)

	if item.ID <= 0 {
		return model.ErrIncorrectItemID
	}

	if !svc.policy.AccessToUpdate(role) {
		return model.ErrAccessDenied
	}

	if err := validateItemUpdate(item); err != nil {
		return err // 400
	}

	if err := svc.repo.UpdateItem(ctx, item, svc.policy.AccessToSeeDeleted(role)); err != nil {
		switch {
		case errors.Is(err, model.ErrItemNotFound):
			return err
		default:
			log.Printf("RID %q Failed to update item in DB in 'UpdateItemByID': %q", rid, err)
			return model.ErrCommon500
		}
	}

	return nil
}

func (svc WHCService) DeleteItemByID(ctx context.Context, id int, role, username string) error {
	rid := model.RequestIDFromCtx(ctx)

	if id <= 0 {
		return model.ErrIncorrectItemID
	}

	if !svc.policy.AccessToDelete(role) {
		return model.ErrAccessDenied
	}

	if err := svc.repo.DeleteItem(ctx, id); err != nil {
		switch {
		case errors.Is(err, model.ErrUserNotFound):
			return err
		default:
			log.Printf("RID %q Failed to delete item in DB in 'DeleteItemByID': %q", rid, err)
			return model.ErrCommon500
		}
	}
	return nil
}

func (svc WHCService) CreateUser(ctx context.Context, user *model.User) (string, error) {
	rid := model.RequestIDFromCtx(ctx)

	// валидируем инфу о пользователе
	if err := validateNormalizeUser(user); err != nil {
		return "", err
	}

	// создаем его в бд
	err := svc.repo.CreateUser(ctx, user)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "unique violation"):
			return "", model.ErrUserAlreadyExists
		default:
			log.Printf("RID %q Failed to put new user to DB in 'CreateUser': %q", rid, err)
			return "", model.ErrCommon500
		}
	}

	// сразу генерируем токен авторизации
	token, err := svc.jwtManager.Generate(user.ID, user.UserName, user.Role)
	if err != nil {
		return "", model.ErrCommon500
	}

	return token, nil
}

func (svc WHCService) LoginUser(ctx context.Context, username string, password string, role string) (string, *model.User, error) {
	rid := model.RequestIDFromCtx(ctx)

	// получаем инфу о пользователе из БД
	user, err := svc.repo.GetUserByName(ctx, username)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrUserNotFound):
			return "", nil, err
		default:
			log.Printf("RID %q Failed to get user from DB in 'LoginUser': %q", rid, err)
			return "", nil, model.ErrCommon500
		}
	}

	// сравниваем предоставленный пароль с хранимым хэшом
	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(password)); err != nil {
		return "", nil, model.ErrInvalidCredentials
	}

	// генерируем токен авторизации
	token, err := svc.jwtManager.Generate(user.ID, user.UserName, role)
	if err != nil {
		return "", nil, model.ErrCommon500
	}

	return token, user, nil
}

func (svc WHCService) GetItemsList(ctx context.Context, rpi *model.RequestParam, role string) ([]*model.Item, error) {
	rid := model.RequestIDFromCtx(ctx)

	if !svc.policy.AccessToGetItems(role) {
		return nil, model.ErrAccessDenied
	}

	if err := validateReqParams(rpi); err != nil {
		return nil, err
	}

	res, err := svc.repo.GetItemsList(ctx, rpi, svc.policy.AccessToSeeDeleted(role))
	if err != nil {
		log.Printf("RID %q Failed to get items list from DB in 'GetItemsList': %q", rid, err)
		return nil, model.ErrCommon500
	}

	return res, nil
}

func (svc WHCService) GetItemHistoryByID(ctx context.Context, rph *model.RequestParam, id int, role string) ([]*model.ItemHistory, error) {
	rid := model.RequestIDFromCtx(ctx)

	if id <= 0 {
		return nil, model.ErrIncorrectItemID
	}

	if !svc.policy.AccessToGetHistory(role) {
		return nil, model.ErrAccessDenied
	}

	if err := validateReqParams(rph); err != nil {
		return nil, err
	}

	res, err := svc.repo.GetItemHistoryByID(ctx, rph, id)
	if err != nil {
		log.Printf("RID %q Failed to get item history from DB in 'GetItemHistoryByID': %q", rid, err)
		return nil, model.ErrCommon500
	}

	return res, nil
}

func (svc WHCService) GetItemHistoryAll(ctx context.Context, rph *model.RequestParam, role string) ([]*model.ItemHistory, error) {
	rid := model.RequestIDFromCtx(ctx)

	if !svc.policy.AccessToGetHistory(role) {
		return nil, model.ErrAccessDenied
	}

	if err := validateReqParams(rph); err != nil {
		return nil, err
	}

	res, err := svc.repo.GetItemHistoryAll(ctx, rph)
	if err != nil {
		log.Printf("RID %q Failed to get all history from DB in 'GetItemHistoryAll': %q", rid, err)
		return nil, model.ErrCommon500
	}

	return res, nil
}
