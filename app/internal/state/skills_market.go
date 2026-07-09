package state

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// skillsRun invokes the `skills` marketplace CLI via npx with colour disabled and
// auto-confirm, returning cleaned combined output.
func skillsRun(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "npx", args...)
	c.Env = append(os.Environ(), "NO_COLOR=1", "FORCE_COLOR=0", "npm_config_yes=true")
	out, err := c.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "(skills command timed out)", err
	}
	return ansiRe.ReplaceAllString(string(out), ""), err
}

func skillsLines(out string, emptyMsg string) []string {
	out = strings.TrimRight(out, "\n")
	if strings.TrimSpace(out) == "" {
		return []string{emptyMsg}
	}
	return strings.Split(out, "\n")
}

// SkillsSearch runs `npx skills search <query>` and returns the result lines.
func (s *FS) SkillsSearch(query string) []string {
	out, err := skillsRun("skills", "search", query)
	if err != nil && strings.TrimSpace(out) == "" {
		return []string{"(skills search failed — is `npx skills` available?)"}
	}
	return skillsLines(out, "(no skills matched \""+query+"\")")
}

// SkillInstall runs `npx skills add <pkg>` and returns the command output.
func (s *FS) SkillInstall(pkg string) []string {
	out, _ := skillsRun("skills", "add", pkg, "--all", "-y")
	return skillsLines(out, "(installed "+pkg+")")
}

// SkillRemove runs `npx skills remove <name>` and returns the command output.
func (s *FS) SkillRemove(name string) []string {
	out, _ := skillsRun("skills", "remove", name, "--all", "-y")
	return skillsLines(out, "(removed "+name+")")
}

// ShipSafeAudit runs `npx ship-safe audit .` and returns its output as lines.
func (s *FS) ShipSafeAudit() []string {
	return runTest(s.Root, []string{"npx", "ship-safe", "audit", "."})
}
