package transport

import (
	"context"
	"errors"
	"strconv"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
)

func convertHistoryToCSV(ctx context.Context, input []*model.ItemHistory) ([][]string, error) {
	result := make([][]string, 0, len(input)+1)
	start := []string{"id", "item_id", "version", "action", "changed_at", "changed_by", "old_data", "new_data"}
	result = append(result, start)

	for _, v := range input {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			row := make([]string, 0, len(start))

			oldData := ""
			if v.OldData != nil {
				oldData = string(*v.OldData)
			}

			newData := ""
			if v.NewData != nil {
				newData = string(*v.NewData)
			}

			row = append(row,
				strconv.Itoa(v.ID),
				strconv.Itoa(v.ItemID),
				strconv.Itoa(v.Version),
				v.Action,
				v.ChangedAt.Format("2006-01-02 15:04:05"),
				v.ChangedBy,
				oldData,
				newData)
			result = append(result, row)
		}
	}
	return result, nil
}

func convertItemsToCSV(ctx context.Context, input []*model.Item) ([][]string, error) {
	result := make([][]string, 0, len(input)+1)
	start := []string{"item_id", "title", "description", "price", "visible", "available_amount", "created_at", "updated_at", "deleted_at"}
	result = append(result, start)

	for _, v := range input {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			row := make([]string, 0, len(start))
			vis := "false"
			if v.Visible {
				vis = "true"
			}

			deletedAt := ""
			if v.DeletedAt != nil {
				deletedAt = v.DeletedAt.Format("2006-01-02 15:04:05")
			}

			row = append(row,
				strconv.Itoa(v.ID),
				v.Title,
				v.Description,
				strconv.Itoa(int(v.Price)),
				vis,
				strconv.Itoa(v.AvailableAmount),
				v.CreatedAt.Format("2006-01-02 15:04:05"),
				v.UpdatedAt.Format("2006-01-02 15:04:05"),
				deletedAt)
			result = append(result, row)
		}
	}
	return result, nil
}

func errorCodeDefiner(err error) int { // потом можно сделать кастомный тип ошибок вместе с кодом HTTP
	switch {
	case errors.Is(err, model.ErrInvalidToken),
		errors.Is(err, model.ErrInvalidCredentials),
		errors.Is(err, model.ErrInvalidOrderBy),
		errors.Is(err, model.ErrInvalidAscDesc),
		errors.Is(err, model.ErrInvalidStartEndTime),
		errors.Is(err, model.ErrInvalidPage),
		errors.Is(err, model.ErrInvalidLimit),
		errors.Is(err, model.ErrInvalidRequestParam),
		errors.Is(err, model.ErrEmptyUser),
		errors.Is(err, model.ErrIncorrectItemID),
		errors.Is(err, model.ErrIncorrectUserName),
		errors.Is(err, model.ErrIncorrectUserRole),
		errors.Is(err, model.ErrEmptyItemInfo),
		errors.Is(err, model.ErrEmptyTitle),
		errors.Is(err, model.ErrInvalidPrice),
		errors.Is(err, model.ErrInvalidAvail),
		errors.Is(err, model.ErrNoFieldsToUpdate):
		return 400
	case errors.Is(err, model.ErrAccessDenied):
		return 403
	case errors.Is(err, model.ErrUserNotFound),
		errors.Is(err, model.ErrItemNotFound):
		return 404
	case errors.Is(err, model.ErrUserAlreadyExists):
		return 409
	default:
		return 500
	}
}
