package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
)

func ListDirectory(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_list_directory",
		mcp.WithDescription("List directory contents in a GitLab repository. Use depth=1 for quick browse, depth=2-3 for project structure overview."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path")),
		mcp.WithString("path", mcp.Description("Directory path. Default: repository root")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit SHA")),
		mcp.WithNumber("depth", mcp.Description("Recursion depth. Default: 1, max: 3")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req.Params.Arguments)
		projectID := argStr(args, "project_id")
		basePath := argStr(args, "path")

		ref := argStr(args, "ref")
		if ref == "" {
			r, err := client.DefaultBranch(projectID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
			}
			ref = r
		}

		depth := 1
		if v, ok := argFloat(args, "depth"); ok && v > 0 {
			depth = minInt(int(v), 3)
		}

		recursive := depth > 1
		tree, err := client.GetTree(projectID, basePath, ref, recursive)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
		}

		// filter by depth
		baseDepth := 0
		if basePath != "" {
			baseDepth = strings.Count(basePath, "/") + 1
		}
		var filtered []gitlab.TreeItem
		for _, item := range tree {
			itemDepth := strings.Count(item.Path, "/") + 1 - baseDepth
			if itemDepth <= depth {
				filtered = append(filtered, item)
			}
		}

		maxItems := 200
		limited := filtered
		truncated := false
		if len(filtered) > maxItems {
			limited = filtered[:maxItems]
			truncated = true
		}

		displayPath := basePath
		if displayPath == "" {
			displayPath = "/"
		}
		header := fmt.Sprintf("Directory: %s (ref: %s, %d items)", displayPath, ref, len(filtered))

		var b strings.Builder
		b.WriteString(header + "\n" + strings.Repeat("─", 40) + "\n")
		for _, item := range limited {
			relPath := item.Path
			if basePath != "" && strings.HasPrefix(item.Path, basePath+"/") {
				relPath = item.Path[len(basePath)+1:]
			}
			indentLevel := strings.Count(relPath, "/")
			indent := strings.Repeat("  ", indentLevel)
			icon := "📄"
			if item.Type == "tree" {
				icon = "📁"
			}
			fmt.Fprintf(&b, "%s%s %s\n", indent, icon, item.Name)
		}

		if truncated {
			fmt.Fprintf(&b, "\n⚠️ Showing %d of %d items. Use a specific path to explore subdirectories.", maxItems, len(filtered))
		}
		return mcp.NewToolResultText(b.String()), nil
	}

	return tool, handler
}
