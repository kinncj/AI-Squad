package portalsock

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// shortRoot returns a short temp dir — unix socket paths are capped near 104 chars on
// macOS, and t.TempDir() paths are far too long.
func shortRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "ps")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// startServer opens a unix socket under root/.claude/state and writes the addr file,
// mimicking the portal. Returns a channel of received JSON lines and a stop func.
func startServer(t *testing.T, root string) (<-chan map[string]any, func()) {
	t.Helper()
	dir := filepath.Join(root, ".claude", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(dir, "maple.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	// Portal writes the addr descriptor clients read.
	os.WriteFile(filepath.Join(dir, "maple-sock.addr"), []byte("unix:"+sock+"\n"), 0o644)

	lines := make(chan map[string]any, 16)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				sc := bufio.NewScanner(c)
				for sc.Scan() {
					var m map[string]any
					if json.Unmarshal(sc.Bytes(), &m) == nil {
						lines <- m
					}
				}
			}(c)
		}
	}()
	return lines, func() { ln.Close() }
}

func TestEmitSendsEvent(t *testing.T) {
	root := shortRoot(t)
	lines, stop := startServer(t, root)
	defer stop()

	if err := Emit(root, map[string]any{"event": "stage", "stage": "wireframe"}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	select {
	case got := <-lines:
		if got["event"] != "stage" || got["stage"] != "wireframe" {
			t.Errorf("server received %v, want stage/wireframe", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive the emitted event")
	}
}

func TestHoldRegistersTUIAndStops(t *testing.T) {
	root := shortRoot(t)
	lines, stop := startServer(t, root)
	defer stop()

	cancel := Hold(root)
	select {
	case got := <-lines:
		if got["role"] != "tui" {
			t.Errorf("Hold should register as tui, got %v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Hold did not connect + register")
	}
	cancel() // should close the held connection without hanging
	time.Sleep(50 * time.Millisecond)
}

func TestResolvePrefersAddrFileThenUnixDefault(t *testing.T) {
	root := shortRoot(t)
	if _, _, ok := resolve(root); ok {
		t.Error("no socket present should resolve ok=false")
	}
	dir := filepath.Join(root, ".claude", "state")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "maple-sock.addr"), []byte("tcp:127.0.0.1:7901\n"), 0o644)
	n, a, ok := resolve(root)
	if !ok || n != "tcp" || a != "127.0.0.1:7901" {
		t.Errorf("addr file resolve = %q/%q ok=%v", n, a, ok)
	}
}

func TestEmitNoServerErrors(t *testing.T) {
	if err := Emit(t.TempDir(), map[string]any{"event": "x"}); err == nil {
		t.Error("Emit with no portal should return an error")
	}
}
