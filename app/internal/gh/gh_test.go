package gh

import "testing"

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name, json, key, want string
	}{
		{"string value", `{"id":"PVT_abc","number":7}`, "id", "PVT_abc"},
		{"numeric value", `{"id":"PVT_abc","number":7}`, "number", "7"},
		{"numeric trailing brace", `{"number":42}`, "number", "42"},
		{"missing key", `{"id":"x"}`, "nope", ""},
		{"spaced after colon", `{"number": 12,"x":1}`, "number", "12"},
		{"pretty printed", "{\n  \"id\": \"PVT_z\",\n  \"number\": 3\n}", "id", "PVT_z"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractJSON(c.json, c.key); got != c.want {
				t.Errorf("extractJSON(%q, %q) = %q, want %q", c.json, c.key, got, c.want)
			}
		})
	}
}
