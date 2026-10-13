# Installer fixtures

These tiny, deterministic ZIP files are intentionally invalid native SDKs.

- `incomplete-sdk.zip` contains only `agora_sdk/README.txt`.
- `unsafe-sdk.zip` contains a traversal member, `agora_sdk/../../escaped.txt`.
- `symlink-sdk.zip` contains a symlink to a path outside the SDK directory.

`test_tooling.sh` copies the installer into a temporary directory and changes
only that copy's checksum pin to trust each fixture. It verifies that validation
rejects the archive before replacing an existing installation or populating the
cache. Neither fixture contains executable code.
