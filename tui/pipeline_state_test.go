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
