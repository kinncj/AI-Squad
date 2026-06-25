# RATE_LIMITED pipeline state — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `RATE_LIMITED` a real, recoverable pipeline state so a rate-limited TAFFY run is flagged yellow, persists across days, and resumes (manual or opt-in auto) when the window clears.

**Architecture:** The agent cooperatively writes `RATE_LIMITED` + `resume_at` to `.claude/state/maple.json` (it knows the real reset time). A TUI safety net promotes a stale `RUNNING` run to `RATE_LIMITED` from a breadcrumb file or a manual key. The TUI owns render and resume; resume reuses the existing `buildLaunchCmd` + `trySpawnCmdForHarness` spawn path and never calls `tea.Quit`.

**Tech Stack:** Go (Bubble Tea TUI, lipgloss), markdown templates. Tests: Go `testing`, `package main`, `t.TempDir()`.

Design spec: `docs/superpowers/specs/2026-06-25-rate-limited-state-design.md`.

## Global Constraints

- Bump is **minor** on the `v4.x` stream (new TUI state, overlay behavior, keybindings).
- `maple.json` is **merge-not-overwrite**: read existing, change only the keys you own, re-write. Never clobber unowned fields.
- Harness launching **never calls `tea.Quit`** — resume uses `trySpawnCmdForHarness`; on spawn failure the existing `spawnFailedMsg` → manual-launch modal handles it.
- Overlay key handlers stay grouped inside the `if m.showPipeline` block, before the global key switch, same order as `View()`.
- Style: no comments except where the *why* is non-obvious; no docstrings; no backwards-compat shims.
- Commit messages: imperative, lowercase, under 72 chars, **no** `Co-Authored-By`, **no** banned words (enhance, leverage, ensure, implement, utilize, facilitate, improve maintainability). Stage specific files; never `git add -A`.
- Build gate before every commit that touches `tui/`: the package must compile.
- Status string compares are case-insensitive via `strings.ToUpper` (existing convention in `pipeline_state.go`).

## Test harness setup (run once at the start of the session)

`go test` and `go build` fail while `tui/template` is a symlink (`embed.go` can't embed a symlink). These two shell functions materialize a real copy, run the command, and **always restore the symlink** so `git add` never sees the copied tree. Paste them once into the working shell:

```bash
gotest() {
  cd /Users/kinncj/Development/kinncj/MAPLE || return 1
  rm -f tui/template && cp -rL template tui/template
  ( cd tui && go test "$@" ); local rc=$?
  rm -rf tui/template && ln -s ../template tui/template
  return $rc
}
gobuild() {
  cd /Users/kinncj/Development/kinncj/MAPLE || return 1
  rm -f tui/template && cp -rL template tui/template
  ( cd tui && go build -o /tmp/maple_test . ); local rc=$?
  rm -rf tui/template && ln -s ../template tui/template
  return $rc
}
```

All `Run:` commands below assume these are defined. Template **doc** edits (`template/.claude/...`, `template/CLAUDE.md`, `template/OPENCODE.md`, ...) always target the real `template/` source, never `tui/template`.

Tests that touch `.claude/state/` use relative paths, so they `chdir` into a temp dir and restore afterward (version-agnostic, not parallel):

```go
func withTempCwd(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}
```

Put `withTempCwd` in the first new test file (`rate_limit_test.go`) created in Task 2; later test files reuse it (same package).

---

### Task 1: State fields, status icon, and pure predicates

**Files:**
- Modify: `tui/pipeline_state.go`
- Test: `tui/pipeline_state_test.go` (create)

**Interfaces:**
- Produces: `pipelineState` fields `ResumeAt`, `RateLimitReason`, `Harness`, `AutoResume`; methods `isRateLimited() bool`, `windowCleared(now time.Time) bool`; `statusIcon()` returns `"⚑"` for `RATE_LIMITED`.

- [ ] **Step 1: Write the failing test**

Create `tui/pipeline_state_test.go`:

```go
package main

import (
	"testing"
	"time"
)

func TestStatusIcon_RateLimited(t *testing.T) {
	if got := (pipelineState{Status: "RATE_LIMITED"}).statusIcon(); got != "⚑" {
		t.Errorf("statusIcon = %q, want ⚑", got)
	}
	if got := (pipelineState{Status: "rate_limited"}).statusIcon(); got != "⚑" {
		t.Errorf("lowercase statusIcon = %q, want ⚑", got)
	}
}

func TestIsRateLimited(t *testing.T) {
	if !(pipelineState{Status: "RATE_LIMITED"}).isRateLimited() {
		t.Error("RATE_LIMITED should be rate-limited")
	}
	if (pipelineState{Status: "RUNNING"}).isRateLimited() {
		t.Error("RUNNING should not be rate-limited")
	}
}

func TestWindowCleared(t *testing.T) {
	now := time.Date(2026, 6, 25, 15, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute).Format(time.RFC3339)
	future := now.Add(time.Minute).Format(time.RFC3339)

	if !(pipelineState{ResumeAt: past}).windowCleared(now) {
		t.Error("past resume_at should be cleared")
	}
	if (pipelineState{ResumeAt: future}).windowCleared(now) {
		t.Error("future resume_at should not be cleared")
	}
	if (pipelineState{ResumeAt: ""}).windowCleared(now) {
		t.Error("empty resume_at should not be cleared")
	}
	if (pipelineState{ResumeAt: "not-a-time"}).windowCleared(now) {
		t.Error("bad resume_at should not be cleared")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotest ./... -run 'TestStatusIcon_RateLimited|TestIsRateLimited|TestWindowCleared' -v`
Expected: build fails / FAIL — `windowCleared`, `isRateLimited` undefined; `statusIcon` returns `·` for `RATE_LIMITED`.

- [ ] **Step 3: Add fields, icon case, and predicates**

In `tui/pipeline_state.go`, replace the struct with the extended one (keep gofmt alignment):

```go
type pipelineState struct {
	Taffy            string `json:"taffy"`
	Stage            string `json:"stage"`
	Status           string `json:"status"` // RUNNING | PAUSED | RATE_LIMITED | DONE | FAILED
	AwaitingApproval string `json:"awaiting_approval"`
	StartedAt        string `json:"started_at"`
	UpdatedAt        string `json:"updated_at"`
	ResumeAt         string `json:"resume_at"`
	RateLimitReason  string `json:"rate_limit_reason"`
	Harness          string `json:"harness"`
	AutoResume       bool   `json:"auto_resume"`
	// recovery marker fields written by the TUI itself
	State string `json:"state"`
	TS    string `json:"ts"`
}
```

Add the icon case inside `statusIcon()` (after the `PAUSED` case):

```go
	case "RATE_LIMITED":
		return "⚑"
```

Add the two methods after `isStale()`:

```go
func (p pipelineState) isRateLimited() bool {
	return strings.ToUpper(p.Status) == "RATE_LIMITED"
}

// windowCleared reports whether the rate-limit window has passed. Returns false
// when resume_at is empty or unparseable — the run stays paused for manual resume.
func (p pipelineState) windowCleared(now time.Time) bool {
	if p.ResumeAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, p.ResumeAt)
	if err != nil {
		return false
	}
	return !now.Before(t)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotest ./... -run 'TestStatusIcon_RateLimited|TestIsRateLimited|TestWindowCleared' -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add tui/pipeline_state.go tui/pipeline_state_test.go
git commit -m "add RATE_LIMITED state fields, icon, and predicates"
```

---

### Task 2: Merge-write helper, breadcrumb parse, and stale reclassify

**Files:**
- Modify: `tui/pipeline_state.go`
- Test: `tui/rate_limit_test.go` (create — also holds `withTempCwd`)

**Interfaces:**
- Consumes: `pipelineState.isStale()`, `loadPipelineState()`.
- Produces: `mergeMapleJSON(updates map[string]interface{}) error`; `parseRateLimitBreadcrumb(content string) (resumeAt, reason string)`; `reclassifyRateLimit(ps pipelineState) (pipelineState, bool)`; `const rateLimitBreadcrumbPath = ".claude/state/rate-limit.txt"`.

- [ ] **Step 1: Write the failing test**

Create `tui/rate_limit_test.go`:

```go
package main

import (
	"os"
	"testing"
	"time"
)

func withTempCwd(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestParseRateLimitBreadcrumb(t *testing.T) {
	ra, rs := parseRateLimitBreadcrumb("2026-06-25T15:00:00Z|5-hour usage limit")
	if ra != "2026-06-25T15:00:00Z" || rs != "5-hour usage limit" {
		t.Errorf("got %q / %q", ra, rs)
	}
	ra, rs = parseRateLimitBreadcrumb("  2026-06-25T15:00:00Z  ")
	if ra != "2026-06-25T15:00:00Z" || rs != "" {
		t.Errorf("no-pipe got %q / %q", ra, rs)
	}
	ra, rs = parseRateLimitBreadcrumb("")
	if ra != "" || rs != "" {
		t.Errorf("empty got %q / %q", ra, rs)
	}
}

func TestMergeMapleJSON_PreservesUnownedKeys(t *testing.T) {
	withTempCwd(t)
	_ = os.MkdirAll(".claude/state", 0o755)
	_ = os.WriteFile(".claude/state/maple.json", []byte(`{"taffy":"impl","stage":"phase-5","status":"RUNNING"}`), 0o644)

	if err := mergeMapleJSON(map[string]interface{}{"status": "RATE_LIMITED", "resume_at": "T"}); err != nil {
		t.Fatal(err)
	}
	ps, err := loadPipelineState()
	if err != nil {
		t.Fatal(err)
	}
	if ps.Taffy != "impl" || ps.Stage != "phase-5" {
		t.Errorf("unowned keys lost: %+v", ps)
	}
	if ps.Status != "RATE_LIMITED" || ps.ResumeAt != "T" {
		t.Errorf("owned keys wrong: %+v", ps)
	}
}

func TestMergeMapleJSON_NilDeletes(t *testing.T) {
	withTempCwd(t)
	_ = os.MkdirAll(".claude/state", 0o755)
	_ = os.WriteFile(".claude/state/maple.json", []byte(`{"status":"RATE_LIMITED","resume_at":"T"}`), 0o644)
	if err := mergeMapleJSON(map[string]interface{}{"resume_at": nil}); err != nil {
		t.Fatal(err)
	}
	ps, _ := loadPipelineState()
	if ps.ResumeAt != "" {
		t.Errorf("resume_at not deleted: %q", ps.ResumeAt)
	}
}

func TestReclassifyRateLimit_PromotesStaleWithBreadcrumb(t *testing.T) {
	withTempCwd(t)
	_ = os.MkdirAll(".claude/state", 0o755)
	old := time.Now().Add(-20 * time.Minute).UTC().Format(time.RFC3339)
	_ = os.WriteFile(".claude/state/maple.json",
		[]byte(`{"taffy":"impl","stage":"phase-5","status":"RUNNING","updated_at":"`+old+`"}`), 0o644)
	_ = os.WriteFile(rateLimitBreadcrumbPath, []byte("2026-06-25T15:00:00Z|weekly limit"), 0o644)

	ps, _ := loadPipelineState()
	got, ok := reclassifyRateLimit(ps)
	if !ok {
		t.Fatal("expected promotion")
	}
	if got.Status != "RATE_LIMITED" || got.ResumeAt != "2026-06-25T15:00:00Z" || got.RateLimitReason != "weekly limit" {
		t.Errorf("promoted state wrong: %+v", got)
	}
	if _, err := os.Stat(rateLimitBreadcrumbPath); !os.IsNotExist(err) {
		t.Error("breadcrumb should be consumed")
	}
}

func TestReclassifyRateLimit_IgnoresFreshOrNoBreadcrumb(t *testing.T) {
	withTempCwd(t)
	_ = os.MkdirAll(".claude/state", 0o755)
	now := time.Now().UTC().Format(time.RFC3339)
	_ = os.WriteFile(".claude/state/maple.json",
		[]byte(`{"taffy":"impl","stage":"p","status":"RUNNING","updated_at":"`+now+`"}`), 0o644)
	_ = os.WriteFile(rateLimitBreadcrumbPath, []byte("2026-06-25T15:00:00Z|x"), 0o644)

	ps, _ := loadPipelineState()
	if _, ok := reclassifyRateLimit(ps); ok {
		t.Error("fresh RUNNING must not be promoted")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotest ./... -run 'TestParseRateLimitBreadcrumb|TestMergeMapleJSON|TestReclassifyRateLimit' -v`
Expected: build fails — `parseRateLimitBreadcrumb`, `mergeMapleJSON`, `reclassifyRateLimit`, `rateLimitBreadcrumbPath` undefined.

- [ ] **Step 3: Add the helpers**

In `tui/pipeline_state.go`, add near the top (after the `stalePipelineThreshold` const):

```go
const rateLimitBreadcrumbPath = ".claude/state/rate-limit.txt"
```

Add at the end of the file:

```go
// mergeMapleJSON applies updates to .claude/state/maple.json, preserving every
// other key. A value of nil deletes its key. Follows the merge-not-overwrite protocol.
func mergeMapleJSON(updates map[string]interface{}) error {
	_ = os.MkdirAll(".claude/state", 0o755)
	merged := map[string]interface{}{}
	if raw, err := os.ReadFile(".claude/state/maple.json"); err == nil {
		_ = json.Unmarshal(raw, &merged)
	}
	for k, v := range updates {
		if v == nil {
			delete(merged, k)
			continue
		}
		merged[k] = v
	}
	data, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	return os.WriteFile(".claude/state/maple.json", append(data, '\n'), 0o644)
}

// parseRateLimitBreadcrumb splits a "resume_at|reason" line. A line with no pipe
// is treated as resume_at only. Both sides are trimmed.
func parseRateLimitBreadcrumb(content string) (resumeAt, reason string) {
	line := strings.TrimSpace(content)
	if line == "" {
		return "", ""
	}
	if i := strings.Index(line, "|"); i >= 0 {
		return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
	}
	return line, ""
}

// reclassifyRateLimit promotes a stale RUNNING pipeline to RATE_LIMITED when the
// agent left a breadcrumb it could not turn into a full state write. It consumes
// (deletes) the breadcrumb and returns the reloaded state plus whether it promoted.
func reclassifyRateLimit(ps pipelineState) (pipelineState, bool) {
	if strings.ToUpper(ps.Status) != "RUNNING" || !ps.isStale() {
		return ps, false
	}
	raw, err := os.ReadFile(rateLimitBreadcrumbPath)
	if err != nil {
		return ps, false
	}
	resumeAt, reason := parseRateLimitBreadcrumb(string(raw))
	if resumeAt == "" {
		resumeAt = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	}
	if reason == "" {
		reason = "rate limit (reclassified from stale run)"
	}
	_ = mergeMapleJSON(map[string]interface{}{
		"status":            "RATE_LIMITED",
		"resume_at":         resumeAt,
		"rate_limit_reason": reason,
	})
	_ = os.Remove(rateLimitBreadcrumbPath)
	if ps2, err := loadPipelineState(); err == nil {
		return ps2, true
	}
	return ps, true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotest ./... -run 'TestParseRateLimitBreadcrumb|TestMergeMapleJSON|TestReclassifyRateLimit' -v`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add tui/pipeline_state.go tui/rate_limit_test.go
git commit -m "add maple.json merge helper and stale rate-limit reclassify"
```

---

### Task 3: Wire reclassify into reload()

**Files:**
- Modify: `tui/dashboard.go` (function `reload`, around line 327)
- Test: `tui/rate_limit_test.go` (add)

**Interfaces:**
- Consumes: `reclassifyRateLimit`.
- Produces: `reload()` promotes a stale+breadcrumb RUNNING run to RATE_LIMITED before reading approval state.

- [ ] **Step 1: Write the failing test**

Add to `tui/rate_limit_test.go`:

```go
func TestReload_PromotesRateLimited(t *testing.T) {
	withTempCwd(t)
	_ = os.MkdirAll(".claude/state", 0o755)
	old := time.Now().Add(-20 * time.Minute).UTC().Format(time.RFC3339)
	_ = os.WriteFile(".claude/state/maple.json",
		[]byte(`{"taffy":"impl","stage":"p","status":"RUNNING","updated_at":"`+old+`"}`), 0o644)
	_ = os.WriteFile(rateLimitBreadcrumbPath, []byte("2026-06-25T15:00:00Z|limit"), 0o644)

	m := &dashboardModel{}
	m.reload()
	if m.pipelineState.Status != "RATE_LIMITED" {
		t.Errorf("reload did not promote: %q", m.pipelineState.Status)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotest ./... -run TestReload_PromotesRateLimited -v`
Expected: FAIL — status still `RUNNING`.

- [ ] **Step 3: Add the reclassify call to reload()**

In `tui/dashboard.go`, in `reload()`, change the pipeline-load block to:

```go
	// always keep pipeline state current — the skill updates maple.json every stage
	if ps, err := loadPipelineState(); err == nil {
		m.pipelineState = ps
	}
	if ps, ok := reclassifyRateLimit(m.pipelineState); ok {
		m.pipelineState = ps
	}
	m.approvalPending = approvalPending()
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotest ./... -run TestReload_PromotesRateLimited -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add tui/dashboard.go tui/rate_limit_test.go
git commit -m "promote stale rate-limited runs on reload"
```

---

### Task 4: In-flight helper for badge count and footer keys

**Files:**
- Modify: `tui/dashboard.go` (badge condition ~line 1614; footer `showPipeline` block ~line 1703)
- Test: `tui/rate_limit_test.go` (add)

**Interfaces:**
- Produces: `isInFlight(status string) bool` (true for RUNNING/PAUSED/RATE_LIMITED).

- [ ] **Step 1: Write the failing test**

Add to `tui/rate_limit_test.go`:

```go
func TestIsInFlight(t *testing.T) {
	for _, s := range []string{"RUNNING", "PAUSED", "RATE_LIMITED", "rate_limited"} {
		if !isInFlight(s) {
			t.Errorf("%q should be in-flight", s)
		}
	}
	for _, s := range []string{"DONE", "FAILED", ""} {
		if isInFlight(s) {
			t.Errorf("%q should not be in-flight", s)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotest ./... -run TestIsInFlight -v`
Expected: build fails — `isInFlight` undefined.

- [ ] **Step 3: Add helper and use it**

In `tui/dashboard.go`, add near `countTaffyWorkflows`:

```go
func isInFlight(status string) bool {
	switch strings.ToUpper(status) {
	case "RUNNING", "PAUSED", "RATE_LIMITED":
		return true
	}
	return false
}
```

Replace the badge condition (was `m.pipelineState.Status == "RUNNING" || m.pipelineState.Status == "PAUSED"`):

```go
	if m.pipelineState.Taffy != "" && isInFlight(m.pipelineState.Status) {
		taffyRunningCount = 1
	}
```

In the footer `case m.showPipeline:` block, add a `RATE_LIMITED` arm to its inner `switch` (before `default`):

```go
		case m.pipelineState.isRateLimited():
			if m.pipelineState.windowCleared(time.Now()) {
				keys = "  [r] resume now · [A] auto-resume toggle · [c] clear · any other key closes"
			} else {
				keys = "  waiting for window · [A] auto-resume toggle · [r] force resume · [c] clear"
			}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotest ./... -run TestIsInFlight -v`
Expected: PASS. Then `gobuild` → expected: builds clean (footer/badge edits compile).

- [ ] **Step 5: Commit**

```bash
git add tui/dashboard.go tui/rate_limit_test.go
git commit -m "count rate-limited runs as in-flight, add footer keys"
```

---

### Task 5: Countdown formatter and pipeline overlay rendering

**Files:**
- Modify: `tui/dashboard_views.go` (function `pipelineView`, the `else` status block ~line 838-885)
- Test: `tui/rate_limit_test.go` (add)

**Interfaces:**
- Consumes: `pipelineState` fields, `t.Warning`.
- Produces: `formatRateLimitCountdown(resumeAt string, now time.Time) string`.

- [ ] **Step 1: Write the failing test**

Add to `tui/rate_limit_test.go`:

```go
func TestFormatRateLimitCountdown(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	got := formatRateLimitCountdown(now.Add(2*time.Hour+14*time.Minute+9*time.Second).Format(time.RFC3339), now)
	if got[:9] != "in 02:14:" {
		t.Errorf("countdown = %q", got)
	}
	if formatRateLimitCountdown(now.Add(-time.Minute).Format(time.RFC3339), now) != "window cleared" {
		t.Error("past should say window cleared")
	}
	if formatRateLimitCountdown("", now) != "unknown" {
		t.Error("empty should say unknown")
	}
	if formatRateLimitCountdown("garbage", now) != "unknown" {
		t.Error("bad should say unknown")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotest ./... -run TestFormatRateLimitCountdown -v`
Expected: build fails — `formatRateLimitCountdown` undefined.

- [ ] **Step 3: Add the formatter and the overlay block**

In `tui/dashboard_views.go`, add the formatter (near the top, after imports/other helpers):

```go
// formatRateLimitCountdown renders time until resume_at as "in HH:MM:SS (3:04 PM)".
// Returns "window cleared" when now is past resume_at, "unknown" when unset/bad.
func formatRateLimitCountdown(resumeAt string, now time.Time) string {
	if resumeAt == "" {
		return "unknown"
	}
	tm, err := time.Parse(time.RFC3339, resumeAt)
	if err != nil {
		return "unknown"
	}
	if !now.Before(tm) {
		return "window cleared"
	}
	d := tm.Sub(now).Round(time.Second)
	return fmt.Sprintf("in %02d:%02d:%02d (%s)",
		int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60, tm.Local().Format("3:04 PM"))
}
```

In `pipelineView`, add a `RATE_LIMITED` color arm to the existing `switch ps.Status` (alongside `PAUSED`/`FAILED`):

```go
		case "RATE_LIMITED":
			iconStyle = lipgloss.NewStyle().Foreground(t.Warning)
```

Then, after the `if stale { ... }` block and before the `if ps.AwaitingApproval != "" ...` block, add:

```go
		if ps.isRateLimited() {
			now := time.Now()
			cleared := ps.windowCleared(now)
			warn := lipgloss.NewStyle().Foreground(t.Warning)
			bodyLines = append(bodyLines, "", warn.Render("  ⚑ Rate-limited: "+ps.RateLimitReason))
			bodyLines = append(bodyLines, warn.Render("  Resets "+formatRateLimitCountdown(ps.ResumeAt, now)))
			auto := "OFF"
			if ps.AutoResume {
				auto = "ON"
			}
			if cleared {
				bodyLines = append(bodyLines,
					lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("  window cleared — [r] resume now"))
			} else {
				bodyLines = append(bodyLines,
					lipgloss.NewStyle().Foreground(t.Muted).Render("  [r] force resume before the window clears"))
			}
			bodyLines = append(bodyLines,
				lipgloss.NewStyle().Foreground(t.Primary).Render("  [A] auto-resume: "+auto+"   [c] clear"))
		}
```

Add `"time"` to the `dashboard_views.go` import block (it currently imports `fmt`, `os`, `path/filepath`, `strings`, `lipgloss` — no `time`):

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)
```

(`fmt` is already imported and used.)

- [ ] **Step 4: Run test to verify it passes**

Run: `gotest ./... -run TestFormatRateLimitCountdown -v`
Expected: PASS. Then `gobuild` → builds clean.

- [ ] **Step 5: Commit**

```bash
git add tui/dashboard_views.go tui/rate_limit_test.go
git commit -m "render rate-limited overlay with countdown and resume keys"
```

---

### Task 6: Resume command builder

**Files:**
- Modify: `tui/dashboard.go`
- Test: `tui/rate_limit_test.go` (add)

**Interfaces:**
- Consumes: `buildLaunchCmd`.
- Produces: `buildResumePrompt(taffy, stage string) string`; `buildResumeCmd(ps pipelineState, pinned map[string]string) (harness string, cmd []string)`.

- [ ] **Step 1: Write the failing test**

Add to `tui/rate_limit_test.go`:

```go
func TestBuildResumeCmd_UsesRecordedHarness(t *testing.T) {
	ps := pipelineState{Taffy: "implement-stories", Stage: "phase-5", Harness: "claude"}
	pinned := map[string]string{"claude": "sess-1"}
	h, cmd := buildResumeCmd(ps, pinned)
	if h != "claude" {
		t.Errorf("harness = %q", h)
	}
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, "--resume") || !strings.Contains(joined, "sess-1") {
		t.Errorf("expected resume of pinned session: %q", joined)
	}
	if !strings.Contains(joined, "<maple-resume>") || !strings.Contains(joined, "phase-5") {
		t.Errorf("expected resume prompt with stage: %q", joined)
	}
}

func TestBuildResumeCmd_FallsBackToSinglePinned(t *testing.T) {
	ps := pipelineState{Taffy: "impl", Stage: "p"} // no Harness recorded
	h, cmd := buildResumeCmd(ps, map[string]string{"opencode": "s2"})
	if h != "opencode" || cmd == nil {
		t.Errorf("expected fallback to single pinned harness, got %q / %v", h, cmd)
	}
}

func TestBuildResumeCmd_NoHarnessNoSingleton(t *testing.T) {
	h, cmd := buildResumeCmd(pipelineState{Taffy: "impl"}, map[string]string{"a": "1", "b": "2"})
	if h != "" || cmd != nil {
		t.Errorf("expected no resume target, got %q / %v", h, cmd)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotest ./... -run TestBuildResumeCmd -v`
Expected: build fails — `buildResumeCmd` undefined.

- [ ] **Step 3: Add the builders**

In `tui/dashboard.go`, near `buildLaunchCmd`:

```go
func buildResumePrompt(taffy, stage string) string {
	return "/pipeline-runner " + taffy + `

<maple-resume>
This pipeline was paused on a rate limit and is now being resumed.
Continue taffy ` + taffy + ` from stage "` + stage + `" — do not restart from the beginning.
Re-read .claude/state/maple.json for context, set status back to RUNNING, and carry on.
</maple-resume>`
}

// buildResumeCmd builds the exec command to relaunch a rate-limited pipeline on
// its harness's pinned session with a continuation prompt. Falls back to the only
// pinned harness when none was recorded. Returns "", nil when no target is known.
func buildResumeCmd(ps pipelineState, pinned map[string]string) (string, []string) {
	harness := ps.Harness
	if harness == "" && len(pinned) == 1 {
		for h := range pinned {
			harness = h
		}
	}
	if harness == "" {
		return "", nil
	}
	return harness, buildLaunchCmd(harness, buildResumePrompt(ps.Taffy, ps.Stage), pinned, false)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotest ./... -run TestBuildResumeCmd -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add tui/dashboard.go tui/rate_limit_test.go
git commit -m "add resume command builder for rate-limited pipelines"
```

---

### Task 7: Overlay key handlers [r] / [A] / [m]

**Files:**
- Modify: `tui/dashboard.go` (the `if m.showPipeline { switch k { ... } }` block ~line 928-957)
- Verify: manual (key handlers in the Bubble Tea `Update` loop are exercised end-to-end in Task 10).

**Interfaces:**
- Consumes: `buildResumeCmd`, `mergeMapleJSON`, `loadPipelineState`, `trySpawnCmdForHarness`, `isRateLimited`, `isStale`.

- [ ] **Step 1: Add the three cases**

In `tui/dashboard.go`, inside `if m.showPipeline { switch k {`, add these cases **before** the `default:` arm:

```go
		case "r":
			if m.pipelineState.isRateLimited() {
				harness, cmd := buildResumeCmd(m.pipelineState, m.pinnedSessions)
				if cmd == nil {
					return m, m.setStatus("can't resume: no recorded harness/session — relaunch with [x]", true)
				}
				_ = mergeMapleJSON(map[string]interface{}{
					"status":            "RUNNING",
					"resume_at":         nil,
					"rate_limit_reason": nil,
					"updated_at":        time.Now().UTC().Format(time.RFC3339),
				})
				if ps, err := loadPipelineState(); err == nil {
					m.pipelineState = ps
				}
				m.showPipeline = false
				return m, trySpawnCmdForHarness(harness, cmd)
			}
		case "A":
			if m.pipelineState.isRateLimited() {
				next := !m.pipelineState.AutoResume
				_ = mergeMapleJSON(map[string]interface{}{"auto_resume": next})
				if ps, err := loadPipelineState(); err == nil {
					m.pipelineState = ps
				}
				state := "OFF"
				if next {
					state = "ON"
				}
				return m, m.setStatus("auto-resume: "+state, false)
			}
		case "m":
			if strings.ToUpper(m.pipelineState.Status) == "RUNNING" && m.pipelineState.isStale() {
				_ = mergeMapleJSON(map[string]interface{}{
					"status":            "RATE_LIMITED",
					"resume_at":         time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
					"rate_limit_reason": "manually marked rate-limited",
				})
				if ps, err := loadPipelineState(); err == nil {
					m.pipelineState = ps
				}
				return m, m.setStatus("marked RATE_LIMITED — resumes in ~1h or press [r]", false)
			}
```

Note: `r`/`A`/`m` that do not match the guard fall through to no-op (the `switch` simply does nothing for them and returns `m, nil` below); they do **not** hit `default:` because they are explicit cases. This keeps the overlay open when the key is inapplicable, which is the intended behavior.

- [ ] **Step 2: Build**

Run: `gobuild`
Expected: builds clean.

- [ ] **Step 3: Sanity-check the no-op path reasoning**

Confirm by reading the block: when `k == "r"` and the pipeline is not rate-limited, the `case "r":` body is skipped and control reaches `return m, nil` at the end of the `if m.showPipeline` block — the overlay stays open, nothing spawns. Correct.

- [ ] **Step 4: Commit**

```bash
git add tui/dashboard.go
git commit -m "add [r] resume, [A] auto-resume toggle, [m] mark keys"
```

---

### Task 8: Auto-resume on tick with thrash guard

**Files:**
- Modify: `tui/dashboard.go` (model struct; `dashTickMsg` case ~line 406)
- Test: `tui/rate_limit_test.go` (add)

**Interfaces:**
- Consumes: `windowCleared`, `isRateLimited`, `buildResumeCmd`, `mergeMapleJSON`.
- Produces: model field `lastAutoResume time.Time`; `rateLimitResumeDecision(ps pipelineState, now, lastResume time.Time) (resume, disarm bool)`; method `maybeAutoResume() tea.Cmd`.

- [ ] **Step 1: Write the failing test**

Add to `tui/rate_limit_test.go`:

```go
func TestRateLimitResumeDecision(t *testing.T) {
	now := time.Date(2026, 6, 25, 15, 0, 0, 0, time.UTC)
	cleared := pipelineState{Status: "RATE_LIMITED", AutoResume: true, ResumeAt: now.Add(-time.Second).Format(time.RFC3339)}
	notYet := pipelineState{Status: "RATE_LIMITED", AutoResume: true, ResumeAt: now.Add(time.Hour).Format(time.RFC3339)}

	if r, d := rateLimitResumeDecision(cleared, now, time.Time{}); !r || d {
		t.Errorf("cleared+armed should resume: r=%v d=%v", r, d)
	}
	if r, _ := rateLimitResumeDecision(notYet, now, time.Time{}); r {
		t.Error("not-yet-cleared should not resume")
	}
	off := cleared
	off.AutoResume = false
	if r, _ := rateLimitResumeDecision(off, now, time.Time{}); r {
		t.Error("auto-resume OFF should not resume")
	}
	// thrash guard: resumed 1 min ago, already rate-limited again
	if r, d := rateLimitResumeDecision(cleared, now, now.Add(-time.Minute)); r || !d {
		t.Errorf("recent resume should disarm: r=%v d=%v", r, d)
	}
	// resumed long ago — fine to resume again
	if r, d := rateLimitResumeDecision(cleared, now, now.Add(-time.Hour)); !r || d {
		t.Errorf("old resume should re-resume: r=%v d=%v", r, d)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotest ./... -run TestRateLimitResumeDecision -v`
Expected: build fails — `rateLimitResumeDecision` undefined.

- [ ] **Step 3: Add decision fn, model field, and tick wiring**

In `tui/dashboard.go`, add the model field next to other pipeline fields (e.g. after `portalAutoStage`):

```go
	lastAutoResume time.Time
```

Add the pure decision function near `buildResumeCmd`:

```go
// rateLimitResumeDecision decides, on a tick, whether to auto-resume a RATE_LIMITED
// pipeline. disarm=true means turn auto_resume off (thrash guard) because we resumed
// within the last 5 minutes and are already rate-limited again.
func rateLimitResumeDecision(ps pipelineState, now, lastResume time.Time) (resume, disarm bool) {
	if !ps.isRateLimited() || !ps.AutoResume || !ps.windowCleared(now) {
		return false, false
	}
	if !lastResume.IsZero() && now.Sub(lastResume) < 5*time.Minute {
		return false, true
	}
	return true, false
}
```

Add the model method near `reload`:

```go
func (m *dashboardModel) maybeAutoResume() tea.Cmd {
	now := time.Now()
	resume, disarm := rateLimitResumeDecision(m.pipelineState, now, m.lastAutoResume)
	if disarm {
		_ = mergeMapleJSON(map[string]interface{}{"auto_resume": false})
		if ps, err := loadPipelineState(); err == nil {
			m.pipelineState = ps
		}
		m.status = "auto-resume disarmed — re-hit limit too soon"
		return nil
	}
	if !resume {
		return nil
	}
	harness, cmd := buildResumeCmd(m.pipelineState, m.pinnedSessions)
	if cmd == nil {
		return nil
	}
	m.lastAutoResume = now
	_ = mergeMapleJSON(map[string]interface{}{
		"status":            "RUNNING",
		"resume_at":         nil,
		"rate_limit_reason": nil,
		"updated_at":        now.UTC().Format(time.RFC3339),
	})
	if ps, err := loadPipelineState(); err == nil {
		m.pipelineState = ps
	}
	m.status = "⚑ window cleared — auto-resuming " + harness
	return trySpawnCmdForHarness(harness, cmd)
}
```

In the `case dashTickMsg:` handler, insert the auto-resume check right after `m.reload()` and before the `autoStage` logic:

```go
	case dashTickMsg:
		m.reload()
		if cmd := m.maybeAutoResume(); cmd != nil {
			return m, tea.Batch(dashTickCmd(), cmd)
		}
		autoStage := strings.TrimSpace(m.approvalPending)
		// ... unchanged ...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotest ./... -run TestRateLimitResumeDecision -v`
Expected: PASS. Then `gobuild` → builds clean.

- [ ] **Step 5: Commit**

```bash
git add tui/dashboard.go tui/rate_limit_test.go
git commit -m "auto-resume rate-limited runs on tick with thrash guard"
```

---

### Task 9: Cooperative detection — launch prompt and recorded harness

**Files:**
- Modify: `tui/dashboard.go` (`buildQuickPromptCmd` ~line 2156-2233; `writeQuickLaunchState` ~line 2127; its caller ~line 904)
- Test: `tui/rate_limit_test.go` (add)

**Interfaces:**
- Produces: `buildQuickPromptCmd` output contains a `<maple-rate-limit>` block; `writeQuickLaunchState(skill, stage, harness string)` records `harness`.

- [ ] **Step 1: Write the failing test**

Add to `tui/rate_limit_test.go`:

```go
func TestBuildQuickPromptCmd_HasRateLimitBlock(t *testing.T) {
	out := buildQuickPromptCmd("implement-stories", "do the thing", "taffy", "claude")
	if !strings.Contains(out, "<maple-rate-limit>") {
		t.Error("taffy launch missing rate-limit block")
	}
	if !strings.Contains(out, "rate-limit.txt") || !strings.Contains(out, "RATE_LIMITED") {
		t.Error("rate-limit block missing breadcrumb/status instructions")
	}
	skillOut := buildQuickPromptCmd("tdd-workflow", "", "skill", "claude")
	if !strings.Contains(skillOut, "<maple-rate-limit>") {
		t.Error("skill launch should also carry rate-limit block")
	}
}

func TestWriteQuickLaunchState_RecordsHarness(t *testing.T) {
	withTempCwd(t)
	writeQuickLaunchState("implement-stories", "phase-1", "opencode")
	ps, err := loadPipelineState()
	if err != nil {
		t.Fatal(err)
	}
	if ps.Harness != "opencode" {
		t.Errorf("harness = %q", ps.Harness)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotest ./... -run 'TestBuildQuickPromptCmd_HasRateLimitBlock|TestWriteQuickLaunchState_RecordsHarness' -v`
Expected: build fails — `writeQuickLaunchState` takes 2 args; no rate-limit block.

- [ ] **Step 3: Add the block, record harness, update caller**

In `buildQuickPromptCmd`, after the `tracking := ...` block, add:

```go
	rateLimit := `

<maple-rate-limit>
If a usage limit, rate limit, HTTP 429, or "resets at <time>" message stops you from continuing, BEFORE stopping:
1. Merge-write .claude/state/maple.json (never overwrite other keys), keeping "taffy" and "stage" intact:
   {"status":"RATE_LIMITED","resume_at":"<ISO-8601 reset time from the message, else now+1h>","rate_limit_reason":"<one line>"}
2. Write a one-line breadcrumb to .claude/state/rate-limit.txt containing exactly:
   <resume_at>|<reason>
   This is a single cheap write — do it even if you can do nothing else.
3. Then stop. Do not burn retries hammering the API; MAPLE resumes you when the window clears.
</maple-rate-limit>`
```

Change the return to append `rateLimit` (after `tracking`):

```go
	return cmd + governance + taffyContext + uiOverride + tracking + rateLimit + progress + portalBlock
```

Change `writeQuickLaunchState` signature and body to record harness:

```go
func writeQuickLaunchState(skill, stage, harness string) {
```

and inside, after `merged["status"] = "RUNNING"`:

```go
	merged["harness"] = harness
```

Update the caller at ~line 904:

```go
				writeQuickLaunchState(name, prompt, m.quickLaunchHarness)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotest ./... -run 'TestBuildQuickPromptCmd_HasRateLimitBlock|TestWriteQuickLaunchState_RecordsHarness' -v`
Expected: PASS. Then `gobuild` → builds clean.

- [ ] **Step 5: Full suite + commit**

Run: `gotest ./... -v`
Expected: all rate-limit tests plus the pre-existing suite PASS.

```bash
git add tui/dashboard.go tui/rate_limit_test.go
git commit -m "inject rate-limit handling block and record launch harness"
```

---

### Task 10: Skill, session-start protocol, and README docs

**Files:**
- Modify: `template/.claude/skills/pipeline-runner/SKILL.md`
- Modify: `template/CLAUDE.md`, `template/COPILOT.md` (only these two carry a session-start status bullet)
- Modify: `README.md` (FAFE rows ~184-185)

**Interfaces:** documentation only — verified by `grep`.

- [ ] **Step 1: Add "Rate-limit handling" to the skill**

In `template/.claude/skills/pipeline-runner/SKILL.md`, after the `## Failure Handling` section, add:

```markdown
## Rate-limit Handling

When a usage limit, rate limit, HTTP 429, or "resets at <time>" message stops you continuing, before stopping:

1. Merge-write `.claude/state/maple.json` (keep `taffy`/`stage`):

```json
{ "status": "RATE_LIMITED", "resume_at": "<iso8601 reset time, else now+1h>", "rate_limit_reason": "<one line>", "updated_at": "<iso8601>" }
```

2. Write a one-line breadcrumb to `.claude/state/rate-limit.txt`: `<resume_at>|<reason>`. One cheap write — do it even if nothing else is possible.
3. Stop. Do not hammer the API. MAPLE resumes the pipeline when the window clears (manual `[r]`, or auto when armed).
```

In the same file's `### .claude/state/maple.json` table, add rows:

```markdown
| `resume_at` | skill/TUI | ISO 8601 — when the rate-limit window clears |
| `rate_limit_reason` | skill/TUI | one-line cause |
| `harness` | TUI | which harness ran this pipeline (for resume) |
| `auto_resume` | TUI | `true` when auto-resume is armed |
```

And after the `approval-pending.txt` subsection, add:

```markdown
### `.claude/state/rate-limit.txt`

Skill/agent writes `<resume_at>|<reason>`. TUI reads, promotes `maple.json` to `RATE_LIMITED`, and deletes the file.
```

- [ ] **Step 2: Update the session-start protocol**

Only `template/CLAUDE.md` and `template/COPILOT.md` carry a session-start status bullet (`OPENCODE.md`, `CURSOR.md`, and `AGENTS.md` have none — leave them alone).

In `template/CLAUDE.md`, replace this exact line:

```markdown
- **`RUNNING` or `PAUSED`** — a pipeline is active. Continue within it; do not start a parallel one.
```

with:

```markdown
- **`RUNNING`, `PAUSED`, or `RATE_LIMITED`** — a pipeline is active (RATE_LIMITED = paused on a rate limit). Continue within it; do not start a parallel one. For RATE_LIMITED, resume it — do not begin new work.
```

In `template/COPILOT.md`, replace this exact line:

```markdown
- **`RUNNING` or `PAUSED`** — pipeline is active. Continue within it.
```

with:

```markdown
- **`RUNNING`, `PAUSED`, or `RATE_LIMITED`** — pipeline is active (RATE_LIMITED = paused on a rate limit). Continue within it; resume a RATE_LIMITED run rather than starting new work.
```

- [ ] **Step 3: Rewrite the README FAFE rows**

In `README.md`, replace lines 184-185 with copy that matches what ships:

```markdown
| **F — Fault-Tolerant** | Hard timeouts kill stuck agents and mark the job `FAILED`. On a rate limit, the agent writes `RATE_LIMITED` with a `resume_at`; MAPLE flags it yellow and resumes the exact stage when the window clears — manually with `[r]`, or automatically when auto-resume is armed. Three consecutive failures escalate to human. |
| **F — File-Synced** | No Redis, no broker. TAFFY writes state to `.claude/state/maple.json`. The TUI reacts: `RUNNING` → spinner, `PAUSED` → gate indicator, `RATE_LIMITED` → yellow flag with a reset countdown, `DONE`/`FAILED` → final status. State persists on disk, so a run rate-limited today resumes tomorrow. |
```

- [ ] **Step 4: Verify the doc edits**

Run:
```bash
cd /Users/kinncj/Development/kinncj/MAPLE
grep -c "Rate-limit Handling" template/.claude/skills/pipeline-runner/SKILL.md
grep -c "RATE_LIMITED" template/CLAUDE.md README.md
```
Expected: `1` for the skill heading; non-zero counts for `RATE_LIMITED` in `template/CLAUDE.md` and `README.md`.

- [ ] **Step 5: Commit**

```bash
git add template/.claude/skills/pipeline-runner/SKILL.md template/CLAUDE.md template/COPILOT.md README.md
git commit -m "document rate-limit handling, resume protocol, and yellow flag"
```

---

### Task 11: Full build dance and end-to-end verification

**Files:** none (verification only). Scratch project: `/tmp/maple_test_rate_limit`.

- [ ] **Step 1: Real build via the Makefile dance**

Run:
```bash
cd /Users/kinncj/Development/kinncj/MAPLE && make build-tui && ls -la maple && ls -ld tui/template
```
Expected: `maple` binary built; `tui/template` is a symlink again (dance restored it).

- [ ] **Step 2: gofmt + full test suite**

Run:
```bash
cd /Users/kinncj/Development/kinncj/MAPLE && make lint && gotest ./... -v
```
Expected: `gofmt: clean`; all tests PASS.

- [ ] **Step 3: Stand up the scratch project**

Run:
```bash
rm -rf /tmp/maple_test_rate_limit && mkdir -p /tmp/maple_test_rate_limit
cd /tmp/maple_test_rate_limit && git init -q
/Users/kinncj/Development/kinncj/MAPLE/maple init || true
mkdir -p .claude/state
```

- [ ] **Step 4: Fake a not-yet-cleared RATE_LIMITED state and eyeball it**

Write a state file with a future `resume_at` and a recorded harness:
```bash
cat > /tmp/maple_test_rate_limit/.claude/state/maple.json <<'JSON'
{"taffy":"implement-stories","stage":"phase-5-implement","status":"RATE_LIMITED","resume_at":"2026-12-31T23:59:00Z","rate_limit_reason":"5-hour usage limit","harness":"claude","auto_resume":false}
JSON
cd /tmp/maple_test_rate_limit && /Users/kinncj/Development/kinncj/MAPLE/maple
```
In the TUI press `P`. Confirm: `⚑` yellow status, `Rate-limited: 5-hour usage limit`, a `Resets in …` countdown, and the `[r] force resume / [A] auto-resume: OFF / [c] clear` lines. Press `A` → footer/line flips to `ON`. Press `c` → state clears. Quit with `q`.

- [ ] **Step 5: Fake a cleared window + breadcrumb reclassify**

```bash
cat > /tmp/maple_test_rate_limit/.claude/state/maple.json <<'JSON'
{"taffy":"implement-stories","stage":"phase-5","status":"RATE_LIMITED","resume_at":"2020-01-01T00:00:00Z","rate_limit_reason":"weekly limit","harness":"claude","auto_resume":false}
JSON
```
Launch `maple`, press `P`: confirm it shows `window cleared — [r] resume now`. (With no real pinned claude session, `[r]` will report the manual-relaunch status — that is the expected no-session path.)

Then test reclassify: write a stale RUNNING state plus a breadcrumb, no manual status edit:
```bash
OLD=$(python3 -c "import datetime;print((datetime.datetime.utcnow()-datetime.timedelta(minutes=20)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
cat > /tmp/maple_test_rate_limit/.claude/state/maple.json <<JSON
{"taffy":"implement-stories","stage":"phase-5","status":"RUNNING","updated_at":"$OLD"}
JSON
printf '2026-12-31T23:59:00Z|reclassified from stale' > /tmp/maple_test_rate_limit/.claude/state/rate-limit.txt
```
Launch `maple` (wait one 5s tick), press `P`: confirm status is now `⚑ RATE_LIMITED` with reason `reclassified from stale`, and that `.claude/state/rate-limit.txt` is gone (`ls .claude/state/`).

- [ ] **Step 6: Clean up the scratch project**

```bash
rm -rf /tmp/maple_test_rate_limit
```

- [ ] **Step 7: Final confirmation**

Confirm the branch is clean apart from the intended changes:
```bash
cd /Users/kinncj/Development/kinncj/MAPLE && git status --short && git log --oneline -12
```
Expected: clean working tree; `tui/template` still a symlink (`ls -ld tui/template`); the task commits present.

---

## Notes for the implementer

- After Task 11, the feature is ready for a PR against `main`. Per repo convention, open a GitHub issue (`enhancement`, milestone `v4.10.0`) and reference it. The release (minor bump + `gh release create`) is a separate, human-triggered step — do **not** tag.
- Detector B (PTY-wrapping launch wrapper) is intentionally out of scope; the TUI owns persist/render/resume so it can be added later without reworking this.
- Soft/transient 429s the harness already auto-retries are out of scope — only run-stopping limits are handled here.
