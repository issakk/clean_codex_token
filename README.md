# clean_codex_token (Go 版)

将原 `clean_codex_accounts.py` 迁移为 Go 项目后的命令行工具。

## 1. 环境要求

- Go 1.22+

## 2. 安装依赖与构建

在项目根目录执行：

```bash
go mod tidy
go build -o clean-codex-accounts ./cmd/clean-codex-accounts
```

构建后会生成可执行文件：

- Linux/macOS: `./clean-codex-accounts`
- Windows: `clean-codex-accounts.exe`

## 3. 快速启动示例

### 3.1 仅检查 401 并导出（命令行模式）

```bash
./clean-codex-accounts \
  --token "你的管理token" \
  --base-url "http://127.0.0.1:8317" \
  --target-type "codex" \
  --provider "openai" \
  --workers 120 \
  --timeout 12 \
  --retries 1 \
  --output "invalid_codex_accounts.json"
```

执行后会输出失效账号（`status_code == 401`）到 `invalid_codex_accounts.json`。

### 3.2 检查后立即删除

```bash
./clean-codex-accounts \
  --token "你的管理token" \
  --base-url "http://127.0.0.1:8317" \
  --delete \
  --yes
```

- `--delete`：检查完后删除 401 失效账号
- `--yes`：跳过二次确认

### 3.3 从 output 直接删除（跳过检查）

```bash
./clean-codex-accounts \
  --token "你的管理token" \
  --base-url "http://127.0.0.1:8317" \
  --delete-from-output \
  --output "invalid_codex_accounts.json"
```

### 3.4 无人值守 cron 定时执行（检查401并自动删除）

```bash
./clean-codex-accounts \
  --token "你的管理token" \
  --base-url "http://127.0.0.1:8317" \
  --cron "*/10 * * * *"
```

- `--cron` 使用 5 段表达式（分 时 日 月 周）
- 启用后将进入无人值守循环模式，并固定为“检查401并自动删除”
- cron 模式下会自动跳过删除确认（等价 `--yes`）
- cron 模式下必须提供 token（`--token`、`MGMT_TOKEN` 或 `--har`）

常见示例：

- 每 10 分钟：`*/10 * * * *`
- 每天 03:30：`30 3 * * *`


## 4. 交互模式

如果你不传 `--delete` 且不传 `--delete-from-output`，程序会进入菜单：

1. 仅检查 401 并导出
2. 检查 401 并立即删除
3. 直接删除 output 文件中的账号
0. 退出

并会提示你输入：

- `workers`
- `delete-workers`
- `timeout`
- `retries`

## 5. 使用 HAR 自动提取参数

你可以通过 `--har` 自动提取：

- token
- base_url
- user_agent
- chatgpt_account_id

示例：

```bash
./clean-codex-accounts --har "./sample.har"
```

如果 HAR/配置中没拿到 token，程序会提示你手动输入。

## 6. 配置文件（config.json）

默认读取 `config.json`（可通过 `--config` 指定路径）。

示例：

```json
{
  "base_url": "http://127.0.0.1:8317",
  "token": "your-token",
  "target_type": "codex",
  "provider": "openai",
  "workers": 120,
  "delete_workers": 20,
  "timeout": 12,
  "retries": 1,
  "output": "invalid_codex_accounts.json"
}
```

## 7. 常用参数

- `--config` 配置文件路径（默认 `config.json`）
- `--base-url` 管理服务地址
- `--token` 管理 token（也可用环境变量 `MGMT_TOKEN`）
- `--har` HAR 文件路径（自动提取上下文）
- `--target-type` 按 `type/typo` 过滤（默认 `codex`）
- `--provider` 按 provider 过滤（可选）
- `--workers` 探测并发（默认 120）
- `--delete-workers` 删除并发（默认 20）
- `--timeout` 请求超时秒数（默认 12）
- `--retries` 探测失败重试次数（默认 1）
- `--output` 输出 JSON 文件（默认 `invalid_codex_accounts.json`）
- `--cron` cron表达式（5段），无人值守定时执行“检查401并自动删除”
- `--delete` 检查后删除
- `--delete-from-output` 从 output 直接删除
- `--yes` 删除时跳过 `DELETE` 二次确认

## 8. 运行测试

```bash
go test ./...
```
