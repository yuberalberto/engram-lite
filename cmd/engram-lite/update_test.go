package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	versioncheck "github.com/yuberalberto/engram-lite/internal/version"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func withUpdateStubs(t *testing.T, currentVersion string, latest string, installed string) *bool {
	t.Helper()
	oldVersion := version
	oldLatest := latestReleaseVersion
	oldInstall := runGoInstall
	oldReadInstalled := readInstalledVersion

	installCalled := false
	version = currentVersion
	latestReleaseVersion = func() (string, versioncheck.CheckResult) {
		return latest, versioncheck.CheckResult{Status: versioncheck.StatusUpToDate, Latest: latest}
	}
	runGoInstall = func(target string) (string, error) {
		installCalled = true
		return "", nil
	}
	readInstalledVersion = func() (string, error) {
		return installed, nil
	}
	t.Cleanup(func() {
		version = oldVersion
		latestReleaseVersion = oldLatest
		runGoInstall = oldInstall
		readInstalledVersion = oldReadInstalled
	})
	return &installCalled
}

func TestCmdUpdate__should_report_up_to_date__when_running_latest(t *testing.T) {
	installCalled := withUpdateStubs(t, "1.2.3", "1.2.3", "1.2.3")

	out := captureStdout(t, cmdUpdate)

	if *installCalled {
		t.Fatal("go install should not run when the installed CLI is already latest")
	}
	if !strings.Contains(out, "Already up to date: 1.2.3") {
		t.Fatalf("expected up-to-date output, got:\n%s", out)
	}
}

func TestCmdUpdate__should_explain_go_proxy_delay__when_installed_version_stays_old(t *testing.T) {
	installCalled := withUpdateStubs(t, "1.2.2", "1.2.3", "1.2.2")

	out := captureStdout(t, cmdUpdate)

	if !*installCalled {
		t.Fatal("go install should run when latest is newer")
	}
	if !strings.Contains(out, "installed binary is still 1.2.2") ||
		!strings.Contains(out, "Try again in a few minutes") ||
		!strings.Contains(out, "GOPROXY=direct go install github.com/yuberalberto/engram-lite/cmd/engram-lite@v1.2.3") {
		t.Fatalf("expected proxy delay guidance, got:\n%s", out)
	}
}

func withRenameStubs(t *testing.T, exePath string, renameErr error) {
	t.Helper()
	oldExe := osExecutable
	oldRename := osRename
	oldRemove := osRemove
	t.Cleanup(func() {
		osExecutable = oldExe
		osRename = oldRename
		osRemove = oldRemove
	})
	osExecutable = func() (string, error) { return exePath, nil }
	osRename = func(oldpath, newpath string) error { return renameErr }
	osRemove = func(name string) error { return nil }
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestCmdUpdate__should_succeed_via_rename__when_binary_in_use(t *testing.T) {
	calls := 0
	oldVersion := version
	oldLatest := latestReleaseVersion
	oldInstall := runGoInstall
	oldReadInstalled := readInstalledVersion
	oldExit := exitFunc
	t.Cleanup(func() {
		version = oldVersion
		latestReleaseVersion = oldLatest
		runGoInstall = oldInstall
		readInstalledVersion = oldReadInstalled
		exitFunc = oldExit
	})

	version = "1.2.2"
	latestReleaseVersion = func() (string, versioncheck.CheckResult) {
		return "1.2.3", versioncheck.CheckResult{Status: versioncheck.StatusUpToDate, Latest: "1.2.3"}
	}
	runGoInstall = func(target string) (string, error) {
		calls++
		if calls == 1 {
			return "open engram-lite.exe: The process cannot access the file because it is being used by another process.", errors.New("exit status 1")
		}
		return "", nil
	}
	readInstalledVersion = func() (string, error) { return "1.2.3", nil }
	exitFunc = func(int) { t.Error("exitFunc called unexpectedly") }
	withRenameStubs(t, "/go/bin/engram-lite.exe", nil)

	out := captureStdout(t, cmdUpdate)

	if calls != 2 {
		t.Fatalf("expected runGoInstall called twice, got %d", calls)
	}
	if !strings.Contains(out, "Binary in use") {
		t.Fatalf("expected retry message, got:\n%s", out)
	}
	if !strings.Contains(out, "Update complete") {
		t.Fatalf("expected success message, got:\n%s", out)
	}
}

func TestCmdUpdate__should_fail_gracefully__when_rename_fails(t *testing.T) {
	oldVersion := version
	oldLatest := latestReleaseVersion
	oldInstall := runGoInstall
	oldExit := exitFunc
	t.Cleanup(func() {
		version = oldVersion
		latestReleaseVersion = oldLatest
		runGoInstall = oldInstall
		exitFunc = oldExit
	})

	version = "1.2.2"
	latestReleaseVersion = func() (string, versioncheck.CheckResult) {
		return "1.2.3", versioncheck.CheckResult{Status: versioncheck.StatusUpToDate, Latest: "1.2.3"}
	}
	runGoInstall = func(target string) (string, error) {
		return "open engram-lite.exe: The process cannot access the file because it is being used by another process.", errors.New("exit status 1")
	}
	exitFunc = func(int) {}
	withRenameStubs(t, "/go/bin/engram-lite.exe", errors.New("access denied"))

	errOut := captureStderr(t, cmdUpdate)

	if !strings.Contains(errOut, "Update failed") {
		t.Fatalf("expected failure message, got:\n%s", errOut)
	}
}
