'use strict';

let _overlayMomentumTimer = null;
let _momentumFlowBar = null;
let _lastMomentumOutput = null;
let _demoModeUntil = 0;
let _timelineStart = 0;
let _timelineSamples = [];
let _timelineEvents = [];
let _lastTimelineEventKey = '';
let _heldDisplayState = null;
let _lastDisplayDominance = null;
let _lastDisplayOutput = null;
let _replayFrozenOutput = null;
let _wasReplayActive = false;
let _postReplayNeutralStarted = 0;
let _postReplayNeutralUntil = 0;
let _postReplaySignalAfter = 0;
let _awaitingPostReplayLiveSignal = false;
let _lastGoalFallbackKey = '';
let _goalFallbackStarted = 0;
let _lastMomentumResetAt = 0;

const MFB_PREFS_KEY = 'oofrl.overlay.momentumFlowBar.v1';
const MAX_TIMELINE_SAMPLES = 180;
const MAX_TIMELINE_EVENTS = 40;
const TIMELINE_MARKER_CLUSTER_MS = 6500;
const DISPLAY_STATE_MIN_HOLD_MS = 3600;
const DISPLAY_STATE_MAX_HOLD_MS = 9000;
const DISPLAY_CONTROL_PERCENT = 56;
const DISPLAY_PRESSURE_PERCENT = 66;
const DISPLAY_CONTESTED_SPREAD_PERCENT = 8;
const DISPLAY_CONTESTED_FLIP_MS = 3200;
const POST_REPLAY_MIN_NEUTRAL_MS = 1200;
const POST_REPLAY_MAX_NEUTRAL_MS = 12000;
const GOAL_FALLBACK_RESET_MS = 3750;

const DEFAULT_MFB_CONFIG = {
  enabled: true,
  variant: 'full',
  showConfidence: true,
  showLabels: true,
  showPercentages: true,
  smoothTransitions: true,
  pulseEnabled: true,
  lowConfidenceDimThreshold: 0.25,
  debug: false,
};

const DEFAULT_HOST_CONFIG = {
  position: 'top-center',
  opacity: 1,
  scale: 1,
};

class MomentumFlowBarWidget {
  constructor(root, config = {}) {
    this.root = root;
    this.config = { ...DEFAULT_MFB_CONFIG, ...config };
    this.lastPulseKey = '';
    this.renderShell();
  }

  setConfig(config = {}) {
    this.config = { ...this.config, ...config };
    this.applyConfig();
    if (this.lastOutput) this.update(this.lastOutput);
  }

  update(rawOutput) {
    if (!this.config.enabled || !this.root) return;
    const output = adaptMomentumFlowOutput(rawOutput);
    this.lastOutput = output;
    this.applyConfig();

    this.root.className = [
      'momentum-flow-bar',
      `momentum-flow-bar--${this.config.variant}`,
      stateClass(output.state),
      this.config.pulseEnabled && output.pulseTeam ? `is-pulse-${output.pulseTeam}` : '',
      output.confidence < this.config.lowConfidenceDimThreshold ? 'is-low-confidence' : '',
    ].filter(Boolean).join(' ');

    const blue = clampPercent(output.bluePercent);
    const orange = 100 - blue;
    this.setText('bluePercent', `${Math.round(blue)}%`);
    this.setText('orangePercent', `${Math.round(orange)}%`);
    this.setText('leftLabel', leftStateLabel(output.state));
    this.setText('centerLabel', this.config.showConfidence ? `CONFIDENCE: ${output.confidence.toFixed(2)}` : '');
    this.setText('rightLabel', rightStateLabel(output.state));
    this.blueBar.style.width = `${blue}%`;
    this.orangeBar.style.width = `${orange}%`;

    this.triggerPulse(output);
  }

  renderShell() {
    this.root.innerHTML = `
      <div class="momentum-flow-bar__panel">
        <div class="momentum-flow-bar__main">
          <div class="momentum-flow-bar__percent momentum-flow-bar__percent--blue" data-mfb="bluePercent">50%</div>
          <div class="momentum-flow-bar__rail-wrap">
            <div class="momentum-flow-bar__rail">
              <div class="momentum-flow-bar__blue" data-mfb="blueBar" style="width:50%"></div>
              <div class="momentum-flow-bar__orange" data-mfb="orangeBar" style="width:50%"></div>
            </div>
            <div class="momentum-flow-bar__pulse" data-mfb="pulse" aria-hidden="true">
              <svg viewBox="0 0 32 32" role="img" aria-label="Momentum pulse">
                <path d="M3 17h6l3-9 6 18 3-9h8" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"></path>
              </svg>
            </div>
          </div>
          <div class="momentum-flow-bar__percent momentum-flow-bar__percent--orange" data-mfb="orangePercent">50%</div>
        </div>
        <div class="momentum-flow-bar__labels">
          <div class="momentum-flow-bar__label momentum-flow-bar__label--left" data-mfb="leftLabel">Neutral</div>
          <div class="momentum-flow-bar__label momentum-flow-bar__label--center" data-mfb="centerLabel">Confidence: 0.00</div>
          <div class="momentum-flow-bar__label momentum-flow-bar__label--right" data-mfb="rightLabel">Neutral</div>
        </div>
      </div>
    `;
    this.blueBar = this.root.querySelector('[data-mfb="blueBar"]');
    this.orangeBar = this.root.querySelector('[data-mfb="orangeBar"]');
    this.applyConfig();
  }

  applyConfig() {
    this.root.classList.toggle('momentum-flow-bar--minimal', this.config.variant === 'minimal');
    this.root.classList.toggle('momentum-flow-bar--compact', this.config.variant === 'compact');
    this.root.querySelectorAll('.momentum-flow-bar__percent').forEach(el => {
      el.hidden = !this.config.showPercentages || this.config.variant === 'minimal';
    });
    const labels = this.root.querySelector('.momentum-flow-bar__labels');
    if (labels) labels.hidden = !this.config.showLabels || this.config.variant === 'minimal';
    const center = this.root.querySelector('[data-mfb="centerLabel"]');
    if (center) center.hidden = !this.config.showConfidence || this.config.variant === 'minimal';
  }

  triggerPulse(output) {
    const pulse = output.pulse || '';
    const pulseKey = pulse ? (output.pulseKey || `${pulse}:${output.state}:${output.pulseTeam || ''}`) : '';
    this.root.classList.remove('is-pulsing', 'is-pulse-goal');
    if (!this.config.pulseEnabled || !pulse || pulseKey === this.lastPulseKey) return;
    this.lastPulseKey = pulseKey;
    requestAnimationFrame(() => {
      this.root.classList.add('is-pulsing');
      if (pulse === 'GOAL_BURST') this.root.classList.add('is-pulse-goal');
      window.setTimeout(() => {
        this.root.classList.remove('is-pulsing', 'is-pulse-goal');
      }, 620);
    });
  }

  setText(key, value) {
    const el = this.root.querySelector(`[data-mfb="${key}"]`);
    if (el) el.textContent = value;
  }
}

window.MomentumFlowBarWidget = MomentumFlowBarWidget;
window.adaptMomentumFlowOutput = adaptMomentumFlowOutput;

window.pluginInit_overlay = () => {
  const prefs = loadMomentumFlowPrefs();
  const widgetConfig = { ...DEFAULT_MFB_CONFIG, ...prefs.widget };
  const hostConfig = { ...DEFAULT_HOST_CONFIG, ...prefs.host };
  const host = document.getElementById('momentum-flow-bar-widget');
  if (host && !_momentumFlowBar) {
    _momentumFlowBar = new MomentumFlowBarWidget(host, widgetConfig);
  } else {
    _momentumFlowBar?.setConfig(widgetConfig);
  }
  applyHostConfig(hostConfig);
  syncMomentumFlowControls(widgetConfig, hostConfig);

  wireSelect('mfb-variant', value => updateMomentumFlowPrefs({ widget: { variant: value } }));
  wireSelect('mfb-position', value => updateMomentumFlowPrefs({ host: { position: value } }));
  wireRange('mfb-opacity', value => updateMomentumFlowPrefs({ host: { opacity: Number(value) } }));
  wireRange('mfb-scale', value => updateMomentumFlowPrefs({ host: { scale: Number(value) } }));
  wireCheck('mfb-show-confidence', checked => updateMomentumFlowPrefs({ widget: { showConfidence: checked } }));
  wireCheck('mfb-show-labels', checked => updateMomentumFlowPrefs({ widget: { showLabels: checked } }));
  wireCheck('mfb-show-percentages', checked => updateMomentumFlowPrefs({ widget: { showPercentages: checked } }));
  wireCheck('mfb-pulse-enabled', checked => updateMomentumFlowPrefs({ widget: { pulseEnabled: checked } }));

  document.querySelectorAll('[data-mfb-demo]').forEach(button => {
    if (button.dataset.wired) return;
    button.dataset.wired = '1';
    button.addEventListener('click', () => {
      _demoModeUntil = Date.now() + 4500;
      const out = demoOutput(button.dataset.mfbDemo);
      _lastMomentumOutput = out;
      renderOverlayMomentum(out);
    });
  });

  const reset = document.getElementById('momentum-reset');
  if (reset && !reset.dataset.wired) {
    reset.dataset.wired = '1';
    reset.addEventListener('click', async () => {
      await fetch('/api/overlay/momentum/reset', { method: 'POST' });
      await refreshOverlayMomentum();
    });
  }

  const clearTimeline = document.getElementById('momentum-timeline-clear');
  if (clearTimeline && !clearTimeline.dataset.wired) {
    clearTimeline.dataset.wired = '1';
    clearTimeline.addEventListener('click', () => {
      resetLocalTimeline();
      renderMomentumTimeline();
    });
  }

  clearInterval(_overlayMomentumTimer);
  refreshOverlayMomentum();
  _overlayMomentumTimer = setInterval(refreshOverlayMomentum, 1000);
};

function loadMomentumFlowPrefs() {
  try {
    const raw = window.localStorage?.getItem(MFB_PREFS_KEY);
    if (!raw) return { widget: {}, host: {} };
    const parsed = JSON.parse(raw);
    return {
      widget: parsed && typeof parsed.widget === 'object' ? parsed.widget : {},
      host: parsed && typeof parsed.host === 'object' ? parsed.host : {},
    };
  } catch {
    return { widget: {}, host: {} };
  }
}

function saveMomentumFlowPrefs(widgetConfig, hostConfig) {
  try {
    window.localStorage?.setItem(MFB_PREFS_KEY, JSON.stringify({
      widget: widgetConfig,
      host: hostConfig,
    }));
  } catch {
    // Local overlay preferences are optional; failing to persist should not
    // interrupt live HUD rendering.
  }
}

function updateMomentumFlowPrefs(patch) {
  const prefs = loadMomentumFlowPrefs();
  const widgetConfig = { ...DEFAULT_MFB_CONFIG, ...prefs.widget, ...(patch.widget || {}) };
  const hostConfig = { ...DEFAULT_HOST_CONFIG, ...prefs.host, ...(patch.host || {}) };
  _momentumFlowBar?.setConfig(widgetConfig);
  applyHostConfig(hostConfig);
  syncMomentumFlowControls(widgetConfig, hostConfig);
  saveMomentumFlowPrefs(widgetConfig, hostConfig);
}

function applyHostConfig(config = DEFAULT_HOST_CONFIG) {
  const host = document.getElementById('overlay-widget-host');
  if (!host) return;
  const position = ['top-center', 'top-left', 'top-right', 'lower-center', 'side-left'].includes(config.position)
    ? config.position
    : DEFAULT_HOST_CONFIG.position;
  host.className = `overlay-widget-host overlay-widget-host--${position}`;
  host.style.setProperty('--mfb-host-opacity', String(clampRange(config.opacity, 0.35, 1, 1)));
  host.style.setProperty('--mfb-host-scale', String(clampRange(config.scale, 0.75, 1.25, 1)));
}

function syncMomentumFlowControls(widgetConfig, hostConfig) {
  setControlValue('mfb-variant', widgetConfig.variant);
  setControlValue('mfb-position', hostConfig.position);
  setControlValue('mfb-opacity', hostConfig.opacity);
  setControlValue('mfb-scale', hostConfig.scale);
  setControlChecked('mfb-show-confidence', widgetConfig.showConfidence);
  setControlChecked('mfb-show-labels', widgetConfig.showLabels);
  setControlChecked('mfb-show-percentages', widgetConfig.showPercentages);
  setControlChecked('mfb-pulse-enabled', widgetConfig.pulseEnabled);
}

function wireSelect(id, onChange) {
  const el = document.getElementById(id);
  if (!el || el.dataset.wired) return;
  el.dataset.wired = '1';
  el.addEventListener('change', () => onChange(el.value));
}

function wireRange(id, onChange) {
  const el = document.getElementById(id);
  if (!el || el.dataset.wired) return;
  el.dataset.wired = '1';
  el.addEventListener('input', () => onChange(el.value));
}

function wireCheck(id, onChange) {
  const el = document.getElementById(id);
  if (!el || el.dataset.wired) return;
  el.dataset.wired = '1';
  el.addEventListener('change', () => onChange(Boolean(el.checked)));
}

function setControlValue(id, value) {
  const el = document.getElementById(id);
  if (el && value !== undefined && value !== null) el.value = String(value);
}

function setControlChecked(id, checked) {
  const el = document.getElementById(id);
  if (el) el.checked = Boolean(checked);
}

async function refreshOverlayMomentum() {
  const root = document.getElementById('view-overlay');
  if (!root) return;
  if (Date.now() < _demoModeUntil) return;
  try {
    const out = await fetch('/api/overlay/momentum').then(r => r.json());
    _lastMomentumOutput = out;
    renderOverlayMomentum(out);
  } catch (err) {
    const debug = document.getElementById('momentum-debug');
    if (debug) debug.textContent = `Momentum Flow unavailable: ${err.message || err}`;
  }
}

function renderOverlayMomentum(out) {
  const replayAdjusted = withReplayDisplayState(out);
  const displayOut = replayAdjusted.skipHold
    ? replayAdjusted.output
    : withHeldDisplayState(replayAdjusted.output);
  _momentumFlowBar?.update(displayOut);
  updateLocalTimeline(displayOut);
  _lastDisplayOutput = displayOut;

  setText('momentum-state', displayOut.state || 'NEUTRAL');
  setText('momentum-confidence', pct(out.confidence));
  setText('momentum-volatility', pct(out.volatility));
  setText('momentum-pulse', out.overlay?.pulse || out.overlaySignals?.pulse || 'None');
  setText('momentum-pulse-team', out.overlay?.pulseTeam || out.overlaySignals?.pulseTeam || 'None');
  setText('momentum-blue-control', num(out.blue?.control));
  setText('momentum-blue-pressure', num(out.blue?.pressure));
  setText('momentum-orange-control', num(out.orange?.control));
  setText('momentum-orange-pressure', num(out.orange?.pressure));
  setText('momentum-last-strong', strongEventSummary(out.debug?.lastStrongEvent));

  const recent = Array.isArray(out.debug?.lastEvents) ? out.debug.lastEvents.slice(-10) : [];
  const reasons = Array.isArray(out.debug?.reasons) ? out.debug.reasons : [];
  const weights = Array.isArray(out.debug?.weightsApplied) ? out.debug.weightsApplied : [];
  const eventCounts = countSummary(out.debug?.eventCounts);
  const sourceCounts = countSummary(out.debug?.sourceCounts);
  const debug = document.getElementById('momentum-debug');
  if (debug) {
    const events = recent.map(ev => `${new Date(ev.time).toLocaleTimeString()}  ${ev.type}  ${ev.team}  ${ev.playerName || ev.playerId || ''}  [${ev.sourceEvent || 'unknown'}]`);
    const applied = weights.map(weightSummary);
    debug.textContent = [
      'Signal counts:',
      eventCounts || 'No normalized signals yet.',
      '',
      'Source counts:',
      sourceCounts || 'No source events yet.',
      '',
      'Recent normalized events:',
      events.length ? events.join('\n') : 'None yet.',
      '',
      'Reasons:',
      reasons.length ? reasons.join('\n') : 'No active weighting reasons.',
      '',
      'Weights applied:',
      applied.length ? applied.join('\n') : 'No active weights.',
    ].join('\n');
  }
}

function updateLocalTimeline(out) {
  const now = Date.now();
  if (!_timelineStart) _timelineStart = now;
  const sample = {
    at: now,
    state: out.state || 'NEUTRAL',
    confidence: clamp01(Number(out.confidence || 0)),
    bluePercent: Number(out.overlay?.momentumBarBluePercent ?? out.overlaySignals?.momentumBarBluePercent ?? 50),
    orangePercent: Number(out.overlay?.momentumBarOrangePercent ?? out.overlaySignals?.momentumBarOrangePercent ?? 50),
  };
  const last = _timelineSamples[_timelineSamples.length - 1];
  if (!last || last.state !== sample.state || Math.abs(last.bluePercent - sample.bluePercent) >= 6 || now - last.at >= 3000) {
    _timelineSamples.push(sample);
    if (_timelineSamples.length > MAX_TIMELINE_SAMPLES) _timelineSamples.shift();
  }

  const ev = out.debug?.lastStrongEvent;
  if (ev && ev.time) {
    const key = `${ev.type}:${ev.team}:${ev.playerId || ev.playerName || ''}:${ev.time}`;
    if (key !== _lastTimelineEventKey) {
      _lastTimelineEventKey = key;
      _timelineEvents.push({
        at: now,
        type: ev.type || 'event',
        team: ev.team || '',
        player: ev.playerName || ev.playerId || 'Unknown',
      });
      if (_timelineEvents.length > MAX_TIMELINE_EVENTS) _timelineEvents.shift();
    }
  }

  renderMomentumTimeline();
}

function renderMomentumTimeline() {
  const bar = document.getElementById('momentum-track-bar');
  const markers = document.getElementById('momentum-markers');
  const feed = document.getElementById('momentum-event-feed');
  if (!bar || !markers || !feed) return;

  if (!_timelineSamples.length) {
    bar.innerHTML = '<div class="momentum-track-segment momentum-track-segment--neutral" style="width:100%"></div>';
    markers.innerHTML = '';
  } else {
    const total = Math.max(1, _timelineSamples[_timelineSamples.length - 1].at - _timelineSamples[0].at);
    bar.innerHTML = _timelineSamples.map((sample, index) => {
      const next = _timelineSamples[index + 1];
      const duration = Math.max(450, (next ? next.at : Date.now()) - sample.at);
      const width = Math.max(2, (duration / total) * 100);
      const state = displayState(sample.state);
      const opacity = 0.35 + sample.confidence * 0.65;
      return `<div class="momentum-track-segment momentum-track-segment--${state.kind}" title="${escapeHtml(state.label)}" style="width:${width.toFixed(2)}%; opacity:${opacity.toFixed(2)}"></div>`;
    }).join('');

    const span = Math.max(1, Date.now() - _timelineSamples[0].at);
    const visibleMarkers = visibleTimelineMarkers(_timelineEvents);
    markers.innerHTML = visibleMarkers.map(event => {
      const left = clampPercent(((event.at - _timelineSamples[0].at) / span) * 100);
      const label = event.label || displayEvent(event.type);
      const teamClass = event.team === 'orange' ? 'momentum-marker--orange' : event.team === 'blue' ? 'momentum-marker--blue' : '';
      const goalClass = event.type === 'goal' ? 'momentum-marker--goal' : '';
      const typeClass = `momentum-marker--${event.type || 'event'}`;
      const details = event.related?.length ? ` (${event.related.map(displayEvent).join(', ')})` : '';
      return `<div class="momentum-marker ${teamClass} ${goalClass} ${typeClass}" style="left:${left.toFixed(2)}%" title="${escapeHtml(label)}${details} - ${escapeHtml(event.player)}">${escapeHtml(event.shortLabel || shortEventLabel(event.type))}</div>`;
    }).join('');
  }

  const visibleFeedEvents = visibleTimelineMarkers(_timelineEvents);
  if (!visibleFeedEvents.length) {
    feed.innerHTML = '<div class="momentum-empty">Waiting for key goals, saves, assists, demos, and pressure shots...</div>';
  } else {
    feed.innerHTML = visibleFeedEvents.slice().reverse().map(event => {
      const time = new Date(event.at).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', second: '2-digit' });
      const label = event.label || displayEvent(event.type);
      const team = displayTeam(event.team);
      return `<div class="momentum-event-row"><span>${escapeHtml(time)}</span><strong>${escapeHtml(label)}</strong><span>${escapeHtml(team)}${team ? ' - ' : ''}${escapeHtml(event.player)}</span></div>`;
    }).join('');
  }
}

function visibleTimelineMarkers(events) {
  if (!events.length) return [];
  const clusters = [];
  for (const event of events) {
    const previous = clusters[clusters.length - 1];
    if (previous && event.at - previous.endAt <= TIMELINE_MARKER_CLUSTER_MS && relatedTimelineEvents(previous.events, event)) {
      previous.events.push(event);
      previous.endAt = event.at;
      continue;
    }
    clusters.push({ events: [event], endAt: event.at });
  }
  return clusters
    .map(cluster => markerFromCluster(cluster.events))
    .filter(Boolean);
}

function relatedTimelineEvents(cluster, event) {
  if (!cluster.length) return false;
  if (event.type === 'goal' || event.type === 'save' || event.type === 'assist') return true;
  return cluster.some(existing => existing.team === event.team || markerPriority(existing.type) >= markerPriority('save'));
}

function markerFromCluster(cluster) {
  const sorted = cluster.slice().sort((a, b) => markerPriority(b.type) - markerPriority(a.type));
  const primary = sorted[0];
  if (!primary) return null;

  const related = uniqueEventTypes(cluster)
    .filter(type => type !== primary.type && markerPriority(type) >= markerPriority('demo'));
  if (primary.type === 'shot' && !cluster.some(event => event.type === 'demo' || event.type === 'save' || event.type === 'goal')) {
    const recentSameTeamShots = cluster.filter(event => event.type === 'shot' && event.team === primary.team);
    if (recentSameTeamShots.length < 2) return null;
  }

  const hasDemo = cluster.some(event => event.type === 'demo' && event.team === primary.team);
  const label = markerDisplayLabel(primary.type, hasDemo, related);
  const shortLabel = markerShortLabel(primary.type, hasDemo, related);
  return {
    ...primary,
    label,
    shortLabel,
    related,
  };
}

function markerPriority(type) {
  switch (type) {
    case 'goal': return 100;
    case 'save': return 80;
    case 'assist': return 70;
    case 'demo': return 55;
    case 'shot': return 45;
    default: return 0;
  }
}

function uniqueEventTypes(events) {
  const seen = new Set();
  const out = [];
  for (const event of events) {
    if (seen.has(event.type)) continue;
    seen.add(event.type);
    out.push(event.type);
  }
  return out;
}

function markerDisplayLabel(type, hasDemo, related) {
  if (type === 'goal' && related.includes('assist')) return 'Assisted Goal';
  if (type === 'goal') return 'Goal';
  if (type === 'save') return 'Save';
  if (type === 'assist') return 'Assist';
  if (type === 'demo' && related.includes('shot')) return 'Demo Pressure';
  if (type === 'demo') return 'Demo';
  if (type === 'shot' && hasDemo) return 'Demo + Shot';
  if (type === 'shot') return 'Pressure Shot';
  return 'Event';
}

function markerShortLabel(type, hasDemo, related) {
  if (type === 'goal' && related.includes('assist')) return 'G+A';
  if (type === 'goal') return 'G';
  if (type === 'save') return 'SV';
  if (type === 'assist') return 'A';
  if (type === 'demo' && related.includes('shot')) return 'DM+SH';
  if (type === 'demo') return 'DM';
  if (type === 'shot' && hasDemo) return 'DM+SH';
  if (type === 'shot') return 'SH';
  return 'E';
}

function resetLocalTimeline() {
  _timelineStart = 0;
  _timelineSamples = [];
  _timelineEvents = [];
  _lastTimelineEventKey = '';
  _heldDisplayState = null;
  _lastDisplayOutput = null;
  _replayFrozenOutput = null;
  _wasReplayActive = false;
  _postReplayNeutralStarted = 0;
  _postReplayNeutralUntil = 0;
  _postReplaySignalAfter = 0;
  _awaitingPostReplayLiveSignal = false;
  _lastGoalFallbackKey = '';
  _goalFallbackStarted = 0;
  _lastMomentumResetAt = 0;
}

function withReplayDisplayState(out = {}) {
  const now = Date.now();
  const resetAt = Number(out.display?.momentumResetAt || 0);
  if (resetAt && resetAt !== _lastMomentumResetAt) {
    _lastMomentumResetAt = resetAt;
    _replayFrozenOutput = null;
    _wasReplayActive = false;
    _postReplayNeutralStarted = 0;
    _postReplayNeutralUntil = 0;
    _postReplaySignalAfter = 0;
    _awaitingPostReplayLiveSignal = false;
    _goalFallbackStarted = 0;
    _heldDisplayState = null;
    _lastDisplayDominance = null;
    return { output: neutralizedDisplayOutput(out), skipHold: true };
  }

  const replayActive = Boolean(out.display?.replayActive);
  const replayFileMode = Boolean(out.display?.replayFileMode);

  if (replayActive && replayFileMode) {
    _wasReplayActive = false;
    _awaitingPostReplayLiveSignal = false;
    _postReplayNeutralStarted = 0;
    _postReplayNeutralUntil = 0;
    _postReplaySignalAfter = 0;
    _goalFallbackStarted = 0;
    _replayFrozenOutput = null;
    return { output: out, skipHold: false };
  }

  if (replayActive) {
    _wasReplayActive = true;
    _awaitingPostReplayLiveSignal = false;
    _postReplayNeutralStarted = 0;
    _postReplayNeutralUntil = 0;
    _postReplaySignalAfter = 0;
    _goalFallbackStarted = 0;
    if (!_replayFrozenOutput && _lastDisplayOutput && isHoldableDisplayState(_lastDisplayOutput.state)) {
      _replayFrozenOutput = cloneDisplayOutput(_lastDisplayOutput);
    }
    if (_replayFrozenOutput) {
      return { output: {
        ...out,
        ...cloneDisplayOutput(_replayFrozenOutput),
        debug: out.debug,
        display: out.display,
      }, skipHold: true };
    }
    return { output: out, skipHold: true };
  }

  if (_wasReplayActive) {
    _wasReplayActive = false;
    _replayFrozenOutput = null;
    _postReplayNeutralStarted = now;
    _postReplayNeutralUntil = now + POST_REPLAY_MAX_NEUTRAL_MS;
    _postReplaySignalAfter = Number(out.display?.replayChanged || now);
    _awaitingPostReplayLiveSignal = true;
    _heldDisplayState = null;
    _lastDisplayDominance = null;
  }

  const goalKey = goalFallbackKey(out);
  if (goalKey && goalKey !== _lastGoalFallbackKey) {
    _lastGoalFallbackKey = goalKey;
    _goalFallbackStarted = now;
  }

  if (!_awaitingPostReplayLiveSignal && _goalFallbackStarted && now - _goalFallbackStarted >= GOAL_FALLBACK_RESET_MS) {
    _goalFallbackStarted = 0;
    _replayFrozenOutput = null;
    _heldDisplayState = null;
    _lastDisplayDominance = null;
    _postReplayNeutralStarted = now;
    _postReplayNeutralUntil = now + POST_REPLAY_MAX_NEUTRAL_MS;
    _postReplaySignalAfter = now;
    _awaitingPostReplayLiveSignal = true;
  }

  if (_awaitingPostReplayLiveSignal) {
    const minNeutralSatisfied = now - _postReplayNeutralStarted >= POST_REPLAY_MIN_NEUTRAL_MS;
    if (minNeutralSatisfied && hasLiveSignalAfterReplay(out)) {
      _awaitingPostReplayLiveSignal = false;
      _postReplayNeutralStarted = 0;
      _postReplayNeutralUntil = 0;
      _postReplaySignalAfter = 0;
    } else if (now <= _postReplayNeutralUntil) {
      return { output: neutralizedDisplayOutput(out), skipHold: true };
    } else {
      _awaitingPostReplayLiveSignal = false;
      _postReplayNeutralStarted = 0;
      _postReplayNeutralUntil = 0;
      _postReplaySignalAfter = 0;
    }
  }

  return { output: out, skipHold: false };
}

function cloneDisplayOutput(out) {
  return {
    ...out,
    blue: { ...(out.blue || {}) },
    orange: { ...(out.orange || {}) },
    overlay: { ...(out.overlay || out.overlaySignals || {}) },
  };
}

function neutralizedDisplayOutput(out = {}) {
  return {
    ...out,
    state: 'NEUTRAL',
    confidence: 0,
    volatility: 0,
    overlay: {
      ...(out.overlay || {}),
      momentumBarBluePercent: 50,
      momentumBarOrangePercent: 50,
      pulse: '',
      pulseTeam: '',
    },
    overlaySignals: {
      ...(out.overlaySignals || {}),
      momentumBarBluePercent: 50,
      momentumBarOrangePercent: 50,
      pulse: '',
      pulseTeam: '',
    },
  };
}

function hasLiveSignalAfterReplay(out = {}) {
  const changed = Number(_postReplaySignalAfter || out.display?.replayChanged || 0);
  const recent = Array.isArray(out.debug?.lastEvents) ? out.debug.lastEvents : [];
  return recent.some(ev => {
    const t = Number(ev.time || 0);
    return t >= changed && ev.type === 'ball_hit';
  });
}

function goalFallbackKey(out = {}) {
  const ev = out.debug?.lastStrongEvent;
  if (ev?.type === 'goal' && ev.time) {
    return `goal:${ev.team || ''}:${ev.playerId || ev.playerName || ''}:${ev.time}`;
  }
  const signals = out.overlaySignals || out.overlay || {};
  if (signals.pulse === 'GOAL_BURST') {
    return `goal:pulse:${signals.pulseTeam || ''}:${out.display?.replayChanged || ''}:${Math.floor(Date.now() / GOAL_FALLBACK_RESET_MS)}`;
  }
  return '';
}

function withHeldDisplayState(out = {}) {
  const now = Date.now();
  const state = deriveDisplayState(out, now);
  const confidence = clamp01(Number(out.confidence || 0));
  const holdable = isHoldableDisplayState(state);
  const displayOut = { ...out, state };

  if (holdable) {
    const holdMs = DISPLAY_STATE_MIN_HOLD_MS + Math.round(confidence * (DISPLAY_STATE_MAX_HOLD_MS - DISPLAY_STATE_MIN_HOLD_MS));
    _heldDisplayState = {
      state,
      until: now + holdMs,
      confidence,
    };
    return displayOut;
  }

  if (state === 'NEUTRAL' && _heldDisplayState && now <= _heldDisplayState.until && _heldDisplayState.confidence >= 0.25) {
    return { ...displayOut, state: _heldDisplayState.state };
  }

  if (state === 'NEUTRAL' || state === 'VOLATILE') {
    _heldDisplayState = null;
  }
  return displayOut;
}

function deriveDisplayState(out = {}, now = Date.now()) {
  const signals = out.overlaySignals || out.overlay || {};
  let blue = Number(signals.momentumBarBluePercent);
  let orange = Number(signals.momentumBarOrangePercent);
  if (!Number.isFinite(blue) || !Number.isFinite(orange)) {
    const adapted = adaptMomentumFlowOutput(out);
    blue = adapted.bluePercent;
    orange = adapted.orangePercent;
  }
  blue = clampPercent(blue);
  orange = clampPercent(orange);

  const spread = Math.abs(blue - orange);
  const confidence = clamp01(Number(out.confidence || 0));
  const dominant = blue >= orange ? 'blue' : 'orange';
  const dominantPercent = Math.max(blue, orange);
  const backendState = out.state || 'NEUTRAL';
  const recentFlip = _lastDisplayDominance
    && _lastDisplayDominance.team !== dominant
    && now - _lastDisplayDominance.at <= DISPLAY_CONTESTED_FLIP_MS;

  if (backendState === 'VOLATILE' || spread <= DISPLAY_CONTESTED_SPREAD_PERCENT || (recentFlip && dominantPercent < DISPLAY_PRESSURE_PERCENT)) {
    _lastDisplayDominance = null;
    return 'VOLATILE';
  }

  if (dominantPercent >= DISPLAY_CONTROL_PERCENT) {
    _lastDisplayDominance = { team: dominant, at: now };
  }

  if (dominantPercent >= DISPLAY_PRESSURE_PERCENT && confidence >= 0.35) {
    return dominant === 'blue' ? 'BLUE_PRESSURE' : 'ORANGE_PRESSURE';
  }

  if (dominantPercent >= DISPLAY_CONTROL_PERCENT && confidence >= 0.22) {
    return dominant === 'blue' ? 'BLUE_CONTROL' : 'ORANGE_CONTROL';
  }

  return 'NEUTRAL';
}

function isHoldableDisplayState(state) {
  return state === 'BLUE_CONTROL'
    || state === 'ORANGE_CONTROL'
    || state === 'BLUE_PRESSURE'
    || state === 'ORANGE_PRESSURE';
}

function adaptMomentumFlowOutput(out = {}) {
  const signals = out.overlaySignals || out.overlay || {};
  let bluePercent = Number(signals.momentumBarBluePercent);
  let orangePercent = Number(signals.momentumBarOrangePercent);
  if (!Number.isFinite(bluePercent) || !Number.isFinite(orangePercent)) {
    const blueShare = Number(out.blue?.pressureShare);
    const orangeShare = Number(out.orange?.pressureShare);
    if (Number.isFinite(blueShare) && Number.isFinite(orangeShare) && blueShare + orangeShare > 0.001) {
      bluePercent = blueShare * 100;
      orangePercent = orangeShare * 100;
    } else {
      bluePercent = 50;
      orangePercent = 50;
    }
  }
  bluePercent = clampPercent(bluePercent);
  orangePercent = clampPercent(orangePercent);
  const total = bluePercent + orangePercent;
  if (total > 0.001) {
    bluePercent = (bluePercent / total) * 100;
    orangePercent = 100 - bluePercent;
  } else {
    bluePercent = 50;
    orangePercent = 50;
  }

  return {
    state: out.state || 'NEUTRAL',
    confidence: clamp01(Number(out.confidence || 0)),
    volatility: clamp01(Number(out.volatility || 0)),
    bluePercent,
    orangePercent,
    pulse: signals.pulse || '',
    pulseTeam: signals.pulseTeam || '',
    pulseKey: pulseIdentity(out, signals),
  };
}

function stateClass(state) {
  switch (state) {
    case 'BLUE_PRESSURE': return 'is-blue-pressure';
    case 'ORANGE_PRESSURE': return 'is-orange-pressure';
    case 'BLUE_CONTROL': return 'is-blue-control';
    case 'ORANGE_CONTROL': return 'is-orange-control';
    case 'VOLATILE': return 'is-volatile';
    default: return 'is-neutral';
  }
}

function leftStateLabel(state) {
  switch (state) {
    case 'BLUE_PRESSURE': return 'BLUE PRESSURE';
    case 'BLUE_CONTROL': return 'BLUE CONTROL';
    case 'VOLATILE': return 'VOLATILE';
    case 'NEUTRAL': return 'NEUTRAL';
    default: return '';
  }
}

function rightStateLabel(state) {
  switch (state) {
    case 'ORANGE_PRESSURE': return 'ORANGE PRESSURE';
    case 'ORANGE_CONTROL': return 'ORANGE CONTROL';
    case 'VOLATILE': return 'VOLATILE';
    case 'NEUTRAL': return 'NEUTRAL';
    default: return '';
  }
}

function displayState(state) {
  switch (state) {
    case 'BLUE_PRESSURE':
      return { label: 'Blue Pressure', kind: 'blue' };
    case 'ORANGE_PRESSURE':
      return { label: 'Orange Pressure', kind: 'orange' };
    case 'VOLATILE':
      return { label: 'Contested', kind: 'contested' };
    case 'BLUE_CONTROL':
      return { label: 'Blue Momentum', kind: 'blue' };
    case 'ORANGE_CONTROL':
      return { label: 'Orange Momentum', kind: 'orange' };
    default:
      return { label: 'Neutral', kind: 'neutral' };
  }
}

function displayEvent(type) {
  switch (type) {
    case 'goal': return 'Goal';
    case 'shot': return 'Shot';
    case 'save': return 'Save';
    case 'demo': return 'Demo';
    case 'assist': return 'Assist';
    default: return 'Event';
  }
}

function shortEventLabel(type) {
  switch (type) {
    case 'goal': return 'G';
    case 'shot': return 'SH';
    case 'save': return 'SV';
    case 'demo': return 'DM';
    case 'assist': return 'A';
    default: return 'E';
  }
}

function displayTeam(team) {
  if (team === 'blue') return 'Blue';
  if (team === 'orange') return 'Orange';
  return '';
}

function demoOutput(kind) {
  switch (kind) {
    case 'blue':
      return { state: 'BLUE_PRESSURE', confidence: 0.74, volatility: 0.22, blue: { pressureShare: 0.62 }, orange: { pressureShare: 0.38 }, overlay: { momentumBarBluePercent: 62, momentumBarOrangePercent: 38, pulse: 'SHOT', pulseTeam: 'blue' } };
    case 'orange':
      return { state: 'ORANGE_PRESSURE', confidence: 0.81, volatility: 0.26, blue: { pressureShare: 0.28 }, orange: { pressureShare: 0.72 }, overlay: { momentumBarBluePercent: 28, momentumBarOrangePercent: 72, pulse: 'SAVE_FORCED', pulseTeam: 'orange' } };
    case 'volatile':
      return { state: 'VOLATILE', confidence: 0.58, volatility: 0.82, blue: { pressureShare: 0.55 }, orange: { pressureShare: 0.45 }, overlay: { momentumBarBluePercent: 55, momentumBarOrangePercent: 45, pulse: 'VOLATILE_CONTEST' } };
    default:
      return { state: 'NEUTRAL', confidence: 0.42, volatility: 0.18, blue: { pressureShare: 0.5 }, orange: { pressureShare: 0.5 }, overlay: { momentumBarBluePercent: 50, momentumBarOrangePercent: 50 } };
  }
}

function weightSummary(w) {
  const deltas = [];
  if (w.controlDelta !== undefined) deltas.push(`control ${signed(w.controlDelta)}`);
  if (w.pressureDelta !== undefined) deltas.push(`pressure ${signed(w.pressureDelta)}`);
  if (w.volatilityDelta !== undefined) deltas.push(`volatility ${signed(w.volatilityDelta)}`);
  const team = w.team ? ` ${w.team}` : '';
  return `${w.eventType || 'event'}${team}: ${deltas.join(', ') || 'no delta'} - ${w.reason || 'no reason'}`;
}

function strongEventSummary(ev) {
  if (!ev) return 'None';
  const who = ev.playerName || ev.playerId || 'unknown';
  return `${ev.type || 'event'} / ${ev.team || 'team?'} / ${who}`;
}

function countSummary(counts) {
  if (!counts || typeof counts !== 'object') return '';
  return Object.keys(counts)
    .sort()
    .map(key => `${key}: ${counts[key]}`)
    .join('  |  ');
}

function pulseIdentity(out, signals) {
  if (!signals.pulse) return '';
  const ev = out.debug?.lastStrongEvent;
  if (ev && ev.time) {
    return `${signals.pulse}:${ev.type || ''}:${ev.team || ''}:${ev.time}`;
  }
  return `${signals.pulse}:${out.state || ''}:${signals.pulseTeam || ''}`;
}

function signed(value) {
  const n = Number(value || 0);
  return `${n >= 0 ? '+' : ''}${n.toFixed(2)}`;
}

function setText(id, value) {
  const el = document.getElementById(id);
  if (el) el.textContent = value;
}

function pct(value) {
  const n = Number(value || 0);
  return `${Math.round(n * 100)}%`;
}

function num(value) {
  const n = Number(value || 0);
  return n.toFixed(2);
}

function clampPercent(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return 50;
  return Math.max(0, Math.min(100, n));
}

function clamp01(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return 0;
  return Math.max(0, Math.min(1, n));
}

function clampRange(value, min, max, fallback) {
  const n = Number(value);
  if (!Number.isFinite(n)) return fallback;
  return Math.max(min, Math.min(max, n));
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}
