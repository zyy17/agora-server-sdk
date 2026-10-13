# 双向 PCM

[English](README.md) · [示例](../README.zh-CN.md)

## 用途

使用数值 RTC UID 加入频道，发送一段合成双声道音调，同时消费接收到的单声道 PCM。无需音频设备或输入文件。

## 运行

按[主 README](../../README.zh-CN.md) 安装原生 SDK。在仓库根目录设置 `AGORA_APP_ID`、`AGORA_CHANNEL`、`AGORA_USER_ID`（例如 `12345`），项目开启鉴权时还需设置匹配的 `AGORA_TOKEN`。通过可信服务端使用 Agora 官方 Token 构建器生成数值 UID 的 RTC Token；`token.RTC` 辅助函数使用字符串账号。

```sh
./scripts/with_agora_env.sh go run ./examples/duplex_pcm -duration 2s -pts 123
```

在同一频道启动另一位已获授权的参与者，使用不同的数值 UID，以收听音调并提供返回音频。请同时启动双方；本示例发送完毕并估算完成后退出。Ctrl+C 会取消发送节奏并关闭会话。运行会加入真实频道并发布音频；构建和测试不会。

## 约定

- `Config.IdentityMode = sdk.NumericIdentity` 对整个 Service 生效。UID 是规范十进制 `uint32` 字符串；请选择正数 UID 及其匹配的 Token。默认身份模式中的纯数字字符串仍属于字符串账号。
- `Audio.Send` 为 16 kHz 双声道；`Audio.Receive` 独立设为 16 kHz 单声道。PCM 为有符号、交错排列的 PCM16LE。接收音频只计数后丢弃，不录制或回传。
- 数据块依次循环使用 7、13、20 ms；最后一块缩短为剩余的整数毫秒。`SendPCMAt` 接受由完整采样组成、时长为整数毫秒的数据。44.1 kHz 下，最小的整数毫秒块为 10 ms。
- 每块都原样传递同一个 `-pts` 值。此处它是调用方的不透明值，不生成时间轴。发送节奏使用可取消的等待；发送变慢不会触发追赶式突发发送。普通文件按 10 ms 节奏发送可直接使用 `PlayPCM`。
- 音频及事件消费在原生回调之外运行。关闭时等待消费者退出并报告队列丢弃数量。这个短示例会在取消时丢弃待消费的接收项，不是无损录音器。`WaitPCM` 只估算本地完成时间，不代表远端播放确认。

## 离线检查

```sh
./scripts/with_agora_env.sh go test ./examples/duplex_pcm
./scripts/with_agora_env.sh go build -o ./bin/duplex_pcm ./examples/duplex_pcm
```

测试使用合成数据及替代发送/等待函数，检查分块边界、采样连续性、PTS 原样传递、取消及参数校验；不代表已验证真实 RTC 互通。
