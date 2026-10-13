# Development and deployment

[简体中文](development.zh-CN.md) · [Home](../README.md)

## Toolchain and installation

Use Go 1.27.2 and a native C/C++ toolchain. Ubuntu requires `build-essential`, `curl`, and `unzip`; macOS requires Xcode Command Line Tools. Keep headers, libraries, OS, and architecture aligned with the repository pin. Linux amd64 is the primary target; upstream limits macOS to development/testing. Linux arm64 and Windows are not installer targets.

```sh
make download-agora-libs
make check
```

The installer verifies the selected archive’s SHA-256, rejects unsafe/incomplete contents, and replaces an existing installation only after validation. To use an already downloaded matching official archive:

```sh
./scripts/download_agora_libs.sh --archive /path/to/sdk.zip
```

| Setting / file | Purpose |
| --- | --- |
| `AGORA_LIBS_DIR` | Install/library directory; default `agora_libs` in the checkout |
| `AGORA_SDK_CACHE_DIR` | Checksum-addressed archive cache; default under the user cache directory |
| `scripts/sdk_versions.sh` | Pinned toolchain/native versions, official URLs, SHA-256 values |
| `scripts/with_agora_env.sh` | Validates installation, enables cgo, sets linker and runtime library paths |
| `OUTPUT_BIN_PATH` | Example binary directory; default `bin` |
| `GO` | Go executable used by Make; default `go` |

Do not replace a checksum to accept an arbitrary archive or mix an older native package with the vendored headers. Change URL/version/hash together only when intentionally upgrading the distribution and rerun native validation. `make sync-agora-libs-headers` refreshes vendored headers from the pinned Linux SDK; it changes tracked files and is not a normal installation step.

## Offline checks

| Command | Coverage |
| --- | --- |
| `make fmt` | Format all Go source; modifies files |
| `make fmt-check` | Reject unformatted source |
| `make test-tooling` | Shell syntax, installer transactions, archive safety |
| `make lint` | Pinned Staticcheck v0.8.2 on first-party APIs and examples |
| `make vet` | All Go packages, including native bindings |
| `make test` | Offline unit tests |
| `make race` | Offline tests with Go race detector |
| `make build-examples` | Compile all executable examples |
| `make native-smoke` | Credential-free native lifecycle with `cgocheck2` and race checks |
| `make test-callbacks` | Synthetic registered C callbacks with `-tags agora_test`, `cgocheck2`, and race checks |
| `make check` | All default checks above except mutating `fmt` |
| `make test-avcodec` | Optional decoder tests; separate from `make check` |
| `make clean` | Remove generated example binaries; keep SDK and caches |

Staticcheck covers the root package, `messaging`, `mediaio`, `token`, and examples; native bindings are covered by vet and tests. Go’s race detector does not instrument the C/C++ SDK internals. Tests use fake transports/callbacks and credential-free native lifecycle paths; none require a channel join.

CI uses Ubuntu 24.04 / Linux amd64 / Go 1.27.2. One job runs `make check` and verifies default builds do not require system FFmpeg development packages. A separate job runs `go test -race -tags avcodec -count=1 ./...` with FFmpeg dependencies. This does not validate macOS or live service interoperability.

## Optional FFmpeg

Default builds do not require FFmpeg development libraries. To enable Session transcoding helpers on Ubuntu:

```sh
sudo apt-get install pkg-config libavcodec-dev libavformat-dev libavutil-dev libswresample-dev
make test-avcodec
./scripts/with_agora_env.sh go build -tags avcodec ./...
```

`-tags avcodec` enables asynchronous decode-and-publish helpers; their input acceptance is not a completion or drain guarantee. Headers and pkg-config metadata are required; installing the `ffmpeg` executable alone is insufficient. Test the optional build and actual media on the target platform.

The `examples/send_media` program is different: it launches the `ffmpeg` executable to decode a file at runtime. It does not require `avcodec` or FFmpeg development headers. Installing development libraries alone does not guarantee that executable is present.

## Live integration

Use a dedicated project, test channels, and distinct user IDs. Provision the required features first. RTC and RTM login tokens are different; RTM Stream Channel join/renewal uses an RTC token. Match App ID, identity, channel, privileges, and expiry. Pass credentials through controlled environment/secret configuration, never source files or command logs.

Validate at least:

1. Two independent clients joining/logging in, publishing, subscribing, receiving, and leaving.
2. PCM pacing, per-user routing, producer cancellation plus interruption, video formats, and timestamp alignment.
3. Compatible AI-server/AI-client behavior, requested codecs, encryption, and optional APM extensions.
4. RTC/RTM token renewal, duplicate identity behavior, network interruption, and reconnection.
5. Repeated create/close, concurrent requests, queue overload, and graceful shutdown.
6. Enabled RTM Presence, storage revisions, locks, history pagination, and Stream publisher subscriptions.

Live examples can publish/record media or mutate messages, state, metadata, and locks. Obtain participant authorization before recording and use disposable test resources. An error or Ctrl+C does not undo completed operations; inspect state before retrying. Build/test success does not establish remote receipt, playback, or application processing.

## Deployment and troubleshooting

Ship the pinned shared libraries with the application and configure `LD_LIBRARY_PATH` on Linux or appropriate macOS loader settings for development/testing. The helper sets `DYLD_LIBRARY_PATH` on macOS; it is not a general production packaging solution. Test on the same OS/architecture/native distribution as deployment; do not substitute an older ARM archive for the pinned x86 package.

| Symptom | Check |
| --- | --- |
| Missing/incomplete/outdated SDK | Reinstall the pinned archive; verify `AGORA_LIBS_DIR` points to the validated directory |
| Linker cannot find a library | Build through `with_agora_env.sh`; check architecture and toolchain |
| Binary builds but loader fails | Deploy required shared libraries; inspect runtime search paths |
| `ErrServiceActive` | Close the existing RTC Service; RTM clients are separately owned |
| APM initialization failure | Verify the requested extensions exist; pinned Linux lacks AINS |
| No media/events | Verify token, role, publisher, subscription/observer settings, peer compatibility, and drop counters |
| RTM timeout after write | Do not assume rollback; inspect remote state before retrying |

Native logs/caches may contain operational data. Set RTC `Config.LogPath`, `Config.ConfigDir`, and `Config.DataDir`, and configure RTM logging through `Config.Native`. Apply application access/retention controls. Monitor queue drops and token expiry without storing secrets in logs. Native SDK and codec licensing remains separate from the Go wrapper’s MIT license.
