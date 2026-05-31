package ide_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuberalberto/engram-lite/internal/ide"
)

func TestCodexConfigWriter__should_create_project_config__when_absent(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")

	result := ide.WriteCodexConfig(ws, dataDir)

	if result.Kind != ide.ResultCreated || result.Err != nil {
		t.Fatalf("expected ResultCreated, got Kind=%q Err=%v", result.Kind, result.Err)
	}
	data, err := os.ReadFile(filepath.Join(ws, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `[mcp_servers."engram-lite"]`) {
		t.Error("Codex config does not contain engram-lite MCP table")
	}
	if !strings.Contains(content, `ENGRAM_DATA_DIR`) {
		t.Error("Codex config does not pin ENGRAM_DATA_DIR")
	}
	if strings.Contains(content, "mem_use_workspace") {
		t.Error("Codex config must not depend on mem_use_workspace")
	}
}

func TestCodexConfigWriter__should_append_server__when_config_exists_without_engram(t *testing.T) {
	ws := t.TempDir()
	dataDir := filepath.Join(ws, ".engram-lite")
	codexDir := filepath.Join(ws, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "model = \"gpt-5\"\n"
	cfgPath := filepath.Join(codexDir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ide.WriteCodexConfig(ws, dataDir)

	if result.Kind != ide.ResultMerged || result.Err != nil {
		t.Fatalf("expected ResultMerged, got Kind=%q Err=%v", result.Kind, result.Err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, existing) {
		t.Error("existing Codex config was removed")
	}
	if !strings.Contains(content, `[mcp_servers."engram-lite"]`) {
		t.Error("engram-lite MCP table was not appended")
	}
}

func TestCodexConfigWriter__should_skip__when_engram_already_configured(t *testing.T) {
	ws := t.TempDir()
	codexDir := filepath.Join(ws, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	original := "[mcp_servers.\"engram-lite\"]\ncommand = \"custom\"\n"
	cfgPath := filepath.Join(codexDir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ide.WriteCodexConfig(ws, filepath.Join(ws, ".engram-lite"))

	if result.Kind != ide.ResultSkipped || result.Err != nil {
		t.Fatalf("expected ResultSkipped, got Kind=%q Err=%v", result.Kind, result.Err)
	}
	after, _ := os.ReadFile(cfgPath)
	if string(after) != original {
		t.Error("config.toml was modified despite existing engram-lite entry")
	}
}
