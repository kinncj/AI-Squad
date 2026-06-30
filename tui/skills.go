package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type skillRow struct {
	pkg      string // "owner/repo@skill"
	installs string // "21.1K"
	url      string // "https://skills.sh/..."
}

type installedSkillRow struct {
	name  string
	pkg   string // source package e.g. "obra/superpowers"
	agent string // "claude-code", "cursor", etc.
	scope string // "project" or "global"
}

type skillsSearchedMsg struct {
	items    []skillRow
	err      error
	searched bool // true = search ran (even if 0 results)
}

type installedLoadedMsg struct {
	items []installedSkillRow
	err   error
}

type skillInstalledMsg struct {
	pkg string
	err error
}

type skillRemovedMsg struct {
	name string
	err  error
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[mGKHF]`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func searchSkillsCmd(query string) tea.Cmd {
	return func() tea.Msg {
		args := []string{"skills", "find"}
		if query != "" {
			args = append(args, query)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeoutNetwork)
		defer cancel()
		cmd := exec.CommandContext(ctx, "npx", args...)
		// npm_config_yes=true auto-confirms package install without needing package.json.
		// NO_COLOR/FORCE_COLOR strip ANSI so parseSkillsOutput gets clean ASCII.
		cmd.Env = append(os.Environ(), "NO_COLOR=1", "FORCE_COLOR=0", "npm_config_yes=true")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		_ = cmd.Run() // ignore exit code — parse whatever stdout we got

		items := parseSkillsOutput(stripANSI(stdout.String()))
		if len(items) == 0 {
			errStr := strings.TrimSpace(stderr.String())
			if errStr != "" {
				return skillsSearchedMsg{searched: true, err: fmt.Errorf("%s", stripANSI(errStr))}
			}
		}
		return skillsSearchedMsg{items: items, searched: true}
	}
}

func listInstalledSkillsCmd() tea.Cmd {
	return func() tea.Msg {
		var allItems []installedSkillRow
		for _, scope := range []struct{ flag, label string }{
			{"", "project"},
			{"-g", "global"},
		} {
			args := []string{"skills", "ls", "--json"}
			if scope.flag != "" {
				args = append(args, scope.flag)
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeoutNetwork)
			cmd := exec.CommandContext(ctx, "npx", args...)
			cmd.Env = append(os.Environ(), "NO_COLOR=1", "FORCE_COLOR=0", "npm_config_yes=true")
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			_ = cmd.Run()
			cancel()

			items := parseInstalledJSON(stdout.String(), scope.label)
			if len(items) == 0 {
				// fallback: plain-text parse
				items = parseInstalledText(stdout.String(), scope.label)
			}
			allItems = append(allItems, items...)
		}
		return installedLoadedMsg{items: allItems}
	}
}

func removeSkillCmd(name string) tea.Cmd {
	return func() tea.Msg {
		env := append(os.Environ(), "NO_COLOR=1", "FORCE_COLOR=0", "npm_config_yes=true")
		out, err := runWithTimeout(timeoutNetwork, env, "npx", "skills", "remove", name, "--all", "-y")
		if err != nil {
			msg := strings.TrimSpace(stripANSI(string(out)))
			if msg == "" {
				msg = err.Error()
			}
			return skillRemovedMsg{name: name, err: fmt.Errorf("%s", msg)}
		}
		return skillRemovedMsg{name: name}
	}
}

func parseInstalledJSON(s, scope string) []installedSkillRow {
	// Try array of objects first: [{"name":"...","package":"...","agent":"..."}]
	var arr []struct {
		Name    string `json:"name"`
		Package string `json:"package"`
		Pkg     string `json:"pkg"`
		Agent   string `json:"agent"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &arr); err == nil {
		var rows []installedSkillRow
		for _, a := range arr {
			pkg := a.Package
			if pkg == "" {
				pkg = a.Pkg
			}
			rows = append(rows, installedSkillRow{name: a.Name, pkg: pkg, agent: a.Agent, scope: scope})
		}
		return rows
	}

	// Try map of agent → []skill
	var m map[string][]struct {
		Name    string `json:"name"`
		Package string `json:"package"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &m); err == nil {
		var rows []installedSkillRow
		for agent, skills := range m {
			for _, sk := range skills {
				rows = append(rows, installedSkillRow{name: sk.Name, pkg: sk.Package, agent: agent, scope: scope})
			}
		}
		return rows
	}
	return nil
}

func parseInstalledText(s, scope string) []installedSkillRow {
	// Plain text lines like "  skill-name   owner/repo   agent"
	var rows []installedSkillRow
	var curAgent string
	for _, line := range strings.Split(s, "\n") {
		stripped := strings.TrimSpace(line)
		if stripped == "" || strings.HasPrefix(stripped, "SKILLS") || strings.HasPrefix(stripped, "███") {
			continue
		}
		// Agent header: "Project skills (claude-code):"
		if strings.HasSuffix(stripped, ":") && strings.Contains(stripped, "(") {
			start := strings.Index(stripped, "(")
			end := strings.Index(stripped, ")")
			if start >= 0 && end > start {
				curAgent = stripped[start+1 : end]
			}
			continue
		}
		parts := strings.Fields(stripped)
		if len(parts) >= 1 && !strings.HasPrefix(stripped, "#") {
			name := parts[0]
			pkg := ""
			if len(parts) >= 2 {
				pkg = parts[1]
			}
			rows = append(rows, installedSkillRow{name: name, pkg: pkg, agent: curAgent, scope: scope})
		}
	}
	return rows
}

func installSkillCmd(pkg string) tea.Cmd {
	return func() tea.Msg {
		env := append(os.Environ(), "NO_COLOR=1", "FORCE_COLOR=0", "npm_config_yes=true")
		out, err := runWithTimeout(timeoutNetwork, env, "npx", "skills", "add", pkg, "--all", "-y")
		if err != nil {
			msg := strings.TrimSpace(stripANSI(string(out)))
			if msg == "" {
				msg = err.Error()
			}
			return skillInstalledMsg{pkg: pkg, err: fmt.Errorf("%s", msg)}
		}
		return skillInstalledMsg{pkg: pkg}
	}
}

// parseSkillsOutput parses the output of `npx skills find <query>`.
// Each result looks like:
//
//	vercel-labs/agent-browser@dogfood 21.1K installs
//	└ https://skills.sh/vercel-labs/agent-browser/dogfood
func parseSkillsOutput(s string) []skillRow {
	var rows []skillRow
	lastWasPkg := false
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			lastWasPkg = false
			continue
		}
		if strings.HasPrefix(line, "└ ") {
			if lastWasPkg && len(rows) > 0 {
				rows[len(rows)-1].url = strings.TrimPrefix(line, "└ ")
			}
			lastWasPkg = false
			continue
		}
		// "owner/repo@skill N installs" or "owner/repo@skill N,NNN installs"
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[len(parts)-1] == "installs" && strings.Contains(parts[0], "/") {
			rows = append(rows, skillRow{
				pkg:      parts[0],
				installs: parts[len(parts)-2],
			})
			lastWasPkg = true
		} else {
			lastWasPkg = false
		}
	}
	return rows
}
