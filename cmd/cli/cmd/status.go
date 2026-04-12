package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vatsalpatel/hostbox/cmd/cli/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status [deployment-id]",
	Short: "Show deployment status",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		projectID, err := resolveProjectID()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			// Show specific deployment
			dep, err := c.GetDeployment(projectID, args[0])
			if err != nil {
				return err
			}
			if flagJSON {
				output.PrintJSON(dep)
				return nil
			}
			fmt.Printf("ID:     %s\n", dep.ID)
			fmt.Printf("Branch: %s\n", dep.Branch)
			fmt.Printf("Status: %s %s\n", output.StatusIcon(dep.Status), dep.Status)
			if dep.CommitSHA != "" {
				fmt.Printf("Commit: %s\n", dep.CommitSHA)
			}
			if dep.DeploymentURL != nil {
				fmt.Printf("URL:    %s\n", *dep.DeploymentURL)
			}
			if dep.ErrorMessage != nil {
				fmt.Printf("Error:  %s\n", *dep.ErrorMessage)
			}
			return nil
		}

		// Show latest deployments
		resp, err := c.ListDeployments(projectID)
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(resp.Deployments)
			return nil
		}

		if len(resp.Deployments) == 0 {
			output.Info("No deployments found")
			return nil
		}

		t := output.NewTable("STATUS", "ID", "BRANCH", "COMMIT", "CREATED")
		for _, d := range resp.Deployments {
			sha := d.CommitSHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			t.Row(
				output.StatusIcon(d.Status)+" "+d.Status,
				d.ID,
				d.Branch,
				sha,
				d.CreatedAt,
			)
		}
		t.Flush()
		return nil
	},
}

func init() {
	statusCmd.Aliases = []string{"st"}
}

var logsCmd = &cobra.Command{
	Use:   "logs <deployment-id>",
	Short: "Stream build logs for a deployment",
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

		deployID := args[0]

		// Get deployment to show initial status
		dep, err := c.GetDeployment(projectID, deployID)
		if err != nil {
			return err
		}

		fmt.Printf("Deployment %s (%s) — %s\n\n", dep.ID, dep.Branch, dep.Status)

		if dep.Status == "ready" || dep.Status == "failed" {
			output.Info("Deployment already completed with status: %s", dep.Status)
			if dep.ErrorMessage != nil {
				output.Error("Error: %s", *dep.ErrorMessage)
			}
		} else {
			output.Info("Deployment is %s — logs will appear as the build progresses", dep.Status)
			output.Info("(Full log streaming requires SSE support — use the dashboard for real-time logs)")
		}

		return nil
	},
}
