'use strict';

let allPlayers = [];
let _expandedMatchId = null;

async function loadHistory() {
  allPlayers = await fetch('/api/players').then(r => r.json()) || [];
  const sel = document.getElementById('history-player-filter');
  const cur = sel.value;
  sel.innerHTML = '<option value="">All Players</option>' +
    allPlayers.map(p => `<option value="${esc(p.PrimaryID)}">${esc(p.Name)}</option>`).join('');
  sel.value = cur;
  await fetchMatches(sel.value);
}

async function fetchMatches(playerID) {
  _expandedMatchId = null;
  _historyDetailReRender = null;
  const url = playerID ? `/api/matches?player=${encodeURIComponent(playerID)}` : '/api/matches';
  const matches = await fetch(url).then(r => r.json()) || [];
  const list = document.getElementById('matches-list');

  const validMatches = matches.filter(m => m.Arena && m.Arena !== '-');

  if (!validMatches.length) {
    list.innerHTML = '<p class="text-gray-600 py-5">No matches found.</p>';
    return;
  }

  list.innerHTML = validMatches.map(m => {
    const blue   = m.team0_goals ?? 0;
    const orange = m.team1_goals ?? 0;
    const result = m.WinnerTeamNum === 0 ? 'blue-win' : m.WinnerTeamNum === 1 ? 'orange-win' : '';
    const type   = friendlyPlaylist(m.PlaylistType) || matchType(m.player_count);
    const arena  = friendlyArena(m.Arena);

    return `
      <div class="match-card ${result}" data-id="${m.ID}">
        <div class="match-card-left">
          <div class="match-card-top">
            <span class="match-card-arena">${esc(arena)}</span>
            ${type ? `<span class="match-mode-badge">${esc(type)}</span>` : ''}
            ${m.Overtime ? '<span class="match-mode-badge match-mode-ot">OT</span>' : ''}
          </div>
          <span class="match-card-date">${formatDate(m.StartedAt)}</span>
        </div>
        <div class="match-card-right">
          <div class="match-card-scores">
            <span class="blue">${blue}</span>
            <span class="sep">—</span>
            <span class="orange">${orange}</span>
          </div>
          <span class="match-expand-chevron">›</span>
        </div>
      </div>`;
  }).join('');

  list.querySelectorAll('.match-card').forEach(card => {
    card.addEventListener('click', () => toggleMatchInline(card, parseInt(card.dataset.id)));
  });
}

async function toggleMatchInline(card, matchId) {
  const existing = card.nextElementSibling;
  if (existing && existing.classList.contains('match-inline-panel')) {
    existing.remove();
    card.classList.remove('expanded');
    _expandedMatchId = null;
    _historyDetailReRender = null;
    return;
  }

  document.querySelectorAll('.match-inline-panel').forEach(el => el.remove());
  document.querySelectorAll('.match-card.expanded').forEach(el => el.classList.remove('expanded'));
  _historyDetailReRender = null;

  card.classList.add('expanded');
  _expandedMatchId = matchId;

  const panel = document.createElement('div');
  panel.className = 'match-inline-panel';
  panel.innerHTML = '<div style="padding:16px;color:var(--muted);font-size:13px">Loading…</div>';
  card.insertAdjacentElement('afterend', panel);

  await loadMatchDetail(matchId, panel);
}

async function loadMatchDetail(matchID, panel) {
  const data    = await fetch(`/api/matches/${matchID}`).then(r => r.json());
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
    const style = t === 'blue' ? `color:var(--rl-blue)` : t === 'orange' ? `color:var(--rl-orange)` : '';
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

  const tbodyId = `match-stats-tbody-${matchID}`;

  const renderStatsTable = () => {
    const tbody = document.getElementById(tbodyId);
    if (!tbody) { _historyDetailReRender = null; return; }
    tbody.innerHTML = statsRows(blue, 'blue-row') + statsRows(orange, 'orange-row');
  };
  _historyDetailReRender = renderStatsTable;

  if (_expandedMatchId !== matchID) return;

  panel.innerHTML = `
    <div class="match-detail-panel">
      <h3 class="detail-section-label">Player Stats</h3>
      <table class="detail-table">
        <thead><tr>
          <th>Player</th><th>G</th><th>A</th><th>Sv</th><th>Sh</th><th>Dm</th><th>Touches</th><th>Score</th>
        </tr></thead>
        <tbody id="${tbodyId}">${statsRows(blue, 'blue-row')}${statsRows(orange, 'orange-row')}</tbody>
      </table>
      <h3 class="detail-section-label" style="margin-top:16px">Goals</h3>
      <table class="detail-table">
        <thead><tr><th>Scorer</th><th>Assist</th><th>Speed (kph)</th><th>Time</th></tr></thead>
        <tbody>${goalsHTML}</tbody>
      </table>
    </div>`;
}

window.pluginInit_history = function() {
  const sel = document.getElementById('history-player-filter');
  if (sel) sel.addEventListener('change', e => fetchMatches(e.target.value));
};