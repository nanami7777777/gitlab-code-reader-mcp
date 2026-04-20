/**
 * Lightweight symbol extraction using regex patterns.
 * Inspired by Claude Code's readCode tool: for large files, return signatures instead of full content.
 * Covers: TS/JS, Python, Go, Java, Rust common declarations.
 */

export interface Symbol {
  name: string;
  kind: string;       // "function" | "class" | "interface" | "type" | "method" | "const" | "enum"
  line: number;
  signature: string;  // The full declaration line
}

interface PatternDef {
  regex: RegExp;
  kind: string;
  nameGroup: number;
}

const TS_JS_PATTERNS: PatternDef[] = [
  { regex: /^(export\s+)?(default\s+)?(?:async\s+)?function\s*\*?\s+(\w+)/,       kind: "function",  nameGroup: 3 },
  { regex: /^(export\s+)?(const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(/,           kind: "function",  nameGroup: 3 },
  { regex: /^(export\s+)?(const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:\([^)]*\)|[a-zA-Z_]\w*)\s*=>/,  kind: "function",  nameGroup: 3 },
  { regex: /^(export\s+)?(abstract\s+)?class\s+(\w+)/,                              kind: "class",     nameGroup: 3 },
  { regex: /^(export\s+)?interface\s+(\w+)/,                                         kind: "interface", nameGroup: 2 },
  { regex: /^(export\s+)?type\s+(\w+)\s*=/,                                          kind: "type",      nameGroup: 2 },
  { regex: /^(export\s+)?enum\s+(\w+)/,                                              kind: "enum",      nameGroup: 2 },
  { regex: /^\s+(?:public|private|protected|static|async|readonly|\s)*(\w+)\s*\(/,   kind: "method",    nameGroup: 1 },
  { regex: /^(export\s+)?(const|let|var)\s+(\w+)\s*[=:]/,                            kind: "const",     nameGroup: 3 },
];

const PYTHON_PATTERNS: PatternDef[] = [
  { regex: /^(?:async\s+)?def\s+(\w+)\s*\(/,   kind: "function", nameGroup: 1 },
  { regex: /^class\s+(\w+)/,                     kind: "class",    nameGroup: 1 },
];

const GO_PATTERNS: PatternDef[] = [
  { regex: /^func\s+(?:\(\w+\s+\*?\w+\)\s+)?(\w+)\s*\(/,  kind: "function", nameGroup: 1 },
  { regex: /^type\s+(\w+)\s+struct/,                         kind: "class",    nameGroup: 1 },
  { regex: /^type\s+(\w+)\s+interface/,                      kind: "interface", nameGroup: 1 },
];

const JAVA_PATTERNS: PatternDef[] = [
  { regex: /^\s*(?:public|private|protected|static|\s)*(?:class|abstract\s+class)\s+(\w+)/,  kind: "class",    nameGroup: 1 },
  { regex: /^\s*(?:public|private|protected|static|\s)*interface\s+(\w+)/,                    kind: "interface", nameGroup: 1 },
  { regex: /^\s*(?:public|private|protected|static|final|synchronized|abstract|\s)+\w[\w<>\[\],\s]*\s+(\w+)\s*\(/, kind: "method", nameGroup: 1 },
];

const RUST_PATTERNS: PatternDef[] = [
  { regex: /^(?:pub\s+)?(?:async\s+)?fn\s+(\w+)/,    kind: "function",  nameGroup: 1 },
  { regex: /^(?:pub\s+)?struct\s+(\w+)/,               kind: "class",     nameGroup: 1 },
  { regex: /^(?:pub\s+)?trait\s+(\w+)/,                 kind: "interface", nameGroup: 1 },
  { regex: /^(?:pub\s+)?enum\s+(\w+)/,                  kind: "enum",      nameGroup: 1 },
  { regex: /^impl(?:<[^>]+>)?\s+(\w+)/,                 kind: "class",     nameGroup: 1 },
];

function getPatterns(filePath: string): PatternDef[] {
  const ext = filePath.split(".").pop()?.toLowerCase() ?? "";
  switch (ext) {
    case "ts": case "tsx": case "js": case "jsx": case "mjs": case "mts":
      return TS_JS_PATTERNS;
    case "py":
      return PYTHON_PATTERNS;
    case "go":
      return GO_PATTERNS;
    case "java": case "kt": case "scala":
      return JAVA_PATTERNS;
    case "rs":
      return RUST_PATTERNS;
    default:
      return [...TS_JS_PATTERNS, ...PYTHON_PATTERNS, ...GO_PATTERNS];
  }
}

export function extractSymbols(content: string, filePath: string): Symbol[] {
  const patterns = getPatterns(filePath);
  const lines = content.split("\n");
  const symbols: Symbol[] = [];
  const seen = new Set<string>();

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    for (const pat of patterns) {
      const match = line.match(pat.regex);
      if (match) {
        const name = match[pat.nameGroup];
        if (!name || name === "constructor" && pat.kind === "method") {
          // Keep constructor
        }
        const key = `${pat.kind}:${name}:${i}`;
        if (name && !seen.has(key)) {
          seen.add(key);
          symbols.push({
            name,
            kind: pat.kind,
            line: i + 1,
            signature: line.trimEnd(),
          });
        }
        break; // First match wins per line
      }
    }
  }

  return symbols;
}

export function formatSymbols(symbols: Symbol[]): string {
  if (symbols.length === 0) return "(no symbols found)";

  const grouped: Record<string, Symbol[]> = {};
  for (const s of symbols) {
    (grouped[s.kind] ??= []).push(s);
  }

  const lines: string[] = [];
  for (const [kind, syms] of Object.entries(grouped)) {
    lines.push(`\n## ${kind}s`);
    for (const s of syms) {
      lines.push(`  L${s.line}: ${s.signature}`);
    }
  }
  return lines.join("\n");
}
