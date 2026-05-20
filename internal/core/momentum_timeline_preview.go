package core

import (
	"net/http"
)

const momentumTimelinePreviewHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Momentum Timeline Preview</title>
<style>
:root {
  color-scheme: dark;
  --bg: #101114;
  --panel: #181b20;
  --panel-strong: #20252b;
  --text: #eef2f4;
  --muted: #97a1ab;
  --line: #2e363f;
  --blue: #58b8ff;
  --orange: #ffb14a;
  --green: #69d28e;
  --red: #f06c72;
}
* {
  box-sizing: border-box;
}
body {
  margin: 0;
  min-height: 100vh;
  background: var(--bg);
  color: var(--text);
  font: 14px/1.45 "Segoe UI", system-ui, sans-serif;
}
button {
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--panel-strong);
  color: var(--text);
  cursor: pointer;
  font: inherit;
  min-height: 34px;
  padding: 0 12px;
}
button:hover {
  border-color: var(--muted);
}
.shell {
  width: min(1180px, calc(100% - 32px));
  margin: 0 auto;
  padding: 24px 0;
}
.topbar {
  align-items: end;
  border-bottom: 1px solid var(--line);
  display: flex;
  gap: 16px;
  justify-content: space-between;
  padding-bottom: 16px;
}
h1 {
  font-size: 22px;
  font-weight: 650;
  letter-spacing: 0;
  margin: 0;
}
.meta {
  color: var(--muted);
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 8px;
}
.pill {
  border: 1px solid var(--line);
  border-radius: 999px;
  color: var(--muted);
  padding: 3px 9px;
}
.pill.ended {
  color: var(--red);
}
.pill.live {
  color: var(--green);
}
.actions {
  display: flex;
  gap: 8px;
}
.summary {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin: 18px 0;
}
.metric {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 12px;
}
.metric span {
  color: var(--muted);
  display: block;
  font-size: 12px;
}
.metric strong {
  display: block;
  font-size: 20px;
  margin-top: 4px;
}
table {
  border-collapse: collapse;
  width: 100%;
}
th,
td {
  border-bottom: 1px solid var(--line);
  padding: 10px 8px;
  text-align: left;
  vertical-align: top;
}
th {
  color: var(--muted);
  font-size: 12px;
  font-weight: 600;
}
tbody tr:hover {
  background: rgba(255, 255, 255, 0.035);
}
.team-blue {
  color: var(--blue);
}
.team-orange {
  color: var(--orange);
}
.empty,
.error {
  border: 1px solid var(--line);
  border-radius: 6px;
  color: var(--muted);
  padding: 16px;
}
.error {
  border-color: rgba(240, 108, 114, 0.5);
  color: var(--red);
}
.signals {
  color: var(--muted);
  display: grid;
  gap: 2px;
  min-width: 180px;
}
@media (max-width: 760px) {
  .topbar {
    align-items: stretch;
    flex-direction: column;
  }
  .summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  table {
    font-size: 13px;
  }
  th:nth-child(5),
  td:nth-child(5) {
    display: none;
  }
}
</style>
</head>
<body>
<main class="shell">
  <section class="topbar">
    <div>
      <h1>Momentum Timeline Preview</h1>
      <div class="meta">
        <span id="match-guid" class="pill">match: none</span>
        <span id="match-state" class="pill live">runtime</span>
        <span id="updated-at" class="pill">updated: pending</span>
      </div>
    </div>
    <div class="actions">
      <button id="refresh" type="button">Refresh</button>
      <button id="auto" type="button" aria-pressed="false">Auto</button>
    </div>
  </section>

  <section class="summary" aria-label="Timeline summary">
    <div class="metric"><span>Entries</span><strong id="entry-count">0</strong></div>
    <div class="metric"><span>Next index</span><strong id="next-index">0</strong></div>
    <div class="metric"><span>Blue pressure</span><strong id="blue-pressure">0.00</strong></div>
    <div class="metric"><span>Orange pressure</span><strong id="orange-pressure">0.00</strong></div>
  </section>

  <section id="content" aria-live="polite"></section>
</main>
<script>
const endpoint = "/internal/momentum-timeline-snapshot";
const content = document.getElementById("content");
const refreshButton = document.getElementById("refresh");
const autoButton = document.getElementById("auto");
let timer = null;

function fmt(value) {
  if (typeof value !== "number" || !Number.isFinite(value)) return "0.00";
  return value.toFixed(2);
}

function teamClass(team) {
  return team === "blue" ? "team-blue" : team === "orange" ? "team-orange" : "";
}

function text(value) {
  return value === undefined || value === null || value === "" ? "-" : String(value);
}

function html(value) {
  return text(value).replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;"
  }[char]));
}

function setText(id, value) {
  document.getElementById(id).textContent = value;
}

function signalLines(entry) {
  const blue = entry.Blue || {};
  const orange = entry.Orange || {};
  return [
    "blue P " + fmt(blue.Pressure) + " C " + fmt(blue.EventDerivedControl) + " V " + fmt(blue.Volatility),
    "orange P " + fmt(orange.Pressure) + " C " + fmt(orange.EventDerivedControl) + " V " + fmt(orange.Volatility),
    "confidence B " + fmt(blue.Confidence) + " O " + fmt(orange.Confidence)
  ];
}

function render(snapshot) {
  const entries = Array.isArray(snapshot.Entries) ? snapshot.Entries : [];
  setText("match-guid", "match: " + text(snapshot.MatchGUID));
  setText("match-state", snapshot.MatchEnded ? "ended" : "runtime");
  document.getElementById("match-state").className = "pill " + (snapshot.MatchEnded ? "ended" : "live");
  setText("updated-at", "updated: " + new Date().toLocaleTimeString());
  setText("entry-count", entries.length);
  setText("next-index", snapshot.NextIndex || 0);

  const last = entries.length ? entries[entries.length - 1] : {};
  setText("blue-pressure", fmt((last.Blue || {}).Pressure));
  setText("orange-pressure", fmt((last.Orange || {}).Pressure));

  if (!entries.length) {
    content.innerHTML = '<div class="empty">No runtime timeline entries.</div>';
    return;
  }

  const rows = entries.slice().reverse().map((entry) => {
    const signals = signalLines(entry).map((line) => '<span>' + html(line) + '</span>').join("");
    return '<tr>' +
      '<td>#' + html(entry.Index) + '</td>' +
      '<td>' + html(entry.Action) + '<br><span class="' + teamClass(entry.ActorTeam) + '">' + html(entry.ActorTeam) + '</span></td>' +
      '<td><span class="' + teamClass(entry.ImpactTeam) + '">' + html(entry.ImpactTeam) + '</span></td>' +
      '<td>' + html(entry.PlayerName) + '<br><span class="pill">' + html(entry.PlayerID) + '</span></td>' +
      '<td>' + html(entry.Category) + '</td>' +
      '<td><div class="signals">' + signals + '</div></td>' +
    '</tr>';
  }).join("");

  content.innerHTML = '<table><thead><tr>' +
    '<th>Index</th><th>Action</th><th>Impact</th><th>Player</th><th>Category</th><th>Signals</th>' +
    '</tr></thead><tbody>' + rows + '</tbody></table>';
}

async function refresh() {
  try {
    const response = await fetch(endpoint, { cache: "no-store" });
    if (!response.ok) throw new Error(response.status + " " + response.statusText);
    render(await response.json());
  } catch (err) {
    content.innerHTML = '<div class="error">' + html(err.message) + '</div>';
  }
}

function toggleAuto() {
  if (timer) {
    clearInterval(timer);
    timer = null;
    autoButton.setAttribute("aria-pressed", "false");
    return;
  }
  refresh();
  timer = setInterval(refresh, 1500);
  autoButton.setAttribute("aria-pressed", "true");
}

refreshButton.addEventListener("click", refresh);
autoButton.addEventListener("click", toggleAuto);
refresh();
</script>
</body>
</html>`

func (s *Server) handleMomentumTimelinePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(momentumTimelinePreviewHTML))
}
