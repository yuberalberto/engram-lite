# 001-02 - IDEConfigWriter: interactive prompt

## What to Build

Extend IDEConfigWriter with a fallback path for when no IDE config directories
are detected in the workspace root.

When detection finds zero IDEs, the module presents a multi-select prompt listing
the three supported IDEs (Windsurf, VS Code, Cursor). The user selects one or
more. For each selected IDE, the module creates the IDE config directory and
writes the `mcp.json` with the engram-lite entry.

The prompt is injected as an interface so it can be replaced in tests with a
deterministic fake — no stdin interaction required in the test suite.

If the user selects nothing (empty selection or cancels), the module exits cleanly
with a message explaining how to re-run init after opening the workspace in an IDE.

## Acceptance Criteria

- [x] When no IDE dirs are detected, the interactive prompt is shown listing Windsurf, VS Code, and Cursor
- [x] Selecting one or more IDEs creates their config directories and writes correct `mcp.json` files
- [x] Selecting nothing (empty selection) exits cleanly with an actionable message — no files created
- [x] The prompt is injected as an interface; tests use a fake implementation and do not touch stdin
- [x] When IDEs are detected (covered by 001-01), the prompt is never shown

## Implementation Notes

- `Prompter` interface added to `internal/ide` package: `SelectIDEs(available []string) ([]string, error)`
- `WriteWorkspaceConfigs` updated to accept `prompter Prompter` as third arg; nil = no prompt
- Prompt only triggers when zero IDE dirs detected; creates IDE dirs before writing configs
- `fakePrompter` in tests covers all prompt paths without stdin interaction

## Blocked By

- 001-01

## Type

AFK

## User Stories Covered

- US 3: No IDEs detected → interactive prompt lets user configure anyway
