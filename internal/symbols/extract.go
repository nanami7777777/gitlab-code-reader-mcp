package symbols

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Symbol represents a code symbol extracted from a file.
type Symbol struct {
	Name      string
	Kind      string // function, class, interface, type, method, enum
	Line      int
	Signature string
}

type pattern struct {
	re        *regexp.Regexp
	kind      string
	nameGroup int
}

var tsPatterns = []pattern{
	{regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s+(\w+)`), "function", 1},
	{regexp.MustCompile(`^(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`), "class", 1},
	{regexp.MustCompile(`^(?:export\s+)?interface\s+(\w+)`), "interface", 1},
	{regexp.MustCompile(`^(?:export\s+)?type\s+(\w+)\s*=`), "type", 1},
	{regexp.MustCompile(`^(?:export\s+)?enum\s+(\w+)`), "enum", 1},
	{regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(`), "function", 1},
	{regexp.MustCompile(`^\s+(?:public|private|protected|static|async|readonly|\s)*(\w+)\s*\(`), "method", 1},
}

var pyPatterns = []pattern{
	{regexp.MustCompile(`^(?:async\s+)?def\s+(\w+)\s*\(`), "function", 1},
	{regexp.MustCompile(`^class\s+(\w+)`), "class", 1},
}

var goPatterns = []pattern{
	{regexp.MustCompile(`^func\s+(?:\(\w+\s+\*?\w+\)\s+)?(\w+)\s*\(`), "function", 1},
	{regexp.MustCompile(`^type\s+(\w+)\s+struct`), "class", 1},
	{regexp.MustCompile(`^type\s+(\w+)\s+interface`), "interface", 1},
}

var javaPatterns = []pattern{
	{regexp.MustCompile(`^\s*(?:public|private|protected|static|\s)*(?:class|abstract\s+class)\s+(\w+)`), "class", 1},
	{regexp.MustCompile(`^\s*(?:public|private|protected|static|\s)*interface\s+(\w+)`), "interface", 1},
}

var rustPatterns = []pattern{
	{regexp.MustCompile(`^(?:pub\s+)?(?:async\s+)?fn\s+(\w+)`), "function", 1},
	{regexp.MustCompile(`^(?:pub\s+)?struct\s+(\w+)`), "class", 1},
	{regexp.MustCompile(`^(?:pub\s+)?trait\s+(\w+)`), "interface", 1},
	{regexp.MustCompile(`^(?:pub\s+)?enum\s+(\w+)`), "enum", 1},
	{regexp.MustCompile(`^impl(?:<[^>]+>)?\s+(\w+)`), "class", 1},
}

func patternsForFile(path string) []pattern {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".mts":
		return tsPatterns
	case ".py":
		return pyPatterns
	case ".go":
		return goPatterns
	case ".java", ".kt", ".scala":
		return javaPatterns
	case ".rs":
		return rustPatterns
	default:
		// fallback: try all
		all := make([]pattern, 0, len(tsPatterns)+len(pyPatterns)+len(goPatterns))
		all = append(all, tsPatterns...)
		all = append(all, pyPatterns...)
		all = append(all, goPatterns...)
		return all
	}
}

// Extract extracts symbols from file content using regex patterns.
func Extract(content, filePath string) []Symbol {
	patterns := patternsForFile(filePath)
	lines := strings.Split(content, "\n")
	var syms []Symbol
	seen := map[string]bool{}

	for i, line := range lines {
		for _, p := range patterns {
			m := p.re.FindStringSubmatch(line)
			if m == nil || len(m) <= p.nameGroup {
				continue
			}
			name := m[p.nameGroup]
			key := fmt.Sprintf("%s:%s:%d", p.kind, name, i)
			if seen[key] {
				break
			}
			seen[key] = true
			syms = append(syms, Symbol{
				Name:      name,
				Kind:      p.kind,
				Line:      i + 1,
				Signature: strings.TrimRight(line, " \t\r"),
			})
			break
		}
	}
	return syms
}

// Format formats symbols into a readable string grouped by kind.
func Format(syms []Symbol) string {
	if len(syms) == 0 {
		return "(no symbols found)"
	}
	grouped := map[string][]Symbol{}
	order := []string{}
	for _, s := range syms {
		if _, ok := grouped[s.Kind]; !ok {
			order = append(order, s.Kind)
		}
		grouped[s.Kind] = append(grouped[s.Kind], s)
	}
	var b strings.Builder
	for _, kind := range order {
		fmt.Fprintf(&b, "\n## %ss\n", kind)
		for _, s := range grouped[kind] {
			fmt.Fprintf(&b, "  L%d: %s\n", s.Line, s.Signature)
		}
	}
	return b.String()
}
