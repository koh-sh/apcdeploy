package display

import "fmt"

// Success formats a success message
func Success(message string) string {
	return fmt.Sprintf("✓ %s", message)
}

// Error formats an error message
func Error(message string) string {
	return fmt.Sprintf("✗ %s", message)
}

// Warning formats a warning message
func Warning(message string) string {
	return fmt.Sprintf("⚠ %s", message)
}

// Progress formats a progress message
func Progress(message string) string {
	return fmt.Sprintf("⏳ %s", message)
}

// Bold formats text in bold (using ANSI codes)
func Bold(text string) string {
	return fmt.Sprintf("\033[1m%s\033[0m", text)
}

// Separator returns a separator line
func Separator() string {
	return "────────────────────────────────────────"
}
