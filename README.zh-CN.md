# Agora Server SDK for Go

用简洁、自动管理资源的 Go API，构建服务端实时音视频与消息应用。

[English](README.md) · [RTC 使用指南](docs/rtc.zh-CN.md) · [RTM 使用指南](messaging/README.zh-CN.md) · [示例](examples/README.zh-CN.md)

## 功能

- 单一 Service、独立频道会话、支持 context 的入会与确定性资源释放
- 按节奏发送 PCM、接收逐用户音频、编码音频、VAD 与音频打断
- 原始及编码视频、订阅控制、RTC 数据消息、元数据和连接事件
- RTM 消息、在线状态、元数据、分布式锁、历史消息及 Stream Topic
- 有界接收队列、可观测的丢弃计数，以及由调用者持有的帧数据
- 面向高级需求的原生绑定与配置入口

## 环境要求

Go **1.27.2**、C/C++ 编译器，以及固定版本的 Agora 原生库。本项目依赖 cgo，不支持 `CGO_ENABLED=0`。

| 平台 | 原生 RTC SDK | 支持情况 |
| --- | --- | --- |
| Linux amd64 | 4.6.31.134 | 主要构建及离线测试平台 |
| macOS amd64 / arm64 | 4.6.3.133 | 支持安装，仍需平台运行验证 |
| Linux arm64 / Windows | — | 当前发行配置不支持 |

绑定基于 [Agora Go SDK v2.6.5](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/v2.6.5)。RTC 与 RTM 使用独立的原生版本号。安装脚本固定下载版本并校验 SHA-256；具体定义见 [`scripts/sdk_versions.sh`](scripts/sdk_versions.sh)。

## 快速开始

```sh
git clone --branch dev https://github.com/zyy17/agora-server-sdk.git
cd agora-server-sdk
make download-agora-libs
make build-examples
```

创建服务、加入频道、读取音频：

```go
service, err := agorasdk.NewService(agorasdk.Config{AppID: appID})
if err != nil {
    return err
}
defer service.Close()

session, err := service.Join(ctx, agorasdk.ChannelConfig{
    Channel: "demo",
    UserID:  "server",
    Token:   token,
})
if err != nil {
    return err
}
defer session.Close()

for {
    frame, err := session.Audio().Next(ctx)
    if err != nil {
        return err
    }
    // frame.Data 属于调用者，可以交给后续处理流程。
    process(frame.UserID, frame.Data)
}
```

上面的片段放在返回 `error` 的函数中；导入路径为 `github.com/zyy17/agora-server-sdk`，包名为 `agorasdk`。`process` 代表业务自己的音频处理函数。

默认音频格式为有符号 PCM16、小端、16 kHz、单声道。实时音频使用 `session.SendPCM(data)`；读取 PCM 文件使用 `session.PlayPCM(ctx, reader)`，由库控制发送节奏。首次发送自动发布轨道。

Token 应由业务自己的服务端签发。本库入会不需要 App Certificate，不要将它写入客户端或示例代码。

### 在其他 Go 项目中使用

```sh
go get github.com/zyy17/agora-server-sdk@dev
```

通过本仓库安装固定版本的原生 SDK，然后在自己的项目目录执行：

```sh
/path/to/agora-server-sdk/scripts/with_agora_env.sh go build ./...
/path/to/agora-server-sdk/scripts/with_agora_env.sh ./your-binary
```

部署时需要同时携带原生动态库并配置运行时库搜索路径。环境脚本负责配置构建和运行路径；编译成功不代表二进制文件可以脱离动态库独立运行。

## 使用入口

- 音视频、RTC 事件及数据消息：[RTC 使用指南](docs/rtc.zh-CN.md)
- RTM 消息及状态管理：[messaging](messaging/README.zh-CN.md)
- 高级原生接口：`agora/rtc`、`agora/rtm`
- 可运行场景：[examples](examples/README.zh-CN.md)

外部音频处理依赖对应的 Agora 扩展库。可选的本地解码需要 `avcodec` 构建标签及 FFmpeg 开发库。实际功能还受平台、SDK 分发包和 Agora 项目配置影响。

## 开发检查

```sh
make fmt          # 格式化 Go 文件
make check        # 格式、工具脚本、lint、vet、单测、race、示例构建、原生冒烟检查
```

离线检查不加入频道，也不需要凭据。真实音视频互通及已开通的 RTM 功能需使用自己的 Agora 项目联调。详见[开发与部署](docs/development.md)。

## 许可证

Go 封装采用 [MIT 许可证](License)。Agora 原生 SDK 与可选第三方编解码器遵循各自的许可证及服务条款。
