# Overlay Widget System Design

## Overview

Replace the current single-purpose overlay window with a general-purpose widget system.
Users compose named **screens** from a catalog of **widgets**, assign hotkeys to each screen,
and position widgets freely in an **edit mode**. Rocket League retains mouse focus at all
times during normal play.

---

## Core Concepts

### Widget
A self-contained UI component registered by a plugin. Examples: scoreboard, boost meter,
session history, career stats, ball cam indicator. Each widget declares a default size and
can respond to game-phase visibility rules (e.g. "only show during a live match").

### Screen
A named collection of positioned widgets with a hotkey and a click-through setting.
Users create as many screens as they want. Two typical screens to start:

- **HUD** — click-through, shown automatically during a match (scoreboard, boost meter, etc.)
- **History** — toggled by hotkey, interactive or click-through, shown on demand (session
  history, career stats, etc.)

There is nothing special about these two screens beyond their defaults. The user can rename,
delete, or reconfigure them freely.

### Mode
The overlay window is always one fullscreen WebView2 instance. A mode controls which screen
is active and whether the window receives mouse input.

| Mode | Click-through | What is visible |
|------|--------------|-----------------|
| Game | ON | Active HUD screen, widget visibility driven by game phase |
| Screen N | OFF (configurable) | That screen's widgets |
| Edit | OFF | Active screen + drag handles on every widget |

---

## Architecture

### Single WebView2 Window

One always-present fullscreen window. `WS_EX_TRANSPARENT` is toggled via
`SetWindowLong(hwnd, GWL_EXSTYLE, ...)` when switching between game mode and any
interactive screen. The window never closes; it shows or hides like today.

```
┌────────────────────────────────────────────────────────┐
│  WebView2 — fullscreen, WS_EX_TOPMOST                  │
│                                                        │
│  [Scoreboard widget]         [Boost widget]            │
│        (20, 20)                  (1800, 900)           │
│                                                        │
│  [History panel]  ← hidden in game mode                │
│       (400, 100)                                       │
└────────────────────────────────────────────────────────┘
```

### Go Side

New fields in `config.Config`:

```toml
[[screens]]
name            = "HUD"
hotkey          = "F9"
click_through   = true
auto_show_phase = ["in_match"]   # optional: auto-show on game phase

  [[screens.widgets]]
  id   = "scoreboard"
  x    = 20
  y    = 20

  [[screens.widgets]]
  id   = "boost_meter"
  x    = 1800
  y    = 900

[[screens]]
name          = "History"
hotkey        = "F8"
click_through = false

  [[screens.widgets]]
  id = "session_history"
  x  = 400
  y  = 100

[overlay]
edit_hotkey = "F10"
```

The hotkey listener goroutine checks all screen hotkeys plus the edit hotkey each tick.
On activation it calls a helper that:
1. Adds or removes `WS_EX_TRANSPARENT` via `SetWindowLong`
2. Dispatches the active screen/mode to JS via `webview.Eval` or a bound function

```go
func setMode(hwnd windows.HWND, ov webview2.WebView, mode string) {
    style, _, _ := procGetWindowLong.Call(uintptr(hwnd), gwlExStyle)
    if mode == "game" {
        style |= wsExTransparent
    } else {
        style &^= wsExTransparent
    }
    procSetWindowLong.Call(uintptr(hwnd), gwlExStyle, style)
    ov.Dispatch(func() {
        ov.Eval(fmt.Sprintf("window.__setOverlayMode(%q)", mode))
    })
}
```

### JS Side

`window.__setOverlayMode(mode)` updates a reactive state variable. Each widget checks:
- Is it on the currently active screen?
- Does the current game phase allow it to render?

Widget visibility matrix evaluated on every mode change and every game-phase event:

```js
function isWidgetVisible(widget, activeScreen, gamePhase) {
    if (widget.screen !== activeScreen) return false;
    if (widget.showPhases && !widget.showPhases.includes(gamePhase)) return false;
    return true;
}
```

---

## Game Phase State Machine

Driven entirely by WebSocket events the frontend already receives.

```
idle ──(match.started)──► in_match
in_match ──(state.updated: bHasWinner)──► post_game
in_match ──(match.ended)──────────────► post_game
post_game ──(match.destroyed)──────────► idle
in_match ──(match.destroyed)──────────► idle
```

Widgets declare which phases they render in. Defaults:

| Widget | idle | in_match | post_game |
|--------|------|----------|-----------|
| Scoreboard | hidden | visible | visible |
| Boost meter | hidden | visible | hidden |
| Session history | visible | visible | visible |
| Career stats | visible | visible | visible |

A widget not declaring `showPhases` is always visible when its screen is active.

---

## Plugin Interface Change

Plugins gain an optional `Widgets()` method returning widget declarations. This populates
the widget catalog in the settings UI.

```go
type WidgetDeclaration struct {
    ID            string // stable key, e.g. "boost_meter"
    Label         string // display name, e.g. "Boost Meter"
    DefaultWidth  int
    DefaultHeight int
    DefaultPhases []string // e.g. ["in_match"]
}

// Added to plugin.Plugin interface (optional — return nil to opt out)
Widgets() []WidgetDeclaration
```

---

## Edit Mode

Edit mode temporarily disables click-through and renders a drag handle and resize grip on
each widget. Releasing a widget saves its new position to the screen's config entry.
Pressing the edit hotkey again restores the previous mode and re-enables click-through
if applicable.

Position is saved per widget per screen, same mechanism as the current overlay rect save
(`GetWindowRect` → config write), but scoped to a widget ID.

---

## Settings UI

A new **Screens** settings card replaces the current overlay card:

- List of screens with hotkey, click-through toggle, and auto-show phase selector
- Add / remove screens
- Per-screen widget list with add-from-catalog button and "edit layout" shortcut that
  activates edit mode for that screen

---

## Implementation Phases

1. **Click-through toggle** — add `WS_EX_TRANSPARENT` constant, `setMode` helper, second
   hotkey field to config. No UI changes yet.
2. **Widget rendering** — overlay JS renders positioned divs from a hardcoded widget list;
   game phase state machine drives visibility.
3. **Plugin widget declarations** — add `Widgets()` to interface; plugins declare their
   widgets; settings UI shows the catalog.
4. **Screen config** — full TOML screen/widget schema; settings UI for managing screens.
5. **Edit mode** — drag handles, per-widget position save.