# Agora Server SDK for Go

用管理资源生命周期、支持 context 的 Go API，构建服务端实时音视频与消息应用。

[English](README.md) · [GoDoc](#api-文档godoc) · [RTC 指南](docs/rtc.zh-CN.md) · [RTM 指南](messaging/README.zh-CN.md) · [示例](examples/README.zh-CN.md) · [API 覆盖范围](docs/api-coverage.zh-CN.md) · [开发与部署](docs/development.zh-CN.md)

## 功能

- 每个进程一个 RTC Service，显式选择数值或字符串身份，独立频道会话、有界入会等待和资源释放
- 独立的收发 PCM 格式、可变时长 PCM、调速播放、逐用户音频、VAD 和音频打断
- CPU/编码视频、角色与轨道控制、RTC 数据流、元数据和连接事件
- 独立 RTM 客户端：消息、在线状态、元数据、锁、历史和 Stream Topic
- 复制后的接收队列、可选同步媒体决策、独立资源管理的 VAD/解码器，以及受控原生访问

## 与官方 SDK 的差异

本独立 Go 模块基于官方 Agora Go SDK v2.6.5，增加高层契约与定向绑定修复。底层媒体和消息能力仍由 Agora 提供，不能只替换 import 就兼容原版。

| 官方 v2.6.5 | 本项目增加 |
| --- | --- |
| 进程初始化、连接、observer | 管理资源的 `Service`/`Session`、有界入会等待、Go 类型化错误 |
| 媒体与结果回调 | context 读取/请求、有界队列与丢弃计数 |
| PCM、编码媒体、VAD、打断、RTM 功能 | 带校验的组合、PCM 调速助手、明确所有权 |
| 原生绑定与安装器 | 定向 ABI/生命周期修复、校验和固定安装、离线检查 |

详见[完整对比文档](docs/upstream-comparison.zh-CN.md)，包含固定基线、默认值变化、上游已有能力、迁移限制与源码依据。 [API 覆盖清单](docs/api-coverage.zh-CN.md)检查官方 Go 的全部 326 个函数与方法，以及导出字段、类型和常量，记录有意的兼容性修复，并区分高层、受控高级访问、仅底层与不可用能力。保留符号本身不等于完整功能覆盖。

## 环境与支持范围

使用 Go **1.27.2**。RTC/RTM 包还需要 C/C++ 编译器、固定原生库和 cgo，不支持 `CGO_ENABLED=0`；`mediaio` 和 `token` 包为纯 Go。仓库工具链要求与上游最低 Go 版本是不同概念。

| 平台 | 原生 RTC 包 | 范围 |
| --- | --- | --- |
| Linux amd64 | 4.6.31.134 | 主要构建及离线 CI 平台 |
| macOS amd64 / arm64 | 4.6.3.133-20260618 | 支持安装；上游仅允许用于开发测试，不用于生产 |
| Linux arm64 / Windows | — | 当前分发配置不支持 |

绑定基于 [Agora Go SDK v2.6.5](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/v2.6.5)，平台限制见其[中文说明](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/v2.6.5/README.zh.md)。下载版本及 SHA-256 固定于 [`scripts/sdk_versions.sh`](scripts/sdk_versions.sh)。RTC 分发包、Go 绑定和 RTM 核心各有独立版本；功能还取决于平台、扩展库、对端 SDK 及项目配置。

## 快速开始

```sh
git clone --branch dev https://github.com/zyy17/agora-server-sdk.git
cd agora-server-sdk
make download-agora-libs
make build-examples
```

上述命令安装库并编译示例，不加入频道。以下只接收程序需要 `AGORA_APP_ID`、`AGORA_CHANNEL`、唯一的 `AGORA_USER_ID`；开启鉴权时还需匹配的 `AGORA_TOKEN`。在同项目、同频道中启动另一个有发布权限的客户端，提供音频。

```go
package main

import (
    "context"
    "errors"
    "io"
    "log"
    "os"
    "os/signal"
    "syscall"

    agorasdk "github.com/zyy17/agora-server-sdk"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    if err := run(ctx); err != nil {
        log.Print(err)
    }
}

func run(ctx context.Context) (err error) {
    service, err := agorasdk.NewService(agorasdk.Config{
        AppID: os.Getenv("AGORA_APP_ID"),
    })
    if err != nil {
        return err
    }
    defer func() { err = errors.Join(err, service.Close()) }()
    session, err := service.Join(ctx, agorasdk.ChannelConfig{
        Channel: os.Getenv("AGORA_CHANNEL"),
        UserID:  os.Getenv("AGORA_USER_ID"),
        Token:   os.Getenv("AGORA_TOKEN"),
        Role:    agorasdk.Audience,
    })
    if err != nil {
        return err
    }
    defer func() { err = errors.Join(err, session.Close()) }()
    for {
        frame, nextErr := session.Audio().Next(ctx)
        if errors.Is(nextErr, io.EOF) {
            return nil
        }
        if nextErr != nil {
            return nextErr
        }
        log.Printf("user=%s bytes=%d", frame.UserID, len(frame.Data))
    }
}
```

将代码保存为应用的 `main.go`，通过下方环境脚本运行。示例在 Ctrl+C/SIGTERM 时退出，并显式关闭 Session 和 Service。生产应用还应独立消费会话事件、更新 Token 和监控队列丢弃数。

默认接收格式为有符号 PCM16、小端、16 kHz、单声道。发送时使用 `Broadcaster` 会话：`SendPCM` 接受完整采样且时长为整数毫秒的 PCM 块（44.1 kHz 要求 10 ms 的整数倍），发送节奏由调用方控制；`PlayPCM` 为 `io.Reader` 控制发送节奏。首次发送自动发布轨道。`Audio.Send` 与 `Audio.Receive` 独立配置；`Config.IdentityMode` 选择字符串账号（默认）或数值 UID。选择格式和生命周期前，参见 [RTC 约定](docs/rtc.zh-CN.md)。

RTC/RTM Token 应由可信服务端签发。RTC 入会无需 App Certificate；项目开启鉴权时必须提供匹配的 Token，否则可传空字符串。可选纯 Go `token.RTC` 仅用于可信服务端生成 RTC AccessToken2，使用字符串账号，不签发 RTM 或 RTC+RTM 联合 Token。需要数值身份或联合权限时，保留已有可信 Token 签发逻辑。不要将 Certificate 和 Token 写入提交文件或日志。

### 在其他 Go 项目中使用

```sh
go get github.com/zyy17/agora-server-sdk@dev
```

通过本仓库检出目录安装固定 SDK，再从应用目录执行：

```sh
/path/to/agora-server-sdk/scripts/with_agora_env.sh go build ./...
/path/to/agora-server-sdk/scripts/with_agora_env.sh go run .
/path/to/agora-server-sdk/scripts/with_agora_env.sh ./your-binary
```

`@dev` 指向可变分支；需要可重复构建时固定已审查的提交。环境脚本配置已安装 SDK 的 cgo、链接和运行路径。部署仍需携带动态库并配置兼容的加载路径，二进制不能独立运行。

## API 文档（GoDoc）

包与导出符号注释是 API 参考。完成受支持的[工具链与原生库配置](docs/development.zh-CN.md#工具链与安装)后，在当前 checkout 阅读标准 GoDoc：

```sh
go doc github.com/zyy17/agora-server-sdk
go doc github.com/zyy17/agora-server-sdk.Session
go doc github.com/zyy17/agora-server-sdk/messaging
go doc github.com/zyy17/agora-server-sdk/mediaio
go doc github.com/zyy17/agora-server-sdk/token
```

包概览：[RTC](doc.go)、[RTM](messaging/doc.go)、[媒体包](mediaio/doc.go)、[token](token/doc.go)。GoDoc 也包含 Go 示例。本地 `go doc` 对应当前代码，在线文档索引可能落后于持续变化的 `dev` 分支。以下 Markdown 指南补充使用流程与部署背景。

## 选择 API

| 需求 | 入口 |
| --- | --- |
| 音视频、RTC 事件和数据 | 根包；[RTC 指南](docs/rtc.zh-CN.md) |
| 业务消息、在线状态、元数据、锁、历史、Topic | `messaging`；[RTM 指南](messaging/README.zh-CN.md) |
| 媒体布局、回调、VAD 与可选解码器 | [媒体契约](docs/media-api.zh-CN.md)与 [API 覆盖范围](docs/api-coverage.zh-CN.md) |
| 高级/底层选项 | `ConfigureNative`、`WithNative`、`agora/rtc`、`agora/rtm`；遵守所有权限制 |
| 可信服务端生成 RTC Token | `token.RTC`；[Token 约定](docs/rtc.zh-CN.md#事件与-token) |
| 可运行场景和媒体文件格式 | [示例](examples/README.zh-CN.md) |
| 安装、检查、可选 FFmpeg、部署 | [开发说明](docs/development.zh-CN.md) |

RTC 数据流和 RTM 是不同传输机制。发送成功和本地队列都不保证接收应用已处理。RTC 队列丢弃最旧项，RTM 队列丢弃最新项。业务需要时，应显式设计背压、回执和持久化。

## 开发检查

```sh
make fmt          # 格式化 Go 文件
make check        # 默认完整离线检查
```

默认检查不需要凭据，也不加入频道。可选 FFmpeg 测试独立运行。离线通过不代表真实音视频互通、鉴权、项目功能或远端播放已验证，详见[联调清单](docs/development.zh-CN.md#真实联调)。

## 许可证

Go 封装采用 [MIT 许可证](License)。Agora 原生 SDK 与可选第三方编解码器遵循各自许可证和服务条款。
