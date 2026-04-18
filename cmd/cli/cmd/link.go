package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/link"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
)

var linkCmd = &cobra.Command{
	Use:   "link <project-id>",
	Short: "Link current directory to a Hostbox project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID := args[0]

		c, err := getClient()
		if err != nil {
			return err
		}

		// Verify project exists
		proj, err := c.GetProject(projectID)
		if err != nil {
			return fmt.Errorf("project not found: %w", err)
		}

		if err := link.Save(proj.ID, c.BaseURL); err != nil {
			return fmt.Errorf("save link: %w", err)
		}

		output.Success("Linked to project: %s (%s)", proj.Name, proj.Slug)
		output.Info("Created .hostbox.json in current directory")
		return nil
	},
}
