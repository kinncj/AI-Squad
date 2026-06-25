package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type designArtifact struct {
	Kind    string // "wireframe" | "mockup" | "a11y"
	Path    string
	Status  string // "approved" | "draft" | "" (a11y has no status)
	Summary string
	Exists  bool
}

type designReview struct {
	StoryID   string
	Artifacts []designArtifact
}

var statusLineRe = regexp.MustCompile(`(?m)^status:\s*(\w+)`)

func parseArtifactStatus(content string) string {
	if m := statusLineRe.FindStringSubmatch(content); m != nil {
		return m[1]
	}
	return "draft"
}

func a11ySummary(jsonBytes []byte) (crit int, total int) {
	var data struct {
		Violations []struct {
			Impact string `json:"impact"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return 0, 0
	}
	total = len(data.Violations)
	for _, v := range data.Violations {
		if v.Impact == "critical" || v.Impact == "serious" {
			crit++
		}
	}
	return crit, total
}

func loadDesignReview(storyID string) designReview {
	r := designReview{StoryID: storyID}
	wf := designArtifact{Kind: "wireframe", Path: "docs/design/wireframes/" + storyID + ".wireframe.md"}
	if b, err := os.ReadFile(wf.Path); err == nil {
		wf.Exists = true
		wf.Status = parseArtifactStatus(string(b))
	}
	mk := designArtifact{Kind: "mockup", Path: "docs/design/mockups/" + storyID + ".mockup.md"}
	if b, err := os.ReadFile(mk.Path); err == nil {
		mk.Exists = true
		mk.Status = parseArtifactStatus(string(b))
	}
	a := designArtifact{Kind: "a11y", Path: "docs/design/mockups/" + storyID + ".a11y.json"}
	if b, err := os.ReadFile(a.Path); err == nil {
		a.Exists = true
		crit, total := a11ySummary(b)
		a.Summary = fmt.Sprintf("%d critical/serious", crit)
		if total == 0 {
			a.Summary = "no audit findings"
		}
	}
	r.Artifacts = []designArtifact{wf, mk, a}
	return r
}

func approveDesignArtifact(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	if statusLineRe.MatchString(content) {
		content = statusLineRe.ReplaceAllString(content, "status: approved")
	} else {
		content = "status: approved\n" + content
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func (m *dashboardModel) designReviewAllApproved() bool {
	for _, a := range m.designReview.Artifacts {
		if (a.Kind == "wireframe" || a.Kind == "mockup") && a.Status != "approved" {
			return false
		}
	}
	return true
}

func (a designArtifact) label() string {
	switch a.Kind {
	case "a11y":
		if !a.Exists {
			return "a11y.json        [missing]"
		}
		return "a11y.json        " + a.Summary
	default:
		st := a.Status
		if !a.Exists {
			st = "missing"
		}
		return fmt.Sprintf("%-16s [%s]", a.Kind+".md", strings.ToUpper(st))
	}
}
