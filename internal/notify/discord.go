package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DiscordNotifier sends notifications to Discord via webhooks.
type DiscordNotifier struct {
	webhookURL string
	username   string
	client     *http.Client
}

// DiscordMessage represents a Discord webhook message.
type DiscordMessage struct {
	Username  string         `json:"username,omitempty"`
	Content   string         `json:"content,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed.
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
}

// DiscordEmbedField represents a field in a Discord embed.
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedFooter represents a footer in a Discord embed.
type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

// Discord embed colors.
const (
	ColorGreen  = 0x2ECC71 // Success
	ColorRed    = 0xE74C3C // Error
	ColorYellow = 0xF1C40F // Warning
	ColorBlue   = 0x3498DB // Info
)

// NewDiscordNotifier creates a new Discord notifier.
func NewDiscordNotifier(webhookURL, username string) *DiscordNotifier {
	if username == "" {
		username = "OtterStack"
	}
	return &DiscordNotifier{
		webhookURL: webhookURL,
		username:   username,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (d *DiscordNotifier) Name() string {
	return "discord"
}

// Send sends a notification to Discord.
func (d *DiscordNotifier) Send(ctx context.Context, event Event) error {
	embed := d.createEmbed(event)

	msg := DiscordMessage{
		Username: d.username,
		Embeds:   []DiscordEmbed{embed},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := retryableSend(ctx, d.client, req, 2)
	if err != nil {
		return fmt.Errorf("failed to send to Discord: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Discord returned status %d", resp.StatusCode)
	}

	return nil
}

// Close cleans up resources.
func (d *DiscordNotifier) Close() error {
	d.client.CloseIdleConnections()
	return nil
}

func (d *DiscordNotifier) createEmbed(event Event) DiscordEmbed {
	embed := DiscordEmbed{
		Title:     GetEventTitle(event),
		Color:     d.getColor(event),
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: "OtterStack",
		},
	}

	// Add fields
	fields := []DiscordEmbedField{
		{Name: "Project", Value: event.Project, Inline: true},
	}

	if event.Service != "" {
		fields = append(fields, DiscordEmbedField{
			Name:   "Service",
			Value:  event.Service,
			Inline: true,
		})
	}

	if event.Status != "" {
		fields = append(fields, DiscordEmbedField{
			Name:   "Status",
			Value:  event.Status,
			Inline: true,
		})
	}

	if event.Message != "" {
		embed.Description = event.Message
	}

	// Add any additional details
	for k, v := range event.Details {
		fields = append(fields, DiscordEmbedField{
			Name:   k,
			Value:  v,
			Inline: true,
		})
	}

	embed.Fields = fields
	return embed
}

func (d *DiscordNotifier) getColor(event Event) int {
	switch event.Type {
	case EventDeploySucceeded, EventServiceRecovered, EventServiceUp:
		return ColorGreen
	case EventDeployFailed, EventServiceDown:
		return ColorRed
	case EventServiceUnhealthy:
		return ColorYellow
	default:
		return ColorBlue
	}
}
