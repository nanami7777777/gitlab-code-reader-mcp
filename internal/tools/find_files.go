package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gobwas/glob"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
)

func FindFiles(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_find_files",
		mcp.WithDescription("Find files by glob pattern in a GitLab repository. Like Claude Code's Glob tool. Use for file discovery before reading."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path")),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Glob pattern to match files (e.g. '**/*.ts', 'src/components/**/*.tsx')")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit SHA")),
		mcp.WithString("path", mcp.Description("Directory to search in. Default: repository root")),
		mcp.WithNumber("max_results", mcp.Description("Maximum results to return. Default: 50, max: 200")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req.Params.Arguments)
		projectID := argStr(args, "project_id")
		pattern := argStr(args, "pattern")

		ref := argStr(args, "ref")
		if ref == "" {
			r, err := client.DefaultBranch(projectID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
			}
			ref = r
		}

		basePath := argStr(args, "path")
		maxResults := argInt(args, "max_results", "", 50)
		if maxResults > 200 {
			maxResults = 200
		}

		tree, err := client.GetTree(projectID, basePath, ref, true)
		if err != nil {
			return mcp.NewToolResultError(guidedError(err, "find_files", args)), nil
		}

		g, err := glob.Compile(pattern, '/')
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Invalid glob pattern: %v\n\nExamples: **/*.go, src/**/*.ts, *.config.{js,json}", err)), nil
		}

		var matched []string
		for _, item := range tree {
			if item.Type != "blob" {
				continue
			}
			relPath := item.Path
			if basePath != "" && strings.HasPrefix(item.Path, basePath+"/") {
				relPath = item.Path[len(basePath)+1:]
			}
			if g.Match(relPath) || g.Match(item.Path) {
				matched = append(matched, item.Path)
			}
		}

		// Sort by path depth (shallow first), then alphabetically
		sort.Slice(matched, func(i, j int) bool {
			di := strings.Count(matched[i], "/")
			dj := strings.Count(matched[j], "/")
			if di != dj {
				return di < dj
			}
			return matched[i] < matched[j]
		})

		limited := matched
		truncated := false
		if len(matched) > maxResults {
			limited = matched[:maxResults]
			truncated = true
		}

		header := fmt.Sprintf("Found %d file(s) matching \"%s\" (ref: %s)", len(matched), pattern, ref)
		var b strings.Builder
		b.WriteString(header + "\n")
		for _, p := range limited {
			fmt.Fprintf(&b, "  %s\n", p)
		}
		if truncated {
			fmt.Fprintf(&b, "\n⚠️ Showing %d of %d matches. Use a more specific pattern to narrow results.", maxResults, len(matched))
		}
		if len(matched) == 0 {
			b.WriteString("\nNo files found. Try a broader pattern or check the path/ref.")
		}
		return mcp.NewToolResultText(b.String()), nil
	}

	return tool, handler
}
