# 有界 Agora 在线验证

手动工作流对 Agora 执行选定的真实 RTC/RTM 测试。默认 **integration** 范围使用
合成双端、音视频和消息；**auth** 仅验证 RTC 加入/离开与 RTM 登录/登出。
它不等于 SDK 全 API、所有权限、编解码器、部署环境或生产负载均已验证。

## GitHub Actions 操作

1. 配置仓库 Actions **Secrets**：`AGORA_APP_ID`、`AGORA_APP_CERTIFICATE`，
   均为 32 位十六进制字符。若没有同名 Secret，App ID 可读取 Actions variable
   `AGORA_APP_ID`；证书必须放在 Secrets。
2. 将工作流和测试实现合入 `main`。GitHub 要求默认分支有工作流才提供手动入口。
3. 打开 **Actions → Agora live verification → Run workflow**，选择
   **Branch: main**，范围选 **integration**（默认）或 **auth**，亲自点击运行。
   其他分支/仓库会被拒绝。
4. 查看摘要的精确提交 SHA 和在线步骤中独立的操作结果。检出锁定 dispatch SHA，
   并核验 HEAD，不跟随之后移动的分支。

没有 push/PR 自动在线测试、凭证输入框或环境保护假设。首步只验证存在性和格式，
不输出值。编译及离线测试不会取得凭证；只有手动在线步骤联系 Agora。建议使用测试
项目，短时合成音视频和消息也可能计入用量或费用。

## integration 实测范围

每次使用随机字符串身份和随机频道名，token 有效期 120 秒，不调用外部 AI/模型服务。

- RTC：一个 Service、一个接收观众和一个发送主播，均须真实连接。发送约四秒
  16 kHz 单声道 440 Hz 测试音、160×120 I420/10 fps 明暗图案，以及小型二进制
  数据消息。接收端核验来源、音频频谱、解码图案和消息完整字节；本地发送成功
  不等于投递成功。音视频允许有损编解码误差。
- RTC 续期：提交新接收端 token，仅证明本地接受，不证明跨过期或断网重连。
- RTM：两个客户端登录并订阅随机消息频道；发送不存历史的二进制消息，要求对端
  回调的频道、发布者、类型、自定义类型和字节内容一致。续期发送端 token，验证
  临时 presence 状态写入/对端读回，并只删除本次使用的状态键。
- RTM 权限读取：查询随机频道的 metadata、lock 定义和一页 history。这是只读
  可用性检查，不证明持久写入或历史消息投递。不创建持久 metadata、lock 或历史消息。
- 清理：双方取消订阅、登出，关闭 RTC 会话和所有客户端/Service。RTC `leave`
  表示本地断开/释放，不是服务端离开回执。

未执行的计划项标记 `not_run`，超时和失败分别报告。后续功能失败不会掩盖双端认证
已成功的结果。RTM 功能可能需要单独开通，请结合原生数字错误码诊断。任何失败、
超时或未执行计划项都会使整体流程失败；单纯生成 token 不算认证成功。

未在线覆盖：其他编码/编码媒体和 CPU 像素格式、AI/VAD/APM、加密、代理/地域变体、设备采集播放、
RTC 角色/订阅组合、RTM Stream topic、持久 metadata/lock 写入、持久 history 内容、
token 过期、强制重连/故障注入、负载压力。离线 API 合约测试不能替代这些真实验证。
认证成功也不证明项目必定按预期强制校验证书。

## 示例/演示覆盖

本测试用独立的确定性双端覆盖 PCM、原始视频、RTC 数据和 RTM 消息示例使用的 API
路径，并没有运行这些示例的命令行程序、voice-agent 演示、外部 AI/模型集成或全部
参数组合。既有 SDK CI 编译示例并执行已有离线示例测试；编译成功不能当作演示的
真实端到端结果。

## auth 范围

创建一个禁用媒体的 RTC 观众会话和一个独立 RTM 客户端，使用仅加入和仅登录权限的
token，等待真实成功后离开、登出、释放。不发送音视频或应用消息。RTC 和 RTM
结果独立；RTM 未开通不否定 RTC 认证成功。

## 时限与凭证保护

RTC/RTM 各自独立子进程，操作总上下文 25 秒；35 秒监督截止，再给 5 秒终止宽限。
第一项失败仍尝试第二项，完整在线命令硬上限 90 秒。含安装编译的任务上限 15 分钟。
正常、失败、取消均尝试清理，RTM 清理用新的有界上下文；原生调用卡住最终强制终止，
因此不能保证硬终止后的服务端清理。只触及本次随机身份/频道和精确临时键，不删除
其他/共享状态。

输出仅含允许的操作/状态名和数字码，不输出原生详情、凭证、token、身份或原生
stdout/stderr。token 仅驻留子进程内存，连 masking 命令也不打印。禁用 core dump；
原生日志和状态位于私有临时目录，每项结束和工作流 always 步骤删除。不上传在线
日志或产物。不要开启 shell tracing 或上传含凭证诊断文件。

## 离线开发

安装仓库固定校验和版本的原生库后运行：

```sh
./scripts/with_agora_env.sh go test -race ./internal/cmd/agoraverify
./scripts/with_agora_env.sh go build -o /tmp/agora-verify ./internal/cmd/agoraverify
```

测试仅用合成凭证/假传输，不连接 Agora，覆盖内容校验、清理、token 权限、报告净化
和子进程超时。`--check` 只检查格式；`--live` 为 auth，`--live --features` 为
integration，两者还需 `AGORA_VERIFY_LIVE=1`。不要将真实凭证放入命令行或源码。

官方来源及相对有效时长说明见[英文说明](credential-verification.md#references)。
