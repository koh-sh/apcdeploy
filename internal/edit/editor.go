package edit

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// defaultEditor is used when $EDITOR is not set.
const defaultEditor = "vi"

// editBuffer writes the given content to a temporary file, launches the user's
// editor ($EDITOR or vi) on it, and returns the modified content. The temp file
// is removed before returning.
//
// $EDITOR may contain arguments (e.g. "code --wait"); whitespace-separated
// tokens are forwarded as additional arguments before the temp file path.
func editBuffer(content []byte, ext string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "apcdeploy-edit-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	name, args := resolveEditorCommand()
	args = append(args, tmpPath)
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor %q failed: %w", name, err)
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read edited file: %w", err)
	}
	return edited, nil
}

// resolveEditorCommand parses $EDITOR (falling back to vi) into an executable
// name and its pre-configured arguments. Splitting on whitespace keeps parity
// with common tools (git, crontab) for values like "code --wait".
func resolveEditorCommand() (string, []string) {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		return defaultEditor, nil
	}
	parts := strings.Fields(editor)
	return parts[0], parts[1:]
}
