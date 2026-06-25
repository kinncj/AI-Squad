---
description: MAPLE pipeline gate — enforces orchestrator-first workflow before any implementation work
alwaysApply: true
---

## Session Start Protocol (mandatory)

Before responding to any implementation request, run this check:

```bash
python3 -c "import json; s=json.load(open('.claude/state/maple.json')); print(s.get('status',''))" 2>/dev/null || echo "none"
```

- **`RUNNING` or `PAUSED`** — pipeline is active, continue within it.
- **anything else** — no pipeline active. Do NOT write to `app/` or `tests/`. Route through `/pipeline-runner` first:

```
/pipeline-runner implement-stories   — implement existing approved stories
/pipeline-runner new-ui-feature      — full UI pipeline with design gates
/pipeline-runner api-endpoint        — API feature pipeline
/pipeline-runner bugfix              — reproduce → fix → validate
```

Writing to `app/` or `tests/` outside a running pipeline is a hard violation. The pre-commit hook will block the commit.
