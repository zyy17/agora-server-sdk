# 安装器测试样本

[English](README.md) · [开发说明](../../docs/development.zh-CN.md)

这些小型、可重复生成的 ZIP 文件是刻意构造的无效 SDK 压缩包，用于安装器回归测试，均不含可执行代码。

- `incomplete-sdk.zip` 仅包含 `agora_sdk/README.txt`。
- `unsafe-sdk.zip` 包含路径穿越条目 `agora_sdk/../../escaped.txt`。
- `symlink-sdk.zip` 包含指向 SDK 目录外的符号链接。

`scripts/test_tooling.sh` 将安装器复制到临时目录，仅修改该副本的校验和配置以接受各样本，再验证安装器会在替换原安装或写入缓存前拒绝无效包。正式安装器及官方校验和不变。
