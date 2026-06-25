---
description: Implementation file guard — reminds the agent to verify pipeline state before editing app/ or tests/
globs: ["app/**", "tests/**"]
alwaysApply: false
---

You are editing implementation files. Before making any changes, verify the MAPLE pipeline is active:

```bash
python3 -c "import json; s=json.load(open('.claude/state/maple.json')); print(s.get('stage',''), s.get('status',''))" 2>/dev/null || echo "no pipeline"
```

If status is not `RUNNING` or `PAUSED`, stop and start the pipeline first via `/pipeline-runner <name>`. Do not write to these files outside a pipeline stage.
