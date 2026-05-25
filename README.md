# engram-lite

**Project-local persistent memory for AI coding agents.**

engram-lite gives your AI agent memory that survives across sessions. It stores decisions, bug fixes, discoveries, and conventions in a local SQLite database — scoped to your project, never leaving your machine.

## Quickstart (Claude Code)

**Step 1 — Install the binary**

Requires Go 1.21+. If you don't have Go, install it from [go.dev/dl](https://go.dev/dl).

```bash
go install github.com/yuberalberto/engram-lite/cmd/engram-lite@latest
```

On Windows this puts the binary at `%USERPROFILE%\go\bin\engram-lite.exe`. On macOS/Linux: `~/go/bin/engram-lite`.

**Step 2 — Add the Claude Code plugin**

```bash
claude plugin add github:yuberalberto/engram-lite
```

**Step 3 — Restart Claude Code**

That's it. No manual configuration needed.

### What happens next

- At session start, the agent is prompted to call `mem_context` and load prior memory for the current project.
- At session end, the agent is prompted to call `mem_session_summary` to persist what was accomplished.
- During work, the agent proactively calls `mem_save` after decisions, bug fixes, and discoveries.

Memory is stored at `<project-root>/.engram-lite/engram.db` — one database per project, nothing shared across projects.

## Initialize a project

Before the first session in a new project, run `init` to create the database and update `.gitignore`:

```bash
cd ~/projects/my-app
engram-lite init
```

This creates `.engram-lite/` with `config.json` and `engram.db`. Database files are gitignored automatically; `config.json` is safe to commit.

## Other AI agents (raw MCP)

If you're not using Claude Code, add this to your agent's MCP config:

```json
{
  "mcpServers": {
    "engram-lite": {
      "command": "engram-lite",
      "args": ["mcp", "--tools=agent"]
    }
  }
}
```

Make sure `engram-lite` is in your `PATH`. If not, use the full path to the binary.

### Tool profiles

| Profile | Tools | Use for |
|---------|-------|---------|
| `agent` | 15 | AI agents during coding sessions |
| `admin` | 4 | Manual curation and dashboards |
| `all` | All | Everything |

## CLI reference

| Command | Description |
|---------|-------------|
| `init` | Initialize engram-lite in the current project |
| `mcp [--tools=PROFILE]` | Start MCP server (stdio) |
| `serve [port]` | Start HTTP API server (default: 7437) |
| `tui` | Launch interactive terminal UI |
| `search <query>` | Search memories |
| `save <title> <content>` | Save a memory |
| `context [project]` | Show recent context |
| `timeline <obs_id>` | Show chronological context |
| `stats` | Show memory stats |
| `doctor` | Run diagnostics |
| `update` | Update to the latest version |
| `export [file]` | Export memories to JSON |
| `import <file>` | Import memories from JSON |
| `projects list` | List projects |
| `projects consolidate` | Merge similar project names |
| `projects prune` | Remove empty projects |
| `version` | Show version |

## Data storage

Database location: `<project-root>/.engram-lite/engram.db`

The project root is the nearest parent directory containing `.git/`. If none is found, the current directory is used.

**Override:** Set `ENGRAM_DATA_DIR` to use a custom path.

## Automatic backups

Every database open creates a backup at:

```
~/.engram-lite/backups/<project-name>/engram.db.bak
```

To restore after an accidental deletion:

```bash
cp ~/.engram-lite/backups/my-project/engram.db.bak ./.engram-lite/engram.db
```

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENGRAM_DATA_DIR` | Override data directory | `<project-root>/.engram-lite` |
| `ENGRAM_PORT` | HTTP server port | `7437` |
| `ENGRAM_PROJECT` | Default project name for MCP | auto-detected |

## Updating

```bash
engram-lite update
```

Or reinstall with `go install ... @latest`.

## How it compares to Engram

engram-lite is a fork of [Engram](https://github.com/Gentleman-Programming/engram) by Gentleman Programming, simplified for single-project, local-only use.

| Feature | Engram | engram-lite |
|---------|--------|-------------|
| Database location | `~/.engram/engram.db` (global) | `<project-root>/.engram-lite/engram.db` (local) |
| Cloud sync | ✅ | ❌ |
| Obsidian export | ✅ | ❌ |
| LLM conflict resolution | ✅ | ❌ |
| MCP server | ✅ | ✅ |
| HTTP API | ✅ | ✅ |
| TUI | ✅ | ✅ (simplified) |

## License

MIT — see [LICENSE](LICENSE).
