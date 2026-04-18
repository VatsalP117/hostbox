package cmd

import (
	"github.com/spf13/cobra"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/config"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.ClearToken(); err != nil {
			return err
		}
		output.Success("Logged out")
		return nil
	},
}
