# 002-01 - mem_use_workspace Tool — Core

## What to Build

Add a `mem_use_workspace` MCP tool to the engram-lite server. When called with an absolute workspace path, it reads `<workspace>/.engram-lite/config.json` to resolve the project name and data directory, then stores this as connection-scoped context for the lifetime of the stdio session.

Update the project resolution fallback chain so that all write and read handlers check the session workspace at step 3, after process-level override (step 1) and CWD detection (step 2). Claude Code always resolves at step 2 and is unaffected.

The tool is idempotent — calling it again within the same session updates the active workspace.

## Acceptance Criteria

- [x] `mem_use_workspace` is registered as an MCP tool with a required `workspace_path` string argument.
- [x] Calling it with a valid workspace path (containing `.engram-lite/config.json`) resolves the project name and returns it in the response.
- [x] Calling it with a path missing `config.json` returns a descriptive error (IsError=true).
- [x] After a successful call, subsequent `mem_save` calls target the project resolved from the workspace, not the CWD project.
- [x] Calling it twice with different paths updates the session context to the second workspace.
- [x] CWD-based resolution (step 2) is unchanged — existing Claude Code behavior is unaffected.

## Implementation Notes

- Workspace binding state lives in `SessionActivity.workspaceResult` — the existing per-connection state tracker, already threaded through all write handlers.
- `projectpkg.DetectProjectFull(workspacePath)` validates config existence; only `SourceConfig` results are accepted (fails loudly if `.engram-lite/config.json` is missing or malformed).
- Resolution precedence: (1) process override (`cfg.DefaultProject`), (2) workspace binding (`mem_use_workspace`), (3) CWD detection. CWD is not checked when a workspace binding is active.
- The tool is registered in `ProfileAgent` (now 16 agent tools, 20 total) with `DeferLoading=true`.

## Blocked By

None — can start immediately.

## Type

AFK

## User Stories Covered

- US1: As Cascade, I want to call `mem_use_workspace` with the current workspace path
- US2: As a developer using Windsurf, I want observations to go to the active project's database
- US5: As a Claude Code user, I want CWD-based detection to remain unchanged
- US6: As a developer, I want misconfigured paths to fail loudly
