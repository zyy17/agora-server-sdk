# 开发与部署

[English](development.md) · [首页](../README.zh-CN.md)

## 工具链与安装

使用 Go 1.27.2 和原生 C/C++ 工具链。Ubuntu 需要 `build-essential`、`curl` 和 `unzip`；macOS 需要 Xcode Command Line Tools。头文件、库、操作系统及架构须匹配仓库固定配置。主要目标为 Linux amd64；上游限制 macOS 仅用于开发测试。安装脚本不支持 Linux arm64 和 Windows。

```sh
make download-agora-libs
make check
```

安装脚本校验压缩包 SHA-256，拒绝不安全或不完整内容；验证通过后才替换原安装。使用已下载且匹配的官方压缩包：

```sh
./scripts/download_agora_libs.sh --archive /path/to/sdk.zip
```

| 设置或文件 | 用途 |
| --- | --- |
| `AGORA_LIBS_DIR` | 安装及库目录；默认检出目录内的 `agora_libs` |
| `AGORA_SDK_CACHE_DIR` | 按校验和保存的压缩包缓存；默认位于用户缓存目录下 |
| `scripts/sdk_versions.sh` | 固定工具链和原生版本、官方地址及 SHA-256 |
| `scripts/with_agora_env.sh` | 校验安装、启用 cgo、配置链接及运行时库路径 |
| `OUTPUT_BIN_PATH` | 示例二进制目录；默认 `bin` |
| `GO` | Make 使用的 Go 可执行文件；默认 `go` |

不要为接受任意压缩包而替换校验和，也不要混用旧原生包和仓库头文件。升级分发包时应同时更新地址、版本和校验和，并重新验证原生行为。`make sync-agora-libs-headers` 从固定 Linux SDK 更新仓库头文件，会修改受版本管理的文件，不属于常规安装步骤。

## 离线检查

| 命令 | 范围 |
| --- | --- |
| `make fmt` | 格式化全部 Go 源码及导入分组，会修改文件 |
| `make fmt-check` | 检查源码格式及导入分组 |
| `make test-tooling` | Shell 语法、安装事务、压缩包安全及格式化回归测试 |
| `make lint` | 固定 Staticcheck v0.8.2，检查高层 API 与示例 |
| `make vet` | 全部 Go 包，包括原生绑定 |
| `make test` | 离线单元测试 |
| `make race` | 使用 Go race detector 运行离线测试 |
| `make build-examples` | 编译全部可执行示例 |
| `make native-smoke` | 无凭据的原生生命周期，启用 `cgocheck2` 和 race 检查 |
| `make test-callbacks` | 用合成数据测试已注册的 C 回调，启用 `-tags agora_test`、`cgocheck2` 和 race 检查 |
| `make check` | 上述默认检查，不包含会修改文件的 `fmt` |
| `make test-avcodec` | 可选解码器测试，不属于 `make check` |
| `make clean` | 删除生成的示例二进制，保留 SDK 和缓存 |

Staticcheck 检查根包、`messaging`、`mediaio`、`token` 和示例；原生绑定由 vet 和测试覆盖。Go race detector 不检测 C/C++ SDK 内部竞态。测试使用模拟传输和回调，以及无凭据的原生生命周期路径，不要求加入频道。

CI 使用 Ubuntu 24.04 / Linux amd64 / Go 1.27.2。一个任务运行 `make check`，并确认默认构建不依赖系统 FFmpeg 开发包；另一个任务安装 FFmpeg 依赖，执行 `go test -race -tags avcodec -count=1 ./...`。这些检查不验证 macOS 或真实服务互通。

## Go 代码风格

`make fmt` 和 `make fmt-check` 使用固定版本 `goimports` v0.51.0 的仅格式化模式：先标准库，再外部模块，最后本模块，各组之间空一行。保留现有导入及别名，不推断或删除导入。格式化范围包括测试、示例、原生 Go 封装和未启用 build tag 的文件；排除已下载的 SDK、vendor 依赖、二进制及工具/缓存目录。cgo 前导注释保持紧邻 `import "C"`。CI 通过 `make check` 执行同一检查。

Make 将固定版本的格式化工具安装到 `.tools/bin`，首次安装需要访问模块下载源。格式化及其回归测试不需要原生 SDK、网络访问或运行时依赖变更。编辑器可使用 `.tools/bin/goimports -format-only -local github.com/zyy17/agora-server-sdk/` 保持一致。

函数应聚焦单一步骤。初始化或事件处理变长时，可按校验、资源创建和单事件处理提取具名辅助函数，并在调用处清楚展示资源归属与清理逻辑；不要仅为满足任意行数限制而拆分代码。

## 可选 FFmpeg

默认构建不需要 FFmpeg 开发库。Ubuntu 上启用 Session 转码辅助接口：

```sh
sudo apt-get install pkg-config libavcodec-dev libavformat-dev libavutil-dev libswresample-dev
make test-avcodec
./scripts/with_agora_env.sh go build -tags avcodec ./...
```

`-tags avcodec` 开启异步解码并发布的辅助接口；接受输入不表示处理完成，也不保证关闭时排空。需要头文件及 pkg-config 信息，仅安装 `ffmpeg` 可执行文件不够。应在目标平台分别验证可选构建和实际媒体。

`examples/send_media` 在运行时启动 `ffmpeg` 可执行文件解码，与上述构建路径不同。它不需要 `avcodec` 或 FFmpeg 开发头文件；仅安装开发库也不保证存在该可执行文件。

## 真实联调

使用专用项目、测试频道及不同用户 ID，先开通所需功能。RTC Token 与 RTM 登录 Token 不同；RTM Stream Channel 的加入和续期使用 RTC Token。App ID、身份、频道、权限和有效期必须匹配。通过受控环境或密钥配置传入凭据，不要写入源文件或命令日志。

至少验证：

1. 两个独立客户端入会或登录、发布、订阅、接收和离开。
2. PCM 节奏、逐用户路由、取消发送后打断、视频格式及时间戳对齐。
3. AI-server/AI-client 兼容、所需编解码器、加密及可选 APM 扩展。
4. RTC/RTM Token 续期、重复身份、断网及重连。
5. 重复创建关闭、并发请求、队列过载及正常退出。
6. 已开通的 RTM Presence、存储版本、锁、历史分页及 Stream 发布者订阅。

真实示例可能发送或录制媒体，也可能发布消息、修改状态、元数据和锁。录制前须取得参与者授权，并使用可清理的测试资源。报错或 Ctrl+C 不会撤销已完成操作，重试前先检查状态。构建或测试通过不能证明远端接收、播放或业务处理完成。

## 部署与排查

随应用部署固定动态库，并配置 Linux `LD_LIBRARY_PATH`；macOS 开发测试需配置对应加载器。环境脚本在 macOS 设置 `DYLD_LIBRARY_PATH`，不是通用生产打包方案。验证环境应与部署使用相同平台、架构及原生分发包；不要用旧 ARM 包替代固定 x86 包。

| 现象 | 检查项 |
| --- | --- |
| SDK 缺失、不完整或过期 | 重新安装固定包，确认 `AGORA_LIBS_DIR` 指向已验证目录 |
| 链接器找不到库 | 通过 `with_agora_env.sh` 构建，检查架构与工具链 |
| 编译成功但运行时加载失败 | 部署必要动态库，检查运行时搜索路径 |
| `ErrServiceActive` | 先关闭已有 RTC Service；RTM 客户端独立管理 |
| APM 初始化失败 | 确认所需扩展存在；固定 Linux 包缺少 AINS |
| 无媒体或事件 | 检查 Token、角色、发布端、订阅与观察者设置、对端兼容性及丢弃计数 |
| RTM 写入后超时 | 不要假设已回滚；重试前检查远端状态 |

原生日志和缓存可能包含运行数据。设置 RTC `Config.LogPath`、`Config.ConfigDir`、`Config.DataDir`，通过 `Config.Native` 配置 RTM 日志，并实施访问和保留策略。监控队列丢弃数及 Token 有效期，不要记录密钥。原生 SDK 和编解码器许可证独立于 Go 封装的 MIT 许可证。
