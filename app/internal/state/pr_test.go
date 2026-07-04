package state

import "testing"

func TestParsePRs(t *testing.T) {
	data := []byte(`[{"number":22,"title":"Cross-harness tracking","state":"OPEN"},{"number":19,"title":"Fix flake","state":"MERGED"}]`)
	got := parsePRs(data)
	if len(got) != 2 {
		t.Fatalf("parsed %d PRs, want 2", len(got))
	}
	if got[0].Number != 22 || got[0].Title != "Cross-harness tracking" || got[0].State != "OPEN" {
		t.Errorf("PR 0 = %+v", got[0])
	}
	if got[1].Number != 19 || got[1].State != "MERGED" {
		t.Errorf("PR 1 = %+v", got[1])
	}
}

func TestParsePRsEmptyAndInvalid(t *testing.T) {
	if got := parsePRs([]byte(`[]`)); len(got) != 0 {
		t.Errorf("empty list should parse to 0 PRs, got %d", len(got))
	}
	if got := parsePRs([]byte(`not json`)); got != nil {
		t.Errorf("invalid json should yield nil, got %+v", got)
	}
}
