'use strict';

const state = {
  data: null,
  selectedSegmentId: '',
  selectedEventId: '',
  drawerOpen: true,
  activeMomentKey: '',
};

const els = {
  summary: document.getElementById('summary-strip'),
  scale: document.getElementById('scale-row'),
  markers: document.getElementById('marker-row'),
  timeline: document.getElementById('timeline-bar'),
  drawer: document.getElementById('detail-drawer'),
  players: document.getElementById('player-summary'),
  clear: document.getElementById('clear-selection'),
};

const EVENT_LABELS = {
  goal: 'Goal',
  assist: 'Assist',
  save: 'Save',
  epic_save: 'Epic save',
  demo: 'Demo',
  shot: 'Shot',
  momentum_swing: 'Momentum swing',
};

const EVENT_SIGNS = {
  goal: 'G',
  assist: 'A',
  save: 'S',
  epic_save: 'ES',
  demo: 'D',
  shot: 'Sh',
  momentum_swing: 'MS',
};

const EVENT_PRIORITY = {
  goal: 60,
  epic_save: 50,
  save: 50,
  demo: 40,
  assist: 30,
  shot: 20,
  momentum_swing: 10,
};

const MAX_MARKER_LANES = 4;
const MIN_MARKER_GAP_PERCENT = 4.2;

function esc(value) {
  return String(value ?? '').replace(/[&<>"']/g, ch => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  }[ch]));
}

function clampPercent(value) {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, value));
}

function formatTime(seconds) {
  if (!Number.isFinite(seconds)) return '--:--';
  const m = Math.floor(Math.max(0, seconds) / 60);
  const s = Math.floor(Math.max(0, seconds) % 60);
  return `${m}:${String(s).padStart(2, '0')}`;
}

function formatRange(segment) {
  if (!segment) return '--:--';
  return `${formatTime(segment.startSecond)}-${formatTime(segment.endSecond)}`;
}

function formatPct(value) {
  if (!Number.isFinite(value)) return 'Fixture missing';
  return `${Math.round(value * 100)}%`;
}

function formatSafeSummary(value, fallback) {
  const text = String(value ?? '').trim();
  return text || fallback;
}

function stateLabel(segment) {
  if (!segment) return 'Unknown';
  const stateName = segment.state ? segment.state.replace('_', ' ') : 'unknown';
  const teamName = segment.team && segment.team !== 'none' ? `${segment.team} ` : '';
  return `${teamName}${stateName}`;
}

function selectionLabel() {
  if (state.selectedEventId) return 'Event marker selected';
  if (state.selectedSegmentId) return 'Pressure sequence selected';
  return 'No selection';
}

function segmentClass(segment) {
  const team = segment.team || 'none';
  const kind = segment.state || 'neutral';
  if (kind === 'contested') return 'segment contested';
  if (kind === 'neutral') return 'segment neutral';
  if (team === 'blue') return 'segment blue';
  if (team === 'orange') return 'segment orange';
  return 'segment neutral';
}

function eventPriority(event) {
  return EVENT_PRIORITY[event?.type] || 0;
}

function eventLabel(event) {
  return EVENT_LABELS[event?.type] || event?.label || 'Event';
}

function eventsForSegment(segmentId) {
  return (state.data?.events || []).filter(event => event.segmentId === segmentId);
}

function segmentForEvent(event) {
  const byId = (state.data?.segments || []).find(segment => segment.id === event.segmentId);
  if (byId) return byId;
  return (state.data?.segments || []).find(segment =>
    event.second >= segment.startSecond && event.second <= segment.endSecond
  );
}

function chronologicalMoments() {
  const segments = (state.data?.segments || []).map(segment => ({
    key: `segment:${segment.id}`,
    type: 'segment',
    id: segment.id,
    second: Number.isFinite(segment.startSecond) ? segment.startSecond : 0,
    priority: 0,
  }));
  const events = (state.data?.events || []).map(event => ({
    key: `event:${event.id}`,
    type: 'event',
    id: event.id,
    second: Number.isFinite(event.second) ? event.second : 0,
    priority: eventPriority(event),
  }));
  return [...segments, ...events].sort((a, b) =>
    a.second - b.second ||
    b.priority - a.priority ||
    a.key.localeCompare(b.key)
  );
}

function activeMomentIndex() {
  const moments = chronologicalMoments();
  if (!moments.length) return -1;
  const key = state.activeMomentKey ||
    (state.selectedEventId ? `event:${state.selectedEventId}` : '') ||
    (state.selectedSegmentId ? `segment:${state.selectedSegmentId}` : '');
  return moments.findIndex(moment => moment.key === key);
}

function focusMoment(moment) {
  requestAnimationFrame(() => {
    const safeId = typeof CSS !== 'undefined' && CSS.escape
      ? CSS.escape(moment.id)
      : String(moment.id).replace(/["\\]/g, '\\$&');
    const selector = moment.type === 'event'
      ? `[data-event-id="${safeId}"]`
      : `[data-segment-id="${safeId}"]`;
    document.querySelector(selector)?.focus();
  });
}

function selectMoment(moment, focusAfterSelect) {
  if (!moment) return;
  if (moment.type === 'event') {
    selectEvent(moment.id, focusAfterSelect);
  } else {
    selectSegment(moment.id, focusAfterSelect);
  }
}

function moveSelection(delta) {
  const moments = chronologicalMoments();
  if (!moments.length) return;
  const current = activeMomentIndex();
  const next = current < 0
    ? (delta > 0 ? 0 : moments.length - 1)
    : Math.max(0, Math.min(moments.length - 1, current + delta));
  selectMoment(moments[next], true);
}

function markerLayout(events, duration) {
  const sorted = [...events].sort((a, b) =>
    (a.second || 0) - (b.second || 0) ||
    eventPriority(b) - eventPriority(a) ||
    String(a.id).localeCompare(String(b.id))
  );
  const bySecond = new Map();
  for (const event of sorted) {
    const secondKey = String(Number.isFinite(event.second) ? event.second : 0);
    if (!bySecond.has(secondKey)) bySecond.set(secondKey, []);
    bySecond.get(secondKey).push(event);
  }

  const laneEnds = Array.from({ length: MAX_MARKER_LANES }, () => -Infinity);
  const out = new Map();
  for (const event of sorted) {
    const sameSecond = bySecond.get(String(Number.isFinite(event.second) ? event.second : 0)) || [];
    const sameSecondSorted = [...sameSecond].sort((a, b) =>
      eventPriority(b) - eventPriority(a) ||
      String(a.id).localeCompare(String(b.id))
    );
    const sameSecondIndex = sameSecondSorted.findIndex(item => item.id === event.id);
    const left = clampPercent(((event.second || 0) / duration) * 100);
    let lane = Math.max(0, sameSecondIndex);
    let offset = 0;

    if (sameSecond.length > 1) {
      lane = sameSecondIndex % MAX_MARKER_LANES;
      offset = sameSecondIndex === 0 ? 0 : (sameSecondIndex % 2 === 0 ? -7 : 7) * Math.ceil(sameSecondIndex / 2);
    } else {
      lane = laneEnds.findIndex(end => left - end >= MIN_MARKER_GAP_PERCENT);
      if (lane < 0) lane = sorted.indexOf(event) % MAX_MARKER_LANES;
    }

    laneEnds[lane] = left;
    out.set(event.id, {
      lane,
      left,
      offset,
      priority: eventPriority(event),
    });
  }
  return out;
}

function renderSummary() {
  const data = state.data;
  if (!data?.match) {
    els.summary.innerHTML = '<div class="empty">Fixture match metadata is unavailable.</div>';
    return;
  }
  const totals = data.totals || {};
  const topPlayer = (data.playerContributions || [])
    .filter(player => Number.isFinite(player.pressureContribution))
    .sort((a, b) => b.pressureContribution - a.pressureContribution)[0];
  const swing = (data.events || []).find(event => event.type === 'momentum_swing');
  const cards = [
    ['Preview source', data.match.source || 'fixture'],
    [`${data.match.blueName || 'Blue'} pressure/control`, `${Number.isFinite(totals.bluePressureSeconds) ? formatTime(totals.bluePressureSeconds) : 'Fixture missing'} / ${Number.isFinite(totals.blueControlSeconds) ? formatTime(totals.blueControlSeconds) : 'Fixture missing'}`],
    [`${data.match.orangeName || 'Orange'} pressure/control`, `${Number.isFinite(totals.orangePressureSeconds) ? formatTime(totals.orangePressureSeconds) : 'Fixture missing'} / ${Number.isFinite(totals.orangeControlSeconds) ? formatTime(totals.orangeControlSeconds) : 'Fixture missing'}`],
    ['Contested / neutral', `${formatTime(totals.contestedSeconds)} / ${formatTime(totals.neutralSeconds)}`],
    ['Top pressure contribution', topPlayer ? `${topPlayer.playerName} ${formatPct(topPlayer.pressureContribution)}` : 'Fixture missing'],
    ['Momentum swing sample', swing ? `${formatTime(swing.second)} ${swing.team || ''}` : 'Fixture missing'],
  ];
  els.summary.innerHTML = cards.map(([label, value]) => `
    <article class="summary-card">
      <span>${esc(label)}</span>
      <strong>${esc(value)}</strong>
    </article>
  `).join('');
}

function renderScale() {
  const duration = state.data?.match?.durationSeconds || 300;
  const marks = [0, duration * 0.25, duration * 0.5, duration * 0.75, duration];
  els.scale.innerHTML = marks.map(second => `<span>${formatTime(second)}</span>`).join('');
}

function renderTimeline() {
  const duration = state.data?.match?.durationSeconds || 1;
  const segments = state.data?.segments || [];
  if (!segments.length) {
    els.timeline.innerHTML = '<div class="empty">No fixture segments available.</div>';
    return;
  }
  els.timeline.innerHTML = segments.map(segment => {
    const start = clampPercent((segment.startSecond / duration) * 100);
    const width = clampPercent(((segment.endSecond - segment.startSecond) / duration) * 100);
    const intensity = 0.35 + Math.max(0, Math.min(1, segment.confidence || 0)) * 0.55;
    const selected = segment.id === state.selectedSegmentId ? ' selected' : '';
    const eventContext = state.selectedEventId && segment.id === state.selectedSegmentId ? ' event-context' : '';
    const label = stateLabel(segment);
    return `<button
      class="${segmentClass(segment)}${selected}${eventContext}"
      type="button"
      data-segment-id="${esc(segment.id)}"
      data-moment-key="segment:${esc(segment.id)}"
      style="left:${start}%;width:${width}%;opacity:${intensity.toFixed(2)}"
      aria-label="${esc(formatRange(segment))} ${esc(label)}"
      aria-pressed="${segment.id === state.selectedSegmentId && !state.selectedEventId ? 'true' : 'false'}">
      <span>${esc(label)}</span>
    </button>`;
  }).join('');

  els.timeline.querySelectorAll('[data-segment-id]').forEach(button => {
    button.addEventListener('click', () => selectSegment(button.dataset.segmentId, false));
  });
}

function renderMarkers() {
  const duration = state.data?.match?.durationSeconds || 1;
  const events = state.data?.events || [];
  if (!events.length) {
    els.markers.innerHTML = '<div class="empty">No fixture event markers available.</div>';
    return;
  }
  const layout = markerLayout(events, duration);
  els.markers.innerHTML = events.map(event => {
    const marker = layout.get(event.id) || { left: 0, lane: 0, offset: 0, priority: 0 };
    const selected = event.id === state.selectedEventId ? ' selected' : '';
    const label = eventLabel(event);
    return `<button
      class="event-marker ${esc(event.team || 'none')}${selected}"
      type="button"
      data-event-id="${esc(event.id)}"
      data-moment-key="event:${esc(event.id)}"
      style="left:${marker.left}%;--lane:${marker.lane};--offset:${marker.offset}px;--priority:${marker.priority}"
      title="${esc(formatTime(event.second))} ${esc(label)}"
      aria-label="${esc(formatTime(event.second))} ${esc(label)} ${esc(event.playerName || event.team || 'Fixture marker')}"
      aria-pressed="${event.id === state.selectedEventId ? 'true' : 'false'}">
      <span>${esc(EVENT_SIGNS[event.type] || label.slice(0, 2))}</span>
    </button>`;
  }).join('');

  els.markers.querySelectorAll('[data-event-id]').forEach(button => {
    button.addEventListener('click', () => selectEvent(button.dataset.eventId, false));
  });
}

function renderDrawerEmpty() {
  els.drawer.innerHTML = `
    <div class="panel-head">
      <h2>Selected moment</h2>
      <p>${selectionLabel()}</p>
    </div>
    <div class="empty drawer-empty">
      No timeline moment selected. Choose a segment or event marker to inspect the fixture narrative.
    </div>
    <button class="replay-placeholder" type="button" disabled>Replay Studio integration later</button>
  `;
}

function drawerToggle() {
  return `<button class="drawer-toggle" type="button" data-drawer-toggle>${state.drawerOpen ? 'Collapse details' : 'Expand details'}</button>`;
}

function bindDrawerToggle() {
  const toggle = els.drawer.querySelector('[data-drawer-toggle]');
  if (!toggle) return;
  toggle.addEventListener('click', () => {
    state.drawerOpen = !state.drawerOpen;
    rerenderInteractive();
  });
}

function renderEventList(events) {
  if (!events.length) return '<div class="empty">No notable fixture events in this window.</div>';
  const sorted = [...events].sort((a, b) =>
    (a.second || 0) - (b.second || 0) ||
    eventPriority(b) - eventPriority(a) ||
    String(a.id).localeCompare(String(b.id))
  );
  return `<ul class="event-list">${sorted.map(event => `
    <li>
      <span class="event-time">${formatTime(event.second)}</span>
      <strong>${esc(eventLabel(event))}</strong>
      <span>${esc(event.playerName || event.team || 'Fixture marker')}</span>
      <p>${esc(formatSafeSummary(event.summary, 'Fixture event summary unavailable.'))}</p>
    </li>
  `).join('')}</ul>`;
}

function renderContributionHighlights(segment) {
  const names = new Set(segment?.involvedPlayers || []);
  const players = (state.data?.playerContributions || []).filter(player => names.has(player.playerName));
  if (!players.length) return '<div class="empty">No segment-specific fixture player list available. Match-wide summary remains below.</div>';
  return `<div class="mini-contribs">${players.map(player => `
    <div>
      <strong>${esc(player.playerName)}</strong>
      <span>pressure contribution ${esc(formatPct(player.pressureContribution))}</span>
    </div>
  `).join('')}</div>`;
}

function renderSegmentDrawer(segment) {
  if (!segment) {
    renderDrawerEmpty();
    return;
  }
  const segmentEvents = eventsForSegment(segment.id);
  const collapsed = state.drawerOpen ? '' : ' collapsed';
  els.drawer.innerHTML = `
    <div class="panel-head">
      <div>
        <p class="eyebrow">${selectionLabel()}</p>
        <h2>Selected pressure sequence</h2>
      </div>
      <div>
        <p>${esc(formatRange(segment))} elapsed</p>
        ${drawerToggle()}
      </div>
    </div>
    <div class="drawer-body${collapsed}">
      <div class="drawer-summary">
        <span class="section-label">Safe summary</span>
        <strong class="moment-title">${esc(stateLabel(segment))}</strong>
        <p>${esc(formatSafeSummary(segment.summary, 'Fixture segment summary unavailable.'))}</p>
      </div>
      <div class="drawer-metrics">
        <div><span>Blue pressure share</span><strong>${esc(formatPct(segment.pressureShareBlue))}</strong></div>
        <div><span>Orange pressure share</span><strong>${esc(formatPct(segment.pressureShareOrange))}</strong></div>
        <div><span>Confidence</span><strong>${esc(formatPct(segment.confidence))}</strong></div>
      </div>
      <div class="drawer-section">
        <span class="section-label">Notable events</span>
        ${renderEventList(segmentEvents)}
      </div>
      <div class="drawer-section">
        <span class="section-label">Pressure contribution highlights</span>
        ${renderContributionHighlights(segment)}
      </div>
      <button class="replay-placeholder" type="button" disabled>Replay jump planned</button>
    </div>
  `;
  bindDrawerToggle();
}

function renderEventDrawer(event) {
  if (!event) {
    renderDrawerEmpty();
    return;
  }
  const segment = segmentForEvent(event);
  const collapsed = state.drawerOpen ? '' : ' collapsed';
  els.drawer.innerHTML = `
    <div class="panel-head">
      <div>
        <p class="eyebrow">${selectionLabel()}</p>
        <h2>Selected event marker</h2>
      </div>
      <div>
        <p>${esc(formatTime(event.second))} elapsed</p>
        ${drawerToggle()}
      </div>
    </div>
    <div class="drawer-body${collapsed}">
      <div class="drawer-summary">
        <span class="section-label">${esc(eventLabel(event))}</span>
        <strong class="moment-title">${esc(event.playerName || event.team || 'Fixture marker')}</strong>
        <p>${esc(formatSafeSummary(event.summary, 'Fixture event summary unavailable.'))}</p>
      </div>
      <div class="drawer-section">
        <span class="section-label">Nearby pressure/control context</span>
        <p>${esc(formatSafeSummary(segment?.summary, 'Containing fixture segment unavailable.'))}</p>
      </div>
      <div class="drawer-metrics">
        <div><span>Segment range</span><strong>${esc(formatRange(segment))}</strong></div>
        <div><span>Segment state</span><strong>${esc(stateLabel(segment))}</strong></div>
        <div><span>Confidence</span><strong>${esc(formatPct(segment?.confidence))}</strong></div>
      </div>
      <button class="replay-placeholder" type="button" disabled>Replay Studio integration later</button>
    </div>
  `;
  bindDrawerToggle();
}

function renderPlayers() {
  const players = state.data?.playerContributions || [];
  if (!players.length) {
    els.players.innerHTML = '<div class="empty">No fixture player contribution summaries available.</div>';
    return;
  }
  const html = ['blue', 'orange'].map(team => {
    const teamPlayers = players.filter(player => player.team === team);
    if (!teamPlayers.length) return '';
    return `<div class="team-summary ${team}">
      <h3>${team === 'blue' ? 'Blue' : 'Orange'}</h3>
      ${teamPlayers.map(player => {
        const events = player.events || {};
        return `<article class="player-card">
          <div>
            <strong>${esc(player.playerName || 'Fixture player')}</strong>
            <span>${esc(player.team || 'team unavailable')}</span>
          </div>
          <dl>
            <div><dt>Pressure contribution</dt><dd>${esc(formatPct(player.pressureContribution))}</dd></div>
            <div><dt>Momentum influence</dt><dd>${esc(formatPct(player.momentumInfluence))}</dd></div>
            <div><dt>Contest involvement</dt><dd>${esc(formatPct(player.contestInvolvement))}</dd></div>
          </dl>
          <p class="event-counts">Goals ${events.goals ?? 0} / Assists ${events.assists ?? 0} / Saves ${events.saves ?? 0} / Shots ${events.shots ?? 0} / Demos ${events.demos ?? 0}</p>
        </article>`;
      }).join('')}
    </div>`;
  }).join('');
  els.players.innerHTML = html || '<div class="empty">No fixture player contribution summaries available.</div>';
}

function rerenderInteractive() {
  renderTimeline();
  renderMarkers();
  if (state.selectedEventId) {
    renderEventDrawer((state.data?.events || []).find(event => event.id === state.selectedEventId));
  } else if (state.selectedSegmentId) {
    renderSegmentDrawer((state.data?.segments || []).find(segment => segment.id === state.selectedSegmentId));
  } else {
    renderDrawerEmpty();
  }
}

function selectEvent(eventId, focusAfterSelect) {
  const event = (state.data?.events || []).find(item => item.id === eventId);
  state.selectedEventId = eventId || '';
  state.selectedSegmentId = event ? (segmentForEvent(event)?.id || '') : '';
  state.activeMomentKey = eventId ? `event:${eventId}` : '';
  state.drawerOpen = true;
  rerenderInteractive();
  if (focusAfterSelect && event) focusMoment({ type: 'event', id: event.id });
}

function selectSegment(segmentId, focusAfterSelect) {
  state.selectedSegmentId = segmentId || '';
  state.selectedEventId = '';
  state.activeMomentKey = segmentId ? `segment:${segmentId}` : '';
  state.drawerOpen = true;
  rerenderInteractive();
  if (focusAfterSelect && segmentId) focusMoment({ type: 'segment', id: segmentId });
}

function clearSelection() {
  state.selectedSegmentId = '';
  state.selectedEventId = '';
  state.activeMomentKey = '';
  rerenderInteractive();
}

function renderAll() {
  renderSummary();
  renderScale();
  renderPlayers();
  rerenderInteractive();
}

async function loadFixture() {
  try {
    const response = await fetch('/internal/momentum-timeline-preview/fixture.json', { cache: 'no-store' });
    if (!response.ok) throw new Error(`Fixture request failed: ${response.status}`);
    state.data = await response.json();
  } catch (err) {
    console.error(err);
    state.data = {
      match: { id: 'fixture-error', source: 'fixture', durationSeconds: 300, blueName: 'Blue', orangeName: 'Orange' },
      totals: {},
      segments: [],
      events: [],
      playerContributions: [],
    };
  }
  renderAll();
}

els.clear.addEventListener('click', () => {
  clearSelection();
});

document.addEventListener('keydown', event => {
  if (event.target?.matches?.('input, textarea, select')) return;
  if (event.key === 'ArrowRight') {
    event.preventDefault();
    moveSelection(1);
  } else if (event.key === 'ArrowLeft') {
    event.preventDefault();
    moveSelection(-1);
  } else if (event.key === 'Escape') {
    event.preventDefault();
    if ((state.selectedSegmentId || state.selectedEventId) && state.drawerOpen) {
      state.drawerOpen = false;
      rerenderInteractive();
    } else {
      clearSelection();
    }
  }
});

loadFixture();
