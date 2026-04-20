package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/symbols"
)

func ReadSymbols(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_read_symbols",
		mcp.WithDescription("Extract code symbols (functions, classes, interfaces) from a file. For small files (<300 lines) returns full content. For large files returns signatures with line numbers. Like Claude Code's readCode tool."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path")),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the file")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit SHA")),
		mcp.WithString("symbol_filter", mcp.Description("Filter symbols by name (case-insensitive substring match)")),
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

		file, err := client.GetFileContent(projectID, filePath, ref)
		if err != nil {
			return mcp.NewToolResultError(guidedError(err, "read_symbols", args)), nil
		}

		if isBinary(file.Content) {
			return mcp.NewToolResultText(fmt.Sprintf("Binary file: %s (%s)", filePath, formatSize(file.Size))), nil
		}

		content, err := decodeBase64(file.Content)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Decode error: %v", err)), nil
		}

		lines := strings.Split(content, "\n")
		totalLines := len(lines)
		header := fmt.Sprintf("File: %s (%s, %d lines, ref: %s)", filePath, formatSize(file.Size), totalLines, ref)

		// Small file: return full content
		if totalLines <= 300 {
			for i, l := range lines {
				lines[i] = truncateLine(l, 500)
			}
			numbered := addLineNumbers(strings.Join(lines, "\n"), 1)
			return mcp.NewToolResultText(fmt.Sprintf("%s\n(small file — returning full content)\n%s\n%s", header, strings.Repeat("─", 60), numbered)), nil
		}

		// Large file: extract symbols
		syms := symbols.Extract(content, filePath)

		filter := argStr(args, "symbol_filter")
		if filter != "" {
			filterLower := strings.ToLower(filter)
			var filtered []symbols.Symbol
			for _, s := range syms {
				if strings.Contains(strings.ToLower(s.Name), filterLower) {
					filtered = append(filtered, s)
				}
			}
			syms = filtered
		}

		output := symbols.Format(syms)
		return mcp.NewToolResultText(fmt.Sprintf("%s\n(large file — showing %d symbol signatures)\n%s\n\n💡 Use gl_read_file with start_line/end_line to read specific implementations.", header, len(syms), output)), nil
	}

	return tool, handler
}
