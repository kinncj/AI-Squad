---
name: redis-patterns
description: "Apply Redis data structure, caching, and pub/sub patterns. Use when integrating Redis into a service."
---

# SKILL: Redis Patterns

## Key Design Convention
```
{app}:{domain}:{entity}:{id}
# Examples:
myapp:user:session:usr_123
myapp:product:cache:prod_456
myapp:rate:api:ip_1.2.3.4
myapp:queue:emails:pending
```

## Cache-Aside Pattern
```typescript
async function getUser(id: string): Promise<User> {
  const key = `myapp:user:data:${id}`;
  const ttl = 3600; // 1 hour

  // Try cache first
  const cached = await redis.get(key);
  if (cached) {
    return JSON.parse(cached) as User;
  }

  // Cache miss — fetch from DB
  const user = await db.users.findById(id);
  if (!user) throw new Error('User not found');

  // Store in cache with TTL
  await redis.setex(key, ttl, JSON.stringify(user));
  return user;
}

// Invalidate on update
async function updateUser(id: string, data: Partial<User>): Promise<User> {
  const user = await db.users.update(id, data);
  await redis.del(`myapp:user:data:${id}`);
  return user;
}
```

## Rate Limiting with Sorted Set
```typescript
async function rateLimit(ip: string, limit: number, windowSeconds: number): Promise<boolean> {
  const key = `myapp:rate:api:${ip}`;
  const now = Date.now();
  const windowStart = now - (windowSeconds * 1000);

  const pipeline = redis.pipeline();
  pipeline.zremrangebyscore(key, 0, windowStart);
  pipeline.zadd(key, now, `${now}`);
  pipeline.zcard(key);
  pipeline.expire(key, windowSeconds);

  const results = await pipeline.exec();
  const count = results?.[2]?.[1] as number;
  return count <= limit;
}
```

## Session Store
```typescript
// Using ioredis with connect-redis
import session from 'express-session';
import RedisStore from 'connect-redis';

app.use(session({
  store: new RedisStore({ client: redis }),
  secret: process.env.SESSION_SECRET!,
  resave: false,
  saveUninitialized: false,
  cookie: {
    secure: process.env.NODE_ENV === 'production',
    httpOnly: true,
    maxAge: 1000 * 60 * 60 * 24, // 24 hours
  }
}));
```

## Rules
- ALWAYS set TTLs on cache keys — never store indefinitely.
- Use SCAN not KEYS in production (non-blocking).
- Use pipelining for multiple operations.
- Key convention: `{app}:{domain}:{entity}:{id}`.
- Monitor with: `redis-cli info memory` and `redis-cli monitor` (dev only).
- Set `maxmemory-policy allkeys-lru` for cache-only Redis instances.
