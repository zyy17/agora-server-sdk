# RTM 场景示例

[English](README.md) · [全部示例](../README.zh-CN.md) · [RTM 指南](../../messaging/README.zh-CN.md)

## 编译与凭据

在仓库根目录编译，不执行登录：

```sh
make download-agora-libs
./scripts/with_agora_env.sh go build -o bin/messaging ./examples/messaging
```

| 环境变量 | 用途 |
| --- | --- |
| `AGORA_APP_ID` | 必填，项目 App ID |
| `AGORA_RTM_USER_ID` | 必填，唯一 RTM 登录身份 |
| `AGORA_RTM_TOKEN` | RTM 登录 Token；项目开启鉴权时必填 |
| `AGORA_RTM_STREAM_TOKEN` | 虽名为 RTM，实际为匹配 Stream Channel 和身份的 **RTC Token**；仅 `stream` 使用 |

Token 由可信业务服务端签发，不要嵌入 App Certificate。使用专用测试用户和频道，提前开通所需功能。不同客户端使用不同登录 ID。只有运行程序时才执行下述真实操作。

## 运行命令

```sh
./scripts/with_agora_env.sh ./bin/messaging -scenario pubsub -channel test-room -listen 30s
./scripts/with_agora_env.sh ./bin/messaging -scenario direct -peer another-user
./scripts/with_agora_env.sh ./bin/messaging -scenario presence -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario metadata -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario locks -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario history -channel test-room
./scripts/with_agora_env.sh ./bin/messaging -scenario stream -channel test-stream -peer another-user
```

| 参数 | 默认值与含义 |
| --- | --- |
| `-scenario` | `pubsub`；下列七种场景之一 |
| `-channel` | `go-sdk-example`；按场景作为 Message Channel 或 Stream Channel |
| `-peer` | 默认空值；`direct` 必填；`stream` 省略时使用自身用户 ID |
| `-listen` | `0`；场景执行**完成后**的额外监听时长，不能为负 |

## 操作与副作用

| 场景 | 操作 | 前提与限制 |
| --- | --- | --- |
| `pubsub` | 订阅并发布一条文本消息 | 用另一个在线客户端验证接收；未开启历史存储 |
| `direct` | 向 `-peer` 发布一条文本消息 | 对端须已登录；不订阅频道 |
| `presence` | 设置两个 `go_example_*` 状态键，查询用户、状态和频道，再删除这些键 | 开通 Presence 并启用事件 |
| `metadata` | 设置、更新、读取、删除频道和用户的 `go_example` 元数据；订阅并取消用户元数据 | 开通存储；写入显式使用版本 `-1` |
| `locks` | 创建 TTL 为 30 秒的 `go_example_lock`，获取、查询、释放并删除 | 开通锁；TTL 表示断开或离开后保留归属的时间，不是锁定义存活期 |
| `history` | 查询一页 Message Channel 历史，数量 20 | 开通历史，且已有消息发布时开启 `StoreInHistory: true`；输出下一页游标，不继续分页 |
| `stream` | 创建并加入 Stream，加入 Topic，订阅发布者，发布消息，查询并取消发布者订阅，离开 Topic/Stream 并关闭 | 按需提供 RTC Stream Token；示例开启了 Presence，需对应权限 |

Stream 使用 `go_example_topic`。`Subscribers` 返回本客户端订阅的发布者 ID，不是其观众列表。Stream 在 `-listen` 开始**之前**已取消订阅、离开并关闭；增加该时长不会使本场景持续接收 Topic。测试接收时，应使用不同客户端，并调整生命周期，让双方保持加入状态。

## 退出与验证

场景中途失败可能留下已发送消息或已修改的状态、元数据和锁。Ctrl+C 只取消等待，不撤销已提交操作；重试前检查测试资源。正常路径执行表中清理，但不提供事务或回滚。

登录成功后，退出时用独立的 3 秒 context 尝试 `Logout`，再释放原生资源。客户端创建成功后，清理阶段会输出事件丢弃数，包括失败退出。溢出可能丢失消息、连接变化和 Token 通知。示例仅记录续期提示；长时间运行前应实现 `Client.RenewToken` 更新 RTM Token，并用 `Stream.RenewToken` 更新 Stream 的 RTC Token。

示例不打印 Token 或 App ID；SDK 日志需另行保护。编译和离线测试不验证真实送达、功能权限、历史可用性或对端行为。参见 [RTM 约定](../../messaging/README.zh-CN.md)和[联调清单](../../docs/development.zh-CN.md#真实联调)。
