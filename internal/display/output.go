package display

import "fmt"

// successMsg formats a success message
func successMsg(message string) string {
	return fmt.Sprintf("✓ %s", message)
}

// errorMsg formats an error message
func errorMsg(message string) string {
	return fmt.Sprintf("✗ %s", message)
}

// warningMsg formats a warning message
func warningMsg(message string) string {
	return fmt.Sprintf("⚠ %s", message)
}

// progressMsg formats a progress message
func progressMsg(message string) string {
	return fmt.Sprintf("⏳ %s", message)
}

// bold formats text in bold (using ANSI codes)
func bold(text string) string {
	return fmt.Sprintf("\033[1m%s\033[0m", text)
}

// separator returns a separator line
func separator() string {
	return "────────────────────────────────────────"
}
