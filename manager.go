package notify

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Provider is implemented by notification providers (discord, telegram, email, ...).
type Provider interface {
	Name() string
	Notify(ctx context.Context, msg Message) error
}

// Manager stores providers and routes notifications by provider name.
type Manager struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewManager creates a manager and optionally registers providers.
func NewManager(providers ...Provider) (*Manager, error) {
	m := &Manager{providers: make(map[string]Provider, len(providers))}
	for _, p := range providers {
		if err := m.Register(p); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Register adds or replaces a provider by its normalized name.
func (m *Manager) Register(p Provider) error {
	if m == nil {
		return fmt.Errorf("notify manager is nil")
	}
	if p == nil {
		return fmt.Errorf("notify provider is nil")
	}

	name := strings.ToLower(strings.TrimSpace(p.Name()))
	if name == "" {
		return fmt.Errorf("notify provider name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = p
	return nil
}

// Notify sends a message using the named provider.
func (m *Manager) Notify(ctx context.Context, providerName string, msg Message) error {
	if m == nil {
		return fmt.Errorf("notify manager is nil")
	}

	name := strings.ToLower(strings.TrimSpace(providerName))
	if name == "" {
		return fmt.Errorf("notify provider name is required")
	}

	m.mu.RLock()
	p, ok := m.providers[name]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("notify provider not found: %s", name)
	}

	return p.Notify(ctx, msg)
}

// Providers returns registered provider names in sorted order.
func (m *Manager) Providers() []string {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	m.mu.RUnlock()

	slices.Sort(names)
	return names
}
