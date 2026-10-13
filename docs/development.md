# Development and deployment / 开发与部署

[English overview](../README.md) · [中文首页](../README.zh-CN.md)

## Toolchain / 工具链

Use Go 1.27.2 and a native C/C++ toolchain. On Ubuntu, install `build-essential`, `curl`, and `unzip`. On macOS, install the Xcode Command Line Tools. The native SDK must match the headers and platform selected by the repository.

使用 Go 1.27.2 和原生 C/C++ 工具链。Ubuntu 需要 `build-essential`、`curl`、`unzip`；macOS 需要 Xcode Command Line Tools。原生库必须与本仓库的头文件及平台配置匹配。

```sh
make download-agora-libs
make check
```

The installer verifies checksums before replacing an installation. To use an already downloaded official archive:

安装脚本先校验文件，再替换安装目录。使用已经下载的官方压缩包：

```sh
./scripts/download_agora_libs.sh --archive /path/to/sdk.zip
```

- `AGORA_LIBS_DIR`: installed native library directory / 原生库安装目录
- `AGORA_SDK_CACHE_DIR`: checksum-addressed archive cache / 按校验和保存的压缩包缓存
- `scripts/sdk_versions.sh`: pinned URLs, versions and hashes / 固定下载地址、版本与校验和

## Checks / 检查

| Command | Purpose / 用途 |
| --- | --- |
| `make fmt` | Format Go files / 格式化 |
| `make fmt-check` | Reject unformatted source / 检查格式 |
| `make test-tooling` | Shell syntax and installer safety / 脚本及安装安全检查 |
| `make lint` | Pinned Staticcheck on first-party APIs and examples / 高层 API 与示例静态检查 |
| `make vet` | Vet all packages, including native bindings / 检查全部包 |
| `make test` | Offline unit tests / 离线单测 |
| `make race` | Go race detector / Go 竞态检测 |
| `make build-examples` | Compile runnable examples / 编译示例 |
| `make native-smoke` | Native lifecycle/linkage without joining a channel / 不入会的原生冒烟检查 |
| `make check` | All default checks above / 默认完整检查 |

Staticcheck covers the root API, `messaging`, `mediaio`, `token`, and examples; imported native bindings are covered by `go vet` and their tests. The Go race detector does not instrument the native SDK’s C/C++ internals.

Staticcheck 检查根包 API、`messaging`、`mediaio`、`token` 和示例；引入的原生绑定通过 `go vet` 及对应测试检查。Go race detector 不检测原生 SDK 内部的 C/C++ 竞态。

CI runs the default offline checks on Linux amd64. It does not require Agora credentials or join a channel. Offline tests cannot establish real network, codec, token, or remote playback interoperability.

CI 在 Linux amd64 上运行默认离线检查，不需要 Agora 凭据，也不加入频道。离线测试不能证明真实网络、编解码、Token 或远端播放互通正确。

## Optional FFmpeg / 可选 FFmpeg

Default builds do not require FFmpeg development libraries. For local decoding helpers on Ubuntu:

默认构建不需要 FFmpeg 开发库。Ubuntu 上启用本地解码辅助能力：

```sh
sudo apt-get install pkg-config libavcodec-dev libavformat-dev libavutil-dev libswresample-dev libswscale-dev
make test-avcodec
./scripts/with_agora_env.sh go build -tags avcodec ./...
```

Installing an `ffmpeg` executable alone is insufficient: headers and pkg-config metadata are required. Validate the optional build and actual media on the target platform separately.

仅安装 `ffmpeg` 可执行文件并不够，还需要头文件及 pkg-config 信息。可选构建与真实媒体处理应在目标平台分别验证。

## Live integration / 真实联调

Use a dedicated Agora test project, channel, and user IDs. Obtain RTC and RTM tokens for the correct service, identity, channel and expiration. Pass them through the environment; never commit them or an App Certificate.

使用独立的 Agora 测试项目、频道和用户 ID。RTC 与 RTM Token 必须分别匹配对应服务、身份、频道及有效期。通过环境变量传入，不要提交 Token 或 App Certificate。

Validate at least:

至少验证：

1. Two clients joining, subscribing, receiving and leaving / 双端入会、订阅、接收和离会
2. PCM timing, per-user routing, interruption and video formats / PCM 节奏、逐用户路由、打断与视频格式
3. Token renewal, network interruption and reconnection / Token 续期、断网与重连
4. Repeated create/close, queue overload and shutdown / 重复创建关闭、队列过载与退出
5. Each enabled RTM feature and its project permissions / 已开通的 RTM 功能与项目权限

Some RTM examples publish data or modify presence, metadata and locks. Read their scenario descriptions before running them. Cancellation can stop waiting after an operation was submitted; inspect remote state before retrying a mutating operation.

部分 RTM 示例会发送消息，或修改状态、元数据与锁。运行前阅读对应场景说明。操作提交后取消等待不一定撤销远端操作；重试写操作前先检查远端状态。

## Deployment / 部署

Ship the pinned native shared libraries and configure `LD_LIBRARY_PATH` (Linux) or the appropriate macOS loader configuration. Use `scripts/with_agora_env.sh` during development. Test the same OS/architecture and native distribution as production; do not substitute an older ARM package for the pinned x86 SDK.

部署时携带固定版本的原生动态库，并配置 Linux 的 `LD_LIBRARY_PATH` 或 macOS 对应加载器设置。开发时可使用 `scripts/with_agora_env.sh`。测试环境应与生产使用相同的平台及原生分发包，不要用较旧的 ARM 包替代固定的 x86 SDK。

Agora logs and caches can contain operational data. Set `Config.LogPath`, `Config.ConfigDir` and `Config.DataDir` to application-controlled locations and apply your retention and access policy.

Agora 日志及缓存可能包含运行数据。通过 `Config.LogPath`、`Config.ConfigDir` 和 `Config.DataDir` 指定应用管理的路径，并按业务要求控制权限及保留周期。
