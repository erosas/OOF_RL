'use strict';

let _sessionSince           = null;
let _sessionPlayerID        = '';
let _sessionElapsedTimer    = null;
let _sessionExpandedMatchId = null;

window.pluginInit_session = async function() {
  _sessionPlayerID = localStorage.getItem('oof_session_player') || '';

  // Fetch server-side session start time (resets on each app launch)
  try {
    const data = await fetch('/api/session/start').then(r => r.json());
    _sessionSince = new Date(data.since);
  } catch(_) {
    _sessionSince = new Date();
  }

  // Populate player dropdown
  try {
    const players = await fetch('/api/players').then(r => r.json());
    const sel = document.getElementById('session-player-select');
    for (const p of (players || [])) {
      const opt = document.createElement('option');
      opt.value = p.PrimaryID;
      opt.textContent = p.Name;
      if (p.PrimaryID === _sessionPlayerID) opt.selected = true;
      sel.appendChild(opt);
    }
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
  const noPlayer = document.getElementById('session-no-player');
  const panel    = document.getElementById('session-stats-panel');
  if (!_sessionPlayerID) {
    noPlayer?.classList.remove('hidden');
    panel?.classList.add('hidden');
    return;
  }
  noPlayer?.classList.add('hidden');
  panel?.classList.remove('hidden');

  try {
    const url  = `/api/session/stats?player=${encodeURIComponent(_sessionPlayerID)}`;
    const data = await fetch(url).then(r => r.json());
    renderSessionStats(data);
  } catch(_) {}
};

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
    listEl.innerHTML = '<p class="text-center text-gray-500 text-sm py-6">No matches in this session yet.</p>';
    return;
  }

  listEl.innerHTML = '';
  for (const m of [...matches].reverse()) {
    const finished  = m.winner_team_num >= 0;
    const won       = finished && m.player_team === m.winner_team_num;
    const lost      = finished && m.player_team !== m.winner_team_num;
    const resultCls = won ? 'text-green-400' : lost ? 'text-red-400' : 'text-gray-500';
    const resultTxt = won ? 'W' : lost ? 'L' : '—';

    const card = document.createElement('div');
    card.className = 'session-match-card bg-surface border border-line rounded-xl px-4 py-3 flex items-center gap-3 cursor-pointer hover:border-rl-blue/50 transition-colors';
    card.dataset.matchId = m.match_id;
    card.innerHTML = `
      <span class="text-2xl font-extrabold tabular-nums w-6 text-center shrink-0 ${resultCls}">${resultTxt}</span>
      <div class="flex-1 min-w-0">
        <div class="font-medium text-sm truncate">${esc(friendlyArena(m.arena))}</div>
        <div class="text-xs text-gray-500">${formatDate(m.started_at)}</div>
      </div>
      <div class="text-right text-sm tabular-nums shrink-0">
        <div class="text-gray-300">${m.goals}G ${m.assists}A ${m.saves}Sv</div>
        <div class="text-xs text-gray-500">Score ${m.score}</div>
      </div>
      <span class="text-gray-500 shrink-0 text-lg session-chevron transition-transform">›</span>
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
  panel.className = 'session-match-panel match-inline-panel';
  panel.innerHTML = '<div style="padding:16px;color:var(--muted);font-size:13px">Loading…</div>';
  card.insertAdjacentElement('afterend', panel);

  await loadSessionMatchDetail(matchId, panel);
}

async function loadSessionMatchDetail(matchId, panel) {
  try {
    const data    = await fetch(`/api/matches/${matchId}`).then(r => r.json());
    const players = data.players || [];
    const goals   = data.goals   || [];

    players.forEach(p => { if (!p.PrimaryId) p.PrimaryId = p.PrimaryID; });
    const realPlayers = players.filter(p => !isBot(p.PrimaryId));
    prefetchTrackerRanks(realPlayers);

    const blue   = realPlayers.filter(p => p.TeamNum === 0);
    const orange = realPlayers.filter(p => p.TeamNum === 1);

    const nameTeam = new Map();
    blue.forEach(p => nameTeam.set(p.Name, 'blue'));
    orange.forEach(p => nameTeam.set(p.Name, 'orange'));

    function goalNameEl(name) {
      if (!name) return '<span style="color:var(--muted)">—</span>';
      const t = nameTeam.get(name);
      const style = t === 'blue' ? 'color:var(--rl-blue)' : t === 'orange' ? 'color:var(--rl-orange)' : '';
      return style ? `<span style="${style}">${esc(name)}</span>` : esc(name);
    }

    const statsRows = (list, cls) => list.map(p => `
      <tr class="${cls}">
        <td>
          <div class="font-medium">${esc(p.Name)}</div>
          ${trackerRankHTML(p.PrimaryId)}
        </td>
        <td>${p.Goals}</td><td>${p.Assists}</td><td>${p.Saves}</td>
        <td>${p.Shots}</td><td>${p.Demos}</td>
        <td>${p.Touches ?? 0}</td>
        <td>${p.Score}</td>
      </tr>`).join('');

    const goalsHTML = goals.length
      ? goals.map(g => `
          <tr>
            <td>${goalNameEl(g.ScorerName)}</td>
            <td>${goalNameEl(g.AssisterName)}</td>
            <td>${g.GoalSpeed != null ? g.GoalSpeed.toFixed(1) : '—'}</td>
            <td>${formatDuration(g.GoalTime)}</td>
          </tr>`).join('')
      : '<tr><td colspan="4" style="color:var(--muted)">No goals recorded</td></tr>';

    if (_sessionExpandedMatchId !== matchId) return;

    panel.innerHTML = `
      <div class="match-detail-panel">
        <h3 class="detail-section-label">Player Stats</h3>
        <table class="detail-table">
          <thead><tr>
            <th>Player</th><th>G</th><th>A</th><th>Sv</th><th>Sh</th><th>Dm</th><th>Touches</th><th>Score</th>
          </tr></thead>
          <tbody>${statsRows(blue, 'blue-row')}${statsRows(orange, 'orange-row')}</tbody>
        </table>
        <h3 class="detail-section-label" style="margin-top:16px">Goals</h3>
        <table class="detail-table">
          <thead><tr><th>Scorer</th><th>Assist</th><th>Speed (kph)</th><th>Time</th></tr></thead>
          <tbody>${goalsHTML}</tbody>
        </table>
      </div>`;
  } catch(_) {
    panel.innerHTML = '<div style="padding:16px;color:var(--muted);font-size:13px">Failed to load match detail.</div>';
  }
}

async function loadSessionHistory() {
  const section = document.getElementById('session-history-section');
  const list    = document.getElementById('session-history-list');
  if (!section || !list) return;
  if (!_sessionPlayerID) {
    section.classList.add('hidden');
    return;
  }

  try {
    const sessions = await fetch(`/api/session/history?player=${encodeURIComponent(_sessionPlayerID)}`).then(r => r.json());
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
  card.className = 'bg-surface border border-line rounded-xl px-4 py-3 cursor-pointer hover:border-rl-blue/50 transition-colors';
  card.innerHTML = `
    <div class="flex items-center justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="text-sm font-medium">${formatDate(s.started_at)}</div>
        <div class="text-xs text-gray-500 mt-0.5">
          ${durStr} &nbsp;·&nbsp;
          <span class="text-green-400">${s.wins}W</span> <span class="text-red-400">${s.losses}L</span>
          &nbsp;·&nbsp; ${s.goals}G ${s.assists}A ${s.saves}Sv ${s.shots}Sh ${s.demos}Dm
        </div>
      </div>
      <span class="text-gray-500 shrink-0 text-lg sess-hist-chevron transition-transform">›</span>
    </div>
  `;

  const editPanel = document.createElement('div');
  editPanel.className = 'hidden bg-surface2 border border-line border-t-0 rounded-b-xl px-4 py-3 -mt-1';
  editPanel.innerHTML = `
    <div class="grid grid-cols-2 gap-3 mb-3">
      <div>
        <label class="text-xs text-gray-500 block mb-1">Start</label>
        <input type="datetime-local" class="sess-hist-start w-full bg-surface border border-line rounded px-2 py-1 text-xs text-gray-200 focus:outline-none focus:border-rl-blue" value="${toDatetimeLocal(startDate)}">
      </div>
      <div>
        <label class="text-xs text-gray-500 block mb-1">End</label>
        <input type="datetime-local" class="sess-hist-end w-full bg-surface border border-line rounded px-2 py-1 text-xs text-gray-200 focus:outline-none focus:border-rl-blue" value="${toDatetimeLocal(endDate)}">
      </div>
    </div>
    <div class="flex items-center gap-2">
      <button class="sess-hist-save text-xs bg-rl-blue text-white rounded px-3 py-1 hover:bg-rl-blue/80">Save</button>
      <button class="sess-hist-delete text-xs text-red-400 hover:text-red-300 border border-red-400/40 rounded px-3 py-1">Delete</button>
      <span class="sess-hist-msg text-xs ml-2 hidden"></span>
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
        msg.className = 'sess-hist-msg text-xs text-green-400 ml-2';
        msg.classList.remove('hidden');
        setTimeout(() => msg.classList.add('hidden'), 3000);
        loadSessionHistory();
      } else {
        const err = await res.json().catch(() => ({}));
        msg.textContent = err.error || 'Error saving.';
        msg.className = 'sess-hist-msg text-xs text-red-400 ml-2';
        msg.classList.remove('hidden');
      }
    } catch(_) {
      msg.textContent = 'Request failed.';
      msg.className = 'sess-hist-msg text-xs text-red-400 ml-2';
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