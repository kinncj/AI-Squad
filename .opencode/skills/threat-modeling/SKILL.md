---
name: threat-modeling
description: "Perform STRIDE threat modeling for each component and trust boundary and produce a threat register. Use when reviewing a feature for security."
---

# SKILL: Threat Modeling (STRIDE)

## Output Location
`docs/specs/{feature-slug}/threat-model.md`

## STRIDE Framework

For each component and trust boundary, evaluate all 6 threat categories:

| Letter | Threat | Question |
|--------|--------|----------|
| S | Spoofing | Can an attacker impersonate a user, service, or system? |
| T | Tampering | Can data be modified in transit or at rest without detection? |
| R | Repudiation | Can a user deny performing an action? |
| I | Information Disclosure | Can sensitive data leak to unauthorized parties? |
| D | Denial of Service | Can an attacker prevent legitimate users from accessing the service? |
| E | Elevation of Privilege | Can an attacker gain capabilities beyond what is authorized? |

## Threat Model Template

```markdown
# Threat Model: {Feature Name}

## Assets
| Asset | Sensitivity | Owner |
|-------|-------------|-------|
| User PII | High | {team} |
| Payment data | Critical | {team} |
| API keys | Critical | {team} |

## Trust Boundaries
- Internet <-> Load Balancer
- Load Balancer <-> Application
- Application <-> Database
- Application <-> Third-party APIs

## STRIDE Analysis

### {Component Name}

**S — Spoofing**
- Risk: {description}
- Likelihood: High/Medium/Low
- Impact: High/Medium/Low
- Mitigation: {control}

**T — Tampering**
...

**R — Repudiation**
...

**I — Information Disclosure**
...

**D — Denial of Service**
...

**E — Elevation of Privilege**
...

## Risk Register
| Threat | Component | Likelihood | Impact | Risk Level | Mitigation | Status |
|--------|-----------|------------|--------|------------|------------|--------|
| SQL Injection | API | High | Critical | Critical | Parameterized queries | Mitigated |

## Mitigations Required Before Launch
- [ ] {Critical mitigation 1}
- [ ] {Critical mitigation 2}
```

## Common Mitigations
- **Authentication:** JWT with short expiry, refresh token rotation.
- **Authorization:** RBAC, RLS on database.
- **Input validation:** Zod/FluentValidation on all inputs.
- **Rate limiting:** Per-IP and per-user limits.
- **Encryption:** TLS 1.3 in transit, AES-256 at rest.
- **Audit logging:** All write operations logged with actor + timestamp.
- **Secrets management:** Never in code; use environment variables or vault.
