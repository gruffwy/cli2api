### English

- Preserve the verified `low`, `high`, and `max` reasoning efforts for WorkBuddy DeepSeek V4.1 Flash even when its catalog omits `supportedEfforts`.
- Use a 128,000-token Qoder output fallback when the client omits a limit, while preserving explicit client limits.

### 中文

- WorkBuddy DeepSeek V4.1 Flash 的目录未提供 `supportedEfforts` 时，继续保留已经验证的 `low`、`high`、`max` 思考档位。
- 客户端未显式提供上限时，Qoder 使用 128,000 token 输出兜底；客户端显式上限仍保持优先。
