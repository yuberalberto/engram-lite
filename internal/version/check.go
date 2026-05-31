// Package version checks for newer engram releases on GitHub.
package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner = "yuberalberto"
	repoName  = "engram-lite"
)

var (
	checkTimeout           = 2 * time.Second
	githubLatestReleaseURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	httpClient             = http.DefaultClient
)

type CheckStatus string

const (
	StatusUpToDate        CheckStatus = "up_to_date"
	StatusUpdateAvailable CheckStatus = "update_available"
	StatusCheckFailed     CheckStatus = "check_failed"
)

type CheckResult struct {
	Status  CheckStatus
	Message string
	Latest  string
}

// githubRelease is the subset of the GitHub releases API we care about.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckLatest compares the running version against the latest GitHub release.
// It distinguishes between up-to-date, update available, and check failures.
func CheckLatest(current string) CheckResult {
	switch current {
	case "":
		return checkFailed("Could not check for updates: current version is unknown.")
	case "dev":
		return checkFailed("Could not check for updates: development builds do not map to a release version.")
	}

	latest, latestResult := LatestReleaseVersion()
	if latestResult.Status == StatusCheckFailed {
		return latestResult
	}

	running := normalizeVersion(current)

	if latest == "" {
		return checkFailed("Could not check for updates: GitHub did not return a release version.")
	}

	if latest == running {
		return CheckResult{Status: StatusUpToDate, Latest: latest}
	}

	if !isNewer(latest, running) {
		return CheckResult{Status: StatusUpToDate, Latest: latest}
	}

	return CheckResult{
		Status: StatusUpdateAvailable,
		Latest: latest,
		Message: fmt.Sprintf(
			"Update available: %s -> %s\nTo update:\n%s",
			running, latest, updateInstructions(),
		),
	}
}

// LatestReleaseVersion returns the latest GitHub release version without
// comparing it to the running binary.
func LatestReleaseVersion() (string, CheckResult) {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseURL, nil)
	if err != nil {
		return "", checkFailed("Could not check for updates: could not create the GitHub request.")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := githubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", checkFailed("Could not check for updates: GitHub took too long to respond.")
		}
		return "", checkFailed(fmt.Sprintf("Could not check for updates: %v.", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", checkFailed(nonOKStatusMessage(resp.Status))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", checkFailed("Could not check for updates: could not read the GitHub response.")
	}

	latest := normalizeVersion(release.TagName)
	if latest == "" {
		return "", checkFailed("Could not check for updates: GitHub did not return a release version.")
	}
	return latest, CheckResult{Status: StatusUpToDate, Latest: latest}
}

// normalizeVersion strips a leading "v" prefix.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// isNewer returns true if latest > current using simple semver comparison.
func isNewer(latest, current string) bool {
	latestParts := splitVersion(latest)
	currentParts := splitVersion(current)

	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}
	return false
}

// splitVersion splits "1.8.1" into [1, 8, 1]. Returns [0,0,0] on parse failure.
func splitVersion(v string) [3]int {
	var parts [3]int
	segments := strings.SplitN(v, ".", 3)
	for i, s := range segments {
		if i >= 3 {
			break
		}
		for _, c := range s {
			if c >= '0' && c <= '9' {
				parts[i] = parts[i]*10 + int(c-'0')
			} else {
				break
			}
		}
	}
	return parts
}

// updateInstructions returns platform-appropriate update commands.
func updateInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "  go install github.com/yuberalberto/engram-lite/cmd/engram-lite@latest"
	case "linux":
		return "  go install github.com/yuberalberto/engram-lite/cmd/engram-lite@latest"
	default:
		return "  go install github.com/yuberalberto/engram-lite/cmd/engram-lite@latest\n  or: https://github.com/yuberalberto/engram-lite/releases/latest"
	}
}

func githubToken() string {
	if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
}

func nonOKStatusMessage(status string) string {
	msg := fmt.Sprintf("Could not check for updates: GitHub API returned %s.", status)
	if strings.HasPrefix(status, "401") || strings.HasPrefix(status, "403") {
		msg += " Set GH_TOKEN or GITHUB_TOKEN to reduce rate limits."
	}
	return msg
}

func checkFailed(message string) CheckResult {
	return CheckResult{Status: StatusCheckFailed, Message: message}
}
