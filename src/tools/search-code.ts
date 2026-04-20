import { z } from "zod";
import type { GitLabClient } from "../gitlab/client.js";

export const searchCodeSchema = z.object({
  project_id: z.string().describe("GitLab project ID or path"),
  query: z.string().describe("Search query (keywords or code pattern)"),
  ref: z.string().optional().describe("Branch to search in"),
  file_pattern: z.string().optional().describe("Filter results to files matching this pattern (e.g. '*.ts')"),
  max_results: z.number().optional().describe("Maximum results. Default: 20"),
});

export type SearchCodeInput = z.infer<typeof searchCodeSchema>;

export async function searchCode(client: GitLabClient, input: SearchCodeInput): Promise<string> {
  const maxResults = input.max_results ?? 20;

  const blobs = await client.searchCode(input.project_id, input.query, input.ref);

  // Filter by file pattern if provided
  let filtered = blobs;
  if (input.file_pattern) {
    const ext = input.file_pattern.replace("*", "");
    filtered = blobs.filter((b) => b.filename.endsWith(ext) || b.path.endsWith(ext));
  }

  const limited = filtered.slice(0, maxResults);

  if (limited.length === 0) {
    return `No results found for "${input.query}" in project ${input.project_id}.\n\nSuggestions:\n- Try different keywords\n- Check spelling\n- Use gl_find_files to locate files by name pattern instead`;
  }

  const header = `Found ${filtered.length} result(s) for "${input.query}"${input.ref ? ` (ref: ${input.ref})` : ""}`;

  const results = limited.map((blob) => {
    const lines = blob.data.split("\n").map((line, i) => {
      const lineNum = blob.startline + i;
      return `  ${String(lineNum).padStart(5)}\t${line}`;
    }).join("\n");

    return `\n📄 ${blob.path}:${blob.startline}\n${lines}`;
  });

  let output = `${header}\n${results.join("\n")}`;

  if (filtered.length > maxResults) {
    output += `\n\n⚠️ Showing ${maxResults} of ${filtered.length} results. Refine your query for more specific results.`;
  }

  return output;
}
