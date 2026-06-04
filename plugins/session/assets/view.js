'use strict';

let _sessionSince           = null;
let _sessionPlayerID        = localStorage.getItem('oof_session_player') || '';
let _sessionElapsedTimer    = null;
let _sessionExpandedMatchId = null;
let _liveStats              = null; // non-null only while a match is active

let _sessionSummaryInstances = [];
let _sessionLiveInstances    = [];
let _sessionMatchInstances   = [];
let _sessionHistoryInstances = [];

window.pluginInit_session = async function() {
  window.registerWidget?.({
    id: 'session-summary', pluginId: 'session', title: 'Session Stats',
    defaultW: 6, defaultH: 6, minW: 4, minH: 4,
    factory: sessionSummaryWidget,
  });
  window.registerWidget?.({
    id: 'session-live-game', pluginId: 'session', title: 'Live Game Stats',
    defaultW: 4, defaultH: 4, minW: 3, minH: 3,
    factory: sessionLiveGameWidget,
  });
  window.registerWidget?.({
    id: 'session-match-list', pluginId: 'session', title: 'Session Matches',
    defaultW: 6, defaultH: 8, minW: 4, minH: 5,
    factory: sessionMatchesWidget,
  });
  window.registerWidget?.({
    id: 'session-previous-sessions', pluginId: 'session', title: 'Previous Sessions',
    defaultW: 6, defaultH: 7, minW: 4, minH: 5,
    factory: sessionHistoryWidget,
  });

  _sessionPlayerID = localStorage.getItem('oof_session_player') || '';

  // Fetch server-side session start time
  try {
    const data = await fetch('/api/session/start').then(r => r.json());
    _sessionSince = data.active ? new Date(data.since) : null;
  } catch(_) {
    _sessionSince = null;
  }

  // Populate player dropdown
  try {
    const players = await fetch('/api/players').then(r => r.json());
    populateSessionPlayerSelect(players || []);
  } catch(_) {}

  updateSinceInput();
  startElapsedTimer();

  // Auto-detect player suggestion (only when no player saved)
  if (!_sessionPlayerID) {
    try {
      const s = await fetch('/api/session/suggest-player').then(r => r.json());
      if (s.primary_id) {
        const banner = document.getElementById('session-suggest-banner');
        document.getElementById('session-suggest-name').textContent = s.name || s.primary_id;
        banner?.classList.remove('hidden');

        document.getElementById('session-suggest-yes').addEventListener('click', () => {
          _sessionPlayerID = s.primary_id;
          localStorage.setItem('oof_session_player', _sessionPlayerID);
          const sel = document.getElementById('session-player-select');
          let found = false;
          for (const opt of sel.options) {
            if (opt.value === s.primary_id) { opt.selected = true; found = true; break; }
          }
          if (!found) {
            const opt = document.createElement('option');
            opt.value = s.primary_id;
            opt.textContent = s.name || s.primary_id;
            opt.selected = true;
            sel.appendChild(opt);
          }
          banner?.classList.add('hidden');
          refreshSession();
          loadSessionHistory();
        });

        document.getElementById('session-suggest-dismiss').addEventListener('click', () => {
          banner?.classList.add('hidden');
        });
      }
    } catch(_) {}
  }

  document.getElementById('session-player-select').addEventListener('change', e => {
    _sessionPlayerID = e.target.value;
    localStorage.setItem('oof_session_player', _sessionPlayerID);
    refreshSession();
    loadSessionHistory();
  });

  // "Start Now" manually starts the session
  document.getElementById('session-start-btn')?.addEventListener('click', async () => {
    try {
      const data = await fetch('/api/session/start', { method: 'POST' }).then(r => r.json());
      _sessionSince = new Date(data.since);
      updateSinceInput();
      startElapsedTimer();
      refreshSession();
    } catch(_) {}
  });

  // "New session" saves current session to history then resets start time
  document.getElementById('session-reset-btn').addEventListener('click', async () => {
    try {
      const data = await fetch('/api/session/new', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ player_id: _sessionPlayerID }),
      }).then(r => r.json());
      _sessionSince = new Date(data.since);
      updateSinceInput();
      updateElapsed();
      refreshSession();
      loadSessionHistory();
    } catch(_) {}
  });

  // Manual datetime picker override
  document.getElementById('session-since-input')?.addEventListener('change', async e => {
    const t = new Date(e.target.value);
    if (isNaN(t.getTime())) return;
    try {
      const data = await fetch('/api/session/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ since: t.toISOString() }),
      }).then(r => r.json());
      _sessionSince = new Date(data.since);
      updateElapsed();
      refreshSession();
    } catch(_) {}
  });

  // Auto-refresh every 30 seconds
  setInterval(refreshSession, 30000);

  refreshSession();
  loadSessionHistory();
};

function toDatetimeLocal(d) {
  const pad = n => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function populateSessionPlayerSelect(players) {
  const sel = document.getElementById('session-player-select');
  if (!sel) return;

  const currentID = _sessionPlayerID;
  sel.innerHTML = '<option value="">Select player...</option>';

  let hasCurrent = !currentID;
  for (const p of players) {
    if (!p?.PrimaryID) continue;
    const opt = document.createElement('option');
    opt.value = p.PrimaryID;
    opt.textContent = p.Name || p.PrimaryID;
    sel.appendChild(opt);
    if (p.PrimaryID === currentID) hasCurrent = true;
  }

  if (currentID && hasCurrent) {
    sel.value = currentID;
  } else if (currentID) {
    _sessionPlayerID = '';
    localStorage.removeItem('oof_session_player');
    sel.value = '';
  }
}

function updateSinceInput() {
  const el = document.getElementById('session-since-input');
  if (el && _sessionSince) el.value = toDatetimeLocal(_sessionSince);
}

function startElapsedTimer() {
  if (_sessionElapsedTimer) clearInterval(_sessionElapsedTimer);
  _sessionElapsedTimer = setInterval(updateElapsed, 30000);
  updateElapsed();
}

function updateElapsed() {
  const el = document.getElementById('session-elapsed');
  if (!el || !_sessionSince) return;
  const secs = Math.floor((Date.now() - _sessionSince.getTime()) / 1000);
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  el.textContent = h > 0 ? `${h}h ${m}m` : `${m}m`;
}

window.refreshSession = async function() {
  const noPlayer   = document.getElementById('session-no-player');
  const notStarted = document.getElementById('session-not-started');
  const panel      = document.getElementById('session-stats-panel');

  if (!_sessionPlayerID) {
    noPlayer?.classList.remove('hidden');
    notStarted?.classList.add('hidden');
    panel?.classList.add('hidden');
    renderLiveGame();
    _sessionLiveInstances.forEach(w => w.render());
    _sessionSummaryInstances.forEach(w => w.renderPlaceholder('Select a player in the Session tab to see stats.'));
    _sessionMatchInstances.forEach(w => w.renderPlaceholder('Select a player in the Session tab to see matches.'));
    _sessionHistoryInstances.forEach(w => w.renderPlaceholder('Select a player in the Session tab to see previous sessions.'));
    return;
  }
  noPlayer?.classList.add('hidden');

  if (!_sessionSince) {
    notStarted?.classList.remove('hidden');
    panel?.classList.add('hidden');
    renderLiveGame();
    _sessionLiveInstances.forEach(w => w.render());
    _sessionSummaryInstances.forEach(w => w.renderPlaceholder('No active session. Use the Session tab to start one.'));
    _sessionMatchInstances.forEach(w => w.renderPlaceholder('No active session yet.'));
    _sessionHistoryInstances.forEach(w => w.refresh());
    return;
  }
  notStarted?.classList.add('hidden');
  panel?.classList.remove('hidden');
  renderLiveGame();
  _sessionLiveInstances.forEach(w => w.render());

  try {
    const url  = `/api/session/stats?player=${encodeURIComponent(_sessionPlayerID)}`;
    const data = await fetch(url).then(r => r.json());
    renderSessionStats(data);
    _sessionSummaryInstances.forEach(w => w.renderData(data));
    _sessionMatchInstances.forEach(w => w.renderData(data));
  } catch(_) {}
};

// Called on every UpdateState WS message — extracts tracked player's current-game stats.
window.handleSessionUpdate = function(data) {
  if (!_sessionPlayerID) return;
  const players = data.Players || [];
  const me = players.find(p => p.PrimaryId === _sessionPlayerID);
  if (!me) return;
  _liveStats = {
    goals:   me.Goals   || 0,
    assists: me.Assists || 0,
    saves:   me.Saves   || 0,
    shots:   me.Shots   || 0,
    demos:   me.Demos   || 0,
  };
  renderLiveGame();
  _sessionLiveInstances.forEach(w => w.render());
};

// Called on MatchCreated/MatchInitialized — if session wasn't started yet, fetch the
// server-set start time and activate the UI.
window.handleSessionMatchStart = async function() {
  if (_sessionSince) return;
  try {
    const data = await fetch('/api/session/start').then(r => r.json());
    if (data.active) {
      _sessionSince = new Date(data.since);
      updateSinceInput();
      startElapsedTimer();
      refreshSession();
    }
  } catch(_) {}
};

// Called on MatchDestroyed — clears live stats before refreshSession re-renders from DB.
window.clearSessionLive = function() {
  _liveStats = null;
  renderLiveGame();
  _sessionLiveInstances.forEach(w => w.renderEmpty());
};

function renderLiveGame() {
  const el = document.getElementById('session-live-game');
  if (!el) return;
  if (!_liveStats || !_sessionPlayerID) {
    el.classList.add('hidden');
    return;
  }
  el.classList.remove('hidden');
  set('live-game-goals',   _liveStats.goals);
  set('live-game-assists', _liveStats.assists);
  set('live-game-saves',   _liveStats.saves);
  set('live-game-shots',   _liveStats.shots);
  set('live-game-demos',   _liveStats.demos);
}

function renderSessionStats(data) {
  const s       = data.summary || {};
  const matches = data.matches || [];

  set('session-games',   s.games   || 0);
  set('session-wins',    s.wins    || 0);
  set('session-losses',  s.losses  || 0);
  set('session-goals',   s.goals   || 0);
  set('session-assists', s.assists || 0);
  set('session-saves',   s.saves   || 0);
  set('session-shots',   s.shots   || 0);
  set('session-demos',   s.demos   || 0);

  renderMatchCards(matches);
}

function renderMatchCards(matches) {
  const listEl = document.getElementById('session-match-list');
  if (!listEl) return;

  _sessionExpandedMatchId = null;

  if (!matches.length) {
    listEl.innerHTML = '<p class="session-empty-message">No matches in this session yet.</p>';
    return;
  }

  listEl.innerHTML = '';
  for (const m of [...matches].reverse()) {
    const finished  = !m.incomplete && m.winner_team_num >= 0;
    const won       = finished && m.player_team === m.winner_team_num;
    const lost      = finished && m.player_team !== m.winner_team_num;
    const resultCls = m.incomplete ? 'text-gray-500' : won ? 'text-green-400' : lost ? 'text-red-400' : 'text-gray-500';
    const rowState  = m.incomplete ? 'incomplete' : won ? 'win' : lost ? 'loss' : 'neutral';
    const resultTxt = m.incomplete ? '?' : won ? 'W' : lost ? 'L' : '—';

    const card = document.createElement('div');
    card.className = `session-match-card ${rowState}`;
    card.dataset.matchId = m.match_id;
    const matchTypeStr = friendlyPlaylist(m.playlist_type) || matchType(m.player_count);
    card.innerHTML = `
      <span class="session-row-result ${resultCls}">${resultTxt}</span>
      <div class="session-row-main">
        <div class="session-row-title">
          ${esc(friendlyArena(m.arena))}
          ${matchTypeStr ? `<span class="match-mode-badge">${esc(matchTypeStr)}</span>` : ''}
          ${m.overtime   ? '<span class="match-mode-badge match-mode-ot">OT</span>' : ''}
          ${m.forfeit    ? '<span class="match-mode-badge" style="background:rgba(234,179,8,0.12);color:#ca8a04">FF</span>' : ''}
          ${m.incomplete ? '<span class="match-mode-badge" style="background:rgba(156,163,175,0.12);color:#9ca3af">Inc</span>' : ''}
        </div>
        <div class="session-row-meta">${formatDate(m.started_at)}</div>
      </div>
      <div class="session-row-stats">
        <div>${m.goals}G ${m.assists}A ${m.saves}Sv</div>
        <small>Score ${m.score}</small>
      </div>
      <span class="session-chevron">›</span>
    `;
    card.addEventListener('click', () => toggleSessionMatchInline(card, m.match_id));
    listEl.appendChild(card);
  }
}

async function toggleSessionMatchInline(card, matchId) {
  const existing = card.nextElementSibling;
  if (existing && existing.classList.contains('session-match-panel')) {
    existing.remove();
    card.querySelector('.session-chevron').style.transform = '';
    _sessionExpandedMatchId = null;
    return;
  }

  document.querySelectorAll('.session-match-panel').forEach(el => el.remove());
  document.querySelectorAll('.session-chevron').forEach(el => { el.style.transform = ''; });
  _sessionExpandedMatchId = matchId;
  card.querySelector('.session-chevron').style.transform = 'rotate(90deg)';

  const panel = document.createElement('div');
  panel.className = 'session-match-panel match-inline-panel session-inline-detail';
  panel.innerHTML = '<div class="session-inline-status">Loading...</div>';
  card.insertAdjacentElement('afterend', panel);

  await loadSessionMatchDetail(matchId, panel);
}

async function loadSessionMatchDetail(matchId, panel) {
  try {
    const data = await fetch(`/api/matches/${matchId}`).then(r => r.json());
    window.renderMatchDetailPanel(data, panel, _sessionExpandedMatchId, matchId);
  } catch(_) {
    panel.innerHTML = '<div class="session-inline-status">Failed to load match detail.</div>';
  }
}

async function loadSessionHistory() {
  const section = document.getElementById('session-history-section');
  const list    = document.getElementById('session-history-list');
  if (!section || !list) return;
  if (!_sessionPlayerID) {
    section.classList.add('hidden');
    _sessionHistoryInstances.forEach(w => w.renderPlaceholder('Select a player in the Session tab to see previous sessions.'));
    return;
  }

  try {
    const sessions = await fetch(`/api/session/history?player=${encodeURIComponent(_sessionPlayerID)}`).then(r => r.json());
    _sessionHistoryInstances.forEach(w => w.renderData(sessions || []));
    if (!sessions || !sessions.length) {
      section.classList.add('hidden');
      return;
    }
    section.classList.remove('hidden');
    list.innerHTML = '';
    for (const s of sessions) {
      list.appendChild(buildSessionHistoryCard(s));
    }
  } catch(_) {
    section.classList.add('hidden');
    _sessionHistoryInstances.forEach(w => w.renderPlaceholder('Failed to load previous sessions.'));
  }
}

function buildSessionHistoryCard(s) {
  const startDate = new Date(s.started_at);
  const endDate   = new Date(s.ended_at);
  const durMins   = Math.round((endDate - startDate) / 60000);
  const h = Math.floor(durMins / 60);
  const m = durMins % 60;
  const durStr = h > 0 ? `${h}h ${m}m` : `${m}m`;

  const wrapper = document.createElement('div');

  const card = document.createElement('div');
  card.className = 'session-history-card';
  card.innerHTML = `
    <div class="session-history-card-row">
      <div class="session-history-card-main">
        <div class="session-history-card-title">${formatDate(s.started_at)}</div>
        <div class="session-history-card-meta">
          ${durStr} &nbsp;·&nbsp;
          <span class="session-win-text">${s.wins}W</span> <span class="session-loss-text">${s.losses}L</span>
          &nbsp;·&nbsp; ${s.goals}G ${s.assists}A ${s.saves}Sv ${s.shots}Sh ${s.demos}Dm
        </div>
      </div>
      <span class="sess-hist-chevron">›</span>
    </div>
  `;

  const editPanel = document.createElement('div');
  editPanel.className = 'hidden session-history-edit-panel';
  editPanel.innerHTML = `
    <div class="session-history-edit-grid">
      <div>
        <label class="ui-label">Start</label>
        <input type="datetime-local" class="ui-input sess-hist-start" value="${toDatetimeLocal(startDate)}">
      </div>
      <div>
        <label class="ui-label">End</label>
        <input type="datetime-local" class="ui-input sess-hist-end" value="${toDatetimeLocal(endDate)}">
      </div>
    </div>
    <div class="session-history-edit-actions">
      <button class="ui-button ui-button--primary sess-hist-save" type="button">Save</button>
      <button class="ui-button sess-hist-delete" type="button">Delete</button>
      <span class="sess-hist-msg hidden"></span>
    </div>
  `;

  card.addEventListener('click', () => {
    const open = !editPanel.classList.contains('hidden');
    if (open) {
      editPanel.classList.add('hidden');
      card.classList.remove('rounded-b-none');
      card.querySelector('.sess-hist-chevron').style.transform = '';
    } else {
      editPanel.classList.remove('hidden');
      card.classList.add('rounded-b-none');
      card.querySelector('.sess-hist-chevron').style.transform = 'rotate(90deg)';
    }
  });

  editPanel.querySelector('.sess-hist-save').addEventListener('click', async e => {
    e.stopPropagation();
    const startVal = editPanel.querySelector('.sess-hist-start').value;
    const endVal   = editPanel.querySelector('.sess-hist-end').value;
    if (!startVal || !endVal) return;
    const msg = editPanel.querySelector('.sess-hist-msg');
    try {
      const res = await fetch(`/api/session/history/${s.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          started_at: new Date(startVal).toISOString(),
          ended_at:   new Date(endVal).toISOString(),
        }),
      });
      if (res.ok) {
        msg.textContent = 'Saved!';
        msg.className = 'sess-hist-msg session-message success';
        msg.classList.remove('hidden');
        setTimeout(() => msg.classList.add('hidden'), 3000);
        loadSessionHistory();
      } else {
        const err = await res.json().catch(() => ({}));
        msg.textContent = err.error || 'Error saving.';
        msg.className = 'sess-hist-msg session-message error';
        msg.classList.remove('hidden');
      }
    } catch(_) {
      msg.textContent = 'Request failed.';
      msg.className = 'sess-hist-msg session-message error';
      msg.classList.remove('hidden');
    }
  });

  editPanel.querySelector('.sess-hist-delete').addEventListener('click', async e => {
    e.stopPropagation();
    if (!confirm('Delete this session from history?')) return;
    try {
      await fetch(`/api/session/history/${s.id}`, { method: 'DELETE' });
      loadSessionHistory();
    } catch(_) {}
  });

  wrapper.appendChild(card);
  wrapper.appendChild(editPanel);
  return wrapper;
}

function set(id, val) {
  const el = document.getElementById(id);
  if (el) el.textContent = val;
}

function _statPill(label, value, valueStyle) {
  return `<div style="background:var(--surface2);border-radius:8px;padding:8px 4px;text-align:center">
    <div style="font-size:18px;font-weight:700;font-variant-numeric:tabular-nums;${valueStyle}">${value}</div>
    <div style="font-size:10px;color:var(--muted);margin-top:2px">${label}</div>
  </div>`;
}

function sessionSummaryWidget(container) {
  function renderPlaceholder(msg) {
    container.innerHTML = `<div style="text-align:center;color:var(--muted);padding:32px 8px;font-size:13px">${msg}</div>`;
  }

  function renderData(data) {
    const s = data.summary || {};
    let elapsedStr = '';
    if (_sessionSince) {
      const secs = Math.floor((Date.now() - _sessionSince.getTime()) / 1000);
      const h = Math.floor(secs / 3600);
      const m = Math.floor((secs % 3600) / 60);
      elapsedStr = h > 0 ? `${h}h ${m}m` : `${m}m`;
    }
    container.innerHTML = `<div style="padding:8px">
      ${elapsedStr ? `<div style="text-align:center;font-size:11px;color:var(--muted);margin-bottom:8px">Session: ${elapsedStr}</div>` : ''}
      <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:6px;margin-bottom:6px">
        ${_statPill('Games',  s.games  || 0, '')}
        ${_statPill('Wins',   s.wins   || 0, 'color:var(--rl-blue)')}
        ${_statPill('Losses', s.losses || 0, 'color:#f87171')}
      </div>
      <div style="display:grid;grid-template-columns:repeat(5,1fr);gap:6px">
        ${_statPill('G',  s.goals   || 0, '')}
        ${_statPill('A',  s.assists || 0, '')}
        ${_statPill('Sv', s.saves   || 0, '')}
        ${_statPill('Sh', s.shots   || 0, '')}
        ${_statPill('Dm', s.demos   || 0, '')}
      </div>
    </div>`;
  }

  async function refresh() {
    if (!_sessionPlayerID) {
      renderPlaceholder('Select a player in the Session tab to see stats.');
      return;
    }
    if (!_sessionSince) {
      renderPlaceholder('No active session. Use the Session tab to start one.');
      return;
    }
    try {
      const data = await fetch(`/api/session/stats?player=${encodeURIComponent(_sessionPlayerID)}`).then(r => r.json());
      renderData(data);
    } catch(_) {
      renderPlaceholder('Failed to load session stats.');
    }
  }

  function destroy() {
    const i = _sessionSummaryInstances.indexOf(entry);
    if (i >= 0) _sessionSummaryInstances.splice(i, 1);
  }

  const entry = { refresh, renderData, renderPlaceholder };
  _sessionSummaryInstances.push(entry);
  return { refresh, destroy };
}

function sessionLiveGameWidget(container) {
  function renderEmpty() {
    const msg = !_sessionPlayerID
      ? 'Select a player in the Session tab.'
      : 'Waiting for a match…';
    container.innerHTML = `<div style="text-align:center;color:var(--muted);padding:32px 8px;font-size:13px">${msg}</div>`;
  }

  function render() {
    if (!_liveStats || !_sessionPlayerID) { renderEmpty(); return; }
    container.innerHTML = `<div style="padding:8px">
      <div style="text-align:center;font-size:11px;color:var(--muted);margin-bottom:8px">Current match</div>
      <div style="display:grid;grid-template-columns:repeat(5,1fr);gap:6px">
        ${_statPill('Goals',   _liveStats.goals,   '')}
        ${_statPill('Assists', _liveStats.assists, '')}
        ${_statPill('Saves',   _liveStats.saves,   '')}
        ${_statPill('Shots',   _liveStats.shots,   '')}
        ${_statPill('Demos',   _liveStats.demos,   '')}
      </div>
    </div>`;
  }

  function destroy() {
    const i = _sessionLiveInstances.indexOf(entry);
    if (i >= 0) _sessionLiveInstances.splice(i, 1);
  }

  const entry = { render, renderEmpty };
  _sessionLiveInstances.push(entry);
  render();
  return { refresh: render, destroy };
}

function sessionMatchesWidget(container) {
  function renderPlaceholder(msg) {
    container.innerHTML = `<div class="ui-widget-empty">${esc(msg)}</div>`;
  }

  function renderData(data) {
    const matches = data.matches || [];
    if (!matches.length) {
      renderPlaceholder('No matches in this session yet.');
      return;
    }

    container.innerHTML = `<div style="display:flex;flex-direction:column;gap:6px">
      ${[...matches].reverse().slice(0, 8).map(sessionWidgetMatchRow).join('')}
    </div>`;
  }

  async function refresh() {
    if (!_sessionPlayerID) {
      renderPlaceholder('Select a player in the Session tab to see matches.');
      return;
    }
    if (!_sessionSince) {
      renderPlaceholder('No active session yet.');
      return;
    }
    try {
      const data = await fetch(`/api/session/stats?player=${encodeURIComponent(_sessionPlayerID)}`).then(r => r.json());
      renderData(data);
    } catch(_) {
      renderPlaceholder('Failed to load session matches.');
    }
  }

  function destroy() {
    const i = _sessionMatchInstances.indexOf(entry);
    if (i >= 0) _sessionMatchInstances.splice(i, 1);
  }

  const entry = { refresh, renderData, renderPlaceholder };
  _sessionMatchInstances.push(entry);
  return { refresh, destroy };
}

function sessionHistoryWidget(container) {
  function renderPlaceholder(msg) {
    container.innerHTML = `<div class="ui-widget-empty">${esc(msg)}</div>`;
  }

  function renderData(sessions) {
    if (!sessions || !sessions.length) {
      renderPlaceholder('No previous sessions yet.');
      return;
    }

    container.innerHTML = `<div style="display:flex;flex-direction:column;gap:6px">
      ${sessions.slice(0, 8).map(sessionWidgetHistoryRow).join('')}
    </div>`;
  }

  async function refresh() {
    if (!_sessionPlayerID) {
      renderPlaceholder('Select a player in the Session tab to see previous sessions.');
      return;
    }
    try {
      const sessions = await fetch(`/api/session/history?player=${encodeURIComponent(_sessionPlayerID)}`).then(r => r.json());
      renderData(sessions || []);
    } catch(_) {
      renderPlaceholder('Failed to load previous sessions.');
    }
  }

  function destroy() {
    const i = _sessionHistoryInstances.indexOf(entry);
    if (i >= 0) _sessionHistoryInstances.splice(i, 1);
  }

  const entry = { refresh, renderData, renderPlaceholder };
  _sessionHistoryInstances.push(entry);
  return { refresh, destroy };
}

function sessionWidgetMatchRow(m) {
  const finished = !m.incomplete && m.winner_team_num >= 0;
  const won = finished && m.player_team === m.winner_team_num;
  const lost = finished && m.player_team !== m.winner_team_num;
  const resultText = m.incomplete ? '?' : won ? 'W' : lost ? 'L' : '-';
  const resultColor = won ? 'var(--green)' : lost ? 'var(--red)' : 'var(--muted)';
  const mode = friendlyPlaylist(m.playlist_type) || matchType(m.player_count);
  return `<div style="display:grid;grid-template-columns:28px minmax(0,1fr) auto;align-items:center;gap:10px;padding:8px 10px;background:var(--surface2);border:1px solid var(--line);border-radius:8px">
    <div style="font-size:18px;font-weight:800;color:${resultColor};text-align:center">${resultText}</div>
    <div style="min-width:0">
      <div style="font-size:12px;font-weight:700;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(friendlyArena(m.arena))}</div>
      <div style="font-size:10px;color:var(--muted);margin-top:2px">${esc(formatDate(m.started_at))}${mode ? ` &middot; ${esc(mode)}` : ''}</div>
    </div>
    <div style="text-align:right;font-size:11px;color:var(--muted);font-variant-numeric:tabular-nums">
      <div style="color:var(--text)">${m.goals}G ${m.assists}A ${m.saves}Sv</div>
      <div>Score ${m.score}</div>
    </div>
  </div>`;
}

function sessionWidgetHistoryRow(s) {
  const startDate = new Date(s.started_at);
  const endDate = new Date(s.ended_at);
  const durMins = Math.max(0, Math.round((endDate - startDate) / 60000));
  const h = Math.floor(durMins / 60);
  const m = durMins % 60;
  const durStr = h > 0 ? `${h}h ${m}m` : `${m}m`;
  return `<div style="display:flex;align-items:center;justify-content:space-between;gap:10px;padding:8px 10px;background:var(--surface2);border:1px solid var(--line);border-radius:8px">
    <div style="min-width:0">
      <div style="font-size:12px;font-weight:700;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(formatDate(s.started_at))}</div>
      <div style="font-size:10px;color:var(--muted);margin-top:2px">${durStr} &middot; ${s.goals}G ${s.assists}A ${s.saves}Sv ${s.shots}Sh</div>
    </div>
    <div style="font-size:12px;font-weight:800;font-variant-numeric:tabular-nums;white-space:nowrap">
      <span style="color:var(--green)">${s.wins}W</span>
      <span style="color:var(--muted)"> / </span>
      <span style="color:var(--red)">${s.losses}L</span>
    </div>
  </div>`;
}
