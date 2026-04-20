# gitlab-code-reader-mcp

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

一个轻量级 MCP（Model Context Protocol）服务器，让 AI 助手能够读取和探索 GitLab 仓库代码。灵感来自 [Claude Code](https://code.claude.com) 读取本地代码的方式（Read/Grep/Glob/LSP 工具）——但面向远程 GitLab 仓库。

**9 个专注工具。单个 Go 二进制。零冗余。**

<p>
  <a href="README.md">English</a> | <a href="README_CN.md">简体中文</a>
</p>

## 为什么做这个？

现有的 GitLab MCP 服务器暴露了 100+ 个工具，覆盖所有 GitLab API。这对全功能 GitLab 自动化很好，但会用工具描述淹没 AI 的上下文窗口，让代码读取变得低效。

本项目采用不同的方法：**只做一件事，做好它**。只做代码读取。工具描述只占 ~1,000 tokens 而非数万，每个工具都针对 AI 实际探索代码的方式进行了优化。

## 工具列表

| 工具 | 灵感来源 | 功能 |
|------|----------|------|
| `gl_read_file` | Claude Code 的 `Read` | 带行号读取文件，支持行范围、智能截断 |
| `gl_read_multiple` | 批量 `Read` | 一次调用读取最多 10 个文件 |
| `gl_find_files` | Claude Code 的 `Glob` | 按 glob 模式查找文件 |
| `gl_search_code` | Claude Code 的 `Grep` | 在仓库中搜索代码内容 |
| `gl_list_directory` | `ls` / `tree` | 浏览目录结构，可配置深度 |
| `gl_read_symbols` | Claude Code 的 `readCode` | 小文件返回全文，大文件返回函数/类签名 |
| `gl_diff` | `git diff` | 查看 MR 差异或两个 ref 的对比 |
| `gl_blame` | `git blame` | 查看每行的修改者和时间 |
| `gl_commit_history` | `git log` | 查看提交历史，含增删统计 |

## 设计原则

借鉴 Claude Code 的架构设计：

- **Token 是预算** — 大文件自动截断 500 行，批量读取每文件 200 行，minified 行截断 500 字符
- **符号优先** — 超过 300 行的文件，`gl_read_symbols` 返回函数/类签名而非全文，按需用行范围读取具体实现
- **缓存** — LRU 缓存：仓库树 5 分钟、文件内容 5 分钟、项目信息 10 分钟
- **行号无处不在** — 所有文件输出带行号，AI 可以引用"第 42 行"并用 `start_line`/`end_line` 精确定位
- **指导性错误** — 错误信息不是裸 404，而是建议下一步操作
- **内置策略** — MCP `instructions` 字段教 AI 按优先级选择工具

## 快速开始

### 安装

```bash
git clone https://github.com/nanami7777777/gitlab-code-reader-mcp.git
cd gitlab-code-reader-mcp
go build -o server ./cmd/server/
```

### 运行

```bash
GITLAB_TOKEN=glpat-xxx ./server
```

### 环境变量

| 变量 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `GITLAB_TOKEN` | 是 | — | GitLab 个人访问令牌（需要 `read_api` 权限） |
| `GITLAB_URL` | 否 | `https://gitlab.com` | GitLab 实例地址 |

## MCP 客户端配置

### Kiro

添加到 `.kiro/settings/mcp.json`：

```json
{
  "mcpServers": {
    "gitlab-code-reader": {
      "command": "/path/to/gitlab-code-reader-mcp/server",
      "args": [],
      "env": {
        "GITLAB_TOKEN": "glpat-xxx",
        "GITLAB_URL": "https://gitlab.example.com"
      }
    }
  }
}
```

### Claude Code

添加到 `.claude/settings.json`：

```json
{
  "mcpServers": {
    "gitlab-code-reader": {
      "command": "/path/to/gitlab-code-reader-mcp/server",
      "args": [],
      "env": {
        "GITLAB_TOKEN": "glpat-xxx",
        "GITLAB_URL": "https://gitlab.example.com"
      }
    }
  }
}
```

### Cursor

添加到 `.cursor/mcp.json`，格式同上。

## 使用示例

连接后，直接用自然语言和 AI 助手对话：

```
"看一下 mygroup/myproject 项目的目录结构"
→ AI 调用 gl_list_directory

"找一下 src 目录下所有的 Go 文件"
→ AI 调用 gl_find_files，pattern: **/*.go

"读一下主入口文件"
→ AI 调用 gl_read_file 读取 cmd/server/main.go

"搜一下哪里处理了认证逻辑"
→ AI 调用 gl_search_code，query: "authenticate"

"看看 MR !42 改了什么"
→ AI 调用 gl_diff，merge_request_iid: 42

"这个配置文件最后是谁改的？"
→ AI 调用 gl_blame
```

## 项目 ID

工具接受 `project_id` 的两种格式：
- **数字 ID**：`609`
- **路径**：`mygroup/myproject`

两种格式都支持。在 GitLab 项目设置页面或通过 API 可以找到项目 ID。

## 符号提取

`gl_read_symbols` 使用正则表达式提取代码符号（无外部依赖）。支持的语言：

| 语言 | 检测的符号 |
|------|-----------|
| TypeScript/JavaScript | 函数、类、接口、类型、枚举、方法、箭头函数 |
| Python | 函数、类 |
| Go | 函数、结构体、接口 |
| Java/Kotlin | 类、接口、方法 |
| Rust | 函数、结构体、trait、枚举、impl 块 |

300 行以下的文件直接返回全部内容（与 Claude Code 的 `readCode` 行为一致）。

## 性能对比：vs [@zereight/mcp-gitlab](https://github.com/zereight/mcp-gitlab)

公平对比——只比较**代码读取**能力（不是全部 141 个工具）。

### 等价工具对照

| 本项目 | @zereight/mcp-gitlab 等价工具 | 差异 |
|---|---|---|
| `gl_read_file` | `get_file_contents` | 带行号、行范围、500 行自动截断、二进制检测 |
| `gl_read_multiple` | _(无)_ | 一次调用批量读取最多 10 个文件 |
| `gl_find_files` | `get_repository_tree` | 支持 glob 模式匹配（`**/*.go`），不只是平铺目录 |
| `gl_search_code` | `search_code` / `search_project_code` | 预格式化输出，带行号和上下文 |
| `gl_list_directory` | `get_repository_tree` | 可配置深度（1-3），树形展示 |
| `gl_read_symbols` | _(无)_ | 大文件只返回签名，节省 90% token |
| `gl_diff` | `list_merge_request_diffs` / `get_commit_diff` | 文件过滤、排除模式、自动截断 |
| `gl_blame` | _(无)_ | 按行范围查看 blame，格式化输出 |
| `gl_commit_history` | `list_commits` | 包含每次提交的增删统计 |

### Token 对比（仅代码读取）

| 场景 | @zereight/mcp-gitlab | 本项目 | 节省 |
|------|---|---|---|
| **工具描述（仅读取相关 ~10 个工具）** | ~2,000 tokens | ~1,000 tokens | **50%** |
| **读 61 行文件** | ~800 tokens (原始 JSON + base64) | ~500 tokens (纯文本+行号) | **37%** |
| **读 2000 行文件** | ~20,000 tokens (全量返回，无截断) | ~3,500 tokens (截断 500 行) | **82%** |
| **理解大文件结构** | 必须读全文 ~5,000 tokens | `gl_read_symbols` ~500 tokens (只返回签名) | **90%** |
| **代码审查读 5 个文件** | 5 次 `get_file_contents` 调用 | 1 次 `gl_read_multiple` | **减少 4 次往返** |

### 响应格式对比

**@zereight/mcp-gitlab** 的 `get_file_contents` 返回原始 GitLab API JSON：
```json
{"file_name":"config.ts","size":1700,"encoding":"base64",
 "content":"dXBzdHJlYW0gZW5yb2xsbWVudHMge...",
 "content_sha256":"abc123...","blob_id":"...","last_commit_id":"..."}
```
→ AI 需要在"脑中"解码 base64。无行号。大文件无截断保护。

**本项目** 的 `gl_read_file` 返回即用文本：
```
File: config.ts (1.7 KB, 61 lines, ref: main)
Showing lines 1-61 of 61
────────────────────────────────────────
 1  upstream enrollments {
 2      server host.docker.internal:8088;
...
```
→ 已解码。带行号便于精确引用。大文件自动截断。

### 速度对比

| 维度 | @zereight/mcp-gitlab | 本项目 | 原因 |
|------|---|---|---|
| **启动** | 3-5 秒 (Node.js + npx) | <0.1 秒 (Go 二进制) | 无运行时依赖 |
| **重复读取** | 每次都打 GitLab API | LRU 缓存（5 分钟 TTL） | 同 ref = 同内容 |
| **批量操作** | N 个文件 = N 次调用 | 最多 10 个文件 = 1 次调用 | `gl_read_multiple` |

### 什么时候用哪个？

- **用本项目**：当 AI 需要**读取和理解代码**时——探索仓库、审查 MR、搜索模式
- **用 @zereight/mcp-gitlab**：当需要**写操作**时——创建 MR、发评论、管理 issue、运行 pipeline

## 项目结构

```
cmd/server/main.go          # MCP 服务器入口 + 工具注册
internal/
├── gitlab/
│   ├── client.go           # GitLab API 客户端（带缓存）
│   ├── cache.go            # LRU 缓存实现
│   └── types.go            # API 响应类型定义
├── tools/
│   ├── helpers.go          # 行号格式化、截断、二进制检测
│   ├── read_file.go
│   ├── read_multiple.go
│   ├── find_files.go
│   ├── search_code.go
│   ├── list_directory.go
│   ├── read_symbols.go
│   ├── diff.go
│   ├── blame.go
│   └── commit_history.go
└── symbols/
    └── extract.go          # 基于正则的符号提取
```

## 贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md)。欢迎 PR！

## 许可证

[MIT](LICENSE)
