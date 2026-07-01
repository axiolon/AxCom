---
title: "Doc Status Badges"
description: "How to add version and review-status badges to documentation pages."
sidebar_position: 1
---

# Doc Status Badges

<DocBadge status="stable" version="v0.1.0-alpha" />

Every documentation page carries a `<DocBadge>` to communicate two things at a glance:

1. **Status** — is this doc accurate and reviewed, or still in progress?
2. **Version** — which release does this doc target?

---

## Usage

Place `<DocBadge>` immediately after the page's `# Title` heading (or after the frontmatter `---` if the page has no H1):

```mdx
# My Page Title

<DocBadge status="under-review" version="v0.1.0-alpha" />

First paragraph...
```

No import is needed — the component is registered globally.

---

## Props

| Prop | Type | Description |
|---|---|---|
| `status` | `string` | Stability / review state (see values below) |
| `version` | `string` | Release version this doc targets, e.g. `v0.1.0-alpha` |
| `reviewedAt` | `string` | Last reviewed against, e.g. `2026-06` |

All props are optional. You can use any combination.

---

## Status Values

| Value | Badge | Meaning |
|---|---|---|
| `stable` | green | Accurate, reviewed, matches the current release |
| `under-review` | amber | Content is present but not yet verified against the current codebase |
| `experimental` | orange | Documents a feature that is subject to breaking changes |
| `beta` | blue | Feature is usable but the API or behaviour may still change |
| `deprecated` | gray | Feature is being phased out; doc retained for reference |

---

## Examples

```mdx
<DocBadge status="stable" version="v0.2.0" />

<DocBadge status="under-review" version="v0.1.0-alpha" reviewedAt="2026-06" />

<DocBadge status="experimental" version="v0.2.0-beta" />

<DocBadge status="deprecated" />
```

---

## Frontmatter Convention

Add matching `status` and `version` fields to the page frontmatter. They are not rendered by Docusaurus today, but they make docs machine-queryable for future tooling (e.g., a CI check that flags stale docs):

```yaml
---
title: "My Page"
description: "..."
status: under-review
version: v0.1.0-alpha
---
```

---

## Updating a Badge

When you update a doc to reflect the current codebase:

1. Change `status` from `under-review` to `stable` (or the appropriate state).
2. Update `version` to match the release the doc now targets.
3. Optionally add `reviewedAt` with the current month.

```mdx
<DocBadge status="stable" version="v0.2.0" reviewedAt="2026-06" />
```
