# 媒体 API 约定

[English](media-api.md) · [覆盖说明](upstream-comparison.zh-CN.md)

原生基线为官方 Go SDK v2.6.5，提交 `d51ff1b5baf85dd53155b31543bceba8ec304dc7`。这些接口补齐可用媒体选项，并明确校验与资源归属。枚举值可传递，不代表原生包、扩展、GPU 路径或对端必然支持。

## 发送与接收

- `SendPCM` / `SendPCMAt`：使用配置的 PCM16 格式，时长须为整数毫秒，可传有符号 presentation 值。
- `SendPCMFrame`：显式指定采样数、PCM16 位宽、uint32 范围的采集时间戳及有符号 presentation 值。采样率/声道须匹配 `AudioConfig.Send`，不发送观察回调的统计字段。完整的亚毫秒采样可通过结构校验，实际支持由原生编码器决定。
- `SendEncodedAudio`：覆盖全部具名原生音频编码，支持 `SendEvenIfEmpty`；`OpusMultiChannel` 支持 3–8 声道，其他编码为单/双声道。连接层将零采集时间戳改为 -1。不解析压缩内容。
- `SendVideo`：紧凑 CPU 像素；`SendCPUVideo` 增加像素 stride、裁剪、色彩空间 ID、alpha 位置/提取及 raw/array 类型。两者共用原始发送器的布局校验。
- `SendEncodedVideo`：透传 VP8、H264、H265、GenericVideo、GenericH264、AV1、VP9、JPEG 元数据。`FrameType` 保留 blank/key/delta/B/droppable/unknown，增加 `TrackID` 与数值 `UID`。`FrameType` 默认是 `VideoFrameAuto`：`KeyFrame=true` 选择 key，否则选择 delta；真正的 blank 帧须显式用 `VideoFrameBlank`。`KeyFrame=false` 不覆盖显式类型，true 与所有显式非 key 类型（包括 blank）冲突。高层帧类型值与 raw 枚举不同；接收时逐项映射固定原生类型，转发保持原分类。SEI 插入仅支持 H264/H265。
- `ReceivedVideoFrame` 保留 Type、AVSyncType、AlphaMode、纹理 ID、变换矩阵及复制的平面。纹理 ID/矩阵仅为描述信息，不代表拥有 GPU 资源；不暴露借用的 shared context。编码视频接收保留原始帧类型、track ID 和 UID。

发送是同步且不节流的，只报告本地原生接收结果；返回前勿修改输入切片。接收切片由 Go 拥有，回调及 Close 后仍有效。Stream 满时可能丢弃旧帧，应查看统计。每条轨道建议只有一个生产者，以保持顺序。

### 带行填充的 CPU 像素

```go
frame := agora.CPUVideoFrame{
    Data: make([]byte, 644*480*4), Stride: 644, Height: 480,
    Format: agora.RGBA, CropRight: 4, FillAlphaBuffer: true,
}
err := session.SendCPUVideo(frame) // 可见图像：640×480
```

Stride 单位是像素，不是字节。Data 包含全部填充行，裁剪不会减少所需缓冲区。I420/NV12/NV21 每像素 1.5 字节，I422 为 2，I010 为 3，RGBA/BGRA 为 4；平面色度使用一半像素 stride。4:2:0 的尺寸和裁剪值须为偶数，I422 的水平值须为偶数。显式 alpha 需 stride×height 字节并包含填充；自动提取仅用于 RGBA/BGRA，且不能同时提供 alpha 切片。

固定版本的 Go 发送器不传递纹理/EGL/CVPixelBuffer/D3D 句柄，这些路径会明确拒绝，避免静默丢失句柄。C 头文件将 metadata 定义为纹理相关字段；CPU metadata 被传入 C 不等于保证对端收到。I010 仍属于原生预览功能；平台二进制和原生编解码能力限制仍适用。

## 轨道控制

`SetAudioEnabled` / `SetVideoEnabled` 开关轨道，不改变发布状态。音频开关、发布音量、发送延时和缓冲容量会在场景切换及轨道重建后保留。已配置音频的观众会话先保存设置，升为主播时应用；原生校验可能在升为主播时失败。`SetVideoEnabled` 要求已有视频轨道，`ClearAudioBuffer` 要求已有音频轨道。所有调用与 Session 生命周期共用操作锁。

Delay 按整数毫秒截断；缓冲容量以 10 ms 帧计。开关、延时和容量的原生 setter 没有返回值，成功仅表示已保存设置或调用原生接口，不代表对端确认。清空只作用于本地原生队列，建议先停止生产者，无法撤回已送达对端的音频。`InterruptAudio` 保留 AI-server 场景取消发布再重新发布的不同语义。

### 外部音频引擎选项

`rtc.SendExternalAudioParameters` 的速度和静音数据设置作用于整个运行时。首次成功的启用请求确定各自的值，直到 Service 成功释放；相同设置可重复调用，冲突设置会失败。关闭连接不会重置设置，后续初始化失败也不会撤销已生效的值。`SendMs` 仍是单轨道预算。原生接口接受设置不代表达到目标发送速度。

## 处理器与语音活动

```go
processor, err := service.NewAudioProcessor(agora.ProcessorConfig{
    SampleRate: 16000, Channels: 1,
    Input: agora.PCMFormat{SampleRate: 48000, Channels: 2},
    DisableAPM: true, VAD: &rtc.AudioVadConfigV2{},
})
```

Service 须启用 `AudioProcessing`。`Input` 设置 SendPCM/SendPCMAt 的默认输入格式，零字段分别继承输出配置。`SendPCMFormat` 可逐次指定格式和有符号 presentation 值；`SendPCMFrame` 从 AudioFrame 读取对应字段。输入须为完整的 10 ms PCM16 倍数、8000–48000 Hz 且可被 100 整除、单或双声道。输出格式独立。逐次格式的零值为 16 kHz 单声道，并非处理器的配置输入。

APM 配置深复制。`DisableAPM` 要求 VAD 且不能同时显式设置 APM，可在不加载可选 APM 滤镜时仅做 VAD。完整 APM 需要安装对应扩展，初始化可能失败。`Stats` 是原生已提交/已处理时长，并非排空或播放确认。Close 可能丢弃待处理数据；Service.Close 会一起关闭其处理器。

- `AudioVAD` / `StereoVAD`：独立原生分类，不需要 Service/频道。每次恰好 10 ms、16 kHz PCM16，即单声道 320 字节、双声道 640 字节。拥有检测器，复制选中语音，串行化 Process/Close。双声道一侧失败时保留另一侧成功结果。
- `TelemetryVAD`：使用 AudioFrame 的 voice/RMS 统计执行 Go SDK v2 分段，不独立分析 PCM。每路固定格式使用一个实例，保留的输入和输出均复制。nil 配置采用上游默认，零结构体与 nil 不同。
- 语音状态包括 silent、start、speaking、end；选中语音可能含前滚，也可能为空。独立检测器的时间戳累计选中输出时长。向识别器转发完整 PCM 或选中语音之一，避免重复。

## 可选解码与转码

`NewVideoDecoder` / `NewOpusDecoder` 需要 `-tags avcodec` 及 FFmpeg 开发/运行库，否则返回 `ErrUnsupported`。解码器拥有资源，串行化 Decode/Close，返回 Go 自有数据，不建立 RTC 连接。

视频接受 H264/H265 Annex-B access unit，输出紧凑 8-bit I420；可指定偶数尺寸缩放。`ErrDecoderNeedsInput` 表示继续提供输入。一组输入产生多图时仅返回最新图；上游未提供 flush 或逐帧时间戳接口，切换编码会丢弃已缓冲图像。

Opus 输入是单/双声道原始 packet，不是 Ogg；可独立选择输出 PCM 采样率/声道。缓冲阶段输出可能为空，上游不提供重采样器 flush。Close 会丢弃待输出数据。

Session 的 `TranscodeVideo` / `TranscodeOpus` 保持异步 worker 语义：接收成功不报告后续解码/发送失败。视频转码只支持 H264/H265，尽管编码传输可接受更多枚举。定时发送可能跳帧或重复，Opus worker 固定输出 48 kHz 单声道。不要混用生产者；需要直接结果、自行发送/节流时使用同步解码器。

## 离线验证

单测覆盖布局/溢出校验、元数据、复制接收数据、解码器构建开关、VAD 归属和 Close 并发。`-tags agora_test` 检查固定 C ABI 的媒体结构往返。Native smoke 无需凭据或加入频道，可验证真实处理器 48 kHz 双声道→16 kHz 单声道输出。可选解码 fixture 在 avcodec CI 作业执行。这些检查不能证明双端实时互通或全部平台/编码组合可用。

## 同步观察决策

`AudioConfig.ReceiveParameters` 可分别设置 Playback、Record、Mixed、EarMonitoring 的 PCM 格式和采样数。未配置项继承通常的 10 ms 只读接收格式；before-mixing 始终只读。`AudioConfig.Callbacks.OnFrame` 同步接受/拒绝帧；仅 `ReadWrite` 位置返回 true 时的原地 Data 编辑影响原生 PCM 及队列快照。拒绝或只读时的编辑、元数据及切片头变更均忽略；队列缓冲区与回调缓冲区独立。

`VideoConfig.OnEncodedFrame` 同步接受/拒绝编码视频，编辑不会改变原生数据或队列快照。两类 hook 即使拒绝仍排入观察队列；panic 会被隔离并产生 `CallbackPanic` 事件、拒绝当前帧。回调须及时返回，不可阻塞 I/O、等待事件或调用需要生命周期锁的 Session/Service 方法，包括 Close/Send/WithNative/SetRole。回调可能并发执行，应用须同步自身状态。
