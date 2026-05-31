package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuberalberto/engram-lite/internal/ide"
	"github.com/yuberalberto/engram-lite/internal/project"
	"github.com/yuberalberto/engram-lite/internal/store"
)

type workspaceInitStatus string

const (
	workspaceNotInitialized workspaceInitStatus = "not_initialized"
	workspacePartial        workspaceInitStatus = "partial"
	workspaceNeedsUpdate    workspaceInitStatus = "needs_update_init"
	workspaceConfigured     workspaceInitStatus = "configured"
)

type workspaceInitEvaluation struct {
	Status     workspaceInitStatus
	Root       string
	DataDir    string
	ConfigPath string
	DBPath     string
	Project    string
	Reasons    []string
}

func runInit(workspaceRoot string, prompter ide.Prompter) error {
	eval := evaluateWorkspaceInit(workspaceRoot)
	if eval.Status != workspaceNotInitialized {
		printInitNoop(eval)
		return nil
	}

	result, err := initializeWorkspaceBase(workspaceRoot, true)
	if err != nil {
		return err
	}

	fmt.Println("Workspace initialized.")
	fmt.Println()
	fmt.Println("Status: initialized")
	fmt.Println()
	printResultGroup("Created", result.Created)
	printResultGroup("Verified", result.Verified)
	fmt.Println("Backup:")
	fmt.Println("  ~/.engram-lite/backups/" + result.Project + "/")
	fmt.Println()

	return runUpdateInit(workspaceRoot, prompter)
}

func runUpdateInit(workspaceRoot string, prompter ide.Prompter) error {
	eval := evaluateWorkspaceInit(workspaceRoot)
	switch eval.Status {
	case workspaceNotInitialized:
		fmt.Println("Workspace is not initialized.")
		fmt.Println()
		fmt.Println("No files changed.")
		fmt.Println()
		fmt.Println("To initialize this workspace, run:")
		fmt.Println("  engram-lite init")
		return nil
	case workspacePartial:
		fmt.Println("Workspace init is incomplete.")
		fmt.Println()
		fmt.Println("Status: partial")
		printReasons(eval.Reasons)
		fmt.Println("No files changed.")
		fmt.Println()
		fmt.Println("To repair base workspace metadata, run:")
		fmt.Println("  engram-lite repair-init")
		return nil
	}

	results := writeGeneratedWorkspaceFiles(workspaceRoot, eval.DataDir, prompter)
	fmt.Println("Workspace generated files refreshed.")
	fmt.Println()
	printResultGroup("Created", results.Created)
	printResultGroup("Updated", results.Updated)
	printResultGroup("Unchanged", results.Unchanged)
	printResultGroup("Errors", results.Errors)
	fmt.Println("Next:")
	fmt.Println("  Restart your IDE or agent.")
	return nil
}

func runRepairInit(workspaceRoot string) error {
	eval := evaluateWorkspaceInit(workspaceRoot)
	if eval.Status == workspaceNotInitialized {
		fmt.Println("Workspace is not initialized.")
		fmt.Println()
		fmt.Println("No files changed.")
		fmt.Println()
		fmt.Println("To initialize this workspace, run:")
		fmt.Println("  engram-lite init")
		return nil
	}

	for _, reason := range eval.Reasons {
		if strings.Contains(reason, "config.json is invalid") {
			fmt.Println("Workspace init cannot be repaired automatically.")
			fmt.Println()
			fmt.Println("Status: partial")
			printReasons(eval.Reasons)
			fmt.Println("No files changed.")
			fmt.Println()
			fmt.Println("Fix .engram-lite/config.json or move it aside, then rerun:")
			fmt.Println("  engram-lite repair-init")
			return nil
		}
	}

	result, err := initializeWorkspaceBase(workspaceRoot, true)
	if err != nil {
		return err
	}

	fmt.Println("Workspace init repaired.")
	fmt.Println()
	printResultGroup("Recreated", result.Created)
	printResultGroup("Verified", result.Verified)
	return nil
}

type workspaceBaseResult struct {
	Project  string
	Created  []string
	Verified []string
}

func initializeWorkspaceBase(workspaceRoot string, createMissingConfig bool) (workspaceBaseResult, error) {
	dataDir := filepath.Join(workspaceRoot, ".engram-lite")
	configPath := filepath.Join(dataDir, "config.json")
	dbPath := filepath.Join(dataDir, "engram.db")
	result := workspaceBaseResult{}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return result, fmt.Errorf("engram-lite init: create directory: %w", err)
	}
	if _, err := os.Stat(dataDir); err == nil {
		result.Verified = append(result.Verified, ".engram-lite/")
	}

	projectName, err := readProjectName(configPath)
	if err != nil {
		if !os.IsNotExist(err) || !createMissingConfig {
			return result, fmt.Errorf("engram-lite init: read config: %w", err)
		}
		projectName = defaultWorkspaceProject(workspaceRoot)
		config := map[string]string{"project_name": projectName}
		configBytes, _ := json.MarshalIndent(config, "", "  ")
		if err := os.WriteFile(configPath, configBytes, 0o644); err != nil {
			return result, fmt.Errorf("engram-lite init: write config: %w", err)
		}
		result.Created = append(result.Created, ".engram-lite/config.json")
	} else {
		result.Verified = append(result.Verified, ".engram-lite/config.json")
	}
	result.Project = projectName

	dbExists := fileExists(dbPath)
	cfg := store.FallbackConfig(dataDir)
	db, err := store.New(cfg)
	if err != nil {
		return result, fmt.Errorf("engram-lite init: create database: %w", err)
	}
	db.Close()
	if dbExists {
		result.Verified = append(result.Verified, ".engram-lite/engram.db")
	} else {
		result.Created = append(result.Created, ".engram-lite/engram.db")
	}

	gitignoreChanged, err := ensureGitignore(workspaceRoot)
	if err != nil {
		return result, fmt.Errorf("engram-lite init: update .gitignore: %w", err)
	}
	if gitignoreChanged {
		result.Created = append(result.Created, ".gitignore database exclusions")
	} else {
		result.Verified = append(result.Verified, ".gitignore database exclusions")
	}
	return result, nil
}

type generatedFileResults struct {
	Created   []string
	Updated   []string
	Unchanged []string
	Errors    []string
}

func writeGeneratedWorkspaceFiles(workspaceRoot, dataDir string, prompter ide.Prompter) generatedFileResults {
	out := generatedFileResults{}
	results := ide.WriteWorkspaceConfigs(workspaceRoot, dataDir, prompter)
	for _, r := range results {
		path := ideConfigDisplayPath(r.IDE)
		switch r.Kind {
		case ide.ResultCreated:
			out.Created = append(out.Created, path)
		case ide.ResultMerged:
			out.Updated = append(out.Updated, path+" (engram-lite entry added)")
		case ide.ResultSkipped:
			out.Unchanged = append(out.Unchanged, path+" (engram-lite already configured)")
		case ide.ResultError:
			out.Errors = append(out.Errors, path+": "+r.Err.Error())
		}
	}

	switch cr := ide.WriteCodexConfig(workspaceRoot, dataDir); cr.Kind {
	case ide.ResultCreated:
		out.Created = append(out.Created, ".codex/config.toml")
	case ide.ResultMerged:
		out.Updated = append(out.Updated, ".codex/config.toml (engram-lite entry added)")
	case ide.ResultSkipped:
		out.Unchanged = append(out.Unchanged, ".codex/config.toml (engram-lite already configured)")
	case ide.ResultError:
		out.Errors = append(out.Errors, ".codex/config.toml: "+cr.Err.Error())
	}

	switch wr := ide.WriteWindsurfRule(workspaceRoot); wr.Kind {
	case ide.ResultCreated:
		out.Created = append(out.Created, ".windsurf/rules.md")
	case ide.ResultMerged:
		out.Updated = append(out.Updated, ".windsurf/rules.md (mem_use_workspace rule appended)")
	case ide.ResultSkipped:
		out.Unchanged = append(out.Unchanged, ".windsurf/rules.md (already configured)")
	case ide.ResultError:
		out.Errors = append(out.Errors, ".windsurf/rules.md: "+wr.Err.Error())
	}
	return out
}

func evaluateWorkspaceInit(workspaceRoot string) workspaceInitEvaluation {
	dataDir := filepath.Join(workspaceRoot, ".engram-lite")
	configPath := filepath.Join(dataDir, "config.json")
	dbPath := filepath.Join(dataDir, "engram.db")
	eval := workspaceInitEvaluation{
		Status:     workspaceNotInitialized,
		Root:       workspaceRoot,
		DataDir:    dataDir,
		ConfigPath: configPath,
		DBPath:     dbPath,
	}

	dataDirExists := dirExists(dataDir)
	configExists := fileExists(configPath)
	if !dataDirExists && !configExists {
		return eval
	}

	projectName, err := readProjectName(configPath)
	if err != nil {
		eval.Status = workspacePartial
		if os.IsNotExist(err) {
			eval.Reasons = append(eval.Reasons, ".engram-lite/config.json is missing")
		} else {
			eval.Reasons = append(eval.Reasons, ".engram-lite/config.json is invalid: "+err.Error())
		}
	} else {
		eval.Project = projectName
	}
	if !fileExists(dbPath) {
		eval.Status = workspacePartial
		eval.Reasons = append(eval.Reasons, ".engram-lite/engram.db is missing")
	}
	if !gitignoreHasDatabaseExclusions(workspaceRoot) {
		eval.Status = workspacePartial
		eval.Reasons = append(eval.Reasons, ".gitignore database exclusions are missing")
	}
	if eval.Status == workspacePartial {
		return eval
	}

	generatedReasons := missingGeneratedWorkspaceFiles(workspaceRoot)
	if len(generatedReasons) > 0 {
		eval.Status = workspaceNeedsUpdate
		eval.Reasons = generatedReasons
		return eval
	}
	eval.Status = workspaceConfigured
	return eval
}

func missingGeneratedWorkspaceFiles(workspaceRoot string) []string {
	var reasons []string
	if !fileContains(filepath.Join(workspaceRoot, ".codex", "config.toml"), `[mcp_servers."engram-lite"]`) &&
		!fileContains(filepath.Join(workspaceRoot, ".codex", "config.toml"), "[mcp_servers.engram-lite]") {
		reasons = append(reasons, ".codex/config.toml is missing the engram-lite MCP server")
	}
	for _, cfg := range []struct {
		dir  string
		file string
		name string
	}{
		{".windsurf", ".windsurf/mcp.json", "Windsurf"},
		{".vscode", ".vscode/mcp.json", "VS Code"},
		{".cursor", ".cursor/mcp.json", "Cursor"},
	} {
		if !dirExists(filepath.Join(workspaceRoot, cfg.dir)) {
			continue
		}
		path := filepath.Join(workspaceRoot, filepath.FromSlash(cfg.file))
		data, err := os.ReadFile(path)
		if err != nil {
			reasons = append(reasons, cfg.file+" is missing")
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			reasons = append(reasons, cfg.file+" is malformed")
			continue
		}
		servers, _ := parsed["mcpServers"].(map[string]any)
		if servers == nil {
			reasons = append(reasons, cfg.file+" has no mcpServers table")
			continue
		}
		if _, ok := servers["engram-lite"]; !ok {
			reasons = append(reasons, cfg.file+" is missing the engram-lite MCP server")
		}
	}
	if !fileContains(filepath.Join(workspaceRoot, ".windsurf", "rules.md"), "mem_use_workspace") {
		reasons = append(reasons, ".windsurf/rules.md is missing the mem_use_workspace rule")
	}
	return reasons
}

func printInitNoop(eval workspaceInitEvaluation) {
	switch eval.Status {
	case workspacePartial:
		fmt.Println("Workspace init is incomplete.")
		fmt.Println()
		fmt.Println("Status: partial")
		printReasons(eval.Reasons)
		fmt.Println("No files changed.")
		fmt.Println()
		fmt.Println("To repair base workspace metadata, run:")
		fmt.Println("  engram-lite repair-init")
	case workspaceNeedsUpdate:
		fmt.Println("Workspace already initialized.")
		fmt.Println()
		fmt.Println("Status: needs_update_init")
		printReasons(eval.Reasons)
		fmt.Println("No files changed.")
		fmt.Println()
		fmt.Println("To refresh generated workspace files after upgrading engram-lite, run:")
		fmt.Println("  engram-lite update-init")
	default:
		fmt.Println("Workspace already initialized.")
		fmt.Println()
		fmt.Println("Status: initialized")
		fmt.Println("No files changed.")
		fmt.Println()
		fmt.Println("To refresh generated workspace files after upgrading engram-lite, run:")
		fmt.Println("  engram-lite update-init")
	}
}

func printReasons(reasons []string) {
	if len(reasons) == 0 {
		return
	}
	fmt.Println("Reason:")
	for _, reason := range reasons {
		fmt.Println("  - " + reason)
	}
	fmt.Println()
}

func printResultGroup(title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Println(title + ":")
	for _, item := range items {
		fmt.Println("  " + item)
	}
	fmt.Println()
}

func readProjectName(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	var cfg map[string]string
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	projectName := strings.TrimSpace(cfg["project_name"])
	if projectName == "" {
		return "", fmt.Errorf("project_name is missing")
	}
	return projectName, nil
}

func defaultWorkspaceProject(workspaceRoot string) string {
	projectName := filepath.Base(workspaceRoot)
	result := project.DetectProjectFull(workspaceRoot)
	if result.Project != "" {
		projectName = result.Project
	}
	return projectName
}

func ensureGitignore(workspaceRoot string) (bool, error) {
	gitignorePath := filepath.Join(workspaceRoot, ".gitignore")
	if gitignoreHasDatabaseExclusions(workspaceRoot) {
		return false, nil
	}
	entry := "# engram-lite database (config.json is safe to commit)\n.engram-lite/*.db\n.engram-lite/*.db-wal\n.engram-lite/*.db-shm\n"
	if content, err := os.ReadFile(gitignorePath); err == nil {
		prefix := "\n"
		if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
			prefix = "\n\n"
		}
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return false, err
		}
		defer f.Close()
		_, err = f.WriteString(prefix + entry)
		return err == nil, err
	}
	if err := os.WriteFile(gitignorePath, []byte(entry), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func gitignoreHasDatabaseExclusions(workspaceRoot string) bool {
	data, err := os.ReadFile(filepath.Join(workspaceRoot, ".gitignore"))
	return err == nil && strings.Contains(string(data), ".engram-lite/*.db")
}

func ideConfigDisplayPath(name string) string {
	switch name {
	case "Windsurf":
		return ".windsurf/mcp.json"
	case "VS Code":
		return ".vscode/mcp.json"
	case "Cursor":
		return ".cursor/mcp.json"
	default:
		return name
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileContains(path, needle string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(data), needle)
}
