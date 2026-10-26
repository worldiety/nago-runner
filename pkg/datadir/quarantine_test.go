package datadir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQuarantineDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ngr", "abc")
	writeFile(t, filepath.Join(dir, "db", "data.bin"))

	dst, err := Quarantine(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected original dir to be gone, got %v", err)
	}

	assertFile(t, filepath.Join(dst, "db", "data.bin"))
	if filepath.Dir(dst) != filepath.Join(root, "ngr", QuarantineDirName) {
		t.Fatalf("unexpected quarantine location: %s", dst)
	}
}

func TestQuarantineSymlink(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "private", "ngr", "abc")
	writeFile(t, filepath.Join(private, "data.bin"))

	link := filepath.Join(root, "ngr", "abc")
	if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../private/ngr/abc", link); err != nil {
		t.Fatal(err)
	}

	dst, err := Quarantine(link)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("expected link to be removed, got %v", err)
	}

	assertFile(t, filepath.Join(dst, "data.bin"))
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	if filepath.Dir(dst) != filepath.Join(resolvedRoot, "private", "ngr", QuarantineDirName) {
		t.Fatalf("expected quarantine next to the resolved dir, got %s", dst)
	}
}

func TestQuarantineMissing(t *testing.T) {
	dst, err := Quarantine(filepath.Join(t.TempDir(), "nope"))
	if err != nil || dst != "" {
		t.Fatalf("expected no-op, got %q %v", dst, err)
	}
}

func writeFile(t *testing.T, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("precious"), 0600); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, name string) {
	t.Helper()
	buf, err := os.ReadFile(name)
	if err != nil || string(buf) != "precious" {
		t.Fatalf("expected preserved file %s: %v", name, err)
	}
}
