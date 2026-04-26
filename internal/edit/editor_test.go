package edit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEditorCommand(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{name: "uses EDITOR when set", envValue: "nano", want: "nano"},
		{name: "defaults to vi when empty", envValue: "", want: "vi"},
		{name: "trims surrounding whitespace", envValue: "  emacs  ", want: "emacs"},
		{name: "preserves internal spaces and args", envValue: "code --wait", want: "code --wait"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.envValue)
			got := editorCommand()
			if got != tt.want {
				t.Errorf("editorCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditBufferLaunchesEditor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shell editor not available on windows")
	}
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "fake-editor.sh")
	script := `#!/bin/sh
echo ' edited' >> "$1"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake editor: %v", err)
	}

	t.Setenv("EDITOR", scriptPath)

	original := []byte("initial content")
	name, edited, err := editBuffer(original, ".txt")
	if err != nil {
		t.Fatalf("editBuffer failed: %v", err)
	}

	if name != scriptPath {
		t.Errorf("editor name = %q, want %q", name, scriptPath)
	}

	got := string(edited)
	if !strings.HasPrefix(got, "initial content") {
		t.Errorf("expected edited content to start with original, got %q", got)
	}
	if !strings.Contains(got, "edited") {
		t.Errorf("expected edited content to contain 'edited', got %q", got)
	}
}

// TestEditBufferHandlesPathWithSpaces verifies that $EDITOR pointing to a
// directory containing spaces works under sh -c parsing. This is the case
// that the previous strings.Fields-based implementation broke.
func TestEditBufferHandlesPathWithSpaces(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell parsing not used on windows")
	}
	parent := t.TempDir()
	dirWithSpaces := filepath.Join(parent, "dir with spaces")
	if err := os.MkdirAll(dirWithSpaces, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	scriptPath := filepath.Join(dirWithSpaces, "ed.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho ok > \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	// Quote the editor path the way a real user would in a shell-parsed env.
	t.Setenv("EDITOR", "'"+scriptPath+"'")

	_, edited, err := editBuffer([]byte("seed"), ".txt")
	if err != nil {
		t.Fatalf("editBuffer failed: %v", err)
	}
	if !strings.Contains(string(edited), "ok") {
		t.Errorf("expected edited content to contain 'ok', got %q", edited)
	}
}

func TestEditBufferForwardsEditorArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shell editor not available on windows")
	}
	// Editor script that writes its received argv (excluding $0) to a record file.
	tempDir := t.TempDir()
	recordPath := filepath.Join(tempDir, "argv.txt")
	scriptPath := filepath.Join(tempDir, "argv-editor.sh")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + recordPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake editor: %v", err)
	}

	t.Setenv("EDITOR", scriptPath+" --wait extra")

	if _, _, err := editBuffer([]byte("x"), ".txt"); err != nil {
		t.Fatalf("editBuffer failed: %v", err)
	}

	got, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("failed to read record: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 args (--wait, extra, tmpPath), got %d: %v", len(lines), lines)
	}
	if lines[0] != "--wait" || lines[1] != "extra" {
		t.Errorf("expected args ['--wait','extra', <tmp>], got %v", lines)
	}
	if !strings.HasSuffix(lines[2], ".txt") {
		t.Errorf("expected last arg to be temp file ending in .txt, got %q", lines[2])
	}
}

func TestEditBufferPropagatesEditorFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shell editor not available on windows")
	}
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "fail-editor.sh")
	script := `#!/bin/sh
exit 2
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake editor: %v", err)
	}

	t.Setenv("EDITOR", scriptPath)

	if _, _, err := editBuffer([]byte("x"), ".txt"); err == nil {
		t.Fatal("expected error when editor fails")
	}
}

func TestEditBufferCleansUpTempFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shell editor not available on windows")
	}
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "record-editor.sh")
	recordPath := filepath.Join(tempDir, "path.txt")
	script := `#!/bin/sh
echo "$1" > "` + recordPath + `"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake editor: %v", err)
	}

	t.Setenv("EDITOR", scriptPath)

	if _, _, err := editBuffer([]byte("x"), ".json"); err != nil {
		t.Fatalf("editBuffer failed: %v", err)
	}

	recorded, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("failed to read recorded path: %v", err)
	}
	tmpPath := strings.TrimSpace(string(recorded))
	if !strings.HasSuffix(tmpPath, ".json") {
		t.Errorf("expected temp file to have .json extension, got %q", tmpPath)
	}
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("expected temp file to be cleaned up, stat err = %v", err)
	}
}
