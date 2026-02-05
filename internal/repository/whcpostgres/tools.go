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

	// работаем уже со значениями - если lim или p был nil, то значение 0
	if limit <= 0 { // задаем значение по умолчанию если лимит пуст/некорректен
		limit = 20
	}

	if limit > 1000 { // защита от слишком больших значений
		limit = 1000
	}

	if page <= 0 { // если страница имеет некорректное значение - ставим 1
		page = 1
	}

	// оба значения теперь точно корректны - добавляем в квери
	offset := limit * (page - 1)
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}

func defineOrderExpr(orderBy *string, asc, desc bool) (string, error) {
	if orderBy == nil {
		return "", nil
	}

	_, okItems := model.OrderByItemsMap[*orderBy]
	_, okHistory := model.OrderByHistoryMap[*orderBy]

	if !(okItems || okHistory) {
		return "", model.ErrInvalidOrderBy
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

	return fmt.Sprintf(" ORDER BY %s %s ", *orderBy, direction), nil
}

func definePeriodExpr(start, end *time.Time, leadOp string, dbField string) string {
	switch {
	case start != nil && end != nil:
		return fmt.Sprintf(" %s %s BETWEEN '%s' AND '%s'", leadOp, dbField, start.Format(time.RFC3339), end.Format(time.RFC3339))
	case start != nil:
		return fmt.Sprintf(" %s %s > '%s'", leadOp, dbField, start.Format(time.RFC3339))
	case end != nil:
		return fmt.Sprintf(" %s %s < '%s'", leadOp, dbField, end.Format(time.RFC3339))
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
