package sanitize

import "testing"

func TestLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii unchanged", "us-east-1/my-app/my-profile/dev", "us-east-1/my-app/my-profile/dev"},
		{"empty unchanged", "", ""},
		{"multibyte unchanged", "東京-本番", "東京-本番"},
		{"CSI escaped", "a\x1b[2Jb", `a\x1b[2Jb`},
		{"OSC escaped", "a\x1b]0;title\x07b", `a\x1b]0;title\ab`},
		{"ESC followed by LF escaped", "a\x1b\n[2Jb", `a\x1b\n[2Jb`},
		{"lone CR escaped", "a\rb", `a\rb`},
		{"LF escaped", "a\nb", `a\nb`},
		{"TAB escaped", "a\tb", `a\tb`},
		{"DEL escaped", "a\x7fb", `a\x7fb`},
		{"C1 CSI escaped", "a\u009b2Jb", `a\u009b2Jb`},
		{"bidi override escaped", "a\u202eb", `a\u202eb`},
		{"invalid UTF-8 byte escaped", "a\x9bb", `a\x9bb`},
		{"legacy workflow command marker broken", "x ##[warning]y", `x #\x23[warning]y`},
		{"leading workflow command broken", "::warning::x", `\x3a:warning::x`},
		{"leading workflow command after spaces broken", "  ::warning::x", `  \x3a:warning::x`},
		{"leading workflow command after NBSP broken", "\u00a0::warning::x", "\u00a0\\x3a:warning::x"},
		{"colons mid-line unchanged", "a::b", "a::b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Line(tt.in); got != tt.want {
				t.Errorf("Line(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii unchanged", "failed to deploy", "failed to deploy"},
		{"LF and TAB kept", "a\n\tb", "a\n\tb"},
		{"CR before LF escaped", "a\r\nb", "a\\r\nb"},
		{"ESC escaped", "a\x1b[2Kb", `a\x1b[2Kb`},
		{"workflow command on a later line broken", "error\n::add-mask::x", "error\n\\x3a:add-mask::x"},
		{"workflow command on an indented later line broken", "error\n  ::warning::x", "error\n  \\x3a:warning::x"},
		{"workflow command on a tab-indented later line broken", "error\n\t::warning::x", "error\n\t\\x3a:warning::x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Text(tt.in); got != tt.want {
				t.Errorf("Text(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
