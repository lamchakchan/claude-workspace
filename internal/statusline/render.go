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

	severityMajor = "major"
	severityMinor = "minor"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// durationStripRE matches parenthesized durations like " (63h 31m)" or " (14m)" in alert strings.
var durationStripRE = regexp.MustCompile(`\s+\(\d+[hm][\d hm]*\)`)

// isWide reports whether r occupies 2 terminal columns.
// Covers emoji (U+1F000+), Miscellaneous Symbols (⚠ etc.), CJK, Hangul, and fullwidth forms.
func isWide(r rune) bool {
	return r >= 0x1F000 || // All emoji blocks (🤖💰🔥🧠🚨 …)
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
// s must not contain ANSI escape sequences; strip them before calling.
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
	// Combine base + reset
	result := strings.TrimRight(base, "\n")
	if result != "" && reset != "" {
		result = result + " | " + reset
	}

	// Parse terminal width and autocompact threshold
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

	// Width compaction: CC places its autocompact indicator on the first output line.
	// Only compact the line that shares space with the indicator.
	ccReserve := ccReserveWidth(inputJSON, threshold)
	if ccReserve > 0 {
		firstLineMaxW := cols - ccReserve
		if firstLineMaxW < 20 {
			firstLineMaxW = 20
		}
		switch {
		case alerts != "":
			alerts = compactAlerts(alerts, firstLineMaxW)
		default:
			result = compactResult(result, reset, inputJSON, firstLineMaxW)
		}
	}

	for _, line := range []string{alerts, result} {
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
	if severity == severityMajor {
		return "🚨 " + ansiRed + text + ansiReset
	}
	return "⚠️ " + ansiYellow + text + ansiReset
}

// statuspageResp is the common response shape for Statuspage-based services.
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

// check fetches all 6 services and returns an ANSI alert string, or "" if all healthy.
func (sc *serviceChecker) check() string {
	var alerts []string
	alerts = append(alerts, sc.checkStatuspages()...)
	if a := sc.checkAWS(); a != "" {
		alerts = append(alerts, a)
	}
	if a := sc.checkGoogleCloud(); a != "" {
		alerts = append(alerts, a)
	}
	if a := sc.checkAzureDevOps(); a != "" {
		alerts = append(alerts, a)
	}
	return strings.Join(alerts, "  ")
}

// checkStatuspages checks GitHub, Claude, and Cloudflare via the Statuspage API.
func (sc *serviceChecker) checkStatuspages() []string {
	const ttl = 5 * time.Minute
	services := []struct{ key, label, url string }{
		{"github", "GitHub", "https://www.githubstatus.com/api/v2/summary.json"},
		{"claude", "Claude", "https://status.claude.com/api/v2/summary.json"},
		{"cloudflare", "Cloudflare", "https://www.cloudflarestatus.com/api/v2/summary.json"},
	}
	var alerts []string
	for _, svc := range services {
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
		sev := severityMinor
		if ind == severityMajor || ind == "critical" {
			sev = severityMajor
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
	return alerts
}

// earliestTime returns the earliest non-zero time from a list of time strings.
func earliestTime(times []string) time.Time {
	var earliest time.Time
	for _, s := range times {
		if t := parseTime(s); !t.IsZero() {
			if earliest.IsZero() || t.Before(earliest) {
				earliest = t
			}
		}
	}
	return earliest
}

// checkAWS checks AWS Health for active incidents.
func (sc *serviceChecker) checkAWS() string {
	const ttl = 5 * time.Minute
	data := sc.fetchCached("aws", "https://health.aws.amazon.com/public/currentevents", ttl)
	if data == nil {
		return ""
	}
	var events []interface{}
	if err := json.Unmarshal(data, &events); err != nil || len(events) == 0 {
		return ""
	}
	var startTimes []string
	for _, ev := range events {
		if m, ok := ev.(map[string]interface{}); ok {
			if st, ok := m["startTime"].(string); ok {
				startTimes = append(startTimes, st)
			}
		}
	}
	text := fmt.Sprintf("AWS: Active Incidents (%d)", len(events))
	if dur := formatDuration(earliestTime(startTimes)); dur != "" {
		text += " (" + dur + ")"
	}
	return alertColor(severityMajor, text)
}

// checkGoogleCloud checks Google Cloud Status for active incidents.
func (sc *serviceChecker) checkGoogleCloud() string {
	const ttl = 5 * time.Minute
	data := sc.fetchCached("google-cloud", "https://status.cloud.google.com/incidents.json", ttl)
	if data == nil {
		return ""
	}
	var incidents []map[string]interface{}
	if err := json.Unmarshal(data, &incidents); err != nil {
		return ""
	}
	var active []map[string]interface{}
	for _, inc := range incidents {
		update, _ := inc["most_recent_update"].(map[string]interface{})
		status, _ := update["status"].(string)
		if status != "AVAILABLE" && status != "" {
			active = append(active, inc)
		}
	}
	if len(active) == 0 {
		return ""
	}
	sev := severityMinor
	for _, inc := range active {
		if inc["severity"] == "high" {
			sev = severityMajor
			break
		}
	}
	var beginTimes []string
	for _, inc := range active {
		if begin, ok := inc["begin"].(string); ok {
			beginTimes = append(beginTimes, begin)
		}
	}
	text := fmt.Sprintf("Google Cloud: Active Incidents (%d)", len(active))
	if dur := formatDuration(earliestTime(beginTimes)); dur != "" {
		text += " (" + dur + ")"
	}
	return alertColor(sev, text)
}

// checkAzureDevOps checks Azure DevOps status health.
// Because the health API does not expose incident start times, onset is tracked
// locally in <cacheDir>/azure-devops-onset.json and cleared when health returns to normal.
func (sc *serviceChecker) checkAzureDevOps() string {
	const ttl = 5 * time.Minute
	data := sc.fetchCached("azure-devops", "https://status.dev.azure.com/_apis/status/health?api-version=7.1-preview.1", ttl)
	if data == nil {
		return ""
	}
	var resp struct {
		Status struct {
			Health  string `json:"health"`
			Message string `json:"message"`
		} `json:"status"`
	}
	onsetFile := filepath.Join(sc.cacheDir, "azure-devops-onset.json")
	if err := json.Unmarshal(data, &resp); err != nil || resp.Status.Health == "" || resp.Status.Health == "healthy" {
		_ = os.Remove(onsetFile)
		return ""
	}
	sev := severityMinor
	if resp.Status.Health == "unhealthy" {
		sev = severityMajor
	}
	msg := resp.Status.Message
	if msg == "" {
		msg = "Issues detected"
	}
	text := "Azure DevOps: " + msg
	if dur := formatDuration(sc.azureOnset(onsetFile)); dur != "" {
		text += " (" + dur + ")"
	}
	return alertColor(sev, text)
}

// azureOnset returns the recorded onset time for an Azure DevOps outage.
// If no onset file exists yet it creates one stamped with the current time.
// Returns the zero time on any error.
func (sc *serviceChecker) azureOnset(onsetFile string) time.Time {
	type onsetRecord struct {
		Onset string `json:"onset"`
	}
	if raw, err := os.ReadFile(onsetFile); err == nil {
		var rec onsetRecord
		if json.Unmarshal(raw, &rec) == nil {
			if t := parseTime(rec.Onset); !t.IsZero() {
				return t
			}
		}
	}
	now := time.Now().UTC()
	rec := onsetRecord{Onset: now.Format(time.RFC3339)}
	if raw, err := json.Marshal(rec); err == nil {
		_ = os.MkdirAll(sc.cacheDir, 0755)
		_ = os.WriteFile(onsetFile, raw, 0644)
	}
	return now
}

// ccReserveWidth returns the terminal columns reserved by CC's autocompact indicator,
// or 0 if the indicator is not active (context usage below threshold).
func ccReserveWidth(inputJSON []byte, threshold float64) int {
	var data struct {
		ContextWindow struct {
			UsedPercentage float64 `json:"used_percentage"`
		} `json:"context_window"`
	}
	if len(inputJSON) > 0 {
		_ = json.Unmarshal(inputJSON, &data)
	}
	if data.ContextWindow.UsedPercentage < threshold {
		return 0
	}
	left := int(math.Round(100 - data.ContextWindow.UsedPercentage))
	return len(fmt.Sprintf("  Context left until auto-compact: %d%%", left))
}

// filterSegments returns a copy of segments with any element containing substr removed.
func filterSegments(segments []string, substr string) []string {
	result := make([]string, 0, len(segments))
	for _, s := range segments {
		if !strings.Contains(s, substr) {
			result = append(result, s)
		}
	}
	return result
}

// dropCostSub removes a "/" sub-segment containing keyword from the cost segment (the one with 💰).
// For example, dropCostSub(segments, "block") removes the "/ $X.XX block (Xh Xm left)" part.
func dropCostSub(segments []string, keyword string) []string {
	result := make([]string, len(segments))
	copy(result, segments)
	for i, s := range result {
		if !strings.Contains(s, "💰") {
			continue
		}
		subs := strings.Split(s, " / ")
		filtered := make([]string, 0, len(subs))
		for _, sub := range subs {
			if !strings.Contains(sub, keyword) {
				filtered = append(filtered, sub)
			}
		}
		result[i] = strings.Join(filtered, " / ")
		break
	}
	return result
}

// compactAlerts progressively compacts the alerts line to fit within maxW display columns.
// Steps: (1) drop durations, (2) abbreviate service names, (3) truncate with ellipsis.
func compactAlerts(alerts string, maxW int) string {
	if maxW <= 0 {
		return alerts
	}
	stripped := ansiRE.ReplaceAllString(alerts, "")
	if displayWidth(stripped) <= maxW {
		return alerts
	}

	// Step 1: Drop duration parentheticals
	cur := durationStripRE.ReplaceAllString(alerts, "")
	stripped = ansiRE.ReplaceAllString(cur, "")
	if displayWidth(stripped) <= maxW {
		return cur
	}

	// Step 2: Abbreviate service names
	replacer := strings.NewReplacer(
		"Cloudflare", "CF",
		"Google Cloud", "GCP",
		"Azure DevOps", "Azure",
		"Active Incidents", "Incidents",
	)
	cur = replacer.Replace(cur)
	stripped = ansiRE.ReplaceAllString(cur, "")
	if displayWidth(stripped) <= maxW {
		return cur
	}

	// Step 3: Truncate (plain text, no ANSI)
	return truncateDisplay(stripped, maxW)
}

// segmentDegrader transforms pipe-separated segments to progressively shed information.
type segmentDegrader func([]string) []string

// applyDegradations runs each degradation step in order, returning as soon as the
// joined result fits within maxW display columns.
func applyDegradations(segments []string, maxW int, steps []segmentDegrader) (string, bool) {
	for _, step := range steps {
		segments = step(segments)
		joined := strings.Join(segments, " | ")
		if displayWidth(ansiRE.ReplaceAllString(joined, "")) <= maxW {
			return joined, true
		}
	}
	return strings.Join(segments, " | "), false
}

// abbreviateSession replaces " session" with " sess" in each segment.
func abbreviateSession(segments []string) []string {
	out := make([]string, len(segments))
	for i, s := range segments {
		out[i] = strings.Replace(s, " session", " sess", 1)
	}
	return out
}

// compactResult progressively degrades the metrics line to fit within maxW display columns.
//
// Degradation steps:
//  0. Full result (base + reset) fits → return as-is.
//  1. Drop reset suffix.
//  2. Drop hourly rate segment (🔥).
//  3. Drop block cost sub-segment from 💰.
//  4. Drop daily cost sub-segment from 💰.
//  5. Abbreviate "session" → "sess".
//  6. Drop tokens/context segment (🧠).
//  7. Plain model + $cost fallback (with reset if it fits).
//  8. Truncate with "…".
func compactResult(result, reset string, inputJSON []byte, maxW int) string {
	var data struct {
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

	if maxW < 20 {
		maxW = 20
	}

	fits := func(s string) bool {
		return displayWidth(ansiRE.ReplaceAllString(s, "")) <= maxW
	}

	// Step 0: Full result fits
	if fits(result) {
		return result
	}

	// Step 1: Drop reset suffix
	cur := result
	if reset != "" {
		if trimmed := strings.TrimSuffix(cur, " | "+reset); trimmed != cur {
			cur = trimmed
		}
	}
	if fits(cur) {
		return cur
	}

	// Steps 2-6: Segment-based progressive degradation
	segments := strings.Split(cur, " | ")
	if joined, ok := applyDegradations(segments, maxW, []segmentDegrader{
		func(s []string) []string { return filterSegments(s, "🔥") },
		func(s []string) []string { return dropCostSub(s, "block") },
		func(s []string) []string { return dropCostSub(s, "today") },
		abbreviateSession,
		func(s []string) []string { return filterSegments(s, "🧠") },
	}); ok {
		return joined
	}

	// Step 7: Plain model + $cost fallback (optionally with reset)
	model := data.Model.DisplayName
	if model == "" {
		model = "Claude"
	}
	compact := fmt.Sprintf("%s | $%.2f", model, data.Cost.TotalCostUSD)
	if reset != "" && fits(compact+" | "+reset) {
		return compact + " | " + reset
	}
	if fits(compact) {
		return compact
	}

	// Step 8: Truncate with "…"
	return truncateDisplay(compact, maxW)
}
