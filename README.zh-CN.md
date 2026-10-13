# Agora Server SDK for Go

用管理资源生命周期、支持 context 的 Go API，构建服务端实时音视频与消息应用。

[English](README.md) · [RTC 指南](docs/rtc.zh-CN.md) · [RTM 指南](messaging/README.zh-CN.md) · [示例](examples/README.zh-CN.md) · [开发与部署](docs/development.zh-CN.md)

## 功能

- 每个进程一个 RTC Service，独立频道会话、有超时的入会等待、显式资源释放
- 按节奏发送 PCM、逐用户音频、编码音视频、VAD 和音频打断
- 原始视频、订阅控制、RTC 数据流、元数据和连接事件
- 独立 RTM 客户端：消息、在线状态、元数据、锁、历史和 Stream Topic
- 复制后的接收数据、有界队列、可观测丢弃计数和受控原生访问

## 环境与支持范围

使用 Go **1.27.2**、C/C++ 编译器及固定原生库。本项目依赖 cgo，不支持 `CGO_ENABLED=0`。仓库工具链要求与上游最低 Go 版本是不同概念。

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

默认接收格式为有符号 PCM16、小端、16 kHz、单声道。发送时使用 `Broadcaster` 会话：`SendPCM` 接受完整的 10 ms 帧，发送节奏由调用方控制；`PlayPCM` 为 `io.Reader` 控制发送节奏。首次发送自动发布轨道。选择格式和生命周期前，参见 [RTC 约定](docs/rtc.zh-CN.md)。

RTC/RTM Token 应由可信服务端签发。RTC 入会无需 App Certificate；项目开启鉴权时必须提供匹配的 Token，否则可传空字符串。可选纯 Go `token.RTC` 仅用于可信服务端生成 RTC AccessToken2，不签发 RTM Token。不要将 Certificate 和 Token 写入提交文件或日志。

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

## 选择 API

| 需求 | 入口 |
| --- | --- |
| 音视频、RTC 事件和数据 | 根包；[RTC 指南](docs/rtc.zh-CN.md) |
| 业务消息、在线状态、元数据、锁、历史、Topic | `messaging`；[RTM 指南](messaging/README.zh-CN.md) |
| 仅原生层提供的选项 | `agora/rtc`、`agora/rtm`；遵守高层指南中的资源归属限制 |
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
