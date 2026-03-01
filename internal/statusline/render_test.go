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

func TestServiceChecker_GitHubMajorOutage(t *testing.T) {
	start := time.Now().Add(-62 * time.Minute)
	fixture := fmt.Sprintf(`{"status":{"indicator":"major","description":"Major outage"},"incidents":[{"created_at":%q,"status":"investigating"}]}`, start.UTC().Format(time.RFC3339))
	client := testServiceClient(map[string]string{
		"githubstatus.com": fixture,
	}, `{"status":{"indicator":"none"}}`)
	got := newServiceChecker(t.TempDir(), client).check()
	if !strings.Contains(got, "GitHub") {
		t.Errorf("expected GitHub alert, got %q", got)
	}
	if !strings.Contains(got, ansiRed) {
		t.Errorf("expected red color for major alert, got %q", got)
	}
	if !strings.Contains(got, "m)") {
		t.Errorf("expected duration in alert, got %q", got)
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

func TestServiceChecker_ClaudeMinorOutage(t *testing.T) {
	start := time.Now().Add(-14 * time.Minute)
	fixture := fmt.Sprintf(`{"status":{"indicator":"minor","description":"Degraded performance"},"incidents":[{"created_at":%q,"status":"identified"}]}`, start.UTC().Format(time.RFC3339))
	client := testServiceClient(map[string]string{
		"status.claude.com": fixture,
	}, `{"status":{"indicator":"none"}}`)
	got := newServiceChecker(t.TempDir(), client).check()
	if !strings.Contains(got, "Claude") {
		t.Errorf("expected Claude alert, got %q", got)
	}
	if !strings.Contains(got, ansiYellow) {
		t.Errorf("expected yellow color for minor alert, got %q", got)
	}
	if !strings.Contains(got, "m)") {
		t.Errorf("expected duration in alert, got %q", got)
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

// --- compactResult ---

func TestCompactResult_NoReserve_AlwaysReturnsFull(t *testing.T) {
	// When context usage is below the threshold, no CC indicator is shown.
	// The full result is returned as-is regardless of terminal width.
	long := strings.Repeat("A very long ANSI-free result ", 10)
	inputJSON := []byte(`{"context_window":{"used_percentage":50},"cost":{"total_cost_usd":0.01},"model":{"display_name":"M"}}`)
	got := compactResult(long, "", inputJSON, 20, 95.0) // cols=20 but no CC reserve
	if got != long {
		t.Errorf("expected full result unchanged (no CC reserve), got %q", got)
	}
}

func TestCompactResult_Reserve_FitsWidth(t *testing.T) {
	// CC indicator active (used=96 >= 95); result fits within available width → unchanged.
	inputJSON := []byte(`{"context_window":{"used_percentage":96},"cost":{"total_cost_usd":0.01},"model":{"display_name":"M"}}`)
	// maxW = 80 - len("  Context left until auto-compact: 4%") = 80 - 38 = 42
	// "M | $0.01" = 9 chars, well under 42.
	short := "M | $0.01"
	got := compactResult(short, "", inputJSON, 80, 95.0)
	if got != short {
		t.Errorf("expected result unchanged, got %q", got)
	}
}

func TestCompactResult_Reserve_DropResetPreservesBase(t *testing.T) {
	// CC indicator active; full line too wide but base (without reset) fits → reset dropped.
	// maxW = 80 - 38 = 42; base = 29 chars ≤ 42; result = base + " | resets in 3d" = 44 chars > 42.
	inputJSON := []byte(`{"context_window":{"used_percentage":96},"cost":{"total_cost_usd":0.01},"model":{"display_name":"M"}}`)
	base := "M | $0.01 session / $10 daily" // 29 visible chars
	result := base + " | resets in 3d"      // 44 chars > maxW=42
	got := compactResult(result, "resets in 3d", inputJSON, 80, 95.0)
	if got != base {
		t.Errorf("expected base with reset dropped, got %q", got)
	}
}

func TestCompactResult_Reserve_CompactFallback(t *testing.T) {
	// CC indicator active; even base is too wide → compact fallback with no ANSI codes.
	inputJSON := []byte(`{"context_window":{"used_percentage":96},"cost":{"total_cost_usd":0.01},"model":{"display_name":"Mod"}}`)
	// maxW = 80 - 38 = 42; 100 visible X chars >> 42 → compact fallback.
	ansiResult := strings.Repeat(ansiGreen+"X"+ansiReset, 100)
	got := compactResult(ansiResult, "resets in 2d", inputJSON, 80, 95.0)
	if strings.Contains(got, "\033[") {
		t.Errorf("compact fallback should not contain ANSI codes, got %q", got)
	}
	if len([]rune(got)) > 42 {
		t.Errorf("compact result too long (%d runes > 42): %q", len([]rune(got)), got)
	}
}

func TestCompactResult_Reserve_WideCharsDisplayWidth(t *testing.T) {
	// CC indicator active; result has wide emoji (2 cols each).
	// 20 emojis (40 display cols) + 20 ASCII (20 display cols) = 40 runes, 60 display cols.
	// maxW = 80 - 38 = 42; 60 display cols > 42 → triggers compaction (step 3).
	// Rune-count-only check would pass (40 ≤ 42) — this test catches that regression.
	inputJSON := []byte(`{"context_window":{"used_percentage":96},"cost":{"total_cost_usd":0.01},"model":{"display_name":"M"}}`)
	wideResult := strings.Repeat("🤖", 20) + strings.Repeat("X", 20)
	got := compactResult(wideResult, "", inputJSON, 80, 95.0)
	if got == wideResult {
		t.Error("expected compaction: display width 60 > maxW 42, but result returned unchanged")
	}
	if got != "M | $0.01" {
		t.Errorf("expected compact fallback \"M | $0.01\", got %q", got)
	}
}

func TestCompactResult_Reserve_TruncatedWithEllipsis(t *testing.T) {
	// CC indicator active; compact fallback itself exceeds maxW (min 20) → truncated with "…".
	// used=99, left=1, reserve=len("  Context left until auto-compact: 1%")=38, cols=25 → maxW=max(25-38,20)=20.
	// compact = "Claude Sonnet 4.6 | $0.00" = 25 chars > 20 → truncated.
	inputJSON := []byte(`{"context_window":{"used_percentage":99},"cost":{"total_cost_usd":0.0},"model":{"display_name":"Claude Sonnet 4.6"}}`)
	long := strings.Repeat("A", 100)
	got := compactResult(long, "", inputJSON, 25, 95.0)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected trailing ellipsis, got %q", got)
	}
	if len([]rune(got)) > 20 {
		t.Errorf("truncated result too long: %d runes, got %q", len([]rune(got)), got)
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
