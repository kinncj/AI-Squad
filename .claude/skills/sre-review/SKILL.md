---
name: sre-review
description: "Evaluate operational readiness: SLOs, alerting, runbooks, and rollback plan before launch. Use when reviewing a feature for production readiness."
---

# SKILL: SRE Review

## Purpose
Evaluate operational readiness of new features before launch.

## Failure Mode Analysis
For each new component, document:
1. **What can fail?** (service down, latency spike, data corruption, cascade)
2. **Blast radius?** (who is affected? how many users?)
3. **Detection time?** (how quickly will alerts fire?)
4. **Recovery time?** (how long to restore service?)
5. **Can it be rolled back?** (feature flag? migration rollback?)

## Observability Requirements

### Metrics (Prometheus/CloudWatch/Datadog)
```
{feature}_requests_total{status="success|error"} counter
{feature}_request_duration_seconds histogram
{feature}_active_connections gauge
{feature}_errors_total{type="validation|timeout|upstream"} counter
```

### Logs (Structured JSON)
```json
{
  "timestamp": "ISO8601",
  "level": "info|warn|error",
  "service": "{service-name}",
  "trace_id": "{distributed-trace-id}",
  "user_id": "{anonymized}",
  "action": "{what happened}",
  "duration_ms": 42,
  "result": "success|error",
  "error": "{message if error}"
}
```

### Traces
- Instrument all cross-service calls with OpenTelemetry.
- Trace IDs propagated via W3C Trace Context headers.

### Alerts
| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| High error rate | error_rate > 1% for 5m | P1 | Page on-call |
| Slow responses | p99 > 2s for 10m | P2 | Notify team |
| Saturation | CPU > 80% for 15m | P2 | Scale out |

## Runbook Template
`docs/runbooks/{feature}-runbook.md`

```markdown
# Runbook: {Feature Name}

## Symptoms
- {Alert name}: {what the user sees}

## Diagnosis
1. Check logs: `kubectl logs -l app={service} --tail=100`
2. Check metrics: {dashboard URL}
3. Check dependencies: {health check commands}

## Mitigation
### Option A: Restart service
```bash
kubectl rollout restart deployment/{service}
kubectl rollout status deployment/{service}
```

### Option B: Feature flag off
```bash
# Disable via environment variable
kubectl set env deployment/{service} FEATURE_{NAME}_ENABLED=false
```

## Escalation
- On-call: {PagerDuty rotation}
- Escalation: {eng manager}
- War room: {Slack channel}
```
