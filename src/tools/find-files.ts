import { z } from "zod";
import picomatch from "picomatch";
import type { GitLabClient } from "../gitlab/client.js";

export const findFilesSchema = z.object({
  project_id: z.string().describe("GitLab project ID or path"),
  pattern: z.string().describe("Glob pattern to match files (e.g. '**/*.ts', 'src/components/**/*.tsx')"),
  ref: z.string().optional().describe("Branch, tag, or commit SHA"),
  path: z.string().optional().describe("Directory to search in. Default: repository root"),
  max_results: z.number().optional().describe("Maximum results to return. Default: 50, max: 200"),
});

export type FindFilesInput = z.infer<typeof findFilesSchema>;

export async function findFiles(client: GitLabClient, input: FindFilesInput): Promise<string> {
  const ref = input.ref ?? (await client.getDefaultBranch(input.project_id));
  const maxResults = Math.min(input.max_results ?? 50, 200);
  const basePath = input.path ?? "";

  // Get full recursive tree
  const tree = await client.getTree(input.project_id, basePath, ref, true);

  // Filter to blobs only (files, not directories)
  const files = tree.filter((item) => item.type === "blob");

  // Apply glob matching
  const isMatch = picomatch(input.pattern, { dot: true });
  const matched = files.filter((f) => {
    // Match against path relative to basePath
    const relativePath = basePath && f.path.startsWith(basePath + "/")
      ? f.path.slice(basePath.length + 1)
      : f.path;
    return isMatch(relativePath) || isMatch(f.path);
  });

  // Limit results
  const limited = matched.slice(0, maxResults);
  const truncated = matched.length > maxResults;

  // Format output
  const header = `Found ${matched.length} file(s) matching "${input.pattern}" (ref: ${ref})`;
  const listing = limited.map((f) => `  ${f.path}`).join("\n");

  let result = `${header}\n${listing}`;
  if (truncated) {
    result += `\n\n⚠️ Showing ${maxResults} of ${matched.length} matches. Use a more specific pattern to narrow results.`;
  }
  if (matched.length === 0) {
    result += `\n\nNo files found. Try a broader pattern or check the path/ref.`;
  }

  return result;
}
