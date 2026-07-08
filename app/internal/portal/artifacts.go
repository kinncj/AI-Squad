package portal

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kinncj/maple/app/internal/state"
)

type artifact struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Platform string `json:"platform"`
	Source   string `json:"source"`
	Created  int64  `json:"created"`
	Updated  int64  `json:"updated"`
}

var artifactExts = map[string]bool{
	".excalidraw": true, ".json": true, ".jpg": true, ".jpeg": true, ".png": true,
	".webp": true, ".gif": true, ".html": true, ".htm": true, ".txt": true,
	".md": true, ".svg": true, ".css": true, ".mp4": true, ".webm": true,
}

func artifactKind(ext string) string {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg":
		return "image"
	case ".html", ".htm":
		return "preview"
	case ".mp4", ".webm":
		return "video"
	case ".md", ".txt", ".excalidraw", ".json", ".css":
		return "text"
	}
	return "file"
}

func artifactPlatform(rel string) string {
	l := strings.ToLower(rel)
	switch {
	case strings.HasSuffix(l, ".html"), strings.HasSuffix(l, ".htm"),
		strings.HasSuffix(l, ".tsx"), strings.HasSuffix(l, ".jsx"):
		return "web"
	case strings.Contains(l, "/tui") || strings.Contains(l, ".tui."):
		return "tui"
	}
	return "general"
}

// fileTimes returns (created, updated) unix seconds. Created uses birthtime where the OS
// exposes it (macOS), else falls back to the modification time.
func fileTimes(fi os.FileInfo) (int64, int64) {
	updated := fi.ModTime().Unix()
	created := birthTime(fi)
	if created == 0 {
		created = updated
	}
	return created, updated
}

// discoverArtifacts globs the design tree plus the declared manifest.
func (s *Server) discoverArtifacts() []artifact {
	seen := map[string]bool{}
	var out []artifact

	add := func(rel, source string) {
		rel = filepath.ToSlash(rel)
		if seen[rel] {
			return
		}
		full := filepath.Join(s.root, rel)
		fi, err := os.Stat(full)
		if err != nil || fi.IsDir() {
			return
		}
		if !artifactExts[strings.ToLower(filepath.Ext(rel))] {
			return
		}
		seen[rel] = true
		created, updated := fileTimes(fi)
		out = append(out, artifact{
			Path: rel, Name: filepath.Base(rel), Kind: artifactKind(strings.ToLower(filepath.Ext(rel))),
			Platform: artifactPlatform(rel), Source: source, Created: created, Updated: updated,
		})
	}

	// 1. declared manifest (.claude/state/design-artifacts.json)
	if b, err := os.ReadFile(filepath.Join(s.root, ".claude", "state", "design-artifacts.json")); err == nil {
		var raw struct {
			Items []any `json:"items"`
		}
		if json.Unmarshal(b, &raw) == nil {
			for _, it := range raw.Items {
				switch v := it.(type) {
				case string:
					add(v, "manifest")
				case map[string]any:
					if p, _ := v["path"].(string); p != "" {
						add(p, "manifest")
					}
				}
			}
		}
	}

	// 2. globbed design tree
	_ = filepath.Walk(filepath.Join(s.root, "docs", "design"), func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		if rel, e := filepath.Rel(s.root, p); e == nil {
			add(rel, "design")
		}
		return nil
	})

	sort.Slice(out, func(i, j int) bool { return out[i].Updated > out[j].Updated })
	return out
}

func (s *Server) handleArtifacts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.discoverArtifacts()})
}

// handleStory returns one story's full spec (Story.md content) plus the design artifacts
// linked to it (path contains the story id/slug) — the portal's story-review entry point.
func (s *Server) handleStory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	var found *state.Story
	for _, st := range s.fs.Stories() {
		if st.ID == id {
			cp := st
			found = &cp
			break
		}
	}
	if found == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "story not found: " + id})
		return
	}
	spec := ""
	if b, err := os.ReadFile(found.Path); err == nil {
		spec = string(b)
	}
	var linked []artifact
	for _, a := range s.discoverArtifacts() {
		if strings.Contains(a.Path, found.ID) || (found.Slug != "" && strings.Contains(a.Path, found.Slug)) {
			linked = append(linked, a)
		}
	}
	title := found.Title
	if title == "" {
		title = found.ID
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": found.ID, "title": title, "phase": found.Phase, "ui": found.UI,
		"status": found.RunStatus, "spec": spec, "artifacts": linked,
	})
}

// handleStories returns the project's stories with their taffy-reported run status, so the
// portal shows the same per-story progress the TUI does.
func (s *Server) handleStories(w http.ResponseWriter, _ *http.Request) {
	out := []map[string]any{}
	for _, st := range s.fs.Stories() {
		title := st.Title
		if title == "" {
			title = st.ID
		}
		out = append(out, map[string]any{
			"id": st.ID, "title": title, "phase": st.Phase,
			"priority": st.Priority, "ui": st.UI, "status": st.RunStatus,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// handleArtifactBytes serves a raw artifact file for previews, strictly confined to root.
func (s *Server) handleArtifactBytes(w http.ResponseWriter, r *http.Request) {
	if !s.validToken(r) {
		http.Error(w, "invalid token", http.StatusForbidden)
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/artifact/")
	full := filepath.Clean(filepath.Join(s.root, rel))
	if !strings.HasPrefix(full, filepath.Clean(s.root)+string(os.PathSeparator)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, full)
}

// ─── uploads ──────────────────────────────────────────────────────────────

func (s *Server) uploadDir() string {
	return filepath.Join(s.root, ".claude", "state", "review-uploads")
}

func (s *Server) handleUploads(w http.ResponseWriter, _ *http.Request) {
	var out []artifact
	entries, _ := os.ReadDir(s.uploadDir())
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(s.root, filepath.Join(s.uploadDir(), e.Name()))
		created, updated := fileTimes(fi)
		out = append(out, artifact{
			Path: filepath.ToSlash(rel), Name: e.Name(), Kind: artifactKind(strings.ToLower(filepath.Ext(e.Name()))),
			Platform: artifactPlatform(e.Name()), Source: "upload", Created: created, Updated: updated,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Updated > out[j].Updated })
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad upload"})
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no files"})
		return
	}
	_ = os.MkdirAll(s.uploadDir(), 0o755)
	var saved []string
	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !artifactExts[ext] || fh.Size > 10<<20 {
			continue
		}
		src, err := fh.Open()
		if err != nil {
			continue
		}
		name := time.Now().UTC().Format("20060102T150405") + "-" + filepath.Base(fh.Filename)
		dst, err := os.Create(filepath.Join(s.uploadDir(), name))
		if err != nil {
			src.Close()
			continue
		}
		_, _ = io.Copy(dst, src)
		dst.Close()
		src.Close()
		saved = append(saved, name)
	}
	s.Publish(map[string]any{"event": "upload"})
	writeJSON(w, http.StatusOK, map[string]any{"saved": saved})
}
