---
name: mermaid-diagrams
description: "Generate and validate Mermaid diagrams (component, sequence, ER, deployment, state machine) for features. Use when documenting architecture."
---

# SKILL: Mermaid Diagrams

## Required Diagrams Per Feature
1. Component diagram (`graph TB`)
2. Sequence diagram (`sequenceDiagram`)
3. ER diagram (`erDiagram`)
4. Deployment diagram (`graph TB` with deployment focus)
5. State machine (`stateDiagram-v2`) — where applicable

## Rules
- Maximum 30 nodes per diagram.
- All diagrams must be syntactically valid.
- Test diagrams in the Mermaid Live Editor before committing.
- Use descriptive node labels.

## Component Diagram
```mermaid
graph TB
    subgraph "Frontend"
        UI["React App"]
    end
    subgraph "Backend"
        API["REST API"]
        DB[("PostgreSQL")]
        Cache[("Redis")]
    end
    UI -->|"HTTPS"| API
    API --> DB
    API --> Cache
```

## Sequence Diagram
```mermaid
sequenceDiagram
    actor U as User
    participant F as Frontend
    participant A as API
    participant D as Database

    U->>F: Action
    F->>A: POST /api/resource
    A->>D: INSERT
    D-->>A: 201 Created
    A-->>F: {id, data}
    F-->>U: Success
```

## ER Diagram
```mermaid
erDiagram
    USER {
        uuid id PK
        string email UK
        timestamp created_at
    }
    ORDER {
        uuid id PK
        uuid user_id FK
        decimal total
        timestamp created_at
    }
    USER ||--o{ ORDER : "places"
```

## State Machine
```mermaid
stateDiagram-v2
    [*] --> Pending
    Pending --> Processing: payment received
    Processing --> Completed: fulfilled
    Processing --> Failed: error
    Failed --> Pending: retry
    Completed --> [*]
```

## Deployment Diagram
```mermaid
graph TB
    subgraph "Vercel Edge"
        FE["Next.js App"]
        EF["Edge Functions"]
    end
    subgraph "AWS"
        LB["Load Balancer"]
        APP["App Servers"]
        RDS[("RDS PostgreSQL")]
        EC["ElastiCache Redis"]
    end
    FE --> EF
    EF --> LB
    LB --> APP
    APP --> RDS
    APP --> EC
```
