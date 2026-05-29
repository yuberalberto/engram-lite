# 001-03 - cmdInit: wire IDEConfigWriter and fix re-run

## What to Build

Update `cmdInit` to call IDEConfigWriter after writing `config.json`, and remove
the early-return that prevents IDE config generation on re-runs.

Specific changes:

1. Remove the guard that exits with "Already initialized" when both the data dir
   and `config.json` exist. Data dir and config.json setup can still be skipped
   when already present — but execution must continue to the IDE config step.
2. Call IDEConfigWriter with the workspace root and data dir after the config.json
   step, on every invocation of `init`.
3. Update the CLI output to report the result for each IDE:
   - Created: `IDE MCP config created: Windsurf`
   - Merged: `IDE MCP config updated: VS Code (engram-lite entry added)`
   - Skipped: `IDE MCP config unchanged: Cursor (engram-lite already configured)`

## Acceptance Criteria

- [x] Fresh workspace → data dir, `config.json`, and IDE configs for all detected IDEs are created
- [x] Re-run on already-initialized workspace → data dir and config.json steps are skipped, IDE config step still runs and reports correct per-IDE status
- [x] Workspace with existing `mcp.json` that already has engram-lite entry → init completes, IDE config reported as unchanged
- [x] Workspace with existing `mcp.json` that has other servers but no engram-lite → entry is merged, other servers preserved
- [x] Two separate workspaces initialized independently → each IDE config points to its own `ENGRAM_DATA_DIR`
- [x] Covered by cmdInit integration tests using `t.TempDir()`

## Implementation Notes

- Extracted `runInit(workspaceRoot string, prompter ide.Prompter) error` from `cmdInit`; `cmdInit` is now a thin wrapper calling `runInit(detectProjectRoot(), &terminalPrompter{})`
- Removed the early-return guard (`if dirExists && configExists { return }`) — replaced with `alreadyInit` flag that skips setup but always falls through to the IDE config step
- Old IDE config loop (unconditional dir creation + file-skip-if-exists) replaced with `ide.WriteWorkspaceConfigs(workspaceRoot, dataDir, prompter)`
- `terminalPrompter` uses `bufio.Scanner` over stdin; presents numbered list, parses space-separated input
- JSON fixture in `TestCmdInit__should_report_unchanged` uses `json.MarshalIndent` (not raw string concat) to avoid Windows path backslash escaping issues

## Blocked By

- 001-01
- 001-02

## Type

AFK

## User Stories Covered

- US 1: Windsurf workspace → correct config generated on init
- US 2: VS Code and Cursor → same
- US 6: Re-running init after adding an IDE → configs are evaluated and updated
- US 7: Two workspaces → isolated configs
