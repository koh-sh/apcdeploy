package edit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEditorCommand(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantName string
		wantArgs []string
	}{
		{name: "uses EDITOR when set", envValue: "nano", wantName: "nano"},
		{name: "defaults to vi when empty", envValue: "", wantName: "vi"},
		{name: "trims whitespace", envValue: "  emacs  ", wantName: "emacs"},
		{name: "splits args (code --wait)", envValue: "code --wait", wantName: "code", wantArgs: []string{"--wait"}},
		{name: "preserves multiple args", envValue: "vim -n -u NONE", wantName: "vim", wantArgs: []string{"-n", "-u", "NONE"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.envValue)
			gotName, gotArgs := resolveEditorCommand()
			if gotName != tt.wantName {
				t.Errorf("resolveEditorCommand() name = %q, want %q", gotName, tt.wantName)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("resolveEditorCommand() args = %v, want %v", gotArgs, tt.wantArgs)
			}
			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("resolveEditorCommand() args[%d] = %q, want %q", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestEditBufferLaunchesEditor(t *testing.T) {
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
	edited, err := editBuffer(original, ".txt")
	if err != nil {
		t.Fatalf("editBuffer failed: %v", err)
	}

	got := string(edited)
	if !strings.HasPrefix(got, "initial content") {
		t.Errorf("expected edited content to start with original, got %q", got)
	}
	if !strings.Contains(got, "edited") {
		t.Errorf("expected edited content to contain 'edited', got %q", got)
	}
}

func TestEditBufferForwardsEditorArgs(t *testing.T) {
	// Editor script that writes its received argv (excluding $0) to a record file.
	tempDir := t.TempDir()
	recordPath := filepath.Join(tempDir, "argv.txt")
	scriptPath := filepath.Join(tempDir, "argv-editor.sh")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + recordPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake editor: %v", err)
	}

	t.Setenv("EDITOR", scriptPath+" --wait extra")

	if _, err := editBuffer([]byte("x"), ".txt"); err != nil {
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
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "fail-editor.sh")
	script := `#!/bin/sh
exit 2
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake editor: %v", err)
	}

	t.Setenv("EDITOR", scriptPath)

	_, err := editBuffer([]byte("x"), ".txt")
	if err == nil {
		t.Fatal("expected error when editor fails")
	}
}

func TestEditBufferCleansUpTempFile(t *testing.T) {
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

	if _, err := editBuffer([]byte("x"), ".json"); err != nil {
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
