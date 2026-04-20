package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/gobwas/glob"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
)

func Diff(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_diff",
		mcp.WithDescription("View code changes between refs or in a merge request. Supports file filtering and exclusion patterns."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path")),
		mcp.WithString("from_ref", mcp.Description("Base ref to compare from (branch/tag/SHA)")),
		mcp.WithString("to_ref", mcp.Description("Target ref to compare to")),
		mcp.WithNumber("merge_request_iid", mcp.Description("MR IID — alternative to from_ref/to_ref")),
		mcp.WithString("file_pattern", mcp.Description("Only show files matching this glob pattern")),
		mcp.WithArray("exclude_patterns", mcp.Description("Exclude files matching these patterns (e.g. ['*.lock', 'dist/**'])")),
		mcp.WithNumber("max_files", mcp.Description("Max files to show. Default: 20")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req.Params.Arguments)
		projectID := argStr(args, "project_id")
		maxFiles := argInt(args, "max_files", "", 20)

		var diffs []gitlab.DiffFile
		var label string

		if mrIID, ok := argFloat(args, "merge_request_iid"); ok && mrIID > 0 {
			d, err := client.GetMRDiffs(projectID, int(mrIID))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
			}
			diffs = d
			label = fmt.Sprintf("MR !%d", int(mrIID))
		} else {
			fromRef := argStr(args, "from_ref")
			toRef := argStr(args, "to_ref")
			if fromRef == "" || toRef == "" {
				return mcp.NewToolResultError("Error: Provide either merge_request_iid or both from_ref and to_ref."), nil
			}
			result, err := client.Compare(projectID, fromRef, toRef)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
			}
			diffs = result.Diffs
			label = fmt.Sprintf("%s...%s", fromRef, toRef)
		}

		// file_pattern filter
		fp := argStr(args, "file_pattern")
		if fp != "" {
			g, err := glob.Compile(fp, '/')
			if err == nil {
				var filtered []gitlab.DiffFile
				for _, d := range diffs {
					if g.Match(d.NewPath) || g.Match(d.OldPath) {
						filtered = append(filtered, d)
					}
				}
				diffs = filtered
			}
		}

		// exclude_patterns filter
		if ep, ok := args["exclude_patterns"].([]any); ok && len(ep) > 0 {
			for _, p := range ep {
				ps, ok := p.(string)
				if !ok {
					continue
				}
				g, err := glob.Compile(ps, '/')
				if err != nil {
					continue
				}
				var filtered []gitlab.DiffFile
				for _, d := range diffs {
					if !g.Match(d.NewPath) && !g.Match(d.OldPath) {
						filtered = append(filtered, d)
					}
				}
				diffs = filtered
			}
		}

		totalFiles := len(diffs)
		limited := diffs
		if len(diffs) > maxFiles {
			limited = diffs[:maxFiles]
		}

		var b strings.Builder
		fmt.Fprintf(&b, "Diff: %s (%d files changed)\n", label, totalFiles)

		for _, d := range limited {
			status := "modified"
			if d.NewFile {
				status = "added"
			} else if d.DeletedFile {
				status = "deleted"
			} else if d.RenamedFile {
				status = fmt.Sprintf("renamed: %s → %s", d.OldPath, d.NewPath)
			}

			diffContent := d.Diff
			if len(diffContent) > 3000 {
				diffContent = diffContent[:3000] + "\n... [diff truncated, use gl_read_file to see full content]"
			}
			fmt.Fprintf(&b, "\n📄 %s (%s)\n%s\n%s\n", d.NewPath, status, strings.Repeat("─", 40), diffContent)
		}

		if totalFiles > maxFiles {
			fmt.Fprintf(&b, "\n⚠️ Showing %d of %d changed files. Use file_pattern or exclude_patterns to filter.", maxFiles, totalFiles)
		}
		return mcp.NewToolResultText(b.String()), nil
	}

	return tool, handler
}
