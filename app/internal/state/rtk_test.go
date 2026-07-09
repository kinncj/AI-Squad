package state

import "testing"

func TestRTKHarnessesRoundTrip(t *testing.T) {
	fs := NewFS(t.TempDir())
	if len(fs.RTKHarnesses()) != 0 {
		t.Error("fresh project should have no rtk harnesses")
	}
	if err := fs.SetRTKHarness("claude", true); err != nil {
		t.Fatalf("SetRTKHarness: %v", err)
	}
	if err := fs.SetRTKHarness("opencode", false); err != nil {
		t.Fatal(err)
	}
	got := fs.RTKHarnesses()
	if !got["claude"] {
		t.Error("claude should be wired")
	}
	if got["opencode"] {
		t.Error("opencode should be off")
	}
	// Toggling off persists.
	if err := fs.SetRTKHarness("claude", false); err != nil {
		t.Fatal(err)
	}
	if fs.RTKHarnesses()["claude"] {
		t.Error("claude should now be off")
	}
}
