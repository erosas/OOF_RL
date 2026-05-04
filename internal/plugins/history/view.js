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
            ${m.Overtime    ? '<span class="match-mode-badge match-mode-ot">OT</span>' : ''}
            ${m.Forfeit     ? '<span class="match-mode-badge" style="background:rgba(234,179,8,0.12);color:#ca8a04">FF</span>' : ''}
            ${m.Incomplete  ? '<span class="match-mode-badge" style="background:rgba(156,163,175,0.12);color:#9ca3af">Incomplete</span>' : ''}
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
  const data = await fetch(`/api/matches/${matchID}`).then(r => r.json());
  window.renderMatchDetailPanel(data, panel, _expandedMatchId, matchID);
}

window.pluginInit_history = function() {
  const sel = document.getElementById('history-player-filter');
  if (sel) sel.addEventListener('change', e => fetchMatches(e.target.value));
};