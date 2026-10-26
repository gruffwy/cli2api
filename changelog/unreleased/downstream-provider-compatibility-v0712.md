### English

- Rebase the private maintenance release on upstream v0.7.12. Preserve WorkBuddy DeepSeek V4.1 Flash explicit `low` / `high` / `max` efforts and fill missing catalog effort options without changing explicitly declared options or other models.
- Keep Qoder's default output budget at 128000 when the caller omits a positive limit; explicit caller limits still win in both the Go direct and legacy worker paths.
- Legacy Qoder direct chat reads credentials refreshed by its worker on every request. PAT-backed accounts retain upstream's in-process Bridge session refresh.

### 中文

- 私有维护版升级至上游 v0.7.12。保留 WorkBuddy DeepSeek V4.1 Flash 显式 `low` / `high` / `max` 档位，并补齐目录缺省的档位选项；不修改目录明确声明的选项或其他模型。
- 客户端未提供正数输出上限时，Qoder 默认预算保持 128000；Go 直连与旧 worker 路径均优先使用客户端显式上限。
- 旧 Qoder 账号的直连聊天逐次读取 worker 刷新后的凭据；PAT 账号继续使用上游进程内 Bridge 的会话刷新机制。
