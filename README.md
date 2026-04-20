# gitlab-code-reader-mcp

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A lightweight MCP (Model Context Protocol) server that lets AI assistants read and explore GitLab repository code. Inspired by how [Claude Code](https://code.claude.com) reads local codebases with its Read/Grep/Glob/LSP tools — but for remote GitLab repos.

**9 focused tools. ~800 lines of TypeScript. Zero bloat.**

## Why?

Existing GitLab MCP servers expose 100+ tools covering every GitLab API. That's great for full GitLab automation, but it floods the AI's context with tool descriptions and makes code reading inefficient.

This project takes a different approach: **do one thing well**. Just code reading. The tool descriptions fit in ~500 tokens instead of thousands, and every tool is optimized for how AI assistants actually explore code.

## Tools

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

## Design Principles

Borrowed from Claude Code's architecture:

- **Token is budget** — Large files auto-truncate at 500 lines. Batch reads cap at 200 lines/file. Minified lines get cut at 500 chars.
- **Symbols first** — For files over 300 lines, `gl_read_symbols` returns function/class signatures instead of dumping the whole file. Read specific sections with line ranges.
- **Caching** — LRU cache for repository trees (5 min), file content (5 min), project info (10 min). Same commit = same content, no need to re-fetch.
- **Line numbers everywhere** — All file output includes line numbers so the AI can reference "line 42" and use `start_line`/`end_line` to zoom in.
- **Guided errors** — Instead of raw 404s, errors suggest what to try next.
- **Built-in strategy** — The MCP `instructions` field teaches the AI when to use which tool.

## Quick Start

### Install

```bash
git clone https://github.com/nanami7777777/gitlab-code-reader-mcp.git
cd gitlab-code-reader-mcp
go build -o server ./cmd/server/
```

### Run

```bash
GITLAB_TOKEN=glpat-xxx ./server
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITLAB_TOKEN` | Yes | — | GitLab personal access token (needs `read_api` scope) |
| `GITLAB_URL` | No | `https://gitlab.com` | GitLab instance URL |

## MCP Client Configuration

### Kiro

Add to `.kiro/settings/mcp.json`:

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

Add to `.claude/settings.json`:

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

Add to `.cursor/mcp.json` with the same format as above.

## Usage Examples

Once connected, just ask your AI assistant naturally:

```
"Show me the directory structure of project mygroup/myproject"
→ AI calls gl_list_directory

"Find all TypeScript files in the src directory"
→ AI calls gl_find_files with pattern **/*.ts

"Read the main entry point"
→ AI calls gl_read_file on src/index.ts

"Search for where authentication is handled"
→ AI calls gl_search_code with query "authenticate"

"Show me what changed in MR !42"
→ AI calls gl_diff with merge_request_iid: 42

"Who last modified the config file?"
→ AI calls gl_blame on config.ts
```

## Project ID

Tools accept `project_id` as either:
- **Numeric ID**: `609`
- **Path**: `mygroup/myproject`

Both formats work. Find your project ID on the GitLab project settings page or via the API.

## Symbol Extraction

`gl_read_symbols` uses regex-based extraction (no external dependencies). Supported languages:

| Language | Detected Symbols |
|----------|-----------------|
| TypeScript/JavaScript | functions, classes, interfaces, types, enums, methods, arrow functions |
| Python | functions, classes |
| Go | functions, structs, interfaces |
| Java/Kotlin | classes, interfaces, methods |
| Rust | functions, structs, traits, enums, impl blocks |

For files under 300 lines, it returns the full content instead (same behavior as Claude Code's `readCode`).

## Architecture

```
cmd/server/main.go          # MCP server entry point + tool registration
internal/
├── gitlab/
│   ├── client.go           # GitLab API client with caching
│   ├── cache.go            # LRU cache implementation
│   └── types.go            # API response types
├── tools/
│   ├── helpers.go          # Line numbers, truncation, binary detection
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
    └── extract.go          # Regex-based symbol extraction
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). PRs welcome!

## License

[MIT](LICENSE)
