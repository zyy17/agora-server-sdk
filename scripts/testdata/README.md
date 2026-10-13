# Installer fixtures

[简体中文](README.zh-CN.md) · [Development](../../docs/development.md)

These small, deterministic ZIP files are deliberately invalid SDK archives for installer regression tests. None contains executable code.

- `incomplete-sdk.zip` contains only `agora_sdk/README.txt`.
- `unsafe-sdk.zip` contains the traversal member `agora_sdk/../../escaped.txt`.
- `symlink-sdk.zip` contains a symlink targeting outside the SDK directory.

`scripts/test_tooling.sh` copies the installer into a temporary directory and changes only that copy’s checksum pin to trust each fixture. It verifies rejection before an existing installation is replaced or the cache is populated. The production installer and its official checksum pins remain unchanged.
