# Bounded Agora live verification

The manual workflow checks selected real RTC/RTM paths against Agora. Its
**integration** scope uses synthetic peers, audio, video, and messages; **auth**
checks only RTC join/leave and RTM login/logout. It does not certify every SDK
API, entitlement, codec, deployment environment, or production load pattern.

## Run in GitHub Actions

1. Configure repository Actions **Secrets** named exactly `AGORA_APP_ID` and
   `AGORA_APP_CERTIFICATE`, each containing 32 hexadecimal characters. An Actions
   variable named `AGORA_APP_ID` is also accepted when that secret is absent;
   the certificate must remain a Secret.
2. Merge the workflow and harness into `main`. GitHub requires the workflow file
   on the default branch before offering manual dispatch.
3. Open **Actions → Agora live verification → Run workflow**, choose
   **Branch: main**, then select **integration** (default) or **auth**. Click
   **Run workflow** yourself. The workflow refuses other branches/repositories.
4. Review the exact evaluated SHA in the summary and the separate operation
   results in the live step. Checkout pins the dispatch SHA and checks HEAD
   against it, so it cannot silently follow a moving branch.

There are no push/PR live triggers, secret inputs, or assumed environment
protections. The first step checks presence/format without exposing values.
Build and offline tests receive no secrets. Only the explicit manual live step
uses credentials with Agora. Prefer a dedicated test project: even synthetic,
short-lived media and messaging can count toward Agora usage/billing.

## Integration coverage

Each run creates random string identities and a private random channel name.
Tokens expire after 120 seconds; no external AI/model provider is called.

- RTC: one Service, an audience receiver and a publisher sender; both must
  actually connect. Send about four seconds of 16 kHz mono PCM containing a
  440 Hz test tone, 160×120 I420 video at 10 fps with a known luminance pattern,
  and a small binary data message. The receiver validates the exact source,
  tone spectral content, decoded video pattern, and exact data bytes. Local
  send acceptance alone does not count as delivery. Audio/video are lossy, so
  media assertions tolerate codec changes rather than requiring byte equality.
- RTC renewal: submit a newly generated receiver token. This checks local
  acceptance, not survival past expiry or forced network reconnection.
- RTM: two clients log in, subscribe to the random message channel, publish a
  binary payload without history storage, and require the peer callback to
  match channel, publisher, payload type/custom type, and exact bytes. Renew
  the sender token and verify a transient presence-state round trip, then
  remove exactly that run's state key.
- RTM capability reads: query channel metadata, lock definitions, and one page
  of history on the random channel. These are read-only access checks, not
  proof of metadata/lock mutation or stored-message retrieval. No durable
  metadata, lock definition, or stored message is created.
- Cleanup: unsubscribe/logout both RTM clients, close both RTC sessions, then
  release all clients and the Service. RTC `leave` is the local disconnect/
  release result; it is not a remote leave acknowledgement.

Operations that could not run appear as `not_run`; timeout and failure are
reported separately. A successful two-peer authentication result remains
visible even when a later feature fails. RTM features may need separate project
entitlements; use the numeric SDK code for diagnosis. Any failed, timed-out, or
not-run planned operation makes the overall workflow fail. Token generation
alone never counts as authentication success.

Not covered live: alternate codecs/encoded media and CPU pixel formats, AI/VAD/APM processing,
encryption, proxy/geographic variants, device capture/playback, RTC role and
subscription permutations, RTM stream topics, durable metadata/lock mutations,
persisted history contents, token expiry, forced reconnect/failure injection, or load/stress.
Offline tests cover more API contracts, but do not replace these live gaps.
Authentication success also does not establish that project settings enforce
certificate authentication as intended.

## Example/demo coverage

This harness exercises API paths used by the PCM, raw-video, RTC-data, and RTM
messaging examples with its own deterministic peers. It does **not** execute
those example command-line programs, the voice-agent demo, external AI/provider
integrations, or every example configuration. Existing SDK CI compiles the
examples and runs available offline example tests; compilation is not a live
end-to-end result for those demos.

## Auth-only scope

`auth` creates one media-disabled RTC audience session and one independent RTM
client. It generates join-only RTC and login-only RTM tokens, waits for actual
connection/login success, then leaves/logs out and closes resources. It sends
no audio, video, or application messages. Interpret RTC and RTM independently;
a disabled RTM service does not invalidate successful RTC authentication.

## Bounds and secret handling

RTC and RTM run in separate subprocesses, with a common 25-second operation
context each, 35-second supervisor deadline, and 5-second termination grace.
The second service is attempted even if the first fails. The complete live
command has an outer 90-second hard limit. The job allows 15 minutes for setup
and build; native cleanup cannot keep a hung process alive indefinitely.

Cleanup is attempted on success, failure, and cancellation, with fresh bounded
RTM cleanup contexts. A stuck native call is ultimately killed, so remote
cleanup cannot be guaranteed after hard termination. All application state
changes are limited to this run's random identities/channel and exact transient
presence key; unknown/shared state is never deleted.

Only allowlisted operation/status names and numeric codes are reported. Native
error details, credentials, tokens, identities, and native stdout/stderr are
never forwarded. Tokens remain in child memory and are not printed even as
masking commands. Core dumps are disabled. Native file logs/state remain in
private temp directories, removed per probe and by an unconditional workflow
cleanup step. No live logs or artifacts are uploaded. Do not enable shell
tracing or upload credential-bearing diagnostic files.

## Offline development

With the checksum-pinned native libraries installed:

```sh
./scripts/with_agora_env.sh go test -race ./internal/cmd/agoraverify
./scripts/with_agora_env.sh go build -o /tmp/agora-verify ./internal/cmd/agoraverify
```

Tests use synthetic credentials and fake transports, including received-content
validation, cleanup/error paths, restricted token claims, result sanitization,
and bounded subprocess termination. They do not connect to Agora. `--check`
only validates credential format; `--live` selects auth, while
`--live --features` selects integration. Both live forms require explicit
`AGORA_VERIFY_LIVE=1`. Never put real credentials in command lines or source.

## References

- [GitHub manual dispatch/default-branch registration](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow)
- [GitHub workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#onworkflow_dispatch)
- [Agora Signaling authentication](https://docs.agora.io/en/realtime-media/rtm/build/connect-and-authenticate/authentication-workflow)
- [Official Go token builders](https://github.com/AgoraIO/Tools/tree/95d820182fef/DynamicKey/AgoraDynamicKey/go/src)

The existing token dependency remains pinned at
`v0.0.0-20240807100336-95d820182fef`. Its AccessToken2 API accepts relative seconds,
not Unix timestamps; token-claim tests enforce the 120-second lifetime.
