// Package portal is the Go-native design-review portal: an in-process net/http server
// that serves the embedded SPA and a small JSON API backed by state.FS. It replaces the
// standalone Python server (ADR-0002) — no python3 dependency, one source of truth for
// pipeline/gate state, and event-driven updates over SSE instead of browser polling.
package portal

import (
	"encoding/json"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kinncj/maple/app/internal/state"
)

//go:embed index.html
var indexHTML string

// Server serves the portal for a single project root.
type Server struct {
	root  string
	token string
	fs    *state.FS

	mu    sync.Mutex
	subs  map[chan string]struct{} // SSE subscribers
	getenv func(string) string

	actMu    sync.Mutex
	activity []map[string]any // in-memory ring of recent events
	actSeen  int              // lines of the activity log already processed
}

// New builds a portal server for root with an access token.
func New(root, token string) *Server {
	return &Server{
		root:   root,
		token:  token,
		fs:     state.NewFS(root),
		subs:   map[chan string]struct{}{},
		getenv: os.Getenv,
	}
}

// Handler returns the HTTP mux for the portal.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/state", s.tokened(s.handleState))
	mux.HandleFunc("/api/artifacts", s.tokened(s.handleArtifacts))
	mux.HandleFunc("/api/stories", s.tokened(s.handleStories))
	mux.HandleFunc("/api/story", s.tokened(s.handleStory))
	mux.HandleFunc("/api/activity", s.tokened(s.handleActivity))
	mux.HandleFunc("/api/uploads", s.tokened(s.handleUploads))
	mux.HandleFunc("/api/upload", s.tokened(s.handleUpload))
	mux.HandleFunc("/api/approve", s.tokened(s.handleApprove))
	mux.HandleFunc("/api/reject", s.tokened(s.handleReject))
	mux.HandleFunc("/api/request-changes", s.tokened(s.handleRequestChanges))
	mux.HandleFunc("/api/stop", s.tokened(s.handleStop))
	mux.HandleFunc("/api/token", s.handleToken)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/artifact/", s.handleArtifactBytes)
	mux.HandleFunc("/", s.handleIndex)
	return mux
}

// Serve starts the portal, watching for state changes to push SSE events, and blocks.
func (s *Server) Serve(addr string) error {
	go s.watch()
	return http.ListenAndServe(addr, s.Handler())
}

// ─── event bus ──────────────────────────────────────────────────────────────

// Publish broadcasts a JSON event line to all SSE subscribers.
func (s *Server) Publish(event map[string]any) {
	b, err := json.Marshal(event)
	if err != nil {
		return
	}
	s.mu.Lock()
	for ch := range s.subs {
		select {
		case ch <- string(b):
		default:
		}
	}
	s.mu.Unlock()
}

// watch polls the state files' mtimes and publishes a "change" event when any changes, so
// the browser refreshes only on real change instead of constant polling.
func (s *Server) watch() {
	files := []string{
		filepath.Join(s.root, ".claude", "state", "maple.json"),
		filepath.Join(s.root, ".claude", "state", "approval-pending.txt"),
		filepath.Join(s.root, ".claude", "state", "design-artifacts.json"),
	}
	last := ""
	for {
		var sig strings.Builder
		for _, f := range files {
			if fi, err := os.Stat(f); err == nil {
				fmt.Fprintf(&sig, "%s:%d;", filepath.Base(f), fi.ModTime().UnixNano())
			}
		}
		// also fold in the design dir's newest mtime
		sig.WriteString(fmt.Sprintf("art:%d", s.designDirSig()))
		if cur := sig.String(); cur != last {
			last = cur
			s.Publish(map[string]any{"event": "change", "ts": time.Now().UTC().Format(time.RFC3339)})
		}
		s.tailActivity() // push any agent-emitted events over SSE
		time.Sleep(1500 * time.Millisecond)
	}
}

func (s *Server) designDirSig() int64 {
	var newest int64
	_ = filepath.Walk(filepath.Join(s.root, "docs", "design"), func(p string, fi os.FileInfo, err error) error {
		if err == nil && fi != nil && !fi.IsDir() {
			if m := fi.ModTime().UnixNano(); m > newest {
				newest = m
			}
		}
		return nil
	})
	return newest
}

// ─── helpers ────────────────────────────────────────────────────────────────

func (s *Server) validToken(r *http.Request) bool {
	if s.token == "" {
		return true
	}
	return r.URL.Query().Get("token") == s.token || r.Header.Get("X-Maple-Token") == s.token
}

func (s *Server) tokened(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validToken(r) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "invalid token"})
			return
		}
		h(w, r)
	}
}

// handleToken lets the SPA (re)fetch its token — served to any local caller.
func (s *Server) handleToken(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"token": s.token})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) readBody(r *http.Request) map[string]any {
	var m map[string]any
	_ = json.NewDecoder(r.Body).Decode(&m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}

// ─── read endpoints ─────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/index.html") {
		// SPA routing: any non-api, non-artifact path serves the app (agent-guessed URLs).
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/artifact/") {
			http.NotFound(w, r)
			return
		}
	}
	tok, _ := json.Marshal(s.token)
	html := strings.Replace(indexHTML, `"__MAPLE_PORTAL_TOKEN__"`, string(tok), 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	p := s.fs.Pipeline()
	out := map[string]any{
		"taffy":            p.Taffy,
		"stage":            p.Stage,
		"phase":            p.Phase,
		"status":           p.Status,
		"harness":          p.Harness,
		"approval_pending": s.fs.ApprovalPending(),
		"updated_at":       s.updatedAt(),
		"maple":            map[string]any{"connected": true, "source": "in-process"},
		"feedback":         s.readFeedback(),
	}
	// What is the pending gate actually asking a human to review? (story + files)
	if rv, ok := s.fs.Review(); ok {
		out["review"] = map[string]any{
			"story": rv.Story, "title": rv.Title, "stage": rv.Stage,
			"artifacts": rv.Artifacts, "inferred": rv.Inferred,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) updatedAt() string {
	if b, err := os.ReadFile(s.mapleJSON()); err == nil {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			if v, ok := m["updated_at"].(string); ok {
				return v
			}
		}
	}
	return ""
}

// ─── SSE ────────────────────────────────────────────────────────────────────

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 16)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()

	fmt.Fprintf(w, "data: %s\n\n", `{"event":"hello"}`)
	fl.Flush()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			fl.Flush()
		case <-ping.C:
			fmt.Fprintf(w, ": ping\n\n")
			fl.Flush()
		}
	}
}

func (s *Server) mapleJSON() string {
	return filepath.Join(s.root, ".claude", "state", "maple.json")
}
func (s *Server) feedbackFile() string {
	return filepath.Join(s.root, ".claude", "state", "design-feedback.json")
}

func (s *Server) readFeedback() map[string]any {
	b, err := os.ReadFile(s.feedbackFile())
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	return m
}

