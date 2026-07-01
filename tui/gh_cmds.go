package main

import (
	"fmt"
	"os"
	"strings"
)

// runLabels bootstraps the canonical MAPLE label set in the current repo.
func runLabels(gh string) error {
	if gh == "" {
		return fmt.Errorf("gh CLI not found — install it from https://cli.github.com")
	}

	type label struct {
		name  string
		color string
		desc  string
	}

	groups := []struct {
		title  string
		labels []label
	}{
		{"Work Type", []label{
			{"type:feature", "0075ca", "New feature or request"},
			{"type:bug", "d73a4a", "Something isn't working"},
			{"type:spike", "e4e669", "Technical investigation"},
			{"type:chore", "cfd3d7", "Maintenance, deps, CI"},
			{"type:docs", "0075ca", "Documentation only"},
			{"type:refactor", "fbca04", "Code restructure, no behaviour change"},
			{"type:hotfix", "b60205", "Critical production fix"},
		}},
		{"Pipeline Phase", []label{
			{"phase:discover", "c5def5", "DISCOVER phase"},
			{"phase:architect", "c5def5", "ARCHITECT phase"},
			{"phase:plan", "c5def5", "PLAN phase"},
			{"phase:infra", "c5def5", "INFRA phase"},
			{"phase:implement", "0e8a16", "IMPLEMENT phase"},
			{"phase:validate", "0e8a16", "VALIDATE phase"},
			{"phase:document", "0e8a16", "DOCUMENT phase"},
			{"phase:gate", "b60205", "FINAL GATE"},
		}},
		{"Priority", []label{
			{"priority:critical", "b60205", "Drop everything"},
			{"priority:high", "d93f0b", "Next sprint"},
			{"priority:medium", "fbca04", "Normal flow"},
			{"priority:low", "c5def5", "When bandwidth allows"},
		}},
		{"Specialist", []label{
			{"specialist:frontend", "1d76db", "Frontend work"},
			{"specialist:backend", "0075ca", "Backend work"},
			{"specialist:infra", "5319e7", "Infrastructure / DevOps"},
			{"specialist:data", "e4e669", "Data / ML / Analytics"},
			{"specialist:ux", "f9d0c4", "UX / Design"},
			{"specialist:qa", "0e8a16", "QA / Testing"},
		}},
		{"Status", []label{
			{"status:blocked", "b60205", "Blocked on dependency"},
			{"status:needs-review", "fbca04", "Awaiting review"},
			{"status:in-progress", "0e8a16", "Active work"},
			{"status:on-hold", "cfd3d7", "Intentionally paused"},
		}},
		{"Design", []label{
			{"design:wireframe-pending", "f9d0c4", "Wireframe not yet approved"},
			{"design:wireframe-approved", "0e8a16", "Wireframe approved"},
			{"design:mockup-pending", "f9d0c4", "Mockup not yet approved"},
			{"design:mockup-approved", "0e8a16", "Mockup approved"},
			{"design:a11y-pending", "fbca04", "A11y audit not yet run"},
			{"design:a11y-passed", "0e8a16", "A11y WCAG 2.2 AA passed"},
		}},
		{"Spec-Kit", []label{
			{"spec:problem", "c5def5", "PROBLEM.md stage"},
			{"spec:spec", "c5def5", "SPEC.md stage"},
			{"spec:plan", "c5def5", "PLAN.md stage"},
			{"spec:tasks", "0e8a16", "TASKS.md approved — ready for pipeline"},
		}},
		{"ADR", []label{
			{"adr:required", "fbca04", "ADR must be created before merge"},
			{"adr:linked", "0e8a16", "ADR created and linked"},
		}},
	}

	var created, skipped, failed int
	for _, g := range groups {
		fmt.Printf("\n  %s\n", g.title)
		for _, l := range g.labels {
			out, err := runWithTimeout(timeoutNetwork, nil, gh, "label", "create", l.name,
				"--color", l.color,
				"--description", l.desc,
				"--force",
			)
			if err != nil {
				if strings.Contains(string(out), "already exists") {
					fmt.Printf("    ~ %s\n", l.name)
					skipped++
				} else {
					fmt.Printf("    ✗ %s: %s\n", l.name, strings.TrimSpace(string(out)))
					failed++
				}
			} else {
				fmt.Printf("    ✓ %s\n", l.name)
				created++
			}
		}
	}

	fmt.Printf("\n  %d created  %d skipped  %d failed\n", created, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("%d labels failed to create", failed)
	}
	return nil
}

// runProject creates a GitHub Project v2 and writes project.config.yaml.
func runProject(gh string) error {
	if gh == "" {
		return fmt.Errorf("gh CLI not found — install it from https://cli.github.com")
	}

	// Get repo owner/name
	repoOut, err := runWithTimeout(timeoutNetwork, nil, gh, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return fmt.Errorf("gh repo view: %w", err)
	}
	repo := strings.TrimSpace(string(repoOut))
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("unexpected repo format: %s", repo)
	}
	owner := parts[0]

	fmt.Printf("  Creating GitHub Project v2 for %s...\n", repo)

	// Create project
	projOut, err := runWithTimeout(timeoutNetwork, nil, gh, "project", "create",
		"--owner", owner,
		"--title", "MAPLE",
		"--format", "json",
	)
	if err != nil {
		return fmt.Errorf("gh project create: %w", err)
	}

	// Parse project number and node ID from JSON
	projJSON := string(projOut)
	number := extractJSON(projJSON, "number")
	nodeID := extractJSON(projJSON, "id")

	if number == "" || nodeID == "" {
		return fmt.Errorf("could not parse project number/id from: %s", projJSON)
	}

	fmt.Printf("  ✓ Project created: number=%s node_id=%s\n", number, nodeID)

	// Update project.config.yaml
	cfg := "project.config.yaml"
	if _, err := os.Stat(cfg); os.IsNotExist(err) {
		fmt.Printf("  ✗ %s not found — run maple init first\n", cfg)
		return nil
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		return err
	}
	content := string(data)
	content = strings.ReplaceAll(content, "project_number: null", "project_number: "+number)
	content = strings.ReplaceAll(content, "project_node_id: null", "project_node_id: \""+nodeID+"\"")
	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Printf("  ✓ project.config.yaml updated\n")

	// Bootstrap custom fields
	if err := bootstrapProjectFields(gh, owner, number); err != nil {
		fmt.Printf("  ~ custom fields: %v (project was still created)\n", err)
	}
	return nil
}

// bootstrapProjectFields adds the standard MAPLE custom fields to a Project v2.
// number is the integer project number returned by gh project create.
func bootstrapProjectFields(gh, owner, number string) error {
	type field struct {
		name     string
		dataType string
		options  []string // for SINGLE_SELECT
	}
	fields := []field{
		{name: "Epic", dataType: "TEXT"},
		{name: "Phase", dataType: "SINGLE_SELECT", options: []string{
			"discover", "architect", "plan", "infra", "implement", "validate", "document", "gate",
		}},
		{name: "Type", dataType: "SINGLE_SELECT", options: []string{
			"feature", "bug", "spike", "chore", "docs", "refactor", "hotfix",
		}},
		{name: "Specialist", dataType: "SINGLE_SELECT", options: []string{
			"frontend", "backend", "infra", "data", "ux", "qa",
		}},
		{name: "ADR Required", dataType: "SINGLE_SELECT", options: []string{"yes", "no"}},
	}

	for _, f := range fields {
		args := []string{"project", "field-create", number,
			"--owner", owner,
			"--name", f.name,
			"--data-type", f.dataType,
		}
		if len(f.options) > 0 {
			args = append(args, "--single-select-options", strings.Join(f.options, ","))
		}
		out, err := runWithTimeout(timeoutNetwork, nil, gh, args...)
		if err != nil {
			if strings.Contains(string(out), "already exists") {
				fmt.Printf("    ~ field %q already exists\n", f.name)
			} else {
				fmt.Printf("    ✗ field %q: %s\n", f.name, strings.TrimSpace(string(out)))
			}
		} else {
			fmt.Printf("    ✓ field %q\n", f.name)
		}
	}
	return nil
}

// extractJSON pulls a top-level string/number value from simple JSON by key.
// Not a full parser — enough for gh CLI output.
func extractJSON(json, key string) string {
	search := fmt.Sprintf(`"%s":`, key)
	idx := strings.Index(json, search)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(json[idx+len(search):])
	if len(rest) == 0 {
		return ""
	}
	if rest[0] == '"' {
		end := strings.Index(rest[1:], `"`)
		if end < 0 {
			return ""
		}
		return rest[1 : end+1]
	}
	// numeric
	end := strings.IndexAny(rest, ",}\n ")
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}
