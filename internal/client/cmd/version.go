package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
)

func NewVersionCmd(rootCmd *cobra.Command) *cobra.Command {
	rootCmd.Version = buildVersion
	rootCmd.SetVersionTemplate("Gophkeeper version: {{.Version}}\n")

	return &cobra.Command{
		Use:   "version",
		Short: "Information about build version and build date",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Gophkeeper client:\n")
			fmt.Printf("  Build version: %s\n", buildVersion)
			fmt.Printf("  Build date: %s\n", buildDate)
		},
	}
}

func SetBuildInfo(version, date string) {
	if version != "" {
		buildVersion = version
	}
	if date != "" {
		buildDate = date
	}
}
