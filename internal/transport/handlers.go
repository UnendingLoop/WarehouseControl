package transport

import (
	"encoding/csv"
	"log"
	"net/http"
	"strconv"

	"github.com/UnendingLoop/EventBooker/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
)

func (eh *WHCHandlers) SimplePinger(ctx *ginext.Context) {
	rid := stringFromCtx(ctx, "request_id")
	ctx.JSON(200, gin.H{rid: "pong"})
}

func (eh *WHCHandlers) SignUpUser(ctx *gin.Context) {
	var newUser model.User

	if err := ctx.ShouldBindJSON(&newUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user payload"})
		return
	}

	token, err := eh.svc.CreateUser(ctx.Request.Context(), &newUser)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}
	resp := convertUserAuthToResponse(&newUser)

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})

	ctx.JSON(http.StatusCreated, resp)
}

func (eh *WHCHandlers) LoginUser(ctx *gin.Context) {
	var req authRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid auth payload"})
		return
	}

	token, user, err := eh.svc.LoginUser(ctx.Request.Context(), req.UserName, req.Password)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}
	resp := convertUserAuthToResponse(user)

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})

	ctx.JSON(http.StatusOK, resp)
}

func (eh *WHCHandlers) CreateItem(ctx *gin.Context) {
	// логируем role-sensitive запрос
	rid := stringFromCtx(ctx, "request_id")
	uid := intFromCtx(ctx, "user_id")
	mail := stringFromCtx(ctx, "email")
	role := stringFromCtx(ctx, "role")

	log.Printf("rid=%q userID=%d userEmail=%q role=%q creating event", rid, uid, mail, role)

	// дальше обычная логика
	if role != "admin" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var item model.Item
	if err := ctx.ShouldBindJSON(&item); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid item payload"})
		return
	}

	if err := eh.svc.CreateItem(ctx.Request.Context(), &item, role); err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusCreated)
}

func (eh *WHCHandlers) GetItemByID(ctx *gin.Context) {
	role := stringFromCtx(ctx, "role")

	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty event id"})
		return
	}
	id := stringToInt(rawID)

	res, err := eh.svc.GetItemByID(ctx.Request.Context(), id, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (eh *WHCHandlers) UpdateItem(ctx *gin.Context) {
	// логируем role-sensitive запрос
	rid := stringFromCtx(ctx, "request_id")
	uid := intFromCtx(ctx, "user_id")
	mail := stringFromCtx(ctx, "email")
	role := stringFromCtx(ctx, "role")

	log.Printf("rid=%q userID=%d userEmail=%q role=%q creating event", rid, uid, mail, role)

	// дальше обычная логика
	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty event id"})
		return
	}

	id := stringToInt(rawID)

	var item model.Item
	if err := ctx.ShouldBindJSON(&item); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid item payload"})
		return
	}

	if err := eh.svc.UpdateItemByID(ctx.Request.Context(), id, &item, role); err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusCreated)
}

func (eh *WHCHandlers) DeleteItem(ctx *gin.Context) {
	// логируем role-sensitive запрос
	rid := stringFromCtx(ctx, "request_id")
	uid := intFromCtx(ctx, "user_id")
	username := stringFromCtx(ctx, "username")
	role := stringFromCtx(ctx, "role")

	log.Printf("rid=%q userID=%d userName=%q role=%q deleting event", rid, uid, username, role)

	// обычный флоу
	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty item id"})
		return
	}

	id := stringToInt(rawID)
	err := eh.svc.DeleteItemByID(ctx.Request.Context(), id, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

func (eh *WHCHandlers) GetItemsList(ctx *gin.Context) {
	// парсим параметры запроса из URL
	rpa := model.RequestParamItems{}
	if err := decodeQueryParams(ctx, &rpa); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := stringFromCtx(ctx, "role")
	res, err := eh.svc.GetItemsList(ctx.Request.Context(), role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (eh *WHCHandlers) GetItemHistoryByID(ctx *gin.Context) {
	role := stringFromCtx(ctx, "role")
	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty event id"})
		return
	}
	id := stringToInt(rawID)

	res, err := eh.svc.GetItemHistoryByID(ctx.Request.Context(), id, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (eh *WHCHandlers) GetItemsListAsCSV(ctx *gin.Context) {
	role := stringFromCtx(ctx, "role")
	res, err := eh.svc.GetItemsList(ctx.Request.Context(), role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (eh *WHCHandlers) GetHistoryListAsCSV(ctx *gin.Context) {
	role := stringFromCtx(ctx, "role")
	res, err := eh.svc.GetItemsList(ctx.Request.Context(), role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (h *OperationHandler) ExportOperationsCSV(ctx *ginext.Context) {
	// парсим параметры запроса операций из URL
	rpo := model.RequestParamOperations{}
	if err := decodeQueryParams(ctx, &rpo); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// получаем массив строк
	res, err := h.svc.GetAllOperations(ctx.Request.Context(), &rpo)
	if err != nil {
		ctx.JSON(errCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	// устанавливаем хедеры под CSV
	rows := convertOperationsToCSV(res)
	ctx.Writer.Header().Set("Cache-Control", "no-store")
	ctx.Writer.Header().Set("Pragma", "no-cache")
	ctx.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Writer.Header().Set("Content-Type", "text/csv")
	ctx.Writer.Header().Set("Content-Disposition", "attachment; filename=operations.csv")

	// пишем данные
	writer := csv.NewWriter(ctx.Writer)
	if err := writer.WriteAll(rows); err != nil {
		log.Printf("failed to Flush csv-writer: %q", err.Error())
		return
	}
}

func (h *OperationHandler) ExportAnalyticsCSV(ctx *ginext.Context) {
	// парсим параметры запроса аналитики из URL
	rpa := model.RequestParamAnalytics{}
	if err := decodeQueryParams(ctx, &rpa); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// получаем массив строк
	res, err := h.svc.GetAnalytics(ctx.Request.Context(), &rpa)
	if err != nil {
		ctx.JSON(errCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	// устанавливаем хедеры под CSV
	ctx.Writer.Header().Set("Cache-Control", "no-store")
	ctx.Writer.Header().Set("Pragma", "no-cache")
	ctx.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Writer.Header().Set("Content-Type", "text/csv")
	ctx.Writer.Header().Set("Content-Disposition", "attachment; filename=analytics.csv")

	// пишем данные
	rows := convertAnalyticsToCSV(res)
	writer := csv.NewWriter(ctx.Writer)
	if err := writer.WriteAll(rows); err != nil {
		log.Printf("failed to Flush csv-writer: %q", err.Error())
		return
	}
}

func convertAnalyticsToCSV(input *model.AnalyticsSummary) [][]string {
	result := make([][]string, 0, len(input.Groups)+1)
	start := []string{"group_key", "total_amount", "average", "operations_in_group", "mediana", "P90"}
	result = append(result, start)

	for _, v := range input.Groups {
		row := make([]string, 0, len(start))
		row = append(row, v.Key, strconv.FormatFloat(v.Sum/100, 'f', 2, 64), strconv.FormatFloat(v.Avg/100, 'f', 2, 64), strconv.Itoa(v.Count), strconv.FormatFloat(v.Median/100, 'f', 2, 64), strconv.FormatFloat(v.P90/100, 'f', 2, 64))
		result = append(result, row)
	}

	end := []string{"TOTALS:", strconv.FormatFloat(input.Sum/100, 'f', 2, 64), strconv.FormatFloat(input.Avg/100, 'f', 2, 64), strconv.Itoa(input.Count), strconv.FormatFloat(input.Median/100, 'f', 2, 64), strconv.FormatFloat(input.P90/100, 'f', 2, 64)}
	result = append(result, end)
	return result
}

func convertOperationsToCSV(input []model.Operation) [][]string {
	result := make([][]string, 0, len(input)+1)
	start := []string{"id", "amount", "type", "category", "actor", "date", "created", "description"}
	result = append(result, start)

	for _, v := range input {
		row := make([]string, 0, len(start))
		descr := ""
		if v.Description != nil {
			descr = *v.Description
		}
		row = append(row, strconv.FormatInt(v.ID, 10), strconv.FormatFloat(float64(v.Amount)/100, 'f', 2, 64), v.Type, v.Category, v.Actor, v.OperationAt.Format("2006-01-02"), v.CreatedAt.Format("2006-01-02"), descr)
		result = append(result, row)
	}

	return result
}
