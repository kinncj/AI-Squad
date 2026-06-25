---
name: finops-review
description: "Identify and annotate cloud cost implications of architectural decisions. Use when reviewing infrastructure changes or designing new services."
---

# SKILL: FinOps Review

## Purpose
Identify and annotate the cloud cost implications of architectural decisions.

## Cost Driver Categories

### Compute
- Lambda/Functions: invocation count x duration x memory
- EC2/VMs: instance type x hours + data transfer
- Containers: CPU/memory reservation x hours
- Edge functions: request count x execution time

### Storage
- Object storage (S3/GCS/Blob): GB stored + requests + egress
- Database: instance size + storage + IOPS + backups
- Cache: node size x hours

### Networking
- Egress: GB transferred out (most expensive)
- CDN: requests + GB served
- Load balancer: hours + LCU

### Third-party APIs
- Stripe: 2.9% + $0.30 per transaction
- Supabase: row reads/writes, storage, Edge Function invocations
- Vercel: bandwidth, function invocations, seats

## Annotation Format (Terraform)
```hcl
resource "aws_db_instance" "main" {
  # finops: ~$180/mo (db.t3.medium, 100GB gp3, 1 AZ)
  # finops: scale-trigger: >70% CPU or >80% storage
  instance_class = "db.t3.medium"
  ...
}
```

## Architecture Review Questions
1. What scales with user count? What scales with data volume?
2. Are there N+1 API calls that will cost $$ at scale?
3. Is egress minimized? (Cache aggressively, colocate services)
4. Are reserved instances/savings plans applicable?
5. What is the cost at 10x current load?

## Monthly Estimate Template
```markdown
## Cost Estimate: {Feature}

| Service | Config | Est. Monthly |
|---------|--------|-------------|
| RDS PostgreSQL | db.t3.medium, 100GB | $180 |
| ElastiCache Redis | cache.t3.micro | $25 |
| ECS Fargate | 2 tasks, 0.5 vCPU, 1GB | $30 |
| ALB | 1 ALB, ~1M requests | $20 |
| **Total** | | **~$255/mo** |

Scales linearly with: [traffic volume / data volume]
```
