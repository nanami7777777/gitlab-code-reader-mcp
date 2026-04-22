package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Tests what the AI sees differently with vs without instructions.
// Sends initialize request to both server variants and compares responses.

func main() {
	for _, variant := range []struct {
		name   string
		binary string
	}{
		{"WITH instructions", "./server"},
		{"WITHOUT instructions", "./server_no_inst"},
	} {
		fmt.Printf("\n=== %s ===\n", variant.name)

		cmd := exec.Command(variant.binary)
		cmd.Env = append(os.Environ(),
			"GITLAB_TOKEN="+os.Getenv("GITLAB_TOKEN"),
			"GITLAB_URL="+os.Getenv("GITLAB_URL"),
		)

		stdin, _ := cmd.StdinPipe()
		stdout, _ := cmd.StdoutPipe()
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting %s: %v\n", variant.binary, err)
			continue
		}

		reader := bufio.NewReader(stdout)

		// Send initialize
		initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"benchmark","version":"1.0"}}}` + "\n"
		io.WriteString(stdin, initReq)
		time.Sleep(500 * time.Millisecond)

		line, _ := reader.ReadString('\n')
		var resp map[string]any
		json.Unmarshal([]byte(line), &resp)

		result, _ := resp["result"].(map[string]any)
		inst, _ := result["instructions"].(string)

		if inst == "" {
			fmt.Println("Instructions: (none)")
			fmt.Println("AI receives: only tool names and descriptions")
		} else {
			fmt.Printf("Instructions: %d chars, %d words\n", len(inst), len(strings.Fields(inst)))
			fmt.Printf("Contains RULES section: %v\n", strings.Contains(inst, "RULES"))
			fmt.Printf("Contains prohibitions (NOT): %d occurrences\n", strings.Count(inst, "NOT"))
			fmt.Printf("Contains MANDATORY: %v\n", strings.Contains(inst, "MANDATORY"))
			fmt.Printf("Contains STOP condition: %v\n", strings.Contains(inst, "STOP"))
			fmt.Printf("Contains priority order: %v\n", strings.Contains(inst, "strict priority"))
			fmt.Println("---")
			// Print first 3 lines
			for i, line := range strings.SplitN(inst, "\n", 4) {
				if i >= 3 {
					break
				}
				fmt.Printf("  %s\n", line)
			}
			fmt.Println("  ...")
		}

		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("WHAT THIS MEANS:")
	fmt.Println("")
	fmt.Println("With instructions, the AI sees behavioral rules BEFORE")
	fmt.Println("making any tool calls. These rules tell it:")
	fmt.Println("  - WHAT to use (gl_read_file, not curl)")
	fmt.Println("  - What NOT to do (don't read entire large files)")
	fmt.Println("  - MANDATORY parallel calls")
	fmt.Println("  - WHEN to stop searching")
	fmt.Println("  - Strict priority order for tool selection")
	fmt.Println("")
	fmt.Println("Without instructions, the AI only sees tool names and")
	fmt.Println("descriptions. It must discover usage patterns on its own,")
	fmt.Println("often leading to suboptimal tool choices.")
}
