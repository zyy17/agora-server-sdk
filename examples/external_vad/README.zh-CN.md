# 独立 VAD

[English](README.md) · [示例](../README.zh-CN.md)

## 用途

使用独立的原生 VAD 处理单声道或双声道 PCM。示例不创建 RTC Service、不加入频道，也不需要 App ID 或 Token。它独立于外部 APM/降噪处理器及接收音频的 VAD v2 选项。

## 运行

按[主 README](../../README.zh-CN.md) 安装原生 SDK。在仓库根目录运行：

```sh
./scripts/with_agora_env.sh go run ./examples/external_vad -channels 1
./scripts/with_agora_env.sh go run ./examples/external_vad -channels 2
./scripts/with_agora_env.sh go run ./examples/external_vad -channels 2 -input /absolute/path/to/stereo.pcm
```

未指定 `-input` 时处理一秒合成静音。指定文件只会被读取，必须是不含 WAV 文件头、声道布局与选项一致的 16 kHz 原始 PCM16LE。输出状态变化及返回字节数，不写入输出文件。Ctrl+C 会在帧之间停止；已阻塞的文件读取本身无法中断。

## 约定

- 单声道使用 `rtc.NewAudioVad`；双声道使用为兼容保留的拼写 `rtc.NewSteroVad`。检查构造结果是否为 `nil`，并始终释放检测器。
- 每次输入恰好 10 ms：每声道 160 个采样，每采样 2 字节，即单声道 320 字节、双声道 640 字节。`AudioFrame` 显式设置 PCM16 类型、采样率、声道数、采样数及字节宽度。不完整的最后一帧会被拒绝。
- 双声道输入按左右声道交错排列。两个独立检测器分别返回单声道语音片段，长度和语音边界可能不同，不能直接把它们拼接为已同步的双声道输出。
- 处理为同步调用，不按实时节奏等待。原生 VAD 状态为 0（非语音）、1（开始）、2（语音中）、3（结束）；负数表示失败。合成静音仅演示 API 用法，不验证识别准确率。
- VAD 可能缓存语音。这里只统计已返回的片段；EOF 不会排空尾部音频，`Release` 也不是排空操作。本示例不保证得到完整的语音录音。

## 离线检查

```sh
./scripts/with_agora_env.sh go test ./examples/external_vad
./scripts/with_agora_env.sh go build -o ./bin/external_vad ./examples/external_vad
```

测试使用合成数据及替代检测回调，检查帧构造、PCM 原样保留、截断输入拒绝、取消及消费者错误。测试不初始化原生 VAD，也不衡量识别效果；“运行”一节的命令会在本地初始化原生 VAD。
