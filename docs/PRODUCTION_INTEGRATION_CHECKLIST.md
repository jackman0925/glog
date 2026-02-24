# Production Integration Checklist

本文档用于评估 `glog` 在大型项目中的落地可行性，并提供可执行的集成步骤。

## 1. 集成策略

- 优先使用 `New()` 或 `NewLogger()` 创建组件级 logger，通过依赖注入传递。
- 仅在“单进程、单入口、全局统一日志”场景使用 `Init()`。
- 避免在同一进程内多次调用 `Init()`；若必须重载配置，需明确切换窗口与影响范围。

## 2. 配置基线（建议）

推荐配置（生产环境起步模板）：

```yaml
encoder: json
path: "/var/log"
directory: "/my-service"
show_line: false
show_goroutine: false
encode_level: Capital
stacktrace_key: stacktrace
log_stdout: false
high_performance: true
separate_levels: false
log_level: info
segment:
  max_size: 100
  max_age: 7
  max_backups: 10
  compress: true
```

- 高吞吐服务：`high_performance: true` + `separate_levels: false`。
- 排障期可临时启用 `show_line: true`，稳定后关闭。
- 容器平台如由 Sidecar/Agent 收集 stdout，可设 `log_stdout: true`，但需评估重复采集。

## 3. 生命周期管理

- 在进程退出前执行 `defer glog.Flush()`。
- 将 logger 初始化放在应用启动早期，确保启动日志可追踪。
- 若使用 `Init()`，确认 `stderr.log` 路径具备写权限（本库会重定向 `os.Stderr`）。

## 4. 可靠性与并发

- 已验证并发安全：内部状态使用 `atomic.Value`，并通过 `go test -race`。
- 建议在 CI 保留 `go test -race ./...` 作为合并门禁。
- 对高频日志路径，避免在业务代码中做重字符串拼接，优先参数化日志。

## 5. 可观测性规范

- 建议统一字段：`service`、`env`、`version`、`trace_id`、`request_id`、`user_id`。
- 约定错误日志结构：`err`（错误摘要）+ 关键上下文字段，不写大块非结构化文本。
- 将日志级别策略固定到环境：
  - `prod`: `info` 或 `warn`
  - `staging`: `debug` 或 `info`

## 6. 资源与容量规划

- 依据峰值日志量调大 `max_size` 和 `max_backups`，避免过快轮转导致日志丢失窗口。
- 若启用 `compress: true`，需评估 CPU 峰值影响。
- 建议预留磁盘告警：日志目录使用率超过 70% 告警，超过 85% 触发降级策略。

## 7. 安全与合规

- 禁止输出敏感信息：密码、Token、完整身份证号/手机号、银行卡号。
- 对可能包含 PII 的字段进行脱敏后再写日志。
- 对日志目录设置最小权限原则（仅运行账户可写）。

## 8. 发布与回滚

- 灰度发布时先在小流量环境验证：
  - 日志是否完整
  - 轮转是否正常
  - stderr 重定向是否符合预期
- 回滚策略：保留旧版配置和二进制，出现异常时可快速回退。

## 9. 最小验收清单（上线前）

- `go test ./...` 通过。
- `go test -race ./...` 通过。
- 压测场景下日志无明显丢失、无异常延迟增长。
- 日志采集链路（Agent/平台）可正常解析 JSON 字段。
- 磁盘占用与轮转策略符合容量预算。

