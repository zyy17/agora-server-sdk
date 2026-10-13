# RTC and RTM together

[简体中文](README.zh-CN.md) · [Examples](../README.md)

## Purpose

Use a numeric RTC UID and an independent RTM string user ID in one process. Send an opaque binary payload through an RTC data stream, RTC audio metadata, and an RTM message channel. This is a transport example with no application message schema.

## Run

Install the native SDK as described in the [main README](../../README.md). Set these variables outside committed files:

- `AGORA_APP_ID`: project shared by this example's two clients
- `AGORA_USER_ID`: numeric RTC UID, for example `12345`
- `AGORA_CHANNEL`: RTC channel; defaults to `go-sdk-example`
- `AGORA_TOKEN`: caller-issued RTC token matching that channel and numeric UID when required
- `AGORA_RTM_USER_ID`: independent RTM string identity, for example `demo-notifier`
- `AGORA_RTM_TOKEN`: caller-issued RTM token matching that RTM identity when required

```sh
./scripts/with_agora_env.sh go run ./examples/rtc_rtm -rtm-channel go-sdk-notifications -listen 5s
```

Run from the repository root with dedicated authorized test participants. Running joins RTC, publishes about one second of silent mono PCM plus a data message and audio metadata, logs in to RTM, subscribes, and publishes one binary notification. RTM history storage is not requested. A second participant with distinct RTC and RTM identities is needed to observe peer delivery; do not assume self-echo. Ctrl+C starts cleanup.

## Contracts

- Numeric RTC mode does not change RTM identity semantics. RTM channel names and identities are configured independently; matching text does not link the transports.
- Use your trusted server and Agora's official token builders for the required privileges. Do not substitute the string-account-only `token.RTC` helper for numeric UID or RTM tokens. Combined token construction, if used by your server, remains the caller's responsibility; never assume either credential can replace the other.
- The payload includes NUL and non-UTF-8 bytes. RTM uses `PublishOptions{Text: false}`. RTC data and audio metadata have their own limits and delivery semantics; this does not define cross-transport ordering, synchronization, or delivery acknowledgments. RTC data-stream delivery here is neither reliable nor ordered.
- Audio metadata accompanies published audio. It is distinct from RTM channel metadata. This demo sends audio metadata but disables automatic audio subscriptions; receiving a peer's audio metadata requires subscribing to that peer's audio. RTC data messages do not require an audio subscription.
- Three consumers run outside native callbacks. Normal cleanup logs out with a fresh bounded context, closes the RTC and RTM producers, drains their event/message queues, waits for consumers, and reports dropped events. Closing queues does not flush network traffic or prove peer delivery. Token notices identify which client's `RenewToken` needs its own refreshed token.

## Offline checks

```sh
./scripts/with_agora_env.sh go test ./examples/rtc_rtm
./scripts/with_agora_env.sh go build -o ./bin/rtc_rtm ./examples/rtc_rtm
```

The test verifies that a closed queue's buffered binary values are consumed before EOF. Builds and tests do not join RTC or log in to RTM and do not establish live interoperability. See [RTM examples](../messaging/README.md) for standalone operations.
