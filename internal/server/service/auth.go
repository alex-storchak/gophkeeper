package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"github.com/alex-storchak/gophkeeper/internal/models"
	"github.com/alex-storchak/gophkeeper/internal/server/repository"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type UserRepo interface {
	Create(ctx context.Context, username, passwordHash string) (models.UserID, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Close()
}

type TokenGenerator interface {
	Generate(userID models.UserID) (string, error)
}

type AuthService struct {
	userRepo UserRepo
	tokenGen TokenGenerator
	logger   *slog.Logger
}

func NewAuthService(u UserRepo, g TokenGenerator, l *slog.Logger) *AuthService {
	return &AuthService{
		userRepo: u,
		tokenGen: g,
		logger:   l,
	}
}

func (s *AuthService) Register(ctx context.Context, username, password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}

	userID, err := s.userRepo.Create(ctx, username, string(hash))
	if err != nil {
		return "", fmt.Errorf("creating user: %w", err)
	}

	token, err := s.tokenGen.Generate(userID)
	if err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}

	s.logger.Info("user registered",
		slog.String("username", username),
		slog.Int64("user_id", int64(userID)),
	)
	return token, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("getting user from repo: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.tokenGen.Generate(user.ID)
	if err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}

	s.logger.Info("user logged in",
		slog.String("username", username),
		slog.Int64("user_id", int64(user.ID)),
	)
	return token, nil
}

func (s *AuthService) Close() {
	s.userRepo.Close()
}
