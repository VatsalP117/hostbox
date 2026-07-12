package cmd

import (
	"fmt"

	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
	"github.com/spf13/cobra"
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

		if len(args) == 1 {
			// Show specific deployment
			dep, err := c.GetDeployment(args[0])
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
		projectID, err := resolveProjectID()
		if err != nil {
			return err
		}
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
	Short: "Show build logs for a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		deployID := args[0]

		// Get deployment to show initial status
		dep, err := c.GetDeployment(deployID)
		if err != nil {
			return err
		}

		logs, err := c.GetDeploymentLogs(deployID)
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(logs)
			return nil
		}

		fmt.Printf("Deployment %s (%s) — %s\n\n", dep.ID, dep.Branch, dep.Status)
		for _, line := range logs.Lines {
			fmt.Println(line)
		}
		if len(logs.Lines) == 0 {
			output.Info("No build logs available")
		}
		if logs.HasMore {
			output.Warn("Showing %d of %d log lines; use the dashboard for the full log", len(logs.Lines), logs.TotalLines)
		}

		return nil
	},
}
