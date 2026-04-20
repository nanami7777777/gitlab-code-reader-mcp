import { z } from "zod";
import type { GitLabClient } from "../gitlab/client.js";

export const blameSchema = z.object({
  project_id: z.string().describe("GitLab project ID or path"),
  file_path: z.string().describe("Path to the file"),
  ref: z.string().optional().describe("Branch, tag, or commit SHA"),
  start_line: z.number().optional().describe("Start line (1-based)"),
  end_line: z.number().optional().describe("End line (inclusive)"),
});

export type BlameInput = z.infer<typeof blameSchema>;

export async function blame(client: GitLabClient, input: BlameInput): Promise<string> {
  const ref = input.ref ?? (await client.getDefaultBranch(input.project_id));

  const ranges = await client.getBlame(input.project_id, input.file_path, ref);

  const header = `Blame: ${input.file_path} (ref: ${ref})`;
  const lines: string[] = [];
  let lineNum = 1;

  for (const range of ranges) {
    for (const line of range.lines) {
      if (input.start_line && lineNum < input.start_line) { lineNum++; continue; }
      if (input.end_line && lineNum > input.end_line) break;

      const date = range.commit.authored_date.slice(0, 10);
      const author = range.commit.author_name.padEnd(15).slice(0, 15);
      const sha = range.commit.id.slice(0, 8);
      lines.push(`${String(lineNum).padStart(5)} │ ${sha} │ ${date} │ ${author} │ ${line}`);
      lineNum++;
    }
    if (input.end_line && lineNum > input.end_line) break;
  }

  if (lines.length === 0) {
    return `${header}\nNo blame data for the specified line range.`;
  }

  return `${header}\n${"─".repeat(80)}\n${lines.join("\n")}`;
}
