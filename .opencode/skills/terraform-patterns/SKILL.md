---
name: terraform-patterns
description: "Apply Terraform module structure, state management, and variable patterns for infrastructure as code. Use when writing Terraform."
---

# SKILL: Terraform Patterns

## Module Structure
```
infra/terraform/
├── modules/
│   ├── networking/
│   ├── database/
│   └── compute/
└── environments/
    ├── dev/
    │   ├── main.tf
    │   ├── variables.tf
    │   └── terraform.tfvars
    ├── staging/
    └── prod/
```

## Module Pattern
```hcl
# modules/database/main.tf
resource "aws_db_instance" "main" {
  # finops: ~$180/mo (db.t3.medium, 100GB gp3, single AZ)
  identifier     = "${var.environment}-${var.name}-db"
  engine         = "postgres"
  engine_version = "16"
  instance_class = var.instance_class

  allocated_storage     = var.storage_gb
  storage_type          = "gp3"
  storage_encrypted     = true

  username = var.username
  password = var.password

  vpc_security_group_ids = [aws_security_group.db.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name

  backup_retention_period = var.environment == "prod" ? 7 : 1
  deletion_protection     = var.environment == "prod"

  tags = merge(var.tags, {
    Environment = var.environment
    ManagedBy   = "terraform"
  })
}
```

## Workflow
```bash
# Initialize (first time or after provider changes)
terraform init

# Format check (CI)
terraform fmt -check -recursive

# Validate syntax
terraform validate

# Plan (always before apply)
terraform plan -out=plan.tfplan -var-file=environments/dev/terraform.tfvars

# Apply (requires explicit instruction)
terraform apply plan.tfplan

# State inspection
terraform show
terraform state list
```

## Rules
- ALWAYS run `terraform validate` before reporting complete.
- ALWAYS run `terraform plan` to show what will change.
- NEVER run `terraform apply` without explicit instruction.
- ALWAYS annotate resources with `# finops:` cost estimate.
- Use workspaces or separate state files per environment.
- Enable state locking (S3 + DynamoDB or Terraform Cloud).
- Tag all resources with environment, managed-by, and cost-center.
