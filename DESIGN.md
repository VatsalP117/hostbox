# Hostbox Design System

## Direction

Hostbox should feel like a focused deployment product: precise, quiet, fast, and infrastructure-first. The user-requested Vercel audit is reference context for common deployment-product interaction patterns, adapted around Hostbox's differentiator: a polished workflow on infrastructure the user owns.

This is an original Hostbox system informed by a broader audit of modern developer platforms. Hostbox keeps its own identity, product language, information architecture, and electric-blue operational accent. Reference products are used to understand interaction conventions, never to copy proprietary brand assets or signature compositions.

## Reference audit

General interaction patterns observed during the requested review of Vercel's public pages and signed-in dashboard (August 2026), recorded for research rather than reproduction:

- Near-black canvas with neutral gray surfaces; hierarchy comes from 1px borders more than shadows.
- Compact navigation, 14px product copy, and large but restrained marketing headlines.
- Simple product marks, restrained identities, and high-contrast actions are common across the category.
- Marketing compositions use thin grid lines, generous empty space, modular bordered panels, and occasional radial light.
- Dashboard layout uses a narrow fixed rail, a slim top context bar, command/search affordances, dense project cards, and low-radius controls.
- Cards remain mostly flat. Hover states lift contrast or border color rather than moving dramatically.
- Status is communicated by a small dot plus plain language; color is semantic and sparse.
- Tabs are underline/border-led. Forms are compact, with clear labels and visible focus rings.

## Product principles

1. **Infrastructure should look trustworthy.** Prefer stable geometry, crisp borders, and direct language.
2. **Density is a feature.** Dashboard screens optimize for scanning; marketing pages preserve breathing room.
3. **One strong action per view.** Primary actions are white on black in dark contexts and black on white in light contexts.
4. **Color reports state.** Blue is Hostbox's interactive accent; green, amber, and red are reserved for status.
5. **Own the machine.** Copy should reinforce control, portability, and the user's VM without leaning on vague “sovereignty” language.

## Foundations

### Color

| Token | Value | Use |
| --- | --- | --- |
| `canvas` | `#000000` | Marketing background, deepest dashboard areas |
| `background` | `#0a0a0a` | Dashboard canvas |
| `surface` | `#111111` | Cards, menus, sidebars |
| `surface-raised` | `#171717` | Hover and selected states |
| `border` | `#2a2a2a` | Default 1px rules |
| `border-subtle` | `#1f1f1f` | Section and grid rules |
| `text` | `#ededed` | Primary text |
| `text-muted` | `#a1a1a1` | Secondary copy |
| `text-faint` | `#707070` | Metadata and placeholders |
| `accent` | `#52a8ff` | Links, focus, active operational state |
| `success` | `#46a758` | Healthy/running |
| `warning` | `#f5a623` | Pending/attention |
| `danger` | `#e5484d` | Failure/destructive |

Avoid gradients on controls. A restrained radial glow or grid fade is acceptable only in marketing hero artwork.

### Typography

- UI and display: Geist-compatible system stack (`Geist`, `Inter`, `-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, sans-serif).
- Code: Geist Mono-compatible stack (`Geist Mono`, `SFMono-Regular`, `JetBrains Mono`, monospace).
- Dashboard body: 14px / 20px.
- Metadata: 12px / 16px.
- Dashboard page title: 24–32px, weight 600.
- Marketing display: `clamp(3rem, 7vw, 6rem)`, weight 600, tight tracking.
- Sentence case by default. Uppercase is limited to tiny technical labels and should not exceed `0.08em` tracking.

### Shape and depth

- Small control radius: 6px.
- Default card radius: 8px.
- Marketing feature panels: 12px maximum.
- Borders: 1px, neutral.
- Shadows: menus/dialogs only; never use colored glows on standard cards or buttons.

### Spacing

Use a 4px base grid. Common values: 8, 12, 16, 24, 32, 48, 64, 96. Dashboard content max-width is 1200px. Marketing content max-width is 1200px with 24px gutters.

## Components

### Brand mark

The Hostbox mark is an original open wireframe box/server form paired with the wordmark. Its divided faces suggest a deployable artifact, container, and owned machine. It uses Hostbox blue so the identity remains recognizable even when the rest of the interface is neutral. Do not use another platform's geometric mark, wordmark, or distinctive logo construction.

### Buttons

- Primary: white fill, black text, neutral border.
- Secondary: transparent/dark fill, light text, neutral border.
- Ghost: transparent with a subtle raised hover.
- Destructive: red only when the action is truly destructive.
- Heights: 32px small, 36px default, 40px large.

### Cards

Cards use `surface`, a `border`, and little or no shadow. Header/content/footer divisions may use 1px rules. Project cards show icon + name, production URL, latest commit/deployment, source, and time in a compact vertical rhythm.

### Navigation

- Marketing: 64px bar, wordmark left, product links center/left, two compact actions right.
- Dashboard: 240px rail on desktop, full-height and independently scrollable; mobile uses a sheet.
- Active sidebar item: raised surface and bright text, no accent stripe.
- Project tabs: horizontal border bar with a white active underline.

### Forms

Labels sit 8px above inputs. Inputs are 36px high with dark surfaces and neutral borders. Focus uses a visible white/blue ring. Validation belongs next to the field; destructive copy is concise.

### Status

Pair a 6–8px dot with text (`Ready`, `Building`, `Failed`, `Stopped`). Avoid fully tinted cards. Badges are for compact metadata, not primary status communication.

### Empty and loading states

Empty states are bordered panels with a small monochrome icon, a direct title, one sentence, and at most one action. Skeletons use quiet neutral blocks without glow.

## Marketing composition

- Hero: centered promise, short supporting sentence, primary + secondary CTA, and a product-shaped deployment preview.
- Use a subtle square grid as a spatial device, especially behind the hero.
- Feature storytelling is modular: bordered rows and asymmetrical 2/3-column panels, not floating colorful cards.
- Include concrete product proof: install command, git-to-production flow, framework support, resource footprint, SSL, rollbacks, and open-source ownership.
- Footer is a compact directory with product, resources, and project links.

## Dashboard composition

- Default landing screen prioritizes projects, recent deployment activity, and VM health.
- Titles are task-oriented (`Projects`, `Deployments`, `System`) rather than theatrical (`Command Center`).
- Each page has a compact header row and places controls close to the data they affect.
- Project details use breadcrumb/context, actions, a thin stats row, then tabs.
- Admin metrics use the same card system as projects. Operational state—not decoration—provides color.

## Responsive behavior

- Breakpoints: mobile under 768px, compact desktop 768–1024px, wide above 1024px.
- Marketing grids collapse to one column; headline sizes reduce without changing hierarchy.
- The dashboard rail becomes a sheet. Top actions remain reachable and filters wrap.
- Tables may scroll horizontally; primary identifiers stay in the first column.
- Touch targets are at least 40px on mobile even when desktop controls are denser.

## Accessibility and motion

- Maintain WCAG AA contrast for body copy and controls.
- Every interactive element needs a visible `:focus-visible` state.
- Never rely on color alone for status.
- Respect `prefers-reduced-motion`.
- Motion is 120–180ms, opacity/background/border based. Avoid scale-on-hover for routine controls.

## Voice

Short, confident, technical, and human.

- Prefer: “Deploy to your VM.”
- Prefer: “Ready in 42s.”
- Prefer: “Your code. Your server. One workflow.”
- Avoid: “Sovereign architect,” “command the cloud,” and inflated claims without proof.

## Implementation notes

- Keep the existing shadcn/Radix component foundation in `web/`; normalize its tokens and shared variants.
- Use Lucide icons in the React dashboard and lightweight inline SVG/CSS icons in Astro marketing.
- Do not add a second UI kit.
- New screens should use semantic tokens rather than hard-coded hex values.
