package email

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	notify "github.com/benpsk/go-notify"
)

const defaultSubject = "Notification"

type sendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// Client sends notifications using SMTP.
type Client struct {
	host      string
	port      int
	from      string
	defaultTo []string
	auth      smtp.Auth
	sendMail  sendMailFunc
}

// NewSMTPClient creates an SMTP email provider.
//
// If username is empty, authentication is disabled.
func NewSMTPClient(host string, port int, username, password, from string, defaultTo []string) (*Client, error) {
	normalizedHost := strings.TrimSpace(host)
	if normalizedHost == "" {
		return nil, fmt.Errorf("email smtp host is required")
	}
	if port <= 0 {
		return nil, fmt.Errorf("email smtp port must be greater than zero")
	}

	normalizedFrom, err := normalizeAddress(from)
	if err != nil {
		return nil, fmt.Errorf("email from address: %w", err)
	}

	normalizedDefaultTo, err := normalizeAddresses(defaultTo)
	if err != nil {
		return nil, fmt.Errorf("email default recipients: %w", err)
	}

	var auth smtp.Auth
	normalizedUsername := strings.TrimSpace(username)
	if normalizedUsername != "" {
		auth = smtp.PlainAuth("", normalizedUsername, password, normalizedHost)
	}

	return &Client{
		host:      normalizedHost,
		port:      port,
		from:      normalizedFrom,
		defaultTo: normalizedDefaultTo,
		auth:      auth,
		sendMail:  smtp.SendMail,
	}, nil
}

// Name returns the provider name used by notify.Manager.
func (c *Client) Name() string {
	return "email"
}

// Notify adapts a generic notify.Message to an SMTP email message.
//
// Set msg.Meta["to"] to a comma-separated recipient list to override default recipients.
func (c *Client) Notify(ctx context.Context, msg notify.Message) error {
	if c == nil {
		return fmt.Errorf("email smtp client is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	subject := cleanHeader(msg.Subject)
	if subject == "" {
		subject = defaultSubject
	}

	body := strings.TrimSpace(msg.Text)
	contentType := "text/plain; charset=UTF-8"
	if body == "" {
		body = strings.TrimSpace(msg.HTML)
		contentType = "text/html; charset=UTF-8"
	}
	if body == "" {
		return fmt.Errorf("email body is required")
	}

	to := append([]string(nil), c.defaultTo...)
	if rawTo := strings.TrimSpace(msg.Meta["to"]); rawTo != "" {
		override, err := parseRecipientList(rawTo)
		if err != nil {
			return fmt.Errorf("email recipients: %w", err)
		}
		to = override
	}
	if len(to) == 0 {
		return fmt.Errorf("email recipients are required")
	}

	raw := buildMessage(c.from, to, subject, contentType, body)
	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	if err := c.sendMail(addr, c.auth, c.from, to, raw); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

func buildMessage(from string, to []string, subject, contentType, body string) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\n")
	b.WriteString("To: ")
	b.WriteString(strings.Join(to, ", "))
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(subject)
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: ")
	b.WriteString(contentType)
	b.WriteString("\r\n\r\n")
	b.WriteString(body)

	return []byte(b.String())
}

func parseRecipientList(raw string) ([]string, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';'
	})
	return normalizeAddresses(parts)
}

func normalizeAddresses(values []string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}

		addr, err := normalizeAddress(value)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, addr)
	}
	return normalized, nil
}

func normalizeAddress(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("address is required")
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid address %q: %w", trimmed, err)
	}

	return parsed.Address, nil
}

func cleanHeader(value string) string {
	cleaned := strings.ReplaceAll(value, "\r", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	return strings.TrimSpace(cleaned)
}
