package ide

import (
	"os"
	"path/filepath"
	"strings"
)

const windsurfRulesSentinel = "mem_use_workspace"

const windsurfRulesBlock = `## engram-lite workspace binding

At the start of every session, call mem_use_workspace with the current workspace
path before invoking any other engram tool.

Example:
  mem_use_workspace(workspace_path: "/path/to/your/project")

This scopes all memory operations to the correct workspace.
`

// WriteWindsurfRule writes the engram-lite workspace-binding rule to
// .windsurf/rules.md. It creates the file (and directory) when absent,
// appends the rule when the file exists without it, and skips the file when
// the rule is already present.
func WriteWindsurfRule(workspaceRoot string) Result {
	rulesDir := filepath.Join(workspaceRoot, ".windsurf")
	rulesPath := filepath.Join(rulesDir, "rules.md")

	data, err := os.ReadFile(rulesPath)
	if err != nil {
		if err := os.MkdirAll(rulesDir, 0o755); err != nil {
			return Result{IDE: "Windsurf", Kind: ResultError, Err: err}
		}
		if err := os.WriteFile(rulesPath, []byte(windsurfRulesBlock), 0o644); err != nil {
			return Result{IDE: "Windsurf", Kind: ResultError, Err: err}
		}
		return Result{IDE: "Windsurf", Kind: ResultCreated}
	}

	if strings.Contains(string(data), windsurfRulesSentinel) {
		return Result{IDE: "Windsurf", Kind: ResultSkipped}
	}

	f, err := os.OpenFile(rulesPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return Result{IDE: "Windsurf", Kind: ResultError, Err: err}
	}
	defer f.Close()
	if _, err := f.WriteString("\n" + windsurfRulesBlock); err != nil {
		return Result{IDE: "Windsurf", Kind: ResultError, Err: err}
	}
	return Result{IDE: "Windsurf", Kind: ResultMerged}
}
