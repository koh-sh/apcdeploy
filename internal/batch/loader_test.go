package batch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeYML is a small helper that drops a YAML file into dir and returns
// its absolute path. Each test gets its own t.TempDir() so file collisions
// across cases are impossible.
func writeYML(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func goodYML(region, app, profile, env, dataFile string) string {
	return strings.Join([]string{
		"region: " + region,
		"application: " + app,
		"configuration_profile: " + profile,
		"environment: " + env,
		"data_file: " + dataFile,
	}, "\n") + "\n"
}

func TestLoadAll_SinglePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := writeYML(t, dir, "apcdeploy.yml", goodYML("us-east-1", "app", "profile", "prod", "data.json"))
	writeYML(t, dir, "data.json", `{"foo":1}`)

	targets, err := LoadAll([]string{cfg})
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("len(targets) = %d, want 1", len(targets))
	}
	if got, want := targets[0].Identifier, "us-east-1/app/profile/prod"; got != want {
		t.Errorf("Identifier = %q, want %q", got, want)
	}
	if targets[0].Path != cfg {
		t.Errorf("Path = %q, want input verbatim", targets[0].Path)
	}
}

func TestLoadAll_PreservesArgumentOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	a := writeYML(t, dir, "a/apcdeploy.yml", goodYML("us-east-1", "app", "profile", "dev", "data.json"))
	b := writeYML(t, dir, "b/apcdeploy.yml", goodYML("us-east-1", "app", "profile", "stg", "data.json"))
	c := writeYML(t, dir, "c/apcdeploy.yml", goodYML("us-east-1", "app", "profile", "prod", "data.json"))
	for _, d := range []string{"a", "b", "c"} {
		writeYML(t, dir, d+"/data.json", `{}`)
	}

	targets, err := LoadAll([]string{c, a, b})
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	got := []string{targets[0].Identifier, targets[1].Identifier, targets[2].Identifier}
	want := []string{
		"us-east-1/app/profile/prod",
		"us-east-1/app/profile/dev",
		"us-east-1/app/profile/stg",
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("targets[%d].Identifier = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadAll_DeduplicatesEquivalentPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := writeYML(t, dir, "apcdeploy.yml", goodYML("us-east-1", "app", "profile", "prod", "data.json"))
	writeYML(t, dir, "data.json", `{}`)

	// Same file referenced via three syntactically different absolute
	// forms — the loader must collapse them to one Target. We avoid
	// t.Chdir/os.Chdir here on purpose because t.Parallel() makes
	// process-global cwd mutation hostile to neighbouring tests.
	withDot := filepath.Join(filepath.Dir(cfg), ".", filepath.Base(cfg))
	withDotDot := filepath.Join(filepath.Dir(cfg), "..", filepath.Base(filepath.Dir(cfg)), filepath.Base(cfg))

	paths := []string{cfg, withDot, withDotDot}
	targets, err := LoadAll(paths)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("len(targets) = %d, want 1 (deduplicated)", len(targets))
	}
}

func TestLoadAll_DuplicateIdentifierIsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	a := writeYML(t, dir, "a/apcdeploy.yml", goodYML("us-east-1", "app", "profile", "prod", "data.json"))
	b := writeYML(t, dir, "b/apcdeploy.yml", goodYML("us-east-1", "app", "profile", "prod", "data.json"))
	for _, d := range []string{"a", "b"} {
		writeYML(t, dir, d+"/data.json", `{}`)
	}

	_, err := LoadAll([]string{a, b})
	if err == nil {
		t.Fatal("LoadAll: want error for duplicate identifier, got nil")
	}
	if !errors.Is(err, ErrDuplicateTarget) {
		t.Errorf("err = %v, want errors.Is(ErrDuplicateTarget)", err)
	}
	// Both paths should appear in the message so the user can locate
	// the conflict.
	msg := err.Error()
	if !strings.Contains(msg, a) || !strings.Contains(msg, b) {
		t.Errorf("err message must mention both paths, got: %s", msg)
	}
	if !strings.Contains(msg, "us-east-1/app/profile/prod") {
		t.Errorf("err message must mention identifier, got: %s", msg)
	}
}

func TestLoadAll_EmptyPathsIsError(t *testing.T) {
	t.Parallel()

	_, err := LoadAll(nil)
	if err == nil {
		t.Fatal("LoadAll(nil): want error, got nil")
	}
	_, err = LoadAll([]string{})
	if err == nil {
		t.Fatal("LoadAll([]): want error, got nil")
	}
}

func TestLoadAll_MissingFileSurfacesPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	good := writeYML(t, dir, "good.yml", goodYML("us-east-1", "app", "profile", "prod", "data.json"))
	writeYML(t, dir, "data.json", `{}`)
	missing := filepath.Join(dir, "no-such.yml")

	_, err := LoadAll([]string{good, missing})
	if err == nil {
		t.Fatal("LoadAll: want error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("err must mention the failing path %q, got: %v", missing, err)
	}
}

func TestLoadAll_InvalidYAMLSurfacesPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bad := writeYML(t, dir, "bad.yml", "::: not yaml :::")

	_, err := LoadAll([]string{bad})
	if err == nil {
		t.Fatal("LoadAll: want error for invalid yaml, got nil")
	}
	if !strings.Contains(err.Error(), bad) {
		t.Errorf("err must mention the failing path %q, got: %v", bad, err)
	}
}

func TestLoadAll_DuplicateRespectsRegionDifference(t *testing.T) {
	t.Parallel()

	// Same app/profile/env in different regions are distinct targets;
	// dedup must NOT collapse them.
	dir := t.TempDir()
	a := writeYML(t, dir, "a/apcdeploy.yml", goodYML("us-east-1", "app", "profile", "prod", "data.json"))
	b := writeYML(t, dir, "b/apcdeploy.yml", goodYML("eu-west-1", "app", "profile", "prod", "data.json"))
	for _, d := range []string{"a", "b"} {
		writeYML(t, dir, d+"/data.json", `{}`)
	}

	targets, err := LoadAll([]string{a, b})
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("len(targets) = %d, want 2", len(targets))
	}
}
