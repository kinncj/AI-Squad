---
name: humanizer
description: Remove AI-isms and artificial language patterns from text. Makes documentation, comments, commit messages, and prose sound more natural and human. Based on Wikipedia's "Signs of AI writing" patterns.
---

# humanizer skill

Polish text by removing detectable AI-generated writing patterns. Ideal for finalizing documentation, commit messages, comments, and any prose before merging.

## When to use

- After `@docs` generates feature documentation
- Before committing code (check commit messages and comments)
- Polishing README updates, CHANGELOG entries, or runbooks
- Making AI-assisted prose sound more natural
- Manual voice calibration for brand consistency

## How to invoke

```
/humanizer

[paste your text here]
```

Or ask directly in chat:
```
Please humanize this text: [your text]
```

## Voice calibration (optional)

Provide a sample of your own writing for the skill to match your style:

```
/humanizer

Here's a sample of my writing for voice matching:
[paste 2-3 paragraphs of your own writing]

Now humanize this text:
[paste AI text to humanize]
```

## What it detects

The skill checks for 29 AI-writing patterns, including:

- **Significance inflation** — "marking a pivotal moment in the evolution of..." → "was established in 1989"
- **Notability name-dropping** — Lists of citations → specific examples with context
- **Superficial -ing phrases** — "symbolizing, reflecting, showcasing" → active explanations
- **Promotional language** — Corporate jargon → straightforward description
- **Overuse of 'indeed'** — Filler words → tighter prose
- **Transition abuse** — Excessive "Furthermore, In conclusion..." → natural flow
- **Hedging redundancy** — "arguably, in some sense, it could be argued..." → clarity
- **Rare word stacking** — Thesaurus abuse → common vocabulary
- **False expert mode** — Generic expertise → grounded examples
- **Generalist fluff** — "many fields, various domains..." → specific scope

## Integration with MAPLE

- Auto-call after Phase 7 (DOCUMENT) if `humanize: true` in feature story frontmatter
- Manual: invoke at any phase before merge
- Works across all harnesses (Claude Code, OpenCode, Cursor, Copilot CLI)

## Output

Returns a humanized version of your text with:
1. First pass: Remove identified AI patterns
2. Final audit: "Obviously AI generated?" check
3. Second rewrite: Catch lingering AI-isms

## Further reading

- [Wikipedia: Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing)
- [Humanizer repo](https://github.com/blader/humanizer)
