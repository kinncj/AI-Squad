package dashboard

import (
	"fmt"
	"strings"

	"github.com/kinncj/maple/app/internal/state"
)

// storySource adapts a slice of stories into a filterable, selectable pane source.
// It implements pane.Source, pane.Selectable, and pane.Filterable.
type storySource struct {
	all    []state.Story
	filter string
}

func newStorySource(stories []state.Story) *storySource {
	return &storySource{all: stories}
}

func (s *storySource) filtered() []state.Story {
	if s.filter == "" {
		return s.all
	}
	q := strings.ToLower(s.filter)
	var out []state.Story
	for _, st := range s.all {
		if strings.Contains(strings.ToLower(st.ID), q) ||
			strings.Contains(strings.ToLower(st.Phase), q) {
			out = append(out, st)
		}
	}
	return out
}

// Rows renders each story as "phase  id  ◈" (◈ marks UI-bearing stories).
func (s *storySource) Rows() []string {
	f := s.filtered()
	rows := make([]string, len(f))
	for i, st := range f {
		mark := ""
		if st.UI {
			mark = " ◈"
		}
		rows[i] = fmt.Sprintf("%-9s %s%s", st.Phase, st.ID, mark)
	}
	return rows
}

func (s *storySource) RowCount() int      { return len(s.filtered()) }
func (s *storySource) SetFilter(q string) { s.filter = q }

// at returns the story at the current filtered index i, if any.
func (s *storySource) at(i int) (state.Story, bool) {
	f := s.filtered()
	if i < 0 || i >= len(f) {
		return state.Story{}, false
	}
	return f[i], true
}

// sessionSource adapts harness sessions into a filterable, selectable pane source.
type sessionSource struct {
	all    []state.Session
	pinned map[string]string // pinned session id per source
	filter string
}

func newSessionSource(sessions []state.Session, pinned map[string]string) *sessionSource {
	if pinned == nil {
		pinned = map[string]string{}
	}
	return &sessionSource{all: sessions, pinned: pinned}
}

// sourceTag maps a session source to its two-letter dashboard badge.
func sourceTag(src string) string {
	switch src {
	case "claude":
		return "cc"
	case "opencode":
		return "oc"
	case "copilot":
		return "gh"
	default:
		return "??"
	}
}

func (s *sessionSource) filtered() []state.Session {
	if s.filter == "" {
		return s.all
	}
	q := strings.ToLower(s.filter)
	var out []state.Session
	for _, se := range s.all {
		if strings.Contains(strings.ToLower(se.Title), q) || strings.Contains(se.Source, q) {
			out = append(out, se)
		}
	}
	return out
}

// Rows renders each session as "● [cc] title  Nt" (● marks the pinned session for
// its source; t = tool calls).
func (s *sessionSource) Rows() []string {
	f := s.filtered()
	rows := make([]string, len(f))
	for i, se := range f {
		pin := "  "
		if s.pinned[se.Source] == se.ID && se.ID != "" {
			pin = "● "
		}
		rows[i] = fmt.Sprintf("%s[%s] %s  %dt", pin, sourceTag(se.Source), se.Title, se.ToolCount)
	}
	return rows
}

func (s *sessionSource) RowCount() int      { return len(s.filtered()) }
func (s *sessionSource) SetFilter(q string) { s.filter = q }

func (s *sessionSource) at(i int) (state.Session, bool) {
	f := s.filtered()
	if i < 0 || i >= len(f) {
		return state.Session{}, false
	}
	return f[i], true
}

// prSource adapts pull requests into a filterable, selectable pane source.
type prSource struct {
	all    []state.PullRequest
	filter string
}

func newPRSource(prs []state.PullRequest) *prSource { return &prSource{all: prs} }

func (s *prSource) filtered() []state.PullRequest {
	if s.filter == "" {
		return s.all
	}
	q := strings.ToLower(s.filter)
	var out []state.PullRequest
	for _, pr := range s.all {
		if strings.Contains(strings.ToLower(pr.Title), q) {
			out = append(out, pr)
		}
	}
	return out
}

// Rows renders each PR as "#22 title".
func (s *prSource) Rows() []string {
	f := s.filtered()
	rows := make([]string, len(f))
	for i, pr := range f {
		rows[i] = fmt.Sprintf("#%d %s", pr.Number, pr.Title)
	}
	return rows
}

func (s *prSource) RowCount() int      { return len(s.filtered()) }
func (s *prSource) SetFilter(q string) { s.filter = q }

func (s *prSource) at(i int) (state.PullRequest, bool) {
	f := s.filtered()
	if i < 0 || i >= len(f) {
		return state.PullRequest{}, false
	}
	return f[i], true
}

// qaSource adapts discovered tests into a filterable, selectable pane source.
type qaSource struct {
	all    []state.Test
	filter string
}

func newQASource(tests []state.Test) *qaSource { return &qaSource{all: tests} }

func (s *qaSource) filtered() []state.Test {
	if s.filter == "" {
		return s.all
	}
	q := strings.ToLower(s.filter)
	var out []state.Test
	for _, tst := range s.all {
		if strings.Contains(strings.ToLower(tst.Path), q) || strings.Contains(tst.Framework, q) {
			out = append(out, tst)
		}
	}
	return out
}

// Rows renders each test as "[go] path".
func (s *qaSource) Rows() []string {
	f := s.filtered()
	rows := make([]string, len(f))
	for i, tst := range f {
		rows[i] = fmt.Sprintf("[%s] %s", tst.Framework, tst.Path)
	}
	return rows
}

func (s *qaSource) RowCount() int      { return len(s.filtered()) }
func (s *qaSource) SetFilter(q string) { s.filter = q }

func (s *qaSource) at(i int) (state.Test, bool) {
	f := s.filtered()
	if i < 0 || i >= len(f) {
		return state.Test{}, false
	}
	return f[i], true
}
