package cmd

import (
	"github.com/spf13/cobra"
	clientpkg "github.com/VatsalP117/hostbox/cmd/cli/internal/client"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <deployment-id>",
	Short: "Rollback to a previous deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		projectID, err := resolveProjectID()
		if err != nil {
			return err
		}

		dep, err := c.Rollback(projectID, clientpkg.RollbackRequest{DeploymentID: args[0]})
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(dep)
			return nil
		}

		output.Success("Rollback triggered: %s", dep.ID)
		output.Info("Rolling back to deployment %s", args[0])
		return nil
	},
}
