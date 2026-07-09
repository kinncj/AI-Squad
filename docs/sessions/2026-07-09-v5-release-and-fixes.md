# Session — 2026-07-09

Shipped MAPLE **v5** to stable (the `app/` rebuild), reconciled the build/release
pipeline, fixed two release-blocking bugs and the copilot session crash, and rebuilt the
README around real end-to-end screenshots.

## What was done

### Pending-parity batch (branch work, then merged)
- `resume-session` (claude) uses the bare UUID, not the JSONL path
- Manual-launch copy-command overlay when a split fails
- `:debug` tees a one-line TUI state snapshot to `.claude/logs/tui.log`
- Narrow-terminal (<80 col) single-column view
- Git-changes overlay: file list + per-file diff nav (`C`, `j/k`)
- Per-story phase: skill writes `{status,phase}` to `story-status.json`; portal stepper shows the exact per-story phase
- Splash PNG inside tmux via passthrough (`allow-passthrough` + `MAPLE_TERM_GFX`)

### Build / release reconciliation
- Makefile: one `build` recipe parametrized by `OUT`/`GOOS`/`GOARCH`/`VERSION`; `build-tui`, `build-app`, `build-tui-all` delegate to it
- CI (`ci.yml`) calls the make targets instead of inlining the embed dance
- `release.yml` calls `make build`; RC/prerelease handling removed
- Aligned self-update repo constant to `kinncj/MAPLE`

### v5 released
- Merged PR #23 → `main`; released **v5.0.0**, then **v5.0.1 / v5.0.2 / v5.0.3**
- README: new logo hero, real screenshots, Go 1.26 build requirement; quickstarts bumped to Go 1.26

## Decisions made
- **Release from `main` via tag push, not `gh release create`.** A pre-created published release is discoverable before its binaries exist → install 404. Workflow now uploads to a **draft** then flips public. Documented in CLAUDE.md.
- **A TAFFY implement/resume launch never resumes a session.** Resuming a specific session belongs only to the explicit session-menu paths (dashboard `o`, `maple resume-session`). This is the user's call and the definitive fix for the copilot crash.
- **README uses real screenshots** (user-provided, ordered by creation timestamp into a run narrative) over synthetic mockups.

## Fixes applied
- **Release binaries wiped (v5.0.0):** `release.yml` triggered on BOTH `push: tags` and `release: published`; two concurrent runs raced softprops' delete-then-upload. Fix: single `push: tags` trigger + `concurrency` guard. (#25-adjacent)
- **install.sh 404 (#25):** release was published before binaries uploaded. Fix: draft→publish in the workflow; retry the download in `install.sh`/`install.ps1`. Fixed in v5.0.2.
- **copilot `No session matched` (#24):** `buildLaunchCmd` passed a pinned session id to `--resume`. v5.0.1 resumed only real sessions; v5.0.2 tried `--session-id`; **v5.0.3** removes session resume from TAFFY launches entirely (the real fix). All three req launchers (`ResumeArgs`, `ImplementArgs`, `ui.go`) now launch fresh.

## Unfinished / follow-up
- **#15 (open):** PostToolUse-hook signal for sub-second TUI refresh — still an enhancement (SSE + 2s tick already cover most of it).
- The synthetic `dashboard.png` render had box-border drift (browser-font terminal glyphs); superseded by real screenshots, so no longer relevant.
- Confirm on the user's machine: `maple --version` reports `v5.0.3` after reinstall (their earlier installs failed on the 404, so they were running an old binary).

## Pending dashboard / manual actions
- **User:** reinstall / `maple self-update` to land on **v5.0.3** (verify `maple --version`). Earlier installs 404'd, so the copilot fix wasn't active.
- None on any external dashboard (Supabase/Vercel/etc.) — not applicable to this repo.

## Commits
- `f5611ec` fix [i] targeting wrong story; portal stops auto-re-rendering feedback box
- `dea4df2` portal: story view shows everything related (all docs/, shared design system, references)
- `999d9a4` portal: NEEDS REVIEW highlighting for harness-flagged files
- `11aa896` resume-session UUID; narrow view; :debug tee; manual-launch command
- `0b2163f` git overlay + per-story phase
- `c0eed2c` splash PNG in tmux via passthrough
- `c523a5e` makefile single build recipe; CI/release use it; drop rc action
- `dcc4132` Merge PR #23 — v5.0.0
- `ac1c888` release: drop release-published trigger + concurrency
- `66deadb` copilot: don't --resume a missing session
- `4ebe430` release: draft→publish (no 404 window) + install retry
- `d45f471` readme: new logo hero + screenshots; go 1.26
- `f5ebdb4` taffy launch: never resume a session
- `c3aaa66` readme: real end-to-end screenshots

## Releases
- v5.0.0 — app/ rebuild to stable
- v5.0.1 — copilot resume fix (installs 404'd; superseded)
- v5.0.2 — no-404 release flow + install retry
- v5.0.3 — TAFFY launch never resumes a session (definitive copilot fix)

## Issues
- Closed: #14, #16, #17, #18 (stale/obsolete), #24 (copilot), #25 (install 404)
- Open: #15 (PostToolUse sub-second refresh)
