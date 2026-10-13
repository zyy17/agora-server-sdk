# Agora Server SDK for Go

Build server-side realtime audio, video, and messaging applications with a small, resource-owning Go API.

[简体中文](README.zh-CN.md) · [RTC guide](docs/rtc.md) · [RTM guide](messaging/README.md) · [Examples](examples/README.md)

## Features

- One service, independent channel sessions, context-aware joins, and deterministic cleanup
- Paced PCM playback, per-user received audio, encoded audio, VAD, and interruption
- Raw and encoded video, subscriptions, RTC data messages, metadata, and connection events
- RTM messaging, presence, metadata, distributed locks, history, and stream topics
- Bounded receive queues with visible drop counts and reader-owned frame data
- Native bindings and configuration hooks for advanced integrations

## Requirements

Go **1.27.2**, a C/C++ compiler, and the pinned Agora native libraries. This is a cgo library; `CGO_ENABLED=0` is not supported.

| Platform | Native RTC SDK | Availability |
| --- | --- | --- |
| Linux amd64 | 4.6.31.134 | Primary build and offline test target |
| macOS amd64 / arm64 | 4.6.3.133 | Installer supported; platform runtime validation required |
| Linux arm64 / Windows | — | Not supported by this distribution |

Bindings follow [Agora Go SDK v2.6.5](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/v2.6.5). RTC and RTM have separate native version numbers. Downloads are pinned and SHA-256 verified; version definitions live in [`scripts/sdk_versions.sh`](scripts/sdk_versions.sh).

## Get started

```sh
git clone --branch dev https://github.com/zyy17/agora-server-sdk.git
cd agora-server-sdk
make download-agora-libs
make build-examples
```

Create a service, join a channel, and read audio:

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    agorasdk "github.com/zyy17/agora-server-sdk"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    service, err := agorasdk.NewService(agorasdk.Config{
        AppID: os.Getenv("AGORA_APP_ID"),
    })
    if err != nil { log.Fatal(err) }
    defer service.Close()

    session, err := service.Join(ctx, agorasdk.ChannelConfig{
        Channel: "demo",
        UserID: "server",
        Token: os.Getenv("AGORA_TOKEN"),
    })
    if err != nil { log.Print(err); return }
    defer session.Close()

    for {
        frame, err := session.Audio().Next(ctx)
        if err != nil { return }
        log.Printf("user=%s bytes=%d", frame.UserID, len(frame.Data))
    }
}
```

The default audio format is signed PCM16 little-endian, 16 kHz, mono. Send live frames with `session.SendPCM(data)` or pace a PCM file with `session.PlayPCM(ctx, reader)`. Sending automatically publishes the track.

Supply channel tokens from your application’s token server. Joining does not require an App Certificate. The optional pure-Go `token.RTC` helper generates tokens on a trusted server; never embed its certificate in client applications or examples.

### Use from another Go module

```sh
go get github.com/zyy17/agora-server-sdk@dev
```

Install the pinned native SDK using a checkout of this repository, then run your build through its environment helper from your application directory:

```sh
/path/to/agora-server-sdk/scripts/with_agora_env.sh go build ./...
/path/to/agora-server-sdk/scripts/with_agora_env.sh ./your-binary
```

Deploy the required shared libraries alongside your application and configure the platform’s runtime library search path. The helper configures build and runtime paths; a successful build does not make the binary standalone.

## Choose an API

| Need | API / documentation |
| --- | --- |
| Audio, video, RTC events and data | Root package; [RTC guide](docs/rtc.md) |
| Reliable application messaging and presence | `messaging`; [RTM guide](messaging/README.md) |
| Native-only options and interfaces | `agora/rtc`, `agora/rtm`; [advanced usage](docs/rtc.md#advanced-configuration) |
| Runnable usage scenarios | [Examples](examples/README.md) |

External audio processing requires the corresponding Agora extensions. Optional local decoding requires the `avcodec` build tag and FFmpeg development libraries. Native feature availability also depends on platform, SDK distribution, and Agora project configuration.

## Development

```sh
make fmt          # Format Go files
make check        # Format, tooling, lint, vet, unit/race tests, examples, native smoke
```

Offline checks do not join channels or require credentials. Live media interoperability and enabled RTM features require a separate integration test against your Agora project. See [development and deployment](docs/development.md).

## License

The Go wrapper is [MIT licensed](License). Agora native SDKs and optional third-party codecs retain their own licenses and terms.
