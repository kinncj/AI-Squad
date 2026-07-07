// Package portalsock is maple's client for the design-review portal's control socket.
// The portal listens on a Unix-domain socket (TCP localhost on Windows) and writes its
// address to .claude/state/maple-sock.addr. maple holds a persistent connection so the
// portal can show live "maple connected/offline", and anything can Emit newline-JSON
// events the portal broadcasts to the browser. State stays file-based; this is the
// real-time signal layer on top.
package portalsock

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	addrFile      = ".claude/state/maple-sock.addr"
	defaultUnix   = ".claude/state/maple.sock"
	dialTimeout   = 2 * time.Second
	reconnectWait = 2 * time.Second
)

// resolve returns the network ("unix"/"tcp") and address to dial. It reads the addr
// file the portal writes; if absent it falls back to the conventional unix path (which
// exists whenever the portal is up on macOS/Linux). ok is false when neither is present.
func resolve(root string) (network, address string, ok bool) {
	if data, err := os.ReadFile(filepath.Join(root, addrFile)); err == nil {
		s := strings.TrimSpace(string(data))
		if n, a, found := strings.Cut(s, ":"); found && n != "" && a != "" {
			if n == "unix" && !filepath.IsAbs(a) {
				a = filepath.Join(root, a)
			}
			return n, a, true
		}
	}
	unix := filepath.Join(root, defaultUnix)
	if _, err := os.Stat(unix); err == nil {
		return "unix", unix, true
	}
	return "", "", false
}

func dial(root string) (net.Conn, error) {
	network, address, ok := resolve(root)
	if !ok {
		return nil, fmt.Errorf("portalsock: no portal socket found")
	}
	d := net.Dialer{Timeout: dialTimeout}
	return d.Dial(network, address)
}

// Emit connects to the portal socket, sends one newline-terminated JSON event, and
// closes. Safe to call when no portal is running (returns an error the caller ignores).
func Emit(root string, event map[string]any) error {
	conn, err := dial(root)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(dialTimeout))
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(line, '\n'))
	return err
}

// Hold keeps a persistent connection to the portal so it can show maple as connected.
// It registers as the TUI, reconnects if the portal restarts, and returns a stop func
// that closes the connection (so the portal sees maple go offline immediately). No-op
// aside from background retries when the portal isn't running.
func Hold(root string) func() {
	ctx, cancel := context.WithCancel(context.Background())
	go holdLoop(ctx, root)
	return cancel
}

func holdLoop(ctx context.Context, root string) {
	for {
		if ctx.Err() != nil {
			return
		}
		conn, err := dial(root)
		if err != nil {
			if sleepCtx(ctx, reconnectWait) {
				return
			}
			continue
		}
		// Register as the persistent TUI connection, then hold until it drops or we stop.
		_, _ = conn.Write([]byte(`{"role":"tui"}` + "\n"))
		closed := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				conn.Close()
			case <-closed:
			}
		}()
		waitClosed(conn) // blocks until the portal closes the connection
		close(closed)
		conn.Close()
		if sleepCtx(ctx, reconnectWait) {
			return
		}
	}
}

// waitClosed blocks until the peer closes conn (read returns EOF/error).
func waitClosed(conn net.Conn) {
	buf := make([]byte, 256)
	for {
		if _, err := conn.Read(buf); err != nil {
			return
		}
	}
}

// sleepCtx waits d or until ctx is cancelled; returns true if cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-t.C:
		return false
	}
}
