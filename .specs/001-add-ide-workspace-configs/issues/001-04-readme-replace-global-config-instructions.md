# 001-04 - README: replace global config instructions

## What to Build

Update README.md so that the setup instructions for Windsurf, VS Code, and Cursor
direct the user to run `engram-lite init` instead of manually editing a global
MCP config file.

The global config approach (editing `~/.codeium/windsurf/mcp_config.json` or
equivalent) must be removed — it produces a server with no workspace context and
is the root cause of the wrong-project-detection bug.

The updated section should:
- State that `engram-lite init` (already documented as Step 2) handles IDE setup
  automatically for all supported IDEs
- List Windsurf, VS Code, and Cursor as supported
- Note that the IDE must be restarted after running `init` to pick up the new config
- Keep Claude Code's existing instructions unchanged (plugin system, no init needed)

## Acceptance Criteria

- [x] Global MCP config file paths and JSON snippets for Windsurf, VS Code, and Cursor are removed from the README
- [x] The Connect section explains that `engram-lite init` generates workspace-level IDE configs automatically
- [x] Supported IDEs (Windsurf, VS Code, Cursor) are listed
- [x] IDE restart requirement after `init` is mentioned
- [x] Claude Code setup instructions are unchanged

## Blocked By

- 001-03

## Type

AFK

## User Stories Covered

- US 8: New user reads README → directed to `engram-lite init`, not manual global config
