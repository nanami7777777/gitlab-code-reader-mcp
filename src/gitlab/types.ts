export interface GitLabConfig {
  baseUrl: string;
  token: string;
}

export interface TreeItem {
  id: string;
  name: string;
  type: "tree" | "blob";
  path: string;
  mode: string;
}

export interface FileContent {
  file_name: string;
  file_path: string;
  size: number;
  encoding: string;
  content: string;       // base64 encoded
  content_sha256: string;
  ref: string;
  last_commit_id: string;
}

export interface SearchBlob {
  basename: string;
  data: string;
  path: string;
  filename: string;
  id: string | null;
  ref: string;
  startline: number;
  project_id: number;
}

export interface ProjectInfo {
  id: number;
  name: string;
  default_branch: string;
  path_with_namespace: string;
}

export interface CommitInfo {
  id: string;
  short_id: string;
  title: string;
  message: string;
  author_name: string;
  author_email: string;
  authored_date: string;
  committed_date: string;
  stats?: { additions: number; deletions: number; total: number };
}

export interface DiffFile {
  old_path: string;
  new_path: string;
  new_file: boolean;
  renamed_file: boolean;
  deleted_file: boolean;
  diff: string;
}

export interface BlameRange {
  commit: {
    id: string;
    message: string;
    author_name: string;
    authored_date: string;
  };
  lines: string[];
}

export interface CompareResult {
  diffs: DiffFile[];
  commits: CommitInfo[];
}
