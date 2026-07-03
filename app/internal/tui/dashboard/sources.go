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
