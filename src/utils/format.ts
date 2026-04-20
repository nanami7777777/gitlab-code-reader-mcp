/**
 * Add line numbers to content, matching Claude Code's Read tool output format.
 * Format: "     1\tcontent"
 */
export function addLineNumbers(content: string, startLine = 1): string {
  const lines = content.split("\n");
  const maxDigits = String(startLine + lines.length - 1).length;
  return lines
    .map((line, i) => {
      const num = String(startLine + i).padStart(maxDigits, " ");
      return `${num}\t${line}`;
    })
    .join("\n");
}

/**
 * Truncate a single line if it exceeds maxLength (for minified files).
 */
export function truncateLine(line: string, maxLength = 500): string {
  if (line.length <= maxLength) return line;
  return line.slice(0, maxLength) + ` [truncated: ${line.length} chars]`;
}

/**
 * Detect if content is likely binary.
 */
export function isBinaryContent(base64Content: string): boolean {
  try {
    const decoded = Buffer.from(base64Content, "base64");
    // Check for null bytes in first 8KB
    const sample = decoded.subarray(0, 8192);
    for (let i = 0; i < sample.length; i++) {
      if (sample[i] === 0) return true;
    }
    return false;
  } catch {
    return true;
  }
}

/**
 * Decode base64 file content to UTF-8 string.
 */
export function decodeFileContent(base64Content: string): string {
  return Buffer.from(base64Content, "base64").toString("utf-8");
}

/**
 * Format file size in human-readable form.
 */
export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/**
 * Build a tree-style directory listing.
 */
export function formatTree(
  items: Array<{ path: string; type: string; name: string }>,
  basePath: string
): string {
  const lines: string[] = [];
  const prefix = basePath ? `${basePath}/` : "";

  // Group by directory depth relative to basePath
  const sorted = [...items].sort((a, b) => {
    // Directories first, then alphabetical
    if (a.type !== b.type) return a.type === "tree" ? -1 : 1;
    return a.path.localeCompare(b.path);
  });

  for (const item of sorted) {
    const relativePath = item.path.startsWith(prefix)
      ? item.path.slice(prefix.length)
      : item.path;
    const depth = relativePath.split("/").length - 1;
    const indent = "  ".repeat(depth);
    const icon = item.type === "tree" ? "📁" : "📄";
    lines.push(`${indent}${icon} ${item.name}`);
  }

  return lines.join("\n");
}
