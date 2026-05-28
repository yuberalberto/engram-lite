package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// readMCPDataDir parses mcp.json and returns the ENGRAM_DATA_DIR value.
func readMCPDataDir(t *testing.T, mcpPath string) string {
	t.Helper()
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("read mcp.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse mcp.json: %v", err)
	}
	servers, _ := cfg["mcpServers"].(map[string]any)
	entry, _ := servers["engram-lite"].(map[string]any)
	env, _ := entry["env"].(map[string]any)
	v, _ := env["ENGRAM_DATA_DIR"].(string)
	return v
}

func TestCmdInit__should_create_data_dir_and_ide_config__when_fresh_workspace(t *testing.T) {
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".windsurf"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := runInit(ws, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	dataDir := filepath.Join(ws, ".engram-lite")

	if _, err := os.Stat(filepath.Join(dataDir, "config.json")); err != nil {
		t.Errorf("config.json not created: %v", err)
	}

	mcpPath := filepath.Join(ws, ".windsurf", "mcp.json")
	if got := readMCPDataDir(t, mcpPath); got != dataDir {
		t.Errorf("ENGRAM_DATA_DIR = %q; want %q", got, dataDir)
	}
}

func TestCmdInit__should_run_ide_config_step__when_already_initialized(t *testing.T) {
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".windsurf"), 0o755); err != nil {
		t.Fatal(err)
	}

	// First run: full init
	if err := runInit(ws, nil); err != nil {
		t.Fatalf("first runInit: %v", err)
	}

	// Add a new IDE dir after first init to verify re-run picks it up
	if err := os.MkdirAll(filepath.Join(ws, ".vscode"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Second run: data dir already exists, but IDE step must still run
	if err := runInit(ws, nil); err != nil {
		t.Fatalf("second runInit: %v", err)
	}

	dataDir := filepath.Join(ws, ".engram-lite")

	// .vscode/mcp.json must have been created on re-run
	mcpPath := filepath.Join(ws, ".vscode", "mcp.json")
	if got := readMCPDataDir(t, mcpPath); got != dataDir {
		t.Errorf("re-run: .vscode ENGRAM_DATA_DIR = %q; want %q", got, dataDir)
	}
}

func TestCmdInit__should_report_unchanged__when_engram_already_configured(t *testing.T) {
	ws := t.TempDir()
	wsDir := filepath.Join(ws, ".windsurf")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(ws, ".engram-lite")

	// Build valid JSON using json.MarshalIndent so Windows paths are properly escaped
	fixture := map[string]any{
		"mcpServers": map[string]any{
			"engram-lite": map[string]any{
				"command": "engram-lite",
				"env":     map[string]any{"ENGRAM_DATA_DIR": dataDir},
			},
		},
	}
	fixtureBytes, _ := json.MarshalIndent(fixture, "", "  ")
	cfgPath := filepath.Join(wsDir, "mcp.json")
	if err := os.WriteFile(cfgPath, fixtureBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runInit(ws, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// File must be byte-for-byte identical (skipped, not re-written)
	after, _ := os.ReadFile(cfgPath)
	if string(after) != string(fixtureBytes) {
		t.Error("mcp.json was modified — skip must leave file untouched when engram-lite already configured")
	}
}

func TestCmdInit__should_merge_entry__when_mcp_json_has_other_servers(t *testing.T) {
	ws := t.TempDir()
	wsDir := filepath.Join(ws, ".vscode")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{"mcpServers":{"other-server":{"command":"other"}}}`
	cfgPath := filepath.Join(wsDir, "mcp.json")
	if err := os.WriteFile(cfgPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runInit(ws, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read mcp.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse mcp.json: %v", err)
	}
	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["other-server"]; !ok {
		t.Error("other-server was removed — merge must preserve existing entries")
	}
	dataDir := filepath.Join(ws, ".engram-lite")
	if got := readMCPDataDir(t, cfgPath); got != dataDir {
		t.Errorf("ENGRAM_DATA_DIR = %q; want %q", got, dataDir)
	}
}

func TestCmdInit__should_use_isolated_data_dirs__when_two_workspaces(t *testing.T) {
	ws1 := t.TempDir()
	ws2 := t.TempDir()
	for _, ws := range []string{ws1, ws2} {
		if err := os.MkdirAll(filepath.Join(ws, ".windsurf"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := runInit(ws1, nil); err != nil {
		t.Fatalf("runInit ws1: %v", err)
	}
	if err := runInit(ws2, nil); err != nil {
		t.Fatalf("runInit ws2: %v", err)
	}

	dataDir1 := filepath.Join(ws1, ".engram-lite")
	dataDir2 := filepath.Join(ws2, ".engram-lite")

	got1 := readMCPDataDir(t, filepath.Join(ws1, ".windsurf", "mcp.json"))
	got2 := readMCPDataDir(t, filepath.Join(ws2, ".windsurf", "mcp.json"))

	if got1 != dataDir1 {
		t.Errorf("ws1 ENGRAM_DATA_DIR = %q; want %q", got1, dataDir1)
	}
	if got2 != dataDir2 {
		t.Errorf("ws2 ENGRAM_DATA_DIR = %q; want %q", got2, dataDir2)
	}
	if got1 == got2 {
		t.Error("both workspaces have the same ENGRAM_DATA_DIR — they must be isolated")
	}
}
