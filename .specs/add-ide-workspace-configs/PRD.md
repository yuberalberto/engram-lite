## Problem Statement

When a developer installs engram-lite and runs `engram-lite init` in their workspace, the IDE MCP server starts with the IDE's own install directory as its working directory — not the workspace. This causes the MCP server to detect the wrong project (e.g. "windsurf" instead of "my-app"), so all memories are stored under the wrong project and are invisible in future sessions.

The root cause is that `engram-lite init` does not generate correct per-workspace IDE configs. The current implementation creates IDE config directories unconditionally for all three supported IDEs, skips files that already exist regardless of their content, exits early on re-runs, and has no fallback when no IDEs are detected. The README compounds the problem by directing users to configure a global IDE MCP config, which has no workspace context.

## Solution

`engram-lite init` generates or updates a workspace-level MCP config for each IDE detected in the workspace. The config hardcodes `ENGRAM_DATA_DIR` pointing to the workspace's `.engram-lite/` directory. The MCP server (already implemented) reads `project_name` from `config.json` in that directory, establishing correct project context independent of the process CWD.

Detection is based on the presence of the IDE's config directory in the workspace root (`.windsurf/`, `.vscode/`, `.cursor/`). If none are detected, `init` presents an interactive multi-select prompt. Merge behavior: skip if `mcpServers["engram-lite"]` already exists in the file; add the entry if the file exists without it; create the file if it doesn't exist. Re-running `init` always re-evaluates IDE configs even if `.engram-lite/config.json` already exists.

The README is updated to replace global IDE config instructions with a single `engram-lite init` step.

## User Stories

1. As a developer setting up engram-lite in a new workspace with Windsurf open, I want `engram-lite init` to generate `.windsurf/mcp.json` automatically, so that Windsurf's MCP server detects the correct project without any manual configuration.
2. As a developer using VS Code or Cursor, I want `engram-lite init` to generate the correct workspace-level MCP config for my IDE, so that engram-lite works the same way regardless of which IDE I use.
3. As a developer running `engram-lite init` in a workspace where no supported IDE directory is detected, I want to be prompted to select which IDEs to configure, so that I can still set up engram-lite even before opening the project in an IDE.
4. As a developer who already has `.windsurf/mcp.json` configured with other MCP servers (e.g. GitHub, Playwright), I want `engram-lite init` to add its entry without touching the others, so that my existing MCP setup is preserved.
5. As a developer who already ran `engram-lite init` and manually customized the engram-lite MCP entry, I want re-running `init` to leave my customization intact, so that I don't lose intentional changes.
6. As a developer re-running `engram-lite init` after upgrading, I want IDE configs to be evaluated even if the workspace was already initialized, so that any IDE added after the first `init` also gets configured.
7. As a developer with two workspaces both using engram-lite, I want each workspace's MCP server to read from its own `.engram-lite/config.json`, so that memories from different projects never mix.
8. As a new user reading the README, I want the setup instructions to tell me to run `engram-lite init` rather than manually edit global IDE configs, so that I don't end up with a broken configuration.

## Implementation Decisions

- **IDEConfigWriter module**: New internal module responsible for all IDE config logic. Interface: given a workspace root, data dir, and project name — detect present IDEs, apply merge logic per file, handle interactive prompt if none detected. All file I/O and JSON merge lives here.
- **Merge rule**: Parse the existing `mcp.json` as JSON; check for `mcpServers["engram-lite"]`; if present → no-op; if absent → inject the entry preserving all other keys; if file missing → write fresh. Preserve original JSON formatting where possible.
- **Detection rule**: IDE is "present" if its config directory exists in the workspace root at the time `init` is run. No PATH or process inspection.
- **Interactive prompt**: Triggered only when zero IDE directories are detected. Multi-select from: Windsurf, VS Code, Cursor. Creates the IDE directory and config file for each selected IDE.
- **`cmdInit` re-run behavior**: Remove the early-return when `dirExists && configExists`. IDE config generation runs unconditionally on every `init` invocation, after the data dir and `config.json` steps.
- **Generated config shape**: Sets `ENGRAM_DATA_DIR` to the absolute path of the workspace's `.engram-lite/`. The MCP server already reads `project_name` from `config.json` in that directory — no `ENGRAM_PROJECT` env var needed in the generated config.
- **README**: The Windsurf, VS Code, and Cursor setup sections are replaced by a single instruction: run `engram-lite init` in the workspace.

## Testing Decisions

Good tests verify observable behavior, not internal implementation. A test should set up a filesystem state (workspace dir, existing IDE dirs, existing mcp.json content), call the function under test, and assert on the resulting file contents — not on which internal functions were called.

**IDEConfigWriter**:
- IDE dir exists, no mcp.json → file is created with correct content
- IDE dir exists, mcp.json exists without engram-lite entry → entry is merged, other servers preserved
- IDE dir exists, mcp.json exists with engram-lite entry → file is not modified
- No IDE dirs detected → interactive prompt is shown; selected IDEs get their dirs and files created
- Multiple IDEs detected → all get correct configs in one run
- Malformed existing mcp.json → handled gracefully, not silently corrupted

**cmdInit**:
- Fresh workspace → creates data dir, config.json, and IDE configs for detected IDEs
- Re-run on already-initialized workspace → skips data dir setup, still evaluates and updates IDE configs
- Workspace with IDE dir but existing engram-lite config in mcp.json → init completes without modifying the IDE config
- Two separate workspaces → each gets isolated `ENGRAM_DATA_DIR` in their generated configs

Prior art: `store_test.go` uses `t.TempDir()` for filesystem isolation — same pattern applies here.

## Out of Scope

- Claude Code (handled by the plugin system, no changes needed)
- JetBrains or other IDEs beyond Windsurf, VS Code, and Cursor
- Detecting IDEs via PATH or running processes
- Automatically restarting the IDE after `init`
- Syncing or updating existing engram-lite MCP entries when the workspace path changes

## Further Notes

The MCP server's project detection from `ENGRAM_DATA_DIR/config.json` is already implemented in `cmdMCP` (reads `project_name` when `ENGRAM_PROJECT` env var is not set). This PRD does not change that logic — it only ensures `init` generates configs that point `ENGRAM_DATA_DIR` to the correct workspace.
