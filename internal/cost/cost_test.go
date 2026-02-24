package cost

import (
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
