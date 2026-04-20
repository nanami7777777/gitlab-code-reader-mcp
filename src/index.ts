#!/usr/bin/env node
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { GitLabClient } from "./gitlab/client.js";
import { readFileSchema, readFile } from "./tools/read-file.js";
import { readMultipleSchema, readMultiple } from "./tools/read-multiple.js";
import { findFilesSchema, findFiles } from "./tools/find-files.js";
import { searchCodeSchema, searchCode } from "./tools/search-code.js";
import { listDirectorySchema, listDirectory } from "./tools/list-directory.js";
import { readSymbolsSchema, readSymbols } from "./tools/read-symbols.js";
import { diffSchema, diff } from "./tools/diff.js";
import { blameSchema, blame } from "./tools/blame.js";
import { commitHistorySchema, commitHistory } from "./tools/commit-history.js";

// --- Config ---
const GITLAB_URL = process.env.GITLAB_URL ?? "https://gitlab.com";
const GITLAB_TOKEN = process.env.GITLAB_TOKEN ?? "";

if (!GITLAB_TOKEN) {
  console.error("Error: GITLAB_TOKEN environment variable is required.");
  process.exit(1);
}

const client = new GitLabClient({ baseUrl: GITLAB_URL, token: GITLAB_TOKEN });

// --- MCP Server ---
const server = new McpServer({
  name: "gitlab-code-reader",
  version: "0.1.0",
}, {
  instructions: `You have access to tools for reading and exploring GitLab repository code.

## Tool Selection Strategy (follow this priority order)

### First contact with a project
1. gl_list_directory(depth=2) → understand project structure
2. gl_read_file("README.md") → understand project purpose
3. gl_find_files("**/package.json") or similar config → understand tech stack

### Finding specific code
1. Know the file path → gl_read_file directly
2. Know the file name pattern → gl_find_files to locate, then gl_read_file
3. Know code content/keywords → gl_search_code
4. Need to understand a large file → gl_read_symbols first, then gl_read_file with line range

### Code review workflow
1. gl_diff(merge_request_iid) → see overall changes
2. gl_read_file → read full context of changed files
3. gl_blame → understand history of changed areas
4. gl_search_code → find related references and impact

### Batch operations
- Need 2+ files → use gl_read_multiple instead of multiple gl_read_file calls
- Diff has too many files → use exclude_patterns to filter noise (e.g. ["*.lock", "dist/**"])
`,
});

// --- Register Tools ---

function wrapHandler<T>(fn: (client: GitLabClient, input: T) => Promise<string>) {
  return async (input: T) => {
    try {
      const result = await fn(client, input);
      return { content: [{ type: "text" as const, text: result }] };
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      return { content: [{ type: "text" as const, text: `❌ Error: ${msg}` }], isError: true };
    }
  };
}

server.tool(
  "gl_read_file",
  "Read a file from a GitLab repository with line numbers. Supports line ranges for large files. Like Claude Code's Read tool but for remote repos.",
  readFileSchema.shape,
  wrapHandler(readFile),
);

server.tool(
  "gl_read_multiple",
  "Read multiple files in one call (max 10). More efficient than multiple gl_read_file calls. Use for batch reading.",
  readMultipleSchema.shape,
  wrapHandler(readMultiple),
);

server.tool(
  "gl_find_files",
  "Find files by glob pattern in a GitLab repository. Like Claude Code's Glob tool. Use for file discovery before reading.",
  findFilesSchema.shape,
  wrapHandler(findFiles),
);

server.tool(
  "gl_search_code",
  "Search for code content across a GitLab repository. Like Claude Code's Grep tool. Returns matching lines with context.",
  searchCodeSchema.shape,
  wrapHandler(searchCode),
);

server.tool(
  "gl_list_directory",
  "List directory contents in a GitLab repository. Use depth=1 for quick browse, depth=2-3 for project structure overview.",
  listDirectorySchema.shape,
  wrapHandler(listDirectory),
);

server.tool(
  "gl_read_symbols",
  "Extract code symbols (functions, classes, interfaces) from a file. For small files (<300 lines) returns full content. For large files returns signatures with line numbers. Like Claude Code's readCode tool.",
  readSymbolsSchema.shape,
  wrapHandler(readSymbols),
);

server.tool(
  "gl_diff",
  "View code changes between refs or in a merge request. Supports file filtering and exclusion patterns.",
  diffSchema.shape,
  wrapHandler(diff),
);

server.tool(
  "gl_blame",
  "View git blame for a file — who changed each line and when. Supports line ranges.",
  blameSchema.shape,
  wrapHandler(blame),
);

server.tool(
  "gl_commit_history",
  "View commit history for a file, directory, or entire project. Includes stats (additions/deletions).",
  commitHistorySchema.shape,
  wrapHandler(commitHistory),
);

// --- Start ---
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error("gitlab-code-reader MCP server running on stdio");
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});
