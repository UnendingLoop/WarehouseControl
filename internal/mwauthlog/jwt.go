package mwauthlog

import (
	"time"

	"github.com/UnendingLoop/EventBooker/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewJWTManager(secret []byte, ttl time.Duration, issuer string) *JWTManager {
	return &JWTManager{secret: secret, ttl: ttl, issuer: issuer}
}

func (j *JWTManager) Generate(uid int, email string, role string) (string, error) {
	claims := Claims{
		UserID: uid,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(j.secret)
	if err != nil {
		return "", err
	}

	return signed, nil
}

func (j *JWTManager) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, model.ErrInvalidToken
			}
			return j.secret, nil
		},
	)
	if err != nil {
		return nil, model.ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, model.ErrInvalidToken
	}

	return claims, nil
}
