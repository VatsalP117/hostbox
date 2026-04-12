package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vatsalpatel/hostbox/cmd/cli/internal/client"
	"github.com/vatsalpatel/hostbox/cmd/cli/internal/config"
	"github.com/vatsalpatel/hostbox/cmd/cli/internal/output"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login [server-url]",
	Short: "Authenticate with a Hostbox server",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := flagServer
		if len(args) > 0 {
			serverURL = args[0]
		}

		if serverURL == "" {
			cfg, _ := config.Load()
			if cfg != nil && cfg.ServerURL != "" {
				serverURL = cfg.ServerURL
			}
		}

		if serverURL == "" {
			return fmt.Errorf("server URL required — usage: hostbox login <server-url>")
		}

		serverURL = strings.TrimRight(serverURL, "/")

		// Prompt for credentials
		email := output.Prompt("Email: ", nil)
		fmt.Fprint(os.Stderr, "Password: ")
		pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}

		c := client.New(serverURL, "")
		resp, err := c.Login(email, string(pwBytes))
		if err != nil {
			return err
		}

		// Save config and token
		cfg := &config.Config{ServerURL: serverURL}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		if err := config.SaveToken(resp.AccessToken); err != nil {
			return fmt.Errorf("save token: %w", err)
		}

		output.Success("Logged in to %s", serverURL)
		return nil
	},
}
