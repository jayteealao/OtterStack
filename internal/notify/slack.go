package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackNotifier sends notifications to Slack via webhooks.
type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
	client     *http.Client
}

// SlackMessage represents a Slack webhook message.
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment.
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Ts         int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment.
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// Slack attachment colors.
const (
	SlackColorGood    = "good"    // Green
	SlackColorWarning = "warning" // Yellow
	SlackColorDanger  = "danger"  // Red
)

// NewSlackNotifier creates a new Slack notifier.
func NewSlackNotifier(webhookURL, channel, username string) *SlackNotifier {
	if username == "" {
		username = "OtterStack"
	}
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (s *SlackNotifier) Name() string {
	return "slack"
}

// Send sends a notification to Slack.
func (s *SlackNotifier) Send(ctx context.Context, event Event) error {
	attachment := s.createAttachment(event)

	msg := SlackMessage{
		Channel:     s.channel,
		Username:    s.username,
		IconEmoji:   ":otter:",
		Attachments: []SlackAttachment{attachment},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := retryableSend(ctx, s.client, req, 2)
	if err != nil {
		return fmt.Errorf("failed to send to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Slack returned status %d", resp.StatusCode)
	}

	return nil
}

// Close cleans up resources.
func (s *SlackNotifier) Close() error {
	s.client.CloseIdleConnections()
	return nil
}

func (s *SlackNotifier) createAttachment(event Event) SlackAttachment {
	attachment := SlackAttachment{
		Title:  GetEventTitle(event),
		Color:  s.getColor(event),
		Footer: "OtterStack",
		Ts:     event.Timestamp.Unix(),
	}

	// Add fields
	fields := []SlackField{
		{Title: "Project", Value: event.Project, Short: true},
	}

	if event.Service != "" {
		fields = append(fields, SlackField{
			Title: "Service",
			Value: event.Service,
			Short: true,
		})
	}

	if event.Status != "" {
		fields = append(fields, SlackField{
			Title: "Status",
			Value: event.Status,
			Short: true,
		})
	}

	if event.Message != "" {
		attachment.Text = event.Message
	}

	// Add any additional details
	for k, v := range event.Details {
		fields = append(fields, SlackField{
			Title: k,
			Value: v,
			Short: true,
		})
	}

	attachment.Fields = fields
	return attachment
}

func (s *SlackNotifier) getColor(event Event) string {
	switch event.Type {
	case EventDeploySucceeded, EventServiceRecovered, EventServiceUp:
		return SlackColorGood
	case EventDeployFailed, EventServiceDown:
		return SlackColorDanger
	case EventServiceUnhealthy:
		return SlackColorWarning
	default:
		return ""
	}
}
