# 消息服务（RTM）

[English](README.md) | [简体中文](README.zh-CN.md)

`messaging` 封装了本仓库固定版本原生 SDK 所支持的 RTM 客户端接口，管理原生资源生命周期，
通过 `context` 等待请求完成，并以普通 Go 值和已复制的队列事件交付数据。
它独立于 RTC 会话，应用代码无需编写回调函数。

```go
client, err := messaging.New(messaging.Config{
    AppID: os.Getenv("AGORA_APP_ID"),
    UserID: os.Getenv("AGORA_RTM_USER_ID"),
})
if err != nil { return err }
defer client.Close()

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := client.Login(ctx, os.Getenv("AGORA_RTM_TOKEN")); err != nil { return err }
if err := client.Subscribe(ctx, "room", messaging.SubscribeOptions{Presence: true}); err != nil { return err }
if err := client.Publish(ctx, "room", []byte("hello"), messaging.PublishOptions{Text: true}); err != nil { return err }

event, err := client.Next(ctx)
if err != nil { return err }
if event.Message != nil { fmt.Printf("%s: %s\n", event.Message.Publisher, event.Message.Data) }
```

## 支持的接口

- 客户端：登录、登出、更新令牌、向频道或用户发送消息、订阅与取消订阅、设置 JSON SDK 参数、查询 SDK 版本
- 在线状态（Presence）：分页查询在线用户、查询用户所在频道、设置／删除／读取状态
- 存储（Storage）：设置／更新／删除／读取频道和用户元数据，订阅与取消订阅用户元数据
- 锁（Locks）：创建、查询、删除、获取、释放和撤销
- 历史消息（History）：分页查询，返回复制后的二进制消息内容和下一页游标
- 流频道（Streams）：创建并管理频道、加入／离开频道、更新令牌、加入／离开主题、
  发布主题消息、订阅／取消订阅发布者、查询已订阅的发布者
- 事件：消息、在线状态、主题、锁、存储、连接状态和令牌事件

`Config.Native` 用于设置区域、日志、代理、加密和私有配置等原生高级选项。
设置该字段时，请先调用 `agorartm.NewRtmConfig()` 获取默认配置。
`WithNative` 提供串行化的原生接口访问，用于其他配置或特殊 API。
传入的函数必须尽快返回；不要保留原生对象、替换事件处理器、释放客户端，或在函数内再次调用该客户端的方法。
上述常用操作无需使用原生指针。

## 生命周期与并发

所有 `Client` 和 `Stream` 方法均支持并发调用。原生请求的发起过程串行执行，
各异步请求独立等待，并通过请求 ID 匹配结果；原生方法返回前到达的回调也能正确关联。
`Close` 可重复调用，先释放流频道，再释放客户端。未完成的请求以 `ErrClosed` 结束等待；
事件队列关闭后，已入队的事件仍可读取。已关闭的流频道不能再发起原生请求。

每个异步请求同时受调用方的 `context` 和 `Config.RequestTimeout` 安全超时约束，后者默认 15 秒。
取消或超时只会停止等待，不会撤销已提交的服务端操作，也无法中断正在执行的同步原生 SDK 调用。
已取消等待的消息发布、元数据修改或获取锁操作仍可能生效。不要盲目重试会改变服务端状态的操作。
本封装禁用了原生 SDK 获取锁时的自动重试。

`Events()` 返回只读的缓冲通道，默认容量为 128，也可通过 `Next(ctx)` 读取。
事件数据是独立的 Go 副本，在回调结束或客户端关闭后仍可保留使用。
队列满时丢弃新到达的事件，并增加 `DroppedEvents()` 计数；请求完成结果走独立路径，不会被丢弃。
请按负载设置队列容量并及时消费；事件队列不是持久化消息日志。
多个消费者共享同一队列，每个事件只会由其中一个消费者收到。

`Error` 包含操作名称和 `ErrorCode` 类型的错误码，保留原生 SDK 的准确数值。
使用 `errors.As` 检查 SDK 错误，使用 `errors.Is` 判断 context 错误、`ErrClosed`、
`ErrInvalidArgument` 和 `ErrUnavailable`。
部分原生接口的提交方法没有返回值，本封装仍会等待最终结果回调，不会将提交成功视为操作成功。
主题订阅部分失败时，会在 SDK 提供结果的情况下，同时返回成功／失败的发布者列表和 SDK 错误。

元数据版本号 `-1` 表示采用原生 SDK 的无条件版本行为；`0` 是实际的版本号。
状态键和元数据键不能是空字符串、包含 NUL 或无效 UTF-8；服务端的其他限制以 SDK 错误为准。
`RemoveState` 的键列表为空时，会删除全部状态键。

## 验证范围与限制

离线测试覆盖提前回调的请求关联、并发请求、同步／异步错误、context 取消、超时、
关闭竞态、队列溢出、多元素结果、深拷贝和二进制消息。原生绑定测试另外覆盖连续 C 数组和回调生命周期。
示例无需凭据即可编译。实际联网运行需要 Agora 项目、适用的令牌、已开通的 RTM 功能和足够的服务配额；
这些测试不验证真实网络行为。

仓库固定的 SDK 发布版本与 RTM 核心报告的版本不是同一概念。
使用 `messaging.Version()` 查询当前加载的 RTM 核心版本。
此客户端包不包含云端 REST 服务、服务端令牌签发或旧版 RTM 服务 API。
