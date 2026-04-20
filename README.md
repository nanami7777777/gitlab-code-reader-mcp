# gitlab-code-reader-mcp

A lightweight MCP server for reading GitLab repository code, inspired by Claude Code's Read/Grep/Glob/LSP tools.

## 9 Tools

| Tool | Claude Code Equivalent | Purpose |
|------|----------------------|---------|
| `gl_read_file` | Read | Read file with line numbers, line ranges, smart truncation |
| `gl_read_multiple` | Read (batch) | Read up to 10 files in one call |
| `gl_find_files` | Glob | Find files by glob pattern |
| `gl_search_code` | Grep | Search code content across repo |
| `gl_list_directory` | ls/tree | Browse directory structure |
| `gl_read_symbols` | readCode/LSP | Extract symbols from large files, full content for small files |
| `gl_diff` | git diff | View MR diffs or ref comparisons |
| `gl_blame` | git blame | See who changed what and when |
| `gl_commit_history` | git log | View commit history with stats |

## Quick Start

```bash
# Install
npm install

# Build
npm run build

# Run (requires GITLAB_TOKEN)
GITLAB_TOKEN=glpat-xxx npm start
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITLAB_TOKEN` | Yes | — | GitLab personal access token |
| `GITLAB_URL` | No | `https://gitlab.com` | GitLab instance URL |

## MCP Configuration

Add to your MCP config (e.g. `.kiro/settings/mcp.json`):

```json
{
  "mcpServers": {
    "gitlab-code-reader": {
      "command": "node",
      "args": ["/path/to/gitlab-code-reader-mcp/dist/index.js"],
      "env": {
        "GITLAB_TOKEN": "glpat-xxx",
        "GITLAB_URL": "https://gitlab.example.com"
      }
    }
  }
}
```

## Design Principles (from Claude Code)

1. **Token is budget** — Large files auto-truncate, symbols-first for big files
2. **Caching** — Repository tree and file content cached (5 min TTL)
3. **Line numbers** — All file output includes line numbers for precise referencing
4. **Guided errors** — Errors suggest next steps instead of raw API messages
5. **Batch operations** — `gl_read_multiple` reduces tool call overhead
6. **Smart sizing** — Binary detection, minified line truncation, result limits
