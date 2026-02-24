# go-notify

Generic notification library for multiple providers (Discord now, Telegram/Line/Email later).

Module path:

- `github.com/benpsk/go-notify`

## Package layout

- `notify` (module root): generic message + provider interface + manager/registry
- `discord`: Discord webhook provider implementation

## Basic usage

```go
package main

import (
	"context"
	"log"
	"time"

	notify "github.com/benpsk/go-notify"
	"github.com/benpsk/go-notify/discord"
)

func main() {
	discordProvider, err := discord.NewClient("https://discord.com/api/webhooks/...")
	if err != nil {
		log.Fatal(err)
	}

	manager, err := notify.NewManager(discordProvider)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = manager.Notify(ctx, "discord", notify.Message{
		Subject: "New publish",
		Text:    "Song: Example Song\nUser: user@example.com",
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

## Adding other providers later

Create a package (for example `telegram`, `line`, `email`) that implements:

```go
type Provider interface {
	Name() string
	Notify(ctx context.Context, msg Message) error
}
```

Then register it with `notify.NewManager(...)` or `manager.Register(...)`.

## Notes

- Keep provider code generic.
- Build app-specific messages in your application/service layer.
- `discord.Client` also exposes `SendContent(...)` for direct Discord-only usage.
