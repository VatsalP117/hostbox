package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type SlackClient struct {
	httpClient *http.Client
}

func (c *SlackClient) Send(ctx context.Context, webhookURL string, payload NotificationPayload) error {
	blocks := c.buildBlocks(payload)

	body := map[string]any{
		"blocks": blocks,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (c *SlackClient) buildBlocks(p NotificationPayload) []map[string]any {
	var blocks []map[string]any

	switch p.Event {
	case EventDeploySuccess:
		blocks = append(blocks, headerBlock(fmt.Sprintf("✅ Deployment Ready — %s", p.Project.Name)))
		if p.Deployment != nil {
			fields := []map[string]any{
				mrkdwnField("*Branch:*\n`" + p.Deployment.Branch + "`"),
				mrkdwnField("*Status:*\n" + strings.Title(p.Deployment.Status)),
			}
			if p.Deployment.CommitSHA != "" {
				commitText := "*Commit:*\n`" + truncate(p.Deployment.CommitSHA, 7) + "`"
				if p.Deployment.CommitMessage != "" {
					commitText += " — " + p.Deployment.CommitMessage
				}
				fields = append(fields, mrkdwnField(commitText))
			}
			if p.Deployment.BuildDurationMs > 0 {
				fields = append(fields, mrkdwnField("*Duration:*\n"+formatDuration(p.Deployment.BuildDurationMs)))
			}
			blocks = append(blocks, sectionFields(fields))

			var buttons []map[string]any
			if p.Deployment.DeploymentURL != "" {
				buttons = append(buttons, buttonElement("Open Preview", p.Deployment.DeploymentURL, ""))
			}
			if p.Deployment.DashboardURL != "" {
				buttons = append(buttons, buttonElement("View Dashboard", p.Deployment.DashboardURL, ""))
			}
			if len(buttons) > 0 {
				blocks = append(blocks, actionsBlock(buttons))
			}
		}

	case EventDeployFailure:
		blocks = append(blocks, headerBlock(fmt.Sprintf("❌ Deployment Failed — %s", p.Project.Name)))
		if p.Deployment != nil {
			fields := []map[string]any{
				mrkdwnField("*Branch:*\n`" + p.Deployment.Branch + "`"),
			}
			if p.Deployment.CommitSHA != "" {
				fields = append(fields, mrkdwnField("*Commit:*\n`"+truncate(p.Deployment.CommitSHA, 7)+"`"))
			}
			blocks = append(blocks, sectionFields(fields))

			if p.Deployment.ErrorMessage != "" {
				blocks = append(blocks, sectionText("*Error:*\n```"+truncate(p.Deployment.ErrorMessage, 2000)+"```"))
			}

			if p.Deployment.DashboardURL != "" {
				blocks = append(blocks, actionsBlock([]map[string]any{
					buttonElement("View Logs", p.Deployment.DashboardURL, "danger"),
				}))
			}
		}

	case EventDomainVerified:
		if p.Domain != nil {
			blocks = append(blocks, headerBlock(fmt.Sprintf("🌐 Domain Verified — %s", p.Domain.Domain)))
			blocks = append(blocks, sectionText(fmt.Sprintf(
				"Domain `%s` has been verified for project *%s*.\nSSL certificate will be provisioned automatically.",
				p.Domain.Domain, p.Project.Name)))
		}

	case EventDomainUnverified:
		if p.Domain != nil {
			blocks = append(blocks, headerBlock(fmt.Sprintf("⚠️ Domain Unverified — %s", p.Domain.Domain)))
			blocks = append(blocks, sectionText(fmt.Sprintf(
				"Domain `%s` for project *%s* failed DNS verification.\nPlease check your DNS configuration.",
				p.Domain.Domain, p.Project.Name)))
		}
	}

	blocks = append(blocks, contextBlock(fmt.Sprintf("Hostbox • <%s|Dashboard>", p.ServerURL)))
	return blocks
}

func headerBlock(text string) map[string]any {
	return map[string]any{
		"type": "header",
		"text": map[string]any{"type": "plain_text", "text": text, "emoji": true},
	}
}

func sectionFields(fields []map[string]any) map[string]any {
	return map[string]any{
		"type":   "section",
		"fields": fields,
	}
}

func sectionText(text string) map[string]any {
	return map[string]any{
		"type": "section",
		"text": map[string]any{"type": "mrkdwn", "text": text},
	}
}

func mrkdwnField(text string) map[string]any {
	return map[string]any{"type": "mrkdwn", "text": text}
}

func actionsBlock(elements []map[string]any) map[string]any {
	return map[string]any{
		"type":     "actions",
		"elements": elements,
	}
}

func buttonElement(text, url, style string) map[string]any {
	btn := map[string]any{
		"type": "button",
		"text": map[string]any{"type": "plain_text", "text": text},
		"url":  url,
	}
	if style != "" {
		btn["style"] = style
	}
	return btn
}

func contextBlock(text string) map[string]any {
	return map[string]any{
		"type": "context",
		"elements": []any{
			map[string]any{"type": "mrkdwn", "text": text},
		},
	}
}
