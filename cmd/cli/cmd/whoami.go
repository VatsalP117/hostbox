package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authenticated user",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		user, err := c.WhoAmI()
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(user)
			return nil
		}

		fmt.Printf("Email: %s\n", user.Email)
		if user.DisplayName != "" {
			fmt.Printf("Name:  %s\n", user.DisplayName)
		}
		if user.IsAdmin {
			fmt.Printf("Role:  %s\n", output.Cyan("admin"))
		}
		return nil
	},
}
