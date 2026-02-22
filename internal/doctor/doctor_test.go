package doctor

import (
	"encoding/json"
	"testing"
)

func TestCountHookCommands(t *testing.T) {
	tests := []struct {
		name  string
		hooks map[string]json.RawMessage
		want  int
	}{
		{
			name:  "empty map",
			hooks: map[string]json.RawMessage{},
			want:  0,
		},
		{
			name: "single matcher with one command hook",
			hooks: map[string]json.RawMessage{
				"PreToolUse": json.RawMessage(`[{"matcher":"Bash","hooks":[{"type":"command","command":"echo hi"}]}]`),
			},
			want: 1,
		},
		{
			name: "multiple matchers with multiple hooks",
			hooks: map[string]json.RawMessage{
				"PreToolUse": json.RawMessage(`[
					{"matcher":"Bash","hooks":[{"type":"command","command":"hook1"},{"type":"command","command":"hook2"}]},
					{"matcher":"Write","hooks":[{"type":"command","command":"hook3"}]}
				]`),
			},
			want: 3,
		},
		{
			name: "non-command hooks are not counted",
			hooks: map[string]json.RawMessage{
				"PreToolUse": json.RawMessage(`[{"matcher":"Bash","hooks":[{"type":"notification","url":"https://example.com"}]}]`),
			},
			want: 0,
		},
		{
			name: "mixed command and non-command hooks",
			hooks: map[string]json.RawMessage{
				"PreToolUse": json.RawMessage(`[{"matcher":"Bash","hooks":[
					{"type":"command","command":"echo ok"},
					{"type":"notification","url":"https://example.com"}
				]}]`),
			},
			want: 1,
		},
		{
			name: "multiple event types",
			hooks: map[string]json.RawMessage{
				"PreToolUse":  json.RawMessage(`[{"matcher":"Bash","hooks":[{"type":"command","command":"pre"}]}]`),
				"PostToolUse": json.RawMessage(`[{"matcher":"Bash","hooks":[{"type":"command","command":"post"}]}]`),
			},
			want: 2,
		},
		{
			name: "malformed JSON is skipped",
			hooks: map[string]json.RawMessage{
				"PreToolUse": json.RawMessage(`not valid json`),
			},
			want: 0,
		},
		{
			name:  "nil map",
			hooks: nil,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countHookCommands(tt.hooks)
			if got != tt.want {
				t.Errorf("countHookCommands() = %d, want %d", got, tt.want)
			}
		})
	}
}
