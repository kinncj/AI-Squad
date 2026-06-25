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
