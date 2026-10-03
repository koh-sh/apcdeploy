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

// bufferFile is the on-disk edit buffer handed back by editBuffer. It lives
// inside a private 0700 directory that the caller owns once editBuffer
// returns successfully.
type bufferFile struct {
	// Path is the buffer file the editor wrote to.
	Path string
	dir  string
}

// Remove deletes the buffer file together with its private directory.
func (b *bufferFile) Remove() error {
	return os.RemoveAll(b.dir)
}

// editBuffer writes content to a temp file, launches $EDITOR (vi fallback) on
// it, and returns the modified content.
//
// Returns the display name of the launched editor along with the edited bytes
// so callers can include it in progress output without re-parsing $EDITOR.
//
// Buffer ownership: when editBuffer returns an error, no edited content
// exists yet and the private directory has already been removed. On success
// the buffer is left on disk and the caller MUST either Remove it (the edit
// was deployed or was a no-op) or keep it and surface its Path so the user
// can recover their edits after a later failure (issue #150).
//
// On Unix, $EDITOR is parsed by sh, which lets users put spaces in the path
// (e.g. EDITOR='/Applications/My Editor/bin/edit') and pre-set arguments
// (e.g. EDITOR='code --wait'). The temp file is passed as the first positional
// argument so its path never needs to be escaped. This matches how git
// launches GIT_EDITOR (RUN_USING_SHELL).
//
// On Windows, $EDITOR is split by whitespace and the first token is exec'd
// directly. Paths with embedded spaces are not supported there.
func editBuffer(content []byte, ext string) (editorName string, edited []byte, buf *bufferFile, err error) {
	// Allocate a private 0700 directory under the system temp root and
	// place the buffer file inside it. os.CreateTemp alone leaves the
	// file in a world-traversable location with a predictable
	// "apcdeploy-edit-*" prefix, so a co-tenant on the same UID can poll
	// for the buffer while the editor is open. Owning the parent
	// directory removes both the listability and the predictable name.
	// The same protection covers a buffer kept after a failed deploy.
	tmpDir, err := os.MkdirTemp("", "apcdeploy-edit-*")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	// Remove the directory on every error path; on success ownership
	// passes to the caller via the returned bufferFile.
	defer func() {
		if err != nil {
			os.RemoveAll(tmpDir)
		}
	}()

	tmp, err := os.CreateTemp(tmpDir, "buffer-*"+ext)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return "", nil, nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", nil, nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	editorSpec := editorCommand()
	cmd := buildEditorCmd(editorSpec, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return editorSpec, nil, nil, fmt.Errorf("editor %q failed: %w", editorSpec, err)
	}

	edited, err = os.ReadFile(tmpPath)
	if err != nil {
		return editorSpec, nil, nil, fmt.Errorf("failed to read edited file: %w", err)
	}
	return editorSpec, edited, &bufferFile{Path: tmpPath, dir: tmpDir}, nil
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
