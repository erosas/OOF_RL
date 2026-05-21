package core

import "net/http"

const momentumTimelineBLitePreviewHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Momentum Timeline B-lite Preview</title>
<style>
:root {
  color-scheme: dark;
  --mtl-bg: #080d12;
  --mtl-panel: #101820;
  --mtl-panel-strong: #14202a;
  --mtl-panel-border: #2b3a48;
  --mtl-panel-edge: #405366;
  --mtl-rail-base: #1b232b;
  --mtl-blue-control: #2f8cff;
  --mtl-orange-control: #ff7a18;
  --mtl-contested: #9b5cff;
  --mtl-neutral: #8d949c;
  --mtl-selected-range: #2ee9d1;
  --mtl-cyan-edge: #35d9ff;
  --mtl-blue-hot: #39a7ff;
  --mtl-orange-hot: #ffb347;
  --mtl-purple-hot: #bd6dff;
  --mtl-confidence-high: #48d86f;
  --mtl-confidence-medium: #ffab22;
  --mtl-confidence-low: #ff3b24;
  --mtl-text-primary: #eef4f7;
  --mtl-text-muted: #9aa8b4;
  --mtl-text-soft: #c6d0d8;
  --mtl-glow-strength: 0.34;
  --mtl-shadow: 0 16px 42px rgba(0, 0, 0, 0.3);
  --mtl-neon-shadow: 0 0 0 1px rgba(92, 130, 160, 0.18), inset 0 1px 0 rgba(255, 255, 255, 0.045), 0 18px 48px rgba(0, 0, 0, 0.42);
}
* {
  box-sizing: border-box;
}
body {
  margin: 0;
  min-height: 100vh;
  background:
    linear-gradient(180deg, rgba(21, 32, 42, 0.98), rgba(6, 10, 14, 0.98) 54%),
    linear-gradient(90deg, rgba(47, 140, 255, 0.08), transparent 24%, transparent 76%, rgba(255, 122, 24, 0.075)),
    var(--mtl-bg);
  color: var(--mtl-text-primary);
  font: 14px/1.45 "Segoe UI", system-ui, sans-serif;
}
.page {
  display: grid;
  gap: 10px;
  grid-template-columns: 158px minmax(650px, 1fr) 168px;
  margin: 0 auto;
  max-width: 1536px;
  padding: 15px 16px 16px;
}
.topbar {
  align-items: center;
  display: grid;
  gap: 10px;
  grid-column: 1 / -1;
  grid-template-columns: 390px minmax(440px, 1fr) minmax(250px, 0.56fr);
}
h1,
h2,
h3,
p {
  margin: 0;
}
h1 {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 0;
}
h1 span {
  color: #1c8dff;
}
.subtitle,
.muted {
  color: var(--mtl-text-muted);
}
.state-strip {
  background: rgba(13, 20, 27, 0.96);
  border: 1px solid var(--mtl-panel-border);
  border-radius: 7px;
  display: grid;
  grid-template-columns: repeat(5, minmax(72px, 1fr));
  min-height: 78px;
  overflow: hidden;
}
.state-item {
  border-right: 1px solid rgba(255, 255, 255, 0.06);
  display: grid;
  gap: 4px;
  grid-template-columns: 17px minmax(0, 1fr);
  padding: 9px 8px;
}
.state-item:last-child {
  border-right: 0;
}
.state-icon {
  align-items: center;
  border: 1px solid currentColor;
  border-radius: 999px;
  display: grid;
  font-size: 11px;
  font-weight: 800;
  height: 18px;
  place-items: center;
  width: 18px;
}
.state-label {
  color: var(--mtl-text-primary);
  font-size: 10px;
  font-weight: 800;
  text-transform: uppercase;
}
.state-copy {
  color: var(--mtl-text-soft);
  font-size: 10px;
  grid-column: 2;
  line-height: 1.3;
}
.fixture {
  align-items: center;
  background: rgba(12, 18, 25, 0.92);
  border: 1px solid var(--mtl-panel-border);
  border-radius: 7px;
  box-shadow: 0 12px 34px rgba(0, 0, 0, 0.22);
  display: flex;
  gap: 10px;
  justify-content: space-between;
  min-height: 54px;
  padding: 9px 12px;
}
.fixture .muted {
  font-size: 12px;
}
.team {
  font-weight: 650;
}
.team.blue {
  color: var(--mtl-blue-control);
}
.team.orange {
  color: var(--mtl-orange-control);
}
.score {
  font-size: 23px;
  font-weight: 750;
}
.timeline-panel,
.side-panel,
.card,
.guide-card,
.journey-panel {
  background:
    linear-gradient(180deg, rgba(22, 34, 45, 0.88), rgba(8, 13, 18, 0.96)),
    rgba(13, 20, 27, 0.94);
  border: 1px solid rgba(78, 99, 119, 0.78);
  border-radius: 12px;
  box-shadow: var(--mtl-neon-shadow);
}
.timeline-panel {
  grid-column: 2 / 3;
  overflow: hidden;
  padding: 18px 18px 18px;
  position: relative;
}
.timeline-panel::before {
  background: linear-gradient(90deg, rgba(47, 140, 255, 0.85), rgba(53, 217, 255, 0.45), rgba(255, 122, 24, 0.74));
  content: "";
  height: 2px;
  left: 0;
  position: absolute;
  right: 0;
  top: 0;
}
.timeline-panel::after {
  background:
    linear-gradient(90deg, rgba(47, 140, 255, 0.08), transparent 28%, transparent 72%, rgba(255, 122, 24, 0.08)),
    repeating-linear-gradient(90deg, transparent 0 30px, rgba(255, 255, 255, 0.025) 31px, transparent 32px);
  bottom: 0;
  content: "";
  left: 0;
  opacity: 0.58;
  pointer-events: none;
  position: absolute;
  right: 0;
  top: 0;
}
.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  padding: 12px 6px 0;
}
.legend-item {
  align-items: center;
  color: var(--mtl-text-muted);
  display: flex;
  gap: 8px;
  min-height: 22px;
}
.swatch {
  border-radius: 3px;
  display: inline-block;
  height: 10px;
  width: 20px;
}
.main-grid {
  display: grid;
  gap: 14px;
  grid-column: 2 / 3;
  grid-template-columns: minmax(230px, 0.9fr) minmax(300px, 1.24fr) minmax(250px, 1fr);
}
.card {
  min-height: 150px;
  overflow: hidden;
  padding: 14px;
}
.card h2 {
  color: var(--mtl-text-soft);
  font-size: 13px;
  font-weight: 650;
  letter-spacing: 0.04em;
  margin-bottom: 12px;
  text-transform: uppercase;
}
.selected-card {
  background:
    linear-gradient(145deg, rgba(47, 140, 255, 0.2), rgba(46, 233, 209, 0.14) 38%, rgba(13, 20, 27, 0.94) 62%),
    rgba(13, 20, 27, 0.94);
  border-color: rgba(46, 233, 209, 0.58);
  box-shadow: inset 4px 0 0 rgba(46, 233, 209, 0.82), 0 0 0 1px rgba(46, 233, 209, 0.2), 0 0 34px rgba(46, 233, 209, 0.16), 0 22px 58px rgba(0, 0, 0, 0.38);
}
.selected-hero {
  align-items: stretch;
  display: grid;
  gap: 14px;
  grid-template-columns: 104px minmax(0, 1fr);
}
.hero-marker {
  align-items: center;
  align-self: stretch;
  background:
    linear-gradient(135deg, rgba(46, 233, 209, 0.28), rgba(46, 233, 209, 0.06)),
    repeating-linear-gradient(180deg, rgba(255, 255, 255, 0.06) 0 1px, transparent 1px 8px);
  border: 2px solid var(--mtl-selected-range);
  border-radius: 8px;
  box-shadow: inset 0 0 36px rgba(46, 233, 209, 0.18), 0 0 38px rgba(46, 233, 209, 0.3);
  color: var(--mtl-selected-range);
  display: grid;
  font-size: 28px;
  font-weight: 800;
  min-height: 122px;
  flex: 0 0 auto;
  place-items: center;
  width: 104px;
}
.selected-meta {
  display: grid;
  gap: 8px;
}
.selected-time {
  color: var(--mtl-selected-range);
  font-size: 28px;
  font-weight: 800;
  line-height: 1.05;
}
.selected-title {
  color: var(--mtl-text-primary);
  font-size: 18px;
  font-weight: 750;
}
.selected-summary {
  color: var(--mtl-text-soft);
  max-width: 28rem;
}
.tag-row,
.chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}
.chip,
.tag {
  background: rgba(255, 255, 255, 0.025);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 6px;
  color: var(--mtl-text-muted);
  padding: 4px 8px;
}
.chip.blue,
.tag.blue {
  border-color: rgba(20, 124, 255, 0.65);
  color: #5db0ff;
}
.chip.orange {
  border-color: rgba(255, 138, 0, 0.65);
  color: #ffb257;
}
.chip.teal {
  border-color: rgba(46, 233, 209, 0.65);
  color: var(--mtl-selected-range);
}
.event-list {
  display: grid;
  gap: 6px;
}
.event-row,
.contribution-row,
.insight-row {
  align-items: center;
  border: 1px solid transparent;
  border-bottom-color: rgba(255, 255, 255, 0.07);
  display: grid;
  gap: 10px;
  grid-template-columns: 46px 44px minmax(0, 1fr) 86px;
  padding: 8px 0;
}
.event-row {
  background: transparent;
  color: inherit;
  cursor: pointer;
  font: inherit;
  min-height: 58px;
  text-align: left;
  transition: background 140ms ease, border-color 140ms ease, transform 140ms ease;
  width: 100%;
}
.event-row:hover {
  background: rgba(255, 255, 255, 0.035);
  border-color: rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  transform: translateY(-1px);
}
.event-row:focus-visible,
.mtl-marker-hit-target:focus-visible {
  outline: 2px solid var(--mtl-selected-range);
  outline-offset: 3px;
}
.event-row.is-selected {
  background:
    linear-gradient(90deg, rgba(46, 233, 209, 0.2), rgba(46, 233, 209, 0.055)),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.035) 0 1px, transparent 1px 14px);
  border: 1px solid rgba(46, 233, 209, 0.5);
  border-radius: 6px;
  padding: 8px;
}
.event-row.is-selected,
.mtl-marker.is-selected .mtl-marker-shape {
  filter: drop-shadow(0 0 10px rgba(46, 233, 209, var(--mtl-glow-strength)));
}
.mini-marker {
  align-items: center;
  background: rgba(6, 10, 16, 0.78);
  border: 1px solid currentColor;
  border-radius: 10px;
  box-shadow: inset 0 0 10px rgba(255, 255, 255, 0.035), 0 0 14px currentColor;
  display: grid;
  font-size: 11px;
  font-weight: 800;
  height: 34px;
  place-items: center;
  width: 34px;
}
.mini-marker.blue {
  color: var(--mtl-blue-control);
}
.mini-marker.orange {
  color: var(--mtl-orange-control);
}
.confidence-dot {
  border-radius: 999px;
  display: inline-block;
  height: 7px;
  margin-left: 6px;
  vertical-align: 1px;
  width: 7px;
}
.confidence-dot.high {
  background: var(--mtl-confidence-high);
}
.confidence-dot.medium {
  background: var(--mtl-confidence-medium);
}
.confidence-dot.low {
  background: var(--mtl-confidence-low);
}
.contribution-row {
  grid-template-columns: 24px minmax(0, 1fr) 120px 42px;
}
.bar {
  background: rgba(255, 255, 255, 0.08);
  border-radius: 999px;
  height: 6px;
  overflow: hidden;
}
.bar span {
  display: block;
  height: 100%;
}
.breakdown {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(6, minmax(0, 1fr));
}
.breakdown-item {
  align-items: center;
  display: grid;
  gap: 6px;
  justify-items: center;
}
.insight-row {
  grid-template-columns: 42px minmax(0, 1fr);
}
.side-panel {
  grid-column: 3 / 4;
  grid-row: 2 / span 3;
  align-self: start;
  display: grid;
  gap: 8px;
  padding: 0;
}
.guide-rail {
  align-self: start;
  display: grid;
  gap: 8px;
}
.left-guide {
  grid-column: 1 / 2;
  grid-row: 2 / span 3;
}
.guide-card {
  min-height: 92px;
  padding: 11px;
}
.guide-card h2 {
  align-items: center;
  display: flex;
  gap: 6px;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.02em;
  margin: 0 0 8px;
  text-transform: uppercase;
}
.guide-card p,
.guide-card li {
  color: var(--mtl-text-soft);
  font-size: 10px;
  line-height: 1.38;
}
.guide-card ul {
  margin: 8px 0 0;
  padding-left: 16px;
}
.guide-number {
  align-items: center;
  border: 1px solid var(--mtl-blue-control);
  border-radius: 999px;
  color: #6bb6ff;
  display: inline-grid;
  flex: 0 0 auto;
  font-size: 11px;
  height: 19px;
  place-items: center;
  width: 19px;
}
.guide-card.teal .guide-number {
  border-color: var(--mtl-selected-range);
  color: var(--mtl-selected-range);
}
.icon-stack {
  display: grid;
  gap: 7px;
  margin-top: 10px;
}
.icon-ref {
  align-items: center;
  background: rgba(255, 255, 255, 0.025);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 7px;
  display: grid;
  gap: 10px;
  grid-template-columns: 44px minmax(0, 1fr);
  padding: 8px;
}
.footer-note {
  align-items: center;
  color: var(--mtl-text-muted);
  display: flex;
  gap: 8px;
  grid-column: 2 / 3;
  justify-content: center;
  padding: 8px;
}
.journey-panel {
  grid-column: 1 / -1;
  padding: 12px 16px 14px;
}
.journey-panel h2 {
  font-size: 13px;
  margin: 0 0 12px;
  text-transform: uppercase;
}
.journey-steps {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(6, minmax(0, 1fr));
}
.journey-step {
  min-height: 92px;
}
.journey-step h3 {
  align-items: center;
  display: flex;
  gap: 7px;
  font-size: 12px;
}
.journey-step p {
  color: var(--mtl-text-soft);
  font-size: 11px;
  margin: 5px 0 8px;
}
.mini-rail {
  align-items: center;
  display: flex;
  gap: 2px;
  height: 18px;
}
.mini-segment {
  border-radius: 2px;
  height: 6px;
}
.mtl-root {
  display: block;
  height: auto;
  margin: 2px 0 4px;
  overflow: visible;
  width: 100%;
  filter: drop-shadow(0 18px 30px rgba(0, 0, 0, 0.28));
}
.mtl-time-label,
.mtl-selection-badge-text,
.mtl-marker-label {
  fill: var(--mtl-text-primary);
  font-family: "Segoe UI", system-ui, sans-serif;
  font-size: 14px;
}
.mtl-time-label {
  fill: var(--mtl-text-soft);
}
.mtl-tick-major {
  stroke: rgba(238, 244, 247, 0.55);
  stroke-width: 1.5;
}
.mtl-tick-minor {
  stroke: rgba(255, 255, 255, 0.22);
  stroke-width: 1;
}
.mtl-band {
  filter: saturate(1.18) url(#mtl-band-glow);
  opacity: 0.96;
}
.mtl-rail-bg {
  fill: url(#mtl-rail-gradient);
  stroke: rgba(93, 122, 145, 0.48);
  stroke-width: 1;
}
.mtl-rail-frame {
  fill: rgba(6, 10, 15, 0.78);
  stroke: rgba(87, 117, 139, 0.7);
  stroke-width: 1.5;
}
.mtl-rail-glow {
  fill: rgba(47, 140, 255, 0.16);
  filter: url(#mtl-rail-glow);
}
.mtl-rail-scanline {
  stroke: rgba(255, 255, 255, 0.22);
  stroke-width: 1;
}
.mtl-band-blue-control {
  fill: url(#mtl-blue-band);
}
.mtl-band-orange-control {
  fill: url(#mtl-orange-band);
}
.mtl-band-contested {
  fill: url(#mtl-contested-band);
}
.mtl-band-neutral {
  fill: url(#mtl-neutral-band);
}
.mtl-segment-divider,
.mtl-ot-divider {
  stroke: rgba(255, 255, 255, 0.42);
  stroke-width: 1;
}
.mtl-selection-box {
  fill: rgba(46, 233, 209, 0.12);
  filter: url(#mtl-selection-glow);
  stroke: var(--mtl-selected-range);
  stroke-width: 2.5;
}
.mtl-selection-corner {
  fill: none;
  stroke: #d9fff9;
  stroke-linecap: square;
  stroke-width: 3;
}
.mtl-selection-badge {
  fill: rgba(5, 12, 18, 0.92);
  filter: url(#mtl-selection-glow);
  stroke: var(--mtl-selected-range);
  stroke-width: 1.5;
}
.mtl-marker-shape {
  fill: rgba(7, 12, 18, 0.98);
  filter: url(#mtl-marker-glow);
  stroke: currentColor;
  stroke-width: 2;
}
.mtl-marker-halo {
  fill: currentColor;
  filter: url(#mtl-marker-glow);
  opacity: 0.12;
}
.mtl-marker-stem {
  stroke: currentColor;
  stroke-opacity: 0.7;
  stroke-width: 1.5;
}
.mtl-marker.is-selected .mtl-marker-shape {
  stroke: var(--mtl-selected-range);
  stroke-width: 3;
}
.mtl-marker.is-selected .mtl-marker-stem {
  stroke: var(--mtl-selected-range);
  stroke-width: 2;
}
.mtl-marker-blue {
  color: var(--mtl-blue-control);
}
.mtl-marker-orange {
  color: var(--mtl-orange-control);
}
.mtl-marker-neutral {
  color: var(--mtl-selected-range);
}
.mtl-marker-hit-target {
  cursor: pointer;
  fill: transparent;
  pointer-events: all;
}
.mtl-marker-hit-target.is-selected {
  fill: rgba(46, 233, 209, 0.12);
  stroke: var(--mtl-selected-range);
  stroke-width: 1.5;
}
.mtl-focus-ring {
  fill: none;
  stroke: var(--mtl-selected-range);
  filter: url(#mtl-selection-glow);
  stroke-width: 2.5;
}
@media (max-width: 1180px) {
  .page {
    grid-template-columns: 172px minmax(0, 1fr);
  }
  .topbar {
    grid-template-columns: 1fr;
  }
  .state-strip {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }
  .timeline-panel,
  .main-grid,
  .footer-note {
    grid-column: 2;
    grid-row: auto;
  }
  .left-guide {
    grid-column: 1;
    grid-row: 2 / span 3;
  }
  .side-panel {
    grid-column: 1 / -1;
    grid-row: auto;
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
  .icon-stack {
    grid-template-columns: 1fr;
  }
  .main-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .journey-steps {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (max-width: 980px) {
  .page,
  .main-grid,
  .topbar,
  .state-strip,
  .side-panel,
  .journey-steps {
    display: block;
  }
  .page {
    padding: 14px;
  }
  .fixture,
  .timeline-panel,
  .card,
  .side-panel {
    margin-top: 12px;
  }
  .fixture {
    align-items: flex-start;
    flex-direction: column;
    gap: 6px;
  }
  .event-row {
    grid-template-columns: 44px 42px minmax(0, 1fr);
  }
  .event-row .event-team {
    grid-column: 3;
  }
  .icon-stack {
    grid-template-columns: 1fr;
  }
  .timeline-panel,
  .main-grid,
  .left-guide,
  .footer-note {
    grid-column: 1;
  }
}
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.001ms !important;
    scroll-behavior: auto !important;
    transition-duration: 0.001ms !important;
  }
}
</style>
</head>
<body>
<main class="page">
  <header class="topbar">
    <div>
      <h1><span>OOF RL</span> - MOMENTUM TIMELINE B-LITE</h1>
      <h2>UI BREAKDOWN & INTERACTION GUIDE</h2>
      <p class="subtitle">Post-Match Story Layer - Event-Derived Insights Only</p>
    </div>
    <section class="state-strip" aria-label="State legend guide">
      <div class="state-item blue"><span class="state-icon">B</span><span class="state-label">Blue Control</span><span class="state-copy">Blue team has sustained pressure / control signal</span></div>
      <div class="state-item contested"><span class="state-icon">C</span><span class="state-label">Contested</span><span class="state-copy">Both teams evenly trading pressure</span></div>
      <div class="state-item orange"><span class="state-icon">O</span><span class="state-label">Orange Control</span><span class="state-copy">Orange team has sustained pressure / control signal</span></div>
      <div class="state-item neutral"><span class="state-icon">N</span><span class="state-label">Neutral / Reset</span><span class="state-copy">Low pressure / reset in play</span></div>
      <div class="state-item teal"><span class="state-icon">S</span><span class="state-label">Selected Range</span><span class="state-copy">User selected moment or pressure sequence</span></div>
    </section>
    <section class="fixture" aria-label="Fixture summary">
      <span id="fixture-label" class="muted">Fixture</span>
      <span id="blue-team" class="team blue"></span>
      <span class="score" id="score"></span>
      <span id="orange-team" class="team orange"></span>
      <span class="muted">Mock Data</span>
    </section>
  </header>

  <aside class="guide-rail left-guide" aria-label="Timeline overview guide">
    <article class="guide-card">
      <h2><span class="guide-number">1</span>Timeline Overview</h2>
      <p>Visualizes match momentum across the full timeline. Bands show event-derived pressure/control states.</p>
      <ul><li>Icons mark key events.</li><li>Click icon to focus moment.</li><li>Hover for details.</li></ul>
    </article>
    <article class="guide-card">
      <h2><span class="guide-number">2</span>Legend</h2>
      <p>Explains what each band color means.</p>
    </article>
    <article class="guide-card teal">
      <h2><span class="guide-number">3</span>Selected Moment Card</h2>
      <p>Shows details about the selected moment or pressure sequence.</p>
    </article>
    <article class="guide-card">
      <h2><span class="guide-number">4</span>Data Confidence</h2>
      <p>Confidence in the event-derived signal quality for this moment/sequence.</p>
    </article>
  </aside>

  <section class="timeline-panel" aria-label="Fixture-only SVG timeline">
    <svg class="mtl-root" viewBox="0 0 1200 220" role="img" aria-labelledby="mtl-title mtl-desc">
      <title id="mtl-title">Momentum Timeline B-lite fixture preview</title>
      <desc id="mtl-desc">Fixture-only event-derived pressure and control signal timeline.</desc>
      <defs id="mtl-defs">
        <linearGradient id="mtl-rail-gradient" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#243342"></stop>
          <stop offset="48%" stop-color="#0b1219"></stop>
          <stop offset="100%" stop-color="#18232d"></stop>
        </linearGradient>
        <linearGradient id="mtl-blue-band" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#5db5ff"></stop>
          <stop offset="48%" stop-color="#147cff"></stop>
          <stop offset="100%" stop-color="#0b3e96"></stop>
        </linearGradient>
        <linearGradient id="mtl-orange-band" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#ffb347"></stop>
          <stop offset="48%" stop-color="#ff7a18"></stop>
          <stop offset="100%" stop-color="#8a3300"></stop>
        </linearGradient>
        <linearGradient id="mtl-contested-band" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#d28cff"></stop>
          <stop offset="50%" stop-color="#8f45ff"></stop>
          <stop offset="100%" stop-color="#4f1f94"></stop>
        </linearGradient>
        <linearGradient id="mtl-neutral-band" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#c6cdd4"></stop>
          <stop offset="48%" stop-color="#7f8a95"></stop>
          <stop offset="100%" stop-color="#404b55"></stop>
        </linearGradient>
        <filter id="mtl-band-glow" x="-10%" y="-120%" width="120%" height="340%">
          <feGaussianBlur stdDeviation="2.8" result="blur"></feGaussianBlur>
          <feMerge><feMergeNode in="blur"></feMergeNode><feMergeNode in="SourceGraphic"></feMergeNode></feMerge>
        </filter>
        <filter id="mtl-rail-glow" x="-10%" y="-220%" width="120%" height="520%">
          <feGaussianBlur stdDeviation="9" result="blur"></feGaussianBlur>
        </filter>
        <filter id="mtl-marker-glow" x="-120%" y="-120%" width="340%" height="340%">
          <feGaussianBlur stdDeviation="4" result="blur"></feGaussianBlur>
          <feMerge><feMergeNode in="blur"></feMergeNode><feMergeNode in="SourceGraphic"></feMergeNode></feMerge>
        </filter>
        <filter id="mtl-selection-glow" x="-40%" y="-80%" width="180%" height="260%">
          <feGaussianBlur stdDeviation="4.5" result="blur"></feGaussianBlur>
          <feMerge><feMergeNode in="blur"></feMergeNode><feMergeNode in="SourceGraphic"></feMergeNode></feMerge>
        </filter>
      </defs>
      <g class="mtl-background"></g>
      <g class="mtl-time-ticks"></g>
      <g class="mtl-momentum-bands"></g>
      <g class="mtl-segment-dividers"></g>
      <g class="mtl-selected-range"></g>
      <g class="mtl-event-markers"></g>
      <g class="mtl-marker-hit-targets"></g>
      <g class="mtl-focus-selection"></g>
      <g class="mtl-effects"></g>
    </svg>
    <div class="legend" aria-label="Timeline legend">
      <span class="legend-item"><span class="swatch" style="background: var(--mtl-blue-control)"></span>Blue Control</span>
      <span class="legend-item"><span class="swatch" style="background: var(--mtl-orange-control)"></span>Orange Control</span>
      <span class="legend-item"><span class="swatch" style="background: var(--mtl-contested)"></span>Contested</span>
      <span class="legend-item"><span class="swatch" style="background: var(--mtl-neutral)"></span>Neutral / Reset</span>
      <span class="legend-item"><span class="swatch" style="background: transparent; border: 1px dashed var(--mtl-selected-range)"></span>Selected Range</span>
    </div>
  </section>

  <aside class="side-panel" aria-label="Timeline interaction guide">
    <article class="guide-card">
      <h2><span class="guide-number">6</span>Match Header</h2>
      <p>Fixture label, teams, score, playlist, and timestamp. Static in this view.</p>
    </article>
    <article class="guide-card">
      <h2><span class="guide-number">7</span>Event Icons (SVG)</h2>
      <p>Approved icon set used for all event markers. This preview still uses fallback labels.</p>
      <div id="icon-reference" class="icon-stack"></div>
    </article>
    <article class="guide-card">
      <h2><span class="guide-number">8</span>Interaction Behaviors</h2>
      <ul>
        <li>Hover on band: show segment info.</li>
        <li>Click event icon: focus that moment.</li>
        <li>Keyboard: arrows move focus, Enter selects.</li>
      </ul>
    </article>
    <article class="guide-card">
      <h2><span class="guide-number">9</span>Honesty Guardrails</h2>
      <p>We show event-derived pressure/control signals.</p>
      <ul><li>No saved data writes.</li><li>No live data mutation.</li><li>No certainty claims.</li></ul>
    </article>
  </aside>

  <section class="main-grid">
    <article class="card selected-card" id="selected-card"></article>
    <article class="card">
      <h2>Events Around This Moment</h2>
      <div id="event-list" class="event-list"></div>
    </article>
    <article class="card">
      <h2>Top Contributions</h2>
      <div id="contributions"></div>
    </article>
    <article class="card">
      <h2>Pressure Sequence Summary</h2>
      <p id="summary-copy" class="muted"></p>
      <div class="chip-row">
        <span class="chip blue">Blue Control</span>
        <span class="chip">Pressure Sequence</span>
        <span class="chip teal">Event-Derived Contribution</span>
      </div>
    </article>
    <article class="card">
      <h2>Event Breakdown</h2>
      <div id="breakdown" class="breakdown"></div>
    </article>
    <article class="card">
      <h2>Quick Insights</h2>
      <div id="insights"></div>
    </article>
  </section>

  <footer class="footer-note">Event-derived data only. Interpret as pressure/control signal context.</footer>

  <section class="journey-panel" aria-label="User journey example">
    <h2>User Journey Example</h2>
    <div class="journey-steps">
      <article class="journey-step"><h3><span class="guide-number">1</span>Load Timeline</h3><p>Timeline shows the full match at a glance.</p><div class="mini-rail"><span class="mini-segment" style="width: 18%; background: var(--mtl-blue-control)"></span><span class="mini-segment" style="width: 20%; background: var(--mtl-contested)"></span><span class="mini-segment" style="width: 22%; background: var(--mtl-orange-control)"></span><span class="mini-segment" style="width: 12%; background: var(--mtl-neutral)"></span></div></article>
      <article class="journey-step"><h3><span class="guide-number">2</span>Hover Band</h3><p>Hover to see segment info, confidence, and timing.</p><div class="mini-rail"><span class="mini-segment" style="width: 24%; background: var(--mtl-blue-control)"></span><span class="mini-segment" style="width: 42%; background: var(--mtl-orange-control)"></span><span class="mini-segment" style="width: 20%; background: var(--mtl-neutral)"></span></div></article>
      <article class="journey-step"><h3><span class="guide-number">3</span>Select Range</h3><p>Click and drag to select a range.</p><div class="mini-rail"><span class="mini-segment" style="width: 20%; background: var(--mtl-contested)"></span><span class="mini-segment" style="width: 48%; background: var(--mtl-selected-range)"></span><span class="mini-segment" style="width: 22%; background: var(--mtl-orange-control)"></span></div></article>
      <article class="journey-step"><h3><span class="guide-number">4</span>Review Moment</h3><p>Details update for selected moment or sequence.</p><div class="tag teal">Confidence: High</div></article>
      <article class="journey-step"><h3><span class="guide-number">5</span>Click Event</h3><p>Click an event icon to focus that specific moment.</p><div class="mini-rail"><span class="mini-segment" style="width: 30%; background: var(--mtl-blue-control)"></span><span class="mini-segment" style="width: 18%; background: var(--mtl-orange-control)"></span><span class="mini-segment" style="width: 22%; background: var(--mtl-contested)"></span></div></article>
      <article class="journey-step"><h3><span class="guide-number">6</span>Explore</h3><p>Use keyboard or mouse to explore the story.</p><div class="tag">Arrow keys / Enter</div></article>
    </div>
  </section>
</main>

<script>
const fixture = {
  metadata: {
    durationSeconds: 300,
    overtimeSeconds: 45,
    blueTeamName: "Zephyr FC",
    orangeTeamName: "Nova United",
    score: { blue: 3, orange: 2 },
    fixtureLabel: "Fixture (Mock Data)"
  },
  segments: [
    { id: "s1", startSecond: 0, endSecond: 70, state: "blue_control", confidence: "high", intensity: 0.72, summary: "Blue Control opening pressure sequence." },
    { id: "s2", startSecond: 70, endSecond: 106, state: "contested", confidence: "medium", intensity: 0.52, summary: "Contested midfield exchange." },
    { id: "s3", startSecond: 106, endSecond: 128, state: "orange_control", confidence: "medium", intensity: 0.64, summary: "Orange Control response." },
    { id: "s4", startSecond: 128, endSecond: 160, state: "neutral", confidence: "low", intensity: 0.34, summary: "Neutral / Reset stretch." },
    { id: "s5", startSecond: 160, endSecond: 194, state: "blue_control", confidence: "high", intensity: 0.8, summary: "Blue Control defensive stand." },
    { id: "s6", startSecond: 194, endSecond: 230, state: "orange_control", confidence: "medium", intensity: 0.68, summary: "Orange Control pressure." },
    { id: "s7", startSecond: 230, endSecond: 268, state: "contested", confidence: "medium", intensity: 0.58, summary: "Contested late pressure." },
    { id: "s8", startSecond: 268, endSecond: 304, state: "orange_control", confidence: "medium", intensity: 0.62, summary: "Orange Control final minute." },
    { id: "s9", startSecond: 304, endSecond: 345, state: "neutral", confidence: "low", intensity: 0.3, summary: "Neutral / Reset overtime reserve." }
  ],
  events: [
    { id: "e1", second: 9, type: "goal", team: "blue", playerName: "ZPH Rixxy", segmentId: "s1", confidence: "high", summary: "Opening Blue Control goal." },
    { id: "e2", second: 55, type: "save", team: "orange", playerName: "NVA Spectre", segmentId: "s1", confidence: "medium", summary: "Orange save slowed pressure." },
    { id: "e3", second: 86, type: "assist", team: "blue", playerName: "ZPH Juno", segmentId: "s2", confidence: "medium", summary: "Assist during Contested exchange." },
    { id: "e4", second: 112, type: "shot", team: "blue", playerName: "ZPH Juno", segmentId: "s3", confidence: "high", summary: "Shot before reset." },
    { id: "e5", second: 146, type: "epic_save", team: "blue", playerName: "Zephyr FC", segmentId: "s4", confidence: "high", summary: "Pressure Sequence stopped." },
    { id: "e6", second: 194, type: "save", team: "orange", playerName: "Nova United", segmentId: "s6", confidence: "medium", summary: "Orange defensive response." },
    { id: "e7", second: 222, type: "shot", team: "blue", playerName: "ZPH Rixxy", segmentId: "s6", confidence: "high", summary: "Blue shot into pressure." },
    { id: "e8", second: 252, type: "assist", team: "blue", playerName: "ZPH Juno", segmentId: "s7", confidence: "medium", summary: "Assist in Contested lane." },
    { id: "e9", second: 278, type: "demo", team: "orange", playerName: "NVA Spectre", segmentId: "s8", confidence: "low", summary: "Demo changed the pressure sequence." },
    { id: "e10", second: 304, type: "goal", team: "blue", playerName: "ZPH Rixxy", segmentId: "s9", confidence: "high", summary: "Overtime Blue Control goal." }
  ],
  selectedMoment: {
    selectionType: "event",
    segmentId: "s4",
    eventId: "e5",
    rangeStartSecond: 140,
    rangeEndSecond: 166,
    badgeSecond: 145
  },
  contributions: [
    { playerName: "ZPH Rixxy", team: "blue", pressureContribution: 72, momentumInfluence: 0.72, contestInvolvement: 0.38, source: "event_derived" },
    { playerName: "ZPH Juno", team: "blue", pressureContribution: 61, momentumInfluence: 0.61, contestInvolvement: 0.44, source: "event_derived" },
    { playerName: "NVA Spectre", team: "orange", pressureContribution: 38, momentumInfluence: 0.38, contestInvolvement: 0.52, source: "event_derived" }
  ],
  insights: [
    "Zephyr FC generated several Blue Control pressure sequences.",
    "Nova United had two strong defensive stands that shifted the pressure sequence.",
    "Both teams generated high Contested signal in the final 90 seconds."
  ]
};

const constants = {
  padding: 40,
  railX: 56,
  railY: 72,
  railWidth: 1088,
  railHeight: 34,
  markerLaneY: 160,
  selectedBadgeY: 198,
  majorTickSpacing: 60,
  markerSize: 34,
  markerHitSize: 50
};

const fallbackLabels = {
  goal: "GL",
  save: "SV",
  epic_save: "ES",
  assist: "AS",
  shot: "SH",
  demo: "DM"
};

const eventNames = {
  goal: "Goal",
  save: "Save",
  epic_save: "Epic Save",
  assist: "Assist",
  shot: "Shot",
  demo: "Demo"
};

let selectedEventId = fixture.selectedMoment.eventId;

function totalSeconds() {
  return fixture.metadata.durationSeconds + fixture.metadata.overtimeSeconds;
}

function secondToX(second) {
  const clamped = Math.max(0, Math.min(second, totalSeconds()));
  return constants.railX + (clamped / totalSeconds()) * constants.railWidth;
}

function formatMatchClock(second) {
  if (second <= fixture.metadata.durationSeconds) {
    const remaining = Math.max(0, fixture.metadata.durationSeconds - Math.round(second));
    return Math.floor(remaining / 60) + ":" + String(remaining % 60).padStart(2, "0");
  }
  const overtime = Math.round(second - fixture.metadata.durationSeconds);
  return "+" + Math.floor(overtime / 60) + ":" + String(overtime % 60).padStart(2, "0");
}

function selectedRangeToRect(startSecond, endSecond) {
  const x1 = secondToX(startSecond);
  const x2 = secondToX(endSecond);
  return {
    x: Math.min(x1, x2),
    y: constants.railY - 16,
    width: Math.max(2, Math.abs(x2 - x1)),
    height: constants.railHeight + 58
  };
}

function markerLane(events, event, index) {
  const near = events.slice(0, index).filter((other) => Math.abs(secondToX(other.second) - secondToX(event.second)) < 32).length;
  return near % 3;
}

function markerY(lane) {
  return constants.markerLaneY + lane * 18;
}

function selectedEvent() {
  return fixture.events.find((event) => event.id === selectedEventId) || fixture.events[0];
}

function selectedSegment() {
  const event = selectedEvent();
  return fixture.segments.find((item) => item.id === event.segmentId) || fixture.segments[0];
}

function syncSelectedMoment(event) {
  const segment = fixture.segments.find((item) => item.id === event.segmentId) || fixture.segments[0];
  selectedEventId = event.id;
  fixture.selectedMoment.eventId = event.id;
  fixture.selectedMoment.segmentId = event.segmentId;
  fixture.selectedMoment.rangeStartSecond = Math.max(0, event.second - 13);
  fixture.selectedMoment.rangeEndSecond = Math.min(totalSeconds(), event.second + 13);
  if (segment) {
    fixture.selectedMoment.rangeStartSecond = Math.max(0, Math.min(fixture.selectedMoment.rangeStartSecond, segment.startSecond));
    fixture.selectedMoment.rangeEndSecond = Math.min(totalSeconds(), Math.max(fixture.selectedMoment.rangeEndSecond, segment.endSecond));
  }
  fixture.selectedMoment.badgeSecond = event.second;
}

function selectEvent(eventId, options = {}) {
  const event = fixture.events.find((item) => item.id === eventId);
  if (!event) return;
  syncSelectedMoment(event);
  renderTimeline();
  renderCards();
  if (options.focusMarker) {
    const target = document.querySelector('.mtl-marker-hit-target[data-event-id="' + event.id + '"]');
    if (target) target.focus();
  }
}

function selectEventByOffset(eventId, offset) {
  const currentIndex = fixture.events.findIndex((event) => event.id === eventId);
  if (currentIndex < 0) return;
  const nextIndex = Math.max(0, Math.min(fixture.events.length - 1, currentIndex + offset));
  selectEvent(fixture.events[nextIndex].id, { focusMarker: true });
}

function teamClass(team) {
  return team === "blue" ? "blue" : team === "orange" ? "orange" : "";
}

function markerClass(team) {
  if (team === "blue") return "mtl-marker-blue";
  if (team === "orange") return "mtl-marker-orange";
  return "mtl-marker-neutral";
}

function segmentClass(state) {
  return "mtl-band-" + state.replace("_", "-");
}

function html(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;"
  }[char]));
}

function svg(tag, attrs = {}, content = "") {
  const attrText = Object.entries(attrs)
    .map(([key, value]) => " " + key + "=\"" + html(value) + "\"")
    .join("");
  return "<" + tag + attrText + ">" + content + "</" + tag + ">";
}

function renderTicks() {
  const ticks = [];
  for (let second = 0; second <= totalSeconds(); second += constants.majorTickSpacing) {
    const x = secondToX(second);
    ticks.push(svg("line", { class: "mtl-tick-major", x1: x, x2: x, y1: 38, y2: 116 }));
    ticks.push(svg("text", { class: "mtl-time-label", x, y: 30, "text-anchor": "middle" }, html(formatMatchClock(second))));
  }
  if (fixture.metadata.overtimeSeconds > 0) {
    const otX = secondToX(fixture.metadata.durationSeconds);
    ticks.push(svg("line", { class: "mtl-ot-divider", x1: otX, x2: otX, y1: 36, y2: 122 }));
    ticks.push(svg("text", { class: "mtl-time-label", x: otX + 36, y: 30, "text-anchor": "middle" }, "OT"));
  }
  return ticks.join("");
}

function renderBands() {
  return fixture.segments.map((segment) => {
    const x = secondToX(segment.startSecond);
    const width = Math.max(2, secondToX(segment.endSecond) - x);
    return svg("rect", {
      class: "mtl-band " + segmentClass(segment.state),
      x,
      y: constants.railY,
      width,
      height: constants.railHeight,
      rx: 3,
      opacity: Math.max(0.35, segment.intensity)
    });
  }).join("");
}

function renderDividers() {
  return fixture.segments.slice(1).map((segment) => {
    const x = secondToX(segment.startSecond);
    return svg("line", { class: "mtl-segment-divider", x1: x, x2: x, y1: constants.railY, y2: constants.railY + constants.railHeight });
  }).join("");
}

function renderSelection() {
  const selected = fixture.selectedMoment;
  const rect = selectedRangeToRect(selected.rangeStartSecond, selected.rangeEndSecond);
  const badgeX = secondToX(selected.badgeSecond);
  const c = 18;
  return [
    svg("rect", { class: "mtl-selection-box", x: rect.x, y: rect.y, width: rect.width, height: rect.height, rx: 6 }),
    svg("path", { class: "mtl-selection-corner", d: "M " + rect.x + " " + (rect.y + c) + " L " + rect.x + " " + rect.y + " L " + (rect.x + c) + " " + rect.y }),
    svg("path", { class: "mtl-selection-corner", d: "M " + (rect.x + rect.width - c) + " " + rect.y + " L " + (rect.x + rect.width) + " " + rect.y + " L " + (rect.x + rect.width) + " " + (rect.y + c) }),
    svg("path", { class: "mtl-selection-corner", d: "M " + rect.x + " " + (rect.y + rect.height - c) + " L " + rect.x + " " + (rect.y + rect.height) + " L " + (rect.x + c) + " " + (rect.y + rect.height) }),
    svg("path", { class: "mtl-selection-corner", d: "M " + (rect.x + rect.width - c) + " " + (rect.y + rect.height) + " L " + (rect.x + rect.width) + " " + (rect.y + rect.height) + " L " + (rect.x + rect.width) + " " + (rect.y + rect.height - c) }),
    svg("line", { class: "mtl-focus-ring", x1: badgeX, x2: badgeX, y1: constants.railY + constants.railHeight, y2: constants.selectedBadgeY - 20 }),
    svg("rect", { class: "mtl-selection-badge", x: badgeX - 28, y: constants.selectedBadgeY - 18, width: 56, height: 26, rx: 5 }),
    svg("text", { class: "mtl-selection-badge-text", x: badgeX, y: constants.selectedBadgeY, "text-anchor": "middle" }, html(formatMatchClock(selected.badgeSecond)))
  ].join("");
}

function renderMarker(event, index, selectedId) {
  const x = secondToX(event.second);
  const y = markerY(markerLane(fixture.events, event, index));
  const label = fallbackLabels[event.type] || "EV";
  const selected = event.id === selectedId;
  const markerRadius = selected ? 23 : 19;
  const points = x + "," + (y - markerRadius) + " " + (x + markerRadius) + "," + (y - 10) + " " + (x + markerRadius) + "," + (y + 12) + " " + x + "," + (y + markerRadius + 3) + " " + (x - markerRadius) + "," + (y + 12) + " " + (x - markerRadius) + "," + (y - 10);
  const title = formatMatchClock(event.second) + " " + eventNames[event.type] + " by " + event.playerName;
  return svg("g", { class: "mtl-marker mtl-marker-" + event.type.replace("_", "-") + " " + markerClass(event.team) + " " + (selected ? "is-selected" : ""), "aria-label": title }, [
    svg("title", {}, html(title)),
    svg("circle", { class: "mtl-marker-halo", cx: x, cy: y, r: selected ? 34 : 26 }),
    svg("line", { class: "mtl-marker-stem", x1: x, x2: x, y1: constants.railY + constants.railHeight + 3, y2: y - markerRadius + 3 }),
    svg("polygon", { class: "mtl-marker-shape", points }),
    svg("text", { class: "mtl-marker-label", x, y: y + 5, "text-anchor": "middle" }, html(label))
  ].join(""));
}

function renderHitTarget(event, index) {
  const x = secondToX(event.second);
  const y = markerY(markerLane(fixture.events, event, index));
  const selected = event.id === selectedEventId;
  return svg("circle", {
    class: "mtl-marker-hit-target " + (selected ? "is-selected" : ""),
    cx: x,
    cy: y,
    r: constants.markerHitSize / 2,
    "data-event-id": event.id,
    tabindex: "0",
    role: "button",
    "aria-pressed": selected ? "true" : "false",
    "aria-label": formatMatchClock(event.second) + " " + eventNames[event.type] + " by " + event.playerName
  });
}

function renderTimeline() {
  const selectedId = selectedEventId;
  document.querySelector(".mtl-background").innerHTML = [
    svg("rect", { class: "mtl-rail-frame", x: constants.railX - 18, y: constants.railY - 18, width: constants.railWidth + 36, height: constants.railHeight + 36, rx: 9 }),
    svg("rect", { class: "mtl-rail-glow", x: constants.railX - 4, y: constants.railY - 8, width: constants.railWidth + 8, height: constants.railHeight + 16, rx: 10 }),
    svg("rect", { class: "mtl-rail-bg", x: constants.railX, y: constants.railY, width: constants.railWidth, height: constants.railHeight, rx: 5 }),
    svg("line", { class: "mtl-rail-scanline", x1: constants.railX + 12, x2: constants.railX + constants.railWidth - 12, y1: constants.railY + 8, y2: constants.railY + 8 }),
    svg("line", { class: "mtl-rail-scanline", x1: constants.railX + 12, x2: constants.railX + constants.railWidth - 12, y1: constants.railY + constants.railHeight - 7, y2: constants.railY + constants.railHeight - 7 })
  ].join("");
  document.querySelector(".mtl-time-ticks").innerHTML = renderTicks();
  document.querySelector(".mtl-momentum-bands").innerHTML = renderBands();
  document.querySelector(".mtl-segment-dividers").innerHTML = renderDividers();
  document.querySelector(".mtl-selected-range").innerHTML = renderSelection();
  document.querySelector(".mtl-event-markers").innerHTML = fixture.events.map((event, index) => renderMarker(event, index, selectedId)).join("");
  document.querySelector(".mtl-marker-hit-targets").innerHTML = fixture.events.map(renderHitTarget).join("");
  const selectedEvent = fixture.events.find((event) => event.id === selectedId);
  if (selectedEvent) {
    const x = secondToX(selectedEvent.second);
    const y = markerY(markerLane(fixture.events, selectedEvent, fixture.events.indexOf(selectedEvent)));
    document.querySelector(".mtl-focus-selection").innerHTML = svg("circle", { class: "mtl-focus-ring", cx: x, cy: y, r: 27 });
  }
  bindMarkerInteractions();
}

function renderCards() {
  const selected = selectedEvent();
  const segment = selectedSegment();
  document.getElementById("fixture-label").textContent = fixture.metadata.fixtureLabel;
  document.getElementById("blue-team").textContent = fixture.metadata.blueTeamName;
  document.getElementById("orange-team").textContent = fixture.metadata.orangeTeamName;
  document.getElementById("score").textContent = fixture.metadata.score.blue + " | " + fixture.metadata.score.orange;
  document.getElementById("selected-card").innerHTML = [
    '<h2>Selected Moment</h2>',
    '<div class="selected-hero">',
    '<div class="hero-marker">' + html(fallbackLabels[selected.type]) + '</div>',
    '<div class="selected-meta">',
    '<div class="selected-time">' + html(formatMatchClock(selected.second)) + '</div>',
    '<div class="selected-title">' + html(eventNames[selected.type]) + '<span class="confidence-dot ' + html(selected.confidence) + '" aria-label="' + html(selected.confidence) + ' confidence"></span></div>',
    '<p class="team ' + teamClass(selected.team) + '">' + html(selected.playerName) + '</p>',
    '<p class="selected-summary">' + html(selected.summary) + '</p>',
    '<div class="tag-row"><span class="tag ' + teamClass(selected.team) + '">' + html(segment.summary) + '</span><span class="tag">Event-derived pressure/control signal</span></div>',
    '</div>',
    '</div>'
  ].join("");
  document.getElementById("event-list").innerHTML = fixture.events.slice(2, 7).map((event) =>
    '<button type="button" class="event-row ' + (event.id === selected.id ? "is-selected" : "") + '" data-event-id="' + html(event.id) + '" aria-pressed="' + (event.id === selected.id ? "true" : "false") + '">' +
    '<span>' + html(formatMatchClock(event.second)) + '</span>' +
    '<span class="mini-marker ' + teamClass(event.team) + '">' + html(fallbackLabels[event.type]) + '</span>' +
    '<span>' + html(eventNames[event.type]) + '<span class="confidence-dot ' + html(event.confidence) + '" aria-label="' + html(event.confidence) + ' confidence"></span><br><span class="muted">' + html(event.summary) + '</span></span>' +
    '<span class="event-team team ' + teamClass(event.team) + '">' + html(event.team === "blue" ? fixture.metadata.blueTeamName : fixture.metadata.orangeTeamName) + '</span>' +
    '</button>'
  ).join("");
  document.getElementById("contributions").innerHTML = fixture.contributions.map((item, index) =>
    '<div class="contribution-row">' +
    '<span>' + (index + 1) + '</span>' +
    '<span>' + html(item.playerName) + '<br><span class="muted">Event-Derived Contribution</span></span>' +
    '<span class="bar"><span style="width: ' + item.pressureContribution + '%; background: var(--mtl-' + (item.team === "blue" ? "blue-control" : "orange-control") + ')"></span></span>' +
    '<span>' + item.pressureContribution + '%</span>' +
    '</div>'
  ).join("");
  const counts = fixture.events.reduce((acc, event) => {
    acc[event.type] = (acc[event.type] || 0) + 1;
    return acc;
  }, {});
  document.getElementById("breakdown").innerHTML = Object.keys(fallbackLabels).map((type) =>
    '<div class="breakdown-item">' +
    '<span class="mini-marker">' + html(fallbackLabels[type]) + '</span>' +
    '<strong>' + (counts[type] || 0) + '</strong>' +
    '</div>'
  ).join("");
  document.getElementById("insights").innerHTML = fixture.insights.map((insight) =>
    '<div class="insight-row"><span class="mini-marker">EV</span><span>' + html(insight) + '</span></div>'
  ).join("");
  document.getElementById("summary-copy").textContent = "Zephyr FC built a Blue Control Pressure Sequence. Nova United answered with defensive events before a fixture overtime moment.";
  document.getElementById("icon-reference").innerHTML = Object.keys(fallbackLabels).map((type) =>
    '<div class="icon-ref">' +
    '<span class="mini-marker">' + html(fallbackLabels[type]) + '</span>' +
    '<span>' + html(eventNames[type]) + '<br><span class="muted">' + html(type) + ' fallback</span></span>' +
    '</div>'
  ).join("");
  bindEventListInteractions();
}

function handleSelectionKeydown(event, eventId) {
  if (event.key === "Enter" || event.key === " ") {
    event.preventDefault();
    selectEvent(eventId, { focusMarker: true });
  } else if (event.key === "ArrowLeft") {
    event.preventDefault();
    selectEventByOffset(eventId, -1);
  } else if (event.key === "ArrowRight") {
    event.preventDefault();
    selectEventByOffset(eventId, 1);
  } else if (event.key === "Home") {
    event.preventDefault();
    selectEvent(fixture.events[0].id, { focusMarker: true });
  } else if (event.key === "End") {
    event.preventDefault();
    selectEvent(fixture.events[fixture.events.length - 1].id, { focusMarker: true });
  }
}

function bindMarkerInteractions() {
  document.querySelectorAll(".mtl-marker-hit-target[data-event-id]").forEach((target) => {
    const eventId = target.getAttribute("data-event-id");
    target.addEventListener("click", () => selectEvent(eventId, { focusMarker: true }));
    target.addEventListener("keydown", (event) => handleSelectionKeydown(event, eventId));
  });
}

function bindEventListInteractions() {
  document.querySelectorAll(".event-row[data-event-id]").forEach((target) => {
    const eventId = target.getAttribute("data-event-id");
    target.addEventListener("click", () => selectEvent(eventId));
    target.addEventListener("keydown", (event) => handleSelectionKeydown(event, eventId));
  });
}

renderTimeline();
renderCards();
</script>
</body>
</html>`

func (s *Server) handleMomentumTimelineBLitePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(momentumTimelineBLitePreviewHTML))
}
