# 同时使用 RTC 与 RTM

[English](README.md) · [示例](../README.zh-CN.md)

## 用途

在同一进程中使用数值 RTC UID 和独立的 RTM 字符串用户 ID。通过 RTC 数据流、RTC 音频元数据以及 RTM 消息频道发送不透明的二进制载荷。示例仅演示传输方式，不定义应用消息格式。

## 运行

按[主 README](../../README.zh-CN.md) 安装原生 SDK。请在不会提交到仓库的位置设置以下变量：

- `AGORA_APP_ID`：示例中两个客户端共用的项目
- `AGORA_USER_ID`：数值 RTC UID，例如 `12345`
- `AGORA_CHANNEL`：RTC 频道，默认为 `go-sdk-example`
- `AGORA_TOKEN`：需要鉴权时，由调用方提供、匹配该频道和数值 UID 的 RTC Token
- `AGORA_RTM_USER_ID`：独立的 RTM 字符串身份，例如 `demo-notifier`
- `AGORA_RTM_TOKEN`：需要鉴权时，由调用方提供、匹配该 RTM 身份的 RTM Token

```sh
./scripts/with_agora_env.sh go run ./examples/rtc_rtm -rtm-channel go-sdk-notifications -listen 5s
```

在仓库根目录运行，并使用已获授权的专用测试参与者。运行会加入 RTC，发布约一秒单声道静音、一条数据消息及音频元数据，同时登录 RTM、订阅并发布一条二进制通知。不请求 RTM 历史存储。要观察对端接收，需要另一位使用不同 RTC 和 RTM 身份的参与者；不要假设会收到自己发送的消息。Ctrl+C 会开始清理。

## 约定

- RTC 数值身份模式不改变 RTM 的身份语义。RTM 频道名及身份独立配置；字符串相同不代表两个传输通道相互关联。
- 在可信服务端使用 Agora 官方 Token 构建器生成所需权限。不要用仅支持字符串账号的 `token.RTC` 辅助函数代替数值 UID 或 RTM Token。若服务端使用组合 Token，构建仍由调用方负责；不要假设两个凭据可以互换。
- 载荷包含 NUL 和非 UTF-8 字节。RTM 使用 `PublishOptions{Text: false}`。RTC 数据及音频元数据有各自的限制和投递语义；示例不定义跨通道顺序、同步或投递确认。此处的 RTC 数据流不保证可靠或有序投递。
- 音频元数据伴随已发布的音频，与 RTM 频道元数据不同。本示例发送音频元数据，但关闭了自动音频订阅；要接收某个对端的音频元数据，必须订阅该对端的音频。RTC 数据消息不需要音频订阅。
- 三个消费者在原生回调之外运行。正常清理使用新的有限时长 Context 登出，关闭 RTC 和 RTM 生产者，排空事件/消息队列，等待消费者退出并报告丢弃数量。关闭队列不代表排空网络数据或确认对端收到。Token 提示会说明哪个客户端的 `RenewToken` 需要其自身的新 Token。

## 离线检查

```sh
./scripts/with_agora_env.sh go test ./examples/rtc_rtm
./scripts/with_agora_env.sh go build -o ./bin/rtc_rtm ./examples/rtc_rtm
```

测试检查关闭队列后，缓存的二进制内容会在 EOF 之前被消费。构建和测试不会加入 RTC 或登录 RTM，也不代表已验证真实互通。独立操作见 [RTM 示例](../messaging/README.zh-CN.md)。
