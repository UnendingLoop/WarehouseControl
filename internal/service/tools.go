package service

import (
	"strings"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func validateNormalizeNewUser(u *model.User) error {
	if u == nil {
		return model.ErrEmptyUser
	}

	u.UserName = strings.TrimSpace(u.UserName)
	u.UserName = strings.ToLower(u.UserName)

	// Генерация хэша из пароля
	passHash, _ := bcrypt.GenerateFromPassword([]byte(u.PassHash), bcrypt.DefaultCost)
	u.PassHash = string(passHash)

	return nil
}

func validateItem(item *model.Item) error {
	if item == nil {
		return model.ErrEmptyItemInfo
	}
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
	if item == nil {
		return model.ErrNoFieldsToUpdate
	}

	if item.Title == nil && item.Description == nil && item.Price == nil && item.Visible == nil && item.AvailableAmount == nil {
		return model.ErrNoFieldsToUpdate
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
	if item.UpdatedBy == "" {
		return model.ErrIncorrectUserName
	}
	return nil
}

func validateReqParams(rp *model.RequestParam) error {
	if rp == nil {
		return model.ErrInvalidRequestParam
	}

	if rp.OrderBy != nil {
		// валидация самого OrderBy
		_, okItems := model.OrderByItemsMap[*rp.OrderBy]
		_, okHistory := model.OrderByHistoryMap[*rp.OrderBy]

		if !okHistory && !okItems {
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
		if rp.Limit == nil || *rp.Limit <= 0 || *rp.Limit >= 1000 {
			return model.ErrInvalidLimit
		}
	}

	return nil
}
