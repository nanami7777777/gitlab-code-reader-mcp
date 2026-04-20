package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var cacheTTL = struct {
	Tree, File, Project, Search time.Duration
}{
	Tree:    5 * time.Minute,
	File:    5 * time.Minute,
	Project: 10 * time.Minute,
	Search:  2 * time.Minute,
}

// Client wraps GitLab REST API calls with caching.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	cache      *Cache
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      NewCache(500),
	}
}

func (c *Client) request(path string, params map[string]string) ([]byte, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v4%s", c.baseURL, path))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GitLab API %d: %s — %s\n%s", resp.StatusCode, resp.Status, path, string(body))
	}
	return body, nil
}

// pid encodes the project ID for URL path segments.
func pid(projectID string) string {
	if _, err := strconv.Atoi(projectID); err == nil {
		return projectID
	}
	return url.PathEscape(projectID)
}

// GetProject returns basic project info (cached).
func (c *Client) GetProject(projectID string) (*ProjectInfo, error) {
	key := "project:" + projectID
	if v, ok := c.cache.Get(key); ok {
		return v.(*ProjectInfo), nil
	}
	data, err := c.request(fmt.Sprintf("/projects/%s", pid(projectID)), nil)
	if err != nil {
		return nil, err
	}
	var p ProjectInfo
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	c.cache.Set(key, &p, cacheTTL.Project)
	return &p, nil
}

// DefaultBranch returns the project's default branch.
func (c *Client) DefaultBranch(projectID string) (string, error) {
	p, err := c.GetProject(projectID)
	if err != nil {
		return "", err
	}
	return p.DefaultBranch, nil
}

// GetTree returns the repository tree (cached).
func (c *Client) GetTree(projectID, path, ref string, recursive bool) ([]TreeItem, error) {
	key := fmt.Sprintf("tree:%s:%s:%s:%v", projectID, ref, path, recursive)
	if v, ok := c.cache.Get(key); ok {
		return v.([]TreeItem), nil
	}

	var all []TreeItem
	page := 1
	for {
		params := map[string]string{
			"path":     path,
			"ref":      ref,
			"per_page": "100",
			"page":     strconv.Itoa(page),
		}
		if recursive {
			params["recursive"] = "true"
		}
		data, err := c.request(fmt.Sprintf("/projects/%s/repository/tree", pid(projectID)), params)
		if err != nil {
			return nil, err
		}
		var items []TreeItem
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) < 100 || len(all) > 5000 {
			break
		}
		page++
	}
	c.cache.Set(key, all, cacheTTL.Tree)
	return all, nil
}

// GetFileContent returns file content from the Repository Files API (cached).
func (c *Client) GetFileContent(projectID, filePath, ref string) (*FileContent, error) {
	key := fmt.Sprintf("file:%s:%s:%s", projectID, ref, filePath)
	if v, ok := c.cache.Get(key); ok {
		return v.(*FileContent), nil
	}
	encodedPath := url.PathEscape(filePath)
	data, err := c.request(
		fmt.Sprintf("/projects/%s/repository/files/%s", pid(projectID), encodedPath),
		map[string]string{"ref": ref},
	)
	if err != nil {
		return nil, err
	}
	var fc FileContent
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, err
	}
	c.cache.Set(key, &fc, cacheTTL.File)
	return &fc, nil
}

// SearchCode searches for code blobs in a project (cached).
func (c *Client) SearchCode(projectID, query, ref string) ([]SearchBlob, error) {
	key := fmt.Sprintf("search:%s:%s:%s", projectID, query, ref)
	if v, ok := c.cache.Get(key); ok {
		return v.([]SearchBlob), nil
	}
	params := map[string]string{"scope": "blobs", "search": query, "per_page": "50"}
	if ref != "" {
		params["ref"] = ref
	}
	data, err := c.request(fmt.Sprintf("/projects/%s/search", pid(projectID)), params)
	if err != nil {
		return nil, err
	}
	var blobs []SearchBlob
	if err := json.Unmarshal(data, &blobs); err != nil {
		return nil, err
	}
	c.cache.Set(key, blobs, cacheTTL.Search)
	return blobs, nil
}

// ListCommits returns commit history for a project.
func (c *Client) ListCommits(projectID string, opts map[string]string) ([]CommitInfo, error) {
	if opts == nil {
		opts = map[string]string{}
	}
	if opts["per_page"] == "" {
		opts["per_page"] = "20"
	}
	opts["with_stats"] = "true"
	data, err := c.request(fmt.Sprintf("/projects/%s/repository/commits", pid(projectID)), opts)
	if err != nil {
		return nil, err
	}
	var commits []CommitInfo
	return commits, json.Unmarshal(data, &commits)
}

// Compare returns the diff between two refs.
func (c *Client) Compare(projectID, from, to string) (*CompareResult, error) {
	data, err := c.request(
		fmt.Sprintf("/projects/%s/repository/compare", pid(projectID)),
		map[string]string{"from": from, "to": to},
	)
	if err != nil {
		return nil, err
	}
	var result CompareResult
	return &result, json.Unmarshal(data, &result)
}

// GetMRDiffs returns the file diffs for a merge request.
func (c *Client) GetMRDiffs(projectID string, mrIID int) ([]DiffFile, error) {
	data, err := c.request(
		fmt.Sprintf("/projects/%s/merge_requests/%d/changes", pid(projectID), mrIID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	var changes MRChanges
	if err := json.Unmarshal(data, &changes); err != nil {
		return nil, err
	}
	return changes.Changes, nil
}

// GetBlame returns blame data for a file.
func (c *Client) GetBlame(projectID, filePath, ref string) ([]BlameRange, error) {
	encodedPath := url.PathEscape(filePath)
	data, err := c.request(
		fmt.Sprintf("/projects/%s/repository/files/%s/blame", pid(projectID), encodedPath),
		map[string]string{"ref": ref},
	)
	if err != nil {
		return nil, err
	}
	var ranges []BlameRange
	return ranges, json.Unmarshal(data, &ranges)
}
