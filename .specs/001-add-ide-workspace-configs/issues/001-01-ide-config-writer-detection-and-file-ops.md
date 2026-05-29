# 001-01 - IDEConfigWriter: detection and file operations

## What to Build

A new internal module — IDEConfigWriter — that encapsulates all logic for writing
workspace-level MCP configs for supported IDEs.

Given a workspace root and data dir path, the module:

1. Detects which IDEs are present by checking for their config directories
   (`.windsurf/`, `.vscode/`, `.cursor/`) in the workspace root.
2. For each detected IDE, applies the merge rule to its `mcp.json`:
   - File missing → create with engram-lite entry
   - File exists, no `mcpServers["engram-lite"]` key → inject the entry, preserve all other keys and formatting
   - File exists, `mcpServers["engram-lite"]` already present → no-op
3. Handles malformed JSON in existing files gracefully — logs a warning and skips
   that IDE rather than corrupting the file or crashing.

The generated entry sets `ENGRAM_DATA_DIR` to the absolute path of the workspace's
`.engram-lite/` directory. The MCP server already reads `project_name` from
`config.json` in that directory, so no `ENGRAM_PROJECT` env var is needed.

Returns a per-IDE result (created / merged / skipped / error) so the caller can
report status to the user.

## Acceptance Criteria

- [x] IDE dir exists, no `mcp.json` → file is created with correct engram-lite entry and `ENGRAM_DATA_DIR` pointing to the workspace data dir
- [x] IDE dir exists, `mcp.json` exists without `mcpServers["engram-lite"]` → entry is injected; all pre-existing servers are preserved verbatim
- [x] IDE dir exists, `mcp.json` exists with `mcpServers["engram-lite"]` already present → file is not modified (byte-for-byte identical after the call)
- [x] Malformed JSON in existing `mcp.json` → file is left untouched, an error result is returned for that IDE
- [x] Multiple IDEs detected in the same workspace → all receive correct, independent configs in one call
- [x] Only IDE dirs that exist in the workspace root are processed — no new directories are created by the detection step
- [x] All behavior is covered by unit tests using `t.TempDir()` for filesystem isolation

## Implementation Notes

- Package `internal/ide`, single exported function `WriteWorkspaceConfigs(workspaceRoot, dataDir string) []Result`
- JSON merge uses `encoding/json` round-trip via `map[string]any` — preserves keys, normalizes formatting
- `json.MarshalIndent` handles Windows path escaping automatically; no manual backslash replacement needed

## Blocked By

None — can start immediately.

## Type

AFK

## User Stories Covered

- US 1: Windsurf workspace → correct config generated automatically
- US 2: VS Code and Cursor → same behavior
- US 4: Existing `mcp.json` with other servers → other servers preserved
- US 5: Existing engram-lite entry → left intact
- US 7: Two workspaces → each config points to its own `ENGRAM_DATA_DIR`
