//go:build linux

package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestReplacedExecutableIsDetected replaces a running executable the same way download does and verifies the
// interpretation of /proc/<pid>/exe.
func TestReplacedExecutableIsDetected(t *testing.T) {
	src, err := os.ReadFile("/bin/sleep")
	if err != nil {
		t.Skip("no /bin/sleep available")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "abc")
	if err := os.WriteFile(target, src, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(target, "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	exe := fmt.Sprintf("/proc/%d/exe", cmd.Process.Pid)
	link, err := os.Readlink(exe)
	if err != nil {
		t.Fatal(err)
	}

	if isReplacedExecutable(link, target) {
		t.Fatalf("a fresh process must not be reported: %q", link)
	}

	tmp, err := os.CreateTemp(dir, "abc.*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tmp.Write(src)
	_ = tmp.Close()

	if err := os.Rename(tmp.Name(), target); err != nil {
		t.Fatal(err)
	}

	link, err = os.Readlink(exe)
	if err != nil {
		t.Fatal(err)
	}

	if !isReplacedExecutable(link, target) {
		t.Fatalf("replaced executable not detected: %q", link)
	}
}
