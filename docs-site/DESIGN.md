---
version: 1.0.0
product: OntoGraph Console
description: "A near-black ontology workbench canvas anchored at #010102 (the deepest dark surface available), with light gray reading text (#f7f8f8) and a single lavender-blue chromatic accent (#5e6ad2) — Linear-grade restraint rebuilt for an admin tooling surface. The product reads as a precision instrument: dense data tables, keyboard-driven inspection, graph visualization, and editor panels dominate the chrome. Display type is set in a tight sans (Inter / SF Pro Display fallback) at 500–600 with measured negative tracking. Cards live as charcoal panels (#0f1011) with hairline 1px borders (#23252a). Lavender-blue is reserved for active selection, focus rings, brand mark, and primary CTAs — never decorative. The composition philosophy is 'frame the data' — surfaces stay quiet so graph nodes, type tables, and editor cursors do the visual work."

tokens:
  colors:
    # Brand
    primary: "#5e6ad2"          # Lavender-blue (workbench accent)
    primary-hover: "#828fff"    # Lavender hover
    primary-active: "#4a55b8"   # Lavender pressed
    primary-focus: "#5e69d1"    # Focus-ring tint
    on-primary: "#ffffff"       # Text on primary
    brand-secure: "#7a7fad"     # Muted lavender for security surfaces

    # Surface ladder
    canvas: "#010102"           # Page background (deepest dark, faint blue tint)
    surface-1: "#0f1011"        # Card surface (default lift)
    surface-2: "#141516"        # Featured card / hovered card
    surface-3: "#18191a"        # Sub-nav, dropdown menu
    surface-4: "#191a1b"        # Deepest lifted surface (modals, overlays)
    surface-hairline: "#23252a" # 1px card border
    surface-hairline-strong: "#34343a" # Stronger border / focus
    surface-hairline-tertiary: "#3e3e44" # Nested surfaces, input borders

    # Inverse (only used for "Schedule a demo" / day-mode audit panels)
    inverse-canvas: "#ffffff"
    inverse-surface-1: "#f5f6f6"
    inverse-surface-2: "#f6f7f7"
    inverse-ink: "#000000"

    # Text
    ink: "#f7f8f8"              # Body & emphasis
    ink-muted: "#d0d6e0"        # Secondary text
    ink-subtle: "#8a8f98"       # Tertiary text (deselected, footer)
    ink-tertiary: "#62666d"     # Disabled / placeholder
    ink-inverse: "#000000"      # Text on inverse surfaces

    # Semantic (kept restrained)
    semantic-success: "#27a644" # Success pills, ingest complete
    semantic-warning: "#f2c94c" # Caution, validation pending
    semantic-error: "#eb5757"   # Failure, validation rejected
    semantic-info: "#56ccf2"    # Informational status
    semantic-overlay: "#000000" # Modal scrim

  typography:
    display-xl:
      fontFamily: Inter Display
      fontSize: 56px
      fontWeight: 600
      lineHeight: 1.05
      letterSpacing: -2.4px
    display-lg:
      fontFamily: Inter Display
      fontSize: 44px
      fontWeight: 600
      lineHeight: 1.10
      letterSpacing: -1.4px
    display-md:
      fontFamily: Inter Display
      fontSize: 32px
      fontWeight: 600
      lineHeight: 1.15
      letterSpacing: -0.8px
    headline:
      fontFamily: Inter Display
      fontSize: 24px
      fontWeight: 600
      lineHeight: 1.25
      letterSpacing: -0.4px
    card-title:
      fontFamily: Inter Display
      fontSize: 20px
      fontWeight: 600
      lineHeight: 1.30
      letterSpacing: -0.3px
    subhead:
      fontFamily: Inter
      fontSize: 18px
      fontWeight: 500
      lineHeight: 1.40
      letterSpacing: -0.2px
    body-lg:
      fontFamily: Inter
      fontSize: 16px
      fontWeight: 400
      lineHeight: 1.55
      letterSpacing: -0.1px
    body:
      fontFamily: Inter
      fontSize: 14px
      fontWeight: 400
      lineHeight: 1.55
      letterSpacing: -0.05px
    body-sm:
      fontFamily: Inter
      fontSize: 13px
      fontWeight: 400
      lineHeight: 1.55
      letterSpacing: 0
    caption:
      fontFamily: Inter
      fontSize: 12px
      fontWeight: 400
      lineHeight: 1.45
      letterSpacing: 0
    button:
      fontFamily: Inter
      fontSize: 13px
      fontWeight: 500
      lineHeight: 1.20
      letterSpacing: 0
    eyebrow:
      fontFamily: Inter
      fontSize: 12px
      fontWeight: 600
      lineHeight: 1.30
      letterSpacing: 0.6px
    mono:
      fontFamily: JetBrains Mono
      fontSize: 12px
      fontWeight: 400
      lineHeight: 1.55
      letterSpacing: 0

  shapes:
    radius-xs: 4px             # Chips, status pills, dense controls
    radius-sm: 6px             # Inline tags
    radius-md: 8px             # Buttons, form inputs, popovers
    radius-lg: 12px            # Cards, panels, dropdowns
    radius-xl: 16px            # Hero containers, large panels
    radius-xxl: 24px           # Oversized surfaces (rare)
    radius-pill: 9999px        # Tab toggles, segmented controls
    radius-full: 9999px        # Avatars, circular affordances

  spacing:
    xxs: 4px
    xs: 8px
    sm: 12px
    md: 16px
    lg: 24px
    xl: 32px
    xxl: 48px
    section: 96px

  elevation:
    flat: none
    lift-1: "0 1px 0 0 {colors.surface-hairline}"      # hairline panel
    lift-2: "0 0 0 1px {colors.surface-hairline-strong}, 0 8px 24px rgba(0,0,0,0.4)"  # modal
    lift-3: "0 16px 48px rgba(0,0,0,0.6)"               # floating menu
    focus-ring: "0 0 0 2px {colors.primary-focus}55"   # 2px focus halo

  breakpoints:
    sm: 640px
    md: 768px
    lg: 1024px
    xl: 1280px
    xxl: 1536px

components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.button}"
    shape: "{shapes.radius-md}"
    padding: "8px 14px"
    border: 1px solid {colors.primary}
    focus: "{elevation.focus-ring}"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
    shape: "{shapes.radius-md}"
  button-primary-pressed:
    backgroundColor: "{colors.primary-active}"
    shape: "{shapes.radius-md}"
  button-secondary:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink}"
    typography: "{typography.button}"
    shape: "{shapes.radius-md}"
    padding: "8px 14px"
    border: 1px solid {colors.surface-hairline}
  button-tertiary:
    backgroundColor: transparent
    textColor: "{colors.ink-subtle}"
    typography: "{typography.button}"
    shape: "{shapes.radius-md}"
    padding: "8px 14px"
    border: 1px solid transparent
    hover:
      textColor: "{colors.ink}"
      backgroundColor: "{colors.surface-1}"
  button-danger:
    backgroundColor: transparent
    textColor: "{colors.semantic-error}"
    typography: "{typography.button}"
    shape: "{shapes.radius-md}"
    padding: "8px 14px"
    border: 1px solid {colors.surface-hairline}
    hover:
      backgroundColor: "{colors.surface-1}"
  card-default:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    shape: "{shapes.radius-lg}"
    padding: "{spacing.lg}"
    border: 1px solid {colors.surface-hairline}
  card-featured:
    backgroundColor: "{colors.surface-2}"
    shape: "{shapes.radius-lg}"
    padding: "{spacing.lg}"
    border: 1px solid {colors.surface-hairline-strong}
  card-dense:
    backgroundColor: "{colors.surface-1}"
    shape: "{shapes.radius-md}"
    padding: "{spacing.md}"
    border: 1px solid {colors.surface-hairline}
  text-input:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    shape: "{shapes.radius-md}"
    padding: "8px 12px"
    border: 1px solid {colors.surface-hairline}
    placeholderColor: "{colors.ink-tertiary}"
    focus:
      borderColor: "{colors.primary-focus}"
      ring: "{elevation.focus-ring}"
  text-area:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    shape: "{shapes.radius-md}"
    padding: "{spacing.sm} {spacing.md}"
    border: 1px solid {colors.surface-hairline}
    minHeight: 96px
  select-trigger:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    shape: "{shapes.radius-md}"
    padding: "8px 12px"
    border: 1px solid {colors.surface-hairline}
  dropdown-menu:
    backgroundColor: "{colors.surface-3}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    shape: "{shapes.radius-lg}"
    padding: "{spacing.xs}"
    border: 1px solid {colors.surface-hairline-strong}
    shadow: "{elevation.lift-3}"
  menu-item-hover:
    backgroundColor: "{colors.surface-2}"
    shape: "{shapes.radius-sm}"
  segmented-control-default:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink-subtle}"
    typography: "{typography.button}"
    shape: "{shapes.radius-pill}"
    padding: "6px 12px"
  segmented-control-active:
    backgroundColor: "{colors.surface-3}"
    textColor: "{colors.ink}"
    shape: "{shapes.radius-pill}"
    padding: "6px 12px"
    border: 1px solid {colors.surface-hairline-strong}
  status-badge:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.ink-muted}"
    typography: "{typography.caption}"
    shape: "{shapes.radius-pill}"
    padding: "2px 10px"
    border: 1px solid {colors.surface-hairline}
  status-badge-success:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.semantic-success}"
    typography: "{typography.caption}"
    shape: "{shapes.radius-pill}"
    padding: "2px 10px"
    border: 1px solid {colors.surface-hairline}
  status-badge-error:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.semantic-error}"
    typography: "{typography.caption}"
    shape: "{shapes.radius-pill}"
    padding: "2px 10px"
    border: 1px solid {colors.surface-hairline}
  status-badge-warning:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.semantic-warning}"
    typography: "{typography.caption}"
    shape: "{shapes.radius-pill}"
    padding: "2px 10px"
    border: 1px solid {colors.surface-hairline}
  table-row-hover:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
  table-row-selected:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.ink}"
    borderLeft: 2px solid {colors.primary}
  table-header:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink-subtle}"
    typography: "{typography.eyebrow}"
    padding: "{spacing.sm} {spacing.md}"
    borderBottom: 1px solid {colors.surface-hairline}
    textTransform: uppercase
  modal-backdrop:
    backgroundColor: rgba(0, 0, 0, 0.6)
  modal-panel:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    shape: "{shapes.radius-lg}"
    padding: "{spacing.xl}"
    border: 1px solid {colors.surface-hairline-strong}
    shadow: "{elevation.lift-2}"
  drawer-panel:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    shape: 0
    padding: "{spacing.xl}"
    border: 1px solid {colors.surface-hairline}
    width: 480px
  header-bar:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.body-sm}"
    height: 56px
    borderBottom: 1px solid {colors.surface-hairline}
    padding: 0 {spacing.lg}
  sidebar-nav:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink-muted}"
    typography: "{typography.body-sm}"
    width: 240px
    padding: "{spacing.md}"
    borderRight: 1px solid {colors.surface-hairline}
  sidebar-item-active:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink}"
    borderLeft: 2px solid {colors.primary}
    shape: 0
  content-area:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    padding: "{spacing.xl}"
  panel-tabs:
    backgroundColor: "{colors.surface-1}"
    textColor: "{colors.ink-subtle}"
    typography: "{typography.button}"
    shape: 0
    padding: "{spacing.md} {spacing.lg}"
    borderBottom: 1px solid {colors.surface-hairline}
  panel-tab-active:
    textColor: "{colors.ink}"
    borderBottom: 2px solid {colors.primary}
  keyboard-hint:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.ink-subtle}"
    typography: "{typography.mono}"
    shape: "{shapes.radius-xs}"
    padding: "2px 6px"
    border: 1px solid {colors.surface-hairline}
  empty-state:
    backgroundColor: transparent
    textColor: "{colors.ink-subtle}"
    typography: "{typography.body}"
    padding: "{spacing.xxl}"
    align: center
  graph-canvas:
    backgroundColor: "{colors.canvas}"
    border: 1px solid {colors.surface-hairline}
    shape: "{shapes.radius-lg}"
  tooltip:
    backgroundColor: "{colors.surface-4}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    shape: "{shapes.radius-sm}"
    padding: "{spacing.xs} {spacing.sm}"
    border: 1px solid {colors.surface-hairline-strong}
    shadow: "{elevation.lift-3}"
---

# OntoGraph Console — Design System

## 1. Visual Theme & Atmosphere

OntoGraph Console is the **deepest-dark ontology workbench canvas** in the Linear tradition: `#010102` (with a faint blue undertone) anchors every surface. The reading field is light gray `#f7f8f8`, the chromatic accent is a single lavender-blue `#5e6ad2`. There is no second bright accent, no atmospheric gradient, no spotlight card.

**Density & philosophy** — *Precision instrument*. Operators spend hours examining node graphs, class tables, and ingestion pipelines. The chrome recedes so that:

- Graph nodes, edge types, and editor cursors can do the visual work.
- The eye moves down columns and across rows without decorative interruption.
- Lavender-blue is **sparingly** spent only on selection, focus, the brand mark, and one primary CTA per page — never decoration.

**Page rhythm** — *Frame the data*. Product panels (graph canvas, ontology class table, episode explorer, version diff viewer, ingestion timeline) live in `{colors.surface-1}` panels with 1px hairline borders. Cards never use shadow on dark; depth is carried by the four-step surface ladder.

**Mood** — quiet, technical, slightly luxurious — like software-craft documentation. The same restraint Linear applies to its marketing canvas, OntoGraph applies to the admin / power-user surface.

> Key characteristics:
> - `{colors.canvas}` #010102 is the deepest dark. Do not use true `#000000`.
> - Lavender-blue (`{colors.primary}`) is reserved for active selection, brand mark, focus rings, and the primary CTA.
> - Four-step surface ladder (`surface-1` → `surface-4`) carries hierarchy in lieu of shadow.
> - Display tracking is aggressively negative (−2.4px at 56px); body holds at −0.05px.
> - Cards use `{shapes.radius-lg}` 12px corners with 1px hairline borders — never pill, rarely 16px.
> - Graphs, tables, and editor panels are the protagonist of every screen.

---

## 2. Color Palette & Roles

### Brand & Accent

| Token | Hex | Functional Role |
|---|---|---|
| `{colors.primary}` | `#5e6ad2` | Lavender-blue — active selection, brand mark, primary CTA, link emphasis |
| `{colors.primary-hover}` | `#828fff` | Hovered lavender (lightened) |
| `{colors.primary-active}` | `#4a55b8` | Pressed lavender (darkened) |
| `{colors.primary-focus}` | `#5e69d1` | Focus-ring tint (focused input, focused button) |
| `{colors.brand-secure}` | `#7a7fad` | Muted lavender for security / audit surfaces |

### Surface Ladder

| Token | Hex | Use |
|---|---|---|
| `{colors.canvas}` | `#010102` | Page background — faint blue tint, never `#000000` |
| `{colors.surface-1}` | `#0f1011` | Default card lift (class cards, ontology panels) |
| `{colors.surface-2}` | `#141516` | Featured card, hovered card, modal surface |
| `{colors.surface-3}` | `#18191a` | Sub-nav, dropdown menus, segmented track |
| `{colors.surface-4}` | `#191a1b` | Deepest lift (floating tooltip, drawer) |
| `{colors.surface-hairline}` | `#23252a` | 1px borders, dividers |
| `{colors.surface-hairline-strong}` | `#34343a` | Stronger borders, modal outline, focus |
| `{colors.surface-hairline-tertiary}` | `#3e3e44` | Nested surface borders, table grid |

### Text (Ink)

| Token | Hex | Use |
|---|---|---|
| `{colors.ink}` | `#f7f8f8` | Primary body & headings |
| `{colors.ink-muted}` | `#d0d6e0` | Secondary text, table values |
| `{colors.ink-subtle}` | `#8a8f98` | Tertiary text, captions, deselected tabs |
| `{colors.ink-tertiary}` | `#62666d` | Disabled state, placeholders, footnotes |

### Inverse (rare — only for audit panels in light context)

| Token | Hex | Use |
|---|---|---|
| `{colors.inverse-canvas}` | `#ffffff` | Light-mode audit panel |
| `{colors.inverse-surface-1}` | `#f5f6f6` | Light-mode surface step 1 |
| `{colors.inverse-ink}` | `#000000` | Text on inverse surface |

### Semantic (kept restrained — never decorative)

| Token | Hex | Use |
|---|---|---|
| `{colors.semantic-success}` | `#27a644` | Ingest succeeded, validation accepted |
| `{colors.semantic-warning}` | `#f2c94c` | Validation pending, deprecated type |
| `{colors.semantic-error}` | `#eb5757` | Validation failed, ingest error |
| `{colors.semantic-info}` | `#56ccf2` | Informational status (rare) |
| `{colors.semantic-overlay}` | `#000000` | Modal scrim (`rgba(0,0,0,0.6)`) |

---

## 3. Typography Rules

### Font Stacks

- **`Inter Display`** — Display cut, tight tracking, weights 500–700. Fallback `SF Pro Display, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto`. Used for `display-xl` → `card-title`.
- **`Inter`** — Text cut, tuned for body sizes. Same fallback. Used for `subhead` → `caption`.
- **`JetBrains Mono`** — Mono stack, weights 400 / 500. Fallback `ui-monospace, SF Mono, Menlo, Consolas`. Used for keyboard hints, IDs, raw JSON, CSPAN-style type identifiers.

> Why Inter / JetBrains Mono: both are open, render cleanly on dark, and approach Linear Display / Linear Mono in optical weight. Their tracking scales match OntoGraph's negative-letter-spacing display ladder.

### Hierarchy Table

| Token | Size | Weight | Line Height | Letter Spacing | Use |
|---|---|---|---|---|---|
| `{typography.display-xl}` | 56px | 600 | 1.05 | −2.4px | Empty-state hero, billing hero |
| `{typography.display-lg}` | 44px | 600 | 1.10 | −1.4px | Dashboard headlines |
| `{typography.display-md}` | 32px | 600 | 1.15 | −0.8px | Section openers, onboarding |
| `{typography.headline}` | 24px | 600 | 1.25 | −0.4px | Card titles, modal titles |
| `{typography.card-title}` | 20px | 600 | 1.30 | −0.3px | Ontology class card titles |
| `{typography.subhead}` | 18px | 500 | 1.40 | −0.2px | Lead paragraphs, panel intros |
| `{typography.body-lg}` | 16px | 400 | 1.55 | −0.1px | Tutorial copy |
| `{typography.body}` | 14px | 400 | 1.55 | −0.05px | **Default body** |
| `{typography.body-sm}` | 13px | 400 | 1.55 | 0 | Table cells, footer, sidebar nav |
| `{typography.caption}` | 12px | 400 | 1.45 | 0 | Metadata, timestamps, hints |
| `{typography.button}` | 13px | 500 | 1.20 | 0 | All button labels |
| `{typography.eyebrow}` | 12px | 600 | 1.30 | +0.6px | Section eyebrows, table headers (uppercase) |
| `{typography.mono}` | 12px | 400 | 1.55 | 0 | Keyboard hints, IDs, JSON snippets |

### Principles

1. **Aggressive negative tracking on display** (−2.4px at 56px ≈ 4% of size).
2. **Single voice from display to body.** `display-xl` at 600 → `body` at 400 — same family, narrower weights.
3. **Eyebrow uses positive tracking** (+0.6px) and uppercase to mark taxonomy — distinct from display's negative tracking.
4. **Mono stays in code contexts.** JetBrains Mono lives inside keyboard hints, IDs, version labels — never on marketing chrome.
5. **Default body is `{typography.body}` 14px / 400.** Never ship 16px as the default — workbench density wants more rows visible.

---

## 4. Component Stylings

### Buttons

| Variant | Background | Text | Border | Hover | Pressed |
|---|---|---|---|---|---|
| `button-primary` | `{colors.primary}` | `{colors.on-primary}` | 1px solid `{colors.primary}` | `{colors.primary-hover}` | `{colors.primary-active}` |
| `button-secondary` | `{colors.surface-1}` | `{colors.ink}` | 1px `{colors.surface-hairline}` | `{colors.surface-2}` | `{colors.surface-2}` |
| `button-tertiary` | transparent | `{colors.ink-subtle}` | transparent | text→ink + bg→surface-1 | text→ink |
| `button-danger` | transparent | `{colors.semantic-error}` | 1px `{colors.surface-hairline}` | bg→surface-1 | bg→surface-2 |

- All buttons: `{shapes.radius-md}` (8px), padding `8px 14px`, `{typography.button}`.
- Focus ring: `{elevation.focus-ring}` (2px halo at 55% opacity of `{colors.primary-focus}`).
- Disabled: opacity 0.4, no pointer, no transition.

### Cards & Containers

| Component | Surface | Border | Radius | Padding |
|---|---|---|---|---|
| `card-default` | `{colors.surface-1}` | 1px `{colors.surface-hairline}` | `{shapes.radius-lg}` | `{spacing.lg}` |
| `card-featured` | `{colors.surface-2}` | 1px `{colors.surface-hairline-strong}` | `{shapes.radius-lg}` | `{spacing.lg}` |
| `card-dense` | `{colors.surface-1}` | 1px `{colors.surface-hairline}` | `{shapes.radius-md}` | `{spacing.md}` |
| `graph-canvas` | `{colors.canvas}` | 1px `{colors.surface-hairline}` | `{shapes.radius-lg}` | 0 |
| `modal-panel` | `{colors.surface-2}` | 1px `{colors.surface-hairline-strong}` | `{shapes.radius-lg}` | `{spacing.xl}` |
| `drawer-panel` | `{colors.surface-1}` | 1px `{colors.surface-hairline}` (right edge) | 0 | `{spacing.xl}` |

### Inputs & Forms

- `text-input`: `{colors.surface-1}` background, 1px `{colors.surface-hairline}` border, `{typography.body}` text, `{shapes.radius-md}` 8px corners, padding `8px 12px`. Focus state swaps border to `{colors.primary-focus}` + adds 2px focus halo.
- `text-area`: same as input but `min-height: 96px`, padding `{spacing.sm} {spacing.md}`.
- `select-trigger`: same as input; caret icon in `{colors.ink-subtle}`.
- Dropdown menu opens to a `{colors.surface-3}` panel with 1px `{colors.surface-hairline-strong}` border + `{elevation.lift-3}` shadow.

### Tables

- `table-header`: `{colors.canvas}` background, `{typography.eyebrow}` (uppercase +0.6px tracking), `{colors.ink-subtle}` text, `{spacing.sm}` / `{spacing.md}` padding, 1px hairline bottom.
- `table-row-hover`: `{colors.surface-1}` background on hover, 120ms ease.
- `table-row-selected`: `{colors.surface-2}` background, `2px solid {colors.primary}` left border (active row marker).
- Row height: 40px compact, 48px comfortable.
- Sticky headers on long lists (versions, episodes, ingest tasks).

### Navigation

- `header-bar`: `{colors.canvas}` background, 56px height, 1px hairline bottom, padding `0 {spacing.lg}`. Holds logo left, search center (where applicable), notification + avatar right.
- `sidebar-nav`: `{colors.canvas}` background, 240px width, padding `{spacing.md}`, 1px hairline right. Holds grouped menu sections with `DownOutlined` collapse arrows.
- `sidebar-item-active`: `{colors.surface-1}` background + 2px `{colors.primary}` left border — same accent treatment as `table-row-selected`.
- `panel-tabs`: `{colors.surface-1}` background, 1px hairline bottom, active tab gains 2px `{colors.primary}` bottom border.

### Status & Feedback

- `status-badge` (neutral): `{colors.surface-2}` background, `{colors.ink-muted}` text, 1px hairline border.
- `status-badge-success` / `-warning` / `-error`: same surface; text color picks the semantic role.
- `keyboard-hint`: `{colors.surface-2}` background, `{typography.mono}` text, 1px hairline border — used inline next to menu items (`⌘K`).
- `tooltip`: `{colors.surface-4}` background, 1px strong hairline border, `{elevation.lift-3}` shadow.

### Special Surfaces

- `modal-backdrop`: `rgba(0, 0, 0, 0.6)` over `{colors.semantic-overlay}`.
- `empty-state`: transparent background, centered `{spacing.xxl}` padding, `{typography.body}` muted text, optional single CTA (`button-primary`).
- `graph-canvas`: `{colors.canvas}` background with 1px hairline frame; nodes use semantic palette (red/orange/yellow/green/blue/purple) to denote priority / type — like Linear's in-product UI.

---

## 5. Layout Principles

### Spacing Scale (4px base)

| Token | Value | Common Use |
|---|---|---|
| `{spacing.xxs}` | 4px | Inline icon-text gap, hairline inset |
| `{spacing.xs}` | 8px | Form field stacking, button gap |
| `{spacing.sm}` | 12px | Card metadata, table cell padding |
| `{spacing.md}` | 16px | Card interior, list item spacing |
| `{spacing.lg}` | 24px | Card padding, section block gap |
| `{spacing.xl}` | 32px | Page padding, modal padding |
| `{spacing.xxl}` | 48px | Block separator, drawer header |
| `{spacing.section}` | 96px | Reserved for marketing / empty-state hero |

### Grid & Container

- **Max content width** ≈ 1280px (settings, ingestion tables); dashboards can extend to 1536px.
- **Three-column app shell**:
  1. `header-bar` — 56px tall, full width, fixed.
  2. `sidebar-nav` — 240px wide, fixed left, collapsible to 64px on icon-only mode.
  3. `content-area` — flex 1, padding `{spacing.xl}`.
- **Workbench layouts** (ontology editor, episode explorer) typically use a 2-column split: tree list left (320px), detail panel right (flex).

### Whitespace Philosophy

> **The dark canvas IS the whitespace.** Sections separate by lift onto surface-1 panels, not by gaps in white.

- Within a panel: `{spacing.lg}` 24px between content blocks, `{spacing.md}` 16px between row groups.
- Between sections of the same page: `{spacing.xl}` 32px (tight) or `{spacing.xxl}` 48px (clear break).
- Workbench screens favor density: rarely exceed `{spacing.xxl}` between major panels — operators want less scrolling.

---

## 6. Depth & Elevation

### Shadow & Surface Layering

| Level | Treatment | Use |
|---|---|---|
| 0 (flat) | No shadow, no border | Default body type, table cells |
| 1 (hairline lift) | `{colors.surface-1}` background on canvas, 1px `{colors.surface-hairline}` border | Default cards, panels, graph canvas frame |
| 2 (surface lift) | `{colors.surface-2}` background, 1px `{colors.surface-hairline-strong}` border | Featured cards, hovered rows, modal panel base |
| 3 (floating surface) | `{colors.surface-3}` background + 1px strong hairline + `{elevation.lift-3}` shadow | Dropdown menus, popovers, tooltips |
| 4 (modal scrim) | `{elevation.lift-2}` shadow + `rgba(0,0,0,0.6)` backdrop | Modal panels, command palette |
| Focus (halo) | `{elevation.focus-ring}` — 2px `{colors.primary-focus}` outline at 55% opacity | Focused input, focused button, focused row |

### Depth Hygiene

- **The brand resists drop shadows on dark almost entirely.** Cards depth = surface ladder + 1px border, never shadow.
- **Subtle white edge highlight** on the top edge of lifted panels (`box-shadow: inset 0 1px 0 rgba(255,255,255,0.04)`) — gives the dark surface a faint "pixel rendered" feel. Optional, apply only to surface-2 and above.
- **No atmospheric gradients, no spotlight cards, no glow effects on lavender.**

---

## 7. Design Guidelines (Do's and Don'ts)

### Do

- Reserve `{colors.canvas}` (`#010102`) as the canvas — never substitute true `#000000`.
- Use `{colors.primary}` lavender ONLY for: brand mark, primary CTA, active selection, focus ring, link emphasis.
- Step through `surface-1` → `surface-4` for hierarchy. Never skip a level on the same view.
- Pair display weight 600 with body weight 400 — the system resists 700+ display weights.
- Apply aggressively negative letter-spacing on display (`−2.4px` at 56px scales to `−0.05px` at body).
- Compose CTAs as `{shapes.radius-md}` 8px corners.
- Frame data-intensive surfaces (graph canvas, ontology tables, version diffs) with 1px hairline borders.
- Use mono (`{typography.mono}`) for keyboard hints, IDs, and JSON snippets — never for body copy.
- Provide a 2px lavender focus halo (`{elevation.focus-ring}`) on every interactive element.
- Run `npm run type-check` (vue-tsc) before considering a UI task complete.

### Don't

- **Don't ship a light-mode console by default.** Inverse surfaces are reserved for print/audit contexts.
- **Don't introduce a second chromatic accent** (orange, pink, green for chrome). Semantic palette is the only exception.
- **Don't add atmospheric gradients or spotlight cards.** Cards live in surface-1 with hairline borders.
- **Don't pill-round CTAs** — buttons are 8px corners. Tabs are pills.
- **Don't use `#000000` true black as the canvas.** The faint blue tint of `#010102` is intentional.
- **Don't combine multiple bright accents** in a single surface — lavender is the singular brand voice.
- **Don't use shadow on flat dark cards.** Lifting = surface step + hairline border.
- **Don't ship 16px body** — workbench density wants 14px default.
- **Don't wrap long type identifiers / CURIEs** without word-break; consider `word-break: break-all` on ontology URIs.

---

## 8. Responsive Behavior

### Breakpoints

| Name | Width | Key Changes |
|---|---|---|
| Desktop-XL | 1536px | Default — full 3-column shell, 1280–1536px content |
| Desktop | 1280px | Same as XL; grid max-width enforced |
| Tablet | 1024px | Sidebar collapses to icon rail (64px); card grids 3-up → 2-up |
| Mobile-Lg | 768px | Sidebar moves into drawer; nav top-bar gains hamburger button |
| Mobile | 640px | Single column; `display-xl` scales 56px → 28px; table cells stack |

### Touch Targets

| Element | Min Tap Size |
|---|---|
| Primary CTA | ≥ 40px height (desktop) → ≥ 44px (touch viewports) |
| Sidebar nav item | ≥ 44px height on touch |
| Form inputs | ≥ 44px tap target |
| Tablet grid cards | ≥ 88px tall |
| Icon-only buttons | 32×32px desktop → 44×44px touch |

### Collapsing Strategy

- **Top bar**: full width down to 768px. Below 768px: hamburger left, compact logo, single avatar right.
- **Sidebar**: full 240px to 1024px. 1024–768px: collapsed icon rail. <768px: off-screen drawer triggered by hamburger.
- **Card grids**: 3-up → 2-up at 1024px → 1-up below 768px. Detail panels become full-screen below 768px.
- **Workbench split panes** (tree + detail): tree list overlays detail panel below 768px (full-screen switching).
- **Tables**: scroll horizontally below 768px — never squeeze columns below 80px.
- **Graph canvas**: maintain aspect ratio; mobile shows legend + zoom controls stacked below canvas.

### Image & Icon Behavior

- Force-graph screenshots maintain aspect ratio; never crop. Zoom level resets to 1.0 on viewport change.
- Customer / partner logos in the marquee collapse from 6-up to 3-up below 768px.
- Icons (Ant Design Icons) use `currentColor` so they track text color across themes.

---

## 9. Agent Prompt Guide

Quick reference for AI coding agents (Claude / Cursor / Copilot) building into OntoGraph Console.

### Token Cheat Sheet (copy-paste-ready)

```
Canvas:           #010102   /* deepest dark, never #000000 */
Surface-1:        #0f1011   /* cards, panels */
Surface-2:        #141516   /* featured cards, modals */
Surface-3:        #18191a   /* dropdowns, segmented track */
Surface-4:        #191a1b   /* tooltips, deepest lift */
Hairline:         #23252a   /* 1px borders */
Hairline-strong:  #34343a   /* modal outline, focus */
Primary:          #5e6ad2   /* accent — CTA, focus, selection */
Primary-hover:    #828fff
Primary-active:   #4a55b8
Primary-focus:    #5e69d1   /* ring tint */
Ink:              #f7f8f8   /* primary text */
Ink-muted:        #d0d6e0
Ink-subtle:       #8a8f98
Ink-tertiary:     #62666d   /* disabled, placeholder */
Success:          #27a644
Warning:          #f2c94c
Error:            #eb5757
Info:             #56ccf2

Default font:     Inter (system fallback: SF Pro Display)
Mono font:        JetBrains Mono (fallback: SF Mono, ui-monospace)
Default body:     14px / 400 / 1.55 / -0.05px tracking
Default radius:   8px (buttons) · 12px (cards) · 16px (hero panels)
Spacing base:     4px
Default padding:  24px (cards) · 32px (pages) · 48px (modals)
```

### Reusable Prompts

**When creating a new card or panel:**
> "Use `{colors.surface-1}` background, 1px `{colors.surface-hairline}` border, 12px radius, `{spacing.lg}` padding. Title in `{typography.card-title}` (Inter 600 20px), body in `{typography.body}` (Inter 400 14px). Title color `{colors.ink}`, body color `{colors.ink-muted}`. Do not add shadow."

**When creating a primary CTA:**
> "Use `{shapes.radius-md}` corners, padding `8px 14px`, `{typography.button}` label. Background `{colors.primary}`, text `{colors.on-primary}`, border 1px solid `{colors.primary}`. Hover → `{colors.primary-hover}`. Pressed → `{colors.primary-active}`. Focus → 2px `{colors.primary-focus}` halo at 55% opacity. No shadow."

**When creating a form field:**
> "Use `{colors.surface-1}` background, `{colors.ink}` text, `{typography.body}`. Border 1px solid `{colors.surface-hairline}`, 8px radius, padding `8px 12px`. Placeholder text `{colors.ink-tertiary}`. On focus: border switches to `{colors.primary-focus}` and adds a 2px halo (`{elevation.focus-ring}`). Disabled: opacity 0.4."

**When creating a status indicator:**
> "Pick `status-badge` for a neutral badge; -success / -warning / -error variants for semantic states. Pill shape (`{shapes.radius-pill}`), `{typography.caption}`, `{colors.surface-2}` background, 1px hairline border. Color the *text* (success/warning/error), not the background."

**When extending the color palette:**
> "Do not introduce additional chromatic accents. Use semantic-success / -warning / -error / -info sparingly. For any new neutral, derive it from the surface ladder (surface-1 through surface-4) or ink ladder (ink / -muted / -subtle / -tertiary)."

**When working in dark mode:**
> "All canvas surfaces use `#010102`. Never use `#000000`. Hairlines are `#23252a`. Borders on cards are 1px. Lifting = surface step + hairline border, not shadow."

**When the design needs elegance:**
> "Mirror Linear-grade restraint. Lavender-blue is the only chromatic voice; it is reserved for active selection, brand mark, focus rings, and the primary CTA. Avoid gradients, glows, spotlight cards, and pill-rounded buttons. Lead with hairline + surface ladder depth."

### Validation Checklist

Before submitting any UI code, verify:

- [ ] No `#000000` in any color hex.
- [ ] No second chromatic accent introduced.
- [ ] All cards use `{shapes.radius-lg}` (12px) + 1px hairline border (no shadow on dark).
- [ ] All buttons use `{shapes.radius-md}` (8px) corners.
- [ ] All primary CTAs use `{colors.primary}` background.
- [ ] Focus ring (`{elevation.focus-ring}`) present on every interactive element.
- [ ] Default body type is `{typography.body}` (14px / 400 / 1.55).
- [ ] Page padding ≥ `{spacing.xl}` (32px) on dashboard root.
- [ ] Tablet breakpoint collapses sidebar to 64px; mobile breakpoint hides it behind drawer.
- [ ] Lavender-blue appears ≤ 2 times per viewport outside of focus / active states.

---

## Iteration Guide

1. Focus on ONE component at a time; reference it by its `components:` token name.
2. When introducing a section, decide first which surface lift it lives on.
3. Default body to `{typography.body}` (14px / 400 / 1.55).
4. Re-run the [Agent Prompt Guide validation checklist](#9-agent-prompt-guide) after edits.
5. Add new variants as separate `components:` entries (`button-primary-disabled`, `card-empty`, etc.).
6. Treat lavender-blue as scarce: brand mark, primary CTA, active selection, link emphasis, focus.
7. Run `npm run type-check` (vue-tsc) before completing a UI task.
8. Avoid inline hex literals in component code — always reference the token so palette edits propagate.

## Known Gaps

- **Light mode** is not documented as a default surface; inverse tokens exist only for audit panels in light context.
- **Form-field error styling** is implied via `status-badge-error` but a dedicated `input-error` token would strengthen consistency.
- **Graph node palette** (red/orange/yellow/green/blue/purple) lives in semantic colors for node priority and edge type — that palette is not enumerated here; add a `node-priority-*` token set in v1.1.
- **Custom fonts** (Inter / JetBrains Mono) require explicit CDN or self-hosted font-face declaration — not bundled by default.
- **Onboarding / marketing surface** within the product (welcome flow, empty-state hero) uses the same dark theme; treat `{spacing.section}` (96px) as the binding upper bound for spacing on those screens.
- **Ant Design Vue** defaults (`#1677ff` blue, `#f5f5f5` panels) are overridden via the design tokens — components rendered without theme tokens may flash the upstream palette during hydration. Always wrap them in OntoGraph-styled surfaces.
