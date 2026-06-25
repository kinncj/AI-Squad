---
name: stripe-patterns
description: "Apply Stripe API integration patterns: webhooks, idempotency keys, and subscription handling. Use when integrating Stripe payments."
---

# SKILL: Stripe Patterns

## Local Testing Setup
```bash
# Listen for webhooks and forward to local server
stripe listen --forward-to localhost:3000/api/webhooks/stripe

# Trigger test events
stripe trigger payment_intent.succeeded
stripe trigger checkout.session.completed
stripe trigger customer.subscription.created

# Monitor logs
stripe logs tail
```

## Webhook Handler Pattern
```typescript
// ALWAYS verify webhook signatures — never skip
import Stripe from 'stripe';

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!);

export async function POST(req: Request) {
  const body = await req.text();
  const sig = req.headers.get('stripe-signature')!;
  const webhookSecret = process.env.STRIPE_WEBHOOK_SECRET!;

  let event: Stripe.Event;
  try {
    event = stripe.webhooks.constructEvent(body, sig, webhookSecret);
  } catch (err) {
    return new Response(`Webhook Error: ${err.message}`, { status: 400 });
  }

  // Idempotency: check if event already processed
  // Store event.id in database, skip if duplicate

  switch (event.type) {
    case 'payment_intent.succeeded': {
      const paymentIntent = event.data.object as Stripe.PaymentIntent;
      // Handle success
      break;
    }
    case 'checkout.session.completed': {
      const session = event.data.object as Stripe.Checkout.Session;
      // Provision access
      break;
    }
    default:
      console.log(`Unhandled event type: ${event.type}`);
  }

  return new Response(JSON.stringify({ received: true }), { status: 200 });
}
```

## Checkout Session Pattern
```typescript
const session = await stripe.checkout.sessions.create({
  mode: 'subscription',
  payment_method_types: ['card'],
  line_items: [{
    price: priceId,
    quantity: 1,
  }],
  success_url: `${baseUrl}/success?session_id={CHECKOUT_SESSION_ID}`,
  cancel_url: `${baseUrl}/cancel`,
  customer_email: user.email,
  metadata: {
    userId: user.id,  // link back to your user
  },
}, {
  idempotencyKey: `checkout-${user.id}-${priceId}`,  // ALWAYS use idempotency keys
});
```

## Rules
- ALWAYS verify webhook signatures with `constructEvent()`.
- ALWAYS use idempotency keys on write operations.
- NEVER log raw card data or full payment method details.
- Use restricted API keys with minimum required permissions per service.
- Test every webhook handler with `stripe trigger`.
- Store Stripe customer IDs in your database for lookup.
