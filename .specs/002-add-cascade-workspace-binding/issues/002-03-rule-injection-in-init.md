# 002-03 - Rule Injection in engram-lite init

## What to Build

Extend `engram-lite init` to write a Windsurf-specific AI rule file (`.windsurf/rules.md`) at the workspace root. The rule instructs Cascade to call `mem_use_workspace` with the current workspace path before invoking any other engram tool in a session.

The operation is idempotent: if `.windsurf/rules.md` already exists and already contains the engram-lite rule, it is left unchanged. If the file exists but lacks the rule, the rule is appended. If the file does not exist, it is created.

`engram-lite init` reports the outcome using the same result kinds used for IDE MCP configs (`ResultCreated`, `ResultMerged`, `ResultSkipped`, `ResultError`).

## Acceptance Criteria

- [x] Running `engram-lite init` in a workspace creates `.windsurf/rules.md` containing the `mem_use_workspace` rule when the file does not exist.
- [x] Running `engram-lite init` in a workspace appends the rule when `.windsurf/rules.md` exists but does not contain it.
- [x] Running `engram-lite init` a second time leaves `.windsurf/rules.md` unchanged when the rule is already present (idempotent).
- [x] The rule text names `mem_use_workspace` and instructs Cascade to call it with the workspace path at session start.
- [x] Init output reports the correct result kind for each case (created / updated / unchanged).

## Implementation Notes

- New public function `ide.WriteWindsurfRule(workspaceRoot string) ide.Result` in `internal/ide/windsurf_rules.go`.
- Reuses existing `ResultKind` enum (`ResultCreated` / `ResultMerged` / `ResultSkipped`).
- Idempotency sentinel: presence of the string `"mem_use_workspace"` in the file content.
- Creates `.windsurf/` directory when absent; does not require the MCP config step to have run first.
- Called from `runInit()` in `cmd/engram-lite/main.go` after `WriteWorkspaceConfigs`.

## Blocked By

None — can start immediately.

## Type

AFK

## User Stories Covered

- US3: As a developer using Windsurf, I want `engram-lite init` to configure the workspace binding rule automatically
