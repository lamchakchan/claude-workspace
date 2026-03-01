package statusline

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ansiRed    = "\033[1;31m"
	ansiYellow = "\033[1;33m"
	ansiGreen  = "\033[1;32m"
	ansiGray   = "\033[0;37m"
	ansiBold   = "\033[1m"
	ansiReset  = "\033[0m"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// isWide reports whether r occupies 2 terminal columns.
// Covers emoji (U+1F000+), Miscellaneous Symbols (⚠ etc.), CJK, Hangul, and fullwidth forms.
func isWide(r rune) bool {
	return r >= 0x1F000 || // All emoji blocks (🤖💰🔥🧠🚨👥 …)
		(r >= 0x2600 && r <= 0x26FF) || // Miscellaneous Symbols (⚠⛔♻ …)
		(r >= 0x2E80 && r <= 0x303F) || // CJK Radicals supplement through CJK Symbols
		(r >= 0x3040 && r <= 0x33FF) || // Japanese/Korean scripts
		(r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
		(r >= 0xFF01 && r <= 0xFF60) // Fullwidth Forms
}

// displayWidth returns the number of terminal columns occupied by s.
// Wide characters (emoji, CJK) count as 2 columns; all others count as 1.
func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		if isWide(r) {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// truncateDisplay truncates s to at most maxCols display columns, appending "…" if needed.
func truncateDisplay(s string, maxCols int) string {
	if displayWidth(s) <= maxCols {
		return s
	}
	if maxCols <= 1 {
		return "…"
	}
	var b strings.Builder
	dw := 0
	for _, r := range s {
		w := 1
		if isWide(r) {
			w = 2
		}
		if dw+w > maxCols-1 { // reserve 1 col for the "…"
			break
		}
		b.WriteRune(r)
		dw += w
	}
	b.WriteRune('…')
	return b.String()
}

// formatDuration returns elapsed time since t as "Xh Ym" or "Ym".
// Returns "" if t is zero or in the future.
func formatDuration(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	mins := int(time.Since(t).Minutes())
	if mins <= 0 {
		return ""
	}
	if mins < 60 {
		return strconv.Itoa(mins) + "m"
	}
	return strconv.Itoa(mins/60) + "h " + strconv.Itoa(mins%60) + "m"
}

// parseTime attempts RFC3339Nano then RFC3339; returns zero time on failure.
func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// RunRender is the entry point for "claude-workspace statusline render".
// It reads the Claude Code JSON blob from stdin and writes the formatted statusline to stdout.
func RunRender(args []string) error {
	base := ""
	for _, a := range args {
		if strings.HasPrefix(a, "--base=") {
			base = strings.TrimPrefix(a, "--base=")
		}
	}
	return Render(os.Stdin, os.Stdout, base, os.Getenv("COLS"), os.Getenv("CLAUDE_AUTOCOMPACT_PCT_OVERRIDE"))
}

// Render reads Claude Code's JSON blob from r, builds the complete multi-line statusline,
// and writes it to w. base is the pre-computed ccusage/jq base line.
// colsStr is the terminal width as a decimal string; autocompactPct is the compaction threshold.
func Render(r io.Reader, w io.Writer, base, colsStr, autocompactPct string) error {
	inputJSON, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(os.TempDir(), "claude-statusline")

	reset := computeWeeklyReset(home)
	alerts := newServiceChecker(cacheDir, nil).check()
	team := renderTeamSummary(home)

	// Combine base + reset
	result := strings.TrimRight(base, "\n")
	if result != "" && reset != "" {
		result = result + " | " + reset
	}

	// Width compaction
	if result != "" {
		cols := 120
		if colsStr != "" {
			if n, err2 := strconv.Atoi(colsStr); err2 == nil && n > 0 {
				cols = n
			}
		}
		threshold := 95.0
		if autocompactPct != "" {
			if f, err2 := strconv.ParseFloat(autocompactPct, 64); err2 == nil {
				threshold = f
			}
		}
		result = compactResult(result, reset, inputJSON, cols, threshold)
	}

	for _, line := range []string{alerts, team, result} {
		if line != "" {
			fmt.Fprintln(w, line)
		}
	}
	return nil
}

// computeWeeklyReset reads ~/.claude.json and returns a reset countdown string
// like "resets today", "resets tomorrow", or "resets in 3d". Returns "" on error.
func computeWeeklyReset(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return ""
	}
	var claude map[string]interface{}
	if err := json.Unmarshal(data, &claude); err != nil {
		return ""
	}
	oauthAccount, _ := claude["oauthAccount"].(map[string]interface{})
	if oauthAccount == nil {
		return ""
	}
	subStr, _ := oauthAccount["subscriptionCreatedAt"].(string)
	if subStr == "" {
		return ""
	}
	sub, err := time.Parse(time.RFC3339, subStr)
	if err != nil {
		return ""
	}
	now := time.Now().UTC()
	days := (int(sub.Weekday()) - int(now.Weekday()) + 7) % 7
	switch days {
	case 0:
		return "resets today"
	case 1:
		return "resets tomorrow"
	default:
		return fmt.Sprintf("resets in %dd", days)
	}
}

// serviceChecker fetches and caches cloud service status pages.
type serviceChecker struct {
	client   *http.Client
	cacheDir string
}

func newServiceChecker(cacheDir string, client *http.Client) *serviceChecker {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	return &serviceChecker{client: client, cacheDir: cacheDir}
}

func (sc *serviceChecker) fetchCached(label, url string, ttl time.Duration) []byte {
	cacheFile := filepath.Join(sc.cacheDir, label+"-status.json")
	_ = os.MkdirAll(sc.cacheDir, 0755)

	if info, err := os.Stat(cacheFile); err == nil && time.Since(info.ModTime()) < ttl {
		if data, err := os.ReadFile(cacheFile); err == nil {
			return data
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		data, _ := os.ReadFile(cacheFile)
		return data
	}
	req.Header.Set("User-Agent", "claude-statusline")
	resp, err := sc.client.Do(req)
	if err != nil {
		data, _ := os.ReadFile(cacheFile)
		return data
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		data, _ := os.ReadFile(cacheFile)
		return data
	}
	_ = os.WriteFile(cacheFile, body, 0644)
	return body
}

func alertColor(severity, text string) string {
	if severity == "major" {
		return "🚨 " + ansiRed + text + ansiReset
	}
	return "⚠️ " + ansiYellow + text + ansiReset
}

// check fetches all 6 services and returns an ANSI alert string, or "" if all healthy.
func (sc *serviceChecker) check() string {
	const ttl = 5 * time.Minute
	var alerts []string

	type statuspageResp struct {
		Status struct {
			Indicator   string `json:"indicator"`
			Description string `json:"description"`
		} `json:"status"`
		Incidents []struct {
			CreatedAt string `json:"created_at"`
			Status    string `json:"status"`
		} `json:"incidents"`
	}
	for _, svc := range []struct{ key, label, url string }{
		{"github", "GitHub", "https://www.githubstatus.com/api/v2/summary.json"},
		{"claude", "Claude", "https://status.claude.com/api/v2/summary.json"},
		{"cloudflare", "Cloudflare", "https://www.cloudflarestatus.com/api/v2/summary.json"},
	} {
		data := sc.fetchCached(svc.key, svc.url, ttl)
		if data == nil {
			continue
		}
		var resp statuspageResp
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}
		ind := resp.Status.Indicator
		if ind == "" || ind == "none" {
			continue
		}
		sev := "minor"
		if ind == "major" || ind == "critical" {
			sev = "major"
		}
		text := svc.label + ": " + resp.Status.Description
		for _, inc := range resp.Incidents {
			if inc.Status == "monitoring" || inc.Status == "postmortem" || inc.Status == "resolved" {
				continue
			}
			if dur := formatDuration(parseTime(inc.CreatedAt)); dur != "" {
				text += " (" + dur + ")"
			}
			break
		}
		alerts = append(alerts, alertColor(sev, text))
	}

	// AWS Health — non-empty JSON array means active incidents
	if data := sc.fetchCached("aws", "https://health.aws.amazon.com/public/currentevents", ttl); data != nil {
		var events []interface{}
		if err := json.Unmarshal(data, &events); err == nil && len(events) > 0 {
			var earliest time.Time
			for _, ev := range events {
				if m, ok := ev.(map[string]interface{}); ok {
					if st, ok := m["startTime"].(string); ok {
						if t := parseTime(st); !t.IsZero() {
							if earliest.IsZero() || t.Before(earliest) {
								earliest = t
							}
						}
					}
				}
			}
			text := fmt.Sprintf("AWS: Active Incidents (%d)", len(events))
			if dur := formatDuration(earliest); dur != "" {
				text += " (" + dur + ")"
			}
			alerts = append(alerts, alertColor("major", text))
		}
	}

	// Google Cloud — active incidents where latest status is not AVAILABLE
	if data := sc.fetchCached("google-cloud", "https://status.cloud.google.com/incidents.json", ttl); data != nil {
		var incidents []map[string]interface{}
		if err := json.Unmarshal(data, &incidents); err == nil {
			var active []map[string]interface{}
			for _, inc := range incidents {
				update, _ := inc["most_recent_update"].(map[string]interface{})
				status, _ := update["status"].(string)
				if status != "AVAILABLE" && status != "" {
					active = append(active, inc)
				}
			}
			if len(active) > 0 {
				sev := "minor"
				for _, inc := range active {
					if inc["severity"] == "high" {
						sev = "major"
						break
					}
				}
				var earliest time.Time
				for _, inc := range active {
					if begin, ok := inc["begin"].(string); ok {
						if t := parseTime(begin); !t.IsZero() {
							if earliest.IsZero() || t.Before(earliest) {
								earliest = t
							}
						}
					}
				}
				text := fmt.Sprintf("Google Cloud: Active Incidents (%d)", len(active))
				if dur := formatDuration(earliest); dur != "" {
					text += " (" + dur + ")"
				}
				alerts = append(alerts, alertColor(sev, text))
			}
		}
	}

	// Azure DevOps — status.health != "healthy"
	if data := sc.fetchCached("azure-devops", "https://status.dev.azure.com/_apis/status/health?api-version=7.1-preview.1", ttl); data != nil {
		var resp struct {
			Status struct {
				Health  string `json:"health"`
				Message string `json:"message"`
			} `json:"status"`
		}
		if err := json.Unmarshal(data, &resp); err == nil && resp.Status.Health != "" && resp.Status.Health != "healthy" {
			sev := "minor"
			if resp.Status.Health == "unhealthy" {
				sev = "major"
			}
			msg := resp.Status.Message
			if msg == "" {
				msg = "Issues detected"
			}
			alerts = append(alerts, alertColor(sev, "Azure DevOps: "+msg))
		}
	}

	return strings.Join(alerts, "  ")
}

// renderTeamSummary reads ~/.claude/team-state.json and returns an ANSI team line, or "".
func renderTeamSummary(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".claude", "team-state.json"))
	if err != nil {
		return ""
	}
	var state struct {
		UpdatedAt  string   `json:"updated_at"`
		AgentsSeen []string `json:"agents_seen"`
		Tasks      struct {
			Pending    int `json:"pending"`
			InProgress int `json:"in_progress"`
			Completed  int `json:"completed"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return ""
	}

	if state.UpdatedAt != "" {
		if updated, err := time.Parse("2006-01-02T15:04:05Z", state.UpdatedAt); err == nil {
			if time.Since(updated.UTC()) > 30*time.Minute {
				return ""
			}
		}
	}

	agents := state.AgentsSeen
	pending := state.Tasks.Pending
	inProgress := state.Tasks.InProgress
	completed := state.Tasks.Completed
	total := pending + inProgress + completed
	if total == 0 && len(agents) == 0 {
		return ""
	}

	teamName := resolveTeamName(home)

	n := len(agents)
	active := inProgress
	if active > n {
		active = n
	}
	var dotParts []string
	for i := 0; i < active; i++ {
		dotParts = append(dotParts, ansiGreen+"▶"+ansiReset)
	}
	for i := 0; i < n-active; i++ {
		dotParts = append(dotParts, ansiYellow+"⏸"+ansiReset)
	}
	dots := strings.Join(dotParts, " ")

	filled := 0
	if total > 0 {
		filled = int(math.Round(10.0 * float64(completed) / float64(total)))
	}
	bar := ansiGreen + strings.Repeat("█", filled) + ansiReset +
		ansiGray + strings.Repeat("░", 10-filled) + ansiReset

	counts := ansiGreen + "▶" + strconv.Itoa(inProgress) + ansiReset + "  " +
		ansiYellow + "⏸" + strconv.Itoa(pending) + ansiReset + "  " +
		ansiGray + "✓" + strconv.Itoa(completed) + ansiReset

	var parts []string
	parts = append(parts, "👥 "+ansiBold+teamName+ansiReset)
	if dots != "" {
		parts = append(parts, "["+dots+"]")
	}
	parts = append(parts, bar)
	if total > 0 {
		parts = append(parts, strconv.Itoa(completed)+"/"+strconv.Itoa(total)+" tasks")
	}
	parts = append(parts, counts)
	return strings.Join(parts, "  ")
}

// resolveTeamName finds the team name from the most-recently-modified config in ~/.claude/teams/.
func resolveTeamName(home string) string {
	teamsDir := filepath.Join(home, ".claude", "teams")
	var candidates []string
	if m, _ := filepath.Glob(filepath.Join(teamsDir, "*/config.json")); len(m) > 0 {
		candidates = append(candidates, m...)
	}
	if m, _ := filepath.Glob(filepath.Join(teamsDir, "*.json")); len(m) > 0 {
		candidates = append(candidates, m...)
	}
	if len(candidates) == 0 {
		return "team"
	}

	newest := candidates[0]
	var newestTime time.Time
	for _, c := range candidates {
		info, err := os.Stat(c)
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newest = c
		}
	}

	data, err := os.ReadFile(newest)
	if err != nil {
		return "team"
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "team"
	}
	if name, ok := cfg["name"].(string); ok && name != "" {
		return name
	}
	if filepath.Base(newest) == "config.json" {
		return filepath.Base(filepath.Dir(newest))
	}
	return strings.TrimSuffix(filepath.Base(newest), ".json")
}

// compactResult returns result unchanged in normal operation.
// Compaction is only applied when Claude Code's "Context left until auto-compact" indicator
// is active (context usage >= threshold), because that indicator occupies the right side of
// the status bar line, potentially overlapping our content.
//
// When compaction is needed it tries three steps:
//  1. Full result (base + reset) fits → return as-is.
//  2. Base without reset fits → drop reset, preserving ccusage session/daily data.
//  3. Everything too wide → plain model+cost+reset fallback, truncated with "…".
func compactResult(result, reset string, inputJSON []byte, cols int, threshold float64) string {
	var data struct {
		ContextWindow struct {
			UsedPercentage float64 `json:"used_percentage"`
		} `json:"context_window"`
		Cost struct {
			TotalCostUSD float64 `json:"total_cost_usd"`
		} `json:"cost"`
		Model struct {
			DisplayName string `json:"display_name"`
		} `json:"model"`
	}
	if len(inputJSON) > 0 {
		_ = json.Unmarshal(inputJSON, &data)
	}

	// No CC autocompact indicator active — return full result, let terminal wrap if needed.
	used := data.ContextWindow.UsedPercentage
	if used < threshold {
		return result
	}

	// CC indicator is active: reserve its width and compact our content to fit alongside it.
	left := int(math.Round(100 - used))
	ccReserve := len(fmt.Sprintf("  Context left until auto-compact: %d%%", left))
	maxW := cols - ccReserve
	if maxW < 20 {
		maxW = 20
	}

	// Step 1: full result fits — return as-is.
	stripped := ansiRE.ReplaceAllString(result, "")
	if displayWidth(stripped) <= maxW {
		return result
	}

	// Step 2: try dropping the reset suffix to preserve ccusage session/daily data.
	base := result
	if reset != "" {
		suffix := " | " + reset
		if strings.HasSuffix(result, suffix) {
			base = result[:len(result)-len(suffix)]
		}
	}
	if base != result {
		strippedBase := ansiRE.ReplaceAllString(base, "")
		if displayWidth(strippedBase) <= maxW {
			return base
		}
	}

	// Step 3: everything too wide — plain model+cost+reset fallback.
	compact := fmt.Sprintf("%s | $%.2f", data.Model.DisplayName, data.Cost.TotalCostUSD)
	if reset != "" {
		compact += " | " + reset
	}
	return truncateDisplay(compact, maxW)
}
