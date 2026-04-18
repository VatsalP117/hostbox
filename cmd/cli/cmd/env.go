package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables",
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

		resp, err := c.ListEnvVars(projectID)
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(resp.EnvVars)
			return nil
		}

		if len(resp.EnvVars) == 0 {
			output.Info("No environment variables. Set one with 'hostbox env set KEY=VALUE'")
			return nil
		}

		t := output.NewTable("KEY", "VALUE")
		for _, v := range resp.EnvVars {
			// Mask value by default
			masked := strings.Repeat("*", min(len(v.Value), 20))
			t.Row(v.Key, masked)
		}
		t.Flush()
		return nil
	},
}

var envSetCmd = &cobra.Command{
	Use:   "set <KEY=VALUE> [KEY=VALUE...]",
	Short: "Set environment variables",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		projectID, err := resolveProjectID()
		if err != nil {
			return err
		}

		for _, arg := range args {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid format %q — use KEY=VALUE", arg)
			}

			if err := c.SetEnvVar(projectID, parts[0], parts[1]); err != nil {
				return fmt.Errorf("set %s: %w", parts[0], err)
			}
			output.Success("Set %s", parts[0])
		}

		return nil
	},
}

var envDeleteCmd = &cobra.Command{
	Use:   "delete <env-var-id>",
	Short: "Delete an environment variable",
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

		if err := c.DeleteEnvVar(projectID, args[0]); err != nil {
			return err
		}

		output.Success("Environment variable deleted")
		return nil
	},
}

var envImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import environment variables from a .env file",
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

		f, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		var count int
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Strip surrounding quotes
			value = strings.Trim(value, `"'`)

			if err := c.SetEnvVar(projectID, key, value); err != nil {
				output.Warn("Failed to set %s: %v", key, err)
				continue
			}
			count++
		}

		output.Success("Imported %d environment variables", count)
		return nil
	},
}

func init() {
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envDeleteCmd)
	envCmd.AddCommand(envImportCmd)
}
