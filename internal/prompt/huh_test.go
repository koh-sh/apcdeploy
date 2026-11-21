package prompt

import (
	"testing"
)

func TestHuhPrompter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Compile-time check that HuhPrompter implements Prompter
	var _ Prompter = (*HuhPrompter)(nil)
}

func TestSelectOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		in        string
		wantLabel string
	}{
		{"plain name unchanged", "my-app", "my-app"},
		// Remote names reach huh's accessible mode (TERM=dumb) verbatim on
		// stdout, so the label must not carry control sequences.
		{"escape sequence neutralised", "b\x1b]0;pwned\x07\x1b[2J", `b\x1b]0;pwned\a\x1b[2J`},
		{"CR/LF neutralised", "t\r\nx", `t\r\nx`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := selectOptions([]string{tt.in})
			if len(opts) != 1 {
				t.Fatalf("got %d options, want 1", len(opts))
			}
			if opts[0].Key != tt.wantLabel {
				t.Errorf("label = %q, want %q", opts[0].Key, tt.wantLabel)
			}
			// The value must stay the exact remote name: it is resolved by
			// exact match afterwards.
			if opts[0].Value != tt.in {
				t.Errorf("value = %q, want %q", opts[0].Value, tt.in)
			}
		})
	}
}
