package email

import (
	"context"
	"errors"
	"net/smtp"
	"strings"
	"testing"

	notify "github.com/benpsk/go-notify"
)

func TestNewSMTPClient_ValidatesInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		port int
		from string
		want string
	}{
		{name: "missing host", host: "", port: 587, from: "sender@example.com", want: "smtp host"},
		{name: "invalid port", host: "smtp.example.com", port: 0, from: "sender@example.com", want: "smtp port"},
		{name: "invalid from", host: "smtp.example.com", port: 587, from: "sender", want: "from address"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewSMTPClient(tt.host, tt.port, "", "", tt.from, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got err %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestName(t *testing.T) {
	t.Parallel()

	c, err := NewSMTPClient("smtp.example.com", 587, "", "", "sender@example.com", nil)
	if err != nil {
		t.Fatalf("new smtp client: %v", err)
	}

	if got := c.Name(); got != "email" {
		t.Fatalf("got provider name %q want %q", got, "email")
	}
}

func TestNotify_SendsEmailWithDefaults(t *testing.T) {
	t.Parallel()

	c, err := NewSMTPClient(
		"smtp.example.com",
		587,
		"mailer@example.com",
		"topsecret",
		"sender@example.com",
		[]string{"ops@example.com"},
	)
	if err != nil {
		t.Fatalf("new smtp client: %v", err)
	}

	var gotAddr string
	var gotFrom string
	var gotTo []string
	var gotBody string
	var gotAuth smtp.Auth
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		gotAddr = addr
		gotAuth = a
		gotFrom = from
		gotTo = append([]string(nil), to...)
		gotBody = string(msg)
		return nil
	}

	err = c.Notify(context.Background(), notify.Message{
		Subject: "Deploy Complete",
		Text:    "Service is healthy",
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}

	if gotAddr != "smtp.example.com:587" {
		t.Fatalf("got smtp addr %q", gotAddr)
	}
	if gotAuth == nil {
		t.Fatal("expected smtp auth to be configured")
	}
	if gotFrom != "sender@example.com" {
		t.Fatalf("got from %q", gotFrom)
	}
	if len(gotTo) != 1 || gotTo[0] != "ops@example.com" {
		t.Fatalf("got recipients %#v", gotTo)
	}
	if !strings.Contains(gotBody, "Subject: Deploy Complete") {
		t.Fatalf("missing subject in message: %q", gotBody)
	}
	if !strings.Contains(gotBody, "Content-Type: text/plain; charset=UTF-8") {
		t.Fatalf("missing text content type in message: %q", gotBody)
	}
	if !strings.Contains(gotBody, "Service is healthy") {
		t.Fatalf("missing body in message: %q", gotBody)
	}
}

func TestNotify_OverrideRecipientsViaMeta(t *testing.T) {
	t.Parallel()

	c, err := NewSMTPClient(
		"smtp.example.com",
		587,
		"",
		"",
		"sender@example.com",
		[]string{"default@example.com"},
	)
	if err != nil {
		t.Fatalf("new smtp client: %v", err)
	}

	var gotTo []string
	var gotBody string
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		gotTo = append([]string(nil), to...)
		gotBody = string(msg)
		return nil
	}

	err = c.Notify(context.Background(), notify.Message{
		Subject: "HTML fallback",
		HTML:    "<p>render this</p>",
		Meta: map[string]string{
			"to": "a@example.com; b@example.com",
		},
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}

	if len(gotTo) != 2 || gotTo[0] != "a@example.com" || gotTo[1] != "b@example.com" {
		t.Fatalf("got recipients %#v", gotTo)
	}
	if !strings.Contains(gotBody, "Content-Type: text/html; charset=UTF-8") {
		t.Fatalf("missing html content type in message: %q", gotBody)
	}
}

func TestNotify_RequiresBodyAndRecipients(t *testing.T) {
	t.Parallel()

	c, err := NewSMTPClient("smtp.example.com", 587, "", "", "sender@example.com", nil)
	if err != nil {
		t.Fatalf("new smtp client: %v", err)
	}
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}

	err = c.Notify(context.Background(), notify.Message{Subject: "x"})
	if err == nil || !strings.Contains(err.Error(), "body") {
		t.Fatalf("expected body error, got %v", err)
	}

	err = c.Notify(context.Background(), notify.Message{Subject: "x", Text: "body"})
	if err == nil || !strings.Contains(err.Error(), "recipients") {
		t.Fatalf("expected recipients error, got %v", err)
	}
}

func TestNotify_PropagatesContextAndSendError(t *testing.T) {
	t.Parallel()

	c, err := NewSMTPClient("smtp.example.com", 587, "", "", "sender@example.com", []string{"ops@example.com"})
	if err != nil {
		t.Fatalf("new smtp client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Notify(ctx, notify.Message{Text: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}

	wantErr := errors.New("smtp unavailable")
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return wantErr
	}

	err = c.Notify(context.Background(), notify.Message{Text: "x"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected send error, got %v", err)
	}
}

func TestNewSMTPClient_NormalizesRecipientsAndOptionalAuth(t *testing.T) {
	t.Parallel()

	c, err := NewSMTPClient(
		" smtp.example.com ",
		587,
		"   ",
		"ignored",
		"Sender <sender@example.com>",
		[]string{" Ops <ops@example.com> ", "dev@example.com"},
	)
	if err != nil {
		t.Fatalf("new smtp client: %v", err)
	}

	if c.host != "smtp.example.com" {
		t.Fatalf("got host %q", c.host)
	}
	if c.from != "sender@example.com" {
		t.Fatalf("got from %q", c.from)
	}
	if len(c.defaultTo) != 2 || c.defaultTo[0] != "ops@example.com" || c.defaultTo[1] != "dev@example.com" {
		t.Fatalf("unexpected default recipients: %#v", c.defaultTo)
	}
	if c.auth != nil {
		t.Fatal("expected auth to be nil when username is empty")
	}
}

func TestNewSMTPClient_InvalidDefaultRecipients(t *testing.T) {
	t.Parallel()

	_, err := NewSMTPClient(
		"smtp.example.com",
		587,
		"",
		"",
		"sender@example.com",
		[]string{"not-an-email"},
	)
	if err == nil || !strings.Contains(err.Error(), "default recipients") {
		t.Fatalf("expected default recipients validation error, got %v", err)
	}
}

func TestNotify_NilClient(t *testing.T) {
	t.Parallel()

	var c *Client
	if err := c.Notify(context.Background(), notify.Message{Text: "x"}); err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestNotify_DefaultSubjectAndCleanHeader(t *testing.T) {
	t.Parallel()

	c, err := NewSMTPClient("smtp.example.com", 587, "", "", "sender@example.com", []string{"ops@example.com"})
	if err != nil {
		t.Fatalf("new smtp client: %v", err)
	}

	var gotBody string
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		gotBody = string(msg)
		return nil
	}

	if err := c.Notify(context.Background(), notify.Message{Text: "hello"}); err != nil {
		t.Fatalf("notify with default subject: %v", err)
	}
	if !strings.Contains(gotBody, "Subject: Notification") {
		t.Fatalf("expected default subject, got %q", gotBody)
	}

	if err := c.Notify(context.Background(), notify.Message{Subject: "hello\r\nbcc: bad", Text: "x"}); err != nil {
		t.Fatalf("notify with sanitized subject: %v", err)
	}
	if !strings.Contains(gotBody, "Subject: hello  bcc: bad") {
		t.Fatalf("expected sanitized subject, got %q", gotBody)
	}
}

func TestNotify_InvalidRecipientOverride(t *testing.T) {
	t.Parallel()

	c, err := NewSMTPClient("smtp.example.com", 587, "", "", "sender@example.com", []string{"ops@example.com"})
	if err != nil {
		t.Fatalf("new smtp client: %v", err)
	}

	err = c.Notify(context.Background(), notify.Message{
		Text: "x",
		Meta: map[string]string{"to": "bad-address"},
	})
	if err == nil || !strings.Contains(err.Error(), "email recipients") {
		t.Fatalf("expected recipients parse error, got %v", err)
	}
}

func TestNormalizeAddress_RequiresValue(t *testing.T) {
	t.Parallel()

	if _, err := normalizeAddress("   "); err == nil {
		t.Fatal("expected error for empty address")
	}
}

func TestNormalizeAddresses_SkipsEmptyAndReturnsError(t *testing.T) {
	t.Parallel()

	got, err := normalizeAddresses([]string{" ", "\t", "a@example.com"})
	if err != nil {
		t.Fatalf("normalize addresses: %v", err)
	}
	if len(got) != 1 || got[0] != "a@example.com" {
		t.Fatalf("unexpected normalized addresses: %#v", got)
	}

	if _, err := normalizeAddresses([]string{"ok@example.com", "bad"}); err == nil {
		t.Fatal("expected error for invalid recipient")
	}
}
