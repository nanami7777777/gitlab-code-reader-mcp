# Contributing

Thanks for your interest in contributing to gitlab-code-reader-mcp!

## Getting Started

```bash
git clone https://github.com/nanami7777777/gitlab-code-reader-mcp.git
cd gitlab-code-reader-mcp
npm install
npm run build
```

## Development

```bash
# Run in dev mode (auto-reload)
npm run dev

# Build
npm run build

# Type check
npx tsc --noEmit
```

## Testing Locally

Set your GitLab token and run:

```bash
GITLAB_TOKEN=glpat-xxx GITLAB_URL=https://gitlab.com npm start
```

Then connect via any MCP client (Claude Code, Kiro, etc.).

## Pull Requests

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Ensure `npm run build` passes with no errors
4. Write a clear PR description explaining what and why
5. Keep PRs focused — one feature or fix per PR

## Code Style

- TypeScript strict mode
- Prefer explicit types over `any` where practical
- Keep tool implementations in separate files under `src/tools/`
- Follow existing patterns for error handling and output formatting

## Adding a New Tool

1. Create `src/tools/your-tool.ts` with schema + handler
2. Register it in `src/index.ts` via `server.tool()`
3. Update README.md tool table
4. Add relevant GitLab API methods to `src/gitlab/client.ts`

## Reporting Issues

Open an issue with:
- What you expected
- What actually happened
- Steps to reproduce
- Your environment (Node version, MCP client, GitLab version)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
