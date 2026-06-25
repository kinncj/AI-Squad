package main

import (
	"os"
	"strings"
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

func TestPipelineStatusView_RateLimitedRender(t *testing.T) {
	m := &dashboardModel{theme: loadTheme(), width: 100, height: 40}
	m.pipelineState = pipelineState{
		Taffy:           "implement-stories",
		Stage:           "phase-5-implement",
		Status:          "RATE_LIMITED",
		ResumeAt:        time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		RateLimitReason: "5-hour usage limit",
		Harness:         "claude",
		AutoResume:      true,
	}
	out := m.pipelineStatusView()
	for _, want := range []string{"⚑", "RATE_LIMITED", "Rate-limited", "5-hour usage limit", "Resets", "[r]", "[A]", "auto-resume: ON"} {
		if !strings.Contains(out, want) {
			t.Errorf("pipelineStatusView missing %q", want)
		}
	}
}

func TestPipelineStatusView_RateLimitClearedRender(t *testing.T) {
	m := &dashboardModel{theme: loadTheme(), width: 100, height: 40}
	m.pipelineState = pipelineState{
		Taffy:    "implement-stories",
		Stage:    "phase-5",
		Status:   "RATE_LIMITED",
		ResumeAt: time.Now().Add(-time.Minute).Format(time.RFC3339),
	}
	out := m.pipelineStatusView()
	if !strings.Contains(out, "window cleared") {
		t.Error("cleared window should render 'window cleared'")
	}
}

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
