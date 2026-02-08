// Package mwauthlog provides UUID-logging to every request + auth methods(require auth + generate token)
package mwauthlog

import (
	"context"
	"net/http"
	"time"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type (
	Claims struct {
		UserID   int    `json:"uid"`
		Username string `json:"username"`
		Role     string `json:"role"`
		jwt.RegisteredClaims
	}
)

var ReqID = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := uuid.New().String()

		ctx := context.WithValue(c.Request.Context(), ReqID, rid)
		c.Request = c.Request.WithContext(ctx)

		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

func GenerateToken(userID int, role string, secret []byte) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func RequireAuth(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie("access_token")
		if err != nil {
			c.AbortWithStatus(401)
			return
		}

		tokenStr := cookie.Value

		token, err := jwt.ParseWithClaims(
			tokenStr,
			&Claims{},
			func(token *jwt.Token) (any, error) {
				return secret, nil
			},
		)

		if err != nil || !token.Valid {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(*Claims)

		// проверяем корректность роли - принадлежит ли она списку ролей приложения
		_, ok := model.RolesMap[claims.Role]
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// прокидываем дальше в контекст
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("username", claims.Username)

		c.Next()
	}
}

func RequireAuthTest(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie("access_token")
		if err != nil {
			c.AbortWithStatus(401)
			return
		}

		token := cookie.Value
		if token != "jwt-token" {
			c.AbortWithStatus(401)
			return
		}

		// прокидываем дальше в контекст
		c.Set("user_id", 300)
		c.Set("role", "testRole")
		c.Set("username", "testUserName")
		c.Set(ReqID, "test-request-UUID")

		c.Next()
	}
}
