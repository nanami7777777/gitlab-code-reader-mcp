package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nanami7777777/gitlab-code-reader-mcp/internal/gitlab"
)

func ReadMultiple(client *gitlab.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("gl_read_multiple",
		mcp.WithDescription("Read multiple files in one call (max 10). More efficient than multiple gl_read_file calls. Use for batch reading."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("GitLab project ID or path")),
		mcp.WithArray("files", mcp.Required(), mcp.Description("Array of files to read (max 10). Each item: {file_path, start_line?, end_line?}")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit SHA")),
		mcp.WithNumber("max_lines_per_file", mcp.Description("Max lines per file. Default: 200")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req.Params.Arguments)
		projectID := argStr(args, "project_id")
		ref := argStr(args, "ref")
		if ref == "" {
			r, err := client.DefaultBranch(projectID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("❌ Error: %v", err)), nil
			}
			ref = r
		}

		maxPerFile := argInt(args, "max_lines_per_file", "", 200)

		filesRaw, ok := args["files"].([]any)
		if !ok || len(filesRaw) == 0 {
			return mcp.NewToolResultError("❌ files parameter is required and must be a non-empty array"), nil
		}
		if len(filesRaw) > 10 {
			filesRaw = filesRaw[:10]
		}

		type fileReq struct {
			FilePath  string
			StartLine int
			EndLine   int
		}
		var reqs []fileReq
		for _, f := range filesRaw {
			m, ok := f.(map[string]any)
			if !ok {
				continue
			}
			fp, _ := m["file_path"].(string)
			if fp == "" {
				continue
			}
			fr := fileReq{FilePath: fp}
			if v, ok := m["start_line"].(float64); ok {
				fr.StartLine = int(v)
			}
			if v, ok := m["end_line"].(float64); ok {
				fr.EndLine = int(v)
			}
			reqs = append(reqs, fr)
		}

		results := make([]string, len(reqs))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 5) // concurrency limit

		for i, fr := range reqs {
			wg.Add(1)
			go func(idx int, fr fileReq) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				innerArgs := map[string]any{
					"project_id": projectID,
					"file_path":  fr.FilePath,
					"ref":        ref,
					"max_lines":  float64(maxPerFile),
				}
				if fr.StartLine > 0 {
					innerArgs["start_line"] = float64(fr.StartLine)
				}
				if fr.EndLine > 0 {
					innerArgs["end_line"] = float64(fr.EndLine)
				}

				innerReq := mcp.CallToolRequest{}
				innerReq.Params.Arguments = innerArgs
				_, readHandler := ReadFile(client)
				res, err := readHandler(ctx, innerReq)
				if err != nil {
					results[idx] = fmt.Sprintf("❌ Error reading %s: %v", fr.FilePath, err)
					return
				}
				if len(res.Content) > 0 {
					if tc, ok := res.Content[0].(mcp.TextContent); ok {
						results[idx] = tc.Text
						return
					}
				}
				results[idx] = fmt.Sprintf("❌ Error reading %s: empty response", fr.FilePath)
			}(i, fr)
		}
		wg.Wait()

		return mcp.NewToolResultText(strings.Join(results, "\n\n"+strings.Repeat("═", 60)+"\n\n")), nil
	}

	return tool, handler
}
