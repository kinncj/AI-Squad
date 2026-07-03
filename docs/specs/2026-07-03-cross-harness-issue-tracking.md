# Design — Cross-harness version / issue / milestone / project tracking

Date: 2026-07-03
Repo: MAPLE (`template/` tree — what `maple init` copies into user projects)
Reference: Heimdall PR #131 (initial rule) + PR #153 (orchestrator follow-up), with
every review finding from both pre-fixed.

## Goal

Make every change a MAPLE-initialised project works on tracked the same way, in
every harness (Claude, OpenCode, Copilot, Cursor):

1. If a GitHub repo is set up → the change gets a **GitHub issue** labelled
   `type:bug` / `type:feature`.
2. **Milestones** exist for **major & minor only** (`vX.Y.0`); patches attach to
   their minor's milestone, never their own — unless the user opts into per-patch.
3. The issue lands on the **configured project board** with a **Status**
   (`In Progress` → `Done`).
4. All of it is driven by `project.config.yaml → github`, never hard-coded.
5. When milestones or a board are **not configured**, the agent **asks the user
   once** and persists the answer so it never re-asks.

This is the MAPLE port of Heimdall PR #131 with every review finding from that PR
fixed up front, plus the interactive bootstrap (Heimdall's config was pre-filled;
MAPLE ships `null`s).

## Non-goals

- No new `maple` subcommand. `maple project bootstrap` / `maple labels` already
  exist; the rule points at them and at the existing skills.
- No change to the SemVer scheme itself — the project may override it in
  `project.config.yaml`; the rule reads whatever is configured.
- No backfill of issues for past work.

## Decisions (locked with the user)

| Decision | Choice |
|---|---|
| Enforcement scope | **Docs + skills + orchestrator** — standing rule in all instruction surfaces, the 3 `gh-*` skills read the new config and drive check→ask→persist, and the orchestrator runs the check at pipeline start. |
| "Declined" state | **Sentinel values in `project.config.yaml`** (`milestone_granularity: none`, `project_number: 0`). `null` = not yet asked; sentinel = asked and declined, never re-ask. |
| Labels | `type:bug` / `type:feature` — the labels `gh-labels-milestones` actually bootstraps. Never `bug`/`enhancement`. |
| Status casing | `In Progress` (capital P) → `Done` — matches `gh-projects` single-select option names (`Todo` / `In Progress` / `In Review` / `Done`). |

## Correctness constraints (Heimdall #131 review findings — all pre-fixed)

1. **Schema.** `.claude/schemas/project.config.schema.json` sets
   `github.additionalProperties: false` and only allows `project_number` +
   `project_node_id`. The new keys (`status_field_id`, `labels`,
   `milestone_granularity`) MUST be added to the schema or config validation /
   IDE tooling rejects the file.
2. **Labels.** Docs/config saying `bug`/`enhancement` cause automation to create
   ad-hoc labels or fail. Use `type:bug`/`type:feature` everywhere, and keep the
   inline config comment in sync.
3. **Status casing.** `In progress` (lowercase p) will not match the Projects v2
   single-select option and the update silently no-ops. Use `In Progress`.
4. **Identical heading & wording (PR #153).** Every surface uses the exact same
   heading "Version & Issue Tracking (mandatory)" and the same patch/milestone
   definition. In the orchestrator agent, slot it into the existing
   `## GitHub Issue Management` section — do not invent a divergent subsection
   title. Divergent wording across harnesses is itself a defect.

## Backwards compatibility (must-not-crash)

Existing MAPLE projects already have a `project.config.yaml` WITHOUT the new keys.
`maple init` skips an existing config (init.go:296) and `maple update` never
overwrites it (menu.go:274), so those old configs persist. Verified safe:

- **No strict yaml decode anywhere.** The binary reads config line-by-line
  (`loadProjectName` in loaders.go; `maple project bootstrap` uses string-replace
  in gh_cmds.go). Unknown/new keys never crash it, and a config missing the new
  keys never crashes it either.
- **Skills/rule must treat a missing key exactly like `null`** — i.e. ask (or use
  the documented default), never assume the key is present. This is the core
  compat requirement: a project inited before this change has no `labels:`,
  `milestone_granularity:`, or `status_field_id:` at all.
- **JSON schema is IDE-only** (`# yaml-language-server: $schema=…`), not runtime.
  New keys are optional with defaults, so an old config still validates against
  the widened schema. `maple update` re-copies the whole `.claude/` tree
  (init.go:260), so existing projects receive the widened schema on update.
- **`maple project bootstrap` string-replace stays compatible.** It replaces the
  literal `project_number: null` / `project_node_id: null`. Adding an inline
  comment after `null` keeps that substring intact, so the replace still fires.
  It does NOT currently write `status_field_id` or `milestone_granularity` — the
  skills cache/populate those (binary extension is a nice-to-have, not required).
- **Verify:** `maple update` also re-syncs the root harness md files + `.github/`
  so existing users actually receive the new rule text (check the init.go copy
  list; if root md/.github are not in it, note the gap).

## Config schema change

Reconciled with Heimdall's fix commit (f4ea31f): **drop the `labels` config key**
(the `type:*` taxonomy is a fixed convention in `gh-labels-milestones`, not user
config — this removes the mapping-drift risk at the root). **Keep
`milestone_granularity` in config** (MAPLE's interactive/configurable requirement,
which Heimdall dropped) and add it to the schema properly.

`template/project.config.yaml` — extend the `github:` block:

```yaml
github:
  project_number: null          # null = ask once; 0 = declined; N = configured board
  project_node_id: null
  status_field_id: null         # cached Status single-select field id for gh-projects
  milestone_granularity: null   # null = ask once; none = declined; minor = major+minor (default when enabled); patch = also per-patch
  # Issue label taxonomy (type:bug | type:feature | type:docs | type:refactor |
  # type:chore) lives in the gh-labels-milestones skill, not here.
```

`template/.claude/schemas/project.config.schema.json` — under `properties.github.properties`, keep `additionalProperties: false` but add:

- `status_field_id`: `{ "type": ["string","null"], "default": null }` (matches Heimdall).
- `milestone_granularity`: `{ "type": ["string","null"], "enum": [null,"none","minor","patch"], "default": null }` (MAPLE-only).
- `project_number`: change to allow the `0` sentinel — `{ "type": ["integer","null"], "minimum": 0, "default": null }`.
- **No `labels` property** — dropped by design.
- Keep the schema's existing single-line enum style; add only the new properties (surgical diff).

## The standing rule (identical intent, harness-appropriate phrasing)

Section title: **"Version & Issue Tracking (mandatory)"**. Body:

1. **Gate on a GitHub repo.** If `project.repo` is empty and there is no
   `git remote`, skip everything silently.
2. **Classify the SemVer bump** — major (breaking) / minor (new
   backward-compatible feature) / patch (fix, no API change), per the scheme in
   `project.config.yaml`.
3. **Milestone — major & minor only.** Read `github.milestone_granularity`.
   - `null` → ask the user once: "Track milestones for this project?" Persist
     `minor` (yes) or `none` (no).
   - `none` → skip milestones.
   - `minor` → ensure `vX.Y.0` exists (create via `gh-labels-milestones`);
     patches attach to their minor's `vX.Y.0`.
   - `patch` → patches also get milestones.
4. **Issue — every change.** Create via `gh-issues`, labelled with the matching
   `type:*` label per `gh-labels-milestones` (`type:bug` fix · `type:feature`
   feature · `type:docs`/`type:refactor`/`type:chore`), assigned to the target
   milestone if milestones are enabled. The taxonomy is the skill's convention,
   not a config mapping.
5. **Project board.** Read `github.project_number`.
   - `null` → ask once: "Add issues to a GitHub project board?" If yes, bootstrap
     (`maple project bootstrap` / `gh-projects`) and write `project_number` +
     `project_node_id` + `status_field_id` back to config. If no, persist `0`.
   - `0` → skip the board.
   - `> 0` → add the issue and set Status `In Progress` while working, `Done`
     when shipped. Field name `Status`; option names exactly `Todo` /
     `In Progress` / `In Review` / `Done`.
6. **Link & close.** PR references the issue (`Closes #N`). On release the
   milestone version equals the release tag. **No Claude/Anthropic attribution**
   anywhere — commits, PR bodies, issues, releases.
7. Mechanics live in the maple skills — use them, don't hand-roll:
   `gh-labels-milestones`, `gh-issues`, `gh-projects`. Config lives in
   `project.config.yaml → github`.

## File surface

Mirror trees `.opencode/` and `.cursor/` are **real copies, not symlinks**
(`maple init` copies each tree). Every skill/agent edit applies to all three.

Standing-rule docs (7) — identical "Version & Issue Tracking (mandatory)" section,
harness-appropriate phrasing, same heading/patch definition everywhere:
- `template/CLAUDE.md` — upgrade pipeline rule #7, add the section.
- `template/OPENCODE.md`
- `template/CURSOR.md`   (root harness md — keep in parity, per harness-doc-parity)
- `template/COPILOT.md`  (root harness md — keep in parity)
- `template/AGENTS.md`
- `template/.github/copilot-instructions.md`
- `template/.cursor/rules/version-and-issues.mdc` (new, `alwaysApply: true`).

The four root files CLAUDE/OPENCODE/CURSOR/COPILOT must stay byte-parallel in this
section (harness-doc-parity rule). COPILOT.md and CURSOR.md are separate from
`.github/copilot-instructions.md` and `.cursor/rules/`; all get the rule.

Config + schema (2):
- `template/project.config.yaml`
- `template/.claude/schemas/project.config.schema.json`

Skills — `type:*` taxonomy + ask/persist + `In Progress` casing, each in 3 mirror
copies (9 files):
- `gh-issues/SKILL.md` — label with the `type:*` taxonomy convention; attach the
  target milestone when milestones are enabled.
- `gh-labels-milestones/SKILL.md` — honour `milestone_granularity`
  (null→ask→persist `minor`/`none`); document the ask/persist helper. The `type:*`
  taxonomy already lives here — this is the single source of truth for labels.
- `gh-projects/SKILL.md` — `project_number` null→ask→persist; `0` sentinel skip;
  cache `status_field_id`; keep `In Progress` (capital P) — MAPLE keeps capital,
  unlike Heimdall's fix which unified on lowercase.
- Copies under `.claude/skills/`, `.opencode/skills/`, `.cursor/skills/`.

Orchestrator agent (3 mirror copies):
- `.claude/agents/orchestrator.md`, `.opencode/agents/orchestrator.md`,
  `.cursor/agents/orchestrator.md` — add the check under the EXISTING
  `## GitHub Issue Management` section (PR #153 precedent), using the identical
  "Version & Issue Tracking" heading/wording, pointing at the rule + skills.

Maple binary (verify, likely no change):
- `tui/gh_cmds.go`, `tui/init.go` — confirm any config read/write preserves the
  new `github` keys. Go yaml ignores unknown keys on read; a write-back must not
  drop them. Build check per CLAUDE.md (`make build-tui`).

Repo docs (MAPLE's own CLAUDE.md at repo root):
- Note the new config keys in the Shared State / config section if relevant.

## Dogfooding

This change itself is a MAPLE `enhancement`. Per MAPLE's own rules it should get
a GitHub issue on the `kinncj/MAPLE` repo, `enhancement` label, `v4.10.0`
milestone, and a card on project 67. Handle in the implementation phase.

Note the label distinction: the **MAPLE repo itself** uses `bug`/`enhancement`
(its own CLAUDE.md convention) — that is what this dogfood issue uses. The
**template** we ship to users uses `type:bug`/`type:feature` (what
`gh-labels-milestones` bootstraps). These are two different repos with two label
sets; not a contradiction.

## Testing / verification

- `make build-tui` green after any `tui/` touch.
- Validate the new `project.config.yaml` against the updated schema (no
  `additionalProperties` violation), AND validate a pre-change config (missing all
  new keys) against the new schema — both must pass (backwards compat).
- Grep the whole `template/` tree for stray `In progress` (lowercase) and
  `bug`/`enhancement` label literals in the new content — must be zero.
- Confirm the skill mirror copies stay byte-identical after edits
  (`diff -q` across `.claude`/`.opencode`/`.cursor` for each of the 3 skills and
  the orchestrator agent).
- Confirm the four root harness md files carry the identical rule section
  (harness-doc-parity).
- Smoke: with a config that has no new keys, the rule/skill path must ask (not
  error). With `milestone_granularity: none` / `project_number: 0`, it must skip
  silently without re-asking.

## Rollout order

1. Schema + config (foundation everything else references).
2. Skills (mechanics) × 3 copies.
3. Standing-rule docs × 7.
4. Orchestrator agent × 3 copies.
5. Binary verification + build.
6. Parity grep + schema validation.
