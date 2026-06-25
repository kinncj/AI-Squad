---
name: vercel-patterns
description: "Apply Vercel deployment, edge function, and environment variable patterns. Use when deploying to Vercel."
---

# SKILL: Vercel Patterns

## vercel.json Configuration
```json
{
  "framework": "nextjs",
  "buildCommand": "npm run build",
  "devCommand": "npm run dev",
  "installCommand": "npm ci",
  "regions": ["iad1"],
  "headers": [
    {
      "source": "/api/(.*)",
      "headers": [
        { "key": "X-Content-Type-Options", "value": "nosniff" },
        { "key": "X-Frame-Options", "value": "DENY" },
        { "key": "Strict-Transport-Security", "value": "max-age=31536000" }
      ]
    }
  ],
  "rewrites": [],
  "redirects": [
    {
      "source": "/old-path",
      "destination": "/new-path",
      "permanent": true
    }
  ]
}
```

## Runtime Selection
```typescript
// Edge Runtime (low-latency, streaming, middleware)
export const runtime = 'edge';

// Node.js Runtime (heavy compute, native modules, file system)
export const runtime = 'nodejs';

// ISR (static with revalidation)
export const revalidate = 3600; // seconds
```

## Environment Variables
```bash
# Pull env vars for local development
vercel env pull .env.local

# Add env var
vercel env add SECRET_KEY production

# List env vars
vercel env ls
```

## Deployment Workflow
```bash
# Build locally first (catch errors before deploy)
vercel build

# Deploy prebuilt (faster, recommended for CI)
vercel deploy --prebuilt

# Deploy to production
vercel deploy --prod --prebuilt
```

## Middleware Pattern
```typescript
// middleware.ts (runs at edge, before every request)
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  // Auth check, rate limiting, A/B testing, etc.
  const token = request.cookies.get('token');
  if (!token && request.nextUrl.pathname.startsWith('/dashboard')) {
    return NextResponse.redirect(new URL('/login', request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ['/dashboard/:path*', '/api/protected/:path*'],
};
```

## Rules
- Run `vercel build` locally before deploying to catch errors.
- Use `vercel env pull` to sync environment variables locally.
- Prefer Edge runtime for middleware and simple API routes.
- Use Node.js runtime for heavy compute or native module requirements.
- Set security headers for all API routes.
