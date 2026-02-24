package notify

import (
	"context"
	"testing"
)

type testProvider struct {
	name    string
	gotMsg  Message
	called  bool
	retErr  error
}

func (p *testProvider) Name() string { return p.name }

func (p *testProvider) Notify(ctx context.Context, msg Message) error {
	p.called = true
	p.gotMsg = msg
	return p.retErr
}

func TestNewManagerAndNotify(t *testing.T) {
	t.Parallel()

	p := &testProvider{name: "discord"}
	m, err := NewManager(p)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	msg := Message{Subject: "s", Text: "hello"}
	if err := m.Notify(context.Background(), "discord", msg); err != nil {
		t.Fatalf("notify: %v", err)
	}

	if !p.called {
		t.Fatal("provider was not called")
	}
	if p.gotMsg.Text != "hello" {
		t.Fatalf("got text %q", p.gotMsg.Text)
	}
}

func TestNotify_UnknownProvider(t *testing.T) {
	t.Parallel()

	m, err := NewManager()
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := m.Notify(context.Background(), "telegram", Message{Text: "x"}); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestProviders_ReturnsSortedNames(t *testing.T) {
	t.Parallel()

	m, err := NewManager(&testProvider{name: "line"}, &testProvider{name: "discord"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	names := m.Providers()
	if len(names) != 2 || names[0] != "discord" || names[1] != "line" {
		t.Fatalf("unexpected provider names: %#v", names)
	}
}
