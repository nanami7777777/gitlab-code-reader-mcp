import { z } from "zod";
import picomatch from "picomatch";
import type { GitLabClient } from "../gitlab/client.js";

export const diffSchema = z.object({
  project_id: z.string().describe("GitLab project ID or path"),
  from_ref: z.string().optional().describe("Base ref to compare from (branch/tag/SHA)"),
  to_ref: z.string().optional().describe("Target ref to compare to"),
  merge_request_iid: z.number().optional().describe("MR IID — alternative to from_ref/to_ref"),
  file_pattern: z.string().optional().describe("Only show files matching this glob pattern"),
  exclude_patterns: z.array(z.string()).optional().describe("Exclude files matching these patterns (e.g. ['*.lock', 'dist/**'])"),
  max_files: z.number().optional().describe("Max files to show. Default: 20"),
});

export type DiffInput = z.infer<typeof diffSchema>;

export async function diff(client: GitLabClient, input: DiffInput): Promise<string> {
  const maxFiles = input.max_files ?? 20;

  let diffs;
  let label: string;

  if (input.merge_request_iid) {
    diffs = await client.getMergeRequestDiffs(input.project_id, input.merge_request_iid);
    label = `MR !${input.merge_request_iid}`;
  } else if (input.from_ref && input.to_ref) {
    const result = await client.compare(input.project_id, input.from_ref, input.to_ref);
    diffs = result.diffs;
    label = `${input.from_ref}...${input.to_ref}`;
  } else {
    return "Error: Provide either merge_request_iid or both from_ref and to_ref.";
  }

  // Filter by pattern
  if (input.file_pattern) {
    const isMatch = picomatch(input.file_pattern, { dot: true });
    diffs = diffs.filter((d) => isMatch(d.new_path) || isMatch(d.old_path));
  }

  // Exclude patterns
  if (input.exclude_patterns && input.exclude_patterns.length > 0) {
    const isExcluded = picomatch(input.exclude_patterns, { dot: true });
    diffs = diffs.filter((d) => !isExcluded(d.new_path) && !isExcluded(d.old_path));
  }

  const totalFiles = diffs.length;
  const limited = diffs.slice(0, maxFiles);

  const header = `Diff: ${label} (${totalFiles} files changed)`;

  const fileOutputs = limited.map((d) => {
    let status = "modified";
    if (d.new_file) status = "added";
    else if (d.deleted_file) status = "deleted";
    else if (d.renamed_file) status = `renamed: ${d.old_path} → ${d.new_path}`;

    const diffContent = d.diff.length > 3000
      ? d.diff.slice(0, 3000) + "\n... [diff truncated, use gl_read_file to see full content]"
      : d.diff;

    return `\n📄 ${d.new_path} (${status})\n${"─".repeat(40)}\n${diffContent}`;
  });

  let result = `${header}\n${fileOutputs.join("\n")}`;

  if (totalFiles > maxFiles) {
    result += `\n\n⚠️ Showing ${maxFiles} of ${totalFiles} changed files. Use file_pattern or exclude_patterns to filter.`;
  }

  return result;
}
