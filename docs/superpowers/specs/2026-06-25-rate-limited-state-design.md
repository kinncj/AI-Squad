# RATE_LIMITED pipeline state — design

Date: 2026-06-25
Status: approved (design), pending implementation plan
Bump: minor (`v4.x` stream) — new TUI state, overlay behavior, keybindings

---

## Context

`README.md:184-185` documents a `RATE_LIMITED` pipeline state: on a `429` the job state is set to `RATE_LIMITED` and "resumes when the window clears." None of it is wired up. The TUI knows four statuses (`RUNNING`, `PAUSED`, `DONE`, `FAILED`) in `pipeline_state.go:17`; a `RATE_LIMITED` value falls through `statusIcon()` to the generic `·` dot. No code writes, reads, or resumes on it.

The harness (Claude Code, opencode, …) is the process that actually hits the `429`. The pipeline-runner skill is a prompt running *inside* that harness. At the moment a hard limit bites, the LLM frequently cannot make the call that would write its own state — being rate-limited is precisely not being able to act. Detection has to account for that.

What already exists and gets reused:
- `maple.json` persists to disk; `reload()` polls it every 5s.
- The pipeline overlay renders per-status colors (`dashboard_views.go:841`).
- Resume machinery: `agentOpenCmd` (`dashboard.go:1879`), `runResumeSession` (`main.go:397`), pinned `sessions.json`.
- A skill→TUI signal-file pattern: `approval-pending.txt` (skill writes, TUI reads-and-deletes).

## Goals

- A real fifth state `RATE_LIMITED` that the TUI renders as a yellow flag with a live countdown.
- Persist enough to resume the exact pipeline stage — including across days.
- Resume when the window clears: manual `[r]` by default, opt-in auto.

## Non-goals

- The PTY-wrapping launch-wrapper detector (detector B). Deferred; the design keeps the TUI owning persist/render/resume so B can be added later without rework.
- Changing how soft/transient `429`s are handled. The harness already auto-retries those; we act only on limits that actually stop the run.

---

## Decisions taken

1. **Resume model: hybrid.** Manual `[r]` is the floor; an opt-in `[A]` arms automatic resume for unattended overnight continuation.
2. **Detection: A + C.** Cooperative agent write (primary, knows the real reset time) plus a TUI reclassify safety net for hard cutoffs. Detector B deferred.

---

## 1. State model — `maple.json`

New status value `RATE_LIMITED` and four fields. Ownership follows the existing merge-not-overwrite protocol.

| Field | Owner | Meaning |
|---|---|---|
| `status: "RATE_LIMITED"` | skill (primary) / TUI (reclassify) | the fifth state |
| `resume_at` | skill / TUI | ISO8601 — when the window clears. Drives countdown + resume trigger |
| `rate_limit_reason` | skill / TUI | one line (`"5-hour usage limit"`, `"API 429 retry-after 3600s"`) |
| `harness` | TUI (written at launch) | which harness ran this pipeline, so resume relaunches the right one |
| `auto_resume` | TUI | armed/disarmed by `[A]`; survives the skill's merge-writes |

`stage` + `taffy` (already present) are the resume coordinates. All fields persist on disk — close the laptop, reopen the next day, state is intact.

`pipeline_state.go` struct gains: `ResumeAt`, `RateLimitReason`, `Harness`, `AutoResume`.

## 2. Detection A — cooperative

Two surfaces get a new instruction block:
- `template/.claude/skills/pipeline-runner/SKILL.md`: a new "Rate-limit handling" section.
- `dashboard.go` `buildQuickPromptCmd` (≈ line 2156): a `<maple-rate-limit>` block appended to launches (taffy and skill kinds).

Instruction text:

> On a usage-limit / 429 / "resets at" message that stops you continuing, before stopping:
> 1. Merge-write `.claude/state/maple.json`: `status=RATE_LIMITED`, keep `stage`/`taffy`, set `resume_at` to the reset time named in the message (ISO8601; if none given, `now + 1h`), set `rate_limit_reason` to a one-line summary.
> 2. Write a one-line breadcrumb `resume_at|reason` to `.claude/state/rate-limit.txt` — a single cheap write; do it even if you can do nothing else.
> 3. Stop. Do not burn retries hammering the API.

## 3. Detection C — reclassify safety net

`.claude/state/rate-limit.txt` is a skill→TUI signal file, same contract as `approval-pending.txt`: **skill writes, TUI reads-and-deletes.**

In `reload()`: when the pipeline is `RUNNING` + `isStale()` **and** the breadcrumb exists, the TUI promotes `maple.json` to `RATE_LIMITED` (merge-write, pulling `resume_at`/`reason` from the breadcrumb), then deletes the breadcrumb. This is the sanctioned exception to "status is skill-owned" — the point of C is that the skill could not finish the write. The transition is owned by the TUI, mirroring how the TUI already owns deleting `approval-pending.txt` on approve.

Manual escape hatch: `[m]` marks a stale `RUNNING` pipeline as `RATE_LIMITED`, defaulting `resume_at = now + 1h`.

`isStale()` needs no change — it already returns false for any non-`RUNNING` status, so `RATE_LIMITED` is never treated as stale.

## 4. Resume — hybrid

Per tick in `reload()`, when `status == RATE_LIMITED`:

| Condition | Behavior |
|---|---|
| not cleared (`now < resume_at`) | render countdown `Resets in 02:14:09 (3:00 PM)` |
| cleared, `auto_resume` OFF | render `cleared — [r] resume` |
| cleared, `auto_resume` ON | fire resume once (guarded), with a status-bar line so it is never silent |

Resume action (shared by `[r]` and auto):
- Relaunch the recorded `harness`'s pinned session through the existing `agentOpenCmd` + `trySpawnCmd` path. **Never `tea.Quit`** — honors the "maple never exits when launching a harness" invariant.
- Inject a `<maple-resume>` prompt: *"resume taffy `<name>` at stage `<stage>`; it was paused on a rate limit; continue from there."*
- Set `status → RUNNING`, clear `resume_at`/`rate_limit_reason`; the agent retakes heartbeats.

**Thrash guard:** if a resumed run re-hits the limit within ~5 min (new `RATE_LIMITED` with `resume_at` close to the last one), `auto_resume` disarms and falls back to manual, so a persistent limit cannot spin a relaunch loop.

When the pipeline has no recorded `harness` or no pinned session (launched outside the TUI), resume falls back to the single pinned session if one exists, else shows the manual-launch modal with the resume command.

## 5. TUI rendering + keys

- `statusIcon()`: add `case "RATE_LIMITED": return "⚑"`.
- Color: the existing `t.Warning` (yellow) theme color (`themes.go:17`) — no theme change needed.
- `pipelineView` gains a `RATE_LIMITED` block: countdown, absolute reset time, `rate_limit_reason`, and `[r] resume now · [A] auto-resume: ON/OFF · [c] clear`.
- Footer keys for `showPipeline` add the `RATE_LIMITED` variant.
- Header taffy badge (`dashboard.go:1614`) counts `RATE_LIMITED` as in-flight, alongside `RUNNING`/`PAUSED`.

Key handlers, grouped in the pipeline overlay block (before the global switch, per the CLAUDE.md ordering rule):
- `[r]` — resume now (when `RATE_LIMITED`).
- `[A]` — toggle `auto_resume`.
- `[m]` — mark a stale `RUNNING` pipeline as `RATE_LIMITED`.

## 6. Docs + session-start protocol

- `template/CLAUDE.md` session-start check (and the `OPENCODE.md` / `CURSOR.md` / `COPILOT.md` mirrors) treats only `RUNNING`/`PAUSED` as "pipeline active." Add `RATE_LIMITED` → "paused on a rate limit; resume it, do not start a parallel pipeline."
- `README.md:184-185` — rewrite the FAFE claim to match what ships.

## 7. Testing / verify

Go unit tests (`tui/*_test.go`, in-package):
- `statusIcon()` returns `⚑` for `RATE_LIMITED`.
- `isRateLimited()` / `windowCleared()` boundaries around `resume_at`.
- `resume_at` parse (valid ISO8601, empty, malformed).
- breadcrumb reclassify: `RUNNING`+stale+breadcrumb → promoted `RATE_LIMITED` with parsed fields; breadcrumb deleted.
- auto-resume guard fires once; thrash guard disarms on a near-immediate re-hit.
- header badge counts `RATE_LIMITED`.
- `buildQuickPromptCmd` output contains the `<maple-rate-limit>` block.

Build gate (CLAUDE.md mandate): `cd tui && go build -o /tmp/maple_test .` green. `bash -n` on any touched script.

End-to-end verification scratch project: **`/tmp/maple_test_rate_limit`**. Run `maple init` there, hand-write `maple.json` with `RATE_LIMITED` + a future and a past `resume_at`, launch `maple`, and confirm: yellow flag + countdown render, `[r]`/`[A]`/`[c]`/`[m]` behavior, badge count, and the cleared→resume transition. Keep this folder out of the repo.

---

## Files to change

| File | Change |
|---|---|
| `tui/pipeline_state.go` | new fields; `statusIcon` case; `isRateLimited()`/`windowCleared()`; breadcrumb-reclassify + parse helpers; breadcrumb path const |
| `tui/dashboard.go` | `reload()` reclassify + auto-resume trigger + thrash guard; `[r]`/`[A]`/`[m]` handlers; `writeQuickLaunchState` records `harness`; `buildQuickPromptCmd` `<maple-rate-limit>` block; resume command + `<maple-resume>` prompt; badge count |
| `tui/dashboard_views.go` | `pipelineView` RATE_LIMITED block; footer keys; reuse `t.Warning` |
| `template/.claude/skills/pipeline-runner/SKILL.md` | "Rate-limit handling" section; state-table rows; `rate-limit.txt` signal-file row |
| `template/CLAUDE.md` (+ `OPENCODE.md`/`CURSOR.md`/`COPILOT.md`) | session-start protocol adds `RATE_LIMITED` |
| `README.md` | rewrite FAFE lines 184-185 |
| `tui/*_test.go` | unit tests above |

## Risks

- **Cooperative detection misses hard mid-think cutoffs.** Mitigated by C (breadcrumb + manual `[m]`). A frozen agent with no breadcrumb still shows as stale, same as today — no regression.
- **Auto-resume spawns a terminal unattended.** Off by default; thrash guard caps relaunch loops; status-bar line keeps it visible.
- **TUI writing a skill-owned field.** Scoped to the reclassify promotion and `[m]`, both documented as sanctioned exceptions and analogous to the existing `approval-pending.txt` deletion.
