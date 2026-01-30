// Package transport provides processing for incoming requests and preparing info for service-layer
package transport

import (
	"context"
	"strconv"

	"github.com/UnendingLoop/EventBooker/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/form"
	"github.com/wb-go/wbf/ginext"
)

type WHCHandlers struct {
	svc WHCService
}

type WHCService interface {
	CreateItem(ctx context.Context, item *model.Item, role string) error
	GetItemByID(ctx context.Context, id int, role string) (*model.Item, error)
	UpdateItemByID(ctx context.Context, id int, item *model.Item, role string) error
	DeleteItemByID(ctx context.Context, id int, role string) error

	CreateUser(ctx context.Context, user *model.User) (string, error)
	LoginUser(ctx context.Context, username string, password string) (string, *model.User, error)

	GetItemsList(ctx context.Context, rpi model.RequestParamItems, role string) ([]*model.Item, error)
	GetItemHistoryByID(ctx context.Context, rph model.RequestParamHistory, id int, role string) ([]*model.ItemHistory, error)
}

func NewEBHandlers(svc WHCService) *WHCHandlers {
	return &WHCHandlers{svc: svc}
}

// ---------------------------------------------------------------
type authRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	User userPublic `json:"user"`
}

type userPublic struct {
	ID       int    `json:"id"`
	UserName string `json:"username"`
	Role     string `json:"role"`
}

func convertUserAuthToResponse(user *model.User) *authResponse {
	return &authResponse{User: userPublic{ID: user.ID, UserName: user.UserName, Role: user.Role}}
}

// ----------------------------------------------------------
func stringFromCtx(ctx *gin.Context, key string) string {
	if v := ctx.Value(key); v != nil {
		return v.(string)
	}
	return ""
}

func intFromCtx(ctx *gin.Context, key string) int {
	if v := ctx.Value(key); v != nil {
		return v.(int)
	}
	return 0
}

func stringToInt(input string) int {
	output, err := strconv.Atoi(input)
	if err != nil {
		return -1
	}
	return output
}

func decodeQueryParams[T *model.RequestParamHistory | *model.RequestParamItems](c *ginext.Context, input T) error {
	decoder := form.NewDecoder()
	if err := decoder.Decode(input, c.Request.URL.Query()); err != nil {
		return err
	}
	return nil
}
