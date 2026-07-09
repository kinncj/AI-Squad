package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// nowRFC3339 is the current UTC time in RFC 3339, matching the skill's timestamps.
func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

// ProjectName reads project.name from project.config.yaml, or "" if absent.
func (s *FS) ProjectName() string {
	data, err := os.ReadFile(filepath.Join(s.Root, "project.config.yaml"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "name:") {
			v := strings.TrimSpace(strings.TrimPrefix(t, "name:"))
			return strings.Trim(strings.SplitN(v, "#", 2)[0], " \"'")
		}
	}
	return ""
}

// TaffyCount counts the named taffy workflows in .claude/taffy/*.yaml, excluding
// the schema file.
func (s *FS) TaffyCount() int {
	files, _ := filepath.Glob(filepath.Join(s.Root, ".claude", "taffy", "*.yaml"))
	n := 0
	for _, f := range files {
		if filepath.Base(f) == "schema.yaml" {
			continue
		}
		n++
	}
	return n
}

// PipelineStatus reads status from .claude/state/maple.json, or "" if absent.
func (s *FS) PipelineStatus() string {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "maple.json"))
	if err != nil {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return ""
	}
	if v, ok := m["status"].(string); ok {
		return v
	}
	return ""
}

// Pipeline captures the maple.json fields needed to drive or resume a run.
type Pipeline struct {
	Taffy      string
	Stage      string
	Phase      string // canonical 8-phase name (DISCOVER…FINAL); "" on older runs
	Status     string
	Harness    string
	ResumeAt   string
	AutoResume bool
}

// Pipeline reads the current run's maple.json fields.
func (s *FS) Pipeline() Pipeline {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "maple.json"))
	if err != nil {
		return Pipeline{}
	}
	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return Pipeline{}
	}
	str := func(k string) string { v, _ := m[k].(string); return v }
	auto, _ := m["auto_resume"].(bool)
	return Pipeline{
		Taffy: str("taffy"), Stage: str("stage"), Phase: str("phase"), Status: str("status"),
		Harness: str("harness"), ResumeAt: str("resume_at"), AutoResume: auto,
	}
}

// MergeMapleJSON merges keys into maple.json (read-modify-write), preserving every other
// key the skill owns. A nil value deletes that key.
func (s *FS) MergeMapleJSON(updates map[string]any) error {
	path := filepath.Join(s.Root, ".claude", "state", "maple.json")
	m := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	for k, v := range updates {
		if v == nil {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// ClearPipeline abandons the current run: removes maple.json and any pending gate file.
func (s *FS) ClearPipeline() error {
	_ = os.Remove(filepath.Join(s.Root, ".claude", "state", "approval-pending.txt"))
	return os.Remove(filepath.Join(s.Root, ".claude", "state", "maple.json"))
}

// InFlight reports whether a pipeline status represents active work.
func InFlight(status string) bool {
	switch strings.ToUpper(status) {
	case "RUNNING", "PAUSED", "RATE_LIMITED":
		return true
	}
	return false
}

// pipelineFields are the maple.json keys shown in the pipeline overlay, in order.
var pipelineFields = []string{
	"pipeline", "taffy", "stage", "status", "state",
	"awaiting_approval", "started_at", "updated_at", "ts",
}

// PipelineLines returns the pipeline state from .claude/state/maple.json as aligned
// "key value" lines for the pipeline overlay.
func (s *FS) PipelineLines() []string {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "maple.json"))
	if err != nil {
		return []string{"(no active pipeline)"}
	}
	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return []string{"(unreadable maple.json)"}
	}
	var out []string
	for _, k := range pipelineFields {
		if v, ok := m[k]; ok && v != nil && v != "" {
			out = append(out, fmt.Sprintf("%-18s %v", k, v))
		}
	}
	if len(out) == 0 {
		out = append(out, "(no pipeline fields)")
	}
	return out
}

// ApprovalPending returns the stage awaiting human approval. A gate is pending when
// EITHER .claude/state/approval-pending.txt exists OR maple.json has a non-empty
// awaiting_approval — matching the design portal's view so the TUI and the portal
// always agree on whether a gate is open. Returns "" when nothing is pending.
func (s *FS) ApprovalPending() string {
	if data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "approval-pending.txt")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			return v
		}
	}
	m := s.readMapleJSON()
	if v, ok := m["awaiting_approval"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// Review describes what a pending approval gate is actually asking a human to review: the
// story, the stage, and the exact artifact files. This is the connective tissue the portal
// and TUI need to show "approve the wireframe for <story> → [these files]".
type Review struct {
	Story     string   `json:"story"`
	Title     string   `json:"title"`
	Stage     string   `json:"stage"`
	Artifacts []string `json:"artifacts"`
	Inferred  bool     `json:"inferred"` // true when derived from filenames, not declared by the skill
}

// Review returns what the pending gate is reviewing. It prefers an explicit `review` object
// the skill wrote to maple.json; failing that it infers the story + files from the pending
// stage by matching design artifacts (named <storyID>.<stage>.*). ok=false when no gate.
func (s *FS) Review() (Review, bool) {
	m := s.readMapleJSON()
	if rv, isObj := m["review"].(map[string]any); isObj {
		r := Review{
			Story: mapStr(rv, "story"), Title: mapStr(rv, "title"), Stage: mapStr(rv, "stage"),
		}
		if arr, ok := rv["artifacts"].([]any); ok {
			for _, a := range arr {
				if p, ok := a.(string); ok {
					r.Artifacts = append(r.Artifacts, p)
				}
			}
		}
		if r.Stage == "" {
			r.Stage = s.ApprovalPending()
		}
		if r.Title == "" && r.Story != "" {
			r.Title = s.storyTitle(r.Story)
		}
		return r, true
	}
	stage := s.ApprovalPending()
	if stage == "" {
		return Review{}, false
	}
	return s.inferReview(stage), true
}

func mapStr(m map[string]any, k string) string { v, _ := m[k].(string); return v }

func (s *FS) storyTitle(id string) string {
	for _, st := range s.Stories() {
		if st.ID == id && st.Title != "" {
			return st.Title
		}
	}
	return id
}

// inferReview derives the story + artifact files under review from the pending stage, using
// the design-artifact naming convention (<storyID>.<stage>.*). It picks the story whose
// matching artifact was modified most recently — the one the agent just produced.
//
// Design stages come in two shapes:
//   - story-scoped (wireframe, mockup): files named <storyID>.<stage>.* → one story's artifacts.
//   - project-wide (visual-identity, design-tokens): shared files (palette/typography/tokens.json,
//     design system) that aren't story-named → the whole directory is under review.
func (s *FS) inferReview(stage string) Review {
	dirs, storyScoped := stageDirs(strings.ToLower(stage))
	type cand struct {
		path string
		mod  time.Time
	}
	var cands []cand
	for _, d := range dirs {
		matches, _ := filepath.Glob(filepath.Join(s.Root, "docs", "design", d, "*"))
		for _, p := range matches {
			if strings.HasPrefix(filepath.Base(p), ".") { // skip .gitkeep and other dotfiles
				continue
			}
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				cands = append(cands, cand{p, fi.ModTime()})
			}
		}
	}
	if len(cands) == 0 {
		return Review{Stage: stage, Story: s.currentStory(), Inferred: true}
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].mod.After(cands[j].mod) })

	rel := func(p string) string {
		if r, err := filepath.Rel(s.Root, p); err == nil {
			return filepath.ToSlash(r)
		}
		return p
	}
	if storyScoped {
		storyID := strings.SplitN(filepath.Base(cands[0].path), ".", 2)[0]
		var arts []string
		for _, c := range cands {
			if strings.SplitN(filepath.Base(c.path), ".", 2)[0] == storyID {
				arts = append(arts, rel(c.path))
			}
		}
		sort.Strings(arts)
		return Review{Story: storyID, Title: s.storyTitle(storyID), Stage: stage, Artifacts: arts, Inferred: true}
	}
	// project-wide: all files in the stage's dir are under review; the story (if any) comes
	// from maple.json since these files aren't story-named.
	var arts []string
	for _, c := range cands {
		arts = append(arts, rel(c.path))
	}
	sort.Strings(arts)
	story := s.currentStory()
	return Review{Story: story, Title: s.storyTitle(story), Stage: stage, Artifacts: arts, Inferred: true}
}

// stageDirs maps a design stage to the docs/design subdirectories that hold its output, and
// whether those files are named per-story (true) or shared project-wide (false).
func stageDirs(stage string) (dirs []string, storyScoped bool) {
	switch {
	case strings.Contains(stage, "wireframe"):
		return []string{"wireframes"}, true
	case strings.Contains(stage, "mockup"):
		return []string{"mockups"}, true
	case strings.Contains(stage, "identity") || strings.Contains(stage, "visual"):
		return []string{"identity"}, false
	case strings.Contains(stage, "token"):
		return []string{"identity", "system"}, false
	}
	return []string{"wireframes", "mockups", "identity", "system"}, true
}

// currentStory returns the story the skill declared it is working on, if any, from a
// top-level `story` field or a `review.story` in maple.json. Empty when not declared.
func (s *FS) currentStory() string {
	m := s.readMapleJSON()
	if v := mapStr(m, "story"); v != "" {
		return v
	}
	if rv, ok := m["review"].(map[string]any); ok {
		return mapStr(rv, "story")
	}
	return ""
}

// ApproveGate clears the pending human gate so both surfaces agree and the harness
// continues: it deletes approval-pending.txt AND clears maple.json's awaiting_approval,
// flipping a PAUSED status back to RUNNING (mirrors the portal's approve). Merges
// maple.json so skill-owned keys are preserved.
func (s *FS) ApproveGate() error {
	err := os.Remove(filepath.Join(s.Root, ".claude", "state", "approval-pending.txt"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// Approving resolves any prior reject/request-changes, so drop the revision feedback —
	// this is what lets the gate-clear auto-nudge fire "continue" again on the next approval.
	_ = os.Remove(filepath.Join(s.Root, ".claude", "state", "design-feedback.json"))
	return s.markResumed()
}

// PendingRevision reports whether the last review action was a reject / request-changes
// (recorded in design-feedback.json) — i.e. the harness was told to revise, not proceed.
// The gate-clear auto-nudge checks this so it never sends "continue" over a revise request.
func (s *FS) PendingRevision() bool {
	b, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "design-feedback.json"))
	if err != nil {
		return false
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return false
	}
	a, _ := m["action"].(string)
	return a == "rejected" || a == "requested_changes"
}

// RejectGate records a rejection for the pending stage (approval-rejected.txt) and
// clears the pending gate the same way ApproveGate does.
func (s *FS) RejectGate() error {
	dir := filepath.Join(s.Root, ".claude", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	stage := s.ApprovalPending()
	if err := os.WriteFile(filepath.Join(dir, "approval-rejected.txt"), []byte(stage+"\n"), 0644); err != nil {
		return err
	}
	return s.ApproveGate()
}

// readMapleJSON returns maple.json as a map (empty on any error).
func (s *FS) readMapleJSON() map[string]any {
	m := map[string]any{}
	if data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "maple.json")); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	return m
}

// markResumed clears awaiting_approval and flips PAUSED→RUNNING in maple.json,
// preserving every other (skill-owned) key. No-op when there's nothing to clear.
func (s *FS) markResumed() error {
	path := filepath.Join(s.Root, ".claude", "state", "maple.json")
	m := s.readMapleJSON()
	if len(m) == 0 {
		return nil // no maple.json — nothing to reconcile
	}
	changed := false
	if v, ok := m["awaiting_approval"].(string); ok && strings.TrimSpace(v) != "" {
		m["awaiting_approval"] = nil
		changed = true
	}
	if v, ok := m["status"].(string); ok && strings.EqualFold(v, "PAUSED") {
		m["status"] = "RUNNING"
		changed = true
	}
	if !changed {
		return nil
	}
	m["updated_at"] = nowRFC3339()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// PortalURL returns the design-review portal URL. The portal script writes it to
// .claude/state/design-portal.url; returns "" when no portal is running.
func (s *FS) PortalURL() string {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "design-portal.url"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Agents lists the agent names (.claude/agents/*.md basenames), sorted.
func (s *FS) Agents() []string {
	files, _ := filepath.Glob(filepath.Join(s.Root, ".claude", "agents", "*.md"))
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, strings.TrimSuffix(filepath.Base(f), ".md"))
	}
	sort.Strings(out)
	return out
}

// Skills lists the skill directories under .claude/skills, sorted.
func (s *FS) Skills() []string {
	entries, err := os.ReadDir(filepath.Join(s.Root, ".claude", "skills"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}
