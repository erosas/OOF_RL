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
  --mtl-panel-border: #20303d;
  --mtl-rail-base: #1b232b;
  --mtl-blue-control: #147cff;
  --mtl-orange-control: #ff8a00;
  --mtl-contested: #8b3ff4;
  --mtl-neutral: #8d949c;
  --mtl-selected-range: #2ee9d1;
  --mtl-confidence-high: #48d86f;
  --mtl-confidence-medium: #ffab22;
  --mtl-confidence-low: #ff3b24;
  --mtl-text-primary: #eef4f7;
  --mtl-text-muted: #9aa8b4;
  --mtl-glow-strength: 0.38;
}
* {
  box-sizing: border-box;
}
body {
  margin: 0;
  min-height: 100vh;
  background: var(--mtl-bg);
  color: var(--mtl-text-primary);
  font: 14px/1.45 "Segoe UI", system-ui, sans-serif;
}
.page {
  display: grid;
  gap: 14px;
  grid-template-columns: minmax(0, 1fr) 170px;
  margin: 0 auto;
  max-width: 1540px;
  padding: 20px 24px 24px;
}
.topbar {
  align-items: center;
  display: grid;
  gap: 16px;
  grid-column: 1 / -1;
  grid-template-columns: minmax(280px, 1fr) minmax(420px, 0.75fr);
}
h1,
h2,
h3,
p {
  margin: 0;
}
h1 {
  font-size: 24px;
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
.fixture {
  align-items: center;
  background: var(--mtl-panel);
  border: 1px solid var(--mtl-panel-border);
  border-radius: 7px;
  display: flex;
  gap: 16px;
  justify-content: space-between;
  min-height: 54px;
  padding: 10px 14px;
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
  font-size: 24px;
  font-weight: 750;
}
.timeline-panel,
.side-panel,
.card {
  background: rgba(16, 24, 32, 0.92);
  border: 1px solid var(--mtl-panel-border);
  border-radius: 7px;
}
.timeline-panel {
  grid-column: 1 / 2;
  padding: 16px 16px 12px;
}
.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
  padding: 10px 6px 0;
}
.legend-item {
  align-items: center;
  color: var(--mtl-text-muted);
  display: flex;
  gap: 8px;
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
  grid-template-columns: 1fr 1.22fr 1.1fr;
}
.card {
  min-height: 150px;
  padding: 14px;
}
.card h2 {
  font-size: 15px;
  font-weight: 650;
  letter-spacing: 0.02em;
  margin-bottom: 12px;
  text-transform: uppercase;
}
.selected-card {
  border-color: rgba(46, 233, 209, 0.42);
}
.selected-hero {
  align-items: center;
  display: flex;
  gap: 14px;
}
.hero-marker {
  align-items: center;
  background: rgba(46, 233, 209, 0.12);
  border: 2px solid var(--mtl-selected-range);
  border-radius: 16px;
  color: var(--mtl-selected-range);
  display: grid;
  font-size: 22px;
  font-weight: 800;
  height: 86px;
  place-items: center;
  width: 86px;
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
  border: 1px solid var(--mtl-panel-border);
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
  gap: 8px;
}
.event-row,
.contribution-row,
.insight-row {
  align-items: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.07);
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
  text-align: left;
  width: 100%;
}
.event-row:focus-visible,
.mtl-marker-hit-target:focus-visible {
  outline: 2px solid var(--mtl-selected-range);
  outline-offset: 3px;
}
.event-row.is-selected {
  background: rgba(46, 233, 209, 0.08);
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
  border: 1px solid currentColor;
  border-radius: 10px;
  display: grid;
  font-size: 11px;
  font-weight: 800;
  height: 34px;
  place-items: center;
  width: 34px;
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
  grid-column: 2 / 3;
  grid-row: 2 / 4;
  padding: 14px;
}
.icon-stack {
  display: grid;
  gap: 14px;
  margin-top: 16px;
}
.icon-ref {
  align-items: center;
  display: grid;
  gap: 10px;
  grid-template-columns: 44px minmax(0, 1fr);
}
.footer-note {
  align-items: center;
  color: var(--mtl-text-muted);
  display: flex;
  gap: 8px;
  grid-column: 1 / 2;
  justify-content: center;
  padding: 8px;
}
.mtl-root {
  display: block;
  height: auto;
  overflow: visible;
  width: 100%;
}
.mtl-time-label,
.mtl-selection-badge-text,
.mtl-marker-label {
  fill: var(--mtl-text-primary);
  font-family: "Segoe UI", system-ui, sans-serif;
  font-size: 14px;
}
.mtl-time-label {
  fill: var(--mtl-text-primary);
}
.mtl-tick-major {
  stroke: rgba(255, 255, 255, 0.6);
  stroke-width: 1.5;
}
.mtl-tick-minor {
  stroke: rgba(255, 255, 255, 0.22);
  stroke-width: 1;
}
.mtl-band {
  opacity: 0.92;
}
.mtl-band-blue-control {
  fill: var(--mtl-blue-control);
}
.mtl-band-orange-control {
  fill: var(--mtl-orange-control);
}
.mtl-band-contested {
  fill: var(--mtl-contested);
}
.mtl-band-neutral {
  fill: var(--mtl-neutral);
}
.mtl-segment-divider,
.mtl-ot-divider {
  stroke: rgba(255, 255, 255, 0.42);
  stroke-width: 1;
}
.mtl-selection-box {
  fill: rgba(46, 233, 209, 0.08);
  stroke: var(--mtl-selected-range);
  stroke-width: 2;
}
.mtl-selection-badge {
  fill: rgba(0, 0, 0, 0.72);
  stroke: var(--mtl-selected-range);
}
.mtl-marker-shape {
  fill: rgba(8, 13, 18, 0.92);
  stroke: currentColor;
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
  stroke-dasharray: 5 4;
  stroke-width: 2;
}
@media (max-width: 980px) {
  .page,
  .main-grid,
  .topbar {
    display: block;
  }
  .fixture,
  .timeline-panel,
  .card,
  .side-panel {
    margin-top: 12px;
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
      <h1><span>OOF RL</span> — MOMENTUM TIMELINE B-LITE</h1>
      <p class="subtitle">Post-Match Story Layer • Event-Derived Insights Only</p>
    </div>
    <section class="fixture" aria-label="Fixture summary">
      <span id="fixture-label" class="muted">Fixture</span>
      <span id="blue-team" class="team blue"></span>
      <span class="score" id="score"></span>
      <span id="orange-team" class="team orange"></span>
      <span class="muted">Mock Data</span>
    </section>
  </header>

  <section class="timeline-panel" aria-label="Fixture-only SVG timeline">
    <svg class="mtl-root" viewBox="0 0 1200 180" role="img" aria-labelledby="mtl-title mtl-desc">
      <title id="mtl-title">Momentum Timeline B-lite fixture preview</title>
      <desc id="mtl-desc">Fixture-only event-derived pressure and control signal timeline.</desc>
      <defs id="mtl-defs">
        <filter id="mtl-soft-glow" x="-30%" y="-80%" width="160%" height="260%">
          <feGaussianBlur stdDeviation="4" result="blur"></feGaussianBlur>
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

  <aside class="side-panel" aria-label="Event marker reference">
    <h2>Event Icons (Fallback)</h2>
    <p class="muted">SVG asset adoption is deferred. Markers use direct SVG shapes and short labels.</p>
    <div id="icon-reference" class="icon-stack"></div>
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
  railY: 58,
  railWidth: 1088,
  railHeight: 22,
  markerLaneY: 126,
  selectedBadgeY: 148,
  majorTickSpacing: 60,
  markerSize: 28,
  markerHitSize: 44
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
    y: constants.railY - 12,
    width: Math.max(2, Math.abs(x2 - x1)),
    height: constants.railHeight + 42
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
    ticks.push(svg("line", { class: "mtl-tick-major", x1: x, x2: x, y1: 32, y2: 86 }));
    ticks.push(svg("text", { class: "mtl-time-label", x, y: 24, "text-anchor": "middle" }, html(formatMatchClock(second))));
  }
  if (fixture.metadata.overtimeSeconds > 0) {
    const otX = secondToX(fixture.metadata.durationSeconds);
    ticks.push(svg("line", { class: "mtl-ot-divider", x1: otX, x2: otX, y1: 28, y2: 92 }));
    ticks.push(svg("text", { class: "mtl-time-label", x: otX + 36, y: 24, "text-anchor": "middle" }, "OT"));
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
  return [
    svg("rect", { class: "mtl-selection-box", x: rect.x, y: rect.y, width: rect.width, height: rect.height, rx: 6 }),
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
  const points = x + "," + (y - 18) + " " + (x + 17) + "," + (y - 8) + " " + (x + 17) + "," + (y + 10) + " " + x + "," + (y + 20) + " " + (x - 17) + "," + (y + 10) + " " + (x - 17) + "," + (y - 8);
  const title = formatMatchClock(event.second) + " " + eventNames[event.type] + " by " + event.playerName;
  return svg("g", { class: "mtl-marker mtl-marker-" + event.type.replace("_", "-") + " " + markerClass(event.team) + " " + (selected ? "is-selected" : ""), "aria-label": title }, [
    svg("title", {}, html(title)),
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
  document.querySelector(".mtl-background").innerHTML = svg("rect", { class: "mtl-rail-bg", x: constants.railX, y: constants.railY, width: constants.railWidth, height: constants.railHeight, rx: 4 });
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
    '<div>',
    '<h3>' + html(formatMatchClock(selected.second)) + ' • ' + html(eventNames[selected.type]) + '</h3>',
    '<p class="team ' + teamClass(selected.team) + '">' + html(selected.playerName) + '</p>',
    '<p class="muted">' + html(selected.summary) + '</p>',
    '<div class="tag-row"><span class="tag ' + teamClass(selected.team) + '">' + html(segment.summary) + '</span><span class="tag">Event-derived pressure/control signal</span></div>',
    '</div>',
    '</div>'
  ].join("");
  document.getElementById("event-list").innerHTML = fixture.events.slice(2, 7).map((event) =>
    '<button type="button" class="event-row ' + (event.id === selected.id ? "is-selected" : "") + '" data-event-id="' + html(event.id) + '" aria-pressed="' + (event.id === selected.id ? "true" : "false") + '">' +
    '<span>' + html(formatMatchClock(event.second)) + '</span>' +
    '<span class="mini-marker ' + teamClass(event.team) + '">' + html(fallbackLabels[event.type]) + '</span>' +
    '<span>' + html(eventNames[event.type]) + '<br><span class="muted">' + html(event.summary) + '</span></span>' +
    '<span class="team ' + teamClass(event.team) + '">' + html(event.team === "blue" ? fixture.metadata.blueTeamName : fixture.metadata.orangeTeamName) + '</span>' +
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
