package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/tools"
)

const instructions = `You have access to 9 read-only tools for exploring GitLab repository code. These tools NEVER modify anything — explore freely.

## RULES (read these first)

### What to use
- gl_read_file for reading files (NOT curl, NOT API calls)
- gl_find_files for finding files by name pattern (NOT gl_list_directory with grep)
- gl_search_code for searching code content (NOT reading files and scanning manually)
- gl_read_multiple for reading 2+ files (NOT multiple gl_read_file calls)
- gl_read_symbols for understanding large files (NOT reading the entire file)

### What NOT to do
- Do NOT read an entire large file to find one function — use gl_read_symbols first, then gl_read_file with start_line/end_line
- Do NOT call gl_read_file multiple times when gl_read_multiple can batch them
- Do NOT browse directories to find a file — use gl_find_files with a glob pattern
- Do NOT guess file paths — use gl_find_files or gl_list_directory to discover them first

### Parallel calls (MANDATORY)
When you need multiple independent pieces of information, call tools in parallel in the same message. Examples:
- Reading README.md AND listing directory → call both in one message
- Reading 3 specific files → use gl_read_multiple, not 3 separate calls

## TOOL SELECTION (strict priority order)

### First contact with a project
1. gl_list_directory(depth=2) → project structure
2. gl_read_file("README.md") → project purpose
3. gl_find_files with config patterns → tech stack

### Finding code
1. Know exact path → gl_read_file
2. Know filename pattern → gl_find_files → gl_read_file
3. Know content/keyword → gl_search_code
4. Large file (300+ lines) → gl_read_symbols first → gl_read_file with line range

### Code review
1. gl_diff(merge_request_iid) → overall changes
2. gl_read_multiple → full context of changed files (batch!)
3. gl_blame → history of changed areas
4. gl_search_code → find related references

### When to STOP searching
- Found the answer? Stop. Don't keep exploring "just in case."
- Got enough context? Stop. You don't need to read every related file.
- File is 300+ lines? Use gl_read_symbols, don't read the whole thing.
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
