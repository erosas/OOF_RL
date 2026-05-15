const stateTone = {
  empty: "danger",
  critical: "danger",
  low: "warn",
  medium: "medium",
  high: "good",
  full: "good",
  unknown: "unknown",
};

const activityLabel = {
  draining: "Using boost",
  using_boost: "Actively using boost",
  pickup: "Boost pickup",
  idle: "Idle",
  stale: "Stale signal",
  unavailable: "Unavailable",
};

const severityLabel = {
  low: "Low",
  medium: "Medium",
  high: "High",
  info: "Info",
};

function clampBoost(value) {
  if (typeof value !== "number" || Number.isNaN(value)) return null;
  return Math.max(0, Math.min(100, value));
}

function displayBoost(sample) {
  const value = clampBoost(sample.boostAmount);
  if (value === null) return "—";
  return String(value);
}

function boostPercent(sample) {
  const value = clampBoost(sample.boostAmount);
  return value === null ? 0 : value;
}

function toneFor(sample) {
  if (!sample || sample.state === "unknown" || sample.freshnessState === "unknown") return "unknown";
  if (sample.freshnessState === "stale") return "stale";
  return stateTone[sample.state] || "medium";
}

function isDimmed(sample) {
  return sample.freshnessState === "stale" || sample.freshnessState === "unknown";
}

function createEl(tag, className, text) {
  const el = document.createElement(tag);
  if (className) el.className = className;
  if (text !== undefined) el.textContent = text;
  return el;
}

function renderSelfGauge(sample) {
  const host = document.getElementById("self-boost");
  host.replaceChildren();
  const percent = boostPercent(sample);
  const tone = toneFor(sample);
  host.className = `self-card tone-${tone}${isDimmed(sample) ? " is-dimmed" : ""}`;

  const head = createEl("div", "self-head");
  head.append(createEl("span", "eyebrow", "Self gauge"));
  head.append(createEl("span", "chip", "Arc Reactor target"));

  const gauge = createEl("div", "arc-gauge");
  gauge.style.setProperty("--boost", `${percent}%`);
  gauge.setAttribute("aria-label", `Self boost ${displayBoost(sample)}`);
  const core = createEl("div", "gauge-core");
  core.append(createEl("strong", "", displayBoost(sample)));
  core.append(createEl("span", "", sample.boostAmount === null ? "Unknown" : "Boost"));
  gauge.append(core);

  const meta = createEl("div", "state-meta");
  meta.append(createEl("strong", "", sample.state || "unknown"));
  meta.append(createEl("span", "", `${activityLabel[sample.activity] || sample.activity} | ${sample.freshnessState}`));
  meta.append(createEl("small", "", sample.reason || "Fixture reason unavailable."));

  host.append(head, gauge, meta);
}

function renderMeter(sample) {
  const meter = createEl("article", `meter-card tone-${toneFor(sample)}${isDimmed(sample) ? " is-dimmed" : ""}`);
  const top = createEl("div", "meter-top");
  top.append(createEl("strong", "", sample.name || "Unknown player"));
  top.append(createEl("span", "boost-value", displayBoost(sample)));

  const track = createEl("div", "meter-track");
  const fill = createEl("span", "meter-fill");
  fill.style.width = `${boostPercent(sample)}%`;
  track.append(fill);

  const details = createEl("div", "meter-details");
  details.append(createEl("span", "", sample.boostAmount === null ? "Unknown boost state" : `${sample.state} boost state`));
  details.append(createEl("span", "", `${sample.freshnessState} | confidence ${sample.confidence}`));

  const reason = createEl("small", "reason", sample.reason || "Fixture reason unavailable.");
  meter.append(top, track, details, reason);
  return meter;
}

function renderTeammates(teammates) {
  const host = document.getElementById("teammate-meters");
  host.replaceChildren();
  if (!Array.isArray(teammates) || teammates.length === 0) {
    host.append(createEl("p", "empty-state", "No fixture teammate boost meters available."));
    return;
  }
  teammates.forEach((sample) => host.append(renderMeter(sample)));
}

function renderWarnings(warnings) {
  const host = document.getElementById("warning-list");
  host.replaceChildren();
  if (!Array.isArray(warnings) || warnings.length === 0) {
    host.append(createEl("p", "empty-state", "No fixture boost warnings available."));
    return;
  }
  warnings.forEach((warning) => {
    const card = createEl("article", `warning-card severity-${warning.severity || "info"}`);
    const top = createEl("div", "warning-top");
    top.append(createEl("strong", "", warning.label || "Boost warning"));
    top.append(createEl("span", "chip", severityLabel[warning.severity] || "Info"));
    card.append(top, createEl("p", "", warning.summary || "Fixture warning summary unavailable."));
    host.append(card);
  });
}

function formatPanelValue(value, fallback = "—") {
  if (value === null || value === undefined) return fallback;
  return String(value);
}

function renderTeamPanels(panels) {
  const host = document.getElementById("team-panels");
  host.replaceChildren();
  if (!Array.isArray(panels) || panels.length === 0) {
    host.append(createEl("p", "empty-state", "No fixture team boost panels available."));
    return;
  }
  panels.forEach((panel) => {
    const card = createEl("article", `team-card team-${panel.team || "unknown"} completeness-${panel.completeness || "unknown"}`);
    const head = createEl("div", "panel-head");
    const title = createEl("div");
    title.append(createEl("p", "eyebrow", panel.completeness || "unknown"));
    title.append(createEl("h2", "", panel.title || "Team boost state"));
    head.append(title, createEl("span", "chip", `${panel.knownPlayers || 0}/${panel.expectedPlayers || 0} known`));

    const stats = createEl("div", "team-stats");
    [
      ["Average", formatPanelValue(panel.averageBoost)],
      ["Lowest known", formatPanelValue(panel.lowestKnownBoost)],
      ["Low players", formatPanelValue(panel.lowBoostPlayers, "0")],
      ["Completeness", panel.completeness || "unknown"],
    ].forEach(([label, value]) => {
      const item = createEl("div", "stat");
      item.append(createEl("span", "", label), createEl("strong", "", value));
      stats.append(item);
    });

    card.append(head, stats, createEl("p", "summary", panel.summary || "Fixture team summary unavailable."));
    host.append(card);
  });
}

function renderStateSamples(samples) {
  const host = document.getElementById("state-grid");
  host.replaceChildren();
  if (!Array.isArray(samples) || samples.length === 0) {
    host.append(createEl("p", "empty-state", "No fixture boost state samples available."));
    return;
  }
  samples.forEach((sample) => {
    const card = createEl("article", `state-card tone-${toneFor(sample)}${isDimmed(sample) ? " is-dimmed" : ""}`);
    const head = createEl("div", "state-card-head");
    head.append(createEl("strong", "", sample.label || sample.state || "State"));
    head.append(createEl("span", "boost-pill", displayBoost(sample)));

    const track = createEl("div", "mini-track");
    const fill = createEl("span", "mini-fill");
    fill.style.width = `${boostPercent(sample)}%`;
    track.append(fill);

    const detail = createEl("p", "", `${activityLabel[sample.activity] || sample.activity} | ${sample.freshnessState} | confidence ${sample.confidence}`);
    const reason = createEl("small", "", sample.reason || "Fixture reason unavailable.");
    card.append(head, track, detail, reason);
    host.append(card);
  });
}

function renderPreview(signal) {
  renderSelfGauge(signal.self || {});
  renderTeammates(signal.teammates);
  renderWarnings(signal.warnings);
  renderTeamPanels(signal.teamPanels);
  renderStateSamples(signal.stateSamples);
}

async function loadFixture() {
  try {
    const response = await fetch("/internal/boost-overlay-preview/fixture.json", { cache: "no-store" });
    if (!response.ok) throw new Error(`fixture request failed: ${response.status}`);
    const fixture = await response.json();
    renderPreview(fixture.boostOverlaySignal || {});
  } catch (err) {
    document.body.append(createEl("p", "load-error", `Fixture Boost Preview failed to load mock data: ${err.message}`));
  }
}

loadFixture();
