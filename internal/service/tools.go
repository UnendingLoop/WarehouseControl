package service

import (
	"strings"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func validateNormalizeUser(u *model.User) error {
	// Проверка роли
	if _, exists := model.RolesMap[u.Role]; !exists {
		return model.ErrIncorrectUserRole
	}
	u.UserName = strings.TrimSpace(u.UserName)
	u.UserName = strings.ToLower(u.UserName)

	// Генерация хэша из пароля
	passHash, _ := bcrypt.GenerateFromPassword([]byte(u.PassHash), bcrypt.DefaultCost)
	u.PassHash = string(passHash)

	return nil
}

func validateItem(item *model.Item) error {
	if item.Title == "" {
		return model.ErrEmptyTitle
	}
	if item.Price < 0 {
		return model.ErrInvalidPrice
	}
	if item.AvailableAmount < 0 {
		return model.ErrInvalidAvail
	}
	return nil
}

func validateItemUpdate(item *model.ItemUpdate) error {
	if item.Title == nil && item.Description == nil && item.Price == nil && item.Visible == nil && item.AvailableAmount == nil {
		return model.ErrEmptyItemInfo
	}

	if item.Title != nil && *item.Title == "" {
		return model.ErrEmptyTitle
	}
	if item.Price != nil && *item.Price < 0 {
		return model.ErrInvalidPrice
	}
	if item.AvailableAmount != nil && *item.AvailableAmount < 0 {
		return model.ErrInvalidAvail
	}
	return nil
}

func validateReqParams(rp *model.RequestParam) error {
	if rp.OrderBy != nil {
		// валидация самого OrderBy
		if _, ok := model.OrderByItemsMap[*rp.OrderBy]; !ok {
			return model.ErrInvalidOrderBy
		}
		// валидация asc/desc
		if rp.ASC == rp.DESC {
			return model.ErrInvalidAscDesc
		}
	}

	if rp.StartTime != nil && rp.EndTime != nil {
		if rp.StartTime.After(*rp.EndTime) {
			return model.ErrInvalidStartEndTime
		}
	}

	if rp.Page != nil {
		if *rp.Page <= 0 {
			return model.ErrInvalidPage
		}
		if *rp.Limit <= 0 || *rp.Limit >= 1000 {
			return model.ErrInvalidLimit
		}
	}

	return nil
}
