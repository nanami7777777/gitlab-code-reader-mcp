# Contributing

Thanks for your interest in contributing to gitlab-code-reader-mcp!

## Getting Started

```bash
git clone https://github.com/nanami7777777/gitlab-code-reader-mcp.git
cd gitlab-code-reader-mcp
go build ./cmd/server/
go test ./...
```

## Development

```bash
# Build
go build -o server ./cmd/server/

# Run
GITLAB_TOKEN=glpat-xxx ./server

# Test
go test ./... -v

# Lint (optional, requires golangci-lint)
golangci-lint run
```

## Pull Requests

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Ensure `go build ./cmd/server/` and `go test ./...` pass
4. Write a clear PR description explaining what and why
5. Keep PRs focused — one feature or fix per PR

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep tool implementations in separate files under `internal/tools/`
- Follow existing patterns for error handling and output formatting
- Add tests for new functionality

## Adding a New Tool

1. Create `internal/tools/your_tool.go` with tool definition + handler
2. Register it in `cmd/server/main.go`
3. Update README.md tool table (both EN and CN sections)
4. Add relevant GitLab API methods to `internal/gitlab/client.go`
5. Add tests

## Reporting Issues

Open an issue with:
- What you expected
- What actually happened
- Steps to reproduce
- Your environment (Go version, MCP client, GitLab version)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
