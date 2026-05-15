'use strict';

const state = {
  data: null,
  selectedSegmentId: '',
  selectedEventId: '',
  drawerOpen: true,
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

function stateLabel(segment) {
  if (!segment) return 'Unknown';
  const stateName = segment.state ? segment.state.replace('_', ' ') : 'unknown';
  const teamName = segment.team && segment.team !== 'none' ? `${segment.team} ` : '';
  return `${teamName}${stateName}`;
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
    [`${data.match.blueName || 'Blue'} pressure/control`, `${formatTime(totals.bluePressureSeconds)} / ${formatTime(totals.blueControlSeconds)}`],
    [`${data.match.orangeName || 'Orange'} pressure/control`, `${formatTime(totals.orangePressureSeconds)} / ${formatTime(totals.orangeControlSeconds)}`],
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
    return `<button
      class="${segmentClass(segment)}${selected}"
      type="button"
      data-segment-id="${esc(segment.id)}"
      style="left:${start}%;width:${width}%;opacity:${intensity.toFixed(2)}"
      aria-label="${esc(formatRange(segment))} ${esc(stateLabel(segment))}">
      <span>${esc(stateLabel(segment))}</span>
    </button>`;
  }).join('');

  els.timeline.querySelectorAll('[data-segment-id]').forEach(button => {
    button.addEventListener('click', () => selectSegment(button.dataset.segmentId));
  });
}

function renderMarkers() {
  const duration = state.data?.match?.durationSeconds || 1;
  const events = state.data?.events || [];
  if (!events.length) {
    els.markers.innerHTML = '<div class="empty">No fixture event markers available.</div>';
    return;
  }
  els.markers.innerHTML = events.map((event, index) => {
    const left = clampPercent((event.second / duration) * 100);
    const lane = index % 2;
    const selected = event.id === state.selectedEventId ? ' selected' : '';
    const label = EVENT_LABELS[event.type] || event.label || 'Event';
    return `<button
      class="event-marker ${esc(event.team || 'none')}${selected}"
      type="button"
      data-event-id="${esc(event.id)}"
      style="left:${left}%;--lane:${lane}"
      title="${esc(formatTime(event.second))} ${esc(label)}">
      <span>${esc(EVENT_SIGNS[event.type] || label.slice(0, 2))}</span>
    </button>`;
  }).join('');

  els.markers.querySelectorAll('[data-event-id]').forEach(button => {
    button.addEventListener('click', () => selectEvent(button.dataset.eventId));
  });
}

function renderDrawerEmpty() {
  els.drawer.innerHTML = `
    <div class="panel-head">
      <h2>Selected moment</h2>
      <p>Choose a segment or event marker to inspect the fixture narrative.</p>
    </div>
    <div class="empty drawer-empty">
      No timeline moment selected.
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
  return `<ul class="event-list">${events.map(event => `
    <li>
      <span class="event-time">${formatTime(event.second)}</span>
      <strong>${esc(EVENT_LABELS[event.type] || event.label || 'Event')}</strong>
      <span>${esc(event.playerName || event.team || 'Fixture marker')}</span>
      <p>${esc(event.summary || 'Fixture event summary unavailable.')}</p>
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
      <h2>Selected pressure sequence</h2>
      <div>
        <p>${esc(formatRange(segment))} elapsed</p>
        ${drawerToggle()}
      </div>
    </div>
    <div class="drawer-body${collapsed}">
      <div class="drawer-section">
        <span class="section-label">Event-derived pressure/control</span>
        <strong class="moment-title">${esc(stateLabel(segment))}</strong>
        <p>${esc(segment.summary || 'Fixture segment summary unavailable.')}</p>
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
      <h2>Selected event marker</h2>
      <div>
        <p>${esc(formatTime(event.second))} elapsed</p>
        ${drawerToggle()}
      </div>
    </div>
    <div class="drawer-body${collapsed}">
      <div class="drawer-section">
        <span class="section-label">${esc(EVENT_LABELS[event.type] || event.label || 'Event')}</span>
        <strong class="moment-title">${esc(event.playerName || event.team || 'Fixture marker')}</strong>
        <p>${esc(event.summary || 'Fixture event summary unavailable.')}</p>
      </div>
      <div class="drawer-section">
        <span class="section-label">Nearby pressure/control context</span>
        <p>${esc(segment?.summary || 'Containing fixture segment unavailable.')}</p>
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
    els.players.innerHTML = '<div class="empty">No fixture player contribution values available.</div>';
    return;
  }
  els.players.innerHTML = ['blue', 'orange'].map(team => {
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

function selectSegment(segmentId) {
  state.selectedSegmentId = segmentId || '';
  state.selectedEventId = '';
  state.drawerOpen = true;
  rerenderInteractive();
}

function selectEvent(eventId) {
  const event = (state.data?.events || []).find(item => item.id === eventId);
  state.selectedEventId = eventId || '';
  state.selectedSegmentId = event ? (segmentForEvent(event)?.id || '') : '';
  state.drawerOpen = true;
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
  state.selectedSegmentId = '';
  state.selectedEventId = '';
  rerenderInteractive();
});

loadFixture();
