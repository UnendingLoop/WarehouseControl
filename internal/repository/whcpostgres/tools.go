package whcpostgres

import (
	"fmt"
	"strings"
	"time"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
)

func defineLimitOffsetExpr(lim, p *int) string {
	if lim == nil && p == nil { // оба значения нил - вообще не применяем их к квери
		return ""
	}

	// избавляемся от указателей
	var limit, page int
	if lim != nil {
		limit = *lim
	}
	if p != nil {
		page = *p
	}

	if limit <= 0 { // задаем значение по умолчанию если лимит пуст/некорректен
		limit = 20
	}

	if limit > 1000 { // защита от слишком больших значений
		limit = 1000
	}

	if page <= 0 { // если страница имеет некорректное значение - ставим 1
		page = 1
	}

	// оба значения корректны - добавляем в квери
	offset := limit * (page - 1)
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}

func defineOrderExpr(orderBy *string, asc, desc bool) (string, error) {
	if orderBy == nil {
		return "", nil
	}

	direction := ""
	switch {
	case asc == desc:
		direction = "DESC" // значение по умолчанию
	case asc:
		direction = "ASC"
	default:
		direction = "DESC"
	}

	switch *orderBy {
	case model.ItemsOrderByID:
		return fmt.Sprintf("ORDER BY id %s ", direction), nil
	case model.ItemsOrderByTitle:
		return fmt.Sprintf("ORDER BY title %s ", direction), nil
	case model.ItemsOrderByPrice:
		return fmt.Sprintf("ORDER BY price %s ", direction), nil
	case model.ItemsOrderByAvailability:
		return fmt.Sprintf("ORDER BY available_amount %s ", direction), nil
	case model.ItemsOrderByVisibility:
		return fmt.Sprintf("ORDER BY visible %s ", direction), nil

	case model.HistoryOrderByAction:
		return fmt.Sprintf("ORDER BY action %s ", direction), nil
	case model.HistoryOrderByVersion:
		return fmt.Sprintf("ORDER BY version %s ", direction), nil
	case model.HistoryOrderByActor:
		return fmt.Sprintf("ORDER BY changed_by %s ", direction), nil
	default:
		return "", model.ErrInvalidOrderBy
	}
}

func definePeriodExpr(start, end *time.Time, dbField string) string {
	switch {
	case start != nil && end != nil:
		return fmt.Sprintf("WHERE %s BETWEEN '%s' AND '%s'", dbField, start.Format(time.RFC3339), end.Format(time.RFC3339))
	case start != nil:
		return fmt.Sprintf("WHERE %s > '%s'", dbField, start.Format(time.RFC3339))
	case end != nil:
		return fmt.Sprintf("WHERE %s < '%s'", dbField, end.Format(time.RFC3339))
	default:
		return ""
	}
}

func updateQueryBuilder(uItem *model.ItemUpdate) (string, []any, error) {
	var sets []string
	var values []any

	counter := 1 // $1 будет id в основном запросе в Where

	// добавляем только нениловые поля
	if uItem.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", counter+1))
		values = append(values, *uItem.Title)
		counter++
	}
	if uItem.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", counter+1))
		values = append(values, *uItem.Description)
		counter++
	}
	if uItem.Price != nil {
		sets = append(sets, fmt.Sprintf("price = $%d", counter+1))
		values = append(values, *uItem.Price)
		counter++
	}
	if uItem.Visible != nil {
		sets = append(sets, fmt.Sprintf("visible = $%d", counter+1))
		values = append(values, *uItem.Visible)
		counter++
	}
	if uItem.AvailableAmount != nil {
		sets = append(sets, fmt.Sprintf("available_amount = $%d", counter+1))
		values = append(values, *uItem.AvailableAmount)
		counter++
	}

	// вставляем обновителя записи
	sets = append(sets, fmt.Sprintf("updated_by = $%d", counter+1))
	values = append(values, uItem.UpdatedBy)

	if len(sets) == 1 {
		return "", nil, model.ErrNoFieldsToUpdate
	}

	// формируем SET часть запроса
	setClause := "SET " + strings.Join(sets, ", ")

	return setClause, values, nil
}
