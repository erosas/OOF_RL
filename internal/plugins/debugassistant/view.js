'use strict';

const DBG_STORAGE_KEY = 'oof-rl-debug-assistant-regression-v1';

const DBG_SCENARIOS = [
  { id: 'online-1v1-pvp', title: 'Normal online 1v1 PvP match', shots: ['History collapsed row', 'Expanded match details', 'Session overview'] },
  { id: 'online-team-pvp', title: 'Normal online 2v2 or 3v3 PvP match', shots: ['History collapsed row', 'Expanded match details', 'Session overview'] },
  { id: 'overtime', title: 'Match that reaches overtime', shots: ['History row with OT badge', 'Expanded match details'] },
  { id: 'forfeit', title: 'Match decided by forfeit with meaningful clock time remaining', shots: ['History row with FF badge', 'Session match row'] },
  { id: 'full-time-no-ff', title: 'Full-time match with no forfeit', shots: ['History row without FF badge', 'Final scoreboard if available'] },
  { id: 'private-bot-1v1', title: 'Private 1v1 against a bot', shots: ['History row with PvE badge', 'Expanded player stats showing bot'] },
  { id: 'private-mixed-bots', title: 'Private mixed human/bot match', shots: ['History row with PvE badge', 'Expanded teams'] },
  { id: 'all-bot-private', title: 'All-bot private match', shots: ['History row', 'Expanded match details'] },
  { id: 'abandoned-destroyed', title: 'Match abandoned or destroyed without normal MatchEnded', shots: ['History row with Incomplete badge', 'Log snippet around MatchDestroyed'] },
];

const DBG_CHECKS = [
  {
    id: 'single-history-row',
    title: 'Exactly one History row is created for the played match.',
    help: 'Use the collapsed History list. A duplicate row is a fail, especially if one row is incomplete.',
    screenshot: true,
  },
  {
    id: 'no-duplicate-incomplete',
    title: 'No duplicate incomplete row appears after match end.',
    help: 'If an incomplete row appears alongside a completed row for the same match, collect log and screenshot evidence.',
    screenshot: true,
  },
  {
    id: 'arena-score',
    title: 'Arena and score are correct.',
    help: 'Compare the History row score/arena against what happened in game.',
    screenshot: true,
  },
  {
    id: 'player-count',
    title: 'Player count badge matches stored players.',
    help: 'Expand the match and compare the visible stored roster to the collapsed count badge.',
    screenshot: true,
  },
  {
    id: 'pvp-badge',
    title: 'PvP badge appears when no bots are stored.',
    help: 'Only pass this for real-player matches with no stored bot identities.',
    screenshot: true,
  },
  {
    id: 'pve-badge',
    title: 'PvE badge appears when one or more bots are stored.',
    help: 'Bot/private scenarios should show PvE and should not pollute Session by default.',
    screenshot: true,
  },
  {
    id: 'ot-badge',
    title: 'OT appears only for overtime matches.',
    help: 'Pass if OT appears for overtime and stays absent for non-OT scenarios.',
    screenshot: true,
  },
  {
    id: 'ff-badge',
    title: 'FF appears only for likely forfeits.',
    help: 'This is heuristic behavior. If false positive/negative appears, collect final clock context and logs.',
    screenshot: true,
  },
  {
    id: 'incomplete-badge',
    title: 'Incomplete appears only for abnormal completion paths.',
    help: 'Expected for abandoned/destroyed paths. Unexpected for normal MatchEnded paths.',
    screenshot: true,
  },
  {
    id: 'expanded-details',
    title: 'Expanded match details show expected players and teams.',
    help: 'Compare the stored roster to the match you played as closely as the API provides.',
    screenshot: true,
  },
  {
    id: 'goal-events',
    title: 'Goal/event details appear when the API emitted them.',
    help: 'If goals happened but events are missing, collect screenshot plus log/capture evidence.',
    screenshot: true,
  },
];

window.pluginInit_debug = function() {
  dbgRenderScenarios();
  dbgLoadMeta();
  dbgRenderChecks();
  dbgWireControls();
  dbgRefreshAll();
  setInterval(dbgRefreshAll, 3000);

  window.registerWidget?.({
    id: 'debug-regression-status',
    pluginId: 'debug',
    title: 'Debug Assistant',
    defaultW: 4,
    defaultH: 4,
    minW: 3,
    minH: 3,
    factory: debugRegressionWidget,
  });
};

function dbgState() {
  try { return JSON.parse(localStorage.getItem(DBG_STORAGE_KEY) || '{}'); }
  catch (_) { return {}; }
}

function dbgSaveState(state) {
  localStorage.setItem(DBG_STORAGE_KEY, JSON.stringify(state));
}

function dbgMetaFields() {
  return ['branch', 'sha', 'exe', 'tester', 'intent', 'rl-mode', 'notes'];
}

function dbgLoadMeta() {
  const meta = dbgState().metadata || {};
  for (const key of dbgMetaFields()) {
    const el = document.getElementById(`dbg-${key}`);
    if (el) el.value = meta[key] || '';
  }
}

function dbgSaveMeta() {
  const state = dbgState();
  state.metadata = {};
  for (const key of dbgMetaFields()) {
    const el = document.getElementById(`dbg-${key}`);
    if (el) state.metadata[key] = el.value.trim();
  }
  dbgSaveState(state);
  dbgMessage('Saved locally');
}

function dbgMessage(text) {
  const el = document.getElementById('dbg-msg');
  if (!el) return;
  el.textContent = text;
  setTimeout(() => { el.textContent = ''; }, 2600);
}

function dbgRenderScenarios() {
  const root = document.getElementById('dbg-scenarios');
  if (!root) return;
  const state = dbgState();
  root.innerHTML = DBG_SCENARIOS.map(s => `
    <button class="dbg-scenario${state.activeScenario === s.id ? ' active' : ''}" data-dbg-scenario="${esc(s.id)}">
      ${esc(s.title)}
      <small>Suggested evidence: ${esc(s.shots.join(', '))}</small>
    </button>
  `).join('');
  root.querySelectorAll('[data-dbg-scenario]').forEach(btn => {
    btn.addEventListener('click', () => {
      const next = btn.dataset.dbgScenario;
      const current = dbgState();
      current.activeScenario = next;
      current.scenarios = current.scenarios || {};
      current.scenarios[next] = current.scenarios[next] || { checks: {}, startedAt: new Date().toISOString(), notes: '' };
      dbgSaveState(current);
      dbgRenderScenarios();
      dbgRenderChecks();
      dbgRefreshWidgetInstances();
    });
  });
}

function dbgActiveScenario() {
  const id = dbgState().activeScenario;
  return DBG_SCENARIOS.find(s => s.id === id) || null;
}

function dbgRenderChecks() {
  const root = document.getElementById('dbg-checks');
  const label = document.getElementById('dbg-active-scenario');
  if (!root) return;

  const state = dbgState();
  const scenario = dbgActiveScenario();
  if (!scenario) {
    root.innerHTML = '<div class="dbg-sub">Pick the match type you are about to test. The checklist will stay tied to that scenario locally.</div>';
    if (label) label.textContent = 'Select a match scenario before queueing.';
    return;
  }

  const scenarioState = state.scenarios?.[scenario.id] || { checks: {} };
  if (label) label.textContent = scenario.title;

  root.innerHTML = DBG_CHECKS.map(check => {
    const value = scenarioState.checks?.[check.id] || {};
    return `
      <div class="dbg-check" data-dbg-check="${esc(check.id)}">
        <div class="dbg-check-status">
          <button class="pass${value.status === 'pass' ? ' active' : ''}" data-status="pass">Pass</button>
          <button class="fail${value.status === 'fail' ? ' active' : ''}" data-status="fail">Fail</button>
          <button class="${value.status === 'skip' ? ' active' : ''}" data-status="skip">N/A</button>
        </div>
        <div>
          <div class="dbg-check-title">${esc(check.title)}</div>
          <div class="dbg-check-help">${esc(check.help)}</div>
          ${check.screenshot ? '<span class="dbg-shot">Screenshot/data recommended if failed or uncertain</span>' : ''}
          <input class="dbg-note" value="${esc(value.note || '')}" placeholder="Evidence note, screenshot filename, log timestamp, or reproduction detail">
        </div>
      </div>`;
  }).join('');

  root.querySelectorAll('.dbg-check').forEach(row => {
    row.querySelectorAll('button[data-status]').forEach(btn => {
      btn.addEventListener('click', () => dbgSetCheck(row.dataset.dbgCheck, btn.dataset.status));
    });
    row.querySelector('.dbg-note')?.addEventListener('input', e => dbgSetCheckNote(row.dataset.dbgCheck, e.target.value));
  });
}

function dbgScenarioState(state) {
  const scenario = dbgActiveScenario();
  if (!scenario) return null;
  state.scenarios = state.scenarios || {};
  state.scenarios[scenario.id] = state.scenarios[scenario.id] || { checks: {}, startedAt: new Date().toISOString() };
  state.scenarios[scenario.id].checks = state.scenarios[scenario.id].checks || {};
  return state.scenarios[scenario.id];
}

function dbgSetCheck(checkID, status) {
  const state = dbgState();
  const scenarioState = dbgScenarioState(state);
  if (!scenarioState) return;
  scenarioState.checks[checkID] = scenarioState.checks[checkID] || {};
  scenarioState.checks[checkID].status = status;
  scenarioState.checks[checkID].updatedAt = new Date().toISOString();
  dbgSaveState(state);
  dbgRenderChecks();
  dbgRefreshWidgetInstances();
}

function dbgSetCheckNote(checkID, note) {
  const state = dbgState();
  const scenarioState = dbgScenarioState(state);
  if (!scenarioState) return;
  scenarioState.checks[checkID] = scenarioState.checks[checkID] || {};
  scenarioState.checks[checkID].note = note;
  scenarioState.checks[checkID].updatedAt = new Date().toISOString();
  dbgSaveState(state);
}

async function dbgRefreshAll() {
  await Promise.all([dbgLoadContext(), dbgLoadEvents()]);
  dbgRefreshWidgetInstances();
}

async function dbgLoadContext() {
  const root = document.getElementById('dbg-context');
  if (!root) return;
  try {
    const [ctx, live, matches] = await Promise.all([
      fetch('/api/debug-assistant/context').then(r => r.json()),
      fetch('/api/live/state').then(r => r.json()).catch(() => ({})),
      fetch('/api/matches').then(r => r.json()).catch(() => []),
    ]);
    const validMatches = Array.isArray(matches) ? matches.filter(m => m.Arena && m.Arena !== '-') : [];
    const latest = validMatches[0] || {};
    root.innerHTML = [
      ['Data dir', ctx.data_dir || ''],
      ['Live active', live.active ? 'yes' : 'no'],
      ['History rows', String(validMatches.length)],
      ['Latest arena', friendlyArena(latest.Arena) || 'none'],
      ['Latest score', latest.ID ? `${latest.team0_goals ?? 0} - ${latest.team1_goals ?? 0}` : 'none'],
      ['Observed events', String(ctx.observed_events ?? 0)],
    ].map(([k, v]) => `<div class="dbg-context-item"><div class="dbg-context-label">${esc(k)}</div><div class="dbg-context-value" title="${esc(v)}">${esc(v)}</div></div>`).join('');
  } catch (e) {
    root.innerHTML = `<div class="dbg-sub">Context load failed: ${esc(e.message || e)}</div>`;
  }
}

async function dbgLoadEvents() {
  const root = document.getElementById('dbg-events');
  if (!root) return;
  try {
    const data = await fetch('/api/debug-assistant/events').then(r => r.json());
    const events = data.events || [];
    if (!events.length) {
      root.innerHTML = '<div class="dbg-sub">No observed events yet.</div>';
      return;
    }
    root.innerHTML = events.slice().reverse().slice(0, 40).map(e => `
      <div class="dbg-event">
        <div class="dbg-event-time">${esc(new Date(e.at).toLocaleTimeString())}</div>
        <div class="dbg-event-name" title="${esc(e.match_guid || '')}">${esc(e.event)}</div>
        <div class="dbg-event-summary">${esc(e.summary || '')}</div>
      </div>
    `).join('');
  } catch (e) {
    root.innerHTML = `<div class="dbg-sub">Event load failed: ${esc(e.message || e)}</div>`;
  }
}

async function dbgSnapshot() {
  try {
    const state = dbgState();
    state.snapshots = state.snapshots || [];
    const [ctx, live, matches] = await Promise.all([
      fetch('/api/debug-assistant/context').then(r => r.json()),
      fetch('/api/live/state').then(r => r.json()).catch(() => ({})),
      fetch('/api/matches').then(r => r.json()).catch(() => []),
    ]);
    state.snapshots.push({
      at: new Date().toISOString(),
      activeScenario: state.activeScenario || '',
      context: ctx,
      liveActive: !!live.active,
      latestMatch: Array.isArray(matches) ? matches[0] : null,
    });
    state.snapshots = state.snapshots.slice(-25);
    dbgSaveState(state);
    dbgMessage('Snapshot saved locally');
  } catch (e) {
    dbgMessage('Snapshot failed');
  }
}

function dbgGenerateReport() {
  const state = dbgState();
  const meta = state.metadata || {};
  const scenario = dbgActiveScenario();
  const scenarioState = scenario ? state.scenarios?.[scenario.id] : null;
  const lines = [];
  lines.push('OOF RL Debug Assistant Report');
  lines.push('');
  lines.push(`Branch: ${meta.branch || ''}`);
  lines.push(`Commit SHA: ${meta.sha || ''}`);
  lines.push(`EXE: ${meta.exe || ''}`);
  lines.push(`Tester: ${meta.tester || ''}`);
  lines.push(`Intent: ${meta.intent || ''}`);
  lines.push(`RL mode/version: ${meta['rl-mode'] || ''}`);
  lines.push(`Generated: ${new Date().toLocaleString()}`);
  lines.push('');
  lines.push(`Active scenario: ${scenario ? scenario.title : 'none'}`);
  lines.push(`Suggested evidence: ${scenario ? scenario.shots.join(', ') : 'none'}`);
  lines.push('');
  lines.push('Checklist:');
  for (const check of DBG_CHECKS) {
    const item = scenarioState?.checks?.[check.id] || {};
    lines.push(`- [${item.status || 'unset'}] ${check.title}`);
    if (item.note) lines.push(`  Note: ${item.note}`);
  }
  lines.push('');
  lines.push(`Snapshots saved locally: ${(state.snapshots || []).length}`);
  lines.push('');
  lines.push('Session notes:');
  lines.push(meta.notes || '');

  const report = lines.join('\n');
  const root = document.getElementById('dbg-report');
  if (root) root.textContent = report;
}

function dbgWireControls() {
  document.getElementById('dbg-save-meta')?.addEventListener('click', dbgSaveMeta);
  document.getElementById('dbg-snapshot')?.addEventListener('click', dbgSnapshot);
  document.getElementById('dbg-export')?.addEventListener('click', dbgGenerateReport);
  document.getElementById('dbg-reset')?.addEventListener('click', () => {
    if (!confirm('Reset local Debug Assistant metadata, checklist, snapshots, and notes?')) return;
    localStorage.removeItem(DBG_STORAGE_KEY);
    dbgLoadMeta();
    dbgRenderScenarios();
    dbgRenderChecks();
    dbgLoadContext();
    dbgGenerateReport();
  });
}

const dbgWidgetInstances = new Set();

function debugRegressionWidget(container) {
  const tpl = document.getElementById('dbg-widget-template');
  container.innerHTML = tpl ? tpl.innerHTML : '<div class="dbg-widget"></div>';
  const instance = { container, refresh: () => dbgRenderWidget(container) };
  dbgWidgetInstances.add(instance);
  container.querySelector('[data-dbgw-refresh]')?.addEventListener('click', instance.refresh);
  instance.refresh();
  return {
    refresh: instance.refresh,
    destroy: () => dbgWidgetInstances.delete(instance),
  };
}

function dbgRefreshWidgetInstances() {
  for (const instance of dbgWidgetInstances) instance.refresh();
}

async function dbgRenderWidget(container) {
  const scenario = dbgActiveScenario();
  const state = dbgState();
  const scenarioState = scenario ? state.scenarios?.[scenario.id] : null;
  const statuses = Object.values(scenarioState?.checks || {}).filter(v => v.status && v.status !== 'skip');
  const passed = statuses.filter(v => v.status === 'pass').length;
  const progressEl = container.querySelector('[data-dbgw-progress]');
  const scenarioEl = container.querySelector('[data-dbgw-scenario]');
  const eventEl = container.querySelector('[data-dbgw-event]');
  if (scenarioEl) scenarioEl.textContent = scenario ? scenario.title.replace(/^Normal /, '') : 'none';
  if (progressEl) progressEl.textContent = `${passed} / ${DBG_CHECKS.length}`;
  try {
    const data = await fetch('/api/debug-assistant/events').then(r => r.json());
    const last = (data.events || []).slice(-1)[0];
    if (eventEl) eventEl.textContent = last ? `${last.event}: ${last.summary}` : 'none';
  } catch (_) {
    if (eventEl) eventEl.textContent = 'unavailable';
  }
}
