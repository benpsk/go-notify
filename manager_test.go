package notify

import (
	"context"
	"errors"
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

func TestRegister_ValidatesInputs(t *testing.T) {
	t.Parallel()

	var nilManager *Manager
	if err := nilManager.Register(&testProvider{name: "discord"}); err == nil {
		t.Fatal("expected error for nil manager")
	}

	m, err := NewManager()
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := m.Register(nil); err == nil {
		t.Fatal("expected error for nil provider")
	}

	if err := m.Register(&testProvider{name: "   "}); err == nil {
		t.Fatal("expected error for empty provider name")
	}
}

func TestRegister_NormalizesAndReplacesProvider(t *testing.T) {
	t.Parallel()

	m, err := NewManager()
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	first := &testProvider{name: " Discord "}
	second := &testProvider{name: "discord"}

	if err := m.Register(first); err != nil {
		t.Fatalf("register first: %v", err)
	}
	if err := m.Register(second); err != nil {
		t.Fatalf("register second: %v", err)
	}

	if err := m.Notify(context.Background(), "DISCORD", Message{Text: "x"}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if first.called {
		t.Fatal("expected first provider to be replaced")
	}
	if !second.called {
		t.Fatal("expected second provider to be called")
	}
}

func TestNotify_ValidatesInputsAndPropagatesProviderError(t *testing.T) {
	t.Parallel()

	var nilManager *Manager
	if err := nilManager.Notify(context.Background(), "discord", Message{Text: "x"}); err == nil {
		t.Fatal("expected error for nil manager")
	}

	wantErr := errors.New("provider failed")
	p := &testProvider{name: "discord", retErr: wantErr}
	m, err := NewManager(p)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := m.Notify(context.Background(), "   ", Message{Text: "x"}); err == nil {
		t.Fatal("expected error for empty provider name")
	}

	err = m.Notify(context.Background(), " discord ", Message{Text: "x"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestProviders_NilManager(t *testing.T) {
	t.Parallel()

	var nilManager *Manager
	if got := nilManager.Providers(); got != nil {
		t.Fatalf("expected nil providers, got %#v", got)
	}
}

func TestNewManager_ReturnsErrorWhenRegistrationFails(t *testing.T) {
	t.Parallel()

	if _, err := NewManager(nil); err == nil {
		t.Fatal("expected error when provider is nil")
	}
	if _, err := NewManager(&testProvider{name: "   "}); err == nil {
		t.Fatal("expected error when provider name is empty")
	}
}
