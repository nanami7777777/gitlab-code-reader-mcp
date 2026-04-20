import { z } from "zod";
import type { GitLabClient } from "../gitlab/client.js";
import { addLineNumbers, truncateLine, isBinaryContent, decodeFileContent, formatSize } from "../utils/format.js";

export const readFileSchema = z.object({
  project_id: z.string().describe("GitLab project ID or path (e.g. 'mygroup/myproject')"),
  file_path: z.string().describe("Path to the file in the repository"),
  ref: z.string().optional().describe("Branch, tag, or commit SHA. Defaults to the project's default branch"),
  start_line: z.number().optional().describe("Start reading from this line number (1-based). Default: 1"),
  end_line: z.number().optional().describe("Stop reading at this line number (inclusive). Default: end of file"),
  max_lines: z.number().optional().describe("Maximum number of lines to return. Default: 500"),
});

export type ReadFileInput = z.infer<typeof readFileSchema>;

export async function readFile(client: GitLabClient, input: ReadFileInput): Promise<string> {
  const ref = input.ref ?? (await client.getDefaultBranch(input.project_id));
  const maxLines = input.max_lines ?? 500;

  const file = await client.getFileContent(input.project_id, input.file_path, ref);

  // Binary detection
  if (isBinaryContent(file.content)) {
    return `Binary file: ${input.file_path} (${formatSize(file.size)})\nCannot display binary content. Use GitLab UI to view this file.`;
  }

  const fullContent = decodeFileContent(file.content);
  const allLines = fullContent.split("\n");
  const totalLines = allLines.length;

  // Apply line range
  const start = Math.max(1, input.start_line ?? 1);
  const end = Math.min(totalLines, input.end_line ?? totalLines);
  let selectedLines = allLines.slice(start - 1, end);

  // Apply max_lines truncation
  let truncated = false;
  if (selectedLines.length > maxLines) {
    selectedLines = selectedLines.slice(0, maxLines);
    truncated = true;
  }

  // Truncate long lines (minified file protection)
  const processed = selectedLines.map((l) => truncateLine(l, 500));

  // Format with line numbers
  const output = addLineNumbers(processed.join("\n"), start);

  // Build header
  const header = `File: ${input.file_path} (${formatSize(file.size)}, ${totalLines} lines, ref: ${ref})`;
  const range = `Showing lines ${start}-${start + selectedLines.length - 1} of ${totalLines}`;

  let result = `${header}\n${range}\n${"─".repeat(60)}\n${output}`;

  if (truncated) {
    result += `\n${"─".repeat(60)}\n⚠️ Output truncated at ${maxLines} lines. Use start_line/end_line to read remaining content.`;
  }

  return result;
}
