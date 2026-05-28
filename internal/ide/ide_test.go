package ide_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuberalberto/engram-lite/internal/ide"
)

// readMCPFile parses mcp.json and returns the mcpServers map.
func readMCPFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mcp.json: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse mcp.json: %v", err)
	}
	return out
}

// helpers

func mcpServers(t *testing.T, path string) map[string]any {
	t.Helper()
	parsed := readMCPFile(t, path)
	servers, ok := parsed["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers missing or wrong type in %s", path)
	}
	return servers
}

func engramDataDir(t *testing.T, path string) string {
	t.Helper()
	servers := mcpServers(t, path)
	entry, ok := servers["engram-lite"].(map[string]any)
	if !ok {
		t.Fatalf("engram-lite entry missing in %s", path)
	}
	env, ok := entry["env"].(map[string]any)
	if !ok {
		t.Fatalf("env missing in engram-lite entry")
	}
	v, _ := env["ENGRAM_DATA_DIR"].(string)
	return v
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestIDEConfigWriter__should_create_config__when_ide_dir_exists_and_no_mcp_json(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	if err := os.MkdirAll(filepath.Join(ws, ".windsurf"), 0o755); err != nil {
		t.Fatal(err)
	}

	results := ide.WriteWorkspaceConfigs(ws, dataDir, nil)

	if len(results) != 1 || results[0].Kind != ide.ResultCreated || results[0].Err != nil {
		t.Fatalf("expected [ResultCreated], got %+v", results)
	}
	cfgPath := filepath.Join(ws, ".windsurf", "mcp.json")
	if got := engramDataDir(t, cfgPath); got != dataDir {
		t.Errorf("ENGRAM_DATA_DIR = %q; want %q", got, dataDir)
	}
}

func TestIDEConfigWriter__should_merge_entry__when_mcp_json_exists_without_engram(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	wsDir := filepath.Join(ws, ".windsurf")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{"mcpServers":{"other-server":{"command":"other"}}}`
	if err := os.WriteFile(filepath.Join(wsDir, "mcp.json"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	results := ide.WriteWorkspaceConfigs(ws, dataDir, nil)

	if len(results) != 1 || results[0].Kind != ide.ResultMerged {
		t.Fatalf("expected [ResultMerged], got %+v", results)
	}
	cfgPath := filepath.Join(wsDir, "mcp.json")
	servers := mcpServers(t, cfgPath)
	if _, ok := servers["other-server"]; !ok {
		t.Error("other-server was removed — merge must preserve existing entries")
	}
	if got := engramDataDir(t, cfgPath); got != dataDir {
		t.Errorf("ENGRAM_DATA_DIR = %q; want %q", got, dataDir)
	}
}

func TestIDEConfigWriter__should_skip__when_engram_entry_already_present(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	wsDir := filepath.Join(ws, ".windsurf")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	original := `{"mcpServers":{"engram-lite":{"command":"engram-lite","customKey":"myvalue"}}}`
	cfgPath := filepath.Join(wsDir, "mcp.json")
	if err := os.WriteFile(cfgPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	results := ide.WriteWorkspaceConfigs(ws, dataDir, nil)

	if len(results) != 1 || results[0].Kind != ide.ResultSkipped {
		t.Fatalf("expected [ResultSkipped], got %+v", results)
	}
	after, _ := os.ReadFile(cfgPath)
	if string(after) != original {
		t.Error("file was modified — skip must leave file byte-for-byte identical")
	}
}

func TestIDEConfigWriter__should_return_error__when_mcp_json_is_malformed(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	wsDir := filepath.Join(ws, ".windsurf")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(wsDir, "mcp.json")
	if err := os.WriteFile(cfgPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(cfgPath)

	results := ide.WriteWorkspaceConfigs(ws, dataDir, nil)

	if len(results) != 1 || results[0].Kind != ide.ResultError || results[0].Err == nil {
		t.Fatalf("expected [ResultError], got %+v", results)
	}
	after, _ := os.ReadFile(cfgPath)
	if string(after) != string(original) {
		t.Error("malformed file was modified — must leave file untouched on error")
	}
}

func TestIDEConfigWriter__should_configure_all_detected_ides__when_multiple_present(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	for _, d := range []string{".windsurf", ".vscode", ".cursor"} {
		if err := os.MkdirAll(filepath.Join(ws, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	results := ide.WriteWorkspaceConfigs(ws, dataDir, nil)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Kind != ide.ResultCreated {
			t.Errorf("IDE %q: expected ResultCreated, got %q (err: %v)", r.IDE, r.Kind, r.Err)
		}
	}
	for _, d := range []string{".windsurf", ".vscode", ".cursor"} {
		cfgPath := filepath.Join(ws, d, "mcp.json")
		if got := engramDataDir(t, cfgPath); got != dataDir {
			t.Errorf("%s ENGRAM_DATA_DIR = %q; want %q", d, got, dataDir)
		}
	}
}

func TestIDEConfigWriter__should_return_empty__when_no_ide_dirs_present(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")

	results := ide.WriteWorkspaceConfigs(ws, dataDir, nil)

	if len(results) != 0 {
		t.Errorf("expected no results, got %+v", results)
	}
	for _, d := range []string{".windsurf", ".vscode", ".cursor"} {
		if _, err := os.Stat(filepath.Join(ws, d)); !os.IsNotExist(err) {
			t.Errorf("IDE dir %q was created — detection must not create dirs", d)
		}
	}
}

func TestIDEConfigWriter__should_ignore_absent_ide_dirs__when_only_some_present(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	// Only Cursor present
	if err := os.MkdirAll(filepath.Join(ws, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}

	results := ide.WriteWorkspaceConfigs(ws, dataDir, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].IDE != "Cursor" || results[0].Kind != ide.ResultCreated {
		t.Errorf("expected Cursor/ResultCreated, got %+v", results[0])
	}
	for _, d := range []string{".windsurf", ".vscode"} {
		if _, err := os.Stat(filepath.Join(ws, d)); !os.IsNotExist(err) {
			t.Errorf("IDE dir %q was created unexpectedly", d)
		}
	}
}

// ── prompt tests ──────────────────────────────────────────────────────────────

type fakePrompter struct {
	returns []string
	called  bool
}

func (f *fakePrompter) SelectIDEs(available []string) ([]string, error) {
	f.called = true
	return f.returns, nil
}

func TestIDEConfigWriter__should_not_create_files__when_prompt_returns_empty(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	prompt := &fakePrompter{returns: []string{}}

	results := ide.WriteWorkspaceConfigs(ws, dataDir, prompt)

	if len(results) != 0 {
		t.Errorf("expected no results, got %+v", results)
	}
	for _, d := range []string{".windsurf", ".vscode", ".cursor"} {
		if _, err := os.Stat(filepath.Join(ws, d)); !os.IsNotExist(err) {
			t.Errorf("IDE dir %q was created — empty selection must not create dirs", d)
		}
	}
}

func TestIDEConfigWriter__should_not_call_prompt__when_ides_detected(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	if err := os.MkdirAll(filepath.Join(ws, ".vscode"), 0o755); err != nil {
		t.Fatal(err)
	}
	prompt := &fakePrompter{returns: []string{"Windsurf"}}

	ide.WriteWorkspaceConfigs(ws, dataDir, prompt)

	if prompt.called {
		t.Error("prompter was called despite IDEs being detected")
	}
}

func TestIDEConfigWriter__should_create_configs__when_prompt_selects_multiple(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	prompt := &fakePrompter{returns: []string{"Windsurf", "Cursor"}}

	results := ide.WriteWorkspaceConfigs(ws, dataDir, prompt)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	for _, r := range results {
		if r.Kind != ide.ResultCreated {
			t.Errorf("IDE %q: expected ResultCreated, got %q", r.IDE, r.Kind)
		}
	}
	for _, d := range []string{".windsurf", ".cursor"} {
		cfgPath := filepath.Join(ws, d, "mcp.json")
		if _, err := os.Stat(cfgPath); err != nil {
			t.Errorf("mcp.json missing at %s", cfgPath)
		}
	}
	if _, err := os.Stat(filepath.Join(ws, ".vscode")); !os.IsNotExist(err) {
		t.Error(".vscode dir created but was not selected")
	}
}

func TestIDEConfigWriter__should_create_configs__when_prompt_selects_ides(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	// No IDE dirs present
	prompt := &fakePrompter{returns: []string{"Windsurf"}}

	results := ide.WriteWorkspaceConfigs(ws, dataDir, prompt)

	if !prompt.called {
		t.Error("prompter was not called despite no IDEs detected")
	}
	if len(results) != 1 || results[0].Kind != ide.ResultCreated || results[0].IDE != "Windsurf" {
		t.Fatalf("expected [Windsurf/ResultCreated], got %+v", results)
	}
	cfgPath := filepath.Join(ws, ".windsurf", "mcp.json")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf("mcp.json not created at %s: %v", cfgPath, err)
	}
}
