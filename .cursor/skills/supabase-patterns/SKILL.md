---
name: supabase-patterns
description: "Apply Supabase schema, RLS policies, and edge function patterns. Use when building on Supabase."
---

# SKILL: Supabase Patterns

## Local Development Workflow
```bash
# Start local Supabase stack
supabase start

# Apply migrations
supabase db push

# Generate TypeScript types
supabase gen types typescript --local > src/types/supabase.ts

# Deploy Edge Functions
supabase functions deploy {function-name}

# Stop
supabase stop
```

## Migration Pattern
```sql
-- supabase/migrations/{timestamp}_{description}.sql
-- Up migration (always additive in production)

CREATE TABLE IF NOT EXISTS public.{table_name} (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
  created_at timestamptz DEFAULT now() NOT NULL,
  updated_at timestamptz DEFAULT now() NOT NULL
);

-- Enable RLS ALWAYS
ALTER TABLE public.{table_name} ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "{table_name}_select_own"
  ON public.{table_name} FOR SELECT
  USING (auth.uid() = user_id);

CREATE POLICY "{table_name}_insert_own"
  ON public.{table_name} FOR INSERT
  WITH CHECK (auth.uid() = user_id);

CREATE POLICY "{table_name}_update_own"
  ON public.{table_name} FOR UPDATE
  USING (auth.uid() = user_id)
  WITH CHECK (auth.uid() = user_id);

CREATE POLICY "{table_name}_delete_own"
  ON public.{table_name} FOR DELETE
  USING (auth.uid() = user_id);
```

## Edge Function Pattern
```typescript
// supabase/functions/{name}/index.ts
import { serve } from 'https://deno.land/std@0.177.0/http/server.ts'
import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

serve(async (req: Request) => {
  const authHeader = req.headers.get('Authorization')
  if (!authHeader) {
    return new Response(JSON.stringify({ error: 'Unauthorized' }), {
      status: 401,
      headers: { 'Content-Type': 'application/json' }
    })
  }

  const supabase = createClient(
    Deno.env.get('SUPABASE_URL') ?? '',
    Deno.env.get('SUPABASE_ANON_KEY') ?? '',
    { global: { headers: { Authorization: authHeader } } }
  )

  // Never use service_role key in Edge Functions unless absolutely necessary
  const { data: { user }, error } = await supabase.auth.getUser()
  if (error || !user) {
    return new Response(JSON.stringify({ error: 'Unauthorized' }), { status: 401 })
  }

  // ... handler logic
})
```

## Rules
- Enable RLS on EVERY table with user data. No exceptions.
- Never expose `service_role` key to client code.
- Always use `supabase gen types typescript` for type safety.
- Migrations are append-only (never modify existing migration files).
- Test with `supabase start` before deploying.
