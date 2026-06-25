---
name: ship-safe
description: "Run ship-safe security and quality audit on the current project. Executes npx ship-safe audit . and reports findings by severity. Use before shipping any feature or PR."
---

# SKILL: Ship-Safe Audit

## What It Does

Runs [ship-safe](https://shipsafecli.com) — a pre-ship security and quality scanner that checks for secrets, vulnerabilities, and risky patterns before code reaches production.

## Usage

```bash
npx ship-safe audit .
```

Run from the project root. No install required (`npx` fetches it on demand).

## Opt-In

Ship-safe is **disabled by default**. To enable it:
- **CI/CD**: set the repository variable `ENABLE_SHIP_SAFE=true` in GitHub → Settings → Variables
- **Agents / local**: set env var `ENABLE_SHIP_SAFE=true` before invoking `/ship-safe`

## When to Use

Only run if `ENABLE_SHIP_SAFE=true` is set. When enabled, appropriate moments are:
- Before opening a PR
- After adding new dependencies
- Before merging any feature branch to main
- After touching auth, secrets handling, or infra config

## Output Interpretation

| Symbol / keyword | Severity | Action |
|---|---|---|
| `✓ PASS` / `ok` / `no issues` | Clean | Safe to ship |
| `⚠ WARN` / `MEDIUM` / `LOW` | Advisory | Review before shipping |
| `✗ FAIL` / `ERROR` / `CRITICAL` / `HIGH` | Blocker | Must fix before shipping |

## Agent Instructions

1. Run `npx ship-safe audit .` from the repo root.
2. Parse stdout for CRITICAL/HIGH findings — these are blockers.
3. For each blocker: report the file, line, and finding description.
4. For MEDIUM/LOW: report as advisory, do not block.
5. If all checks pass: confirm "ship-safe: clean" and proceed.
6. If blockers found: halt the task, report findings to Orchestrator.

## Example Integration (architect / pre-ship checklist)

```
/ship-safe
```

The skill runs the audit, colors findings by severity, and surfaces blockers before any merge action.
