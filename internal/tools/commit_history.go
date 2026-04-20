package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
)

func CommitHistory(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_commit_history",
		mcp.WithDescription("View commit history for a file, directory, or entire project. Includes stats (additions/deletions)."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path")),
		mcp.WithString("path", mcp.Description("Limit to commits affecting this file/directory")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit SHA")),
		mcp.WithNumber("max_count", mcp.Description("Max commits to return. Default: 20")),
		mcp.WithString("since", mcp.Description("Only commits after this date (ISO 8601)")),
		mcp.WithString("author", mcp.Description("Filter by author name or email")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req.Params.Arguments)
		projectID := argStr(args, "project_id")

		opts := map[string]string{}
		if v := argStr(args, "ref"); v != "" {
			opts["ref_name"] = v
		}
		if v := argStr(args, "path"); v != "" {
			opts["path"] = v
		}
		if v := argStr(args, "since"); v != "" {
			opts["since"] = v
		}
		if v := argStr(args, "author"); v != "" {
			opts["author"] = v
		}
		if v, ok := argFloat(args, "max_count"); ok && v > 0 {
			opts["per_page"] = strconv.Itoa(int(v))
		}

		commits, err := client.ListCommits(projectID, opts)
		if err != nil {
			return mcp.NewToolResultError(guidedError(err, "commit_history", args)), nil
		}

		if len(commits) == 0 {
			pathInfo := ""
			if p := argStr(args, "path"); p != "" {
				pathInfo = " for " + p
			}
			return mcp.NewToolResultText(fmt.Sprintf("No commits found%s.", pathInfo)), nil
		}

		pathInfo := ""
		if p := argStr(args, "path"); p != "" {
			pathInfo = " for " + p
		}
		header := fmt.Sprintf("Commit history%s (%d commits)", pathInfo, len(commits))

		var b strings.Builder
		b.WriteString(header + "\n" + strings.Repeat("─", 70) + "\n")
		for _, c := range commits {
			date := c.CommittedDate
			if len(date) >= 10 {
				date = date[:10]
			}
			stats := ""
			if c.Stats != nil {
				stats = fmt.Sprintf(" (+%d -%d)", c.Stats.Additions, c.Stats.Deletions)
			}
			fmt.Fprintf(&b, "  %s │ %s │ %s │ %s%s\n", c.ShortID, date, c.AuthorName, c.Title, stats)
		}
		return mcp.NewToolResultText(b.String()), nil
	}

	return tool, handler
}
