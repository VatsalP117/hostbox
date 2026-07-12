package cmd

import (
	"fmt"

	clientpkg "github.com/VatsalP117/hostbox/cmd/cli/internal/client"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Trigger a new deployment",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		projectID, err := resolveProjectID()
		if err != nil {
			return err
		}

		branch, _ := cmd.Flags().GetString("branch")
		branch, err = resolveDeployBranch(c, projectID, branch)
		if err != nil {
			return err
		}

		dep, err := c.TriggerDeploy(projectID, clientpkg.TriggerDeployRequest{Branch: branch})
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(dep)
			return nil
		}

		output.Success("Deployment triggered: %s", dep.ID)
		output.Info("Branch: %s | Status: %s", dep.Branch, dep.Status)
		if dep.DeploymentURL != nil {
			output.Info("URL: %s", *dep.DeploymentURL)
		}
		output.Info("Stream logs with: hostbox logs %s", dep.ID)
		return nil
	},
}

func init() {
	deployCmd.Flags().String("branch", "", "Branch to deploy (default: production branch)")
}

func resolveDeployBranch(c *clientpkg.Client, projectID, requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}

	project, err := c.GetProject(projectID)
	if err != nil {
		return "", fmt.Errorf("resolve production branch: %w", err)
	}
	if project.ProductionBranch == "" {
		return "", fmt.Errorf("project %s has no production branch configured", projectID)
	}
	return project.ProductionBranch, nil
}
