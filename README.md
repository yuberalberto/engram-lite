# engram-lite

**Project-local persistent memory for AI coding agents.**

A lightweight fork of [Engram](https://github.com/Gentleman-Programming/engram) by Gentleman Programming, simplified for single-project, local-only use.

## What's Different from Engram?

| Feature | Engram | engram-lite |
|---------|--------|-------------|
| Database location | `~/.engram-lite/engram.db` (global) | `<project-root>/.engram-lite/engram.db` (local) |
| Cloud sync | ✅ | ❌ Removed |
| Obsidian export | ✅ | ❌ Removed |
| LLM conflict resolution | ✅ | ❌ Removed |
| Agent plugin setup | ✅ | ❌ Removed |
| Docker support | ✅ | ❌ Removed |
| MCP server | ✅ | ✅ |
| HTTP API | ✅ | ✅ |
| TUI | ✅ | ✅ (simplified) |
| CLI commands | Full | Core subset |

## Installation

```bash
go install github.com/yuberalberto/engram-lite/cmd/engram-lite@latest
```

Or build from source:

```bash
git clone https://github.com/yuberalberto/engram-lite.git
cd engram-lite
go build -o engram-lite ./cmd/engram-lite
```

## Quick Start

```bash
# Navigate to your project directory
cd ~/projects/my-app

# Save a memory
engram-lite save "Architecture decision" "Using PostgreSQL for persistence due to JSONB support"

# Search memories
engram-lite search "database"

# Start MCP server (for AI agents)
engram-lite mcp --tools=agent

# Launch TUI
engram-lite tui

# View stats
engram-lite stats
```

## Data Storage

engram-lite stores its SQLite database at `<project-root>/.engram-lite/engram.db`.

The project root is detected by walking up from the current working directory to find a `.git/` directory. If no `.git/` is found, the current directory is used.

**Override:** Set `ENGRAM_DATA_DIR` environment variable to use a custom path.

**Tip:** Add `.engram-lite/` to your project's `.gitignore`:
```
.engram-lite/
```

## MCP Configuration

Add to your AI agent's MCP config:

```json
{
  "mcp": {
    "engram": {
      "type": "stdio",
      "command": "engram-lite",
      "args": ["mcp", "--tools=agent"]
    }
  }
}
```

### Tool Profiles

- **agent** (15 tools) — Tools AI agents use during coding sessions
- **admin** (4 tools) — Tools for manual curation and dashboards
- **all** (default) — Every tool registered

## CLI Commands

| Command | Description |
|---------|-------------|
| `serve [port]` | Start HTTP API server (default: 7437) |
| `mcp [--tools=PROFILE]` | Start MCP server (stdio) |
| `tui` | Launch interactive terminal UI |
| `search <query>` | Search memories |
| `save <title> <content>` | Save a memory |
| `timeline <obs_id>` | Show chronological context |
| `context [project]` | Show recent context |
| `stats` | Show memory stats |
| `doctor` | Run diagnostics |
| `export [file]` | Export memories to JSON |
| `import <file>` | Import memories from JSON |
| `projects list` | List projects |
| `projects consolidate` | Merge similar project names |
| `projects prune` | Remove empty projects |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENGRAM_DATA_DIR` | Override data directory | `<project-root>/.engram` |
| `ENGRAM_PORT` | HTTP server port | `7437` |
| `ENGRAM_PROJECT` | Default project name for MCP | auto-detected |

## License

MIT — see [LICENSE](LICENSE).

## Attribution

This project is a fork of [Engram](https://github.com/Gentleman-Programming/engram) created by [Gentleman Programming](https://github.com/Gentleman-Programming). The original project is licensed under MIT.

engram-lite removes cloud synchronization, LLM integration, and plugin setup features to provide a simpler, project-local memory store for AI coding agents.
