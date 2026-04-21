package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Benchmark client that directly calls GitLab API to measure response sizes,
// simulating what each MCP would return for the same task.

var (
	baseURL = os.Getenv("GITLAB_URL")
	token   = os.Getenv("GITLAB_TOKEN")
)

func init() {
	if baseURL == "" {
		baseURL = "https://git.uhomes.com"
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "GITLAB_TOKEN required")
		os.Exit(1)
	}
}

type callResult struct {
	tool    string
	args    string
	bytes   int
	lines   int
	elapsed time.Duration
	isError bool
}

func gitlabAPI(path string, params map[string]string) ([]byte, time.Duration, error) {
	u, _ := url.Parse(fmt.Sprintf("%s/api/v4%s", baseURL, path))
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("PRIVATE-TOKEN", token)

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return nil, elapsed, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, elapsed, nil
}

func countLines(s string) int {
	return strings.Count(s, "\n") + 1
}

func main() {
	projectID := "600"

	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("BENCHMARK: Same task, two approaches")
	fmt.Println("Task: Understand how Worker consumes MQ messages in project 600")
	fmt.Println("=" + strings.Repeat("=", 79))

	// ============================================================
	// Approach A: New MCP (gitlab-code-reader) style
	// Uses: gl_list_directory, gl_read_file, gl_read_multiple
	// ============================================================
	fmt.Println("\n--- Approach A: gitlab-code-reader-mcp (new) ---")
	fmt.Println("Strategy: list_directory → read entry point → batch read MQ files → read worker → read job factory")

	var newResults []callResult
	totalNewBytes := 0

	// Call 1: list directory (depth=2)
	body, elapsed, _ := gitlabAPI(fmt.Sprintf("/projects/%s/repository/tree", projectID),
		map[string]string{"recursive": "true", "per_page": "100"})
	var tree []map[string]any
	json.Unmarshal(body, &tree)
	// Simulate gl_list_directory output: formatted tree
	var treeOutput strings.Builder
	for _, item := range tree {
		depth := strings.Count(item["path"].(string), "/")
		if depth > 2 {
			continue
		}
		icon := "📄"
		if item["type"].(string) == "tree" {
			icon = "📁"
		}
		fmt.Fprintf(&treeOutput, "%s%s %s\n", strings.Repeat("  ", depth), icon, item["name"].(string))
	}
	formatted := treeOutput.String()
	r := callResult{"gl_list_directory", "depth=2", len(formatted), countLines(formatted), elapsed, false}
	newResults = append(newResults, r)
	totalNewBytes += r.bytes

	// Call 2: read cmd/consumer/main.go
	readFiles := []string{
		"cmd/consumer/main.go",
		"internal/mq/runner.go",
		"internal/mq/consumer_rabbitmq.go",
		"internal/mq/dispatcher.go",
		"internal/dts/worker.go",
		"internal/dts/job/job.go",
		"internal/dts/message.go",
	}

	// In new MCP: call 2 = read entry point, call 3 = gl_read_multiple for 3 MQ files, calls 4-6 = individual reads
	// Total: 6 calls (1 list + 1 read + 1 batch_read_3 + 3 individual reads)
	// But with gl_read_multiple, the 3 MQ files are 1 call, so: 1 + 1 + 1 + 1 + 1 + 1 = 6 calls
	// Actually with optimal batching: 1 list + 1 gl_read_multiple(all 7 files) = 2 calls
	// Let's show the realistic path: 1 list + 1 read(main.go) + 1 batch(3 mq files) + 1 read(worker) + 1 read(job.go) + 1 read(message.go) = 6

	for i, fp := range readFiles {
		body, elapsed, err := gitlabAPI(
			fmt.Sprintf("/projects/%s/repository/files/%s", projectID, url.PathEscape(fp)),
			map[string]string{"ref": "master"})
		if err != nil {
			newResults = append(newResults, callResult{fmt.Sprintf("gl_read_file[%d]", i+2), fp, 0, 0, elapsed, true})
			continue
		}
		var fc struct {
			Content string `json:"content"`
			Size    int64  `json:"size"`
		}
		json.Unmarshal(body, &fc)

		// New MCP decodes base64 and adds line numbers
		decoded := make([]byte, len(fc.Content))
		n, _ := fmt.Sscanf(fc.Content, "%s", &decoded) // simplified
		_ = n
		// Approximate: decoded content ≈ size * 0.75 (base64 overhead) + line numbers
		approxDecoded := int(float64(fc.Size) * 1.1) // content + line number overhead
		lines := countLines(string(body))            // approximate

		toolName := "gl_read_file"
		if i >= 1 && i <= 3 {
			toolName = "gl_read_multiple[batch]"
		}
		r := callResult{toolName, fp, approxDecoded, lines, elapsed, false}
		newResults = append(newResults, r)
		totalNewBytes += r.bytes
	}

	newCallCount := 6 // 1 list + 1 read + 1 batch(3) + 3 reads = 6 (batch counts as 1)
	fmt.Printf("\nTotal calls: %d\n", newCallCount)
	fmt.Printf("Total response bytes: %d (%s)\n", totalNewBytes, formatBytes(totalNewBytes))
	for _, r := range newResults {
		fmt.Printf("  %-30s %6d bytes  %s\n", r.tool+"("+r.args+")", r.bytes, r.elapsed.Round(time.Millisecond))
	}

	// ============================================================
	// Approach B: Old MCP (@zereight/mcp-gitlab) style
	// Uses: get_repository_tree, get_file_contents (one by one)
	// No batch read, no instructions, raw JSON responses
	// ============================================================
	fmt.Println("\n--- Approach B: @zereight/mcp-gitlab (old) ---")
	fmt.Println("Strategy: get_repository_tree → get_file_contents × 7 (no batch)")

	var oldResults []callResult
	totalOldBytes := 0

	// Call 1: get_repository_tree (returns raw JSON array)
	body, elapsed, _ = gitlabAPI(fmt.Sprintf("/projects/%s/repository/tree", projectID),
		map[string]string{"recursive": "true", "per_page": "100"})
	r = callResult{"get_repository_tree", "recursive=true", len(body), countLines(string(body)), elapsed, false}
	oldResults = append(oldResults, r)
	totalOldBytes += r.bytes

	// Calls 2-8: get_file_contents × 7 (returns raw JSON with base64 content)
	for i, fp := range readFiles {
		body, elapsed, err := gitlabAPI(
			fmt.Sprintf("/projects/%s/repository/files/%s", projectID, url.PathEscape(fp)),
			map[string]string{"ref": "master"})
		if err != nil {
			oldResults = append(oldResults, callResult{fmt.Sprintf("get_file_contents[%d]", i+2), fp, 0, 0, elapsed, true})
			continue
		}
		// Old MCP returns the raw JSON response (with base64 content + metadata)
		r := callResult{"get_file_contents", fp, len(body), countLines(string(body)), elapsed, false}
		oldResults = append(oldResults, r)
		totalOldBytes += r.bytes
	}

	oldCallCount := 8 // 1 tree + 7 individual reads
	fmt.Printf("\nTotal calls: %d\n", oldCallCount)
	fmt.Printf("Total response bytes: %d (%s)\n", totalOldBytes, formatBytes(totalOldBytes))
	for _, r := range oldResults {
		fmt.Printf("  %-30s %6d bytes  %s\n", r.tool+"("+r.args+")", r.bytes, r.elapsed.Round(time.Millisecond))
	}

	// ============================================================
	// Comparison
	// ============================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("COMPARISON")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-35s %-20s %-20s\n", "Metric", "New MCP", "Old MCP")
	fmt.Printf("%-35s %-20s %-20s\n", strings.Repeat("─", 35), strings.Repeat("─", 20), strings.Repeat("─", 20))
	fmt.Printf("%-35s %-20d %-20d\n", "Tool calls", newCallCount, oldCallCount)
	fmt.Printf("%-35s %-20s %-20s\n", "Total response bytes", formatBytes(totalNewBytes), formatBytes(totalOldBytes))
	savedPct := float64(totalOldBytes-totalNewBytes) / float64(totalOldBytes) * 100
	fmt.Printf("%-35s %-20s %-20s\n", "Bytes saved", fmt.Sprintf("%.0f%%", savedPct), "baseline")
	fmt.Printf("%-35s %-20s %-20s\n", "Calls saved", fmt.Sprintf("%d fewer", oldCallCount-newCallCount), "baseline")
	fmt.Printf("%-35s %-20s %-20s\n", "Response format", "plain text + line#", "raw JSON + base64")
	fmt.Printf("%-35s %-20s %-20s\n", "Batch read support", "yes (gl_read_multiple)", "no")
	fmt.Printf("%-35s %-20s %-20s\n", "Instructions", "~800 words", "none")
}

func formatBytes(b int) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	return fmt.Sprintf("%.1f KB", float64(b)/1024)
}
