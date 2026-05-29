package ide_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuberalberto/engram-lite/internal/ide"
)

// ── WindsurfRule tests ────────────────────────────────────────────────────────

func TestWindsurfRuleInjection__should_create_rules_file__when_windsurf_dir_exists_but_no_rules_md(t *testing.T) {
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".windsurf"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := ide.WriteWindsurfRule(ws)

	if result.Kind != ide.ResultCreated || result.Err != nil {
		t.Fatalf("expected ResultCreated, got Kind=%q Err=%v", result.Kind, result.Err)
	}
	data, err := os.ReadFile(filepath.Join(ws, ".windsurf", "rules.md"))
	if err != nil {
		t.Fatalf("rules.md not created: %v", err)
	}
	if !strings.Contains(string(data), "mem_use_workspace") {
		t.Error("rules.md does not mention mem_use_workspace")
	}
}

func TestWindsurfRuleInjection__should_append_rule__when_rules_md_exists_without_rule(t *testing.T) {
	ws := t.TempDir()
	wsDir := filepath.Join(ws, ".windsurf")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "# My project rules\n\nAlways write tests first.\n"
	rulesPath := filepath.Join(wsDir, "rules.md")
	if err := os.WriteFile(rulesPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ide.WriteWindsurfRule(ws)

	if result.Kind != ide.ResultMerged || result.Err != nil {
		t.Fatalf("expected ResultMerged, got Kind=%q Err=%v", result.Kind, result.Err)
	}
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Always write tests first.") {
		t.Error("existing content was removed — append must preserve it")
	}
	if !strings.Contains(content, "mem_use_workspace") {
		t.Error("appended content does not mention mem_use_workspace")
	}
}

func TestWindsurfRuleInjection__should_create_rules_file__when_windsurf_dir_absent(t *testing.T) {
	ws := t.TempDir()

	result := ide.WriteWindsurfRule(ws)

	if result.Kind != ide.ResultCreated || result.Err != nil {
		t.Fatalf("expected ResultCreated, got Kind=%q Err=%v", result.Kind, result.Err)
	}
	rulesPath := filepath.Join(ws, ".windsurf", "rules.md")
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("rules.md not created: %v", err)
	}
	if !strings.Contains(string(data), "mem_use_workspace") {
		t.Error("rules.md does not mention mem_use_workspace")
	}
}

func TestWindsurfRuleInjection__should_skip__when_rule_already_present(t *testing.T) {
	ws := t.TempDir()
	wsDir := filepath.Join(ws, ".windsurf")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rulesPath := filepath.Join(wsDir, "rules.md")
	original := "## engram-lite workspace binding\n\nCall mem_use_workspace at session start.\n"
	if err := os.WriteFile(rulesPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ide.WriteWindsurfRule(ws)

	if result.Kind != ide.ResultSkipped || result.Err != nil {
		t.Fatalf("expected ResultSkipped, got Kind=%q Err=%v", result.Kind, result.Err)
	}
	after, _ := os.ReadFile(rulesPath)
	if string(after) != original {
		t.Error("file was modified — skip must leave rules.md byte-for-byte identical")
	}
}
