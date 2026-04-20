package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
)

func SearchCode(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_search_code",
		mcp.WithDescription("Search for code content across a GitLab repository. Like Claude Code's Grep tool. Returns matching lines with context."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path")),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query (keywords or code pattern)")),
		mcp.WithString("ref", mcp.Description("Branch to search in")),
		mcp.WithString("file_pattern", mcp.Description("Filter results to files matching this pattern (e.g. '*.ts')")),
		mcp.WithNumber("max_results", mcp.Description("Maximum results. Default: 20")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req.Params.Arguments)
		projectID := argStr(args, "project_id")
		query := argStr(args, "query")
		ref := argStr(args, "ref")
		filePattern := argStr(args, "file_pattern")
		maxResults := argInt(args, "max_results", "", 20)

		blobs, err := client.SearchCode(projectID, query, ref)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
		}

		// filter by file pattern
		if filePattern != "" {
			ext := strings.TrimPrefix(filePattern, "*")
			var filtered []gitlab.SearchBlob
			for _, b := range blobs {
				if strings.HasSuffix(b.Filename, ext) || strings.HasSuffix(b.Path, ext) {
					filtered = append(filtered, b)
				}
			}
			blobs = filtered
		}

		if len(blobs) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf(
				"No results found for \"%s\" in project %s.\n\nSuggestions:\n- Try different keywords\n- Check spelling\n- Use gl_find_files to locate files by name pattern instead",
				query, projectID,
			)), nil
		}

		limited := blobs
		if len(blobs) > maxResults {
			limited = blobs[:maxResults]
		}

		var b strings.Builder
		refInfo := ""
		if ref != "" {
			refInfo = fmt.Sprintf(" (ref: %s)", ref)
		}
		fmt.Fprintf(&b, "Found %d result(s) for \"%s\"%s\n", len(blobs), query, refInfo)

		for _, blob := range limited {
			fmt.Fprintf(&b, "\n📄 %s:%d\n", blob.Path, blob.Startline)
			lines := strings.Split(blob.Data, "\n")
			for i, line := range lines {
				fmt.Fprintf(&b, "  %5d\t%s\n", blob.Startline+i, line)
			}
		}

		if len(blobs) > maxResults {
			fmt.Fprintf(&b, "\n⚠️ Showing %d of %d results. Refine your query for more specific results.", maxResults, len(blobs))
		}
		return mcp.NewToolResultText(b.String()), nil
	}

	return tool, handler
}
