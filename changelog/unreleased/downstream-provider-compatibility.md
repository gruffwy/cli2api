### English

- Base the maintenance release on official CLI2API v0.6.22, preserving verified WorkBuddy DeepSeek V4.1 Flash low/high/max efforts and forwarding an explicit max unchanged.
- Keep the 128,000-token Qoder output fallback in both the new in-process chat client and the worker path; explicit client output limits still take precedence.
- Read refreshed Qoder credentials from the existing account runtime before signing in-process chat requests, so tokens renewed by the CLI worker do not leave direct chat using stale persisted tokens.

### 中文

- 维护版本基于官方 CLI2API v0.6.22，保留已验证的 WorkBuddy DeepSeek V4.1 Flash low/high/max 思考档位，并原样转发显式 max。
- Qoder 新版进程内聊天与 worker 路径均保留 128,000 token 输出兜底，客户端显式输出上限仍优先。
- 进程内聊天签名前读取账号运行目录中的最新 Qoder 凭据，避免 CLI worker 刷新令牌后，直连聊天仍使用数据库中的旧令牌。
