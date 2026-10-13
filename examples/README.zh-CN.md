# 示例

[English](README.md) · [首页](../README.zh-CN.md)

只编译，不连接 Agora：

```sh
make download-agora-libs
make build-examples
```

## RTC 配置

设置 `AGORA_APP_ID`、唯一的 `AGORA_USER_ID`；项目开启鉴权时还需设置 `AGORA_TOKEN`。`AGORA_CHANNEL` 默认为 `go-sdk-example`。每个参与者使用不同的身份及匹配的 Token。Token 从自己的服务端获取，示例不需要 App Certificate。

在仓库根目录运行。环境脚本自动配置原生库路径，Ctrl+C 会关闭 Service 及其会话。

```sh
./scripts/with_agora_env.sh go run ./examples/recv_pcm -output recordings
./scripts/with_agora_env.sh go run ./examples/send_pcm -input examples/testdata/send_audio_16k_1ch.pcm
```

在两个终端分别启动发送端、接收端，并使用不同的 `AGORA_USER_ID`。默认格式为有符号 PCM16、小端、16 kHz、单声道；两个示例均支持 `-rate` 和 `-channels`。接收文件以用户 ID 的十六进制编码命名，已存在时追加写入；需要全新录音时选择新目录。

## 场景

| 目录 | 用途 |
| --- | --- |
| [send_pcm](send_pcm) | 按节奏发送 PCM 文件并估算完成时间 |
| [recv_pcm](recv_pcm) | 保存逐用户 PCM，无需音频设备 |
| [loopback](loopback) | 将接收到的 PCM 返回频道，注意避免反馈循环 |
| [ai_audio](ai_audio) | VAD、打断、音量及 Token/连接事件 |
| [messages](messages) | 收发 RTC 数据流消息 |
| [multi_sessions](multi_sessions) | 一个 Service 管理多个只接收会话 |
| [send_media](send_media) | 将本地 MP4 等文件解码为按节奏发送的 PCM 和原始视频，使用共同时间戳基准 |
| [media](media) | 原始视频、编码音视频与可选解码发布 |
| [process_audio](process_audio) | 不入会的外部音频处理 |
| [messaging](messaging/README.zh-CN.md) | RTM 消息、状态、元数据、锁、历史与 Topic |

```sh
./scripts/with_agora_env.sh go run ./examples/ai_audio -reply response.pcm
./scripts/with_agora_env.sh go run ./examples/messages -send 'hello from Go'
./scripts/with_agora_env.sh go run ./examples/multi_sessions -count 2
```

`multi_sessions` 使用 `${AGORA_USER_ID}-0`、`${AGORA_USER_ID}-1` 等身份，并分别读取 `AGORA_TOKEN_0`、`AGORA_TOKEN_1`。不会把同一个 `AGORA_TOKEN` 复用于不同身份。

`ai_audio` 演示事件处理，需要将 Token 续期提示替换为自己的 Token 服务调用。`process_audio` 依赖原生 APM 扩展；当前 Linux 包缺少 AINS，默认处理器可能初始化失败，不能将它视为跨平台可直接使用的降噪方案。

## 媒体格式

```sh
./scripts/with_agora_env.sh go run ./examples/media -mode send-yuv -input input.yuv -width 640 -height 360 -fps 15
./scripts/with_agora_env.sh go run ./examples/media -mode recv-yuv -user remote-user -output received.yuv
./scripts/with_agora_env.sh go run ./examples/media -mode send-aac -input input.aac
./scripts/with_agora_env.sh go run ./examples/media -mode send-h264 -input frames.packets -width 640 -height 360 -fps 15
```

- `send-yuv` / `recv-yuv`：原始 I420（YUV420p），不含容器头
- `send-aac`：AAC-LC ADTS 帧
- `send-opus`：长度前缀的 20 ms Opus 包，通过 `-rate`、`-channels` 指定实际格式
- `send-h264` / `recv-h264`：长度前缀的 Annex-B 编码访问单元
- `recv-encoded-audio`：长度前缀编码包
- `transcode-h264` / `transcode-opus`：需要 `-tags avcodec`，异步解码并发布；Opus 输出为 48 kHz 单声道。关闭不保证排空处理队列，见 [RTC 转码限制](../docs/rtc.zh-CN.md#外部处理与可选解码)和[开发说明](../docs/development.zh-CN.md)

所有接收模式要求指定 `-user`，避免混合多个独立码流，并会覆盖 `-output` 文件。包格式为 4 字节大端无符号长度，随后是对应包内容，单包上限 16 MiB。可通过 `mediaio.WritePacket` 准备输入。该格式保留包边界，但不存储完整的编解码、时间戳及容器信息，不是 MP4、Ogg 或通用录制格式。Opus 示例假设每包恰好为 20 ms。H.264 载荷需要使用 Annex-B 起始码，并预先正确划分访问单元，不能按任意读取块分帧。

## 运行与验证

运行频道类 RTC 示例会加入真实频道，并可能发送或录制媒体；`process_audio` 仅运行外部处理器，不入会。请使用专用测试资源，只在取得参与者授权后录制。构建及离线检查通过不代表已完成真实互通验证。RTM 示例可能修改远端状态，运行前先阅读[对应操作说明](messaging/README.zh-CN.md)。

## 发送媒体文件

安装 `ffmpeg` 命令行工具后执行：

```sh
./scripts/with_agora_env.sh go run ./examples/send_media -input clip.mp4 -width 640 -height 360 -fps 15
```

输入文件必须同时包含音频和视频。示例使用独立的 FFmpeg 进程解码，将音频统一为 16 kHz 单声道 PCM、视频统一为固定帧率 I420，并使用共同的时间戳基准。它不保留任意源容器的原始时间信息，也不保证逐帧精确的音画同步。运行时需要 FFmpeg 可执行文件，不需要 FFmpeg 开发头文件或 `avcodec` 构建标签。
