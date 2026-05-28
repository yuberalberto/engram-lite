package ide

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ResultKind string

const (
	ResultCreated ResultKind = "created"
	ResultMerged  ResultKind = "merged"
	ResultSkipped ResultKind = "skipped"
	ResultError   ResultKind = "error"
)

type Result struct {
	IDE  string
	Kind ResultKind
	Err  error
}

var supportedIDEs = []struct {
	name string
	dir  string
}{
	{"Windsurf", ".windsurf"},
	{"VS Code", ".vscode"},
	{"Cursor", ".cursor"},
}

// Prompter is called when no IDE directories are detected in the workspace.
// SelectIDEs receives the list of supported IDE names and returns the ones
// the user chose. Return an empty slice to skip config generation.
type Prompter interface {
	SelectIDEs(available []string) ([]string, error)
}

// WriteWorkspaceConfigs detects IDEs present in workspaceRoot and writes or
// merges a workspace-level MCP config for each. When no IDEs are detected and
// prompter is non-nil, it calls prompter.SelectIDEs so the user can choose.
// Returns one Result per configured IDE.
func WriteWorkspaceConfigs(workspaceRoot, dataDir string, prompter Prompter) []Result {
	var results []Result
	for _, ide := range supportedIDEs {
		ideDir := filepath.Join(workspaceRoot, ide.dir)
		if _, err := os.Stat(ideDir); os.IsNotExist(err) {
			continue
		}
		results = append(results, writeConfig(ide.name, ideDir, dataDir))
	}

	if len(results) == 0 && prompter != nil {
		names := make([]string, len(supportedIDEs))
		for i, ide := range supportedIDEs {
			names[i] = ide.name
		}
		selected, err := prompter.SelectIDEs(names)
		if err != nil || len(selected) == 0 {
			return results
		}
		dirByName := make(map[string]string, len(supportedIDEs))
		for _, ide := range supportedIDEs {
			dirByName[ide.name] = ide.dir
		}
		for _, name := range selected {
			dir, ok := dirByName[name]
			if !ok {
				continue
			}
			ideDir := filepath.Join(workspaceRoot, dir)
			if err := os.MkdirAll(ideDir, 0o755); err != nil {
				results = append(results, Result{IDE: name, Kind: ResultError, Err: err})
				continue
			}
			results = append(results, writeConfig(name, ideDir, dataDir))
		}
	}

	return results
}

func writeConfig(ideName, ideDir, dataDir string) Result {
	cfgPath := filepath.Join(ideDir, "mcp.json")

	entry := map[string]any{
		"command": "engram-lite",
		"args":    []string{"mcp", "--tools=agent"},
		"env": map[string]any{
			"ENGRAM_DATA_DIR": dataDir,
		},
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return createConfig(ideName, cfgPath, entry)
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Result{IDE: ideName, Kind: ResultError, Err: err}
	}

	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		cfg["mcpServers"] = servers
	}
	if _, exists := servers["engram-lite"]; exists {
		return Result{IDE: ideName, Kind: ResultSkipped}
	}

	servers["engram-lite"] = entry
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return Result{IDE: ideName, Kind: ResultError, Err: err}
	}
	if err := os.WriteFile(cfgPath, out, 0o644); err != nil {
		return Result{IDE: ideName, Kind: ResultError, Err: err}
	}
	return Result{IDE: ideName, Kind: ResultMerged}
}

func createConfig(ideName, cfgPath string, entry map[string]any) Result {
	cfg := map[string]any{
		"mcpServers": map[string]any{
			"engram-lite": entry,
		},
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return Result{IDE: ideName, Kind: ResultError, Err: err}
	}
	if err := os.WriteFile(cfgPath, out, 0o644); err != nil {
		return Result{IDE: ideName, Kind: ResultError, Err: err}
	}
	return Result{IDE: ideName, Kind: ResultCreated}
}

