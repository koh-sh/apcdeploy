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

// Info formats an info message
func Info(message string) string {
	return message
}
