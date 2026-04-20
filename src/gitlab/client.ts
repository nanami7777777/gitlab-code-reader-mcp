import { LRUCache } from "./cache.js";
import type {
  GitLabConfig,
  TreeItem,
  FileContent,
  SearchBlob,
  ProjectInfo,
  CommitInfo,
  DiffFile,
  BlameRange,
  CompareResult,
} from "./types.js";

const CACHE_TTL = {
  tree: 5 * 60 * 1000,       // 5 min
  file: 5 * 60 * 1000,       // 5 min
  project: 10 * 60 * 1000,   // 10 min
  search: 2 * 60 * 1000,     // 2 min
};

export class GitLabClient {
  private baseUrl: string;
  private token: string;
  private cache = new LRUCache(500);

  constructor(config: GitLabConfig) {
    this.baseUrl = config.baseUrl.replace(/\/+$/, "");
    this.token = config.token;
  }

  private async request<T>(path: string, params?: Record<string, string | number | boolean>): Promise<T> {
    const url = new URL(`${this.baseUrl}/api/v4${path}`);
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        if (v !== undefined && v !== null && v !== "") {
          url.searchParams.set(k, String(v));
        }
      }
    }

    const res = await fetch(url.toString(), {
      headers: {
        "PRIVATE-TOKEN": this.token,
        "Accept": "application/json",
      },
    });

    if (!res.ok) {
      const body = await res.text().catch(() => "");
      throw new Error(`GitLab API ${res.status}: ${res.statusText} — ${path}\n${body}`);
    }

    return res.json() as Promise<T>;
  }

  private pid(projectId: string): string {
    // If it's a numeric ID, use as-is. If it's a path like "group/project", encode the slash.
    if (/^\d+$/.test(projectId)) return projectId;
    return encodeURIComponent(projectId);
  }

  // --- Project ---

  async getProject(projectId: string): Promise<ProjectInfo> {
    const key = `project:${projectId}`;
    const cached = this.cache.get<ProjectInfo>(key);
    if (cached) return cached;

    const data = await this.request<ProjectInfo>(`/projects/${this.pid(projectId)}`);
    this.cache.set(key, data, CACHE_TTL.project);
    return data;
  }

  async getDefaultBranch(projectId: string): Promise<string> {
    const project = await this.getProject(projectId);
    return project.default_branch;
  }

  // --- Repository Tree ---

  async getTree(projectId: string, path: string, ref: string, recursive: boolean): Promise<TreeItem[]> {
    const key = `tree:${projectId}:${ref}:${path}:${recursive}`;
    const cached = this.cache.get<TreeItem[]>(key);
    if (cached) return cached;

    const allItems: TreeItem[] = [];
    let page = 1;
    const perPage = 100;

    while (true) {
      const items = await this.request<TreeItem[]>(
        `/projects/${this.pid(projectId)}/repository/tree`,
        { path, ref, recursive, per_page: perPage, page }
      );
      allItems.push(...items);
      if (items.length < perPage) break;
      page++;
      if (allItems.length > 5000) break; // safety limit
    }

    this.cache.set(key, allItems, CACHE_TTL.tree);
    return allItems;
  }

  // --- File Content ---

  async getFileContent(projectId: string, filePath: string, ref: string): Promise<FileContent> {
    const key = `file:${projectId}:${ref}:${filePath}`;
    const cached = this.cache.get<FileContent>(key);
    if (cached) return cached;

    const data = await this.request<FileContent>(
      `/projects/${this.pid(projectId)}/repository/files/${encodeURIComponent(filePath)}`,
      { ref }
    );
    this.cache.set(key, data, CACHE_TTL.file);
    return data;
  }

  // --- Search ---

  async searchCode(projectId: string, query: string, ref?: string): Promise<SearchBlob[]> {
    const key = `search:${projectId}:${query}:${ref ?? ""}`;
    const cached = this.cache.get<SearchBlob[]>(key);
    if (cached) return cached;

    const params: Record<string, string | number | boolean> = {
      scope: "blobs",
      search: query,
      per_page: 50,
    };
    if (ref) params.ref = ref;

    const data = await this.request<SearchBlob[]>(
      `/projects/${this.pid(projectId)}/search`,
      params
    );
    this.cache.set(key, data, CACHE_TTL.search);
    return data;
  }

  // --- Commits ---

  async listCommits(
    projectId: string,
    opts: { ref?: string; path?: string; since?: string; author?: string; perPage?: number }
  ): Promise<CommitInfo[]> {
    const params: Record<string, string | number | boolean> = {
      per_page: opts.perPage ?? 20,
    };
    if (opts.ref) params.ref_name = opts.ref;
    if (opts.path) params.path = opts.path;
    if (opts.since) params.since = opts.since;
    if (opts.author) params.author = opts.author;
    params.with_stats = true;

    return this.request<CommitInfo[]>(
      `/projects/${this.pid(projectId)}/repository/commits`,
      params
    );
  }

  // --- Compare / Diff ---

  async compare(projectId: string, from: string, to: string): Promise<CompareResult> {
    return this.request<CompareResult>(
      `/projects/${this.pid(projectId)}/repository/compare`,
      { from, to }
    );
  }

  async getMergeRequestDiffs(projectId: string, mrIid: number): Promise<DiffFile[]> {
    const changes = await this.request<{ changes: DiffFile[] }>(
      `/projects/${this.pid(projectId)}/merge_requests/${mrIid}/changes`
    );
    return changes.changes;
  }

  // --- Blame ---

  async getBlame(projectId: string, filePath: string, ref: string): Promise<BlameRange[]> {
    return this.request<BlameRange[]>(
      `/projects/${this.pid(projectId)}/repository/files/${encodeURIComponent(filePath)}/blame`,
      { ref }
    );
  }
}
