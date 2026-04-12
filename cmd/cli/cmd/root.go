package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vatsalpatel/hostbox/cmd/cli/internal/client"
	"github.com/vatsalpatel/hostbox/cmd/cli/internal/config"
	"github.com/vatsalpatel/hostbox/cmd/cli/internal/link"
)

var (
	flagJSON    bool
	flagServer  string
	flagToken   string
	flagNoColor bool
	flagProject string
	flagVerbose bool
)

var rootCmd = &cobra.Command{
	Use:   "hostbox",
	Short: "Hostbox CLI — deploy frontend apps to your own server",
	Long: `Hostbox is a self-hosted deployment platform for frontend applications.
Use this CLI to manage projects, trigger deployments, configure domains,
and administer your Hostbox instance.`,
	Version:       "0.0.0-dev",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().StringVar(&flagServer, "server", "", "Hostbox server URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "Auth token (overrides stored credentials)")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringVar(&flagProject, "project", "", "Project ID (overrides .hostbox.json)")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(projectCreateCmd)
	rootCmd.AddCommand(linkCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(domainsCmd)
	rootCmd.AddCommand(envCmd)
	rootCmd.AddCommand(adminCmd)
}

// getClient returns an authenticated API client.
func getClient() (*client.Client, error) {
	serverURL := flagServer
	token := flagToken

	if serverURL == "" || token == "" {
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		if serverURL == "" {
			serverURL = cfg.ServerURL
		}
		if token == "" {
			t, err := config.LoadToken()
			if err != nil {
				return nil, fmt.Errorf("load token: %w", err)
			}
			token = t
		}
	}

	if serverURL == "" {
		return nil, fmt.Errorf("no server configured — run 'hostbox login <server-url>' first")
	}
	if token == "" {
		return nil, fmt.Errorf("not authenticated — run 'hostbox login' first")
	}

	return client.New(serverURL, token), nil
}

// resolveProjectID returns the project ID from flag, link file, or error.
func resolveProjectID() (string, error) {
	if flagProject != "" {
		return flagProject, nil
	}

	lnk, err := link.Discover()
	if err != nil {
		return "", err
	}
	return lnk.ProjectID, nil
}
