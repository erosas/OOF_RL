# OOF RL UI Rules Foundation

Status: PR #2 foundation for the UI overhaul.

## Purpose

This document defines the shared visual rules future OOF RL page refreshes should use. PR #2 adds the design foundation only: CSS tokens, reusable primitive classes, and density guidance. It does not redesign Dashboard, Live, Ranks, Session, History, Ballchasing, Debug Assistant, Settings, Overlay Lab, Boost, Scoreboard, or engine UI.

## Visual Direction

OOF RL should feel like structured Rocket League telemetry software: dense, readable, technical, and game-aware. Pages should be built from panels, stat blocks, tables, feeds, badges, and compact toolbars. Avoid marketing-page layout, oversized hero spacing, decorative empty space, and page-specific CSS one-offs when a shared primitive fits.

Color should communicate state:

- Blue and orange are team identity colors.
- Green means healthy, positive, or running.
- Yellow means warning or attention.
- Red means error, destructive, or negative.
- Cyan/blue may be used for system/runtime information.
- Purple is reserved for highlights, overlays, or advanced systems.

## Density And Spacing

Use compact desktop density by default.

- App/page padding should stay around 12-16px.
- Panel padding should stay around 12-16px.
- Dense rows should use 6-10px vertical padding.
- Page headers should be short and functional.
- Prefer 2-3 column panel grids on normal desktop widths.
- Stack panels only when width requires it.
- Do not create vertical scroll by padding alone.

Use the shared spacing tokens in `web/style.css`:

- `--space-1` through `--space-8`
- `--page-gap`
- `--panel-padding`
- `--table-row-padding-y`

## Token Usage

New CSS should prefer the namespaced `--oof-*` tokens. Legacy aliases such as `--bg`, `--surface`, `--line`, `--rl-blue`, and `--rl-orange` remain for existing views and Tailwind-injected fragments.

Use:

- `--oof-bg`, `--oof-surface`, `--oof-surface-2`, `--oof-surface-3`
- `--oof-line`, `--oof-line-strong`
- `--oof-text`, `--oof-text-soft`, `--oof-muted`
- `--oof-team-blue`, `--oof-team-orange`
- `--oof-good`, `--oof-warning`, `--oof-danger`, `--oof-info`, `--oof-highlight`
- `--radius-sm` through `--radius-xl`
- `--shadow-panel`, `--glow-blue`, `--glow-orange`

Do not introduce new page-local colors unless the existing token set cannot express the state.

## Primitive Classes

Future page refreshes should compose from `.ui-*` primitives before adding page-specific classes.

Layout:

- `.ui-page`
- `.ui-page-header`
- `.ui-toolbar`
- `.ui-action-row`
- `.ui-grid`, `.ui-grid-2`, `.ui-grid-3`, `.ui-grid-auto`

Content:

- `.ui-panel`
- `.ui-card`
- `.ui-panel-header`
- `.ui-section-header`
- `.ui-section-title`
- `.ui-section-kicker`

Metrics and tables:

- `.ui-stat-grid`
- `.ui-stat-card`
- `.ui-stat-label`
- `.ui-stat-value`
- `.ui-compact-row`
- `.ui-table`

Status and actions:

- `.ui-chip`
- `.ui-badge`
- `.ui-button`
- `.ui-icon-button`
- `.ui-empty-state`
- `.ui-callout`

Team and state variants should use suffix classes such as `.ui-badge-blue`, `.ui-badge-orange`, `.ui-chip-good`, `.ui-callout-warning`, and `.ui-callout-danger`.

## Future Page PR Rules

When redesigning a page:

- Keep routing IDs and plugin mount points stable.
- Keep per-view scroll behavior intact.
- Use existing app data exactly as-is unless the PR explicitly scopes data work.
- Prefer shared primitives over new page-specific component systems.
- Keep page CSS scoped and small.
- Test narrow and desktop widths.
- Preserve reduced vertical waste from the sidebar shell.

## Not In Scope For This Foundation

This foundation does not add React, TypeScript, JSX, Vite, Webpack, Tailwind build steps, a skinning plugin, persisted density modes, sidebar customization, Overlay Lab productionization, Boost Engine, Scoreboard Engine, or overlay runtime changes.

It also does not change Live, Session, History, saved matches, replay capture, SQLite schema, Momentum Engine math, or plugin contracts.
