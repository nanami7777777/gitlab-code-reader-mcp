package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
)

func ReadFile(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_read_file",
		mcp.WithDescription("Read a file from a GitLab repository with line numbers. Supports line ranges for large files. Like Claude Code's Read tool but for remote repos."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path (e.g. 'mygroup/myproject')")),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the file in the repository")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit SHA. Defaults to the project's default branch")),
		mcp.WithNumber("start_line", mcp.Description("Start reading from this line number (1-based). Default: 1")),
		mcp.WithNumber("end_line", mcp.Description("Stop reading at this line number (inclusive). Default: end of file")),
		mcp.WithNumber("max_lines", mcp.Description("Maximum number of lines to return. Default: 500")),
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

		maxLines := argInt(args, "max_lines", "", 500)

		file, err := client.GetFileContent(projectID, filePath, ref)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
		}

		if isBinary(file.Content) {
			return mcp.NewToolResultText(fmt.Sprintf("Binary file: %s (%s)\nCannot display binary content.", filePath, formatSize(file.Size))), nil
		}

		content, err := decodeBase64(file.Content)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Decode error: %v", err)), nil
		}

		allLines := strings.Split(content, "\n")
		totalLines := len(allLines)

		startLine := argInt(args, "start_line", "", 1)
		endLine := argInt(args, "end_line", "", totalLines)

		startLine = maxInt(1, startLine)
		endLine = minInt(totalLines, endLine)
		selected := allLines[startLine-1 : endLine]

		truncated := false
		if len(selected) > maxLines {
			selected = selected[:maxLines]
			truncated = true
		}

		for i, l := range selected {
			selected[i] = truncateLine(l, 500)
		}

		output := addLineNumbers(strings.Join(selected, "\n"), startLine)
		header := fmt.Sprintf("File: %s (%s, %d lines, ref: %s)", filePath, formatSize(file.Size), totalLines, ref)
		rangeInfo := fmt.Sprintf("Showing lines %d-%d of %d", startLine, startLine+len(selected)-1, totalLines)

		result := fmt.Sprintf("%s\n%s\n%s\n%s", header, rangeInfo, strings.Repeat("─", 60), output)
		if truncated {
			result += fmt.Sprintf("\n%s\n⚠️ Output truncated at %d lines. Use start_line/end_line to read remaining content.", strings.Repeat("─", 60), maxLines)
		}
		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}
