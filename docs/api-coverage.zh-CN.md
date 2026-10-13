# 官方 Go API 覆盖范围

[English](api-coverage.md) · [首页](../README.zh-CN.md) · [官方对比](upstream-comparison.zh-CN.md)

## 基线与含义

基线为官方 **Go v2.6.5**，固定提交 [`d51ff1b5baf85dd53155b31543bceba8ec304dc7`](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/d51ff1b5baf85dd53155b31543bceba8ec304dc7)。本项目是独立模块，包含有意的源码兼容性调整，不能直接替换 import，也不宣称“100% 功能覆盖”。

检入的[参考清单](api-reference/official-go.tsv)包含 **40 个经过 SHA 校验的官方 Go 文件中的 1,159 个声明**：

| 官方接口 | RTC | RTM |
| --- | ---: | ---: |
| 函数 | 65 | 46 |
| 方法 | 118 | 97 |
| 导出结构体字段，包括回调 | 336 | 172 |
| 类型 | 104 | 51 |
| 常量 | 130 | 37 |
| 变量 | 3 | 0 |

全部 **326 个函数/方法名**均保留。这本身不能证明参数正确、回调可达、原生封送正确、项目已开通功能，或已有高层入口。检查器另行核对签名和全部导出字段，并记录下述 **33 项精确兼容性例外**。忽略参数名和私有结构体布局；保留参数类型、顺序、数量、指针接收者、回调返回值、导出字段类型、别名及常量表达式。检查所有构建条件下的源码，相同声明只计一次。

## 覆盖分类

机器可读的[覆盖映射](api-reference/coverage.json)将每项能力关联到官方声明、当前公共 API/配置/事件、具名回归测试及限制。每个官方可调用接口都有分类；未包装的构造器、转换器和资源方法明确归为 `raw-only`。

| 分类 | 契约 |
| --- | --- |
| `high-level` | 具名的资源管理 API/配置/事件覆盖记录中明确限定的行为 |
| `managed-advanced` | 配置或借用原生访问，须遵守所有权和组合限制 |
| `raw-only` | 保留的底层绑定；自行管理生命周期、并发和原生返回码 |
| `safe-compatibility` | 对不安全或不完整上游签名的有意修改，附测试依据 |
| `native-unavailable` | 固定 Go/C 边界未提供，或依赖当前缺失的扩展/平台 |
| `planned` | 尚未集成，不计入已实现覆盖范围 |

### 当前应用入口

| 能力 | 公共入口 | 依据 / 限制 |
| --- | --- | --- |
| RTC 生命周期与身份 | `NewService`、`Service.Join`、`Session.Close`、`Config.IdentityMode` | [入会/关闭测试](../session_test.go)、[身份测试](../identity_test.go) |
| RTC 高级配置 | `ConfigureNative`、`Session.WithNative`、Service/Session 的 `SetParameters` | [配置测试](../session_config_test.go)；不得保留或释放借用句柄 |
| PCM、编码音频、调速与打断 | `AudioConfig.Send/Receive`、`SendPCMAt`、`PlayPCM`、`SendEncodedAudio`、`InterruptAudio`、`WaitPCM` | [音频测试](../audio_test.go)；`WaitPCM` 是估算，不是远端播放确认 |
| CPU/编码视频与订阅 | `SendCPUVideo`、`SendVideo`、`SendEncodedVideo`、接收流、发布/订阅方法 | [布局/元数据测试](../video_parity_test.go)、[原生 ABI](../agora/rtc/media_abi_test.go)；拒绝不支持的硬件句柄 |
| 角色、回退、空闲/代理/扩展控制 | `SetRole`、`SetAudioFallback`、`Config.IdleMode/LocalAccessPoint`、Service 扩展方法 | [控制测试](../control_test.go)；仍受原生能力限制 |
| 同步媒体决策与观察格式 | `Audio.ReceiveParameters`、`Audio.Callbacks.OnFrame`、`Video.OnEncodedFrame` | [回调隔离/写回/panic 测试](../media_callbacks_test.go)；须遵守回调线程限制 |
| 独立 VAD/解码器与轨道控制 | `NewAudioVAD`、`NewStereoVAD`、`NewTelemetryVAD`、`NewVideoDecoder`、`NewOpusDecoder`、Session 轨道 setter | [VAD](../vad_test.go)、[解码构建开关](../decoder_test.go)、[轨道测试](../audio_parity_test.go)；解码器需要 `avcodec` |
| RTC 数据与事件 | `SendMessage`、`Messages`、`Events` | [回调测试](../session_test.go)、[有界队列测试](../stream_test.go) |
| 外部音频处理 | `Service.NewAudioProcessor`、`SendPCM`、`Output` | [处理器测试](../processor_test.go)；仍受扩展可用性限制 |
| RTM 生命周期、消息与请求关联 | `messaging.New`、登录/登出/续期、发布/订阅 | [请求测试](../messaging/client_test.go)；取消等待不会回滚原生操作 |
| RTM Presence、元数据、锁、历史 | `OnlineUsers`、`UserChannels`、状态/元数据/锁方法、`History` | [结果/复制测试](../messaging/client_test.go)、[历史校验](../messaging/history_validation_test.go) |
| RTM Topic | `NewStream`、加入/退出/续期、Topic 发布/订阅 | [部分结果及所有权测试](../messaging/client_test.go)；Stream Channel 使用 RTC Token |
| RTM 选项、身份与原生访问 | `Config.Identity/Connection/Log/Proxy/Encryption/Private`、`DisableMessages`、`ExcludeUserID`、`AcquireLockWithOptions`、`WithNative` | [映射/校验测试](../messaging/config_parity_test.go)；重试可能在调用方停止等待后继续 |
| 完整 RTM 结果事件 | `Config.ResultEvents`、`EventRequestResult`、`RequestResult`、`Events`、`Next` | [完整回调/结果/复制测试](../messaging/results_test.go)；可选事件丢弃不影响请求完成 |

精确的符号、测试链接及更窄的行为契约见覆盖映射，不能仅凭一行表格推断完整功能覆盖。已集成的控制/媒体/RTM 扩展均有明确 API 和回归测试依据。当前没有按计划工作计算的能力记录；以后新增功能须保持 `planned`，直到实现与测试真实存在。精确默认值和回调限制见[媒体契约](media-api.zh-CN.md)与 [RTM 覆盖范围](../messaging/README.zh-CN.md)。

## 有意的兼容性修改

检查器仅允许以下精确差异；进一步变化会失败，不会被宽泛排除项隐藏：

- RTC：删除三个调试文件全局变量；删除固定原生统计结构体不存在的 `RemoteAudioTrackStats.MosValue`、`PublishDuration`、`TotalActiveTime`；`GoRemoteAudioStats` 使用正确的 C 结构体；`VideoFrameObserver.OnFrame` 改为无返回值以匹配 C ABI。参见[类型化 observer 注册](../agora/rtc/observer_bindings.c)、[合成 ABI 测试](../agora/rtc/callback_abi_test.go)和 [VAD 测试](../agora/rtc/audio_vad_test.go)。
- RTM：删除固定 `C_IRtmEventHandler` 不存在的旧 `OnConnectionStateChanged`，保留 `OnLinkStateEvent`。五个结果回调改为复制后的切片，保留全部原生数组元素。十九个锁/存储/Topic 方法保留错误处理与关联请求所需的即时状态或请求 ID。参见[绑定测试](../agora/rtm/bindings_test.go)及覆盖映射中的逐项精确替换。

这些都是有意的 API 变化。使用官方签名的应用需要迁移；保留名称不代表源码兼容。

## 边界与剩余缺口

- 原生媒体播放器、RTMP 推流和设备 API **不在本官方 Go 基线内**。其他 Agora SDK 提供这些功能，不代表本项目丢失了官方 Go 函数。
- GPU 纹理、共享/原生硬件缓冲区、平台图像句柄需要受支持的封送和生命周期规则。底层 Go 字段或枚举存在，不代表高层发送器支持该输入。
- 固定 RTM C 桥接未提供仅 C++ 可用的 Stream 参数及部分事件时间戳/人数统计字段。不能编造数值，也不能称其为已丢失的 Go 功能。
- 原生编解码器/扩展、平台支持、对端兼容性、项目权限、网络行为和实际媒体质量仍需单独验证。固定 Linux 包缺少 AINS；可选 FFmpeg 路径有独立构建和运行要求。
- 生命周期、回调和复制队列有意区别于直接拥有 observer。不得保留借用原生句柄、替换托管 observer、释放托管对象，或在高级访问回调中重入包装对象。

## 离线回归检查

```sh
make check-api-coverage
```

该 Go AST 工具仅使用标准库，在 `CGO_ENABLED=0` 下运行，不需要原生库、凭据或网络，同时纳入 `make check`。它核对固定签名、精确兼容性替换/删除、完整可调用接口分类、公共入口和具名测试是否存在。工具自身的变异测试覆盖回调/返回值变化、指针/参数数量保留、依据缺失、构建条件冲突、不安全字段重新引入，以及来源版本被移动。

覆盖映射键采用 `package:kind:name`；例外项的空 replacement 表示有意删除。AST 指纹不计算 C 枚举值、默认值或方法实现。检查通过**不代表执行了引用的测试，也不证明原生封送或行为**。仍需运行 `make check`、`make test-callbacks`、相关可选解码器检查，以及[在线集成清单](development.zh-CN.md#真实联调)。Go 竞态检测器不检测原生内部实现。

如需有意重新生成参考清单，使用固定官方提交的未修改检出目录：

```sh
go run ./internal/cmd/apicoverage -snapshot /path/to/official-checkout
```

生成器逐文件校验 Git blob ID，记录源码路径和哈希，拒绝不同提交或被修改的源码。通过连接器获取的源码目录可提供 `source-manifest.json`，包含相同的 repository/ref/tag 及逐文件 Git ID；生成器会校验本地字节，而清单本身必须来自对该精确提交的可信读取。日常检查只使用检入的小型清单，不依赖上游检出目录或复制的实现代码。切勿从本项目重新生成基线来掩盖回归。
