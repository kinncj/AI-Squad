package main

import (
	"strings"
	"testing"
	"time"
)

func TestRunWithTimeout_Success(t *testing.T) {
	out, err := runWithTimeout(timeoutLocal, nil, "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(string(out)) != "hello" {
		t.Errorf("got %q, want hello", out)
	}
}

func TestRunWithTimeout_TimesOut(t *testing.T) {
	start := time.Now()
	_, err := runWithTimeout(100*time.Millisecond, nil, "sleep", "5")
	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error should mention timeout, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("timeout did not kill child promptly: took %s", elapsed)
	}
}

func TestRunWithTimeout_NonZeroExit(t *testing.T) {
	_, err := runWithTimeout(timeoutLocal, nil, "sh", "-c", "exit 3")
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
	if strings.Contains(err.Error(), "timed out") {
		t.Errorf("non-zero exit misreported as timeout: %v", err)
	}
}

func TestRunWithTimeoutStdin_FeedsStdin(t *testing.T) {
	out, err := runWithTimeoutStdin(timeoutLocal, nil, "from-stdin\n", "cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(string(out)) != "from-stdin" {
		t.Errorf("got %q, want from-stdin", out)
	}
}
