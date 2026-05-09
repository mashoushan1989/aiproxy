package mode

import "testing"

func TestModePersistedValuesRemainStable(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
		want int
	}{
		{name: "Gemini", mode: Gemini, want: 21},
		{name: "WebSearch", mode: WebSearch, want: 22},
		{name: "PPIONative", mode: PPIONative, want: 23},
		{name: "ResponsesCompact", mode: ResponsesCompact, want: 24},
		{name: "ResponsesInputTokens", mode: ResponsesInputTokens, want: 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := int(tt.mode); got != tt.want {
				t.Fatalf("%s = %d, want persisted value %d", tt.name, got, tt.want)
			}
		})
	}
}
