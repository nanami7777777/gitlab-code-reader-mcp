package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
)

func Blame(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_blame",
		mcp.WithDescription("View git blame for a file — who changed each line and when. Supports line ranges."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path")),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the file")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit SHA")),
		mcp.WithNumber("start_line", mcp.Description("Start line (1-based)")),
		mcp.WithNumber("end_line", mcp.Description("End line (inclusive)")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req.Params.Arguments)
		projectID := argStr(args, "project_id")
		filePath := argStr(args, "file_path")

		ref := argStr(args, "ref")
		if ref == "" {
			r, err := client.DefaultBranch(projectID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
			}
			ref = r
		}

		startLine := 0
		if v, ok := argFloat(args, "start_line"); ok {
			startLine = int(v)
		}
		endLine := 0
		if v, ok := argFloat(args, "end_line"); ok {
			endLine = int(v)
		}

		ranges, err := client.GetBlame(projectID, filePath, ref)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
		}

		header := fmt.Sprintf("Blame: %s (ref: %s)", filePath, ref)
		var b strings.Builder
		b.WriteString(header + "\n" + strings.Repeat("─", 80) + "\n")

		lineNum := 1
		for _, r := range ranges {
			for _, line := range r.Lines {
				if startLine > 0 && lineNum < startLine {
					lineNum++
					continue
				}
				if endLine > 0 && lineNum > endLine {
					break
				}
				date := r.Commit.AuthoredDate
				if len(date) >= 10 {
					date = date[:10]
				}
				author := r.Commit.AuthorName
				if len(author) > 15 {
					author = author[:15]
				}
				sha := r.Commit.ID
				if len(sha) > 8 {
					sha = sha[:8]
				}
				fmt.Fprintf(&b, "%5d │ %s │ %s │ %-15s │ %s\n", lineNum, sha, date, author, line)
				lineNum++
			}
			if endLine > 0 && lineNum > endLine {
				break
			}
		}

		return mcp.NewToolResultText(b.String()), nil
	}

	return tool, handler
}
