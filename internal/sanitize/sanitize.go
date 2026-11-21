// Package sanitize neutralises untrusted text before apcdeploy writes it for
// humans to read: the terminal and CI logs.
//
// Much of what apcdeploy prints comes from AWS and can be written by other
// principals (resource names, deployment descriptions, strategy names,
// validator messages). Control characters in that text could move the cursor,
// rewrite earlier lines or set terminal state, and GitHub Actions parses
// workflow commands (`::cmd::` at the start of a line, `##[cmd]` anywhere)
// out of a step's stdout and stderr. Line and Text render every such
// sequence as visible escapes instead.
//
// Machine-readable payloads (Reporter.Data, piped Reporter.Diff) must stay
// byte-for-byte and are never passed through this package.
package sanitize

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Line returns s as a single line: every control character, including LF and
// TAB, is escaped. Use it for fields rendered on one line (identifiers, names,
// table cells, labels, short messages).
func Line(s string) string {
	return escape(s, false)
}

// Text is Line but keeps LF and TAB, for messages that are multi-line by
// design (errors listing several issues).
func Text(s string) string {
	return escape(s, true)
}

func escape(s string, keepLayout bool) string {
	if !needsEscape(s, keepLayout) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	lineStart := true
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			b.WriteString(`\x`)
			b.WriteString(strconv.FormatUint(uint64(s[i]), 16))
		case keepLayout && r == '\n':
			b.WriteRune(r)
			lineStart = true
			i += size
			continue
		case keepLayout && r == '\t':
			b.WriteRune(r)
			i += size
			continue
		case lineStart && unicode.IsSpace(r) && !isEscaped(r):
			// The runner trims leading whitespace before matching "::".
			b.WriteRune(r)
			i += size
			continue
		case lineStart && strings.HasPrefix(s[i:], "::"):
			b.WriteString(`\x3a`)
		case strings.HasPrefix(s[i:], "##["):
			b.WriteString(`#\x23`)
			size = 2
		case isEscaped(r):
			q := strconv.QuoteRune(r)
			b.WriteString(q[1 : len(q)-1])
		default:
			b.WriteRune(r)
		}
		lineStart = false
		i += size
	}
	return b.String()
}

// needsEscape is the fast path: most strings contain nothing to escape and are
// returned unchanged without allocating.
func needsEscape(s string, keepLayout bool) bool {
	if strings.Contains(s, "##[") || startsWithCommand(s) {
		return true
	}
	for i, r := range s {
		switch {
		case r == utf8.RuneError:
			if _, size := utf8.DecodeRuneInString(s[i:]); size == 1 {
				return true
			}
		case keepLayout && r == '\n':
			if startsWithCommand(s[i+1:]) {
				return true
			}
		case keepLayout && r == '\t':
		case isEscaped(r):
			return true
		}
	}
	return false
}

// startsWithCommand reports whether s, after leading whitespace, starts with
// the "::" prefix of a GitHub Actions workflow command.
func startsWithCommand(s string) bool {
	return strings.HasPrefix(strings.TrimLeftFunc(s, unicode.IsSpace), "::")
}

// isEscaped reports whether r is rendered as an escape: C0 / DEL / C1 control
// characters and the bidirectional formatting characters that can reorder
// what is displayed.
func isEscaped(r rune) bool {
	return unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r)
}
