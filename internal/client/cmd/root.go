package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/alex-storchak/gophkeeper/internal/client/config"
)

type UserClient interface {
	UserRegisterer
	UserLoginer
}

type DataClient interface {
	DataAdder
	DataLister
	DataGetter
	DataDeleter
}

type Deps struct {
	Config     *config.Config
	Logger     *slog.Logger
	UserClient UserClient
	DataClient DataClient
}

func NewRootCmd(d *Deps) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "gophkeeper-client",
		Short:         "Gophkeeper — password manager",
		Long:          "Gophkeeper — secure storage of credentials, files and bank card data.",
		SilenceErrors: true,
	}

	rootCmd.AddCommand(NewVersionCmd(rootCmd))
	rootCmd.AddCommand(NewRegisterCmd(d.UserClient, d.Config))
	rootCmd.AddCommand(NewLoginCmd(d.UserClient, d.Config))
	rootCmd.AddCommand(NewAddCmd(d.DataClient))
	rootCmd.AddCommand(NewListCmd(d.Logger, d.DataClient))
	rootCmd.AddCommand(NewGetCmd(d.Logger, d.DataClient))
	rootCmd.AddCommand(NewDeleteCmd(d.Logger, d.DataClient))

	return rootCmd
}
