'use strict';

const DBG_STORAGE_KEY = 'oof-rl-debug-assistant-regression-v1';
const DBG_SESSION_KEY = 'oof-rl-debug-assistant-session-active-v1';

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
  { id: 'debug-assistant-track-b', title: 'Track B: Debug Assistant verification', shots: ['Debug page startup', 'Report export folder', 'Generated report panels'] },
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

const DBG_TRACK_B_CHECKS = [
  {
    id: 'debug-clean-startup',
    title: 'Debug page starts clean on fresh app startup.',
    help: 'Close all OOF RL instances, open the test EXE, go to Debug, and confirm no old checklist data appears unless JSON was manually imported.',
    screenshot: true,
  },
  {
    id: 'debug-check-buttons',
    title: 'Pass / Fail / N/A buttons select and unselect correctly.',
    help: 'Click each status, then click the active status again to confirm it clears and updates scenario/overall stats.',
    screenshot: true,
  },
  {
    id: 'debug-note-formatting',
    title: 'Checklist notes wrap and remain readable.',
    help: 'Add a long multi-line note and confirm it behaves like a body text field and remains readable in generated reports.',
    screenshot: true,
  },
  {
    id: 'debug-custom-conditions',
    title: 'Custom conditions can be added, tracked, reported, and removed.',
    help: 'Add a custom condition, mark it, add notes, confirm it affects stats/reports, then remove it and confirm saved check data is removed.',
    screenshot: true,
  },
  {
    id: 'debug-json-import',
    title: 'JSON state import is manual-only.',
    help: 'Export a JSON state, restart the app, confirm Debug starts clean, then manually import the JSON and confirm state appears only after selection.',
    screenshot: true,
  },
  {
    id: 'debug-report-generation',
    title: 'Plain and doc reports generate correctly.',
    help: 'Generate both report views and verify metadata, summaries, scenario details, failure groups, notes, and screenshot filenames are included.',
    screenshot: true,
  },
  {
    id: 'debug-export-files',
    title: 'Report export creates .md, .html, and .json files.',
    help: 'Export reports and verify matching files are created in AppData Local OOF_RL debug_reports.',
    screenshot: true,
  },
  {
    id: 'debug-duplicate-export',
    title: 'Duplicate report exports are skipped.',
    help: 'Export twice without changing Debug state and confirm no duplicate files are created and duplicate export notice appears.',
    screenshot: true,
  },
  {
    id: 'debug-scroll-state',
    title: 'Debug and other pages retain separate scroll positions.',
    help: 'Scroll Debug, switch to History, scroll there, and confirm each page restores its own scroll position when revisited.',
    screenshot: true,
  },
  {
    id: 'debug-read-only-safety',
    title: 'Debug Assistant does not mutate core OOF RL data.',
    help: 'During Track A playtesting, confirm Debug use does not change Live, Session, History, saved matches, event handling, or app config except debug-local state/export files.',
    screenshot: true,
  },
];

window.pluginInit_debug = function() {
  dbgInitializeSessionState();
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

function dbgInitializeSessionState() {
  if (sessionStorage.getItem(DBG_SESSION_KEY)) return;
  localStorage.removeItem(DBG_STORAGE_KEY);
  sessionStorage.setItem(DBG_SESSION_KEY, '1');
}

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
  dbgRenderOverallSummary(state);
  root.innerHTML = DBG_SCENARIOS.map(s => {
    const stats = dbgScenarioStats(state.scenarios?.[s.id]);
    const touched = stats.touched;
    const queued = !touched && !!state.scenarios?.[s.id]?.startedAt;
    const failed = stats.fail > 0;
    return `
    <button class="dbg-scenario${state.activeScenario === s.id ? ' active' : ''}${queued ? ' queued' : ''}${touched ? ' touched' : ''}${failed ? ' failed' : ''}" data-dbg-scenario="${esc(s.id)}">
      ${esc(s.title)}
      <small>Suggested evidence: ${esc(s.shots.join(', '))}</small>
      <span class="dbg-scenario-meta">
        <span class="dbg-pill pass">${stats.pass} pass</span>
        <span class="dbg-pill fail">${stats.fail} fail</span>
        <span class="dbg-pill skip">${stats.skip} N/A</span>
        <span class="dbg-pill">${stats.percent}%</span>
      </span>
    </button>
  `}).join('');
  root.querySelectorAll('[data-dbg-scenario]').forEach(btn => {
    btn.addEventListener('click', () => {
      const next = btn.dataset.dbgScenario;
      const current = dbgState();
      current.activeScenario = next;
      current.scenarios = current.scenarios || {};
      current.scenarios[next] = current.scenarios[next] || { checks: {}, startedAt: new Date().toISOString(), notes: '' };
      current.scenarios[next].checklistType = next === 'debug-assistant-track-b' ? 'debug-assistant' : 'match';
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
  const customForm = document.getElementById('dbg-custom-condition');
  if (!root) return;

  const state = dbgState();
  const scenario = dbgActiveScenario();
  if (!scenario) {
    root.innerHTML = '<div class="dbg-sub">Pick the test scenario you are about to run. The checklist will stay tied to that scenario locally.</div>';
    if (label) label.textContent = 'Select a test scenario before queueing.';
    if (customForm) customForm.style.display = 'none';
    return;
  }

  const scenarioState = state.scenarios?.[scenario.id] || { checks: {} };
  if (label) label.textContent = scenario.title;
  if (customForm) customForm.style.display = 'grid';
  dbgRenderScenarioSummary(scenarioState);

  root.innerHTML = dbgChecksForScenario(scenarioState).map(check => {
    const value = scenarioState.checks?.[check.id] || {};
    return `
      <div class="dbg-check" data-dbg-check="${esc(check.id)}">
        <div class="dbg-check-status">
          <button class="pass${value.status === 'pass' ? ' active' : ''}" data-status="pass">Pass</button>
          <button class="fail${value.status === 'fail' ? ' active' : ''}" data-status="fail">Fail</button>
          <button class="skip${value.status === 'skip' ? ' active' : ''}" data-status="skip">N/A</button>
        </div>
        <div>
          <div class="dbg-check-title">${esc(check.title)}${check.custom ? '<span class="dbg-custom-tag">custom</span>' : ''}</div>
          <div class="dbg-check-help">${esc(check.help)}</div>
          ${check.screenshot ? '<span class="dbg-shot">Screenshot/data recommended if failed or uncertain</span>' : ''}
          <textarea class="dbg-note" placeholder="Evidence note, screenshot filename, log timestamp, or reproduction detail">${esc(value.note || '')}</textarea>
          <input class="dbg-images" value="${esc(value.images || '')}" placeholder="Optional screenshot filenames, comma-separated">
          ${check.custom ? '<button class="dbg-check-remove" data-remove-custom="1">Remove custom condition</button>' : ''}
        </div>
      </div>`;
  }).join('');

  root.querySelectorAll('.dbg-check').forEach(row => {
    row.querySelectorAll('button[data-status]').forEach(btn => {
      btn.addEventListener('click', () => dbgSetCheck(row.dataset.dbgCheck, btn.dataset.status));
    });
    row.querySelector('.dbg-note')?.addEventListener('input', e => dbgSetCheckNote(row.dataset.dbgCheck, e.target.value));
    row.querySelector('.dbg-images')?.addEventListener('input', e => dbgSetCheckImages(row.dataset.dbgCheck, e.target.value));
    row.querySelector('[data-remove-custom]')?.addEventListener('click', () => dbgRemoveCustomCheck(row.dataset.dbgCheck));
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
  scenarioState.checks[checkID].status = scenarioState.checks[checkID].status === status ? '' : status;
  scenarioState.checks[checkID].updatedAt = new Date().toISOString();
  dbgSaveState(state);
  dbgRenderScenarios();
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
  dbgRenderScenarios();
  dbgRenderScenarioSummary(scenarioState);
  dbgRefreshWidgetInstances();
}

function dbgSetCheckImages(checkID, images) {
  const state = dbgState();
  const scenarioState = dbgScenarioState(state);
  if (!scenarioState) return;
  scenarioState.checks[checkID] = scenarioState.checks[checkID] || {};
  scenarioState.checks[checkID].images = images;
  scenarioState.checks[checkID].updatedAt = new Date().toISOString();
  dbgSaveState(state);
  dbgRenderScenarios();
  dbgRenderScenarioSummary(scenarioState);
  dbgRefreshWidgetInstances();
}

function dbgChecksForScenario(scenarioState) {
  const baseChecks = scenarioState?.checklistType === 'debug-assistant' ? DBG_TRACK_B_CHECKS : DBG_CHECKS;
  return baseChecks.concat((scenarioState?.customChecks || []).map(check => ({
    id: check.id,
    title: check.title || 'Untitled custom condition',
    help: check.help || 'Custom verification condition added during playtest.',
    screenshot: true,
    custom: true,
  })));
}

function dbgAddCustomCheck() {
  const titleEl = document.getElementById('dbg-custom-title');
  const helpEl = document.getElementById('dbg-custom-help');
  const title = (titleEl?.value || '').trim();
  const help = (helpEl?.value || '').trim();
  if (!title) {
    dbgMessage('Add a condition title first');
    return;
  }

  const state = dbgState();
  const scenarioState = dbgScenarioState(state);
  if (!scenarioState) return;
  scenarioState.customChecks = scenarioState.customChecks || [];
  scenarioState.customChecks.push({
    id: `custom-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    title,
    help,
    createdAt: new Date().toISOString(),
  });
  dbgSaveState(state);
  if (titleEl) titleEl.value = '';
  if (helpEl) helpEl.value = '';
  dbgRenderScenarios();
  dbgRenderChecks();
  dbgRefreshWidgetInstances();
}

function dbgRemoveCustomCheck(checkID) {
  const state = dbgState();
  const scenarioState = dbgScenarioState(state);
  if (!scenarioState) return;
  scenarioState.customChecks = (scenarioState.customChecks || []).filter(check => check.id !== checkID);
  if (scenarioState.checks) delete scenarioState.checks[checkID];
  dbgSaveState(state);
  dbgRenderScenarios();
  dbgRenderChecks();
  dbgRefreshWidgetInstances();
}

function dbgScenarioStats(scenarioState) {
  const checks = scenarioState?.checks || {};
  const scenarioChecks = dbgChecksForScenario(scenarioState);
  const values = scenarioChecks.map(check => checks[check.id] || {});
  const pass = values.filter(v => v.status === 'pass').length;
  const fail = values.filter(v => v.status === 'fail').length;
  const skip = values.filter(v => v.status === 'skip').length;
  const marked = pass + fail + skip;
  const scored = pass + fail;
  return {
    pass,
    fail,
    skip,
    marked,
    untouched: Math.max(0, scenarioChecks.length - marked),
    percent: scored ? Math.round(pass / scored * 100) : 0,
    touched: marked > 0 || values.some(v => (v.note || '').trim()),
  };
}

function dbgOverallStats(state) {
  const scenarioStates = Object.values(state.scenarios || {});
  return scenarioStates.reduce((acc, scenarioState) => {
    const stats = dbgScenarioStats(scenarioState);
    acc.pass += stats.pass;
    acc.fail += stats.fail;
    acc.skip += stats.skip;
    acc.marked += stats.marked;
    acc.touched += stats.touched ? 1 : 0;
    return acc;
  }, { pass: 0, fail: 0, skip: 0, marked: 0, touched: 0 });
}

function dbgSummaryHTML(stats) {
  const scored = stats.pass + stats.fail;
  const rate = scored ? Math.round(stats.pass / scored * 100) : 0;
  return [
    ['Pass', stats.pass, 'pass'],
    ['Fail', stats.fail, 'fail'],
    ['N/A', stats.skip, 'skip'],
    ['Pass rate', `${rate}%`, ''],
  ].map(([label, value, cls]) => `
    <div class="dbg-summary-item">
      <div class="dbg-summary-value ${cls ? `dbg-pill ${cls}` : ''}">${esc(value)}</div>
      <div class="dbg-summary-label">${esc(label)}</div>
    </div>`).join('');
}

function dbgRenderOverallSummary(state = dbgState()) {
  const root = document.getElementById('dbg-overall-summary');
  if (!root) return;
  const stats = dbgOverallStats(state);
  root.innerHTML = dbgSummaryHTML(stats);
}

function dbgRenderScenarioSummary(scenarioState) {
  const root = document.getElementById('dbg-scenario-summary');
  if (!root) return;
  root.innerHTML = dbgSummaryHTML(dbgScenarioStats(scenarioState));
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
  const report = dbgBuildPlainReport();
  const root = document.getElementById('dbg-report');
  if (root) root.textContent = report;
  return report;
}

function dbgBuildPlainReport() {
  const state = dbgState();
  const meta = state.metadata || {};
  const scenario = dbgActiveScenario();
  const scenarioState = scenario ? state.scenarios?.[scenario.id] : null;
  const overall = dbgOverallStats(state);
  const failures = dbgFailureGroups(state);
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
  lines.push('Overall summary:');
  lines.push(`- Scenarios touched: ${overall.touched} / ${DBG_SCENARIOS.length}`);
  lines.push(`- Pass: ${overall.pass}`);
  lines.push(`- Fail: ${overall.fail}`);
  lines.push(`- N/A: ${overall.skip}`);
  lines.push(`- Pass rate: ${overall.pass + overall.fail ? Math.round(overall.pass / (overall.pass + overall.fail) * 100) : 0}%`);
  lines.push('');
  lines.push(`Active scenario: ${scenario ? scenario.title : 'none'}`);
  lines.push(`Suggested evidence: ${scenario ? scenario.shots.join(', ') : 'none'}`);
  lines.push('');
  lines.push('Checklist:');
  for (const check of dbgChecksForScenario(scenarioState)) {
    const item = scenarioState?.checks?.[check.id] || {};
    lines.push(`- [${item.status || 'unset'}] ${check.title}`);
    if (item.note) lines.push(`  Note: ${item.note}`);
    for (const image of dbgImageNames(item)) lines.push(`  Screenshot: ${image}`);
  }
  lines.push('');
  lines.push('Scenario details:');
  for (const s of DBG_SCENARIOS) {
    const scopedState = state.scenarios?.[s.id];
    const stats = dbgScenarioStats(scopedState);
    if (!stats.touched) continue;
    lines.push(`- ${s.title}`);
    for (const check of dbgChecksForScenario(scopedState)) {
      const item = scopedState?.checks?.[check.id] || {};
      if (!item.status && !item.note && !item.images) continue;
      lines.push(`  - [${item.status || 'unset'}] ${check.title}`);
      if (item.note) lines.push(`    Note: ${item.note}`);
      for (const image of dbgImageNames(item)) lines.push(`    Screenshot: ${image}`);
    }
  }
  lines.push('');
  lines.push('Failure groups:');
  if (!failures.length) {
    lines.push('- No failed checks recorded.');
  } else {
    for (const group of failures) {
      lines.push(`- ${group.title}`);
      for (const item of group.items) {
        lines.push(`  - ${item.scenario}${item.note ? `: ${item.note}` : ''}`);
      }
    }
  }
  lines.push('');
  lines.push('Touched scenario summary:');
  for (const s of DBG_SCENARIOS) {
    const stats = dbgScenarioStats(state.scenarios?.[s.id]);
    if (!stats.touched) {
      lines.push(`- [untested] ${s.title}`);
      continue;
    }
    lines.push(`- ${s.title}: ${stats.pass} pass, ${stats.fail} fail, ${stats.skip} N/A, ${stats.percent}% pass rate`);
  }
  lines.push('');
  lines.push(`Snapshots saved locally: ${(state.snapshots || []).length}`);
  lines.push('');
  lines.push('Session notes:');
  lines.push(meta.notes || '');

  return lines.join('\n');
}

function dbgGenerateDocReport() {
  const html = dbgBuildDocReportHTML(false);
  const root = document.getElementById('dbg-report-doc');
  if (root) root.innerHTML = html;
  return html;
}

function dbgBuildDocReportHTML(exportMode) {
  const state = dbgState();
  const meta = state.metadata || {};
  const overall = dbgOverallStats(state);
  const failures = dbgFailureGroups(state);
  const parts = [];
  parts.push('<h2>OOF RL Debug Assistant Report</h2>');
  parts.push('<h3>Build</h3>');
  parts.push('<ul>');
  parts.push(`<li><strong>Branch:</strong> ${esc(meta.branch || '')}</li>`);
  parts.push(`<li><strong>Commit SHA:</strong> ${esc(meta.sha || '')}</li>`);
  parts.push(`<li><strong>EXE:</strong> ${esc(meta.exe || '')}</li>`);
  parts.push(`<li><strong>Tester:</strong> ${esc(meta.tester || '')}</li>`);
  parts.push(`<li><strong>Intent:</strong> ${esc(meta.intent || '')}</li>`);
  parts.push(`<li><strong>Rocket League mode/version:</strong> ${esc(meta['rl-mode'] || '')}</li>`);
  parts.push(`<li><strong>Generated:</strong> ${esc(new Date().toLocaleString())}</li>`);
  parts.push('</ul>');

  parts.push('<h3>Summary</h3>');
  parts.push('<ul>');
  parts.push(`<li>Scenarios touched: ${overall.touched} / ${DBG_SCENARIOS.length}</li>`);
  parts.push(`<li><span class="pass">Pass:</span> ${overall.pass}</li>`);
  parts.push(`<li><span class="fail">Fail:</span> ${overall.fail}</li>`);
  parts.push(`<li><span class="skip">N/A:</span> ${overall.skip}</li>`);
  parts.push(`<li>Pass rate: ${overall.pass + overall.fail ? Math.round(overall.pass / (overall.pass + overall.fail) * 100) : 0}%</li>`);
  parts.push('</ul>');

  parts.push('<h3>Failures Grouped By Check</h3>');
  if (!failures.length) {
    parts.push('<p>No failed checks recorded.</p>');
  } else {
    for (const group of failures) {
      parts.push(`<h4>${esc(group.title)}</h4><ul>`);
      for (const item of group.items) {
        parts.push(`<li><strong>${esc(item.scenario)}</strong>${item.note ? `: ${esc(item.note)}` : ''}</li>`);
      }
      parts.push('</ul>');
    }
  }

  parts.push('<h3>Scenario Details</h3>');
  for (const scenario of DBG_SCENARIOS) {
    const scenarioState = state.scenarios?.[scenario.id];
    const stats = dbgScenarioStats(scenarioState);
    if (!stats.touched) continue;
    parts.push(`<h4>${esc(scenario.title)}</h4>`);
    parts.push(`<p>${stats.pass} pass, ${stats.fail} fail, ${stats.skip} N/A, ${stats.percent}% pass rate</p>`);
    parts.push('<ul>');
    for (const check of dbgChecksForScenario(scenarioState)) {
      const item = scenarioState?.checks?.[check.id] || {};
      if (!item.status && !item.note && !item.images) continue;
      const statusClass = item.status === 'pass' ? 'pass' : item.status === 'fail' ? 'fail' : item.status === 'skip' ? 'skip' : '';
      parts.push(`<li><span class="${statusClass}">[${esc(item.status || 'unset')}]</span> ${esc(check.title)}${item.note ? `<br><em>${esc(item.note)}</em>` : ''}</li>`);
      for (const image of dbgImageNames(item)) {
        const href = exportMode ? dbgExportScreenshotPath(image) : dbgScreenshotURL(image);
        parts.push(`<div class="shot-link"><a href="${href}" target="_blank" rel="noopener">Open screenshot: ${esc(image)}</a></div>`);
        if (!exportMode) parts.push(`<img alt="${esc(image)}" src="${href}" onerror="this.style.display='none'">`);
      }
    }
    parts.push('</ul>');
  }

  if (meta.notes) {
    parts.push('<h3>Session Notes</h3>');
    parts.push(`<p>${esc(meta.notes)}</p>`);
  }
  return parts.join('');
}

function dbgBuildStandaloneDocHTML() {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>OOF RL Debug Assistant Report</title>
<style>
body { background:#0a0b0e; color:#e5e7eb; font-family:Inter,Segoe UI,Arial,sans-serif; line-height:1.5; padding:28px; }
main { max-width:980px; margin:0 auto; }
h2 { margin-top:0; }
h3 { color:#4a9eff; margin-top:24px; }
h4 { margin-bottom:6px; }
a { color:#4a9eff; font-weight:700; }
ul { margin-top:6px; }
li { margin-bottom:6px; }
.pass { color:#3ecf72; font-weight:800; }
.fail { color:#e05252; font-weight:800; }
.skip { color:#f59e0b; font-weight:800; }
.shot-link { margin:8px 0; }
</style>
</head>
<body><main>${dbgBuildDocReportHTML(true)}</main></body>
</html>`;
}

function dbgImageNames(item) {
  return String(item?.images || '')
    .split(/[\n,]+/)
    .map(s => dbgCleanImageName(s.trim()))
    .filter(Boolean);
}

function dbgCleanImageName(name) {
  if (!name) return '';
  return name.split(/[\\/]/).filter(Boolean).pop() || '';
}

function dbgScreenshotURL(name) {
  return `/api/debug-assistant/screenshot/${encodeURIComponent(dbgCleanImageName(name))}`;
}

function dbgExportScreenshotPath(name) {
  return `../debug_screenshots/${encodeURIComponent(dbgCleanImageName(name))}`;
}

function dbgFailureGroups(state) {
  const out = [];
  const checksByID = new Map(DBG_CHECKS.map(check => [check.id, check]));
  for (const scenario of DBG_SCENARIOS) {
    for (const check of dbgChecksForScenario(state.scenarios?.[scenario.id])) {
      if (!checksByID.has(check.id)) checksByID.set(check.id, check);
    }
  }
  for (const check of checksByID.values()) {
    const items = [];
    for (const scenario of DBG_SCENARIOS) {
      const item = state.scenarios?.[scenario.id]?.checks?.[check.id];
      if (item?.status === 'fail') {
        items.push({ scenario: scenario.title, note: item.note || '' });
      }
    }
    if (items.length) out.push({ id: check.id, title: check.title, items });
  }
  return out;
}

function dbgWireControls() {
  document.getElementById('dbg-save-meta')?.addEventListener('click', dbgSaveMeta);
  document.getElementById('dbg-snapshot')?.addEventListener('click', dbgSnapshot);
  document.getElementById('dbg-export')?.addEventListener('click', dbgGenerateReport);
  document.getElementById('dbg-export-doc')?.addEventListener('click', dbgGenerateDocReport);
  document.getElementById('dbg-save-reports')?.addEventListener('click', dbgExportReportFiles);
  document.getElementById('dbg-import-json')?.addEventListener('click', () => {
    if (!confirm('Import a Debug Assistant JSON state file? This replaces the current local Debug checklist state.')) return;
    document.getElementById('dbg-import-file')?.click();
  });
  document.getElementById('dbg-import-file')?.addEventListener('change', dbgImportJSONState);
  document.getElementById('dbg-add-condition')?.addEventListener('click', dbgAddCustomCheck);
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

async function dbgExportReportFiles() {
  try {
    if (!confirm('Export the current Debug Assistant report files to the OOF RL debug_reports folder? Duplicate exports of the same state will be skipped.')) return;
    const plain = dbgGenerateReport();
    const html = dbgBuildStandaloneDocHTML();
    const state = JSON.stringify(dbgState(), null, 2);
    const exportID = dbgExportID(state);
    const response = await fetch('/api/debug-assistant/export-report', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ plain, html, state: JSON.parse(state), export_id: exportID }),
    });
    if (!response.ok) throw new Error(await response.text());
    const result = await response.json();
    dbgGenerateDocReport();
    if (result.duplicate) {
      dbgMessage(result.message || 'Report already exported. Duplicate export skipped.');
      return;
    }
    dbgMessage(`Exported reports to ${result.dir}`);
  } catch (e) {
    dbgMessage(`Export failed: ${e.message || e}`);
  }
}

async function dbgImportJSONState(event) {
  const file = event.target.files?.[0];
  event.target.value = '';
  if (!file) return;
  try {
    const imported = JSON.parse(await file.text());
    if (!imported || typeof imported !== 'object' || Array.isArray(imported)) {
      throw new Error('Selected file is not a valid Debug Assistant state object');
    }
    dbgSaveState(imported);
    dbgLoadMeta();
    dbgRenderScenarios();
    dbgRenderChecks();
    dbgGenerateReport();
    dbgGenerateDocReport();
    dbgRefreshWidgetInstances();
    dbgMessage(`Imported ${file.name}`);
  } catch (e) {
    dbgMessage(`Import failed: ${e.message || e}`);
  }
}

function dbgExportID(text) {
  let hash = 2166136261;
  for (let i = 0; i < text.length; i++) {
    hash ^= text.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return `state-${(hash >>> 0).toString(16)}`;
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
  const statuses = dbgChecksForScenario(scenarioState).map(check => scenarioState?.checks?.[check.id] || {}).filter(v => v.status && v.status !== 'skip');
  const passed = statuses.filter(v => v.status === 'pass').length;
  const stats = dbgScenarioStats(scenarioState);
  const progressEl = container.querySelector('[data-dbgw-progress]');
  const scenarioEl = container.querySelector('[data-dbgw-scenario]');
  const rateEl = container.querySelector('[data-dbgw-rate]');
  const eventEl = container.querySelector('[data-dbgw-event]');
  if (scenarioEl) scenarioEl.textContent = scenario ? scenario.title.replace(/^Normal /, '') : 'none';
  if (progressEl) progressEl.textContent = `${passed} pass, ${stats.fail} fail, ${stats.skip} N/A`;
  if (rateEl) rateEl.textContent = `${stats.percent}%`;
  try {
    const data = await fetch('/api/debug-assistant/events').then(r => r.json());
    const last = (data.events || []).slice(-1)[0];
    if (eventEl) eventEl.textContent = last ? `${last.event}: ${last.summary}` : 'none';
  } catch (_) {
    if (eventEl) eventEl.textContent = 'unavailable';
  }
}
