# RTM 场景示例

[English](README.md) | [简体中文](README.zh-CN.md)

仅编译，不连接 Agora：

```sh
make download-agora-libs
./scripts/with_agora_env.sh go build -o bin/messaging ./examples/messaging
```

运行前必须显式设置 `AGORA_APP_ID` 和 `AGORA_RTM_USER_ID`；如果项目要求令牌，还需设置 `AGORA_RTM_TOKEN`。
生产环境的 RTM 令牌应由自己的服务端签发；不要将 App Certificate 写入本示例。
涉及修改服务端状态的场景，请使用专用测试频道和测试用户。
各场景仅开启所需的订阅事件；请确保项目已开通所选场景需要的 RTM 功能。

```sh
./scripts/with_agora_env.sh ./bin/messaging -scenario pubsub -channel test-room -listen 30s
./scripts/with_agora_env.sh ./bin/messaging -scenario direct -peer another-user
./scripts/with_agora_env.sh ./bin/messaging -scenario presence -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario metadata -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario locks -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario history -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario stream -channel test-stream -peer another-user
```

- `pubsub`：订阅频道并发布一条文本消息；可通过 `-listen` 指定继续监听的时长
- `direct`：向 `-peer` 指定的用户发送消息，该参数必填
- `presence`：设置两个 `go_example_*` 状态键，查询用户、状态和频道，再删除这些键
- `metadata`：设置／更新／读取／删除频道和用户的 `go_example` 元数据键，并订阅／取消订阅用户元数据；需要开通存储功能
- `locks`：创建有效期为 30 秒的 `go_example_lock`，获取／查询／释放／删除该锁；需要开通锁功能
- `history`：读取一页历史消息；需要开通历史消息功能，并且消息发布时已启用历史存储
- `stream`：创建并加入流频道，加入主题，订阅／查询／取消订阅发布者，发布消息后离开；
  如需流频道令牌，设置 `AGORA_RTM_STREAM_TOKEN`

只有显式运行场景时，才会执行上述真实服务操作。
场景中途失败可能留下先前的状态或元数据修改；请使用独立的新测试资源，并在重试前检查已有结果。
Ctrl+C 只取消等待，不会撤销已提交的服务端操作。
退出时会释放原生资源；登录成功后，还会使用独立的短超时尝试登出。
事件队列可能在过载时丢弃事件；只要客户端创建成功，示例就会在退出清理时输出丢弃计数，包括出错退出时。
示例不会打印令牌或 App ID；请妥善保护 SDK 自身生成的日志。
