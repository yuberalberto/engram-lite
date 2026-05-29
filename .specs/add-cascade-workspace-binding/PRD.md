## Problem Statement

Cascade (Windsurf's AI agent) connects to engram-lite via a single, globally-configured MCP server process. Windsurf does not support per-workspace MCP server instances, so the server process CWD is always the Windsurf install directory — not the user's project workspace. CWD-based project detection always fails for Cascade: observations are written to the wrong project's database, or the server falls back to a hardcoded `ENGRAM_DATA_DIR` that does not change when the user switches workspaces.

Claude Code is unaffected: it launches a dedicated MCP server process per project with the correct CWD.

## Solution

Introduce a `mem_use_workspace` MCP tool that Cascade calls once at session start with the absolute path of the current workspace. The MCP server reads `<workspace>/.engram-lite/config.json` to resolve the project name and data directory, then stores this as connection-scoped context. All subsequent tool calls use this context when CWD detection is unavailable.

`engram-lite init` automatically injects a rule into `.windsurf/rules.md` instructing Cascade to call `mem_use_workspace` before any other engram tool.

## User Stories

1. As Cascade, I want to call `mem_use_workspace` with the current workspace path, so that all subsequent memory operations target the correct project's database.
2. As a developer using Windsurf, I want engram-lite to write observations to the active project's local database, so that memories from different projects do not mix.
3. As a developer using Windsurf, I want `engram-lite init` to automatically configure the workspace binding rule for Cascade, so that I do not have to write it manually.
4. As Cascade, I want a clear error when I attempt to write without a workspace context, so that I know to call `mem_use_workspace` first.
5. As a Claude Code user, I want CWD-based project detection to remain unchanged, so that the new feature does not affect my workflow.
6. As a developer, I want `mem_use_workspace` to validate that the given path contains a valid `.engram-lite/config.json`, so that misconfigured paths fail loudly.

## Implementation Decisions

- **Connection-scoped workspace state**: the MCP server (stdio, one connection per session) stores a `sessionWorkspace` field in memory after `mem_use_workspace` is called. Not persisted to the database.

- **Updated project resolution fallback chain**:
  1. Process-level override (`ENGRAM_PROJECT` env / `--project` flag / `ENGRAM_DATA_DIR/config.json`) — highest priority, trusted
  2. CWD detection — works for Claude Code
  3. Session workspace (set via `mem_use_workspace`) — Cascade's path
  4. None available → error: `"call mem_use_workspace(<path>) before writing"`

- **`mem_use_workspace` tool contract**:
  - Input: `workspace_path` (required string, absolute path)
  - Reads `<workspace_path>/.engram-lite/config.json` — fails loudly if missing or malformed
  - Returns: project name and data dir on success, descriptive error on failure
  - Does not write to the database
  - Idempotent — can be called again to switch workspace within a session

- **Rule injection in `init`**: the IDE config writer (`internal/ide`) is extended to write or append to `.windsurf/rules.md` with a rule instructing Cascade to call `mem_use_workspace` at the start of every session.

- **No changes to Claude Code's resolution path**: steps 1 and 2 are tried before step 3. Claude Code always resolves at step 2 and never reaches step 3.

## Testing Decisions

Good tests verify external behavior, not implementation details.

- **`mem_use_workspace` handler**: verify that calling it with a valid workspace path makes subsequent write calls target the correct project; verify that a missing `config.json` returns an error; verify idempotency.
- **Write tool enforcement**: verify that a write call before `mem_use_workspace` (with no CWD detection and no process override) returns the expected error message.
- **Rule injection**: verify that `init` writes or appends the correct rule to `.windsurf/rules.md` whether or not the file already exists.
- Prior art: `internal/mcp/mcp_test.go`, `internal/ide/ide_test.go`.

## Out of Scope

- Multi-workspace support per session (one active workspace per Cascade session is sufficient; alternating between projects within the same session is not a target use case)
- Persisting workspace binding across MCP server restarts
- Support for other globally-configured IDEs beyond Windsurf
- Automatic workspace detection via process tree inspection (deferred — too fragile across OS versions)

## Further Notes

- `.windsurf/mcp.json` at workspace root is not used by Windsurf (global config only) — may be removed or kept as reference.
- `ENGRAM_DATA_DIR` + `config.json` remains valid for CI/server environments without an IDE.
- `mem_debug_env` was removed after confirming root cause; `mem_use_workspace` is the permanent fix.
