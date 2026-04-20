import { z } from "zod";
import type { GitLabClient } from "../gitlab/client.js";
import { readFile } from "./read-file.js";

export const readMultipleSchema = z.object({
  project_id: z.string().describe("GitLab project ID or path"),
  files: z.array(z.object({
    file_path: z.string().describe("Path to the file"),
    start_line: z.number().optional().describe("Start line (1-based)"),
    end_line: z.number().optional().describe("End line (inclusive)"),
  })).min(1).max(10).describe("Array of files to read (max 10)"),
  ref: z.string().optional().describe("Branch, tag, or commit SHA"),
  max_lines_per_file: z.number().optional().describe("Max lines per file. Default: 200"),
});

export type ReadMultipleInput = z.infer<typeof readMultipleSchema>;

export async function readMultiple(client: GitLabClient, input: ReadMultipleInput): Promise<string> {
  const maxPerFile = input.max_lines_per_file ?? 200;
  const results: string[] = [];

  // Concurrent reads with concurrency limit of 5
  const chunks: typeof input.files[] = [];
  for (let i = 0; i < input.files.length; i += 5) {
    chunks.push(input.files.slice(i, i + 5));
  }

  for (const chunk of chunks) {
    const promises = chunk.map(async (f) => {
      try {
        return await readFile(client, {
          project_id: input.project_id,
          file_path: f.file_path,
          ref: input.ref,
          start_line: f.start_line,
          end_line: f.end_line,
          max_lines: maxPerFile,
        });
      } catch (err) {
        return `❌ Error reading ${f.file_path}: ${err instanceof Error ? err.message : String(err)}`;
      }
    });
    const chunkResults = await Promise.all(promises);
    results.push(...chunkResults);
  }

  return results.join("\n\n" + "═".repeat(60) + "\n\n");
}
