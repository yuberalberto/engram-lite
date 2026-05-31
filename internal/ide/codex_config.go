package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const codexServerTable = `[mcp_servers."engram-lite"]`

// WriteCodexConfig writes a workspace-level Codex MCP config to
// .codex/config.toml. Codex supports project config, so this pins the
// engram-lite MCP server to the workspace data dir without requiring
// mem_use_workspace.
func WriteCodexConfig(workspaceRoot, dataDir string) Result {
	codexDir := filepath.Join(workspaceRoot, ".codex")
	cfgPath := filepath.Join(codexDir, "config.toml")

	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		return Result{IDE: "Codex", Kind: ResultError, Err: err}
	}

	block := codexConfigBlock(workspaceRoot, dataDir)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if err := os.WriteFile(cfgPath, []byte(block), 0o644); err != nil {
			return Result{IDE: "Codex", Kind: ResultError, Err: err}
		}
		return Result{IDE: "Codex", Kind: ResultCreated}
	}

	content := string(data)
	if strings.Contains(content, codexServerTable) || strings.Contains(content, "[mcp_servers.engram-lite]") {
		return Result{IDE: "Codex", Kind: ResultSkipped}
	}

	sep := "\n"
	if strings.TrimSpace(content) != "" && !strings.HasSuffix(content, "\n") {
		sep = "\n\n"
	}
	out := content + sep + block
	if err := os.WriteFile(cfgPath, []byte(out), 0o644); err != nil {
		return Result{IDE: "Codex", Kind: ResultError, Err: err}
	}
	return Result{IDE: "Codex", Kind: ResultMerged}
}

func codexConfigBlock(workspaceRoot, dataDir string) string {
	return fmt.Sprintf(`[mcp_servers."engram-lite"]
command = "engram-lite"
args = ["mcp", "--tools=agent"]
cwd = %q
enabled = true

[mcp_servers."engram-lite".env]
ENGRAM_DATA_DIR = %q
`, filepath.ToSlash(workspaceRoot), filepath.ToSlash(dataDir))
}
