package cmd

import (
	"github.com/spf13/cobra"
	clientpkg "github.com/VatsalP117/hostbox/cmd/cli/internal/client"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
)

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"ps"},
	Short:   "List all projects",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		resp, err := c.ListProjects()
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(resp.Projects)
			return nil
		}

		if len(resp.Projects) == 0 {
			output.Info("No projects found. Create one with 'hostbox project create'")
			return nil
		}

		t := output.NewTable("NAME", "SLUG", "FRAMEWORK", "CREATED")
		for _, p := range resp.Projects {
			fw := "-"
			if p.Framework != nil {
				fw = *p.Framework
			}
			t.Row(p.Name, p.Slug, fw, p.CreatedAt)
		}
		t.Flush()
		return nil
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "project create <name>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		req := clientCreateProjectReq(args[0], cmd)
		proj, err := c.CreateProject(req)
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(proj)
			return nil
		}

		output.Success("Project created: %s (%s)", proj.Name, proj.Slug)
		output.Info("Link it with: hostbox link %s", proj.ID)
		return nil
	},
}

func init() {
	projectCreateCmd.Flags().String("slug", "", "Project slug")
	projectCreateCmd.Flags().String("git-repo", "", "Git repository URL")
	projectCreateCmd.Flags().String("branch", "", "Production branch (default: main)")
	projectCreateCmd.Flags().String("framework", "", "Framework (react, vue, next, etc.)")
	projectCreateCmd.Flags().String("build-command", "", "Build command")
	projectCreateCmd.Flags().String("output-dir", "", "Output directory")
}

func clientCreateProjectReq(name string, cmd *cobra.Command) clientpkg.CreateProjectRequest {
	req := clientpkg.CreateProjectRequest{Name: name}
	if v, _ := cmd.Flags().GetString("slug"); v != "" {
		req.Slug = v
	}
	if v, _ := cmd.Flags().GetString("git-repo"); v != "" {
		req.GitRepo = v
	}
	if v, _ := cmd.Flags().GetString("branch"); v != "" {
		req.ProductionBranch = v
	}
	if v, _ := cmd.Flags().GetString("framework"); v != "" {
		req.Framework = v
	}
	if v, _ := cmd.Flags().GetString("build-command"); v != "" {
		req.BuildCommand = v
	}
	if v, _ := cmd.Flags().GetString("output-dir"); v != "" {
		req.OutputDir = v
	}
	return req
}
