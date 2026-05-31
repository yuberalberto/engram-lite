---
name: engram-lite-memory
description: "Project-local persistent memory protocol for Codex. Use when engram-lite MCP tools are available or when working in a repository initialized with .engram-lite."
---

# engram-lite Memory

engram-lite stores memory in the current workspace at `.engram-lite/engram.db`.
Codex supports project-level MCP configuration in `.codex/config.toml`; the
workspace init step pins `ENGRAM_DATA_DIR` to the current workspace.

## Required Startup

At the start of a coding session, call:

1. `mem_current_project` to verify the active project.
2. `mem_context` for the detected project.

If the detected project is wrong, tell the user to rerun `engram-lite init` from
the workspace root and restart Codex. Do not use global Engram tools as a
substitute for engram-lite.

## During Work

Save durable project knowledge proactively:

- Architectural decisions and tradeoffs.
- Bug fixes: what was wrong, why, and where it changed.
- New patterns, conventions, or configuration.
- Important discoveries and gotchas.

Prefer `mem_save` for structured observations and `mem_session_summary` at the
end of substantial work.

## Important

- Do not use global Engram tools as a substitute for engram-lite.
- Do not rely on hooks for startup or shutdown behavior in Codex.
- `mem_use_workspace` is a compatibility workaround for hosts without reliable
  workspace MCP config. It is not the normal Codex path.
