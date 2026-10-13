# RTM scenarios

[English](README.md) | [简体中文](README.zh-CN.md)

Build without connecting to Agora:

```sh
make download-agora-libs
./scripts/with_agora_env.sh go build -o bin/messaging ./examples/messaging
```

To run, explicitly set `AGORA_APP_ID` and `AGORA_RTM_USER_ID`, plus
`AGORA_RTM_TOKEN` when tokens are required by your project. Obtain production
RTM tokens from your own server; never put an App Certificate into this example.
Use a dedicated test channel and test user for the mutating scenarios.
Each scenario enables only the subscription events it needs. Ensure your project
has access to the RTM features required by the selected scenario.

```sh
./scripts/with_agora_env.sh ./bin/messaging -scenario pubsub -channel test-room -listen 30s
./scripts/with_agora_env.sh ./bin/messaging -scenario direct -peer another-user
./scripts/with_agora_env.sh ./bin/messaging -scenario presence -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario metadata -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario locks -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario history -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario stream -channel test-stream -peer another-user
```

- `pubsub`: subscribe and publish a text message; optionally listen
- `direct`: publish a message to the required peer
- `presence`: set two `go_example_*` state keys, query users/state/channels,
  then remove those keys
- `metadata`: set/update/read/remove the `go_example` channel/user metadata key,
  and subscribe/unsubscribe user metadata; requires storage feature access
- `locks`: create `go_example_lock` with a 30-second TTL, acquire/query/release/remove
  it; requires lock feature access
- `history`: read one history page; existing messages must have been published
  with history storage enabled, and the history feature must be enabled
- `stream`: create/join a stream, join a topic, subscribe/query/unsubscribe its
  publisher, publish a message, and leave; set `AGORA_RTM_STREAM_TOKEN` if required

Scenarios perform the listed live service operations only when explicitly run.
A scenario that fails midway may leave its earlier state or metadata changes in
place. Use fresh dedicated test resources and inspect them before retrying.
Ctrl+C cancels waiting but does not undo operations already submitted to the
service. Native resources are closed on exit; after a successful login, logout is
attempted with a separate short shutdown timeout. Event queues can drop on
overload; once a client has been created, the example prints the drop count during
shutdown, including on errors. The example does not print tokens or App IDs;
protect any SDK-generated logs.
