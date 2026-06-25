---
name: postgresql-patterns
description: "Apply PostgreSQL schema, migration, and query patterns with idempotent SQL. Use when writing database schemas or migrations."
---

# SKILL: PostgreSQL Patterns

## Migration Naming
```
migrations/
  V001__create_users.sql          # Flyway
  0001_create_users.sql           # Generic
  20240115_create_users.sql       # Timestamp-based
```

## Table Pattern with RLS
```sql
-- Create table
CREATE TABLE IF NOT EXISTS users (
  id          uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  email       text UNIQUE NOT NULL,
  name        text NOT NULL,
  role        text NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
  created_at  timestamptz DEFAULT now() NOT NULL,
  updated_at  timestamptz DEFAULT now() NOT NULL
);

-- Indexes
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_users_role ON users(role) WHERE role != 'user';

-- Updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_select_own ON users
  FOR SELECT USING (id = current_user_id());

CREATE POLICY users_admin_all ON users
  FOR ALL USING (current_user_role() = 'admin');
```

## Query Optimization
```sql
-- Always EXPLAIN ANALYZE in development
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT u.*, COUNT(o.id) as order_count
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.created_at > now() - interval '30 days'
GROUP BY u.id;

-- Partial index for common filter
CREATE INDEX idx_orders_pending
  ON orders(created_at)
  WHERE status = 'pending';

-- GIN index for full-text search
CREATE INDEX idx_products_search
  ON products USING gin(to_tsvector('english', name || ' ' || description));
```

## Connection Pooling (PgBouncer)
```ini
# pgbouncer.ini
[databases]
app = host=127.0.0.1 port=5432 dbname=app

[pgbouncer]
pool_mode = transaction  # transaction pooling for most apps
max_client_conn = 100
default_pool_size = 20
```

## Rules
- NEVER modify existing migration files.
- Always create new migration for schema changes.
- Index all foreign keys.
- Use CONCURRENTLY for index creation on large tables.
- EXPLAIN ANALYZE all queries > 100ms.
- Enable RLS on tables with user data.
