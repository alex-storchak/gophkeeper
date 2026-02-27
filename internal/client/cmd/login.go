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

type UserLoginer interface {
	Login(ctx context.Context, username, password string) (string, error)
}

func NewLoginCmd(ul UserLoginer, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login existing user",
		RunE:  newRunLogin(ul, cfg),
	}
}

func newRunLogin(ul UserLoginer, cfg *config.Config) func(*cobra.Command, []string) error {
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
			return fmt.Errorf("reading user password from input: %w; username: %s", err, creds.username)
		}

		err = processLogin(cmd.Context(), creds, ul, cfg)
		if err != nil {
			return fmt.Errorf("process login for user %s: %w", creds.username, err)
		}

		fmt.Printf("Successful login for user %s.\n", creds.username)
		return nil
	}
}

func processLogin(
	ctx context.Context,
	creds credentials,
	ul UserLoginer,
	cfg *config.Config,
) error {
	_, err := validator.IsValid(creds)
	if err != nil {
		return err
	}

	token, err := ul.Login(ctx, creds.username, creds.password)
	if err != nil {
		return fmt.Errorf("login user: %w", err)
	}

	sess := session.NewSession(&cfg.Session, token, creds.username)
	if err := sess.Save(); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}
