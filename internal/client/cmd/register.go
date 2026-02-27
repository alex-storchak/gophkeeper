package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alex-storchak/gophkeeper/internal/client/config"
	"github.com/alex-storchak/gophkeeper/internal/client/input"
	"github.com/alex-storchak/gophkeeper/internal/client/session"
	"github.com/alex-storchak/gophkeeper/internal/client/validator"
)

const (
	MsgEmptyUsername = "username cannot be empty"
	MsgEmptyPassword = "password cannot be empty"
)

type credentials struct {
	username string
	password string
}

func (c credentials) Valid() validator.Problems {
	problems := make(validator.Problems)

	if c.username == "" {
		problems["username"] = MsgEmptyUsername
	}

	if c.password == "" {
		problems["password"] = MsgEmptyPassword
	}

	return problems
}

type UserRegisterer interface {
	Register(ctx context.Context, username, password string) (string, error)
}

func NewRegisterCmd(ur UserRegisterer, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register a new user",
		RunE:  newRunRegister(ur, cfg),
	}
}

func newRunRegister(ur UserRegisterer, cfg *config.Config) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		var creds credentials
		var err error

		creds.username, err = input.ReadLine("Enter username: ")
		if err != nil {
			return fmt.Errorf("reading username from input: %w", err)
		}
		creds.password, err = input.ReadPassword("Enter user password: ")
		if err != nil {
			return fmt.Errorf("reading user password from input for user %s: %w", creds.username, err)
		}

		err = processRegister(cmd.Context(), creds, ur, cfg)
		if err != nil {
			return fmt.Errorf("process register for user %s: %w", creds.username, err)
		}

		fmt.Println("User registered successfully. Session saved.")
		return nil
	}
}

func processRegister(
	ctx context.Context,
	creds credentials,
	ur UserRegisterer,
	cfg *config.Config,
) error {
	_, err := validator.IsValid(creds)
	if err != nil {
		return err
	}

	token, err := ur.Register(ctx, creds.username, creds.password)
	if err != nil {
		return fmt.Errorf("register user: %w", err)
	}

	sess := session.NewSession(&cfg.Session, token, creds.username)
	if err := sess.Save(); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}
