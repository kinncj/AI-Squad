---
name: kubernetes-patterns
description: "Apply Kubernetes and Kustomize patterns for deployments, services, and overlays. Use when writing or reviewing k8s manifests."
---

# SKILL: Kubernetes Patterns

## Kustomize Structure
```
infra/k8s/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── patches/
    ├── staging/
    │   ├── kustomization.yaml
    │   └── patches/
    └── prod/
        ├── kustomization.yaml
        └── patches/
```

## Deployment with Required Fields
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {app-name}
  labels:
    app: {app-name}
spec:
  replicas: 2
  selector:
    matchLabels:
      app: {app-name}
  template:
    metadata:
      labels:
        app: {app-name}
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1001
      containers:
        - name: {app-name}
          image: {image}:{tag}
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
          readinessProbe:
            httpGet:
              path: /health
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /health
              port: 3000
            initialDelaySeconds: 15
            periodSeconds: 20
          startupProbe:
            httpGet:
              path: /health
              port: 3000
            failureThreshold: 30
            periodSeconds: 10
```

## PodDisruptionBudget
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {app-name}-pdb
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: {app-name}
```

## HPA
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {app-name}-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {app-name}
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

## Validation Workflow
```bash
# Always validate before applying
kubectl apply --dry-run=client -f {manifest}
kustomize build overlays/dev | kubectl apply --dry-run=client -f -
helm template {release} {chart} | kubectl apply --dry-run=client -f -
```
