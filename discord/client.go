package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	notify "github.com/benpsk/go-notify"
)

const defaultTimeout = 5 * time.Second

// Client sends messages to a Discord webhook URL.
type Client struct {
	webhookURL string
	httpClient *http.Client
}

// Message is a minimal Discord webhook payload.
// Extend this later (embeds, username override, avatar_url, etc.) if needed.
type Message struct {
	Content string `json:"content"`
}

// NewClient builds a Discord webhook client with a default timeout.
func NewClient(webhookURL string) (*Client, error) {
	return NewClientWithHTTPClient(webhookURL, &http.Client{Timeout: defaultTimeout})
}

// NewClientWithHTTPClient builds a Discord webhook client with a custom HTTP client.
func NewClientWithHTTPClient(webhookURL string, httpClient *http.Client) (*Client, error) {
	url := strings.TrimSpace(webhookURL)
	if url == "" {
		return nil, fmt.Errorf("discord webhook url is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		webhookURL: url,
		httpClient: httpClient,
	}, nil
}

// SendContent sends a simple content-only message.
func (c *Client) SendContent(ctx context.Context, content string) error {
	return c.Send(ctx, Message{Content: strings.TrimSpace(content)})
}

// Name returns the provider name used by notify.Manager.
func (c *Client) Name() string {
	return "discord"
}

// Notify adapts a generic notify.Message to a Discord content-only payload.
func (c *Client) Notify(ctx context.Context, msg notify.Message) error {
	content := strings.TrimSpace(msg.Text)
	subject := strings.TrimSpace(msg.Subject)

	if subject != "" && content != "" {
		content = subject + "\n" + content
	} else if subject != "" {
		content = subject
	}

	return c.SendContent(ctx, content)
}

// Send posts the webhook payload to Discord.
func (c *Client) Send(ctx context.Context, msg Message) error {
	if c == nil {
		return fmt.Errorf("discord webhook client is nil")
	}

	msg.Content = strings.TrimSpace(msg.Content)
	if msg.Content == "" {
		return fmt.Errorf("discord webhook content is required")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("discord webhook status: %s", resp.Status)
	}

	return nil
}
