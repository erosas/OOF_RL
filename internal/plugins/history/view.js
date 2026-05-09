'use strict';

let allPlayers = [];
let _expandedMatchId = null;
let _historyRecentInstances = [];

async function loadHistory() {
  _historyRecentInstances.forEach(w => w.refresh());
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
    const badges = historyMatchBadges(m);
    const arena  = friendlyArena(m.Arena);

    return `
      <div class="match-card ${result}" data-id="${m.ID}">
        <div class="match-card-left">
          <div class="match-card-top">
            <span class="match-card-arena">${esc(arena)}</span>
            ${badges.map(b => `<span class="${esc(b.className)}"${b.style ? ` style="${esc(b.style)}"` : ''}>${esc(b.label)}</span>`).join('')}
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

function historyMatchBadges(m) {
  const badges = [];
  const size = matchType(m.player_count);
  const kind = historyPlaylistKind(m.PlaylistType);

  if (size) badges.push({ label: size, className: 'match-mode-badge' });
  if (kind) badges.push({ label: kind, className: 'match-mode-badge' });
  if ((m.bot_count || 0) > 0) {
    badges.push({
      label: 'PvE',
      className: 'match-mode-badge',
      style: 'background:rgba(34,197,94,0.12);color:#22c55e',
    });
  } else if ((m.player_count || 0) > 0) {
    badges.push({
      label: 'PvP',
      className: 'match-mode-badge',
      style: 'background:rgba(59,130,246,0.12);color:#3b82f6',
    });
  }
  if (m.Overtime) badges.push({ label: 'OT', className: 'match-mode-badge match-mode-ot' });
  if (m.Forfeit) {
    badges.push({
      label: 'FF',
      className: 'match-mode-badge',
      style: 'background:rgba(234,179,8,0.12);color:#ca8a04',
    });
  }
  if (m.Incomplete) {
    badges.push({
      label: 'Incomplete',
      className: 'match-mode-badge',
      style: 'background:rgba(156,163,175,0.12);color:#9ca3af',
    });
  }
  return badges;
}

function historyPlaylistKind(playlistID) {
  if (playlistID == null) return '';
  if ([10, 11, 13, 14].includes(playlistID)) return 'Ranked';
  if ([1, 2, 3, 34].includes(playlistID)) return 'Casual';
  if (playlistID === 6) return 'Private';
  if (playlistID === 22) return 'Tournament';
  return friendlyPlaylist(playlistID);
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

function historyRecentWidget(container) {
  let _expandedId = null;

  async function refresh() {
    try {
      const matches = await fetch('/api/matches').then(r => r.json());
      const valid = (matches || []).filter(m => m.Arena && m.Arena !== '-').slice(0, 10);
      if (!valid.length) {
        container.innerHTML = '<div style="text-align:center;color:var(--muted);padding:24px 8px;font-size:13px">No matches yet.</div>';
        return;
      }
      render(valid);
    } catch(_) {
      container.innerHTML = '<div style="text-align:center;color:var(--muted);padding:24px 8px;font-size:13px">Failed to load matches.</div>';
    }
  }

  function render(matches) {
    const listEl = document.createElement('div');
    for (const m of matches) {
      const blue   = m.team0_goals ?? 0;
      const orange = m.team1_goals ?? 0;
      const result = m.WinnerTeamNum === 0 ? 'blue-win' : m.WinnerTeamNum === 1 ? 'orange-win' : '';
      const badges = historyMatchBadges(m);
      const card = document.createElement('div');
      card.className = `match-card ${result}`;
      card.dataset.id = m.ID;
      card.innerHTML = `
        <div class="match-card-left">
          <div class="match-card-top">
            <span class="match-card-arena">${esc(friendlyArena(m.Arena))}</span>
            ${badges.map(b => `<span class="${esc(b.className)}"${b.style ? ` style="${esc(b.style)}"` : ''}>${esc(b.label)}</span>`).join('')}
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
        </div>`;
      card.addEventListener('click', () => toggleInline(card, m.ID));
      listEl.appendChild(card);
    }
    container.innerHTML = '';
    container.appendChild(listEl);
  }

  async function toggleInline(card, matchId) {
    const existing = card.nextElementSibling;
    if (existing?.classList.contains('match-inline-panel')) {
      existing.remove();
      card.classList.remove('expanded');
      _expandedId = null;
      return;
    }
    container.querySelectorAll('.match-inline-panel').forEach(el => el.remove());
    container.querySelectorAll('.match-card.expanded').forEach(el => el.classList.remove('expanded'));
    _expandedId = matchId;
    card.classList.add('expanded');
    const panel = document.createElement('div');
    panel.className = 'match-inline-panel';
    panel.innerHTML = '<div style="padding:16px;color:var(--muted);font-size:13px">Loading…</div>';
    card.insertAdjacentElement('afterend', panel);
    try {
      const data = await fetch(`/api/matches/${matchId}`).then(r => r.json());
      window.renderMatchDetailPanel(data, panel, _expandedId, matchId);
    } catch(_) {
      panel.innerHTML = '<div style="padding:16px;color:var(--muted);font-size:13px">Failed to load.</div>';
    }
  }

  function destroy() {
    const i = _historyRecentInstances.indexOf(entry);
    if (i >= 0) _historyRecentInstances.splice(i, 1);
  }

  const entry = { refresh };
  _historyRecentInstances.push(entry);
  refresh();
  return { refresh, destroy };
}

window.pluginInit_history = function() {
  window.registerWidget?.({
    id: 'history-recent', pluginId: 'history', title: 'Recent Matches',
    defaultW: 6, defaultH: 10, minW: 4, minH: 6,
    factory: historyRecentWidget,
  });
  const sel = document.getElementById('history-player-filter');
  if (sel) sel.addEventListener('change', e => fetchMatches(e.target.value));
};
