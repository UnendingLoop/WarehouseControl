// Package mwauthlog provides UUID-logging to every request
package mwauthlog

import (
	"context"
	"net/http"
	"time"

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
		c.Set(ReqID, rid)

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

		// прокидываем дальше
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("username", claims.Username)

		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, exists := c.Get("role")
		if !exists {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		found := false
		for _, v := range roles {
			if r == v {
				found = true
				break
			}
		}

		if !found {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}
