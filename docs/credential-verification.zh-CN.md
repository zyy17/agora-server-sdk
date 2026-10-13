# Agora 凭证验证：仅验证身份认证

该手动流程分别验证 **RTC 加入频道认证**和 **RTM 登录认证**。它不是 SDK
全功能测试，不验证音视频、消息投递、AI/模型服务、续期、重连或生产负载。

## 在 GitHub Actions 中运行

1. 在仓库 Actions **Secrets** 中配置名称完全一致的 `AGORA_APP_ID` 和
   `AGORA_APP_CERTIFICATE`，值均为 32 位十六进制字符。若没有同名 Secret，
   App ID 也可从 Actions variable `AGORA_APP_ID` 读取；证书只能放在 Secrets。
2. 默认分支必须包含 `.github/workflows/agora-verify.yml`，GitHub 才会显示
   手动运行入口；完整实现也必须已进入 `main`。
3. 打开 **Actions → Agora credential verification → Run workflow**，选择
   **Branch: main**，由你亲自点击 **Run workflow**。流程拒绝其他分支和
   其他仓库；没有凭证输入框，没有 push/PR 自动触发，也不依赖环境保护配置。
4. 在运行摘要中核对实际评估的提交 SHA，并查看 live-check 步骤内独立的 RTC、
   RTM 结果。检出的是 dispatch 对应的精确 SHA，不会跟随点击后移动的分支。

首个步骤只检查是否配置及格式是否正确，不输出值。编译和离线测试不会取得凭证。
只有明确手动触发的在线步骤使用凭证联系 Agora。建议使用测试项目；短时连接仍可能
计入 Agora 用量或费用。

## 成功的含义

- RTC：为随机字符串账号和频道生成 120 秒有效、仅含加入权限的 AccessToken2，
  创建一个 Service，等待加入成功，然后执行 Session Close（断开/释放）和
  Service Close。`leave` 表示本地断开/释放结果，不是服务端离开回执。
- RTM：独立生成随机字符串账号的 120 秒登录 token，等待登录成功，然后使用新的
  5 秒上下文尝试登出并关闭客户端。不创建消息频道或 Stream 频道。
- 仅成功生成 token 不能证明认证有效。RTM 未开通或验证失败不否定 RTC 的独立成功；
  数字错误码应按固定版本 SDK 对应服务的文档解释。
- 任一验证失败或不完整，工作流均失败。格式和认证验证不能证明所有功能已开通，
  也不证明项目认证配置一定按预期强制校验证书。

## 时限与凭证保护

RTC 和 RTM 使用不同子进程。每项操作最多 25 秒，监督进程在 35 秒发出终止信号，
再给 5 秒清理宽限。第一项失败后仍尝试第二项。整个在线命令硬上限为 90 秒；
包含安装和编译的任务上限为 15 分钟。正常、出错、取消路径均尝试清理已创建的资源；
若原生调用卡住，最终强制终止，不能保证此时的服务端清理。

输出仅含允许的操作名、状态和数字错误码。SDK 错误详情、token、账号、证书和原生
stdout/stderr 不会输出。派生 token 只驻留子进程内存，连 masking 命令也不打印。
原生日志和 SDK 状态位于私有临时目录，每项结束后删除，并有 always 清理步骤兜底。
不上传在线日志或其他产物。不要开启 shell tracing 或上传含凭证的诊断文件。

## 离线开发

安装仓库固定校验和版本的原生库后运行：

```sh
./scripts/with_agora_env.sh go test -race ./internal/cmd/agoraverify
./scripts/with_agora_env.sh go build -o /tmp/agora-verify ./internal/cmd/agoraverify
```

测试仅使用合成凭证和假传输，不连接 Agora，覆盖错误路径清理、独立结果、token
权限/时长及结果字段检查。命令的 `--check` 只检查格式；`--live` 还要求显式设置
`AGORA_VERIFY_LIVE=1`。不要将真实凭证写入命令行或源码。

官方来源及 token 相对有效时长说明见[英文说明](credential-verification.md#references)。
