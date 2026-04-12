package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vatsalpatel/hostbox/cmd/cli/internal/output"
)

var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "Manage custom domains",
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

		resp, err := c.ListDomains(projectID)
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(resp.Domains)
			return nil
		}

		if len(resp.Domains) == 0 {
			output.Info("No custom domains. Add one with 'hostbox domains add <domain>'")
			return nil
		}

		t := output.NewTable("DOMAIN", "VERIFIED", "CREATED")
		for _, d := range resp.Domains {
			verified := output.Red("no")
			if d.Verified {
				verified = output.Green("yes")
			}
			t.Row(d.Domain, fmt.Sprint(verified), d.CreatedAt)
		}
		t.Flush()
		return nil
	},
}

var domainsAddCmd = &cobra.Command{
	Use:   "add <domain>",
	Short: "Add a custom domain",
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

		d, err := c.AddDomain(projectID, args[0])
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(d)
			return nil
		}

		output.Success("Domain added: %s", d.Domain)
		output.Info("Point a CNAME record to your Hostbox server to verify")
		return nil
	},
}

var domainsRemoveCmd = &cobra.Command{
	Use:   "remove <domain-id>",
	Short: "Remove a custom domain",
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

		if err := c.DeleteDomain(projectID, args[0]); err != nil {
			return err
		}

		output.Success("Domain removed")
		return nil
	},
}

var domainsVerifyCmd = &cobra.Command{
	Use:   "verify <domain-id>",
	Short: "Verify a custom domain's DNS",
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

		d, err := c.VerifyDomain(projectID, args[0])
		if err != nil {
			return err
		}

		if flagJSON {
			output.PrintJSON(d)
			return nil
		}

		if d.Verified {
			output.Success("Domain %s verified!", d.Domain)
		} else {
			output.Warn("Domain %s not yet verified — check DNS configuration", d.Domain)
		}
		return nil
	},
}

func init() {
	domainsCmd.AddCommand(domainsAddCmd)
	domainsCmd.AddCommand(domainsRemoveCmd)
	domainsCmd.AddCommand(domainsVerifyCmd)
}
