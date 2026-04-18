package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/VatsalP117/hostbox/cmd/cli/internal/output"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin operations (backup, restore)",
}

var adminBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a database backup",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		compress, _ := cmd.Flags().GetBool("compress")

		result, err := c.CreateBackup(compress)
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(result)
			return nil
		}

		output.Success("Backup created: %s", result.Filename)
		output.Info("Path: %s", result.Path)
		output.Info("Size: %s", formatBytes(result.SizeBytes))
		return nil
	},
}

var adminRestoreCmd = &cobra.Command{
	Use:   "restore <backup-path>",
	Short: "Restore database from backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		if err := c.RestoreBackup(args[0]); err != nil {
			return err
		}

		output.Success("Database restored from %s", args[0])
		output.Warn("Server is restarting...")
		return nil
	},
}

var adminBackupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient()
		if err != nil {
			return err
		}

		resp, err := c.ListBackups()
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(resp.Backups)
			return nil
		}

		if len(resp.Backups) == 0 {
			output.Info("No backups found")
			return nil
		}

		t := output.NewTable("FILENAME", "SIZE", "PATH")
		for _, b := range resp.Backups {
			t.Row(b.Filename, formatBytes(b.SizeBytes), b.Path)
		}
		t.Flush()
		return nil
	},
}

func init() {
	adminBackupCmd.Flags().Bool("compress", true, "Compress backup with gzip")
	adminCmd.AddCommand(adminBackupCmd)
	adminCmd.AddCommand(adminRestoreCmd)

	adminBackupCmd.AddCommand(adminBackupsListCmd)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
