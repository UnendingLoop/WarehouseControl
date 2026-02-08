package transport

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
)

func (whc *WHCHandlers) SimplePinger(ctx *ginext.Context) {
	rid := stringFromCtx(ctx, "request_id")
	ctx.JSON(200, gin.H{rid: "pong"})
}

func (whc *WHCHandlers) SignUpUser(ctx *gin.Context) {
	var newUser model.User

	if err := ctx.ShouldBindJSON(&newUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user payload"})
		return
	}

	token, err := whc.svc.CreateUser(ctx.Request.Context(), &newUser)
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

func (whc *WHCHandlers) LoginUser(ctx *gin.Context) {
	var req authRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid auth payload"})
		return
	}

	token, user, err := whc.svc.LoginUser(ctx.Request.Context(), req.UserName, req.Password, req.Role)
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

func (whc *WHCHandlers) CreateItem(ctx *gin.Context) {
	// логируем role-sensitive запрос
	rid := stringFromCtx(ctx, "request_id")
	uid := intFromCtx(ctx, "user_id")
	userName := stringFromCtx(ctx, "username")
	role := stringFromCtx(ctx, "role")
	log.Printf("rid=%q userID=%d userName=%q role=%q creating event", rid, uid, userName, role)

	// читаем данные нового товара
	var item model.Item
	item.UpdatedBy = userName
	if err := ctx.ShouldBindJSON(&item); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid item payload"})
		return
	}

	// передаем в ервис
	if err := whc.svc.CreateItem(ctx.Request.Context(), &item, role); err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusCreated)
}

func (whc *WHCHandlers) GetItemByID(ctx *gin.Context) {
	// определяем id и роль
	role := stringFromCtx(ctx, "role")
	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty item id"})
		return
	}
	id := stringToInt(rawID)

	// передаем в сервис
	res, err := whc.svc.GetItemByID(ctx.Request.Context(), id, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (whc *WHCHandlers) UpdateItem(ctx *gin.Context) {
	// логируем role-sensitive запрос
	rid := stringFromCtx(ctx, "request_id")
	uid := intFromCtx(ctx, "user_id")
	userName := stringFromCtx(ctx, "username")
	role := stringFromCtx(ctx, "role")

	// определяем id
	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty event id"})
		return
	}
	id := stringToInt(rawID)
	log.Printf("rid=%q userID=%d userName=%q role=%q updating item #%d", rid, uid, userName, role, id)

	// читаем обновленные данные товара
	var item model.ItemUpdate
	if err := ctx.ShouldBindJSON(&item); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid item payload"})
		return
	}
	item.UpdatedBy = userName
	item.ID = id

	// передаем в сервис
	if err := whc.svc.UpdateItemByID(ctx.Request.Context(), &item, role); err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (whc *WHCHandlers) DeleteItem(ctx *gin.Context) {
	// логируем role-sensitive запрос
	rid := stringFromCtx(ctx, "request_id")
	uid := intFromCtx(ctx, "user_id")
	username := stringFromCtx(ctx, "username")
	role := stringFromCtx(ctx, "role")

	// определяем id
	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty item id"})
		return
	}
	id := stringToInt(rawID)

	log.Printf("rid=%q userID=%d userName=%q role=%q deleting item #%d", rid, uid, username, role, id)

	// передаем в сервис
	err := whc.svc.DeleteItemByID(ctx.Request.Context(), id, role, username)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

func (whc *WHCHandlers) GetItemsList(ctx *gin.Context) {
	// парсим параметры запроса из URL
	rpi := model.RequestParam{}
	if err := decodeQueryParams(ctx, &rpi); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := stringFromCtx(ctx, "role")
	res, err := whc.svc.GetItemsList(ctx.Request.Context(), &rpi, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (whc *WHCHandlers) GetItemHistoryByID(ctx *gin.Context) {
	// парсим параметры запроса из URL
	rph := model.RequestParam{}
	if err := decodeQueryParams(ctx, &rph); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// определяем id и роль
	role := stringFromCtx(ctx, "role")
	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty item id"})
		return
	}
	id := stringToInt(rawID)

	// обращаемся к сервису
	res, err := whc.svc.GetItemHistoryByID(ctx.Request.Context(), &rph, id, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (whc *WHCHandlers) GetItemsHistoryList(ctx *gin.Context) {
	// парсим параметры запроса из URL
	rph := model.RequestParam{}
	if err := decodeQueryParams(ctx, &rph); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// определяем роль
	role := stringFromCtx(ctx, "role")

	// обращаемся к сервису
	res, err := whc.svc.GetItemHistoryAll(ctx.Request.Context(), &rph, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (whc *WHCHandlers) ExportItemsHistoryCSV(ctx *gin.Context) {
	// парсим параметры запроса из URL
	rph := model.RequestParam{}
	if err := decodeQueryParams(ctx, &rph); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// определяем роль
	role := stringFromCtx(ctx, "role")

	// обращаемся к сервису
	res, err := whc.svc.GetItemHistoryAll(ctx.Request.Context(), &rph, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	// устанавливаем хедеры под CSV
	ctx.Writer.Header().Set("Cache-Control", "no-store")
	ctx.Writer.Header().Set("Pragma", "no-cache")
	ctx.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Writer.Header().Set("Content-Type", "text/csv")
	ctx.Writer.Header().Set("Content-Disposition", "attachment; filename=itemsHistory.csv")

	// готовим и пишем данные
	rows, err := convertHistoryToCSV(ctx.Request.Context(), res)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			ctx.Status(http.StatusGatewayTimeout)
			return
		}
	}

	writer := csv.NewWriter(ctx.Writer)
	if err := writer.WriteAll(rows); err != nil {
		log.Printf("failed to Flush csv-writer: %q", err.Error())
		return
	}
}

func (whc *WHCHandlers) ExportItemsCSV(ctx *ginext.Context) {
	// парсим параметры запроса из URL
	rpi := model.RequestParam{}
	if err := decodeQueryParams(ctx, &rpi); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// определяем роль
	role := stringFromCtx(ctx, "role")

	// получаем массив строк
	res, err := whc.svc.GetItemsList(ctx.Request.Context(), &rpi, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	// устанавливаем хедеры под CSV
	ctx.Writer.Header().Set("Cache-Control", "no-store")
	ctx.Writer.Header().Set("Pragma", "no-cache")
	ctx.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Writer.Header().Set("Content-Type", "text/csv")
	ctx.Writer.Header().Set("Content-Disposition", "attachment; filename=operations.csv")

	// готовим и пишем данные
	rows, err := convertItemsToCSV(ctx.Request.Context(), res)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			ctx.Status(http.StatusGatewayTimeout)
			return
		}
	}
	writer := csv.NewWriter(ctx.Writer)
	if err := writer.WriteAll(rows); err != nil {
		log.Printf("failed to Flush csv-writer: %q", err.Error())
		return
	}
}

func (whc *WHCHandlers) ExportItemIDHistoryCSV(ctx *ginext.Context) {
	// парсим параметры запроса из URL
	rpa := model.RequestParam{}
	if err := decodeQueryParams(ctx, &rpa); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// определяем id товара и роль юзера
	rawID, ok := ctx.Params.Get("id")
	if !ok {
		ctx.JSON(400, gin.H{"error": "empty item id"})
		return
	}
	id := stringToInt(rawID)
	role := stringFromCtx(ctx, "role")

	// получаем массив History от сервиса
	res, err := whc.svc.GetItemHistoryByID(ctx.Request.Context(), &rpa, id, role)
	if err != nil {
		ctx.JSON(errorCodeDefiner(err), gin.H{"error": err.Error()})
		return
	}

	// устанавливаем хедеры под CSV
	ctx.Writer.Header().Set("Cache-Control", "no-store")
	ctx.Writer.Header().Set("Pragma", "no-cache")
	ctx.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Writer.Header().Set("Content-Type", "text/csv")
	ctx.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=item%dHistory.csv", id))

	// готовим и пишем данные
	rows, err := convertHistoryToCSV(ctx.Request.Context(), res)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			ctx.Status(http.StatusGatewayTimeout)
			return
		}
	}

	writer := csv.NewWriter(ctx.Writer)
	if err := writer.WriteAll(rows); err != nil {
		log.Printf("failed to Flush csv-writer: %q", err.Error())
		return
	}
}
