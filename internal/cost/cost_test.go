package cost

import (
	"math"
	"testing"
)

func TestDetectRuntime_ReturnsKnownForm(t *testing.T) {
	runtime, prefix := detectRuntime()
	if runtime == "" {
		// Neither bun nor npx available in test environment — acceptable
		if prefix != nil {
			t.Error("expected nil prefix when runtime is empty")
		}
		return
	}
	switch runtime {
	case "bun":
		if len(prefix) != 2 || prefix[0] != "x" || prefix[1] != "ccusage" {
			t.Errorf("bun prefix = %v, want [x ccusage]", prefix)
		}
	case "npx":
		if len(prefix) != 2 || prefix[0] != "-y" || prefix[1] != "ccusage" {
			t.Errorf("npx prefix = %v, want [-y ccusage]", prefix)
		}
	default:
		t.Errorf("unexpected runtime: %q", runtime)
	}
}

func TestParseCostJSON_Daily(t *testing.T) {
	input := `{"daily":[{"date":"2026-02-25","totalCost":5.12},{"date":"2026-02-26","totalCost":3.45}]}`
	entries, err := ParseCostJSON("daily", input)
	if err != nil {
		t.Fatalf("ParseCostJSON: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Label != "2026-02-25" {
		t.Errorf("entries[0].Label = %q, want %q", entries[0].Label, "2026-02-25")
	}
	if math.Abs(entries[0].Value-5.12) > 0.001 {
		t.Errorf("entries[0].Value = %f, want 5.12", entries[0].Value)
	}
	if entries[1].Label != "2026-02-26" {
		t.Errorf("entries[1].Label = %q, want %q", entries[1].Label, "2026-02-26")
	}
}

func TestParseCostJSON_Monthly(t *testing.T) {
	input := `{"monthly":[{"month":"2026-01","totalCost":2.17},{"month":"2026-02","totalCost":8.50}]}`
	entries, err := ParseCostJSON("monthly", input)
	if err != nil {
		t.Fatalf("ParseCostJSON: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Label != "2026-01" {
		t.Errorf("entries[0].Label = %q, want %q", entries[0].Label, "2026-01")
	}
	if entries[1].Label != "2026-02" {
		t.Errorf("entries[1].Label = %q, want %q", entries[1].Label, "2026-02")
	}
}

func TestParseCostJSON_Weekly(t *testing.T) {
	input := `{"weekly":[{"week":"2026-W08","totalCost":4.33}]}`
	entries, err := ParseCostJSON("weekly", input)
	if err != nil {
		t.Fatalf("ParseCostJSON: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Label != "2026-W08" {
		t.Errorf("entries[0].Label = %q, want %q", entries[0].Label, "2026-W08")
	}
}

func TestParseCostJSON_Empty(t *testing.T) {
	entries, err := ParseCostJSON("daily", `{"daily":[]}`)
	if err != nil {
		t.Fatalf("ParseCostJSON: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestParseCostJSON_InvalidJSON(t *testing.T) {
	_, err := ParseCostJSON("daily", `not json`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseCostJSON_MissingKey(t *testing.T) {
	_, err := ParseCostJSON("daily", `{"weekly":[]}`)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestCostRecordLabel(t *testing.T) {
	tests := []struct {
		name   string
		record costRecord
		want   string
	}{
		{"date", costRecord{Date: "2026-02-27"}, "2026-02-27"},
		{"month", costRecord{Month: "2026-01"}, "2026-01"},
		{"week", costRecord{Week: "2026-W08"}, "2026-W08"},
		{"title", costRecord{Title: "My Session"}, "My Session"},
		{"name", costRecord{Name: "block-1"}, "block-1"},
		{"id", costRecord{ID: "abc123"}, "abc123"},
		{"empty", costRecord{}, "?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.record.label()
			if got != tt.want {
				t.Errorf("label() = %q, want %q", got, tt.want)
			}
		})
	}
}
