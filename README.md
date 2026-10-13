# Agora Server SDK for Go

Build server-side realtime audio, video, and messaging applications with an owned, context-aware Go API.

[简体中文](README.zh-CN.md) · [GoDoc](#api-documentation-godoc) · [RTC guide](docs/rtc.md) · [RTM guide](messaging/README.md) · [Examples](examples/README.md) · [API coverage](docs/api-coverage.md) · [Development](docs/development.md)

## Features

- One RTC service per process, explicit numeric/string identities, independent channel sessions, bounded joins, and cleanup
- Independent send/receive PCM formats, variable-duration PCM, paced playback, per-user audio, VAD, and interruption
- Raw video, subscription control, RTC data streams, metadata, and connection events
- Independent RTM clients: messaging, Presence, metadata, locks, history, and Stream Topics
- Copied receive data, bounded queues, visible drop counts, and controlled native access

## Differences from the official SDK

This separate Go module adds high-level contracts and targeted binding fixes on top of the official Agora Go SDK v2.6.5. Agora still supplies the native media and messaging capabilities; this is not an import-compatible replacement.

| Official v2.6.5 | This SDK adds |
| --- | --- |
| Process initialization, connections, observers | Owned `Service`/`Session`, bounded joins, typed Go errors |
| Media and result callbacks | Context-based reads/requests, bounded queues and drop counters |
| PCM, encoded media, VAD, interruption, RTM facilities | Validated combinations, PCM pacing helper, explicit ownership |
| Native bindings and installer | Targeted ABI/lifetime fixes, checksum-pinned installation and offline checks |

Read the [detailed comparison](docs/upstream-comparison.md) for the pinned baseline, changed defaults, upstream capabilities, migration limits, and sources. The [API coverage inventory](docs/api-coverage.md) checks all 326 official Go callables plus exported fields/types/constants, records deliberate compatibility fixes, and separates high-level, managed advanced, raw-only, and unavailable capabilities. Symbol retention alone does not establish feature parity.

## Requirements and scope

Use Go **1.27.2**. RTC/RTM packages also require a C/C++ compiler, the pinned native libraries, and cgo; they do not support `CGO_ENABLED=0`. The `mediaio` and `token` packages are pure Go. The repository toolchain requirement is distinct from upstream’s minimum Go version.

| Platform | Native RTC archive | Scope |
| --- | --- | --- |
| Linux amd64 | 4.6.31.134 | Primary build and offline CI target |
| macOS amd64 / arm64 | 4.6.3.133-20260618 | Installer supported; upstream limits use to development/testing, not production |
| Linux arm64 / Windows | — | Unsupported by this distribution |

Bindings follow [Agora Go SDK v2.6.5](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/v2.6.5); see its [Chinese support notes](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/v2.6.5/README.zh.md). Downloads are SHA-256 pinned in [`scripts/sdk_versions.sh`](scripts/sdk_versions.sh). RTC archive, Go binding, and RTM core versions are separate. Native feature availability also depends on platform, extension libraries, peer SDKs, and project configuration.

## Quick start

```sh
git clone --branch dev https://github.com/zyy17/agora-server-sdk.git
cd agora-server-sdk
make download-agora-libs
make build-examples
```

These commands install libraries and compile examples; they do not join a channel. For a receive-only program, set `AGORA_APP_ID`, `AGORA_CHANNEL`, a unique `AGORA_USER_ID`, and a matching `AGORA_TOKEN` when authentication is enabled. Start a separate authorized broadcaster in the same project/channel to provide audio.

```go
package main

import (
    "context"
    "errors"
    "io"
    "log"
    "os"
    "os/signal"
    "syscall"

    agorasdk "github.com/zyy17/agora-server-sdk"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    if err := run(ctx); err != nil {
        log.Print(err)
    }
}

func run(ctx context.Context) (err error) {
    service, err := agorasdk.NewService(agorasdk.Config{
        AppID: os.Getenv("AGORA_APP_ID"),
    })
    if err != nil {
        return err
    }
    defer func() { err = errors.Join(err, service.Close()) }()
    session, err := service.Join(ctx, agorasdk.ChannelConfig{
        Channel: os.Getenv("AGORA_CHANNEL"),
        UserID:  os.Getenv("AGORA_USER_ID"),
        Token:   os.Getenv("AGORA_TOKEN"),
        Role:    agorasdk.Audience,
    })
    if err != nil {
        return err
    }
    defer func() { err = errors.Join(err, session.Close()) }()
    for {
        frame, nextErr := session.Audio().Next(ctx)
        if errors.Is(nextErr, io.EOF) {
            return nil
        }
        if nextErr != nil {
            return nextErr
        }
        log.Printf("user=%s bytes=%d", frame.UserID, len(frame.Data))
    }
}
```

Save this as an application’s `main.go` and run it through the environment helper below. The example exits on Ctrl+C/SIGTERM and explicitly closes its session and service. Production applications should separately consume session events, renew tokens, and watch queue drops.

Default received audio is signed PCM16 little-endian, 16 kHz, mono. For sending, use a `Broadcaster` session: `SendPCM` accepts complete integral-millisecond PCM chunks (10 ms multiples at 44.1 kHz) with caller-controlled pacing; `PlayPCM` paces an `io.Reader`. First send publishes the track. Set `Audio.Send` and `Audio.Receive` independently; `Config.IdentityMode` selects string accounts (default) or numeric UIDs. See the [RTC contracts](docs/rtc.md) before choosing a format or lifecycle.

Supply RTC/RTM tokens from a trusted token server. RTC joining does not need an App Certificate; authenticated projects require a matching token, while unsecured projects accept an empty string. The optional pure-Go `token.RTC` helper generates RTC AccessToken2 on trusted servers only; it uses string accounts and does not mint RTM or combined RTC+RTM tokens. Keep existing trusted token generation when numeric identities or combined privileges are required. Keep certificates and tokens out of committed files and logs.

### Use from another Go module

```sh
go get github.com/zyy17/agora-server-sdk@dev
```

Install the pinned SDK from a checkout of this repository, then run from your application directory:

```sh
/path/to/agora-server-sdk/scripts/with_agora_env.sh go build ./...
/path/to/agora-server-sdk/scripts/with_agora_env.sh go run .
/path/to/agora-server-sdk/scripts/with_agora_env.sh ./your-binary
```

`@dev` tracks a moving branch; pin a reviewed commit for reproducible application builds. The helper configures cgo/linker/runtime paths for the installed SDK. Deployment still requires its shared libraries and a compatible runtime loader configuration; the binary is not standalone.

## API documentation (GoDoc)

Package and exported-symbol comments are the API reference. Read the checkout's standard GoDoc after the supported [toolchain/native setup](docs/development.md#toolchain-and-installation):

```sh
go doc github.com/zyy17/agora-server-sdk
go doc github.com/zyy17/agora-server-sdk.Session
go doc github.com/zyy17/agora-server-sdk/messaging
go doc github.com/zyy17/agora-server-sdk/mediaio
go doc github.com/zyy17/agora-server-sdk/token
```

Package overviews: [RTC](doc.go), [RTM](messaging/doc.go), [media packets](mediaio/doc.go), [tokens](token/doc.go). GoDoc also includes Go examples. Local `go doc` reflects this checkout; online documentation indexes may lag the moving `dev` branch. The Markdown guides below provide walkthroughs and deployment context.

## Choose an API

| Need | Entry point |
| --- | --- |
| Audio/video, RTC events and data | Root package; [RTC guide](docs/rtc.md) |
| Application messages, Presence, metadata, locks, history, Topics | `messaging`; [RTM guide](messaging/README.md) |
| Native-only options | `agora/rtc`, `agora/rtm`; follow the high-level guides’ ownership restrictions |
| Trusted-server RTC token generation | `token.RTC`; [token contract](docs/rtc.md#events-and-tokens) |
| Runnable scenarios and media file formats | [Examples](examples/README.md) |
| Installation, checks, optional FFmpeg, deployment | [Development](docs/development.md) |

RTC data streams and RTM are different transports. Neither a successful send nor a local queue guarantees recipient processing. RTC queues drop the oldest item; RTM queues drop the newest. Choose explicit backpressure, acknowledgments, and persistence where the application requires them.

## Development checks

```sh
make fmt          # Format Go files
make check        # All default offline checks
```

Default checks do not need credentials or join channels. Optional FFmpeg tests run separately. Offline success does not verify live audio/video interoperability, authentication, project features, or remote playback. See the [validation checklist](docs/development.md#live-integration).

## License

The Go wrapper is [MIT licensed](License). Agora native SDKs and optional third-party codecs retain their own licenses and terms.
