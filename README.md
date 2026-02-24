# go-notify

Generic notification library for multiple providers (Discord now, Telegram/Line/Email later).

`go-notify` provides:
- a provider-agnostic `notify` package (message model + provider interface + manager)
- provider implementations as subpackages (currently `discord`)

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
)
```

## Package layout

- module root (`notify` package): generic message + provider interface + manager/registry
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
