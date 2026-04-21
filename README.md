# gitlab-code-reader-mcp

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/nanami7777777/gitlab-code-reader-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/nanami7777777/gitlab-code-reader-mcp/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nanami7777777/gitlab-code-reader-mcp)](https://goreportcard.com/report/github.com/nanami7777777/gitlab-code-reader-mcp)

<p align="center">
  <a href="#english">English</a> | <a href="#中文">中文</a>
</p>

---

<a id="english"></a>

## English

A lightweight MCP (Model Context Protocol) server that lets AI assistants read and explore GitLab repository code. Inspired by how [Claude Code](https://code.claude.com) reads local codebases with its Read/Grep/Glob/LSP tools — but for remote GitLab repos.

**9 focused tools. Single Go binary. Zero bloat.**

### Why?

Existing GitLab MCP servers expose 100+ tools covering every GitLab API. That's great for full GitLab automation, but it floods the AI's context with tool descriptions and makes code reading inefficient.

This project takes a different approach: **do one thing well**. Just code reading. The tool descriptions fit in ~500 tokens instead of thousands, and every tool is optimized for how AI assistants actually explore code.

### Tools

| Tool | Inspired by | What it does |
|------|-------------|-------------|
| `gl_read_file` | Claude Code's `Read` | Read a file with line numbers, line ranges, smart truncation |
| `gl_read_multiple` | Batch `Read` | Read up to 10 files in one call |
| `gl_find_files` | Claude Code's `Glob` | Find files by glob pattern (`**/*.ts`, `src/**/*.go`) |
| `gl_search_code` | Claude Code's `Grep` | Search code content across the repo |
| `gl_list_directory` | `ls` / `tree` | Browse directory structure with configurable depth |
| `gl_read_symbols` | Claude Code's `readCode` | Small files → full content. Large files → function/class signatures |
| `gl_diff` | `git diff` | View MR diffs or compare two refs, with file filtering |
| `gl_blame` | `git blame` | See who changed each line and when |
| `gl_commit_history` | `git log` | View commit history with addition/deletion stats |

### Design Principles

Borrowed from Claude Code's architecture:

- **Token is budget** — Large files auto-truncate at 500 lines. Batch reads cap at 200 lines/file. Minified lines get cut at 500 chars.
- **Symbols first** — For files over 300 lines, `gl_read_symbols` returns function/class signatures instead of dumping the whole file.
- **Caching** — LRU cache for repository trees (5 min), file content (5 min), project info (10 min).
- **Line numbers everywhere** — All file output includes line numbers so the AI can reference "line 42" and use `start_line`/`end_line` to zoom in.
- **Guided errors** — Instead of raw 404s, errors suggest what to try next.
- **Built-in strategy** — The MCP `instructions` field teaches the AI when to use which tool.

### Quick Start

**Option 1: Go install (recommended)**
```bash
go install github.com/nanami7777777/gitlab-code-reader-mcp/cmd/server@latest
GITLAB_TOKEN=glpat-xxx gitlab-code-reader-mcp
```

**Option 2: Build from source**
```bash
git clone https://github.com/nanami7777777/gitlab-code-reader-mcp.git
cd gitlab-code-reader-mcp
go build -o server ./cmd/server/
GITLAB_TOKEN=glpat-xxx ./server
```

**Option 3: Docker**
```bash
docker run --rm -e GITLAB_TOKEN=glpat-xxx -e GITLAB_URL=https://gitlab.example.com \
  ghcr.io/nanami7777777/gitlab-code-reader-mcp
```

**Option 4: Download binary**

Grab a pre-built binary from [Releases](https://github.com/nanami7777777/gitlab-code-reader-mcp/releases).

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITLAB_TOKEN` | Yes | — | GitLab personal access token (needs `read_api` scope) |
| `GITLAB_URL` | No | `https://gitlab.com` | GitLab instance URL |

### MCP Client Configuration

Add to your MCP config (Kiro: `.kiro/settings/mcp.json`, Claude Code: `.claude/settings.json`, Cursor: `.cursor/mcp.json`):

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

### Usage Examples

```
"Show me the directory structure of project mygroup/myproject"  →  gl_list_directory
"Find all Go files in the src directory"                        →  gl_find_files
"Read the main entry point"                                     →  gl_read_file
"Search for where authentication is handled"                    →  gl_search_code
"Show me what changed in MR !42"                                →  gl_diff
"Who last modified the config file?"                            →  gl_blame
```

### Symbol Extraction

`gl_read_symbols` uses regex-based extraction. Supported: TypeScript/JavaScript, Python, Go, Java/Kotlin, Rust. Files under 300 lines return full content instead.

### Benchmarks: vs [@zereight/mcp-gitlab](https://github.com/zereight/mcp-gitlab)

**40% less data transferred. 25% fewer tool calls. Measured, not estimated.**

We benchmarked both MCP servers on the same task: *"Understand how the Worker consumes MQ messages"* in a real Go project (7 source files, ~1,500 lines total). The benchmark script calls the same GitLab API endpoints and measures the actual response sizes each MCP would return.

```
Task: Read 7 source files to understand MQ consumption architecture
────────────────────────────────────────────────────────────────────
                          New MCP          Old MCP (@zereight)
Tool calls                6                8
Total response size       31.5 KB          52.3 KB
Response format           plain text       raw JSON + base64
Batch read support        ✓                ✗
────────────────────────────────────────────────────────────────────
Savings                   40% smaller      baseline
                          2 fewer calls
```

**Where the savings come from:**

| Step | This project | @zereight/mcp-gitlab | Difference |
|------|---|---|---|
| List project structure | 1.5 KB (formatted tree) | 13.5 KB (raw JSON array) | **89% smaller** |
| Read 3 MQ files | 1 call via `gl_read_multiple` | 3 separate `get_file_contents` calls | **2 fewer calls** |
| Each file response | Decoded text + line numbers | JSON wrapper + base64 content + sha256/blob_id metadata | **~25% smaller per file** |

**Run the benchmark yourself:**
```bash
cd benchmark
GITLAB_TOKEN=glpat-xxx GITLAB_URL=https://gitlab.example.com go run main.go
```

**Equivalent Tools**

| This project | @zereight/mcp-gitlab | Difference |
|---|---|---|
| `gl_read_file` | `get_file_contents` | Line numbers, line ranges, auto-truncation, binary detection |
| `gl_read_multiple` | _(none)_ | Batch read up to 10 files in one call |
| `gl_find_files` | `get_repository_tree` | Glob pattern matching, not just flat listing |
| `gl_search_code` | `search_code` | Deduplicated results grouped by file |
| `gl_read_symbols` | _(none)_ | Large files → signatures only, saving 90% tokens |
| `gl_diff` | `list_merge_request_diffs` | File filtering, exclude patterns, auto-truncation |
| `gl_blame` | _(none)_ | Line-range blame with formatted output |

**When to use which?**
- **This project** → read and understand code
- **@zereight/mcp-gitlab** → write operations (create MRs, post comments, manage issues)

### Architecture

```
cmd/server/main.go            # Entry point
internal/gitlab/              # API client + cache
internal/tools/               # 9 tool implementations
internal/symbols/             # Regex-based symbol extraction
```

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). PRs welcome!

### License

[MIT](LICENSE)

---

<a id="中文"></a>

## 中文

一个轻量级 MCP（Model Context Protocol）服务器，让 AI 助手能够读取和探索 GitLab 仓库代码。灵感来自 [Claude Code](https://code.claude.com) 读取本地代码的方式（Read/Grep/Glob/LSP 工具）——但面向远程 GitLab 仓库。

**9 个专注工具。单个 Go 二进制。零冗余。**

### 为什么做这个？

现有的 GitLab MCP 服务器暴露了 100+ 个工具，覆盖所有 GitLab API。这对全功能自动化很好，但会用工具描述淹没 AI 的上下文窗口，让代码读取变得低效。

本项目只做一件事：**代码读取**。工具描述只占 ~1,000 tokens，每个工具都针对 AI 探索代码的方式优化。

### 工具列表

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

### 设计原则

- **Token 是预算** — 大文件自动截断 500 行，批量每文件 200 行，minified 行截断 500 字符
- **符号优先** — 超过 300 行的文件返回函数/类签名而非全文
- **缓存** — LRU 缓存：仓库树 5 分钟、文件 5 分钟、项目信息 10 分钟
- **行号无处不在** — 所有输出带行号，AI 可精确引用
- **指导性错误** — 错误信息建议下一步操作
- **内置策略** — MCP instructions 教 AI 按优先级选工具

### 快速开始

**方式一：Go install（推荐）**
```bash
go install github.com/nanami7777777/gitlab-code-reader-mcp/cmd/server@latest
GITLAB_TOKEN=glpat-xxx gitlab-code-reader-mcp
```

**方式二：从源码构建**
```bash
git clone https://github.com/nanami7777777/gitlab-code-reader-mcp.git
cd gitlab-code-reader-mcp
go build -o server ./cmd/server/
GITLAB_TOKEN=glpat-xxx ./server
```

**方式三：Docker**
```bash
docker run --rm -e GITLAB_TOKEN=glpat-xxx -e GITLAB_URL=https://gitlab.example.com \
  ghcr.io/nanami7777777/gitlab-code-reader-mcp
```

**方式四：下载二进制**

从 [Releases](https://github.com/nanami7777777/gitlab-code-reader-mcp/releases) 下载预编译二进制。

| 变量 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `GITLAB_TOKEN` | 是 | — | GitLab 个人访问令牌（需要 `read_api` 权限） |
| `GITLAB_URL` | 否 | `https://gitlab.com` | GitLab 实例地址 |

### MCP 客户端配置

添加到 MCP 配置文件（Kiro: `.kiro/settings/mcp.json`，Claude Code: `.claude/settings.json`，Cursor: `.cursor/mcp.json`）：

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

### 使用示例

```
"看一下 mygroup/myproject 的目录结构"    →  gl_list_directory
"找一下 src 下所有 Go 文件"              →  gl_find_files
"读一下主入口文件"                        →  gl_read_file
"搜一下哪里处理了认证"                    →  gl_search_code
"看看 MR !42 改了什么"                    →  gl_diff
"这个文件最后谁改的？"                    →  gl_blame
```

### 性能对比：vs [@zereight/mcp-gitlab](https://github.com/zereight/mcp-gitlab)

**数据传输量减少 40%。工具调用减少 25%。实测数据，非估算。**

我们用同一个任务对两个 MCP 做了基准测试：*"理解 Worker 如何消费 MQ 消息"*，真实 Go 项目（7 个源文件，约 1,500 行）。Benchmark 脚本调用相同的 GitLab API，测量每个 MCP 实际返回的数据量。

```
任务：读取 7 个源文件，理解 MQ 消费架构
────────────────────────────────────────────────────────────────────
                          新 MCP            旧 MCP (@zereight)
工具调用次数              6                 8
总响应大小                31.5 KB           52.3 KB
响应格式                  纯文本            原始 JSON + base64
批量读取                  ✓                 ✗
────────────────────────────────────────────────────────────────────
节省                      40% 更小          基准线
                          少 2 次调用
```

**节省来自哪里：**

| 步骤 | 本项目 | @zereight/mcp-gitlab | 差异 |
|------|---|---|---|
| 列出项目结构 | 1.5 KB（格式化树） | 13.5 KB（原始 JSON 数组） | **小 89%** |
| 读取 3 个 MQ 文件 | 1 次 `gl_read_multiple` | 3 次 `get_file_contents` | **少 2 次调用** |
| 每个文件响应 | 解码文本 + 行号 | JSON 包装 + base64 + sha256/blob_id 元数据 | **每文件小 ~25%** |

**自己运行 benchmark：**
```bash
cd benchmark
GITLAB_TOKEN=glpat-xxx GITLAB_URL=https://gitlab.example.com go run main.go
```

**等价工具**

| 本项目 | @zereight/mcp-gitlab | 差异 |
|---|---|---|
| `gl_read_file` | `get_file_contents` | 行号、行范围、500 行截断、二进制检测 |
| `gl_read_multiple` | _(无)_ | 批量读取最多 10 个文件 |
| `gl_find_files` | `get_repository_tree` | glob 模式匹配 |
| `gl_search_code` | `search_code` | 按文件去重合并的结果 |
| `gl_read_symbols` | _(无)_ | 大文件只返回签名，省 90% token |
| `gl_diff` | `list_merge_request_diffs` | 文件过滤、排除、截断 |
| `gl_blame` | _(无)_ | 行范围 blame |

**什么时候用哪个？**
- **本项目** → 读取和理解代码
- **@zereight/mcp-gitlab** → 写操作（创建 MR、发评论、管理 issue）

### 贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md)。欢迎 PR！

### 许可证

[MIT](LICENSE)
