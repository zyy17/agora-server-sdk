# RTC 使用指南

[English](rtc.md) · [首页](../README.zh-CN.md)

## 生命周期与默认值

每个进程创建一个 `Service`，不同频道或身份分别调用 `Service.Join`。Token 属于单个会话。`Service.Close` 先关闭所有会话及音频处理器，再释放原生运行时。Service 和 Session 均可重复关闭。

`Join(ctx, config)` 等待连接成功或失败，默认超时为 30 秒。入会成功后取消该 context 不会关闭会话，需要显式调用 `session.Close()`。读取和其他操作可以使用各自的 context。

默认使用主播模式、16 kHz 单声道 PCM16、AI-server 音频场景、自动音频订阅，以及混音前的逐用户音频观察点。视频接收需要显式开启。

## 发送与接收音频

```go
// PCM 文件或能及时返回的 reader：默认 PCM16、16 kHz、单声道。
if err := session.PlayPCM(ctx, reader); err != nil {
    return err
}

// 已由上游控制节奏的实时音频：一个或多个完整的 10 ms 帧。
if err := session.SendPCM(pcm); err != nil {
    return err
}

frame, err := session.Audio().Next(ctx)
if err != nil {
    return err
}
// 使用 frame.UserID、frame.Data、frame.SampleRate 和 frame.Channels。
```

- `PlayPCM` 使用有界缓冲，按 10 ms 节奏发送。末尾不足一帧时补静音；不足一个完整 PCM 采样则报错。context 无法中断阻塞中的 `io.Reader`。
- `SendPCM` 不控制发送节奏，也不重采样。默认格式下每个 10 ms 帧为 320 字节。其他格式通过 `ChannelConfig.Audio` 配置。
- `SendPCMAt` 可指定呈现时间戳，`PlayPCMAt` 可从指定的初始呈现时间戳开始按节奏发送；关联音视频应使用一致的时间基准。
- `InterruptAudio` 请求原生音频打断。`WaitPCM` 根据原生时长时钟及余量估算发送完成，不代表远端已经播放，也不保证原生队列已经排空。
- `AudioFrame.Data` 与 `Speech` 已从原生内存复制，调用者可以保留。

### VAD 与观察点

设置 `AudioConfig{VAD: true}`，对收到的 16 kHz 单声道音频启用原生 VAD v2；可通过 `VADConfig` 调整参数。结果见 `AudioFrame.VAD` 与 `Speech`。VAD 不负责语音转文字。

`Receive.AudioPositions` 可以按位组合 `AudioPlayback`、`AudioRecord`、`AudioMixed`、`AudioBeforeMixing` 和 `AudioEarMonitoring`。零值选择混音前观察点，可得到远端用户 ID；其他观察点不一定对应单个用户。是否产生回调取决于原生 SDK 语义及当前媒体路径。

### 编码音频

设置 `AudioConfig{Mode: agorasdk.EncodedAudio}`，通过 `SendEncodedAudio` 发送 `EncodedAudioFrame`。传入完整的编解码数据包及其实际格式、时间信息。接口支持 Opus、AAC，不负责解析媒体容器。开启 `Receive.EncodedAudio` 后，通过 `session.EncodedAudio().Next(ctx)` 接收编码包。

## 视频

创建 Service 时开启 `Config.Video`。发送格式与接收选项分别配置：

```go
cfg := agorasdk.ChannelConfig{
    Channel: channel, UserID: userID, Token: token,
    Video: agorasdk.VideoConfig{Mode: agorasdk.RawVideo},
    Receive: agorasdk.ReceiveConfig{Video: true},
}
```

- `SendVideo(VideoFrame)` 接受紧密排列的 I420、NV12、RGBA 或 BGRA CPU 帧，需要明确宽高；I420/NV12 宽高必须为偶数。
- 接收的原始帧提供 Y/U/V 平面及各自的 stride，不能假设每一行没有填充。
- 发送 H.264/H.265 时选择 `EncodedVideo`，调用 `SendEncodedVideo`，提供完整帧边界、宽高、时间戳及关键帧信息。不能把整个 MP4 文件当成一帧。
- 开启 `Receive.EncodedVideo` 后可以读取编码帧；`RequestKeyFrame`、`SetVideoEncoder`、`SetSimulcast` 提供常用控制。
- 高层发送接口的 SEI 载荷最多为 1,024 字节。具体编解码能力取决于原生分发包。

## 订阅

默认自动订阅。设置 `Receive.ManualSubscriptions` 后，可以通过 `SubscribeAudio`、`UnsubscribeAudio`、`SubscribeVideo`、`UnsubscribeVideo` 控制单个用户。视频订阅使用 `VideoSubscription{LowQuality, EncodedOnly}` 配置。

大频道模式需要同时设置 `Mode: agorasdk.LargeScale` 和手动订阅。必须按用户 ID 订阅，不支持 subscribe-all 方法。观众会话可以接收，但不能发布媒体。

## 事件、Token 与消息

在独立的业务循环中读取 `session.Events().Next(ctx)`。事件涵盖连接、用户进出、发布订阅、错误、音量、统计、元数据和 Token 过期。通过 `SetVolumeIndication` 开启音量事件。

收到 `TokenExpiring` 或 `TokenExpired` 后，从自己的服务端获取新 Token，再调用 `RenewToken`。Session 不签发 Token，也不读取业务密钥。独立的纯 Go `token.RTC` 辅助函数可在可信服务端生成 RTC AccessToken2，App Certificate 必须留在服务端。同时监控连接状态与事件队列丢弃计数。

`SendMessage` 发送 1–1,024 字节的 RTC 数据流消息，通过 `Messages().Next(ctx)` 接收。该数据流配置为无序、不可靠传输。需要应用层消息服务时，使用独立的 [RTM API](../messaging/README.md)。

`SendAudioMetadata` 发送 1–1,024 字节的音频元数据；接收结果通过 `AudioMetadata` 事件交付。

## 队列语义

每个 RTC 接收流默认容纳 64 项，可通过 `ChannelConfig.QueueCapacity` 调整。队列满时丢弃最旧的一项，避免阻塞原生回调。通过 `stream.Stats()` 查看容量、缓冲数量、累计丢弃数与关闭状态。

`Next(ctx)` 等待一个元素；关闭后先排空缓冲，再返回 `io.EOF`。多个读者竞争消费，并不会各自收到完整副本。多个消费者需要同一份数据时，应在业务层显式分发。RTM 使用不同的[队列规则](../messaging/README.md)。

## 外部音频处理与解码

设置 `Config.AudioProcessing: true`，通过 `Service.NewAudioProcessor` 创建由 Service 管理的处理器。调用 `processor.SendPCM` 输入，通过 `processor.Output().Next(ctx)` 读取结果。按需配置 `ProcessorConfig.APM` 与 VAD。

原生分发包必须包含所需的 APM 扩展。当前固定的 Linux 分发包不包含 AINS 扩展，不能假设 AI 降噪可用；缺少扩展时可能返回原生初始化错误。部署前应在目标分发包上验证具体处理器。

`TranscodeVideo`（H.264/H.265，8 位 4:2:0 Annex-B）与 `TranscodeOpus` 是可选的本地解码辅助接口。默认构建返回 `ErrUnsupported`；需要安装 FFmpeg 开发依赖并启用 `-tags avcodec`。这不会改变原生 RTC SDK 的编解码能力。

## 高级配置

`Config.ConfigureNative` 在初始化前调整进程配置；`ChannelConfig.ConfigureNative` 在分配资源前调整连接和发布配置。不要保留这些回调收到的配置指针。

`session.WithNative(func(*rtc.RtcConnection) error)` 将高级原生操作与会话操作串行化。不要在回调中递归调用 Session 方法、关闭或释放借用的原生对象、保留它供后续使用，或替换 Session 管理的观察者。

`agora/rtc` 提供低层绑定。高层 `Service` 存活期间，不要混用底层全局初始化与释放。原生接口可能需要额外的调用顺序及资源管理，不自动获得高层封装的全部保障。

## 错误处理

使用 `errors.Is` 判断 `ErrClosed`、`ErrUnsupported` 等哨兵错误；使用 `errors.As` 读取 `*agorasdk.Error` 的操作名及原生错误码。发送成功代表本地 SDK 接受数据，不代表远端收到或已经播放。
