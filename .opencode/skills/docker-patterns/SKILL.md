---
name: docker-patterns
description: "Apply Docker and Docker Compose best practices for containerising services. Use when writing or reviewing Dockerfiles and compose configs."
---

# SKILL: Docker Patterns

## Multi-Stage Dockerfile Pattern
```dockerfile
# syntax=docker/dockerfile:1
FROM node:22-alpine AS base
WORKDIR /app

FROM base AS deps
COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --only=production

FROM base AS dev-deps
COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

FROM base AS build
COPY --from=dev-deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM gcr.io/distroless/nodejs22-debian12 AS production
WORKDIR /app
USER 1001
COPY --from=deps /app/node_modules ./node_modules
COPY --from=build /app/dist ./dist
EXPOSE 3000
CMD ["dist/main.js"]
```

## docker-compose Health Check Pattern
```yaml
services:
  app:
    build:
      context: .
      target: production
    ports:
      - "3000:3000"
    environment:
      DATABASE_URL: postgres://postgres:pass@db:5432/app
    depends_on:
      db:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 5s
      timeout: 5s
      retries: 5
      start_period: 10s

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: pass
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 2s
      timeout: 5s
      retries: 10
```

## .dockerignore
```
node_modules
.git
.github
dist
build
coverage
.next
*.log
.env*
!.env.example
README.md
docs
tests
```

## Key Rules
- ALWAYS use multi-stage builds for production images.
- ALWAYS run as non-root user (USER 1001 or named user).
- ALWAYS add health checks to all services.
- NEVER copy .env files into images.
- Use `--mount=type=cache` for package manager caches.
- Use distroless or alpine base images.
- Use `docker compose up -d --wait` to wait for health checks.

## Verification
```bash
# Build test
docker build --target production -t test-image .

# Security scan
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy:latest image test-image

# Start and verify health
docker compose up -d --wait
docker compose ps
```
