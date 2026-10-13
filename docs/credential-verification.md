# Agora credential verification: authentication only

This manual check validates **live RTC authentication and RTM authentication
separately**. It is not a full-feature SDK test and does not validate media,
message delivery, AI/model providers, renewal, reconnect, or production load.

## Run in GitHub Actions

1. Set repository Actions **Secrets** with these exact names:
   `AGORA_APP_ID` and `AGORA_APP_CERTIFICATE`. Each must be exactly 32 hexadecimal
   characters. An Actions variable named `AGORA_APP_ID` is also accepted if the
   corresponding secret is absent; keep the certificate in Secrets only.
2. The workflow file `.github/workflows/agora-verify.yml` must be registered on
   the default branch for GitHub to offer manual dispatch. Its complete
   implementation must be present on `main`.
3. Open **Actions → Agora credential verification → Run workflow**, select
   **Branch: main**, and click **Run workflow** yourself. The workflow refuses
   other branches and other repositories. There are no secret inputs,
   automatic push/PR triggers, or assumed environment-protection rules.
4. Review the exact evaluated commit SHA in the run summary and the separate
   RTC and RTM lines in the live-check step. Checkout uses the dispatch SHA, not
   a branch tip that could change after the click.

The first step validates presence/shape without exposing values and fails
clearly when a secret is absent. Build and offline tests do not receive secrets.
Only this explicit manual live step uses the credentials to contact Agora.
Use a dedicated test project if available: short-lived connections can still
count toward Agora usage or billing.

## What constitutes success

- RTC: generate a 120-second, join-only AccessToken2 for a random string account
  and channel, initialize one Service, receive successful channel connection,
  then call session Close (disconnect/release) and Service Close. The `leave`
  result is the local disconnect/release result, not a remote leave receipt.
- RTM: independently generate a 120-second RTM login token for a new random
  string account, receive a successful login result, then attempt logout with a
  fresh 5-second context and close the client. No message or stream channel is
  created.
- Token generation alone never counts as authentication success. A failed or
  unavailable RTM service does not invalidate an independently successful RTC
  result. RTM may require separate project enablement; interpret numeric errors
  with the pinned SDK's service-specific error-code documentation.
- A failed or incomplete probe makes the workflow fail. Format checks and
  authentication success do not establish every account capability or prove
  that the project's authentication settings enforce certificates as expected.

## Bounds and secret handling

RTC and RTM run in separate subprocesses. Each has a 25-second operation budget,
35-second supervisor deadline, and 5-second termination grace. The second
service is still attempted if the first fails. The entire live command has an
outer hard limit of 90 seconds; the workflow job allows 15 minutes for build and
installation. Native cleanup is always attempted for created resources on
normal/error/cancellation paths. A hung native call is ultimately terminated,
so remote cleanup cannot be guaranteed after a hard kill.

Only allowlisted operation/status strings and numeric codes are reported. SDK
error details, tokens, identities, certificate values, and native stdout/stderr
are never forwarded. Derived tokens remain in child memory and are never
printed, even as masking commands. Native file logs and SDK state stay in private
temporary directories and are removed after each probe and by an unconditional
workflow cleanup step. No logs or other live artifacts are uploaded. Do not turn
on shell tracing, native diagnostic output, or credential-bearing artifacts.

## Offline development

With the checksum-pinned native libraries installed, run:

```sh
./scripts/with_agora_env.sh go test -race ./internal/cmd/agoraverify
./scripts/with_agora_env.sh go build -o /tmp/agora-verify ./internal/cmd/agoraverify
```

Tests use synthetic credentials and fake transports, including error-path
cleanup, independent result reporting, restricted token claims, and rejection
of untrusted result fields. They do not connect to Agora. The built command's
`--check` mode only checks credential format; `--live` additionally requires
`AGORA_VERIFY_LIVE=1`. Never place real credentials on command lines or in source.

## References

- [GitHub manual dispatch and default-branch registration](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow)
- [GitHub workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#onworkflow_dispatch)
- [Agora Signaling token authentication](https://docs.agora.io/en/realtime-media/rtm/build/connect-and-authenticate/authentication-workflow)
- [Official Agora Go token builders](https://github.com/AgoraIO/Tools/tree/95d820182fef/DynamicKey/AgoraDynamicKey/go/src)

The token dependency is already pinned in `go.mod` at
`v0.0.0-20240807100336-95d820182fef`. Its AccessToken2 builder accepts relative
seconds for expiry; token-claim tests lock this down rather than copying an
absolute timestamp from a documentation example.
