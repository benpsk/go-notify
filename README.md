# go-notify

Generic notification library for multiple providers (Discord + Email now, Telegram/Line later).

`go-notify` provides:
- a provider-agnostic `notify` package (message model + provider interface + manager)
- provider implementations as subpackages (`discord`, `email`)

## Installation

Install the latest tagged release:

```bash
go get github.com/benpsk/go-notify@latest
```

Install a specific version (recommended for production):

```bash
go get github.com/benpsk/go-notify@v0.1.0
```

## Import

```go
import (
	notify "github.com/benpsk/go-notify"
	"github.com/benpsk/go-notify/discord"
	"github.com/benpsk/go-notify/email"
)
```

## Package layout

- module root (`notify` package): generic message + provider interface + manager/registry
- `discord`: Discord webhook provider implementation
- `email`: SMTP provider implementation

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

## Email usage

```go
package main

import (
	"context"
	"log"

	notify "github.com/benpsk/go-notify"
	"github.com/benpsk/go-notify/email"
)

func main() {
	emailProvider, err := email.NewSMTPClient(
		"smtp.example.com",
		587,
		"smtp-user",
		"smtp-pass",
		"noreply@example.com",
		[]string{"ops@example.com"},
	)
	if err != nil {
		log.Fatal(err)
	}

	manager, err := notify.NewManager(emailProvider)
	if err != nil {
		log.Fatal(err)
	}

	err = manager.Notify(context.Background(), "email", notify.Message{
		Subject: "Job failed",
		Text:    "Nightly import exited with code 1",
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

Set `Message.Meta["to"]` (comma/semicolon-separated) to override default recipients for a specific message.

## Versioning

This module follows Go module versioning with Git tags (SemVer):

- `v0.x.y`: initial releases, API may still change
- `v1.x.y`: stable API
- `v2+`: breaking changes require a new import path suffix (for example `/v2`)

Examples:

```bash
go get github.com/benpsk/go-notify@v0.1.0
go get github.com/benpsk/go-notify@latest
```

## Adding other providers later

Create a package (for example `telegram`, `line`) that implements:

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
