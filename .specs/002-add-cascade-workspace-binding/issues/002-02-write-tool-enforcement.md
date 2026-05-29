# 002-02 - Write Tool Enforcement

## What to Build

When all three steps of the project resolution fallback chain fail (process-level override, CWD detection, and session workspace), write tools must return a clear, actionable error rather than silently falling through or writing to the wrong project.

The error message must instruct the caller to invoke `mem_use_workspace` first:

> `project could not be determined — call mem_use_workspace(<workspace_path>) before writing`

Read tools remain permissive and do not enforce this requirement.

## Acceptance Criteria

- [x] A `mem_save` call made before `mem_use_workspace` (with no CWD project and no process override) returns IsError=true with the enforcement message.
- [x] The enforcement message names `mem_use_workspace` explicitly.
- [x] After calling `mem_use_workspace` successfully, the same `mem_save` call succeeds.
- [x] Read tools (`mem_search`, `mem_context`, `mem_get_observation`) are unaffected — they do not enforce workspace context.
- [x] Existing tests for write tools under normal CWD detection continue to pass.

## Implementation Notes

- Added `noProjectContextError` type with message: `project could not be determined — call mem_use_workspace(<workspace_path>) before writing`
- Added `resolveWriteProjectWithEnforcement(activity)` — checks workspace binding first, then git-based CWD; returns `noProjectContextError` when only `dir_basename` detection is available
- Updated all write handlers (`mem_save`, `mem_save_prompt`, `mem_session_summary`, `mem_session_start`, `mem_capture_passive`) to use enforcement resolution; `mem_session_end` retains existing tolerance intentionally
- `writeProjectErrorResult` handles `no_project_context` error code
- Read tools (`mem_search`, `mem_context`, `mem_get_observation`) unchanged — enforcement is write-only

## Blocked By

- 002-01 (session workspace context must exist before enforcement can check it)

## Type

AFK

## User Stories Covered

- US4: As Cascade, I want a clear error when I attempt to write without a workspace context
