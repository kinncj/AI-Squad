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

// sessionSource adapts harness sessions into a filterable, selectable pane source.
type sessionSource struct {
	all    []state.Session
	filter string
}

func newSessionSource(sessions []state.Session) *sessionSource {
	return &sessionSource{all: sessions}
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

// Rows renders each session as "[cc] title  Nt" (t = tool calls).
func (s *sessionSource) Rows() []string {
	f := s.filtered()
	rows := make([]string, len(f))
	for i, se := range f {
		rows[i] = fmt.Sprintf("[%s] %s  %dt", sourceTag(se.Source), se.Title, se.ToolCount)
	}
	return rows
}

func (s *sessionSource) RowCount() int      { return len(s.filtered()) }
func (s *sessionSource) SetFilter(q string) { s.filter = q }
