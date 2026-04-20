import { z } from "zod";
import type { GitLabClient } from "../gitlab/client.js";

export const commitHistorySchema = z.object({
  project_id: z.string().describe("GitLab project ID or path"),
  path: z.string().optional().describe("Limit to commits affecting this file/directory"),
  ref: z.string().optional().describe("Branch, tag, or commit SHA"),
  max_count: z.number().optional().describe("Max commits to return. Default: 20"),
  since: z.string().optional().describe("Only commits after this date (ISO 8601)"),
  author: z.string().optional().describe("Filter by author name or email"),
});

export type CommitHistoryInput = z.infer<typeof commitHistorySchema>;

export async function commitHistory(client: GitLabClient, input: CommitHistoryInput): Promise<string> {
  const commits = await client.listCommits(input.project_id, {
    ref: input.ref,
    path: input.path,
    since: input.since,
    author: input.author,
    perPage: input.max_count ?? 20,
  });

  if (commits.length === 0) {
    return `No commits found${input.path ? ` for ${input.path}` : ""}.`;
  }

  const header = `Commit history${input.path ? ` for ${input.path}` : ""} (${commits.length} commits)`;

  const lines = commits.map((c) => {
    const date = c.committed_date.slice(0, 10);
    const stats = c.stats ? ` (+${c.stats.additions} -${c.stats.deletions})` : "";
    return `  ${c.short_id} │ ${date} │ ${c.author_name} │ ${c.title}${stats}`;
  });

  return `${header}\n${"─".repeat(70)}\n${lines.join("\n")}`;
}
