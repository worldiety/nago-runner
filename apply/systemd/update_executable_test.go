package systemd

import (
	"bytes"
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/pkg/linux"
)

func executable(t *testing.T, buf []byte) configuration.Executable {
	t.Helper()
	hash, err := linux.Sha3Bytes(buf)
	if err != nil {
		t.Fatal(err)
	}

	return configuration.Executable{Size: int64(len(buf)), Hash: hash}
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, f := range files {
		names = append(names, f.Name())
	}

	return names
}

func TestDownloadReplacesTarget(t *testing.T) {
	payload := make([]byte, 3<<20)
	_, _ = rand.Read(payload)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "abc")
	if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := download(slog.Default(), srv.URL, "secret", executable(t, payload), target); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got, payload) {
		t.Fatal("target does not contain the download")
	}

	info, _ := os.Stat(target)
	if info.Mode().Perm()&0100 == 0 {
		t.Fatalf("target is not executable: %v", info.Mode())
	}

	if names := entries(t, dir); len(names) != 1 {
		t.Fatalf("temporary files left: %v", names)
	}
}

func TestDownloadKeepsTargetOnHashMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("tampered"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "abc")
	if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	want := executable(t, []byte("expected"))
	err := download(slog.Default(), srv.URL, "", want, target)
	if err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("expected hash mismatch, got %v", err)
	}

	if got, _ := os.ReadFile(target); string(got) != "old" {
		t.Fatalf("target was modified: %q", got)
	}

	if names := entries(t, dir); len(names) != 1 {
		t.Fatalf("temporary files left: %v", names)
	}
}

func TestDownloadFailsOnStall(t *testing.T) {
	old := downloadStallTimeout
	downloadStallTimeout = 200 * time.Millisecond
	defer func() { downloadStallTimeout = old }()

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	dir := t.TempDir()
	target := filepath.Join(dir, "abc")

	start := time.Now()
	err := download(slog.Default(), srv.URL, "", configuration.Executable{Size: 1000, Hash: "x"}, target)
	if err == nil {
		t.Fatal("expected an error for a stalled download")
	}

	if time.Since(start) > 5*time.Second {
		t.Fatalf("stall was not detected in time: %v", time.Since(start))
	}

	if names := entries(t, dir); len(names) != 0 {
		t.Fatalf("temporary files left: %v", names)
	}
}

func TestDownloadToleratesSlowProgress(t *testing.T) {
	old := downloadStallTimeout
	downloadStallTimeout = 300 * time.Millisecond
	defer func() { downloadStallTimeout = old }()

	payload := bytes.Repeat([]byte("x"), 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 10 chunks of 100ms each take longer than the stall timeout in total
		for _, b := range payload {
			_, _ = w.Write([]byte{b})
			w.(http.Flusher).Flush()
			time.Sleep(100 * time.Millisecond)
		}
	}))
	defer srv.Close()

	target := filepath.Join(t.TempDir(), "abc")
	if err := download(slog.Default(), srv.URL, "", executable(t, payload), target); err != nil {
		t.Fatal(err)
	}
}

func TestIsReplacedExecutable(t *testing.T) {
	if !isReplacedExecutable("/opt/ngr/abc (deleted)", "/opt/ngr/abc") {
		t.Fatal("expected a replaced executable")
	}

	if isReplacedExecutable("/opt/ngr/abc", "/opt/ngr/abc") {
		t.Fatal("the current executable is not replaced")
	}

	if isReplacedExecutable("/opt/ngr/abcd (deleted)", "/opt/ngr/abc") {
		t.Fatal("another executable must not match")
	}
}
