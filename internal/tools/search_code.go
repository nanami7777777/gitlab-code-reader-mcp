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
			return mcp.NewToolResultError(guidedError(err, "search_code", args)), nil
		}

		// Deduplicate: merge multiple matches from the same file
		type fileMatch struct {
			path    string
			entries []gitlab.SearchBlob
		}
		seen := map[string]int{} // path -> index in deduped
		var deduped []fileMatch
		for _, blob := range blobs {
			if idx, ok := seen[blob.Path]; ok {
				deduped[idx].entries = append(deduped[idx].entries, blob)
			} else {
				seen[blob.Path] = len(deduped)
				deduped = append(deduped, fileMatch{path: blob.Path, entries: []gitlab.SearchBlob{blob}})
			}
		}

		// filter by file pattern
		if filePattern != "" {
			ext := strings.TrimPrefix(filePattern, "*")
			var filtered []fileMatch
			for _, fm := range deduped {
				if strings.HasSuffix(fm.path, ext) {
					filtered = append(filtered, fm)
				}
			}
			deduped = filtered
		}

		totalMatches := 0
		for _, fm := range deduped {
			totalMatches += len(fm.entries)
		}

		if len(deduped) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf(
				"No results found for \"%s\" in project %s.\n\n"+
					"Suggestions:\n"+
					"  → Try different keywords or check spelling\n"+
					"  → gl_find_files(pattern: \"**/*%s*\") to search by filename\n"+
					"  → gl_list_directory(depth: 2) to browse the project structure",
				query, projectID, query,
			)), nil
		}

		limited := deduped
		if len(deduped) > maxResults {
			limited = deduped[:maxResults]
		}

		var b strings.Builder
		refInfo := ""
		if ref != "" {
			refInfo = fmt.Sprintf(" (ref: %s)", ref)
		}
		fmt.Fprintf(&b, "Found %d match(es) across %d file(s) for \"%s\"%s\n", totalMatches, len(deduped), query, refInfo)

		for _, fm := range limited {
			fmt.Fprintf(&b, "\n📄 %s", fm.path)
			if len(fm.entries) > 1 {
				fmt.Fprintf(&b, " (%d matches)", len(fm.entries))
			}
			b.WriteString("\n")
			for _, blob := range fm.entries {
				lines := strings.Split(blob.Data, "\n")
				for i, line := range lines {
					fmt.Fprintf(&b, "  %5d\t%s\n", blob.Startline+i, line)
				}
			}
		}

		if len(deduped) > maxResults {
			fmt.Fprintf(&b, "\n⚠️ Showing %d of %d files. Refine your query for more specific results.", maxResults, len(deduped))
		}
		return mcp.NewToolResultText(b.String()), nil
	}

	return tool, handler
}
