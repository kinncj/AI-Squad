// Package req implements `maple req`: it turns a free-text requirement into one or
// more Gherkin stories via an AI harness and writes them to docs/stories/. This file
// holds the pure (TUI-free, testable) parsing core. Ported from tui/req.go.
package req

import (
	"regexp"
	"strings"
)

// Story is one generated Gherkin story.
type Story struct {
	Title   string
	Gherkin string
	UI      bool
	SavedTo string
}

var storyHeaderRe = regexp.MustCompile(`(?m)^=== STORY: (.+?) ===$`)
var telemetryLineRe = regexp.MustCompile(`^(Changes|Requests|Tokens)\b`)

// ParseStories splits AI output into individual Story values. If no === STORY: ===
// delimiters are found the whole output is treated as one story.
func ParseStories(output string) []Story {
	output = sanitizeModelOutput(output)
	matches := storyHeaderRe.FindAllStringIndex(output, -1)
	if len(matches) == 0 {
		cleaned := cleanGherkin(output)
		return []Story{{
			Title:   extractFeatureTitle(cleaned),
			Gherkin: cleaned,
		}}
	}

	var stories []Story
	for i, match := range matches {
		titleMatch := storyHeaderRe.FindStringSubmatch(output[match[0]:match[1]])
		title := strings.TrimSpace(titleMatch[1])

		start := match[1]
		if start < len(output) && output[start] == '\n' {
			start++
		}
		var content string
		if i+1 < len(matches) {
			content = output[start:matches[i+1][0]]
		} else {
			content = output[start:]
		}
		cleaned := cleanGherkin(content)
		stories = append(stories, Story{
			Title:   title,
			Gherkin: cleaned,
		})
	}
	return stories
}

func sanitizeModelOutput(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var out []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "```" || trimmed == "```gherkin" {
			continue
		}
		if telemetryLineRe.MatchString(trimmed) {
			continue
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// extractFeatureTitle pulls the Feature: name from gherkin text, or returns "Untitled".
func extractFeatureTitle(s string) string {
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "Feature:") {
			t := strings.TrimSpace(strings.TrimPrefix(l, "Feature:"))
			if t != "" {
				return t
			}
		}
	}
	return "Untitled"
}

func stripFences(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if (i == 0 && (trimmed == "```gherkin" || trimmed == "```")) || trimmed == "```" || trimmed == "```gherkin" {
			continue
		}
		if telemetryLineRe.MatchString(trimmed) {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func cleanGherkin(s string) string {
	trimmed := strings.TrimSpace(stripFences(s))
	lines := strings.Split(trimmed, "\n")
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if telemetryLineRe.MatchString(t) {
			continue
		}
		if strings.HasPrefix(t, "```") {
			continue
		}
		out = append(out, l)
	}
	cleaned := strings.TrimSpace(strings.Join(out, "\n"))
	if idx := strings.Index(cleaned, "Feature:"); idx > 0 {
		cleaned = strings.TrimSpace(cleaned[idx:])
	}
	return cleaned
}

// InferUIStory reports whether the requirement/title/gherkin describe a visual feature.
func InferUIStory(requirements, title, gherkin string) bool {
	corpus := strings.ToLower(requirements + "\n" + title + "\n" + gherkin)
	for _, k := range []string{
		" ui ", "spa", "frontend", "front-end", "react", "vue", "angular",
		"screen", "button", "form", "layout", "look and feel", "design",
		"visual", "glass", "apple", "theme", "css", "tailwind",
	} {
		if strings.Contains(corpus, k) {
			return true
		}
	}
	return false
}
