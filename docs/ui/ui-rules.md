# OOF RL UI Reskin Rules

This document defines the naming rules for the UI reskin foundation. It is
candidate PR behavior until the reskin foundation PR is merged.

## Scope

The reskin foundation should refresh existing app and plugin surfaces without
adding new user-facing behavior.

Safe in this PR:

- shared visual tokens
- shared layout primitives
- app shell and navigation structure
- stable containers for major plugin regions
- page refresh and reformat work for existing plugin pages

Out of scope:

- new plugin features
- customization controls
- arbitrary CSS injection
- data model changes
- DB schema changes
- replay capture changes
- Live, Session, or History persistence changes

## Token Naming

Use namespaced semantic tokens for app-wide styling. Prefer meaning over exact
color names. Global foundation tokens must follow `--oof-category-name` so app
styling remains isolated from plugin-local CSS and future customization work.

- Color tokens use `--oof-color-*`.
- RGB color tokens use `--oof-color-*-rgb` for Tailwind opacity support.
- Spacing tokens use `--oof-space-*`.
- Radius tokens use `--oof-radius-*`.
- Shadow tokens use `--oof-shadow-*`.
- Type tokens use `--oof-font-size-*`, `--oof-line-height-*`, and `--oof-font-weight-*`.
- Motion tokens use `--oof-duration-*` and `--oof-ease-*`.
- App layout tokens use `--oof-app-*`.

Good examples:

- `--oof-color-bg`
- `--oof-color-surface`
- `--oof-color-surface-raised`
- `--oof-color-border`
- `--oof-color-text-muted`
- `--oof-color-accent`
- `--oof-color-accent-rgb`
- `--oof-space-4`
- `--oof-radius-panel`
- `--oof-app-main-max-width`

Avoid:

- page names in global tokens, such as `--session-card-bg`
- literal color names for semantic surfaces, such as `--blue-panel-bg`
- one-off token aliases that are only used once
- page-local color systems when a global token already fits

Legacy aliases such as `--bg`, `--surface`, `--line`, `--rl-blue`, and
`--rl-orange` should remain during the migration so existing plugin code keeps
working while pages move onto the new token names.

Tailwind CDN color config must reference these CSS variables instead of
duplicating hardcoded app hex values. Use `rgb(var(--oof-color-*-rgb) /
<alpha-value>)` so utilities such as `bg-surface`, `border-line`, and
`border-rl-blue/30` follow the same customization source as normal CSS.

## Class Naming

Use three class namespaces:

- `app-*` for the shell shared by every plugin view.
- `ui-*` for reusable view primitives.
- `<plugin>-*` for page-specific layout and visual composition.

Examples:

- `app-shell-header`
- `app-main`
- `plugin-views`
- `ui-page`
- `ui-panel`
- `ui-stat-card`
- `session-shell`
- `session-summary`
- `live-scoreboard`
- `dashboard-toolbar`

Shared `ui-*` classes should be boring, composable primitives. Page-specific
classes should express the page anatomy and any unique layout decisions.

## Container IDs

IDs are for stable behavior hooks and major reskinnable regions. Do not add IDs
to every tiny element.

Preserve existing JavaScript hook IDs. If an ID is used by app or plugin JS, do
not rename it without updating and testing all callers.

Recommended plugin page regions:

- `<plugin>-page`
- `<plugin>-shell`
- `<plugin>-header`
- `<plugin>-controls`
- `<plugin>-summary`
- `<plugin>-live`
- `<plugin>-list`
- `<plugin>-history`
- `<plugin>-empty`

For Session specifically, preserve the existing behavior IDs and add stable
containers around major regions, such as:

- `session-page`
- `session-shell`
- `session-controls`
- `session-summary`
- `session-live-game`
- `session-stat-totals`
- `session-match-list`
- `session-history`

## Navigation Discovery

The app shell may restyle navigation, but it must not hardcode plugin
availability. Navigation must remain discovery-first.

- Settings schema/plugin discovery decides which plugin views exist and whether
  they are enabled.
- `/api/nav` may provide ordering, labels, grouping hints, or visibility hints.
- `/api/nav` must not become the source of available plugins.
- Never assume `view_id` equals `plugin_id`.
- Use `plugin_id` for plugin asset fetch/init paths.
- Use `view_id` for view DOM IDs, active-view state, and nav identity.
- Unknown future plugin views should appear without editing `web/app.js`.

## Global vs Page Local

Put a style in `web/style.css` when it is:

- an app shell rule
- a design token
- a reusable primitive
- a shared component used by multiple pages
- a compatibility alias for existing plugin styles

Keep a style page-local when it is:

- unique to one plugin's anatomy
- tied to markup that only one plugin owns
- experimental during refresh/reformat

Move repeated page-local styles into `ui-*` only after at least two pages need
the same pattern.

## UI Workflow

Use this order:

1. Refresh: add page anatomy, stable containers, and modernized chrome.
2. Reformat: tighten density, spacing, hierarchy, and responsive behavior.
3. Final Polish: app-wide visual flare, edge states, icons, glow, and microstates.

Final polish and new features should wait until existing pages have completed
their refresh/reformat pass.

## Safety

Live, Session, and History are data-sensitive surfaces.

If a UI change touches any data behavior, explicitly document:

- what is read
- what is written
- what is not touched
- why the behavior change is necessary

Do not hide bad source data with display-only patches. If source data is wrong,
fix or log the source-data problem separately.
