package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/alex-storchak/gophkeeper/internal/models"
	"github.com/alex-storchak/gophkeeper/internal/server/config"
)

var (
	ErrInvalidUserIDInClaims = errors.New("invalid user_id in JWT claims")
	ErrInvalidJWTClaims      = errors.New("invalid JWT token claims")
)

type Manager struct {
	secret   []byte
	tokenTTL time.Duration
}

func NewManager(cfg *config.Auth) *Manager {
	return &Manager{
		secret:   []byte(cfg.JWTSecret),
		tokenTTL: cfg.TokenTTL,
	}
}

func (m *Manager) Generate(userID models.UserID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": int64(userID),
		"exp":     time.Now().Add(m.tokenTTL).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("signing JWT token: %w", err)
	}

	return signed, nil
}

func (m *Manager) Validate(tokenStr string) (models.UserID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return 0, fmt.Errorf("parsing JWT token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, ErrInvalidJWTClaims
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, ErrInvalidUserIDInClaims
	}

	return models.UserID(int64(userIDFloat)), nil
}
