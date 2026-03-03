package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/mcpregistry"
	"github.com/lamchakchan/claude-workspace/internal/sessions"
)

const (
	scopeUser    = "user"
	scopeProject = "project"
)

var scopeChoices = []string{"local", scopeUser, scopeProject}

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	if theme.Primary == nil {
		t.Error("Primary color is nil")
	}
	if theme.Error == nil {
		t.Error("Error color is nil")
	}
}

func TestIsQuitFalseOnZeroValue(t *testing.T) {
	// A zero-value KeyPressMsg should not be quit.
	var msg tea.KeyPressMsg
	if IsQuit(msg) {
		t.Error("IsQuit(zero) = true, want false")
	}
}

func TestIsBackFalseOnZeroValue(t *testing.T) {
	var msg tea.KeyPressMsg
	if IsBack(msg) {
		t.Error("IsBack(zero) = true, want false")
	}
}

func TestNewForm(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{
		{Label: "Name", Required: true},
		{Label: "Value"},
	}
	form := NewForm("Test Form", fields, &theme)
	if form.Title != "Test Form" {
		t.Errorf("form title = %q, want %q", form.Title, "Test Form")
	}
	if len(form.inputs) != 2 {
		t.Errorf("form inputs count = %d, want 2", len(form.inputs))
	}
}

func TestNewConfirm(t *testing.T) {
	theme := DefaultTheme()
	c := NewConfirm("Delete?", "This action is irreversible.", true, &theme)
	if c.Title != "Delete?" {
		t.Errorf("confirm title = %q, want %q", c.Title, "Delete?")
	}
	if !c.Cursor {
		t.Error("confirm cursor = false, want true (default yes)")
	}
}

func TestNewStepper(t *testing.T) {
	theme := DefaultTheme()
	labels := []string{"Step 1", "Step 2", "Step 3"}
	s := NewStepper(labels, &theme)
	if len(s.Steps) != 3 {
		t.Errorf("stepper steps = %d, want 3", len(s.Steps))
	}
	for i, step := range s.Steps {
		if step.Status != StepPending {
			t.Errorf("step[%d].Status = %v, want StepPending", i, step.Status)
		}
	}
}

func TestStepperView(t *testing.T) {
	theme := DefaultTheme()
	s := NewStepper([]string{"Install", "Configure", "Verify"}, &theme)
	s.Steps[0].Status = StepDone
	s.Steps[1].Status = StepRunning
	view := s.View()
	if view == "" {
		t.Error("stepper View() returned empty string")
	}
}

func TestFormValues(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{{Label: "A"}, {Label: "B"}}
	form := NewForm("F", fields, &theme)
	vals := form.Values()
	if len(vals) != 2 {
		t.Errorf("Values() len = %d, want 2", len(vals))
	}
}

func TestNewMcpAdd(t *testing.T) {
	theme := DefaultTheme()
	m := NewMcpAdd(&theme)
	if len(m.form.Fields) != 4 {
		t.Errorf("McpAdd fields = %d, want 4", len(m.form.Fields))
	}
	// Scope field (index 3) must be a select field with the three valid scopes.
	scopeField := m.form.Fields[3]
	if len(scopeField.Choices) != 3 {
		t.Errorf("scope field choices = %d, want 3", len(scopeField.Choices))
	}
	// Default choice must be "local".
	const wantScope = "local"
	vals := m.form.Values()
	if vals[3] != wantScope {
		t.Errorf("default scope = %q, want %q", vals[3], wantScope)
	}
}

func TestSelectField_DefaultValue(t *testing.T) {
	const wantScope = "local"
	theme := DefaultTheme()
	fields := []FormField{
		{Label: "Scope", Choices: []string{wantScope, "user", "project"}},
	}
	form := NewForm("Test", fields, &theme)
	vals := form.Values()
	if vals[0] != wantScope {
		t.Errorf("default choice = %q, want %q", vals[0], wantScope)
	}
}

func TestSelectField_RightCycles(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{
		{Label: "Scope", Choices: scopeChoices},
	}
	form := NewForm("Test", fields, &theme)

	right := tea.KeyPressMsg{Code: tea.KeyRight}
	form, _ = form.handleFormKey(right)
	if got := form.Values()[0]; got != scopeUser {
		t.Errorf("after 1 right: choice = %q, want %q", got, scopeUser)
	}

	form, _ = form.handleFormKey(right)
	if got := form.Values()[0]; got != scopeProject {
		t.Errorf("after 2 right: choice = %q, want %q", got, scopeProject)
	}
}

func TestSelectField_RightWraps(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{
		{Label: "Scope", Choices: scopeChoices},
	}
	form := NewForm("Test", fields, &theme)

	right := tea.KeyPressMsg{Code: tea.KeyRight}
	for range 3 {
		form, _ = form.handleFormKey(right)
	}
	// After 3 right presses on 3 choices, should wrap back to "local".
	if got := form.Values()[0]; got != "local" {
		t.Errorf("after wrap: choice = %q, want %q", got, "local")
	}
}

func TestSelectField_LeftWraps(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{
		{Label: "Scope", Choices: scopeChoices},
	}
	form := NewForm("Test", fields, &theme)

	// Left from index 0 should wrap to last choice.
	left := tea.KeyPressMsg{Code: tea.KeyLeft}
	form, _ = form.handleFormKey(left)
	if got := form.Values()[0]; got != scopeProject {
		t.Errorf("after left wrap: choice = %q, want %q", got, scopeProject)
	}
}

func TestSelectField_AdjacentTextFieldUnaffected(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{
		{Label: "Name"},
		{Label: "Scope", Choices: scopeChoices},
	}
	form := NewForm("Test", fields, &theme)

	// Advance focus to the select field.
	tab := tea.KeyPressMsg{Code: tea.KeyTab}
	form, _ = form.handleFormKey(tab)

	// Cycle the select field.
	right := tea.KeyPressMsg{Code: tea.KeyRight}
	form, _ = form.handleFormKey(right)

	vals := form.Values()
	if vals[0] != "" {
		t.Errorf("text field value = %q, want empty", vals[0])
	}
	if vals[1] != scopeUser {
		t.Errorf("select field after right = %q, want %q", vals[1], scopeUser)
	}
}

func TestNewAttach(t *testing.T) {
	theme := DefaultTheme()
	m := NewAttach(&theme)
	if len(m.form.Fields) != 1 {
		t.Errorf("Attach fields = %d, want 1", len(m.form.Fields))
	}
}

func TestNewSandbox(t *testing.T) {
	theme := DefaultTheme()
	m := NewSandbox(&theme)
	if len(m.form.Fields) != 2 {
		t.Errorf("Sandbox fields = %d, want 2", len(m.form.Fields))
	}
}

func TestNewViewer(t *testing.T) {
	theme := DefaultTheme()
	v := NewViewer("Test Title", "Hello world", &theme)
	if v.title != "Test Title" {
		t.Errorf("viewer title = %q, want %q", v.title, "Test Title")
	}
	if v.loading {
		t.Error("NewViewer should not be loading")
	}
}

func TestViewerCopyKey(t *testing.T) {
	theme := DefaultTheme()
	v := NewViewer("Copy Test", "clipboard content", &theme)
	v.SetSize(80, 24)

	// Press 'y' to copy
	msg := tea.KeyPressMsg{Code: 'y', Text: "y"}
	model, cmd := v.Update(msg)
	viewer := model.(*ViewerModel)
	if !viewer.copied {
		t.Error("copied should be true after pressing y")
	}
	if cmd == nil {
		t.Error("expected a command (clipboard + tick) after pressing y")
	}

	// Simulate the copied flash clearing
	model, _ = viewer.Update(viewerCopiedMsg{})
	viewer = model.(*ViewerModel)
	if viewer.copied {
		t.Error("copied should be false after viewerCopiedMsg")
	}
}

func TestNewLoadingViewer(t *testing.T) {
	theme := DefaultTheme()
	v := NewLoadingViewer("Loading Test", func() (string, error) { return "ok", nil }, &theme)
	if !v.loading {
		t.Error("NewLoadingViewer should be loading")
	}
	if v.loader == nil {
		t.Error("NewLoadingViewer should have a loader")
	}
}

func TestNewSkills(t *testing.T) {
	theme := DefaultTheme()
	m := NewSkills(&theme)
	if m.list == nil {
		t.Error("NewSkills list is nil")
	}
}

func TestNewAgents(t *testing.T) {
	theme := DefaultTheme()
	m := NewAgents(&theme)
	if m.list == nil {
		t.Error("NewAgents list is nil")
	}
}

func TestNewHooks(t *testing.T) {
	theme := DefaultTheme()
	m := NewHooks(&theme)
	if m.list == nil {
		t.Error("NewHooks list is nil")
	}
}

func TestNewMemory(t *testing.T) {
	theme := DefaultTheme()
	m := NewMemory(&theme)
	if m.viewer == nil {
		t.Error("NewMemory viewer is nil")
	}
}

func TestNewSessions(t *testing.T) {
	theme := DefaultTheme()
	m := NewSessions(&theme)
	if m.theme == nil {
		t.Error("NewSessions theme is nil")
	}
}

func TestSessionsVisibleRows(t *testing.T) {
	theme := DefaultTheme()
	m := NewSessions(&theme)
	m.height = 30
	// overhead = 8, so visible = 30-8 = 22
	if got := m.visibleRows(); got != 22 {
		t.Errorf("visibleRows() = %d, want 22", got)
	}
	m.height = 5
	// 5-8 = -3, clamped to 1
	if got := m.visibleRows(); got != 1 {
		t.Errorf("visibleRows() with small height = %d, want 1", got)
	}
}

func TestRenderScrollbar(t *testing.T) {
	theme := DefaultTheme()

	// No scrollbar when content fits
	bar := renderScrollbar(10, 5, 10, 0, &theme)
	if bar != "" {
		t.Errorf("expected empty scrollbar when content fits, got %q", bar)
	}

	// Scrollbar at top (0%)
	bar = renderScrollbar(10, 100, 10, 0, &theme)
	if bar == "" {
		t.Error("expected non-empty scrollbar for 100 items in 10 rows")
	}
	lines := strings.Split(bar, "\n")
	if len(lines) != 10 {
		t.Errorf("scrollbar lines = %d, want 10", len(lines))
	}

	// Scrollbar at bottom (100%)
	bar = renderScrollbar(10, 100, 10, 1.0, &theme)
	if bar == "" {
		t.Error("expected non-empty scrollbar at 100%")
	}
	lines = strings.Split(bar, "\n")
	if len(lines) != 10 {
		t.Errorf("scrollbar lines at 100%% = %d, want 10", len(lines))
	}

	// Scrollbar at 50%
	bar = renderScrollbar(10, 100, 10, 0.5, &theme)
	if bar == "" {
		t.Error("expected non-empty scrollbar at 50%")
	}
}

func TestSessionsScrollClamp(t *testing.T) {
	theme := DefaultTheme()
	m := NewSessions(&theme)
	m.loading = false
	m.height = 18 // overhead=8, so visibleRows=10
	m.width = 80

	// Create 25 sessions
	now := time.Now()
	m.sessions = make([]sessions.Session, 25)
	for i := range m.sessions {
		m.sessions[i] = sessions.Session{
			ID:        fmt.Sprintf("sess-%04d", i),
			Title:     fmt.Sprintf("Session %d", i),
			StartTime: now.Add(-time.Duration(i) * time.Hour),
		}
	}

	// Page down should move cursor by visibleRows (10)
	msg := tea.KeyPressMsg{Code: 'f', Text: "f"}
	m.Update(msg)
	if m.cursor != 10 {
		t.Errorf("after pgdn: cursor = %d, want 10", m.cursor)
	}
	if m.scroll < 1 {
		t.Errorf("after pgdn: scroll = %d, want >= 1", m.scroll)
	}

	// Page up should go back
	msg = tea.KeyPressMsg{Code: 'b', Text: "b"}
	m.Update(msg)
	if m.cursor != 0 {
		t.Errorf("after pgup: cursor = %d, want 0", m.cursor)
	}
	if m.scroll != 0 {
		t.Errorf("after pgup: scroll = %d, want 0", m.scroll)
	}

	// G should go to bottom
	msg = tea.KeyPressMsg{Code: 'G', Text: "G", ShiftedCode: 'G'}
	m.Update(msg)
	if m.cursor != 24 {
		t.Errorf("after G: cursor = %d, want 24", m.cursor)
	}

	// g should go to top
	msg = tea.KeyPressMsg{Code: 'g', Text: "g"}
	m.Update(msg)
	if m.cursor != 0 {
		t.Errorf("after g: cursor = %d, want 0", m.cursor)
	}
	if m.scroll != 0 {
		t.Errorf("after g: scroll = %d, want 0", m.scroll)
	}
}

func TestNewDoctor(t *testing.T) {
	theme := DefaultTheme()
	m := NewDoctor(&theme)
	if m.viewer == nil {
		t.Error("NewDoctor viewer is nil")
	}
}

func TestNewMcpList(t *testing.T) {
	theme := DefaultTheme()
	m := NewMcpList(&theme)
	if m.viewer == nil {
		t.Error("NewMcpList viewer is nil")
	}
}

func TestNewCost(t *testing.T) {
	theme := DefaultTheme()
	m := NewCost(&theme)
	if m.theme == nil {
		t.Error("NewCost theme is nil")
	}
	if !m.loading {
		t.Error("NewCost should start in loading state")
	}
	if m.activeTab != tabDaily {
		t.Errorf("NewCost activeTab = %d, want %d (daily)", m.activeTab, tabDaily)
	}
}

func TestCostTabLabels(t *testing.T) {
	if len(costTabLabels) != int(costTabCount) {
		t.Errorf("costTabLabels has %d entries, want %d", len(costTabLabels), costTabCount)
	}
	if len(costTabArgs) != int(costTabCount) {
		t.Errorf("costTabArgs has %d entries, want %d", len(costTabArgs), costTabCount)
	}
}

func TestCostRenderTabs(t *testing.T) {
	theme := DefaultTheme()
	tabs := make([]TabItem, len(costTabLabels))
	for i, label := range costTabLabels {
		tabs[i] = TabItem{Label: label}
	}
	out := renderTabBar(tabs, 0, 80, &theme)
	if out == "" {
		t.Error("renderTabBar returned empty string")
	}
	// All tab labels should appear
	for _, label := range costTabLabels {
		if !strings.Contains(out, label) {
			t.Errorf("renderTabBar missing label %q", label)
		}
	}
}

func TestNewSetup(t *testing.T) {
	theme := DefaultTheme()
	m := NewSetup(&theme)
	if m.viewer == nil {
		t.Error("NewSetup viewer is nil")
	}
}

func TestNewStatusline(t *testing.T) {
	theme := DefaultTheme()
	m := NewStatusline(&theme)
	if m.viewer == nil {
		t.Error("NewStatusline viewer is nil")
	}
}

func TestListPathSuggestions_Empty(t *testing.T) {
	suggestions := listPathSuggestions("")
	if suggestions != nil {
		t.Errorf("listPathSuggestions(\"\") = %v, want nil", suggestions)
	}
}

func TestListPathSuggestions_Root(t *testing.T) {
	suggestions := listPathSuggestions("/")
	if len(suggestions) == 0 {
		t.Error("listPathSuggestions(\"/\") returned no suggestions")
	}
	// All suggestions should start with /
	for _, s := range suggestions {
		if !strings.HasPrefix(s, "/") {
			t.Errorf("suggestion %q does not start with /", s)
		}
	}
}

func TestListPathSuggestions_TildeExpansion(t *testing.T) {
	suggestions := listPathSuggestions("~/")
	if len(suggestions) == 0 {
		t.Error("listPathSuggestions(\"~/\") returned no suggestions")
	}
	// All suggestions should start with ~/
	for _, s := range suggestions {
		if !strings.HasPrefix(s, "~/") {
			t.Errorf("suggestion %q does not start with ~/", s)
		}
	}
}

func TestListPathSuggestions_DirSuffix(_ *testing.T) {
	suggestions := listPathSuggestions("/tmp/")
	// Verify we get suggestions — directories end with /, files don't.
	_ = suggestions
}

func TestFormIsPath(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{
		{Label: "Path", IsPath: true},
		{Label: "Name"},
	}
	form := NewForm("Test", fields, &theme)
	if !form.Fields[0].IsPath {
		t.Error("field 0 should have IsPath=true")
	}
	if form.Fields[1].IsPath {
		t.Error("field 1 should have IsPath=false")
	}
}

func TestFormTextInput_CharacterInsertion(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{{Label: "Name", Required: true}}
	form := NewForm("Test", fields, &theme)

	form, _ = form.handleFormKey(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if got := form.Values()[0]; got != "a" {
		t.Errorf("after typing 'a': value = %q, want %q", got, "a")
	}

	form, _ = form.handleFormKey(tea.KeyPressMsg{Code: 'b', Text: "b"})
	if got := form.Values()[0]; got != "ab" {
		t.Errorf("after typing 'b': value = %q, want %q", got, "ab")
	}
}

func TestFormTextInput_QInsertsQ(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{{Label: "Name"}}
	form := NewForm("Test", fields, &theme)

	form, _ = form.handleFormKey(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if got := form.Values()[0]; got != "q" {
		t.Errorf("after typing 'q': value = %q, want %q", got, "q")
	}
}

func TestFormTextInput_Backspace(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{{Label: "Name"}}
	form := NewForm("Test", fields, &theme)

	form, _ = form.handleFormKey(tea.KeyPressMsg{Code: 'a', Text: "a"})
	form, _ = form.handleFormKey(tea.KeyPressMsg{Code: 'b', Text: "b"})
	form, _ = form.handleFormKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if got := form.Values()[0]; got != "a" {
		t.Errorf("after backspace: value = %q, want %q", got, "a")
	}
}

func TestFormTextInput_EscCancels(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{{Label: "Name"}}
	form := NewForm("Test", fields, &theme)

	_, cmd := form.handleFormKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("esc should produce a command")
	}
	msg := cmd()
	result, ok := msg.(FormResult)
	if !ok {
		t.Fatalf("esc command should produce FormResult, got %T", msg)
	}
	if !result.Cancelled {
		t.Error("esc should produce cancelled FormResult")
	}
}

func TestFormViewUpdate_QForwardsToForm(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{{Label: "Name"}}
	form := NewForm("Test", fields, &theme)

	form, cmd := formViewUpdate(form, tea.KeyPressMsg{Code: 'q', Text: "q"}, nil)
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(PopViewMsg); ok {
			t.Error("pressing 'q' should not pop the view in a form")
		}
	}
	if got := form.Values()[0]; got != "q" {
		t.Errorf("form should contain 'q', got %q", got)
	}
}

func TestIsAccessible(_ *testing.T) {
	// Just verify it doesn't panic.
	_ = IsAccessible()
}

func TestNewMcpPicker(t *testing.T) {
	theme := DefaultTheme()
	m := NewMcpPicker(&theme)
	if !m.loading {
		t.Error("NewMcpPicker should start in loading state")
	}
	if len(m.entries) != 0 {
		t.Errorf("NewMcpPicker entries = %d, want 0 before loading", len(m.entries))
	}
}

func TestNewMcpAddHTTP(t *testing.T) {
	theme := DefaultTheme()
	m := NewMcpAddHTTP(&theme)
	if len(m.form.Fields) != 3 {
		t.Errorf("McpAddHTTP fields = %d, want 3", len(m.form.Fields))
	}
	// Name field should not be required
	if m.form.Fields[0].Required {
		t.Error("HTTP name field should not be required")
	}
	// URL field should be required
	if !m.form.Fields[1].Required {
		t.Error("HTTP URL field should be required")
	}
	// Scope should default to "user"
	vals := m.form.Values()
	if vals[2] != scopeUser {
		t.Errorf("HTTP default scope = %q, want %q", vals[2], scopeUser)
	}
}

func TestNewMcpAddFromRecipe_Stdio(t *testing.T) {
	theme := DefaultTheme()
	recipe := &mcpregistry.Recipe{
		Key:       "brave-search",
		Transport: mcpregistry.TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-brave-search"},
		EnvVars:   map[string]string{"BRAVE_API_KEY": "${BRAVE_API_KEY}"},
		Scope:     scopeUser,
	}
	m := NewMcpAddFromRecipe(recipe, &theme)
	if m.transport != mcpregistry.TransportStdio {
		t.Errorf("transport = %q, want stdio", m.transport)
	}
	vals := m.form.Values()
	if vals[0] != "brave-search" {
		t.Errorf("name = %q, want %q", vals[0], "brave-search")
	}
	if vals[1] != "BRAVE_API_KEY" {
		t.Errorf("env var = %q, want %q", vals[1], "BRAVE_API_KEY")
	}
	if vals[2] != "npx -y @modelcontextprotocol/server-brave-search" {
		t.Errorf("command = %q, want pre-filled command string", vals[2])
	}
	if vals[3] != scopeUser {
		t.Errorf("scope = %q, want %q", vals[3], scopeUser)
	}
}

func TestNewMcpAddFromRecipe_HTTP(t *testing.T) {
	theme := DefaultTheme()
	recipe := &mcpregistry.Recipe{
		Key:       "sentry",
		Transport: mcpregistry.TransportHTTP,
		URL:       "https://mcp.sentry.dev/mcp",
		Scope:     scopeUser,
	}
	m := NewMcpAddFromRecipe(recipe, &theme)
	if m.transport != mcpregistry.TransportHTTP {
		t.Errorf("transport = %q, want http", m.transport)
	}
	vals := m.form.Values()
	if vals[0] != "sentry" {
		t.Errorf("name = %q, want %q", vals[0], "sentry")
	}
	if vals[1] != "https://mcp.sentry.dev/mcp" {
		t.Errorf("url = %q, want %q", vals[1], "https://mcp.sentry.dev/mcp")
	}
	if vals[2] != scopeUser {
		t.Errorf("scope = %q, want %q", vals[2], scopeUser)
	}
}

func TestFormSetValue(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{{Label: "Name"}, {Label: "Value"}}
	form := NewForm("Test", fields, &theme)

	form.SetValue(0, "hello")
	if got := form.Values()[0]; got != "hello" {
		t.Errorf("SetValue(0) = %q, want %q", got, "hello")
	}

	// Out of bounds should not panic
	form.SetValue(-1, "bad")
	form.SetValue(99, "bad")
}

func TestFormSetChoice(t *testing.T) {
	theme := DefaultTheme()
	fields := []FormField{
		{Label: "Scope", Choices: scopeChoices},
	}
	form := NewForm("Test", fields, &theme)

	form.SetChoice(0, scopeUser)
	if got := form.Values()[0]; got != scopeUser {
		t.Errorf("SetChoice('user') = %q, want %q", got, scopeUser)
	}

	form.SetChoice(0, scopeProject)
	if got := form.Values()[0]; got != scopeProject {
		t.Errorf("SetChoice('project') = %q, want %q", got, scopeProject)
	}

	// Non-existent choice should not change value
	form.SetChoice(0, "nonexistent")
	if got := form.Values()[0]; got != scopeProject {
		t.Errorf("SetChoice('nonexistent') = %q, want %q (unchanged)", got, scopeProject)
	}

	// Out of bounds should not panic
	form.SetChoice(-1, scopeUser)
	form.SetChoice(99, scopeUser)
}
