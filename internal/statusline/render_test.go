package statusline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- formatDuration ---

func TestFormatDuration_Zero(t *testing.T) {
	if got := formatDuration(time.Time{}); got != "" {
		t.Errorf("expected empty for zero time, got %q", got)
	}
}

func TestFormatDuration_Future(t *testing.T) {
	if got := formatDuration(time.Now().Add(5 * time.Minute)); got != "" {
		t.Errorf("expected empty for future time, got %q", got)
	}
}

func TestFormatDuration_62m(t *testing.T) {
	got := formatDuration(time.Now().Add(-62 * time.Minute))
	if got != "1h 2m" {
		t.Errorf("expected %q, got %q", "1h 2m", got)
	}
}

func TestFormatDuration_45m(t *testing.T) {
	got := formatDuration(time.Now().Add(-45 * time.Minute))
	if got != "45m" {
		t.Errorf("expected %q, got %q", "45m", got)
	}
}

// --- parseTime ---

func TestParseTime_RFC3339(t *testing.T) {
	s := "2024-01-15T10:00:00Z"
	got := parseTime(s)
	if got.IsZero() {
		t.Errorf("expected non-zero time for %q", s)
	}
}

func TestParseTime_RFC3339Millis(t *testing.T) {
	s := "2024-01-15T10:00:00.000Z"
	got := parseTime(s)
	if got.IsZero() {
		t.Errorf("expected non-zero time for millisecond format %q", s)
	}
}

func TestParseTime_Invalid(t *testing.T) {
	if got := parseTime("not-a-time"); !got.IsZero() {
		t.Errorf("expected zero time for invalid input, got %v", got)
	}
}

// --- computeWeeklyReset ---

func TestComputeWeeklyReset_NoFile(t *testing.T) {
	got := computeWeeklyReset(t.TempDir())
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestComputeWeeklyReset_MalformedJSON(t *testing.T) {
	home := t.TempDir()
	_ = os.WriteFile(filepath.Join(home, ".claude.json"), []byte("not json"), 0644)
	if got := computeWeeklyReset(home); got != "" {
		t.Errorf("expected empty for malformed JSON, got %q", got)
	}
}

func TestComputeWeeklyReset_MissingField(t *testing.T) {
	home := t.TempDir()
	data, _ := json.Marshal(map[string]interface{}{"oauthAccount": map[string]interface{}{}})
	_ = os.WriteFile(filepath.Join(home, ".claude.json"), data, 0644)
	if got := computeWeeklyReset(home); got != "" {
		t.Errorf("expected empty for missing field, got %q", got)
	}
}

func TestComputeWeeklyReset_ResetsToday(t *testing.T) {
	home := t.TempDir()
	// Same weekday as today (7 days ago)
	sub := time.Now().UTC().AddDate(0, 0, -7)
	writeClaudeJSON(t, home, sub)
	if got := computeWeeklyReset(home); got != "resets today" {
		t.Errorf("expected 'resets today', got %q", got)
	}
}

func TestComputeWeeklyReset_ResTomorrow(t *testing.T) {
	home := t.TempDir()
	sub := time.Now().UTC().AddDate(0, 0, 1)
	writeClaudeJSON(t, home, sub)
	if got := computeWeeklyReset(home); got != "resets tomorrow" {
		t.Errorf("expected 'resets tomorrow', got %q", got)
	}
}

func TestComputeWeeklyReset_ResetsIn3d(t *testing.T) {
	home := t.TempDir()
	sub := time.Now().UTC().AddDate(0, 0, 3)
	writeClaudeJSON(t, home, sub)
	if got := computeWeeklyReset(home); got != "resets in 3d" {
		t.Errorf("expected 'resets in 3d', got %q", got)
	}
}

func writeClaudeJSON(t *testing.T, home string, sub time.Time) {
	t.Helper()
	data, _ := json.Marshal(map[string]interface{}{
		"oauthAccount": map[string]interface{}{
			"subscriptionCreatedAt": sub.Format(time.RFC3339),
		},
	})
	_ = os.WriteFile(filepath.Join(home, ".claude.json"), data, 0644)
}

// --- serviceChecker ---

// mapTransport routes HTTP requests to fixed responses keyed by URL substring.
type mapTransport struct {
	responses map[string]string
	fallback  string
}

func (t *mapTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body := t.fallback
	for key, val := range t.responses {
		if strings.Contains(req.URL.String(), key) {
			body = val
			break
		}
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func testServiceClient(responses map[string]string, fallback string) *http.Client {
	return &http.Client{Transport: &mapTransport{responses: responses, fallback: fallback}}
}

func TestServiceChecker_AllHealthy(t *testing.T) {
	client := testServiceClient(nil, `{"status":{"indicator":"none","description":"All good"}}`)
	got := newServiceChecker(t.TempDir(), client).check()
	if got != "" {
		t.Errorf("expected empty alerts for healthy services, got %q", got)
	}
}

func TestServiceChecker_OutageAlert(t *testing.T) {
	for _, tt := range []struct {
		name, urlKey, label, indicator, wantColor string
		elapsed                                   time.Duration
	}{
		{"GitHubMajor", "githubstatus.com", "GitHub", "major", ansiRed, 62 * time.Minute},
		{"ClaudeMinor", "status.claude.com", "Claude", "minor", ansiYellow, 14 * time.Minute},
	} {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now().Add(-tt.elapsed)
			fixture := fmt.Sprintf(
				`{"status":{"indicator":%q,"description":"Outage"},"incidents":[{"created_at":%q,"status":"investigating"}]}`,
				tt.indicator, start.UTC().Format(time.RFC3339),
			)
			client := testServiceClient(map[string]string{tt.urlKey: fixture}, `{"status":{"indicator":"none"}}`)
			got := newServiceChecker(t.TempDir(), client).check()
			if !strings.Contains(got, tt.label) {
				t.Errorf("expected %s alert, got %q", tt.label, got)
			}
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("expected color %q in alert, got %q", tt.wantColor, got)
			}
			if !strings.Contains(got, "m)") {
				t.Errorf("expected duration in alert, got %q", got)
			}
		})
	}
}

func TestServiceChecker_SkipsMonitoringIncident(t *testing.T) {
	old := time.Now().Add(-14400 * time.Minute) // 10 days ago, monitoring
	recent := time.Now().Add(-30 * time.Minute) // 30 min ago, investigating
	fixture := fmt.Sprintf(
		`{"status":{"indicator":"minor","description":"Degraded"},"incidents":[{"created_at":%q,"status":"monitoring"},{"created_at":%q,"status":"investigating"}]}`,
		old.UTC().Format(time.RFC3339), recent.UTC().Format(time.RFC3339),
	)
	client := testServiceClient(map[string]string{
		"githubstatus.com": fixture,
	}, `{"status":{"indicator":"none"}}`)
	got := newServiceChecker(t.TempDir(), client).check()
	if strings.Contains(got, "14400m") || strings.Contains(got, "14399m") {
		t.Errorf("should not show duration from monitoring incident, got %q", got)
	}
	if !strings.Contains(got, "m)") {
		t.Errorf("expected duration from investigating incident, got %q", got)
	}
}

func TestServiceChecker_AWSIncidents(t *testing.T) {
	start := time.Now().Add(-30 * time.Minute)
	fixture := fmt.Sprintf(`[{"id":"001","region":"us-east-1","startTime":%q},{"id":"002","region":"us-west-2"}]`, start.UTC().Format(time.RFC3339))
	client := testServiceClient(map[string]string{
		"health.aws.amazon.com": fixture,
	}, `{"status":{"indicator":"none"}}`)
	got := newServiceChecker(t.TempDir(), client).check()
	if !strings.Contains(got, "AWS") {
		t.Errorf("expected AWS alert, got %q", got)
	}
	if !strings.Contains(got, "2") {
		t.Errorf("expected incident count in alert, got %q", got)
	}
	if !strings.Contains(got, "m)") {
		t.Errorf("expected duration in alert, got %q", got)
	}
}

func TestServiceChecker_AWSHealthy(t *testing.T) {
	client := testServiceClient(map[string]string{
		"health.aws.amazon.com": `[]`,
	}, `{"status":{"indicator":"none"}}`)
	got := newServiceChecker(t.TempDir(), client).check()
	if strings.Contains(got, "AWS") {
		t.Errorf("expected no AWS alert for empty array, got %q", got)
	}
}

func TestServiceChecker_AzureDevOpsUnhealthy(t *testing.T) {
	client := testServiceClient(map[string]string{
		"status.dev.azure.com": `{"status":{"health":"unhealthy","message":"Service disruption"}}`,
	}, `{"status":{"indicator":"none"}}`)
	got := newServiceChecker(t.TempDir(), client).check()
	if !strings.Contains(got, "Azure DevOps") {
		t.Errorf("expected Azure DevOps alert, got %q", got)
	}
	if !strings.Contains(got, ansiRed) {
		t.Errorf("expected red color for unhealthy Azure, got %q", got)
	}
}

// errTransport always returns an error, used to verify the cache is used without network.
type errTransport struct{}

func (e *errTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("network disabled in test")
}

func TestServiceChecker_UsesFreshCache(t *testing.T) {
	cacheDir := t.TempDir()

	// Pre-populate a fresh cache file for GitHub
	cacheFile := filepath.Join(cacheDir, "github-status.json")
	freshData := `{"status":{"indicator":"minor","description":"FromCache"}}`
	_ = os.WriteFile(cacheFile, []byte(freshData), 0644)
	now := time.Now()
	_ = os.Chtimes(cacheFile, now, now)

	errClient := &http.Client{Transport: &errTransport{}}
	got := newServiceChecker(cacheDir, errClient).check()
	if !strings.Contains(got, "FromCache") {
		t.Errorf("expected cached response to be used, got %q", got)
	}
}

// --- renderTeamSummary ---

func TestRenderTeamSummary_NoFile(t *testing.T) {
	if got := renderTeamSummary(t.TempDir()); got != "" {
		t.Errorf("expected empty for missing file, got %q", got)
	}
}

func TestRenderTeamSummary_MalformedJSON(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0755)
	_ = os.WriteFile(filepath.Join(home, ".claude", "team-state.json"), []byte("bad"), 0644)
	if got := renderTeamSummary(home); got != "" {
		t.Errorf("expected empty for malformed JSON, got %q", got)
	}
}

func TestRenderTeamSummary_StaleState(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0755)
	stale := time.Now().UTC().Add(-31 * time.Minute)
	writeTeamState(t, home, stale, []string{"agent-1"}, 1, 0, 0)
	if got := renderTeamSummary(home); got != "" {
		t.Errorf("expected empty for stale state, got %q", got)
	}
}

func TestRenderTeamSummary_ZeroTasksNoAgents(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0755)
	writeTeamState(t, home, time.Now().UTC(), []string{}, 0, 0, 0)
	if got := renderTeamSummary(home); got != "" {
		t.Errorf("expected empty for zero tasks/agents, got %q", got)
	}
}

func TestRenderTeamSummary_FreshState(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0755)
	writeTeamState(t, home, time.Now().UTC(), []string{"a1", "a2"}, 2, 1, 3)
	got := renderTeamSummary(home)
	if got == "" {
		t.Fatal("expected non-empty team summary")
	}
	if !strings.Contains(got, "👥") {
		t.Error("expected team emoji")
	}
	if !strings.Contains(got, "3/6 tasks") {
		t.Errorf("expected task counts, got %q", got)
	}
	if !strings.Contains(got, "█") {
		t.Error("expected progress bar")
	}
}

func writeTeamState(t *testing.T, home string, updatedAt time.Time, agents []string, pending, inProgress, completed int) {
	t.Helper()
	state := map[string]interface{}{
		"updated_at":  updatedAt.Format("2006-01-02T15:04:05Z"),
		"agents_seen": agents,
		"tasks": map[string]interface{}{
			"pending":     pending,
			"in_progress": inProgress,
			"completed":   completed,
		},
	}
	data, _ := json.Marshal(state)
	_ = os.WriteFile(filepath.Join(home, ".claude", "team-state.json"), data, 0644)
}

func TestRenderTeamSummary_ActiveTeam(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0755)
	writeTeamState(t, home, time.Now().UTC(), []string{"agent-a", "agent-b"}, 1, 1, 2)
	got := renderTeamSummary(home)
	if got == "" {
		t.Fatal("expected non-empty team summary")
	}
	if !strings.Contains(got, "👥") {
		t.Error("expected team emoji 👥")
	}
	if !strings.Contains(got, "▶") {
		t.Error("expected active agent indicator ▶")
	}
	if !strings.Contains(got, "✓") {
		t.Error("expected completed task indicator ✓")
	}
	if !strings.Contains(got, "████") {
		t.Errorf("expected partial progress bar with ████, got %q", got)
	}
}

func TestRenderTeamSummary_Stale(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0755)
	stale := time.Now().Add(-35 * time.Minute).UTC()
	writeTeamState(t, home, stale, []string{"agent-a"}, 0, 1, 0)
	if got := renderTeamSummary(home); got != "" {
		t.Errorf("expected empty for stale state (35min), got %q", got)
	}
}

func TestRenderTeamSummary_AllComplete(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0755)
	writeTeamState(t, home, time.Now().UTC(), []string{"agent-a", "agent-b"}, 0, 0, 4)
	got := renderTeamSummary(home)
	if got == "" {
		t.Fatal("expected non-empty team summary")
	}
	if !strings.Contains(got, "██████████") {
		t.Errorf("expected full progress bar ██████████, got %q", got)
	}
}

// testModelCostTokens is the expected metrics line after dropping 🔥 but keeping all other segments.
const testModelCostTokens = "🤖 Opus 4.6 | 💰 $16.84 session | 🧠 5,803 (3%)"

// --- ccReserveWidth ---

func TestCCReserveWidth_BelowThreshold(t *testing.T) {
	inputJSON := []byte(`{"context_window":{"used_percentage":50}}`)
	if got := ccReserveWidth(inputJSON, 95.0); got != 0 {
		t.Errorf("expected 0 below threshold, got %d", got)
	}
}

func TestCCReserveWidth_AtThreshold(t *testing.T) {
	inputJSON := []byte(`{"context_window":{"used_percentage":96}}`)
	got := ccReserveWidth(inputJSON, 95.0)
	// "  Context left until auto-compact: 4%" = 37 chars
	if got != 37 {
		t.Errorf("expected 37, got %d", got)
	}
}

// --- compactResult ---

func TestCompactResult_FitsMaxW_ReturnsUnchanged(t *testing.T) {
	// When result fits within maxW, return unchanged regardless of length.
	long := strings.Repeat("A very long ANSI-free result ", 10)
	inputJSON := []byte(`{"cost":{"total_cost_usd":0.01},"model":{"display_name":"M"}}`)
	got := compactResult(long, "", inputJSON, 1000) // maxW=1000, everything fits
	if got != long {
		t.Errorf("expected full result unchanged, got %q", got)
	}
}

func TestCompactResult_FitsWidth(t *testing.T) {
	// Result fits within maxW → unchanged.
	inputJSON := []byte(`{"cost":{"total_cost_usd":0.01},"model":{"display_name":"M"}}`)
	short := "M | $0.01"
	got := compactResult(short, "", inputJSON, 42) // maxW=42, "M | $0.01" = 9 chars
	if got != short {
		t.Errorf("expected result unchanged, got %q", got)
	}
}

func TestCompactResult_DropResetPreservesBase(t *testing.T) {
	// Full line too wide but base (without reset) fits → reset dropped.
	// base = 29 chars ≤ 42; result = base + " | resets in 3d" = 44 chars > 42.
	inputJSON := []byte(`{"cost":{"total_cost_usd":0.01},"model":{"display_name":"M"}}`)
	base := "M | $0.01 session / $10 daily" // 29 visible chars
	result := base + " | resets in 3d"      // 44 chars > maxW=42
	got := compactResult(result, "resets in 3d", inputJSON, 42)
	if got != base {
		t.Errorf("expected base with reset dropped, got %q", got)
	}
}

func TestCompactResult_DropHourlyRate(t *testing.T) {
	// After dropping reset, hourly rate segment (🔥) is dropped next.
	inputJSON := []byte(`{"cost":{"total_cost_usd":16.84},"model":{"display_name":"Opus 4.6"}}`)
	base := "🤖 Opus 4.6 | 💰 $16.84 session | 🔥 $4.88/hr | 🧠 5,803 (3%)"
	want := testModelCostTokens
	maxW := displayWidth(want) // exact fit after dropping 🔥
	got := compactResult(base, "", inputJSON, maxW)
	if got != want {
		t.Errorf("expected hourly rate dropped:\n  want: %q\n  got:  %q", want, got)
	}
}

func TestCompactResult_DropBlockCost(t *testing.T) {
	// After dropping hourly rate, block cost sub-segment is dropped.
	inputJSON := []byte(`{"cost":{"total_cost_usd":16.84},"model":{"display_name":"Opus 4.6"}}`)
	base := "🤖 Opus 4.6 | 💰 $16.84 session / $18.35 today / $18.35 block (1h 1m left) | 🧠 5,803 (3%)"
	want := "🤖 Opus 4.6 | 💰 $16.84 session / $18.35 today | 🧠 5,803 (3%)"
	maxW := displayWidth(want)
	got := compactResult(base, "", inputJSON, maxW)
	if got != want {
		t.Errorf("expected block cost dropped:\n  want: %q\n  got:  %q", want, got)
	}
}

func TestCompactResult_DropDailyCost(t *testing.T) {
	// After dropping block cost, daily cost sub-segment is dropped.
	inputJSON := []byte(`{"cost":{"total_cost_usd":16.84},"model":{"display_name":"Opus 4.6"}}`)
	base := "🤖 Opus 4.6 | 💰 $16.84 session / $18.35 today | 🧠 5,803 (3%)"
	want := testModelCostTokens
	maxW := displayWidth(want)
	got := compactResult(base, "", inputJSON, maxW)
	if got != want {
		t.Errorf("expected daily cost dropped:\n  want: %q\n  got:  %q", want, got)
	}
}

func TestCompactResult_AbbreviateSession(t *testing.T) {
	// After dropping daily cost, "session" is abbreviated to "sess".
	inputJSON := []byte(`{"cost":{"total_cost_usd":16.84},"model":{"display_name":"Opus 4.6"}}`)
	base := testModelCostTokens
	want := "🤖 Opus 4.6 | 💰 $16.84 sess | 🧠 5,803 (3%)"
	maxW := displayWidth(want)
	got := compactResult(base, "", inputJSON, maxW)
	if got != want {
		t.Errorf("expected session abbreviated:\n  want: %q\n  got:  %q", want, got)
	}
}

func TestCompactResult_DropTokens(t *testing.T) {
	// After abbreviating, tokens/context segment (🧠) is dropped.
	inputJSON := []byte(`{"cost":{"total_cost_usd":16.84},"model":{"display_name":"Opus 4.6"}}`)
	base := "🤖 Opus 4.6 | 💰 $16.84 sess | 🧠 5,803 (3%)"
	want := "🤖 Opus 4.6 | 💰 $16.84 sess"
	maxW := displayWidth(want)
	got := compactResult(base, "", inputJSON, maxW)
	if got != want {
		t.Errorf("expected tokens dropped:\n  want: %q\n  got:  %q", want, got)
	}
}

func TestCompactResult_CompactFallback(t *testing.T) {
	// Everything too wide → compact fallback with no ANSI codes.
	inputJSON := []byte(`{"cost":{"total_cost_usd":0.01},"model":{"display_name":"Mod"}}`)
	ansiResult := strings.Repeat(ansiGreen+"X"+ansiReset, 100)
	got := compactResult(ansiResult, "resets in 2d", inputJSON, 42)
	// Fallback tries "Mod | $0.01 | resets in 2d" (26 chars) which fits in 42
	want := "Mod | $0.01 | resets in 2d"
	if got != want {
		t.Errorf("expected compact fallback %q, got %q", want, got)
	}
}

func TestCompactResult_WideCharsDisplayWidth(t *testing.T) {
	// Result has wide emoji (2 cols each).
	// 20 emojis (40 display cols) + 20 ASCII (20 display cols) = 60 display cols > maxW 42.
	inputJSON := []byte(`{"cost":{"total_cost_usd":0.01},"model":{"display_name":"M"}}`)
	wideResult := strings.Repeat("🤖", 20) + strings.Repeat("X", 20)
	got := compactResult(wideResult, "", inputJSON, 42)
	if got == wideResult {
		t.Error("expected compaction: display width 60 > maxW 42, but result returned unchanged")
	}
	if got != "M | $0.01" {
		t.Errorf("expected compact fallback \"M | $0.01\", got %q", got)
	}
}

func TestCompactResult_TruncatedWithEllipsis(t *testing.T) {
	// Compact fallback itself exceeds maxW (min 20) → truncated with "…".
	// compact = "Claude Sonnet 4.6 | $0.00" = 25 chars > 20 → truncated.
	inputJSON := []byte(`{"cost":{"total_cost_usd":0.0},"model":{"display_name":"Claude Sonnet 4.6"}}`)
	long := strings.Repeat("A", 100)
	got := compactResult(long, "", inputJSON, 20)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected trailing ellipsis, got %q", got)
	}
	if len([]rune(got)) > 20 {
		t.Errorf("truncated result too long: %d runes, got %q", len([]rune(got)), got)
	}
}

// --- compactAlerts ---

func TestCompactAlerts_FitsUnchanged(t *testing.T) {
	alerts := "⚠️ " + ansiYellow + "CF: Minor issue" + ansiReset
	got := compactAlerts(alerts, 200)
	if got != alerts {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestCompactAlerts_DropDurations(t *testing.T) {
	alerts := "⚠️ " + ansiYellow + "Cloudflare: Minor Service Outage (63h 31m)" + ansiReset
	// Full stripped display width ≈ 47 cols; without duration ≈ 36 cols.
	// Use maxW=40 so full doesn't fit but duration-stripped does.
	got := compactAlerts(alerts, 40)
	if strings.Contains(got, "63h") {
		t.Errorf("expected duration removed, got %q", got)
	}
	if !strings.Contains(got, "Cloudflare") {
		t.Errorf("expected service name preserved, got %q", got)
	}
}

func TestCompactAlerts_AbbreviateServices(t *testing.T) {
	alerts := "⚠️ " + ansiYellow + "Cloudflare: Minor Service Outage (63h 31m)" + ansiReset
	// Very narrow: must abbreviate after dropping duration
	got := compactAlerts(alerts, 30)
	if strings.Contains(got, "63h") {
		t.Errorf("expected duration removed, got %q", got)
	}
	if strings.Contains(got, "Cloudflare") {
		t.Errorf("expected Cloudflare abbreviated, got %q", got)
	}
}

func TestCompactAlerts_Truncate(t *testing.T) {
	alerts := "⚠️ " + ansiYellow + "Cloudflare: Minor Service Outage (63h 31m)" + ansiReset
	got := compactAlerts(alerts, 22)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected trailing ellipsis, got %q", got)
	}
}

func TestCompactAlerts_MultipleAlerts(t *testing.T) {
	// Two alerts joined with double-space (as check() produces)
	alerts := "🚨 " + ansiRed + "GitHub: Major outage (1h 2m)" + ansiReset +
		"  " +
		"⚠️ " + ansiYellow + "Cloudflare: Minor Service Outage (63h 31m)" + ansiReset
	// At maxW=65, durations are stripped (both alerts visible without abbreviation)
	got := compactAlerts(alerts, 65)
	if strings.Contains(got, "1h 2m") || strings.Contains(got, "63h") {
		t.Errorf("expected all durations removed, got %q", got)
	}
	if !strings.Contains(got, "GitHub") || !strings.Contains(got, "Cloudflare") {
		t.Errorf("expected both service names preserved, got %q", got)
	}
}

// --- filterSegments ---

func TestFilterSegments(t *testing.T) {
	segments := []string{"🤖 Opus", "💰 $16.84 session", "🔥 $4.88/hr", "🧠 5,803 (3%)"}
	got := filterSegments(segments, "🔥")
	if len(got) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(got), got)
	}
	for _, s := range got {
		if strings.Contains(s, "🔥") {
			t.Errorf("expected 🔥 segment removed, got %v", got)
		}
	}
}

// --- dropCostSub ---

func TestDropCostSub_Block(t *testing.T) {
	segments := []string{"🤖 Opus", "💰 $16.84 session / $18.35 today / $18.35 block (1h left)", "🧠 5,803"}
	got := dropCostSub(segments, "block")
	if len(got) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(got), got)
	}
	if strings.Contains(got[1], "block") {
		t.Errorf("expected block sub-segment removed from %q", got[1])
	}
	if !strings.Contains(got[1], "session") || !strings.Contains(got[1], "today") {
		t.Errorf("expected session and today preserved in %q", got[1])
	}
}

func TestDropCostSub_Today(t *testing.T) {
	segments := []string{"🤖 Opus", "💰 $16.84 session / $18.35 today", "🧠 5,803"}
	got := dropCostSub(segments, "today")
	if strings.Contains(got[1], "today") {
		t.Errorf("expected today sub-segment removed from %q", got[1])
	}
	if !strings.Contains(got[1], "session") {
		t.Errorf("expected session preserved in %q", got[1])
	}
}

func TestDropCostSub_NoCostSegment(t *testing.T) {
	// No 💰 segment — should return unchanged
	segments := []string{"Model", "$0.01", "10% ctx"}
	got := dropCostSub(segments, "block")
	if len(got) != 3 {
		t.Fatalf("expected 3 segments unchanged, got %d: %v", len(got), got)
	}
}

// --- Render integration ---

func TestRender_EmptyInput(t *testing.T) {
	var buf bytes.Buffer
	err := Render(strings.NewReader("{}"), &buf, "", "120", "")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// With empty/non-existent home files, no alerts/team/reset — just no output is fine
}

func TestRender_BasePassedThrough(t *testing.T) {
	inputJSON := `{"context_window":{"used_percentage":10},"cost":{"total_cost_usd":0.01},"model":{"display_name":"Claude"}}`
	var buf bytes.Buffer
	err := Render(strings.NewReader(inputJSON), &buf, "Claude | $0.01 | 10% ctx", "200", "95")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Claude | $0.01 | 10% ctx") {
		t.Errorf("expected base line in output, got %q", out)
	}
}
