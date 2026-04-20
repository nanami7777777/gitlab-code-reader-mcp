import { z } from "zod";
import type { GitLabClient } from "../gitlab/client.js";
import { isBinaryContent, decodeFileContent, formatSize } from "../utils/format.js";
import { extractSymbols, formatSymbols } from "../utils/symbols.js";
import { addLineNumbers, truncateLine } from "../utils/format.js";

export const readSymbolsSchema = z.object({
  project_id: z.string().describe("GitLab project ID or path"),
  file_path: z.string().describe("Path to the file"),
  ref: z.string().optional().describe("Branch, tag, or commit SHA"),
  symbol_filter: z.string().optional().describe("Filter symbols by name (case-insensitive substring match)"),
});

export type ReadSymbolsInput = z.infer<typeof readSymbolsSchema>;

/**
 * Inspired by Claude Code's readCode tool:
 * - Small files (<300 lines): return full content
 * - Large files: return symbol signatures with line numbers
 */
export async function readSymbols(client: GitLabClient, input: ReadSymbolsInput): Promise<string> {
  const ref = input.ref ?? (await client.getDefaultBranch(input.project_id));
  const file = await client.getFileContent(input.project_id, input.file_path, ref);

  if (isBinaryContent(file.content)) {
    return `Binary file: ${input.file_path} (${formatSize(file.size)})`;
  }

  const content = decodeFileContent(file.content);
  const lines = content.split("\n");
  const totalLines = lines.length;

  const header = `File: ${input.file_path} (${formatSize(file.size)}, ${totalLines} lines, ref: ${ref})`;

  // Small file: return full content (like Claude Code's readCode for <10k chars)
  if (totalLines <= 300) {
    const processed = lines.map((l) => truncateLine(l, 500));
    const numbered = addLineNumbers(processed.join("\n"), 1);
    return `${header}\n(small file — returning full content)\n${"─".repeat(60)}\n${numbered}`;
  }

  // Large file: extract and return symbols
  let symbols = extractSymbols(content, input.file_path);

  if (input.symbol_filter) {
    const filter = input.symbol_filter.toLowerCase();
    symbols = symbols.filter((s) => s.name.toLowerCase().includes(filter));
  }

  const symbolOutput = formatSymbols(symbols);

  return `${header}\n(large file — showing ${symbols.length} symbol signatures)\n${symbolOutput}\n\n💡 Use gl_read_file with start_line/end_line to read specific implementations.`;
}
