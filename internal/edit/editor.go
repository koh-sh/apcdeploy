package edit

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// defaultEditor is used when $EDITOR is not set.
const defaultEditor = "vi"

// editBuffer writes content to a temp file, launches $EDITOR (vi fallback) on
// it, and returns the modified content. The temp file is removed before return.
//
// Returns the display name of the launched editor along with the edited bytes
// so callers can include it in progress output without re-parsing $EDITOR.
//
// On Unix, $EDITOR is parsed by sh, which lets users put spaces in the path
// (e.g. EDITOR='/Applications/My Editor/bin/edit') and pre-set arguments
// (e.g. EDITOR='code --wait'). The temp file is passed as the first positional
// argument so its path never needs to be escaped. This matches how git
// launches GIT_EDITOR (RUN_USING_SHELL).
//
// On Windows, $EDITOR is split by whitespace and the first token is exec'd
// directly. Paths with embedded spaces are not supported there.
func editBuffer(content []byte, ext string) (editorName string, edited []byte, err error) {
	tmp, err := os.CreateTemp("", "apcdeploy-edit-*"+ext)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	editorSpec := editorCommand()
	cmd := buildEditorCmd(editorSpec, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return editorSpec, nil, fmt.Errorf("editor %q failed: %w", editorSpec, err)
	}

	edited, err = os.ReadFile(tmpPath)
	if err != nil {
		return editorSpec, nil, fmt.Errorf("failed to read edited file: %w", err)
	}
	return editorSpec, edited, nil
}

// editorCommand returns the raw $EDITOR string (or the default).
func editorCommand() string {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		return defaultEditor
	}
	return editor
}

// buildEditorCmd constructs the exec.Cmd that runs the editor on tmpPath.
//
// Unix: invokes `sh -c '<editor> "$@"' -- <tmpPath>` so the editor string is
// shell-parsed (handling quotes, spaces in paths) while tmpPath is forwarded
// as a positional argument that never needs quoting.
//
// Windows: falls back to whitespace splitting since cmd.exe quoting rules
// differ. Paths with embedded spaces in $EDITOR are not supported there.
func buildEditorCmd(editor, tmpPath string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		parts := strings.Fields(editor)
		args := make([]string, 0, len(parts))
		args = append(args, parts[1:]...)
		args = append(args, tmpPath)
		return exec.Command(parts[0], args...)
	}
	return exec.Command("sh", "-c", editor+` "$@"`, "--", tmpPath)
}
