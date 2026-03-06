package session

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/alex-storchak/gophkeeper/internal/client/config"
	"github.com/alex-storchak/gophkeeper/internal/client/crypto"
)

var ErrUnauthenticated = errors.New("session file not found, please login first")

type Session struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	path     string
	key      []byte
}

func NewSession(cfg *config.Session, token, username string) *Session {
	key := sha256.Sum256([]byte(cfg.Secret))
	return &Session{
		Token:    token,
		Username: username,
		path:     cfg.File,
		key:      key[:],
	}
}

func (s *Session) Save() error {
	plaintext, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling session data: %w", err)
	}

	encrypted, err := crypto.Encrypt(plaintext, s.key)
	if err != nil {
		return fmt.Errorf("encrypting session data: %w", err)
	}

	if err := os.WriteFile(s.path, encrypted, 0600); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

func (s *Session) Load() error {
	encrypted, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrUnauthenticated
		}
		return fmt.Errorf("reading session file: %w", err)
	}

	plaintext, err := crypto.Decrypt(encrypted, s.key)
	if err != nil {
		return fmt.Errorf("decrypting session data: %w", err)
	}

	if err := json.Unmarshal(plaintext, s); err != nil {
		return fmt.Errorf("unmarshaling session data: %w", err)
	}

	return nil
}

func (s *Session) UpdateToken(newToken string) error {
	if err := s.Load(); err != nil {
		return fmt.Errorf("loading session to update token: %w", err)
	}

	s.Token = newToken
	return s.Save()
}
