# engram-lite

**Project-local persistent memory for AI coding agents.**

engram-lite gives your AI agent memory that survives across sessions. It stores decisions, bug fixes, discoveries, and conventions in a local SQLite database — scoped to your project, never leaving your machine.

## Quickstart

### Step 1 — Install the binary

Requires Go 1.21+. If you don't have Go, install it from [go.dev/dl](https://go.dev/dl).

```bash
go install github.com/yuberalberto/engram-lite/cmd/engram-lite@latest
engram-lite version  # verify
```

### Step 2 — Initialize your project

```bash
cd ~/projects/my-app
engram-lite init
```

This creates `.engram-lite/` with a config file and the database. Database files are added to `.gitignore` automatically; `config.json` is safe to commit.

### Step 3 — Connect your agent

#### Claude Code

```bash
claude plugin add github:yuberalberto/engram-lite
```

Restart Claude Code. No MCP config needed — the plugin handles everything.

#### Codex

`engram-lite init` writes a project-level Codex MCP config at
`.codex/config.toml`. Restart Codex after init.

Codex only loads project-scoped `.codex/config.toml` for trusted projects. If
Codex does not show the `engram-lite` MCP server after restart, trust the
workspace in Codex and restart again.

For plugin-based distribution, use the Codex plugin package in `plugin/codex`.
It exposes the MCP server and a Codex-specific memory skill without hooks. The
Codex package intentionally does not reuse the Claude Code plugin because Claude
hooks are not part of the Codex runtime contract.

#### Windsurf, VS Code, Cursor

`engram-lite init` (Step 2) automatically writes a workspace-level MCP config for each IDE it detects. If no IDE config directories are found, it prompts you to select which ones to configure.

**Supported IDEs:** Windsurf, VS Code, Cursor

After running `init`, restart your IDE to pick up the new config.

> `engram-lite` must be in your PATH. After `go install` it will be at `~/go/bin/engram-lite` (macOS/Linux) or `%USERPROFILE%\go\bin\engram-lite.exe` (Windows). If your agent can't find it, use the full path as the `command` value in the generated `mcp.json`.

---

## Memory protocol

Once connected, the agent works proactively:

- **Session start** — loads prior context for the current project (`mem_context`)
- **During work** — saves decisions, bug fixes, and discoveries as they happen (`mem_save`)
- **Session end** — persists a summary of what was accomplished (`mem_session_summary`)

Memory is stored at `<project-root>/.engram-lite/engram.db` — one database per project, nothing shared across projects.

## Tool profiles

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

Or reinstall: `go install github.com/yuberalberto/engram-lite/cmd/engram-lite@latest`

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
