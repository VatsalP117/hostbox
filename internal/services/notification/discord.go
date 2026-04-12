package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	colorGreen  = 3066993  // #2ECC71
	colorRed    = 15158332 // #E74C3C
	colorBlue   = 3447003  // #3498DB
	colorOrange = 15105570 // #E67E22
)

type DiscordClient struct {
	httpClient *http.Client
}

func (c *DiscordClient) Send(ctx context.Context, webhookURL string, payload NotificationPayload) error {
	embed := c.buildEmbed(payload)

	body := map[string]any{
		"embeds": []any{embed},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (c *DiscordClient) buildEmbed(p NotificationPayload) map[string]any {
	embed := map[string]any{
		"timestamp": p.Timestamp,
	}

	switch p.Event {
	case EventDeploySuccess:
		embed["title"] = fmt.Sprintf("✅ Deployment Ready — %s", p.Project.Name)
		embed["color"] = colorGreen
		if p.Deployment != nil {
			desc := fmt.Sprintf("Branch: `%s`", p.Deployment.Branch)
			if p.Deployment.CommitSHA != "" {
				desc += fmt.Sprintf(" | Commit: `%s`", truncate(p.Deployment.CommitSHA, 7))
			}
			if p.Deployment.CommitMessage != "" {
				desc += "\n" + p.Deployment.CommitMessage
			}
			embed["description"] = desc
			if p.Deployment.DeploymentURL != "" {
				embed["url"] = p.Deployment.DeploymentURL
			}

			var fields []map[string]any
			if p.Deployment.BuildDurationMs > 0 {
				fields = append(fields, map[string]any{
					"name": "Duration", "value": formatDuration(p.Deployment.BuildDurationMs), "inline": true,
				})
			}
			fields = append(fields, map[string]any{
				"name": "Status", "value": strings.Title(p.Deployment.Status), "inline": true,
			})
			if p.Deployment.DeploymentURL != "" {
				fields = append(fields, map[string]any{
					"name": "URL", "value": fmt.Sprintf("[Open Preview](%s)", p.Deployment.DeploymentURL), "inline": false,
				})
			}
			embed["fields"] = fields
		}

	case EventDeployFailure:
		embed["title"] = fmt.Sprintf("❌ Deployment Failed — %s", p.Project.Name)
		embed["color"] = colorRed
		if p.Deployment != nil {
			desc := fmt.Sprintf("Branch: `%s`", p.Deployment.Branch)
			if p.Deployment.CommitSHA != "" {
				desc += fmt.Sprintf(" | Commit: `%s`", truncate(p.Deployment.CommitSHA, 7))
			}
			if p.Deployment.CommitMessage != "" {
				desc += "\n" + p.Deployment.CommitMessage
			}
			embed["description"] = desc

			var fields []map[string]any
			if p.Deployment.ErrorMessage != "" {
				fields = append(fields, map[string]any{
					"name": "Error", "value": truncate(p.Deployment.ErrorMessage, 1024), "inline": false,
				})
			}
			if p.Deployment.DashboardURL != "" {
				fields = append(fields, map[string]any{
					"name": "Logs", "value": fmt.Sprintf("[View Logs](%s)", p.Deployment.DashboardURL), "inline": false,
				})
			}
			embed["fields"] = fields
		}

	case EventDomainVerified:
		if p.Domain != nil {
			embed["title"] = fmt.Sprintf("🌐 Domain Verified — %s", p.Domain.Domain)
			embed["description"] = fmt.Sprintf("Domain `%s` has been verified for project **%s**.\nSSL certificate will be provisioned automatically.", p.Domain.Domain, p.Project.Name)
		}
		embed["color"] = colorBlue

	case EventDomainUnverified:
		if p.Domain != nil {
			embed["title"] = fmt.Sprintf("⚠️ Domain Unverified — %s", p.Domain.Domain)
			embed["description"] = fmt.Sprintf("Domain `%s` for project **%s** failed DNS verification.\nPlease check your DNS configuration.", p.Domain.Domain, p.Project.Name)
		}
		embed["color"] = colorOrange
	}

	return embed
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func formatDuration(ms int64) string {
	secs := ms / 1000
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm%ds", secs/60, secs%60)
}
