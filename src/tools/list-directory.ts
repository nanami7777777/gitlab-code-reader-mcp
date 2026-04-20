import { z } from "zod";
import type { GitLabClient } from "../gitlab/client.js";

export const listDirectorySchema = z.object({
  project_id: z.string().describe("GitLab project ID or path"),
  path: z.string().optional().describe("Directory path. Default: repository root"),
  ref: z.string().optional().describe("Branch, tag, or commit SHA"),
  depth: z.number().optional().describe("Recursion depth. Default: 1, max: 3"),
});

export type ListDirectoryInput = z.infer<typeof listDirectorySchema>;

export async function listDirectory(client: GitLabClient, input: ListDirectoryInput): Promise<string> {
  const ref = input.ref ?? (await client.getDefaultBranch(input.project_id));
  const depth = Math.min(input.depth ?? 1, 3);
  const basePath = input.path ?? "";

  const recursive = depth > 1;
  const tree = await client.getTree(input.project_id, basePath, ref, recursive);

  // Filter by depth
  const baseDepth = basePath ? basePath.split("/").length : 0;
  const filtered = tree.filter((item) => {
    const itemDepth = item.path.split("/").length - baseDepth;
    return itemDepth <= depth;
  });

  // Limit
  const maxItems = 200;
  const limited = filtered.slice(0, maxItems);
  const truncated = filtered.length > maxItems;

  // Build tree display
  const header = `Directory: ${basePath || "/"} (ref: ${ref}, ${filtered.length} items)`;

  const lines = limited.map((item) => {
    const relativePath = basePath && item.path.startsWith(basePath + "/")
      ? item.path.slice(basePath.length + 1)
      : item.path;
    const indent = "  ".repeat(relativePath.split("/").length - 1);
    const icon = item.type === "tree" ? "📁" : "📄";
    return `${indent}${icon} ${item.name}`;
  });

  let result = `${header}\n${"─".repeat(40)}\n${lines.join("\n")}`;

  if (truncated) {
    result += `\n\n⚠️ Showing ${maxItems} of ${filtered.length} items. Use a specific path to explore subdirectories.`;
  }

  return result;
}
