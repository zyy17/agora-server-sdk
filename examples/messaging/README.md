# RTM scenarios

[简体中文](README.zh-CN.md) · [All examples](../README.md) · [RTM reference](../../messaging/README.md)

## Build and credentials

Build from the repository root without logging in:

```sh
make download-agora-libs
./scripts/with_agora_env.sh go build -o bin/messaging ./examples/messaging
```

| Environment variable | Purpose |
| --- | --- |
| `AGORA_APP_ID` | Required project App ID |
| `AGORA_RTM_USER_ID` | Required, unique RTM login identity |
| `AGORA_RTM_TOKEN` | RTM login token; required when project authentication is enabled |
| `AGORA_RTM_STREAM_TOKEN` | Despite the variable name, an **RTC Token** for the Stream Channel/identity; used only by `stream` |

Obtain tokens from a trusted application server; never embed an App Certificate. Use dedicated test users/channels and enable required project features. Different clients must use different login IDs. Only running an executable performs the live operations below.

## Commands

```sh
./scripts/with_agora_env.sh ./bin/messaging -scenario pubsub -channel test-room -listen 30s
./scripts/with_agora_env.sh ./bin/messaging -scenario direct -peer another-user
./scripts/with_agora_env.sh ./bin/messaging -scenario presence -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario metadata -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario locks -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario history -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario stream -channel test-stream -peer another-user
```

| Flag | Default / meaning |
| --- | --- |
| `-scenario` | `pubsub`; one of the seven scenarios below |
| `-channel` | `go-sdk-example`; Message Channel or Stream Channel depending on scenario |
| `-peer` | Empty; required by `direct`; `stream` falls back to its own user ID |
| `-listen` | `0`; nonnegative extra listening duration **after** the scenario finishes |

## Operations and side effects

| Scenario | Operations | Prerequisites / limits |
| --- | --- | --- |
| `pubsub` | Subscribe; publish one text message | Use another online client to verify reception; does not enable history storage |
| `direct` | Publish one text message to `-peer` | Peer must be logged in; no channel subscription |
| `presence` | Set two `go_example_*` state keys; query users/state/channels; remove those keys | Presence access and events enabled |
| `metadata` | Set/update/read/remove `go_example` channel/user metadata; subscribe/unsubscribe user metadata | Storage access; writes explicitly use revision `-1` |
| `locks` | Define `go_example_lock` with TTL 30 seconds; acquire/query/release/remove | Lock access; TTL is retained ownership after disconnect/leave, not lock-definition lifetime |
| `history` | Query one Message Channel history page, count 20 | History enabled and earlier messages published with `StoreInHistory: true`; prints next cursor but does not paginate |
| `stream` | Create/join Stream; join Topic; subscribe to publisher; publish; query/unsubscribe publisher; leave Topic/Stream; close | RTC Stream token as required; Presence access because the example enables it |

The stream topic is `go_example_topic`. `Subscribers` reports publisher IDs the client subscribes to, not its audience. The stream is unsubscribed, left, and closed **before** `-listen` begins; increasing that flag does not turn this scenario into a persistent topic receiver. For reception testing, run distinct clients and adapt the lifetime so both stay joined.

## Shutdown and verification

A scenario that fails midway can leave earlier messages/state/metadata/lock changes in place. Ctrl+C cancels waiting, not operations already submitted; inspect test resources before retrying. The normal path performs the cleanup listed above, but it is not a transaction or rollback mechanism.

After a successful login, shutdown attempts `Logout` with a separate three-second context, then releases native resources. Once a client is created, cleanup prints its event drop count, including on failure. Event overflow may lose messages, link changes, or token notifications. The example only logs token-renewal notices; implement `Client.RenewToken` for RTM tokens and `Stream.RenewToken` for the stream’s RTC token before using long sessions.

The example does not print tokens or App IDs; protect SDK-generated logs separately. Compilation/offline tests do not establish live delivery, feature permissions, history availability, or correct peer behavior. Follow the [RTM contracts](../../messaging/README.md) and [live integration checklist](../../docs/development.md#live-integration).
