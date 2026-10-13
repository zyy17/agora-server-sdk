# 与官方 Agora Go Server SDK 的差异

[English](upstream-comparison.md) · [首页](../README.zh-CN.md) · [RTC 指南](rtc.zh-CN.md) · [RTM 指南](../messaging/README.zh-CN.md)

本项目在修改后的 Agora Go 绑定之上，提供管理资源生命周期、支持 context 的 Go API。主要差异是面向应用的 API 契约、生命周期管理、参数校验、回调交付方式、绑定修复和开发工具。音视频传输、编解码、网络行为及 RTM 服务仍由 Agora 提供。本项目是独立模块，不能只替换 import 就兼容官方 SDK。

## 对比基线

本文的“官方原版”固定指 **AgoraIO-Extensions/Agora-Golang-Server-SDK v2.6.5**，提交 [`d51ff1b5baf85dd53155b31543bceba8ec304dc7`](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/d51ff1b5baf85dd53155b31543bceba8ec304dc7)，不指持续变化的最新分支。本地源码链接对应本文所在的代码版本。

原生分发包与该[上游安装脚本][up-install]的选择一致：Linux amd64 为 **4.6.31.134**，macOS amd64/arm64 为 **4.6.3.133-20260618**。URL 和 SHA-256 见 [`scripts/sdk_versions.sh`](../scripts/sdk_versions.sh)。Go 绑定、RTC 分发包和 RTM 运行库各有独立版本；可用 `messaging.Version()` 查看实际链接的 RTM 版本。

## 差异概览

| 方面 | 官方 v2.6.5 | 本项目 |
| --- | --- | --- |
| RTC 入口 | 进程级 `Initialize`/`Release`、`RtcConnection`、observer | `Service`、`Session`、类型化接收队列 `Stream[T]` |
| 完成与错误 | 原生返回码和结果回调 | 有界入会/请求等待、context、Go 哨兵错误和类型化错误 |
| 接收方式 | 应用提供 observer 回调 | 复制后的数据进入有界队列，显式统计丢弃 |
| 媒体 | PCM/编码音频、原始/编码视频、VAD、打断、APM、可选解码 | 带校验的媒体方法、独立收发配置、PCM reader 调速助手 |
| RTM | 客户端、Presence、存储、锁、历史、Stream Channel 绑定 | 客户端/Stream 生命周期、请求关联、Go 结果与事件队列 |
| 原生访问 | 绑定层 API | 修改后的底层绑定，以及受约束的配置/访问入口 |
| 集成 | 上游安装脚本、示例与 README | 校验和固定安装、环境助手、离线检查、GoDoc 和双语指南 |

多连接、逐用户音频、VAD、打断、编码媒体观察及 RTM 功能都已存在于上游。接收复制也并非全是新增：[上游 PCM/视频转换][up-convert]已有 `C.GoBytes`。本项目主要补充一致的高层数据所有权与队列契约。

## 生命周期与具体调用方式

官方[连接实现][up-connection]已经负责创建、释放配置对应的轨道和 sender。应用仍需组织初始化、observer、异步入会结果、断开及最终释放。官方[中文 README][up-zh]要求每个进程仅一个 service，并禁止在回调中调用 SDK 或执行耗时工作。

同样是接收音频，调用流程可这样对应。下表是 API 流程，不是完整程序：

| 步骤 | 官方绑定 | 本项目高层封装 |
| --- | --- | --- |
| 初始化 | `NewAgoraServiceConfig`、`Initialize` | `NewService(Config)` |
| 配置并入会 | `NewRtcConnection`、`RegisterObserver`、`RegisterAudioFrameObserver`、`Connect` | `service.Join(ctx, ChannelConfig)` 配置自有 observer 并等待连接结果 |
| 消费 PCM | `OnPlaybackAudioFrameBeforeMixing` 回调 | 在应用代码中调用 `session.Audio().Next(ctx)` |
| 结束 | `Disconnect`、连接 `Release`、进程 `Release` | `session.Close()`、`service.Close()` |

配置细节见[完整示例](../README.zh-CN.md#快速开始)与官方实现；`Connect` 返回零本身不等于收到 `OnConnected`。

[`Service`](../service.go) 通过 `ErrServiceActive` 限制一个活跃的高层 RTC 实例，管理其会话与音频处理器，先关闭子资源，再释放运行库。[`Session`](../session.go) 管理连接、轨道、observer 与队列。Close 幂等；`Config.IdleMode` 默认为 false，开启后立即停用回调并清理 Go 资源，延迟销毁原生连接句柄；`Service.Close` 在释放运行库前排空这些句柄。这些约束适用于高层 API，不能在活跃 Service 旁混用原生 `Initialize`/`Release`。

## Context、错误与接收队列

- `Join` 等待成功、失败、调用方取消或 `JoinTimeout`（默认 30 秒），失败时关闭未完成会话。成功返回后取消原 context 不会关闭 Session。
- `Next(ctx)` 和 `PlayPCM(ctx, reader)` 支持取消等待，但不能中断阻塞 reader 或已经执行的同步原生调用。Session 串行化原生访问；媒体生产者的发送顺序仍由应用管理。
- 用 `errors.Is` 判断哨兵/context 错误，用 `errors.As` 获取 `*agorasdk.Error` 或 `*messaging.Error`，保留原生错误码。通过本地校验或提交成功，不代表对端收到或播放完成。

RTC 的 `Audio`、`Video`、`EncodedAudio`、`EncodedVideo`、`Messages`、`Events` 六个队列默认各 **64 项**。满时丢弃**最旧**项，`Stats()` 提供计数。context 有效时，关闭后读完缓存再返回 `io.EOF`。多个读者竞争消费，不会各收到一份。

RTM 默认缓存 **128 个事件**，满时丢弃**最新**事件，通过 `DroppedEvents()` 统计；读完关闭队列后，`Next` 返回 `messaging.ErrClosed`。请求完成走独立关联路径，不经过可丢弃的事件队列。两者排队后的高层接收载荷均为副本，但都不是无损日志，也不按字节限制内存。可选 RTC 同步音频/编码视频 hook 是另一套回调线程契约：须及时返回，不得重入生命周期操作或保留缓冲区；仅在读写观察点且返回接受时，音频修改才会写回原生层。应用需持续消费 token/网络事件并处理过载。实现见 [`stream.go`](../stream.go) 和 [`messaging/client.go`](../messaging/client.go)。

## 媒体与发送节奏

封装把既有原生能力组合成独立 `Audio.Send`/`Audio.Receive` 格式、独立发布/接收选择、逐用户 PCM/VAD、编码接收、手动订阅和独立音频处理器。例如“16 kHz 单声道接收、16 kHz 双声道发送”是对既有媒体路径的配置，不是新增原生编解码器或重采样器。

`SendPCM`/`SendPCMAt` 接受完整、整数毫秒的 PCM16 数据；44.1 kHz 需为 10 ms 的倍数。它们不调速、不补齐、不重采样。上游已有可变时长 PCM；本绑定修复其按整数 kHz 计算造成的 44.1 kHz 样本问题，并保留传入的呈现时间值。`PlayPCM` 在 `io.Reader` 上增加可取消的 10 ms 软件调度，末尾完整但不足一帧的数据补静音。`WaitPCM` 只是已发送字节/墙钟估算，不是原生队列排空或对端播放确认。打断前先停止生产者，否则后续发送仍会重新填充。

编码音视频方法接收 codec packet/frame，不接收任意容器文件片段，也不自动调速。高层编码传输提供固定版本的八个 codec ID，并保留帧类型、轨道 ID 和数值 UID；接受枚举不代表原生 codec 可用。可选 SEI 插入和解码仍限 H.264/H.265；上游将 H.265 列为仅解码。`SendCPUVideo` 保留 CPU 步长、裁剪、色彩和 alpha 选项，并拒绝不支持的硬件句柄。实际能力仍受原生平台与 codec 限制，详见[媒体契约](media-api.zh-CN.md)。

接收音频 VAD v2、独立原生单/双声道 VAD、外部 APM 是不同路径。VAD 不提供语音识别。外部 APM 需要对应扩展；固定 Linux 包缺少 AINS，默认处理器可能初始化失败。独立资源管理的 `AudioVAD`、`StereoVAD`、`TelemetryVAD` 提供不同的检测器/遥测契约。可选同步 `VideoDecoder`/`OpusDecoder` 返回复制后的帧；`Session.Transcode*` 仍异步解码并发布。工作线程接受输入不等于完成，关闭也不保证排空。完整契约见 [RTC 指南](rtc.zh-CN.md)。

## 身份与 Token

[上游 service 配置][up-service]提供 `UseStringUid`，默认 false。本项目默认使用**字符串账号**，也可显式设置进程内 Service 统一的 `NumericIdentity`。数值 UID 用规范十进制字符串表示 uint32，`"0"` 请求原生分配。纯数字字符串账号不会自动变成数值 UID；token 和对端必须匹配选定模式。

入会接收外部 RTC token，不需要 App Certificate。应用处理过期事件并调用 `RenewToken`，封装不会获取或自动刷新 token。可选 [`token.RTC`](../token/token.go) 使用官方 AccessToken2 builder，仅在可信服务端生成**字符串账号的 RTC token**。数值 UID 或 RTC+RTM 组合权限继续使用对应官方生成流程；该助手不能替代。RTM 身份/token 独立配置。不同版本对字符串账号长度的说明也有差异，迁移前见[身份契约](rtc.zh-CN.md#配置与校验)。

## RTM 功能与默认行为

官方 [RTM 绑定][up-rtm]已包含消息、Presence、元数据存储、锁、历史和 Stream Topic。[`messaging`](../messaging/README.zh-CN.md) 增加独立客户端、Stream 生命周期、类型化 Go 结果和 request ID 关联，也处理结果回调早于请求函数返回的情况。异步方法等待结果，受调用方截止时间与默认 **15 秒**请求超时中较早者约束。

取消仅停止等待，不撤销已提交的消息、元数据写入或锁获取。重试前查询状态或使用应用幂等机制。`Close` 结束等待，先释放 Stream 再释放客户端，但不等待优雅 `Logout`/`Leave`。

需检查封装默认值：`SubscribeOptions{}` 只启用消息，Presence/元数据/锁事件默认关闭；`DisableMessages` 可显式选择仅 Presence 订阅。`AcquireLock` 禁用原生自动重试；`AcquireLockWithOptions` 可开启重试，通过可选 `Config.ResultEvents` 观察迟到结果。取消/超时不会停止原生重试。元数据 revision 的零值原样传递，不转为无条件 `-1`。未提供 `Config.Native` 时 RTM 默认使用字符串身份；`IdentityDefault` 保留传入原生配置的身份选择，显式 `IdentityString`/`IdentityNumeric` 则覆盖它。类型化连接/日志/代理/加密/私有配置和复制后的结果事件覆盖固定 C 边界可用接口。历史需要项目开通且发布时启用存储。RTM Stream Channel 入会/续期使用 RTC token。`messaging.Stream` 是有生命周期的 RTM Stream Channel，RTC `Stream[T]` 是本地接收队列。RTC 数据流和 RTM 是不同传输，均不承诺接收应用已处理消息。

## 高级访问与覆盖范围

`Config.ConfigureNative`、`ChannelConfig.ConfigureNative` 提供初始化/分配前配置；`Service.SetParameters`、`Session.WithNative` 和 RTM `WithNative` 提供受控高级访问。这些入口不增加原生功能，也不保证所有选项都兼容高层默认行为。

禁止保留借用指针、释放自有对象、替换封装管理的 observer/event handler，或在 `WithNative` 回调中重入同一高层对象。直接使用 `agora/rtc`、`agora/rtm` 仍需处理原生返回码，自行管理生命周期与并发。高层 API 覆盖[有源码依据的覆盖清单](api-coverage.zh-CN.md)中列出的应用功能族；底层工具和不可用的原生专属路径仍有明确分类，不是即插即用的兼容层。

## 本项目的绑定修复

以下仅相对于固定上游源码，不意味着所有上游版本都有同样问题：

- **回调 ABI：** 远端统计回调按原生不透明 track handle 和正确结构处理，再映射为公开用户 ID；原始视频回调返回类型与头文件一致。类型化 C observer 注册使签名漂移成为编译错误。见 [`observer_bindings.c`](../agora/rtc/observer_bindings.c)、[`remote_track_registry.go`](../agora/rtc/remote_track_registry.go)、[合成 C 回调测试](../agora/rtc/callback_abi_test.go)，以及对应[上游统计声明][up-callbacks]、[视频声明][up-video-callbacks]。
- **回调与资源生命周期：** RTC observer/生命周期清理前等待在途回调结束，并同步管理远端 track 身份映射；RTM 使用 handler 注册表，避免原生代码持有包含 Go 指针的 Go 对象地址。见 [`callback_gate.go`](../agora/rtc/callback_gate.go)、[`callback_registry.go`](../agora/rtm/callback_registry.go)。
- **PCM 与 VAD：** 校验完整样本并避免 44.1 kHz 截断；VAD v2 归一化前复制配置，防止不同用户共享配置时被改写；独立 VAD 严格接收 10 ms、16 kHz PCM16。独立检测器的处理/释放串行化，C++ 异常转为 `-2002` 并停用相关检测器。见 [PCM 测试](../agora/rtc/pcm_frame_test.go)、[VAD v2 测试](../agora/rtc/audio_vad2_test.go)、[独立 VAD 测试](../agora/rtc/audio_vad_test.go)、[异常桥接](../agora/rtc/audio_vad_bridge.cc)。

这些是有回归测试的定向防护，不代表原生代码无缺陷、完成全面安全审计或性能优于官方。

## 依赖、验证、文档与许可

两者的 RTC/RTM 集成都依赖 cgo 和原生库；本项目的 `mediaio`、`token` 助手为纯 Go。本项目固定 **Go 1.27.2**；[上游 README][up-en] 要求 **Go 1.21+**，虽然其[模块文件][up-module]写的是 `go 1.20`。安装器先校验 SHA-256 和包内容再替换安装，`with_agora_env.sh` 设置对应构建/运行路径。RTC/RTM 应用部署仍须带上共享库；Go 编译成功不代表 RTC/RTM 二进制可独立运行。

Linux amd64 是主要构建/CI 平台。安装器也支持 macOS amd64/arm64，但上游限制其仅用于开发测试。当前分发配置不支持 Linux arm64/Windows。能安装或 Linux 测试通过，不代表 macOS 可用于生产。

上游已通过 `avcodec` 将解码功能设为可选。本项目同样让默认构建不依赖**系统 FFmpeg 开发库**，并在 CI 检查；可选解码测试单独执行。`send_media` 示例则在运行时使用 FFmpeg 可执行程序。这不表示 Agora 原生包内没有 FFmpeg 相关组件。

[`make check`](../Makefile) 覆盖格式、工具脚本用例、固定离线 API 覆盖检查、第一方 Staticcheck、全部包 vet/test/race、示例构建、无凭据原生生命周期及严格 cgo 检查下的合成注册 C 回调。Go race detector 不检测原生内部。离线通过不能证明真实迁移成功、远端媒体质量、鉴权正确或项目功能权限，部署前需完成[真实集成清单](development.zh-CN.md#真实联调)。

标准包/符号 GoDoc 说明 API 契约；双语 [RTC](rtc.zh-CN.md)、[RTM](../messaging/README.zh-CN.md)、[示例](../examples/README.zh-CN.md)、[开发](development.zh-CN.md)指南说明工作流程。官方[英文][up-en]/[中文][up-zh] README 仍是上游行为的重要来源。[官方中文 Go API](https://doc.shengwang.cn/api-ref/rtc-server-sdk/go/go-api/overview)仅作补充，不能替代固定版本源码与头文件。

仓库中的 Go 封装按 [MIT License](../License) 分发；这不会改变 Agora 专有原生 SDK、引入的第三方材料或可选 codec 的许可证，仍须保留声明并遵守各自条款。本项目不声称由 Agora 官方维护或背书。

## 迁移前检查

1. 明确改写 import 与回调逻辑，将所有权和关闭流程映射到 `Service`、`Session` 或独立 RTM Client。
2. 显式设置身份模式、角色、媒体格式、订阅和 token 权限，不假定默认值与上游一致。
3. 处理队列丢弃、异步错误、取消不回滚，以及需要时的应用层确认。
4. 针对固定分发包确认所需原生选项、codec/扩展、RTM 权限及平台，并编译、执行双端集成测试。

需要生命周期管理、有界队列和 context 等待时，可选择高层 API。需要不同回调模型或原生接口范围时，可保留官方/底层绑定，同时承担相应资源与并发管理。

[up-connection]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/rtc_connection.go
[up-rtm]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtm
[up-en]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/README.md
[up-zh]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/README.zh.md
[up-callbacks]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/rtc_callbacks_cgo.h
[up-video-callbacks]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/video_observer_cgo.h
[up-install]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/scripts/install_agora_sdk.sh
[up-convert]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/type_convert.go
[up-service]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/agora_service.go
[up-module]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go.mod
