package gitlab

// TreeItem represents a file or directory in the repository tree.
type TreeItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "tree" or "blob"
	Path string `json:"path"`
	Mode string `json:"mode"`
}

// FileContent represents the response from the Repository Files API.
type FileContent struct {
	FileName      string `json:"file_name"`
	FilePath      string `json:"file_path"`
	Size          int64  `json:"size"`
	Encoding      string `json:"encoding"`
	Content       string `json:"content"` // base64
	ContentSHA256 string `json:"content_sha256"`
	Ref           string `json:"ref"`
	LastCommitID  string `json:"last_commit_id"`
}

// SearchBlob represents a code search result.
type SearchBlob struct {
	Basename  string `json:"basename"`
	Data      string `json:"data"`
	Path      string `json:"path"`
	Filename  string `json:"filename"`
	Ref       string `json:"ref"`
	Startline int    `json:"startline"`
	ProjectID int    `json:"project_id"`
}

// ProjectInfo holds basic project metadata.
type ProjectInfo struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	PathWithNS    string `json:"path_with_namespace"`
}

// CommitInfo represents a single commit.
type CommitInfo struct {
	ID            string       `json:"id"`
	ShortID       string       `json:"short_id"`
	Title         string       `json:"title"`
	Message       string       `json:"message"`
	AuthorName    string       `json:"author_name"`
	AuthorEmail   string       `json:"author_email"`
	AuthoredDate  string       `json:"authored_date"`
	CommittedDate string       `json:"committed_date"`
	Stats         *CommitStats `json:"stats,omitempty"`
}

// CommitStats holds addition/deletion counts.
type CommitStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Total     int `json:"total"`
}

// DiffFile represents a single file diff.
type DiffFile struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
	Diff        string `json:"diff"`
}

// CompareResult is the response from the Compare API.
type CompareResult struct {
	Diffs   []DiffFile   `json:"diffs"`
	Commits []CommitInfo `json:"commits"`
}

// MRChanges wraps the MR changes response.
type MRChanges struct {
	Changes []DiffFile `json:"changes"`
}

// BlameRange represents a blame chunk.
type BlameRange struct {
	Commit struct {
		ID           string `json:"id"`
		Message      string `json:"message"`
		AuthorName   string `json:"author_name"`
		AuthoredDate string `json:"authored_date"`
	} `json:"commit"`
	Lines []string `json:"lines"`
}
