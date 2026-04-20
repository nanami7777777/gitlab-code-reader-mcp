package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/tools"
)

const instructions = `You have access to tools for reading and exploring GitLab repository code.

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
`

func main() {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: GITLAB_TOKEN environment variable is required.")
		os.Exit(1)
	}
	baseURL := os.Getenv("GITLAB_URL")
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	client := gitlab.NewClient(baseURL, token)

	s := server.NewMCPServer(
		"gitlab-code-reader",
		"0.2.0",
		server.WithInstructions(instructions),
	)

	// Register all 9 tools
	t1, h1 := tools.ReadFile(client)
	s.AddTool(t1, h1)

	t2, h2 := tools.ReadMultiple(client)
	s.AddTool(t2, h2)

	t3, h3 := tools.FindFiles(client)
	s.AddTool(t3, h3)

	t4, h4 := tools.SearchCode(client)
	s.AddTool(t4, h4)

	t5, h5 := tools.ListDirectory(client)
	s.AddTool(t5, h5)

	t6, h6 := tools.ReadSymbols(client)
	s.AddTool(t6, h6)

	t7, h7 := tools.Diff(client)
	s.AddTool(t7, h7)

	t8, h8 := tools.Blame(client)
	s.AddTool(t8, h8)

	t9, h9 := tools.CommitHistory(client)
	s.AddTool(t9, h9)

	fmt.Fprintln(os.Stderr, "gitlab-code-reader MCP server running on stdio")
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
		os.Exit(1)
	}
}
