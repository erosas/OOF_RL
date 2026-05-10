'use strict';

const DBG_STORAGE_KEY = 'oof-rl-debug-assistant-regression-v1';
const DBG_SESSION_KEY = 'oof-rl-debug-assistant-session-active-v1';
const DBG_SCROLL_POSITIONS = {};
let DBG_LAST_LIVE_STATE = null;

const DBG_SCENARIOS = [
  { id: 'track-a-app-regression', title: 'Track A: App Match Regression', shots: ['History rows', 'Expanded match details', 'Session overview', 'Debug report'] },
  { id: 'debug-assistant-track-b', title: 'Track B: Debug Assistant verification', shots: ['Debug page startup', 'Report export folder', 'Generated report panels'] },
  { id: 'track-c-commit-verification', title: 'Track C: Commit fix verification', shots: ['Track C grouped checks', 'Generated report with Track C sections'] },
  { id: 'track-d-debug-link-watch', title: 'Track D: Debug Match Linking Bug Watch', shots: ['Track D grouped checks', 'Linked match cards', 'Generated report with Track D sections'] },
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

const DBG_TRACK_A_CHECKS = {
  'track-a-online-1v1': [
    DBG_CHECKS.find(check => check.id === 'single-history-row'),
    DBG_CHECKS.find(check => check.id === 'arena-score'),
    DBG_CHECKS.find(check => check.id === 'expanded-details'),
  ],
  'track-a-online-team': [
    DBG_CHECKS.find(check => check.id === 'player-count'),
    DBG_CHECKS.find(check => check.id === 'pvp-badge'),
  ],
  'track-a-outcomes': [
    DBG_CHECKS.find(check => check.id === 'ot-badge'),
    DBG_CHECKS.find(check => check.id === 'ff-badge'),
    {
      id: 'full-time-clean-end',
      title: 'Full-time matches do not receive false FF or Incomplete tags.',
      help: 'Validate full regulation endings against History badges and Session rows.',
      screenshot: true,
    },
  ],
  'track-a-private-bot-abnormal': [
    DBG_CHECKS.find(check => check.id === 'pve-badge'),
    DBG_CHECKS.find(check => check.id === 'incomplete-badge'),
    DBG_CHECKS.find(check => check.id === 'goal-events'),
  ],
};

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

const DBG_TRACK_C_CHECKS = {
  'track-c-note-layout': [
    {
      id: 'note-wraps',
      title: 'Long checklist notes wrap instead of scrolling sideways.',
      help: 'Add a long note with several sentences. Pass if it wraps cleanly inside the checklist card.',
      screenshot: true,
    },
    {
      id: 'note-autosizes',
      title: 'Note field expands vertically as text is added.',
      help: 'Type multiple lines. Pass if the note body grows and remains readable without awkward horizontal overflow.',
      screenshot: true,
    },
    {
      id: 'note-report-readable',
      title: 'Saved notes remain readable in generated reports.',
      help: 'Generate plain and doc reports. Pass if the long note is readable and not clipped or hard to scan.',
      screenshot: true,
    },
  ],
  'track-c-scroll-state': [
    {
      id: 'debug-scroll-restores',
      title: 'Debug page restores its own scroll position.',
      help: 'Scroll down Debug, switch away, return to Debug. Pass if the Debug position is retained.',
      screenshot: true,
    },
    {
      id: 'history-scroll-restores',
      title: 'History page restores its own scroll position separately.',
      help: 'Scroll History to a different position, switch away, return. Pass if History restores independently from Debug.',
      screenshot: true,
    },
    {
      id: 'async-load-does-not-reset',
      title: 'Async page refresh does not force the view back to top.',
      help: 'Wait for History/Debug content refresh after returning. Pass if delayed loaders do not erase the restored scroll position.',
      screenshot: true,
    },
  ],
  'track-c-json-import': [
    {
      id: 'import-requires-picker',
      title: 'JSON import requires manual file selection.',
      help: 'Click Import JSON state. Pass if the app opens a file picker instead of silently loading an old backup.',
      screenshot: true,
    },
    {
      id: 'no-auto-restore',
      title: 'Old backup state does not auto-load on fresh startup.',
      help: 'Restart the app and open Debug. Pass if checklist state starts clean unless a JSON file is manually imported.',
      screenshot: true,
    },
    {
      id: 'imported-state-applies',
      title: 'Selected JSON state imports and replaces local Debug state.',
      help: 'Import a known report JSON. Pass if metadata, scenarios, notes, and checks appear only after confirmation.',
      screenshot: true,
    },
  ],
  'track-c-report-readability': [
    {
      id: 'doc-report-structured',
      title: 'Doc report has readable developer-facing sections.',
      help: 'Generate the doc report. Pass if metadata, summary, scenario details, failures, and evidence are clearly separated.',
      screenshot: true,
    },
    {
      id: 'failure-groups-readable',
      title: 'Failure groups are easy to scan.',
      help: 'Mark several failures with notes. Pass if the report groups failures clearly without burying the important context.',
      screenshot: true,
    },
    {
      id: 'notes-and-images-listed',
      title: 'Notes and screenshot filenames are preserved in the report.',
      help: 'Attach note text and image filenames. Pass if both appear in the report in a readable format.',
      screenshot: true,
    },
  ],
  'track-c-export-result': [
    {
      id: 'export-creates-files',
      title: 'Export creates Markdown, HTML, and JSON files.',
      help: 'Export report files. Pass if matching .md, .html, and .json files are created in the debug_reports folder.',
      screenshot: true,
    },
    {
      id: 'export-result-paths',
      title: 'Export result panel shows exact output paths.',
      help: 'After export, pass if the UI lists folder, Markdown, HTML, and JSON paths with readable wrapping.',
      screenshot: true,
    },
    {
      id: 'duplicate-export-skipped',
      title: 'Duplicate export is skipped without creating extra copies.',
      help: 'Click export twice without changing state. Pass if duplicate export is skipped and the user-facing notice appears.',
      screenshot: true,
    },
  ],
  'track-c-commit-scenarios': [
    {
      id: 'track-c-scenarios-visible',
      title: 'Track C appears as one grouped verification scenario.',
      help: 'Open Debug and inspect the scenario list. Pass if Track C appears as one top-level scenario with grouped commit-fix checks inside it.',
      screenshot: true,
    },
    {
      id: 'track-c-independent-status',
      title: 'Each Track C grouped check tracks pass/fail/N/A independently.',
      help: 'Mark checks in two different Track C sections. Pass if their check states and report entries remain separate inside the grouped Track C scenario.',
      screenshot: true,
    },
    {
      id: 'track-c-report-output',
      title: 'Generated reports include Track C grouped results.',
      help: 'Generate reports after marking Track C checks. Pass if the report clearly includes the Track C scenario and its commit-specific sections.',
      screenshot: true,
    },
  ],
};

const DBG_TRACK_D_CHECKS = {
  'track-d-stale-guid': [
    {
      id: 'second-link-unique-guid',
      title: 'Second issue-linked match does not inherit the previous match GUID.',
      help: 'Pass: each issue-linked match has its own correct GUID/History ID. Fail: a new linked match shows a previous match GUID or mismatched History row. Notes: record both check names, both GUIDs, both History IDs, and match start/end timestamps.',
      screenshot: true,
    },
    {
      id: 'history-row-matches-link',
      title: 'Linked History ID points to the correct match row.',
      help: 'Pass: History ID, arena, score, and timestamp match the played scenario. Fail: link points to a prior or unrelated History row. Notes: include the History row screenshot and linked card screenshot.',
      screenshot: true,
    },
  ],
  'track-d-duplicate-links': [
    {
      id: 'single-confirmed-batch',
      title: 'One confirmed issue batch creates one match evidence record.',
      help: 'Pass: MatchCreated, MatchInitialized, and UpdateState do not create duplicate evidence for the same confirmed batch. Fail: duplicate linked evidence appears for the same check/match. Notes: include the event timeline around match start if duplicates appear.',
      screenshot: true,
    },
    {
      id: 'batch-consumed-after-link',
      title: 'Confirmed issue batch is consumed after the next match links.',
      help: 'Pass: the selected checks no longer remain armed after one match is linked. Fail: the next unrelated match also attaches to the previous confirmed issue batch. Notes: record the armed count before and after the first linked match.',
      screenshot: true,
    },
  ],
  'track-d-multi-issue-linking': [
    {
      id: 'multiple-checks-same-match',
      title: 'Multiple issue checks can link to the same match.',
      help: 'Pass: selecting multiple checks, confirming links, and playing one match shows compact evidence inside every selected check. Fail: only one check receives evidence or the wrong checks receive evidence. Notes: list selected checks and compare them to the inline evidence sections.',
      screenshot: true,
    },
    {
      id: 'confirm-links-required',
      title: 'Issue selection requires Confirm Links before match tagging.',
      help: 'Pass: checked issue boxes alone do not tag a match until Confirm Links is used. Fail: a match links from an unconfirmed draft selection. Notes: record whether the pending/armed message was shown before match start.',
      screenshot: true,
    },
  ],
  'track-d-deselect-unarmed': [
    {
      id: 'deselect-clears-active-scenario',
      title: 'Deselect track returns Debug Assistant to a clean unarmed state.',
      help: 'Pass: the Track Verification panel returns to the no-selection state and clears draft/confirmed issue links. Fail: the old track remains armed or visually active. Notes: screenshot the panel after deselect.',
      screenshot: true,
    },
    {
      id: 'next-match-not-tagged',
      title: 'Next match is not tagged after deselect.',
      help: 'Pass: after deselecting, the next played match does not create debug evidence. Fail: a match links to a deselected track/check. Notes: include the match timestamp and verification panel after match end.',
      screenshot: true,
    },
  ],
  'track-d-inline-evidence': [
    {
      id: 'evidence-expands-inside-check',
      title: 'Linked Match Evidence expands inside the issue check.',
      help: 'Pass: the compact badge stays inside the linked check and expands there without jumping to a top-level match panel. Fail: evidence appears only in a separate standalone panel or forces the user away from the check. Notes: include collapsed and expanded screenshots.',
      screenshot: true,
    },
    {
      id: 'expanded-evidence-fields',
      title: 'Expanded evidence includes all required match fields.',
      help: 'Pass: GUID, History ID, score, player stats snapshot, arena display name, raw API arena value, timestamp, linked check name, and notes/evidence status are visible. Fail: any required field is missing or misleading. Notes: compare against History expanded details when available.',
      screenshot: true,
    },
  ],
  'track-d-link-actions': [
    {
      id: 'expand-collapse-works',
      title: 'Inline evidence expand/collapse controls execute correctly.',
      help: 'Pass: Expand opens the evidence body and Collapse returns it to the compact badge. Fail: the button does nothing, opens an alert, renders blank, or expands the wrong check. Notes: record the linked check name and GUID.',
      screenshot: true,
    },
    {
      id: 'remove-check-evidence-works',
      title: 'Remove evidence affects only the selected issue check.',
      help: 'Pass: removing evidence from one check does not remove the same match evidence from other linked checks unless it was the last check. Fail: wrong check evidence is removed or core match data changes. Notes: capture before/after Debug and History state.',
      screenshot: true,
    },
    {
      id: 'no-standalone-panels',
      title: 'No duplicate large standalone match panels appear.',
      help: 'Pass: linked evidence is rendered compactly under each check, with no duplicate large top-level match panel for the same match. Fail: the same match appears as a large standalone panel and inline evidence. Notes: screenshot the top of Track D and the linked check.',
      screenshot: true,
    },
  ],
  'track-d-stale-storage': [
    {
      id: 'stale-link-removable',
      title: 'Old or bad debug links can be removed from Debug metadata.',
      help: 'Pass: stale links can be unlinked or cleared from Debug Assistant metadata only. Fail: stale links persist after unlink/clear. Notes: record whether the stale link came from an older build or current test.',
      screenshot: true,
    },
    {
      id: 'core-data-unchanged',
      title: 'Removing stale debug metadata does not alter core match data.',
      help: 'Pass: History and Session rows remain unchanged after stale debug metadata removal. Fail: core History/Session data changes. Notes: include before/after History or Session screenshots.',
      screenshot: true,
    },
  ],
  'track-d-arena-comparison': [
    {
      id: 'arena-values-visible',
      title: 'Inline evidence shows arena display name and raw API arena value.',
      help: 'Pass: both values are visible in expanded evidence and useful for mismatch detection. Fail: one or both values are missing, swapped, or misleading. Notes: record the display name and raw API value exactly.',
      screenshot: true,
    },
    {
      id: 'arena-compares-history',
      title: 'Arena values can be compared against the History row.',
      help: 'Pass: expanded inline evidence makes it easy to compare arena display/API values with History. Fail: comparison requires digging elsewhere or values conflict without clarity. Notes: include linked evidence and History row screenshots.',
      screenshot: true,
    },
  ],
  'track-d-report-placement': [
    {
      id: 'report-evidence-under-check',
      title: 'Reports render linked match evidence under the correct issue check.',
      help: 'Pass: generated plain/doc reports list GUID, score, arena, and stats under the check the match proves. Fail: evidence is only grouped globally or appears under the wrong check. Notes: include report preview/export screenshots.',
      screenshot: true,
    },
  ],
};

const DBG_TRACK_C_SECTIONS = {
  'track-c-note-layout': 'd0384a9 note layout fix',
  'track-c-scroll-state': '6e9a144 scroll-state fix',
  'track-c-json-import': '39426e2 JSON import fix',
  'track-c-report-readability': '4834f91 report readability fix',
  'track-c-export-result': '405617b export result panel fix',
  'track-c-commit-scenarios': 'dde4ccf commit scenario fix',
};

const DBG_TRACK_A_SECTIONS = {
  'track-a-online-1v1': 'Normal online 1v1 PvP match',
  'track-a-online-team': 'Normal online 2v2 or 3v3 PvP match',
  'track-a-outcomes': 'Overtime / forfeit / full-time outcomes',
  'track-a-private-bot-abnormal': 'Private / bot / abnormal end paths',
};

const DBG_TRACK_D_SECTIONS = {
  'track-d-stale-guid': 'Stale GUID carryover',
  'track-d-duplicate-links': 'Duplicate link creation',
  'track-d-multi-issue-linking': 'Multi-issue match linking',
  'track-d-deselect-unarmed': 'Scenario deselect / unarmed state',
  'track-d-inline-evidence': 'Inline linked evidence',
  'track-d-link-actions': 'Linked action button wiring',
  'track-d-stale-storage': 'Stale localStorage metadata',
  'track-d-arena-comparison': 'Arena raw/display comparison',
  'track-d-report-placement': 'Report evidence placement',
};

window.pluginInit_debug = function() {
  dbgInitializeSessionState();
  dbgRenderScenarios();
  dbgLoadMeta();
  dbgRenderChecks();
  dbgWireControls();
  dbgWireInternalScrollMemory();
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

function dbgIsDebugViewActive() {
  return window.oofActiveViewName === 'debug' || document.getElementById('view-debug')?.classList.contains('active');
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

function dbgShowExportResult(result) {
  const el = document.getElementById('dbg-export-result');
  if (!el || !result) return;
  const status = result.duplicate
    ? '<span class="duplicate">Duplicate export skipped.</span>'
    : 'Report files exported.';
  el.classList.add('visible');
  el.innerHTML = `
    <h4>Export Result</h4>
    <div>${status}</div>
    <ul>
      <li><strong>Folder:</strong> ${esc(result.dir || '')}</li>
      <li><strong>Markdown:</strong> ${esc(result.markdown || '')}</li>
      <li><strong>HTML:</strong> ${esc(result.html || '')}</li>
      <li><strong>JSON:</strong> ${esc(result.json || '')}</li>
    </ul>`;
}

function dbgWireInternalScrollMemory() {
  ['dbg-events', 'dbg-report', 'dbg-report-doc'].forEach(id => {
    const el = document.getElementById(id);
    if (!el || el.dataset.dbgScrollMemory === '1') return;
    el.dataset.dbgScrollMemory = '1';
    el.addEventListener('scroll', () => {
      DBG_SCROLL_POSITIONS[id] = el.scrollTop;
    }, { passive: true });
  });
}

function dbgRestoreInternalScroll(id) {
  const el = document.getElementById(id);
  if (!el) return;
  const y = DBG_SCROLL_POSITIONS[id] || 0;
  requestAnimationFrame(() => { el.scrollTop = y; });
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
      if (!dbgConfirmScenarioSwitch(current, next)) return;
      current.activeScenario = next;
      current.linkDraft = null;
      current.confirmedLinkBatch = null;
      current.debug_match = false;
      current.scenarios = current.scenarios || {};
      current.scenarios[next] = current.scenarios[next] || { checks: {}, startedAt: new Date().toISOString(), notes: '' };
      current.scenarios[next].checklistType = dbgChecklistTypeForScenario(next);
      current.scenarios[next].scenarioID = next;
      dbgSaveState(current);
      dbgRenderScenarios();
      dbgRenderChecks();
      dbgScrollToVerificationTop();
      dbgRefreshWidgetInstances();
    });
  });
}

function dbgActiveScenario() {
  const id = dbgState().activeScenario;
  return DBG_SCENARIOS.find(s => s.id === id) || null;
}

function dbgConfirmScenarioSwitch(state, nextScenarioID) {
  const currentID = state.activeScenario;
  if (!currentID || currentID === nextScenarioID) return true;
  const currentScenario = DBG_SCENARIOS.find(s => s.id === currentID);
  const currentState = state.scenarios?.[currentID];
  const stats = dbgScenarioStats(currentState);
  const total = dbgChecksForScenario(currentState).length;
  if (!currentScenario || !stats.touched || total === 0 || stats.untouched === 0) return true;
  return confirm(`"${currentScenario.title}" is not complete yet (${stats.marked}/${total} checks marked). Switch anyway?`);
}

function dbgRenderChecks() {
  const root = document.getElementById('dbg-checks');
  const label = document.getElementById('dbg-active-scenario');
  const title = document.getElementById('dbg-verification-title');
  const progress = document.getElementById('dbg-scenario-progress');
  const links = document.getElementById('dbg-linked-matches');
  const toolbar = document.getElementById('dbg-scenario-toolbar');
  const customForm = document.getElementById('dbg-custom-condition');
  if (!root) return;

  const state = dbgState();
  const scenario = dbgActiveScenario();
  if (!scenario) {
    dbgPlaceReportCard(false);
    root.innerHTML = `
      <div class="dbg-sub">Pick Track A, B, C, or D before queueing. The checklist will stay tied to that track locally.</div>
      <div id="dbg-inline-report-slot" class="dbg-inline-report-slot"></div>`;
    dbgPlaceReportCard(true);
    if (title) title.textContent = 'Track Verification';
    if (label) label.textContent = 'Select a debug track before queueing.';
    if (progress) progress.textContent = '';
    if (links) links.innerHTML = '';
    if (toolbar) toolbar.style.display = 'none';
    if (customForm) customForm.style.display = 'none';
    return;
  }
  dbgPlaceReportCard(false);

  const scenarioState = state.scenarios?.[scenario.id] || { checks: {} };
  const stats = dbgScenarioStats(scenarioState);
  const total = dbgChecksForScenario(scenarioState).length;
  const completionPercent = total ? Math.round(stats.marked / total * 100) : 0;
  const draftIDs = dbgLinkDraftIDs(state, scenario.id);
  const confirmedIDs = dbgConfirmedLinkIDs(state, scenario.id);
  const linkedIDs = dbgLinkedIssueIDs(state, scenario.id);
  if (title) title.textContent = `Testing: ${scenario.title}`;
  if (label) label.textContent = `Active scenario: ${scenario.title}`;
  if (progress) progress.textContent = `Scenario Progress: ${stats.marked}/${total} checks complete - ${completionPercent}%${confirmedIDs.length ? ` | ${confirmedIDs.length} issue link(s) armed for next match` : ''}`;
  if (links) links.innerHTML = dbgPendingLinkBatchHTML(state, scenario.id);
  if (toolbar) toolbar.style.display = 'flex';
  if (customForm) customForm.style.display = 'grid';
  dbgRenderScenarioSummary(scenarioState);

  let lastSection = '';
  root.innerHTML = dbgChecksForScenario(scenarioState).map(check => {
    const value = scenarioState.checks?.[check.id] || {};
    const sectionHTML = check.section && check.section !== lastSection
      ? `<div class="dbg-check-section">${esc(check.section)}</div>`
      : '';
    if (check.section) lastSection = check.section;
    const linkSelected = draftIDs.includes(check.id);
    const linkConfirmed = confirmedIDs.includes(check.id);
    const linkSaved = linkedIDs.includes(check.id);
    const evidenceHTML = dbgCheckEvidenceHTML(state, scenario.id, scenario.title, check.id, check.title, value);
    return `${sectionHTML}
      <div class="dbg-check${linkSelected ? ' link-selected' : ''}${linkConfirmed || linkSaved ? ' link-confirmed' : ''}" data-dbg-check="${esc(check.id)}">
        <div class="dbg-check-status">
          <button class="pass${value.status === 'pass' ? ' active' : ''}" data-status="pass">Pass</button>
          <button class="fail${value.status === 'fail' ? ' active' : ''}" data-status="fail">Fail</button>
          <button class="skip${value.status === 'skip' ? ' active' : ''}" data-status="skip">N/A</button>
        </div>
        <div>
          <div class="dbg-check-title">${esc(check.title)}${check.custom ? '<span class="dbg-custom-tag">custom</span>' : ''}</div>
          <div class="dbg-check-help">${esc(check.help)}</div>
          ${check.screenshot ? '<span class="dbg-shot">Screenshot/data recommended if failed or uncertain</span>' : ''}
          <label class="dbg-link-toggle">
            <input type="checkbox" data-link-check="1"${linkSelected || linkConfirmed ? ' checked' : ''}>
            Link this issue to the next confirmed debug match
          </label>
          ${linkConfirmed ? '<div class="dbg-link-help">Confirmed for the next match. Uncheck and confirm again to change the armed set.</div>' : ''}
          ${linkSaved ? '<div class="dbg-link-help">This issue has linked match evidence. Use the linked match card above to inspect or unlink it.</div>' : ''}
          ${evidenceHTML}
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
    const note = row.querySelector('.dbg-note');
    if (note) {
      dbgAutosizeTextarea(note);
      note.addEventListener('input', e => {
        dbgAutosizeTextarea(e.target);
        dbgSetCheckNote(row.dataset.dbgCheck, e.target.value);
      });
    }
    row.querySelector('.dbg-images')?.addEventListener('input', e => dbgSetCheckImages(row.dataset.dbgCheck, e.target.value));
    row.querySelector('[data-link-check]')?.addEventListener('change', e => dbgToggleLinkDraft(row.dataset.dbgCheck, e.target.checked));
    row.querySelector('[data-remove-custom]')?.addEventListener('click', () => dbgRemoveCustomCheck(row.dataset.dbgCheck));
  });
  links?.querySelectorAll('[data-dbg-link-action]').forEach(btn => {
    btn.addEventListener('click', () => dbgHandleLinkedMatchAction(btn.dataset.dbgLinkAction, btn.dataset.dbgLinkId));
  });
  root.querySelectorAll('[data-dbg-evidence-action]').forEach(btn => {
    btn.addEventListener('click', () => dbgHandleEvidenceAction(btn.dataset.dbgEvidenceAction, btn.dataset.dbgLinkId, btn.dataset.dbgCheckId));
  });
  document.getElementById('dbg-deselect-scenario')?.addEventListener('click', dbgDeselectScenario);
  document.getElementById('dbg-confirm-links')?.addEventListener('click', dbgConfirmLinkDraft);
  document.getElementById('dbg-clear-link-selection')?.addEventListener('click', dbgClearLinkDraft);
}

function dbgDeselectScenario() {
  const state = dbgState();
  if (!dbgConfirmScenarioSwitch(state, '')) return;
  state.activeScenario = '';
  state.debug_match = false;
  state.linkDraft = null;
  state.confirmedLinkBatch = null;
  dbgSaveState(state);
  dbgRenderScenarios();
  dbgRenderChecks();
  dbgScrollToVerificationTop();
  dbgRefreshWidgetInstances();
}

function dbgLinkDraftIDs(state, scenarioID) {
  return state.linkDraft?.scenarioID === scenarioID && Array.isArray(state.linkDraft.checkIDs)
    ? state.linkDraft.checkIDs
    : [];
}

function dbgConfirmedLinkIDs(state, scenarioID) {
  return state.confirmedLinkBatch?.scenarioID === scenarioID && Array.isArray(state.confirmedLinkBatch.checkIDs)
    ? state.confirmedLinkBatch.checkIDs
    : [];
}

function dbgLinkedIssueIDs(state, scenarioID) {
  const ids = new Set();
  for (const link of (state.matchLinks || [])) {
    if (link.scenarioID !== scenarioID || !Array.isArray(link.checkIDs)) continue;
    for (const id of link.checkIDs) ids.add(id);
  }
  return [...ids];
}

function dbgLinksForCheck(state, scenarioID, checkID) {
  return (state.matchLinks || []).filter(link =>
    link.scenarioID === scenarioID &&
    Array.isArray(link.checkIDs) &&
    link.checkIDs.includes(checkID)
  );
}

function dbgToggleLinkDraft(checkID, selected) {
  const state = dbgState();
  const scenario = dbgActiveScenario();
  if (!scenario) return;
  state.linkDraft = state.linkDraft?.scenarioID === scenario.id
    ? state.linkDraft
    : { scenarioID: scenario.id, checkIDs: [] };
  const ids = new Set(state.linkDraft.checkIDs || []);
  if (selected) ids.add(checkID);
  else ids.delete(checkID);
  state.linkDraft.checkIDs = [...ids];
  if (!state.linkDraft.checkIDs.length) state.linkDraft = null;
  state.confirmedLinkBatch = null;
  dbgSaveState(state);
  dbgRenderChecks();
  dbgRefreshWidgetInstances();
}

function dbgConfirmLinkDraft() {
  const state = dbgState();
  const scenario = dbgActiveScenario();
  if (!scenario) return;
  const draftIDs = dbgLinkDraftIDs(state, scenario.id);
  if (!draftIDs.length) {
    dbgMessage('Select one or more issue checks before confirming links');
    return;
  }
  const checksByID = new Map(dbgChecksForScenario(state.scenarios?.[scenario.id]).map(check => [check.id, check]));
  const labels = draftIDs.map(id => checksByID.get(id)?.title || id);
  const ok = confirm(`Confirm the next played match should link to ${draftIDs.length} issue check(s)?\n\n- ${labels.join('\n- ')}`);
  if (!ok) return;
  state.confirmedLinkBatch = {
    scenarioID: scenario.id,
    scenarioName: scenario.title,
    trackName: dbgTrackName(scenario),
    checkIDs: draftIDs,
    checkTitles: labels,
    confirmedAt: new Date().toISOString(),
  };
  state.debug_match = true;
  dbgSaveState(state);
  dbgRenderChecks();
  dbgMessage(`Armed ${draftIDs.length} issue link(s) for the next match`);
}

function dbgClearLinkDraft() {
  const state = dbgState();
  state.linkDraft = null;
  state.confirmedLinkBatch = null;
  state.debug_match = false;
  dbgSaveState(state);
  dbgRenderChecks();
  dbgRefreshWidgetInstances();
  dbgMessage('Cleared pending debug match link selection');
}

function dbgPlaceReportCard(inline) {
  const card = document.getElementById('dbg-report-card');
  if (!card) return;
  const target = inline
    ? document.getElementById('dbg-inline-report-slot')
    : document.getElementById('dbg-report-home');
  if (!target || card.parentElement === target) return;
  target.appendChild(card);
}

function dbgAutosizeTextarea(el) {
  if (!el) return;
  el.style.height = 'auto';
  el.style.height = `${el.scrollHeight}px`;
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
  const activeScenarioID = state.activeScenario;
  const scenarioState = dbgScenarioState(state);
  if (!scenarioState) return;
  scenarioState.checks[checkID] = scenarioState.checks[checkID] || {};
  scenarioState.checks[checkID].status = scenarioState.checks[checkID].status === status ? '' : status;
  scenarioState.checks[checkID].updatedAt = new Date().toISOString();
  const postAction = dbgMaybePromptScenarioComplete(state, activeScenarioID);
  dbgSaveState(state);
  dbgRenderScenarios();
  dbgRenderChecks();
  if (postAction === 'next') dbgScrollToVerificationTop();
  if (postAction === 'complete') dbgScrollToExportTop();
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
  const checklistType = scenarioState?.checklistType || 'match';
  const baseChecks = checklistType === 'app-regression'
    ? dbgGroupedTrackChecks(DBG_TRACK_A_CHECKS, DBG_TRACK_A_SECTIONS)
    : checklistType === 'debug-assistant'
    ? DBG_TRACK_B_CHECKS
    : checklistType === 'bugfix'
      ? dbgGroupedTrackChecks(DBG_TRACK_C_CHECKS, DBG_TRACK_C_SECTIONS)
      : checklistType === 'bug-watch'
        ? dbgGroupedTrackChecks(DBG_TRACK_D_CHECKS, DBG_TRACK_D_SECTIONS)
        : DBG_CHECKS;
  return baseChecks.concat((scenarioState?.customChecks || []).map(check => ({
    id: check.id,
    title: check.title || 'Untitled custom condition',
    help: check.help || 'Custom verification condition added during playtest.',
    screenshot: true,
    custom: true,
  })));
}

function dbgChecklistTypeForScenario(scenarioID) {
  if (scenarioID === 'track-a-app-regression') return 'app-regression';
  if (scenarioID === 'debug-assistant-track-b') return 'debug-assistant';
  if (scenarioID === 'track-c-commit-verification') return 'bugfix';
  if (scenarioID === 'track-d-debug-link-watch') return 'bug-watch';
  return 'match';
}

function dbgGroupedTrackChecks(groups, labels) {
  return Object.entries(groups).flatMap(([sectionID, checks]) => {
    const section = labels[sectionID] || sectionID;
    return checks.filter(Boolean).map(check => ({ ...check, id: `${sectionID}-${check.id}`, section }));
  });
}

function dbgTrackName(scenario) {
  if (!scenario) return '';
  if (scenario.title.startsWith('Track A:')) return 'Track A';
  if (scenario.title.startsWith('Track B:')) return 'Track B';
  if (scenario.title.startsWith('Track C:')) return 'Track C';
  if (scenario.title.startsWith('Track D:')) return 'Track D';
  return 'Track A';
}

function dbgScrollToVerificationTop() {
  document.getElementById('dbg-scenarios')?.closest('.dbg-grid')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function dbgScrollToExportTop() {
  document.getElementById('dbg-save-reports')?.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function dbgScenarioProgressSnapshot(scenarioState) {
  const stats = dbgScenarioStats(scenarioState);
  const total = dbgChecksForScenario(scenarioState).length;
  return {
    pass: stats.pass,
    fail: stats.fail,
    skip: stats.skip,
    marked: stats.marked,
    total,
    percent: stats.percent,
  };
}

function dbgLinkedMatchesForScenario(state, scenarioID) {
  return (state.matchLinks || []).filter(link => link.scenarioID === scenarioID);
}

function dbgPendingLinkBatchHTML(state, scenarioID) {
  const confirmedIDs = dbgConfirmedLinkIDs(state, scenarioID);
  const draftIDs = dbgLinkDraftIDs(state, scenarioID);
  if (confirmedIDs.length) {
    return `<div class="dbg-sub">${confirmedIDs.length} issue check(s) confirmed for the next match. Linked evidence will appear inside each selected check after the match starts.</div>`;
  }
  if (draftIDs.length) {
    return `<div class="dbg-sub">${draftIDs.length} issue check(s) selected. Use Confirm Links before the next match starts.</div>`;
  }
  return '<div class="dbg-sub">Select issue checks below, then confirm links to attach the next match as inline evidence.</div>';
}

function dbgLinkedMatchesHTML(state, scenarioID) {
  const links = dbgLinkedMatchesForScenario(state, scenarioID);
  if (!links.length) {
    return '<div class="dbg-sub">No matches linked to this scenario yet. Keep Debug open on this scenario before the next match starts to auto-link it.</div>';
  }
  return links.slice().reverse().map(link => {
    const score = link.score ? `${link.score.blue ?? 0}-${link.score.orange ?? 0}` : 'score pending';
    const status = link.completed ? `completed via ${link.completionEvent || 'match end'}` : 'active/pending';
    const mode = link.playlistName || link.matchType || 'mode pending';
    const source = link.autoTagged ? 'auto-tagged' : 'manual';
    const arenaRaw = link.arenaRaw || link.arena || '';
    const arenaDisplay = link.arenaDisplay || dbgArenaDisplayName(arenaRaw);
    const linkedIssues = Array.isArray(link.checkTitles) && link.checkTitles.length
      ? link.checkTitles
      : (Array.isArray(link.checkIDs) ? link.checkIDs : []);
    return `
      <div class="dbg-linked-match">
        <strong>Debug Match: ${esc(link.scenarioName || 'scenario')}</strong>
        <small>${esc(status)} - ${esc(source)} - ${esc(mode)} - ${esc(score)}</small>
        ${linkedIssues.length ? `<small>Linked issues: ${esc(linkedIssues.join(' | '))}</small>` : ''}
        <small>Arena: ${esc(arenaDisplay || 'pending')} ${arenaRaw ? `(API: ${esc(arenaRaw)})` : ''}</small>
        <small>GUID: ${esc(link.matchGuid || 'pending')} ${link.matchID ? `- History ID: ${esc(link.matchID)}` : ''}</small>
        <small>Started: ${esc(link.startedAt || '')}</small>
        <div class="dbg-linked-actions">
          <button data-dbg-link-action="scenario" data-dbg-link-id="${esc(link.linkID)}">View linked debug scenario</button>
          <button data-dbg-link-action="match" data-dbg-link-id="${esc(link.linkID)}">View linked match stats</button>
          <button data-dbg-link-action="unlink" data-dbg-link-id="${esc(link.linkID)}">Unlink match</button>
        </div>
        ${state.expandedDebugMatchLink === link.linkID ? dbgLinkedMatchStatsHTML(link) : ''}
      </div>`;
  }).join('');
}

function dbgCheckEvidenceHTML(state, scenarioID, scenarioTitle, checkID, checkTitle, checkState) {
  const links = dbgLinksForCheck(state, scenarioID, checkID);
  if (!links.length) return '';
  return `<div class="dbg-evidence-list">
    ${links.slice().reverse().map(link => dbgEvidenceCardHTML(state, link, checkID, checkTitle, scenarioTitle, checkState)).join('')}
  </div>`;
}

function dbgEvidenceCardHTML(state, link, checkID, checkTitle, scenarioTitle, checkState) {
  const expandedKey = `${link.linkID}:${checkID}`;
  const expanded = state.expandedEvidenceLink === expandedKey;
  const score = link.score ? `${link.score.blue ?? 0}-${link.score.orange ?? 0}` : 'score pending';
  const arenaRaw = link.arenaRaw || link.arena || '';
  const arenaDisplay = link.arenaDisplay || dbgArenaDisplayName(arenaRaw);
  const source = link.autoTagged ? 'auto-tagged' : 'manual';
  const mode = link.playlistName || link.matchType || 'mode pending';
  const status = dbgStatusLabel(checkState?.status);
  const note = (checkState?.note || '').trim();
  const images = dbgImageNames(checkState || {});
  return `<div class="dbg-evidence">
    <div class="dbg-evidence-head">
      <span class="dbg-evidence-badge">Linked Match Evidence</span>
      <div class="dbg-evidence-actions">
        <button data-dbg-evidence-action="toggle" data-dbg-link-id="${esc(link.linkID)}" data-dbg-check-id="${esc(checkID)}">${expanded ? 'Collapse' : 'Expand'}</button>
        <button data-dbg-evidence-action="unlink-check" data-dbg-link-id="${esc(link.linkID)}" data-dbg-check-id="${esc(checkID)}">Remove evidence</button>
      </div>
    </div>
    ${expanded ? `<div class="dbg-evidence-body">
      <small>Linked scenario: ${esc(scenarioTitle || link.scenarioName || 'scenario')}</small>
      <small>Linked check: ${esc(checkTitle)}</small>
      <small>GUID: ${esc(link.matchGuid || 'pending')} ${link.matchID ? `- History ID: ${esc(link.matchID)}` : ''}</small>
      <small>Score: ${esc(score)}</small>
      <small>Arena: ${esc(arenaDisplay || 'pending')} ${arenaRaw ? `(API: ${esc(arenaRaw)})` : ''}</small>
      <small>Mode/source: ${esc(mode)} - ${esc(source)}</small>
      <small>Timestamp: ${esc(link.startedAt || '')}</small>
      <small>Confirmed at: ${esc(link.confirmedAt || '')}</small>
      <small>Status: ${esc(link.completed ? `completed via ${link.completionEvent || 'match end'}` : 'active/pending')}</small>
      <small>Notes/evidence status: ${esc(status)}${note ? ` - ${esc(note)}` : ''}${images.length ? ` - screenshots: ${esc(images.join(', '))}` : ''}</small>
      ${dbgLinkedMatchStatsHTML(link)}
    </div>` : ''}
  </div>`;
}

function dbgLinkedMatchStatsHTML(link) {
  const players = Array.isArray(link.players) ? link.players : [];
  const arenaRaw = link.arenaRaw || link.arena || '';
  const arenaDisplay = link.arenaDisplay || dbgArenaDisplayName(arenaRaw);
  if (!players.length) {
    return `<div class="dbg-link-stats">
      <small>Arena: ${esc(arenaDisplay || 'pending')} ${arenaRaw ? `(API: ${esc(arenaRaw)})` : ''}</small>
      <small>No live player snapshot was captured for this linked match yet.</small>
    </div>`;
  }
  return `
    <div class="dbg-link-stats">
      <small>Arena: ${esc(arenaDisplay || 'pending')} ${arenaRaw ? `(API: ${esc(arenaRaw)})` : ''}</small>
      <small>Captured live player snapshot</small>
      <table>
        <thead><tr><th>Player</th><th>G</th><th>A</th><th>Sv</th><th>Sh</th><th>Dm</th><th>Tch</th><th>Score</th></tr></thead>
        <tbody>
          ${players.map(p => `
            <tr>
              <td>${esc(p.name || 'Unknown')}</td>
              <td>${esc(p.goals ?? 0)}</td>
              <td>${esc(p.assists ?? 0)}</td>
              <td>${esc(p.saves ?? 0)}</td>
              <td>${esc(p.shots ?? 0)}</td>
              <td>${esc(p.demos ?? 0)}</td>
              <td>${esc(p.touches ?? 0)}</td>
              <td>${esc(p.score ?? 0)}</td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function dbgHandleLinkedMatchAction(action, linkID) {
  const state = dbgState();
  const link = (state.matchLinks || []).find(item => item.linkID === linkID);
  if (!link) return;
  if (action === 'scenario') {
    state.activeScenario = link.scenarioID;
    dbgSaveState(state);
    dbgRenderScenarios();
    dbgRenderChecks();
    document.getElementById('dbg-active-scenario')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    return;
  }
  if (action === 'match') {
    state.expandedDebugMatchLink = state.expandedDebugMatchLink === linkID ? '' : linkID;
    dbgSaveState(state);
    dbgRenderChecks();
    return;
  }
  if (action === 'unlink') {
    if (!confirm('Unlink this match from the debug scenario? Core History and Session data will not be changed.')) return;
    const scenarioID = link.scenarioID;
    state.matchLinks = (state.matchLinks || []).filter(item => item.linkID !== linkID);
    state.debug_match = false;
    if (state.expandedDebugMatchLink === linkID) state.expandedDebugMatchLink = '';
    if (!state.matchLinks.some(item => item.scenarioID === scenarioID) && state.scenarios?.[scenarioID]) {
      delete state.scenarios[scenarioID].matchLinkedAt;
    }
    dbgSaveState(state);
    dbgRenderScenarios();
    dbgRenderChecks();
  }
}

function dbgHandleEvidenceAction(action, linkID, checkID) {
  const state = dbgState();
  const link = (state.matchLinks || []).find(item => item.linkID === linkID);
  if (!link) return;
  if (action === 'toggle') {
    const key = `${linkID}:${checkID}`;
    state.expandedEvidenceLink = state.expandedEvidenceLink === key ? '' : key;
    dbgSaveState(state);
    dbgRenderChecks();
    return;
  }
  if (action === 'unlink-check') {
    if (!confirm('Remove this linked match evidence from this issue check? Core History and Session data will not be changed.')) return;
    const index = Array.isArray(link.checkIDs) ? link.checkIDs.indexOf(checkID) : -1;
    if (index >= 0) {
      link.checkIDs.splice(index, 1);
      if (Array.isArray(link.checkTitles)) link.checkTitles.splice(index, 1);
    }
    if (!Array.isArray(link.checkIDs) || !link.checkIDs.length) {
      state.matchLinks = (state.matchLinks || []).filter(item => item.linkID !== linkID);
    }
    if (state.expandedEvidenceLink === `${linkID}:${checkID}`) state.expandedEvidenceLink = '';
    dbgSaveState(state);
    dbgRenderScenarios();
    dbgRenderChecks();
  }
}

function dbgMaybePromptScenarioComplete(state, scenarioID) {
  if (!scenarioID) return '';
  const scenario = DBG_SCENARIOS.find(s => s.id === scenarioID);
  const scenarioState = state.scenarios?.[scenarioID];
  if (!scenario || !scenarioState) return '';
  const total = dbgChecksForScenario(scenarioState).length;
  const stats = dbgScenarioStats(scenarioState);
  if (!total || stats.marked < total) return '';
  state.scenarioCompletePrompted = state.scenarioCompletePrompted || {};
  if (state.scenarioCompletePrompted[scenarioID]) {
    return dbgAllDebugScenariosComplete(state) && !state.debugCompletePrompted
      ? dbgMarkDebugComplete(state)
      : '';
  }
  state.scenarioCompletePrompted[scenarioID] = new Date().toISOString();

  if (dbgAllDebugScenariosComplete(state)) return dbgMarkDebugComplete(state);

  const currentIndex = DBG_SCENARIOS.findIndex(s => s.id === scenarioID);
  const nextScenario = DBG_SCENARIOS
    .slice(currentIndex + 1)
    .concat(DBG_SCENARIOS.slice(0, Math.max(currentIndex, 0)))
    .find(s => !dbgIsScenarioComplete(state, s.id));
  if (!nextScenario) return '';
  const moveNext = confirm(`${scenario.title} appears complete.\n\nWould you like to move to ${nextScenario.title}?\n\nOK = move to next track\nCancel = stay and review`);
  if (!moveNext) return '';
  state.activeScenario = nextScenario.id;
  state.scenarios = state.scenarios || {};
  state.scenarios[nextScenario.id] = state.scenarios[nextScenario.id] || { checks: {}, startedAt: new Date().toISOString(), notes: '' };
  state.scenarios[nextScenario.id].checklistType = dbgChecklistTypeForScenario(nextScenario.id);
  state.scenarios[nextScenario.id].scenarioID = nextScenario.id;
  return 'next';
}

function dbgIsScenarioComplete(state, scenarioID) {
  const scenarioState = state.scenarios?.[scenarioID];
  if (!scenarioState) return false;
  const total = dbgChecksForScenario(scenarioState).length;
  const stats = dbgScenarioStats(scenarioState);
  return total > 0 && stats.marked >= total;
}

function dbgAllDebugScenariosComplete(state) {
  return DBG_SCENARIOS.every(scenario => dbgIsScenarioComplete(state, scenario.id));
}

function dbgMarkDebugComplete(state) {
  if (state.debugCompletePrompted) return;
  state.debugCompletePrompted = new Date().toISOString();
  alert([
    'Debug session appears complete.',
    'Review notes.',
    'Export reports.',
    'Save/backup JSON.',
    'Confirm screenshots/captures are attached or named properly.',
  ].join('\n'));
  return 'complete';
}

function dbgMaybePromptDebugComplete(state) {
  if (state.debugCompletePrompted || !dbgAllDebugScenariosComplete(state)) return;
  dbgMarkDebugComplete(state);
}

function dbgLinkedMatchLine(link) {
  const score = link.score ? `${link.score.blue ?? 0}-${link.score.orange ?? 0}` : 'score pending';
  const mode = link.playlistName || link.matchType || 'mode pending';
  const status = link.completed ? `completed via ${link.completionEvent || 'match end'}` : 'pending';
  const source = link.autoTagged ? 'auto-tagged' : 'manual';
  const issues = Array.isArray(link.checkTitles) && link.checkTitles.length
    ? ` | issues: ${link.checkTitles.join('; ')}`
    : '';
  return `${link.startedAt || ''} | ${source} | ${status} | GUID ${link.matchGuid || 'pending'} | History ID ${link.matchID || 'pending'} | ${mode} | ${link.arena || 'arena pending'} | ${score}${issues}`;
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
    touched: marked > 0 || !!scenarioState?.matchLinkedAt || values.some(v => (v.note || '').trim()),
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
      dbgRestoreInternalScroll('dbg-events');
      return;
    }
    root.innerHTML = events.slice().reverse().slice(0, 40).map(e => `
      <div class="dbg-event">
        <div class="dbg-event-time">${esc(new Date(e.at).toLocaleTimeString())}</div>
        <div class="dbg-event-name" title="${esc(e.match_guid || '')}">${esc(e.event)}</div>
        <div class="dbg-event-summary">${esc(e.summary || '')}</div>
      </div>
    `).join('');
    dbgRestoreInternalScroll('dbg-events');
  } catch (e) {
    root.innerHTML = `<div class="dbg-sub">Event load failed: ${esc(e.message || e)}</div>`;
    dbgRestoreInternalScroll('dbg-events');
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

window.handleDebugAssistantEvent = function(msg) {
  if (!msg || !msg.Event) return;
  if (msg.Event === 'UpdateState') {
    DBG_LAST_LIVE_STATE = msg.Data || null;
    dbgUpdateActiveDebugMatch(msg.Data || {});
    return;
  }
  if (msg.Event === 'MatchCreated' || msg.Event === 'MatchInitialized') {
    dbgMaybeAutoLinkMatch(msg.Event, msg.Data || {});
    return;
  }
  if (msg.Event === 'MatchEnded' || msg.Event === 'MatchDestroyed') {
    dbgCompleteLinkedMatch(msg.Event, msg.Data || {});
  }
};

function dbgMatchGuidFromPayload(data, allowLiveFallback = false) {
  return data?.MatchGuid || (allowLiveFallback ? DBG_LAST_LIVE_STATE?.MatchGuid : '') || '';
}

function dbgMatchKey(data, eventName = 'match-start') {
  return dbgMatchGuidFromPayload(data, false) || `${eventName}-${Date.now()}`;
}

function dbgArenaDisplayName(rawArena) {
  if (!rawArena) return '';
  return typeof friendlyArena === 'function' ? friendlyArena(rawArena) : rawArena;
}

function dbgLiveMatchSummary(live) {
  const game = live?.Game || {};
  const teams = game.Teams || [];
  const blue = teams.find(t => t.TeamNum === 0) || teams[0] || {};
  const orange = teams.find(t => t.TeamNum === 1) || teams[1] || {};
  const playlistID = game.Playlist ?? null;
  const playlistName = playlistID != null && typeof friendlyPlaylist === 'function' ? friendlyPlaylist(playlistID) : '';
  const arenaRaw = game.Arena || '';
  return {
    arena: arenaRaw,
    arenaRaw,
    arenaDisplay: dbgArenaDisplayName(arenaRaw),
    playlistID,
    playlistName,
    matchType: typeof matchType === 'function' ? matchType((live?.Players || []).length) : '',
    playerCount: (live?.Players || []).length,
    score: {
      blue: blue.Score ?? 0,
      orange: orange.Score ?? 0,
    },
    players: (live?.Players || []).map(p => ({
      name: p.Name || '',
      primaryID: p.PrimaryId || '',
      teamNum: p.TeamNum ?? 0,
      score: p.Score ?? 0,
      goals: p.Goals ?? 0,
      assists: p.Assists ?? 0,
      saves: p.Saves ?? 0,
      shots: p.Shots ?? 0,
      demos: p.Demos ?? 0,
      touches: p.Touches ?? 0,
    })),
  };
}

function dbgMaybeAutoLinkMatch(eventName, data) {
  if (!dbgIsDebugViewActive()) return;

  const state = dbgState();
  const batch = state.confirmedLinkBatch;
  const scenario = batch?.scenarioID
    ? DBG_SCENARIOS.find(s => s.id === batch.scenarioID)
    : null;
  if (!scenario) {
    state.debug_match = false;
    dbgSaveState(state);
    return;
  }

  const scenarioState = state.scenarios?.[scenario.id] || { checks: {} };
  const progress = dbgScenarioProgressSnapshot(scenarioState);
  if (progress.total > 0 && progress.marked >= progress.total && !batch.allowCompleteScenario) {
    const ok = confirm('This scenario appears complete. Do you still want to tag the next match to this scenario?');
    if (!ok) {
      state.debug_match = false;
      state.confirmedLinkBatch = null;
      state.debugWarnings = state.debugWarnings || [];
      state.debugWarnings.push({
        at: new Date().toISOString(),
        scenarioID: scenario.id,
        scenarioName: scenario.title,
        message: 'Auto-link canceled because selected scenario was already complete.',
      });
      dbgSaveState(state);
      dbgRenderChecks();
      return;
    }
    batch.allowCompleteScenario = true;
  }

  const key = dbgMatchKey(data, eventName);
  state.matchLinks = state.matchLinks || [];
  const existing = state.matchLinks.find(link => !link.completed && (link.matchGuid === key || link.startKey === key));
  if (existing) return;

  const startGuid = dbgMatchGuidFromPayload(data, false);
  const canUseLastLiveState = !!startGuid && DBG_LAST_LIVE_STATE?.MatchGuid === startGuid;
  const liveSummary = dbgLiveMatchSummary(canUseLastLiveState ? DBG_LAST_LIVE_STATE : {});
  state.scenarios = state.scenarios || {};
  state.scenarios[scenario.id] = state.scenarios[scenario.id] || { checks: {}, startedAt: new Date().toISOString() };
  state.scenarios[scenario.id].matchLinkedAt = new Date().toISOString();
  const link = {
    linkID: `link-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    startKey: key,
    matchGuid: startGuid,
    startEvent: eventName,
    scenarioID: scenario.id,
    scenarioName: scenario.title,
    trackName: dbgTrackName(scenario),
    checkIDs: Array.isArray(batch.checkIDs) ? batch.checkIDs : [],
    checkTitles: Array.isArray(batch.checkTitles) ? batch.checkTitles : [],
    confirmedAt: batch.confirmedAt || '',
    startedAt: new Date().toISOString(),
    autoTagged: true,
    progressAtStart: progress,
    completed: false,
    ...liveSummary,
  };
  state.debug_match = true;
  state.matchLinks.push(link);
  state.matchLinks = state.matchLinks.slice(-50);
  state.linkDraft = null;
  state.confirmedLinkBatch = null;
  dbgSaveState(state);
  dbgRenderScenarios();
  dbgRenderChecks();
  dbgMessage(`Linked next match to ${scenario.title} (${link.checkIDs.length} issue check(s))`);
}

function dbgUpdateActiveDebugMatch(live) {
  const state = dbgState();
  const guid = live?.MatchGuid || '';
  const active = [...(state.matchLinks || [])].reverse().find(link => !link.completed && (!guid || link.matchGuid === guid || !link.matchGuid));
  if (!active) return;
  Object.assign(active, dbgLiveMatchSummary(live));
  if (guid) active.matchGuid = guid;
  active.lastUpdatedAt = new Date().toISOString();
  dbgSaveState(state);
}

function dbgCompleteLinkedMatch(eventName, data) {
  const state = dbgState();
  const guid = dbgMatchGuidFromPayload(data, true);
  const link = [...(state.matchLinks || [])].reverse().find(item => !item.completed && (!guid || item.matchGuid === guid || !item.matchGuid));
  if (!link) return;
  link.completed = true;
  link.completionEvent = eventName;
  link.completedAt = new Date().toISOString();
  if (guid) link.matchGuid = guid;
  Object.assign(link, dbgLiveMatchSummary(DBG_LAST_LIVE_STATE || {}));
  dbgSaveState(state);
  dbgAttachHistoryMatch(link.linkID);
  dbgRenderScenarios();
  dbgRenderChecks();
}

async function dbgAttachHistoryMatch(linkID) {
  try {
    const state = dbgState();
    const link = (state.matchLinks || []).find(item => item.linkID === linkID);
    if (!link || !link.matchGuid) return;
    const matches = await fetch('/api/matches').then(r => r.json());
    const match = Array.isArray(matches) ? matches.find(m => m.MatchGUID === link.matchGuid) : null;
    if (!match) return;
    link.matchID = match.ID;
    const arenaRaw = match.Arena || link.arenaRaw || link.arena || '';
    link.arena = link.arena || arenaRaw;
    link.arenaRaw = link.arenaRaw || arenaRaw;
    link.arenaDisplay = link.arenaDisplay || dbgArenaDisplayName(arenaRaw);
    link.score = {
      blue: match.team0_goals ?? link.score?.blue ?? 0,
      orange: match.team1_goals ?? link.score?.orange ?? 0,
    };
    link.historyLinkedAt = new Date().toISOString();
    dbgSaveState(state);
    dbgRenderChecks();
  } catch (_) {
    // Debug metadata only; history lookup failure should never affect app flow.
  }
}

function dbgGenerateReport() {
  const report = dbgBuildPlainReport();
  const root = document.getElementById('dbg-report');
  if (root) root.textContent = report;
  dbgRestoreInternalScroll('dbg-report');
  return report;
}

function dbgBuildPlainReport() {
  const state = dbgState();
  const meta = state.metadata || {};
  const scenario = dbgActiveScenario();
  const scenarioState = scenario ? state.scenarios?.[scenario.id] : null;
  const overall = dbgOverallStats(state);
  const failures = dbgFailureGroups(state);
  const touchedScenarios = DBG_SCENARIOS.filter(s => dbgScenarioStats(state.scenarios?.[s.id]).touched);
  const untestedScenarios = DBG_SCENARIOS.filter(s => !dbgScenarioStats(state.scenarios?.[s.id]).touched);
  const lines = [];
  lines.push('# OOF RL Debug Assistant Report');
  lines.push('');
  lines.push('## Build');
  lines.push(`- **Branch:** ${meta.branch || ''}`);
  lines.push(`- **Commit SHA:** ${meta.sha || ''}`);
  lines.push(`- **EXE:** ${meta.exe || ''}`);
  lines.push(`- **Tester:** ${meta.tester || ''}`);
  lines.push(`- **Intent:** ${meta.intent || ''}`);
  lines.push(`- **Rocket League mode/version:** ${meta['rl-mode'] || ''}`);
  lines.push(`- **Generated:** ${new Date().toLocaleString()}`);
  lines.push('');
  lines.push('## Summary');
  lines.push('| Metric | Value |');
  lines.push('| --- | ---: |');
  lines.push(`| Scenarios touched | ${overall.touched} / ${DBG_SCENARIOS.length} |`);
  lines.push(`| Pass | ${overall.pass} |`);
  lines.push(`| Fail | ${overall.fail} |`);
  lines.push(`| N/A | ${overall.skip} |`);
  lines.push(`| Pass rate | ${overall.pass + overall.fail ? Math.round(overall.pass / (overall.pass + overall.fail) * 100) : 0}% |`);
  lines.push('');
  lines.push('## Failure Groups');
  if (!failures.length) {
    lines.push('- No failed checks recorded.');
  } else {
    for (const group of failures) {
      lines.push(`- **${group.title}**`);
      for (const item of group.items) {
        lines.push(`  - ${item.scenario}${item.note ? `: ${item.note}` : ''}`);
      }
    }
  }
  lines.push('');

  if (scenario) {
    lines.push('## Active Scenario');
    lines.push(`- **Scenario:** ${scenario.title}`);
    lines.push(`- **Suggested evidence:** ${scenario.shots.join(', ')}`);
    const activeLinks = dbgLinkedMatchesForScenario(state, scenario.id);
    lines.push(`- **Linked debug matches:** ${activeLinks.length} total, rendered under their linked checks`);
    lines.push('');
    lines.push('### Active Checklist');
    let activeSection = '';
    for (const check of dbgChecksForScenario(scenarioState)) {
      const item = scenarioState?.checks?.[check.id] || {};
      if (check.section && check.section !== activeSection) {
        lines.push(`- **${check.section}**`);
        activeSection = check.section;
      }
      lines.push(`- **${dbgStatusLabel(item.status)}** - ${check.title}`);
      if (item.note) lines.push(`  - Note: ${item.note}`);
      for (const image of dbgImageNames(item)) lines.push(`  - Screenshot: ${image}`);
      for (const link of dbgLinksForCheck(state, scenario.id, check.id)) {
        lines.push(`  - Linked Match Evidence: ${dbgLinkedMatchLine(link)}`);
      }
    }
    lines.push('');
  }

  lines.push('## Touched Scenario Details');
  if (!touchedScenarios.length) {
    lines.push('- No scenarios touched yet.');
  }
  for (const s of touchedScenarios) {
    const scopedState = state.scenarios?.[s.id];
    const stats = dbgScenarioStats(scopedState);
    lines.push(`- **${s.title}:** ${stats.pass} pass, ${stats.fail} fail, ${stats.skip} N/A, ${stats.percent}% pass rate`);
    const links = dbgLinkedMatchesForScenario(state, s.id);
    if (links.length) {
      lines.push(`  - Linked debug matches: ${links.length} total, rendered under their linked checks`);
    }
    let scopedSection = '';
    for (const check of dbgChecksForScenario(scopedState)) {
      const item = scopedState?.checks?.[check.id] || {};
      const evidenceLinks = dbgLinksForCheck(state, s.id, check.id);
      if (!item.status && !item.note && !item.images && !evidenceLinks.length) continue;
      if (check.section && check.section !== scopedSection) {
        lines.push(`  - **${check.section}**`);
        scopedSection = check.section;
      }
      lines.push(`  - **${dbgStatusLabel(item.status)}** - ${check.title}`);
      if (item.note) lines.push(`    Note: ${item.note}`);
      for (const image of dbgImageNames(item)) lines.push(`    Screenshot: ${image}`);
      for (const link of evidenceLinks) lines.push(`    Linked Match Evidence: ${dbgLinkedMatchLine(link)}`);
    }
  }
  lines.push('');
  lines.push('## Untested Scenarios');
  lines.push(`- ${untestedScenarios.length} remaining: ${untestedScenarios.map(s => s.title).join('; ') || 'none'}`);
  lines.push('');
  lines.push('## Evidence');
  lines.push(`- Snapshots saved locally: ${(state.snapshots || []).length}`);
  lines.push(`- Linked debug matches: ${(state.matchLinks || []).length}`);
  if ((state.debugWarnings || []).length) {
    lines.push('');
    lines.push('## Debug Warnings');
    for (const warning of state.debugWarnings) {
      lines.push(`- ${warning.at || ''} - ${warning.scenarioName || warning.scenarioID || 'scenario'}: ${warning.message}`);
    }
  }
  lines.push('');
  lines.push('## Session Notes');
  lines.push(meta.notes || '');

  return lines.join('\n');
}

function dbgStatusLabel(status) {
  if (status === 'pass') return 'PASS';
  if (status === 'fail') return 'FAIL';
  if (status === 'skip') return 'N/A';
  return 'UNSET';
}

function dbgGenerateDocReport() {
  const html = dbgBuildDocReportHTML(false);
  const root = document.getElementById('dbg-report-doc');
  if (root) root.innerHTML = html;
  dbgRestoreInternalScroll('dbg-report-doc');
  return html;
}

function dbgBuildDocReportHTML(exportMode) {
  const state = dbgState();
  const meta = state.metadata || {};
  const overall = dbgOverallStats(state);
  const failures = dbgFailureGroups(state);
  const touchedScenarios = DBG_SCENARIOS.filter(s => dbgScenarioStats(state.scenarios?.[s.id]).touched);
  const untestedScenarios = DBG_SCENARIOS.filter(s => !dbgScenarioStats(state.scenarios?.[s.id]).touched);
  const parts = [];
  parts.push('<h2>OOF RL Debug Assistant Report</h2>');
  parts.push('<section class="report-card">');
  parts.push('<h3>Build</h3>');
  parts.push('<dl class="report-meta">');
  parts.push(`<dt>Branch</dt><dd>${esc(meta.branch || '')}</dd>`);
  parts.push(`<dt>Commit SHA</dt><dd>${esc(meta.sha || '')}</dd>`);
  parts.push(`<dt>EXE</dt><dd>${esc(meta.exe || '')}</dd>`);
  parts.push(`<dt>Tester</dt><dd>${esc(meta.tester || '')}</dd>`);
  parts.push(`<dt>Intent</dt><dd>${esc(meta.intent || '')}</dd>`);
  parts.push(`<dt>Rocket League mode/version</dt><dd>${esc(meta['rl-mode'] || '')}</dd>`);
  parts.push(`<dt>Generated</dt><dd>${esc(new Date().toLocaleString())}</dd>`);
  parts.push('</dl>');
  parts.push('</section>');

  parts.push('<section class="report-card">');
  parts.push('<h3>Summary</h3>');
  parts.push('<div class="report-summary">');
  parts.push(`<div><strong>${overall.touched} / ${DBG_SCENARIOS.length}</strong><span>Scenarios touched</span></div>`);
  parts.push(`<div><strong class="pass">${overall.pass}</strong><span>Pass</span></div>`);
  parts.push(`<div><strong class="fail">${overall.fail}</strong><span>Fail</span></div>`);
  parts.push(`<div><strong class="skip">${overall.skip}</strong><span>N/A</span></div>`);
  parts.push(`<div><strong>${overall.pass + overall.fail ? Math.round(overall.pass / (overall.pass + overall.fail) * 100) : 0}%</strong><span>Pass rate</span></div>`);
  parts.push('</div>');
  parts.push('</section>');

  parts.push('<section class="report-card">');
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
  parts.push('</section>');

  parts.push('<section class="report-card">');
  parts.push('<h3>Scenario Details</h3>');
  if (!touchedScenarios.length) {
    parts.push('<p>No scenarios touched yet.</p>');
  }
  for (const scenario of touchedScenarios) {
    const scenarioState = state.scenarios?.[scenario.id];
    const stats = dbgScenarioStats(scenarioState);
    parts.push(`<h4>${esc(scenario.title)}</h4>`);
    parts.push(`<p>${stats.pass} pass, ${stats.fail} fail, ${stats.skip} N/A, ${stats.percent}% pass rate</p>`);
    const links = dbgLinkedMatchesForScenario(state, scenario.id);
    if (links.length) {
      parts.push(`<p><strong>Linked debug matches:</strong> ${links.length} total, rendered under their linked checks.</p>`);
    }
    let docSection = '';
    let docListOpen = false;
    for (const check of dbgChecksForScenario(scenarioState)) {
      const item = scenarioState?.checks?.[check.id] || {};
      const evidenceLinks = dbgLinksForCheck(state, scenario.id, check.id);
      if (!item.status && !item.note && !item.images && !evidenceLinks.length) continue;
      if (check.section && check.section !== docSection) {
        if (docListOpen) parts.push('</ul>');
        parts.push(`<h5>${esc(check.section)}</h5><ul>`);
        docSection = check.section;
        docListOpen = true;
      } else if (!docListOpen) {
        parts.push('<ul>');
        docListOpen = true;
      }
      const statusClass = item.status === 'pass' ? 'pass' : item.status === 'fail' ? 'fail' : item.status === 'skip' ? 'skip' : '';
      parts.push(`<li><span class="${statusClass}">[${esc(item.status || 'unset')}]</span> ${esc(check.title)}${item.note ? `<br><em>${esc(item.note)}</em>` : ''}</li>`);
      for (const link of evidenceLinks) {
        parts.push(`<li><strong>Linked Match Evidence:</strong> ${esc(dbgLinkedMatchLine(link))}</li>`);
      }
      for (const image of dbgImageNames(item)) {
        const href = exportMode ? dbgExportScreenshotPath(image) : dbgScreenshotURL(image);
        parts.push(`<div class="shot-link"><a href="${href}" target="_blank" rel="noopener">Open screenshot: ${esc(image)}</a></div>`);
        if (!exportMode) parts.push(`<img alt="${esc(image)}" src="${href}" onerror="this.style.display='none'">`);
      }
    }
    if (docListOpen) parts.push('</ul>');
  }
  parts.push('</section>');

  parts.push('<section class="report-card">');
  parts.push('<h3>Untested Scenarios</h3>');
  parts.push(`<p>${untestedScenarios.length} remaining${untestedScenarios.length ? `: ${esc(untestedScenarios.map(s => s.title).join('; '))}` : ': none'}</p>`);
  parts.push('</section>');

  if (meta.notes) {
    parts.push('<section class="report-card">');
    parts.push('<h3>Session Notes</h3>');
    parts.push(`<p>${esc(meta.notes)}</p>`);
    parts.push('</section>');
  }
  if ((state.debugWarnings || []).length) {
    parts.push('<section class="report-card">');
    parts.push('<h3>Debug Warnings</h3><ul>');
    for (const warning of state.debugWarnings) {
      parts.push(`<li>${esc(warning.at || '')} - <strong>${esc(warning.scenarioName || warning.scenarioID || 'scenario')}</strong>: ${esc(warning.message || '')}</li>`);
    }
    parts.push('</ul></section>');
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
.report-card { background:#11151d; border:1px solid #273244; border-radius:10px; padding:16px; margin:16px 0; }
.report-card h3 { margin-top:0; }
.report-meta { display:grid; grid-template-columns:210px 1fr; gap:8px 14px; margin:0; }
.report-meta dt { color:#94a3b8; font-weight:800; }
.report-meta dd { margin:0; overflow-wrap:anywhere; }
.report-summary { display:grid; grid-template-columns:repeat(auto-fit,minmax(135px,1fr)); gap:10px; }
.report-summary div { background:#0c1017; border:1px solid #273244; border-radius:8px; padding:10px; }
.report-summary strong { display:block; font-size:1.35rem; }
.report-summary span { display:block; color:#94a3b8; font-size:.78rem; font-weight:800; text-transform:uppercase; }
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
  document.getElementById('dbg-import-file')?.addEventListener('change', dbgImportJSONState);
  document.getElementById('dbg-add-condition')?.addEventListener('click', dbgAddCustomCheck);
  document.getElementById('dbg-reset')?.addEventListener('click', async () => {
    if (!confirm('Reset local Debug Assistant metadata, checklist, snapshots, and notes?')) return;
    await dbgResetLocalState();
  });
}

async function dbgResetLocalState() {
  localStorage.removeItem(DBG_STORAGE_KEY);
  for (const key of Object.keys(DBG_SCROLL_POSITIONS)) delete DBG_SCROLL_POSITIONS[key];
  DBG_LAST_LIVE_STATE = null;
  await fetch('/api/debug-assistant/reset', {method: 'POST'}).catch(() => null);

  const report = document.getElementById('dbg-report');
  if (report) report.textContent = 'Generate a report after your scenario or session.';
  const docReport = document.getElementById('dbg-report-doc');
  if (docReport) docReport.textContent = 'Generate a doc report to preview a developer-readable report with tagged screenshots.';
  const exportResult = document.getElementById('dbg-export-result');
  if (exportResult) {
    exportResult.classList.remove('visible');
    exportResult.innerHTML = '';
  }
  const importInput = document.getElementById('dbg-import-file');
  if (importInput) importInput.value = '';

  dbgLoadMeta();
  dbgRenderScenarios();
  dbgRenderChecks();
  await Promise.all([dbgLoadContext(), dbgLoadEvents()]);
  dbgRefreshWidgetInstances();
  dbgMessage('Debug Assistant reset to default state');
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
      dbgShowExportResult(result);
      dbgMessage(result.message || 'Report already exported. Duplicate export skipped.');
      return;
    }
    dbgShowExportResult(result);
    dbgMessage(`Exported reports to ${result.dir}`);
  } catch (e) {
    dbgMessage(`Export failed: ${e.message || e}`);
  }
}

async function dbgImportJSONState(event) {
  const file = event.target.files?.[0];
  event.target.value = '';
  if (!file) return;
  if (!confirm(`Import "${file.name}" as the current Debug Assistant state? This replaces the local checklist state.`)) {
    dbgMessage('Import cancelled');
    return;
  }
  try {
    const imported = JSON.parse(await file.text());
    dbgValidateImportedState(imported);
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

function dbgValidateImportedState(imported) {
  if (!imported || typeof imported !== 'object' || Array.isArray(imported)) {
    throw new Error('Selected file is not a valid Debug Assistant state object');
  }
  const hasDebugShape = imported.metadata || imported.scenarios || imported.snapshots || imported.activeScenario;
  if (!hasDebugShape) {
    throw new Error('Selected JSON does not look like a Debug Assistant state export');
  }
  if (imported.scenarios && (typeof imported.scenarios !== 'object' || Array.isArray(imported.scenarios))) {
    throw new Error('Debug Assistant scenarios must be a JSON object');
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
