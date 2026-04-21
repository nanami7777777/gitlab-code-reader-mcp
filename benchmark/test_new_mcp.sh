#!/bin/bash
# Benchmark: Test new gitlab-code-reader-mcp
# Simulates the tool call sequence an AI would make to answer:
# "How does the Worker consume MQ messages in project 600?"
#
# Measures: response bytes, response time per call

set -e

SERVER="../server"
export GITLAB_TOKEN="${GITLAB_TOKEN:?Error: GITLAB_TOKEN environment variable is required}"
export GITLAB_URL="${GITLAB_URL:-https://gitlab.com}"

LOG_FILE="benchmark_new_mcp.log"
> "$LOG_FILE"

echo "=== Benchmark: gitlab-code-reader-mcp (new) ===" | tee -a "$LOG_FILE"
echo "Task: Understand Worker MQ consumption in project 600" | tee -a "$LOG_FILE"
echo "Started: $(date)" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"

# Helper: send a JSON-RPC request to the server and measure response
call_tool() {
    local tool_name="$1"
    local args="$2"
    local call_id="$3"
    
    local request="{\"jsonrpc\":\"2.0\",\"id\":$call_id,\"method\":\"tools/call\",\"params\":{\"name\":\"$tool_name\",\"arguments\":$args}}"
    
    local start_time=$(python3 -c "import time; print(time.time())")
    
    # Send init + tool call via stdio
    local init='{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"benchmark","version":"1.0"}}}'
    local initialized='{"jsonrpc":"2.0","method":"notifications/initialized"}'
    
    local response=$(echo -e "$init\n$initialized\n$request" | timeout 30 $SERVER 2>/dev/null | tail -1)
    
    local end_time=$(python3 -c "import time; print(time.time())")
    local elapsed=$(python3 -c "print(f'{$end_time - $start_time:.2f}s')")
    
    local resp_bytes=$(echo "$response" | wc -c | tr -d ' ')
    
    echo "[$call_id] $tool_name | ${resp_bytes} bytes | ${elapsed}" | tee -a "$LOG_FILE"
}

echo "--- Tool Calls ---" | tee -a "$LOG_FILE"

# Simulate the AI's exploration sequence:
# Step 1: List project structure
call_tool "gl_list_directory" '{"project_id":"600","depth":2}' 1

# Step 2: Read entry point
call_tool "gl_read_file" '{"project_id":"600","file_path":"cmd/consumer/main.go"}' 2

# Step 3: Read MQ layer files (batch)
call_tool "gl_read_multiple" '{"project_id":"600","files":[{"file_path":"internal/mq/runner.go"},{"file_path":"internal/mq/consumer_rabbitmq.go"},{"file_path":"internal/mq/dispatcher.go"}]}' 3

# Step 4: Read worker
call_tool "gl_read_file" '{"project_id":"600","file_path":"internal/dts/worker.go"}' 4

# Step 5: Read job factory
call_tool "gl_read_file" '{"project_id":"600","file_path":"internal/dts/job/job.go"}' 5

# Step 6: Read message struct
call_tool "gl_read_file" '{"project_id":"600","file_path":"internal/dts/message.go"}' 6

echo "" | tee -a "$LOG_FILE"
echo "Total calls: 6" | tee -a "$LOG_FILE"
echo "Finished: $(date)" | tee -a "$LOG_FILE"
