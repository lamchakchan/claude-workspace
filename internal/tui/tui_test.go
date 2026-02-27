package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

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
	if len(m.form.Fields) != 3 {
		t.Errorf("McpAdd fields = %d, want 3", len(m.form.Fields))
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

func TestIsAccessible(_ *testing.T) {
	// Just verify it doesn't panic.
	_ = IsAccessible()
}
