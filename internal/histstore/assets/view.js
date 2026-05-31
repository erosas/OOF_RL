'use strict';

let allPlayers = [];
let _expandedMatchId = null;
let _selectedHistoryMatchId = null;
let _historyMatches = [];
let _historyRecentInstances = [];
let _historySummaryInstances = [];

async function loadHistory() {
  _historyRecentInstances.forEach(w => w.refresh());
  _historySummaryInstances.forEach(w => w.refresh());
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
  _historyMatches = matches.filter(m => m.Arena && m.Arena !== '-');

  const count = document.getElementById('history-match-count');
  if (count) count.textContent = `${_historyMatches.length} match${_historyMatches.length === 1 ? '' : 'es'}`;

  renderHistoryList(_historyMatches);

  if (!_historyMatches.length) {
    renderHistoryEmpty('No matches found', 'Saved matches will appear here after Rocket League games are recorded.');
    return;
  }

  const selectedStillVisible = _historyMatches.some(m => m.ID === _selectedHistoryMatchId);
  await selectHistoryMatch(selectedStillVisible ? _selectedHistoryMatchId : _historyMatches[0].ID);
}

function renderHistoryList(matches) {
  const list = document.getElementById('matches-list');
  if (!list) return;

  if (!matches.length) {
    list.innerHTML = '<div class="history-list-empty">No matches found.</div>';
    return;
  }

  list.innerHTML = matches.map(historyMatchCardHTML).join('');
  list.querySelectorAll('.history-match-card').forEach(card => {
    card.addEventListener('click', () => selectHistoryMatch(parseInt(card.dataset.id, 10)));
  });
}

function historyMatchCardHTML(m) {
  const blue = m.team0_goals ?? 0;
  const orange = m.team1_goals ?? 0;
  const result = historyMatchResult(m);
  const badges = historyMatchBadges(m);
  const arena = friendlyArena(m.Arena);
  const duration = historyMatchDuration(m);

  return `
    <button type="button" class="match-card history-match-card ${result.matchClass}" data-id="${m.ID}">
      <span class="history-result-token ${result.tokenClass}">${esc(result.token)}</span>
      <span class="history-match-card-main">
        <span class="match-card-top">
          ${badges.map(b => `<span class="${esc(b.className)}"${b.style ? ` style="${esc(b.style)}"` : ''}>${esc(b.label)}</span>`).join('')}
        </span>
        <span class="match-card-arena">${esc(arena)}</span>
        <span class="match-card-date">${formatDate(m.StartedAt)}</span>
      </span>
      <span class="match-card-right">
        <span class="match-card-scores">
          <span class="blue">${blue}</span>
          <span class="sep">-</span>
          <span class="orange">${orange}</span>
        </span>
        ${duration ? `<span class="history-card-duration">${esc(duration)}</span>` : ''}
      </span>
    </button>`;
}

async function selectHistoryMatch(matchId) {
  const id = Number(matchId);
  const match = _historyMatches.find(m => m.ID === id);
  if (!match) {
    renderHistoryEmpty('Match unavailable', 'This match is no longer in the current filtered list.');
    return;
  }

  _selectedHistoryMatchId = id;
  _expandedMatchId = id;
  _historyDetailReRender = null;

  document.querySelectorAll('#matches-list .history-match-card').forEach(card => {
    const selected = Number(card.dataset.id) === id;
    card.classList.toggle('selected', selected);
    card.setAttribute('aria-current', selected ? 'true' : 'false');
  });

  const empty = document.getElementById('history-detail-empty');
  const wrapper = document.getElementById('history-detail-content-wrapper');
  const summary = document.getElementById('history-selected-summary');
  const detail = document.getElementById('history-selected-detail');
  if (!empty || !wrapper || !summary || !detail) return;

  empty.classList.add('hidden');
  wrapper.classList.remove('hidden');
  summary.innerHTML = renderHistorySelectedSummary(match);
  detail.innerHTML = '<div class="ui-widget-loading">Loading match detail...</div>';

  await loadSelectedMatchDetail(id, detail);
}

async function loadSelectedMatchDetail(matchID, panel) {
  try {
    const data = await fetch(`/api/matches/${matchID}`).then(r => r.json());
    renderHistoryMatchDetailPanel(data, panel, _expandedMatchId, matchID);
  } catch(_) {
    if (_expandedMatchId === matchID) {
      panel.innerHTML = '<div class="ui-widget-error">Failed to load match detail.</div>';
    }
  }
}

function renderHistoryEmpty(title, copy) {
  _selectedHistoryMatchId = null;
  const empty = document.getElementById('history-detail-empty');
  const wrapper = document.getElementById('history-detail-content-wrapper');
  if (wrapper) wrapper.classList.add('hidden');
  if (!empty) return;
  empty.classList.remove('hidden');
  empty.innerHTML = `
    <div class="history-empty-icon">-</div>
    <div>
      <div class="history-empty-title">${esc(title)}</div>
      <div class="history-empty-copy">${esc(copy)}</div>
    </div>`;
}

function renderHistorySelectedSummary(m) {
  const result = historyMatchResult(m);
  const mode = [historyPlaylistKind(m.PlaylistType), matchType(m.player_count)].filter(Boolean).join(' ');
  const duration = historyMatchDuration(m);
  const guid = historyShortGuid(m.MatchGUID);
  const meta = [
    formatDate(m.StartedAt),
    duration,
    guid ? `Game ID: ${guid}` : '',
  ].filter(Boolean);
  const badges = historyMatchBadges(m);

  return `
    <div class="history-match-hero ${result.matchClass}">
      <div class="history-hero-main">
        <div class="history-hero-mode">${esc(mode || 'Saved Match')}</div>
        <h3>${esc(friendlyArena(m.Arena))}</h3>
        <div class="history-hero-meta">${meta.map(item => `<span>${esc(item)}</span>`).join('')}</div>
      </div>
      <div class="history-hero-score">
        <div>
          <span class="blue">${m.team0_goals ?? 0}</span>
          <span class="sep">-</span>
          <span class="orange">${m.team1_goals ?? 0}</span>
        </div>
        <span class="history-hero-result ${result.tokenClass}">${esc(result.label)}</span>
      </div>
    </div>
    <div class="history-detail-strip">
      ${badges.map(b => `<span class="${esc(b.className)}"${b.style ? ` style="${esc(b.style)}"` : ''}>${esc(b.label)}</span>`).join('')}
    </div>`;
}

function historyMatchResult(m) {
  if (m.Incomplete) {
    return { token: 'INC', label: 'Incomplete', matchClass: 'incomplete', tokenClass: 'neutral' };
  }
  if (m.WinnerTeamNum === 0) {
    return { token: 'B', label: m.Forfeit ? 'Blue FF win' : 'Blue win', matchClass: 'blue-win', tokenClass: 'blue' };
  }
  if (m.WinnerTeamNum === 1) {
    return { token: 'O', label: m.Forfeit ? 'Orange FF win' : 'Orange win', matchClass: 'orange-win', tokenClass: 'orange' };
  }
  return { token: '-', label: 'No result', matchClass: '', tokenClass: 'neutral' };
}

function historyMatchDuration(m) {
  if (!m.StartedAt || !m.EndedAt) return '';
  const start = Date.parse(m.StartedAt);
  const end = Date.parse(m.EndedAt);
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return '';
  const secs = Math.round((end - start) / 1000);
  const minutes = Math.floor(secs / 60);
  const seconds = String(secs % 60).padStart(2, '0');
  return `${minutes}m ${seconds}s`;
}

function historyShortGuid(guid) {
  if (!guid) return '';
  const s = String(guid);
  return s.length > 8 ? s.slice(0, 8).toUpperCase() : s.toUpperCase();
}

function renderHistoryMatchDetailPanel(data, panel, activeMatchId, matchID) {
  if (activeMatchId !== matchID) return;

  const players = data.players || [];
  const goals = data.goals || [];
  const events = data.events || [];
  const match = data.match || {};
  const matchStart = match.StartedAt ? new Date(match.StartedAt).getTime() : null;

  players.forEach(p => { if (!p.PrimaryId) p.PrimaryId = p.PrimaryID; });
  prefetchTrackerRanks(players.filter(p => !isBot(p.PrimaryId)));

  const blue = players.filter(p => p.TeamNum === 0);
  const orange = players.filter(p => p.TeamNum === 1);

  const nameTeam = new Map();
  blue.forEach(p => nameTeam.set(p.Name, 'blue'));
  orange.forEach(p => nameTeam.set(p.Name, 'orange'));

  panel.innerHTML = `
    <div class="history-match-detail">
      <section class="history-detail-section">
        <div class="history-detail-section-title">Team Scoreboard</div>
        <div class="history-team-scoreboard">
          ${historyTeamPanel('Blue Team', 'blue', blue)}
          ${historyTeamPanel('Orange Team', 'orange', orange)}
        </div>
      </section>
      <section class="history-detail-section">
        <div class="history-detail-section-title">Event Feed</div>
        ${historyEventFeed(events, goals, nameTeam, matchStart)}
      </section>
    </div>`;

  window._historyDetailReRender = () => {
    if (_expandedMatchId === matchID) renderHistoryMatchDetailPanel(data, panel, matchID, matchID);
  };
}

function historyTeamPanel(teamName, teamClass, players) {
  const rows = players.length
    ? players.map(p => historyTeamPlayerRow(p)).join('')
    : '<tr><td colspan="8" class="history-team-empty">No players recorded.</td></tr>';

  return `
    <div class="history-team-panel ${teamClass}">
      <div class="history-team-panel-header">
        <span>${esc(teamName)}</span>
      </div>
      <div class="history-team-table-wrap">
        <table class="history-team-table">
          <thead>
            <tr>
              <th>Player</th><th>Score</th><th>G</th><th>A</th><th>Sv</th><th>Sh</th><th>Dm</th><th>Tch</th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>`;
}

function historyTeamPlayerRow(p) {
  return `
    <tr>
      <td class="history-player-cell">
        <div class="history-player-name">${esc(p.Name)}${isBot(p.PrimaryId) ? ' <span class="player-platform-badge">BOT</span>' : ''}</div>
        ${isBot(p.PrimaryId) ? '' : trackerRankHTML(p.PrimaryId)}
      </td>
      <td>${p.Score ?? 0}</td>
      <td>${p.Goals ?? 0}</td>
      <td>${p.Assists ?? 0}</td>
      <td>${p.Saves ?? 0}</td>
      <td>${p.Shots ?? 0}</td>
      <td>${p.Demos ?? 0}</td>
      <td>${p.Touches ?? 0}</td>
    </tr>`;
}

function historyEventFeed(events, goals, nameTeam, matchStart) {
  if (events.length) {
    return `<div class="history-event-feed">${events.map(e => historyEventRow(e, nameTeam, matchStart)).join('')}</div>`;
  }
  if (goals.length) {
    return `<div class="history-event-feed">${goals.map(g => historyGoalRow(g, nameTeam)).join('')}</div>`;
  }
  return '<div class="ui-widget-empty">No match events recorded.</div>';
}

function historyEventRow(e, nameTeam, matchStart) {
  const teamClass = e.team_num === 0 ? 'blue' : e.team_num === 1 ? 'orange' : 'neutral';
  const type = historyEventLabel(e.event_type);
  const actor = historyColoredName(e.player_name, nameTeam);
  const target = e.target_name ? ` -> ${historyColoredName(e.target_name, nameTeam)}` : '';
  return `
    <div class="history-event-row ${teamClass}">
      <span class="history-event-time">${esc(historyRelTime(e.occurred_at, matchStart))}</span>
      <span class="history-event-type">${esc(type)}</span>
      <span class="history-event-player">${actor}${target}</span>
      <span class="history-event-note">${esc(historyEventNote(e.event_type))}</span>
    </div>`;
}

function historyGoalRow(g, nameTeam) {
  const scorer = historyColoredName(g.ScorerName, nameTeam);
  const assist = g.AssisterName ? `Assist ${historyColoredName(g.AssisterName, nameTeam)}` : 'Unassisted';
  const speed = g.GoalSpeed != null ? `${g.GoalSpeed.toFixed(1)} kph` : '';
  return `
    <div class="history-event-row goal">
      <span class="history-event-time">${esc(formatDuration(g.GoalTime))}</span>
      <span class="history-event-type">Goal</span>
      <span class="history-event-player">${scorer}</span>
      <span class="history-event-note">${assist}${speed ? ` - ${esc(speed)}` : ''}</span>
    </div>`;
}

function historyColoredName(name, nameTeam) {
  if (!name) return '<span class="history-name-muted">-</span>';
  const team = nameTeam.get(name);
  const cls = team === 'blue' ? 'blue' : team === 'orange' ? 'orange' : '';
  return cls ? `<span class="${cls}">${esc(name)}</span>` : esc(name);
}

function historyRelTime(occurredAt, matchStart) {
  if (!matchStart || !occurredAt) return '';
  const secs = Math.max(0, Math.round((new Date(occurredAt).getTime() - matchStart) / 1000));
  const m = Math.floor(secs / 60);
  const s = String(secs % 60).padStart(2, '0');
  return `+${m}:${s}`;
}

function historyEventLabel(type) {
  return ({
    Goal: 'Goal',
    OwnGoal: 'Own Goal',
    Save: 'Save',
    EpicSave: 'Epic Save',
    Assist: 'Assist',
    Demolish: 'Demo',
    Shot: 'Shot',
  })[type] || type || 'Event';
}

function historyEventNote(type) {
  return ({
    Goal: 'Scored',
    OwnGoal: 'Own goal',
    Save: 'Saved shot',
    EpicSave: 'Epic save',
    Assist: 'Assisted',
    Demolish: 'Demolition',
    Shot: 'Shot on target',
  })[type] || '';
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
      const blue = m.team0_goals ?? 0;
      const orange = m.team1_goals ?? 0;
      const result = historyMatchResult(m);
      const badges = historyMatchBadges(m);
      const card = document.createElement('div');
      card.className = `match-card ${result.matchClass}`;
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
            <span class="sep">-</span>
            <span class="orange">${orange}</span>
          </div>
          <span class="match-expand-chevron">></span>
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
    panel.innerHTML = '<div style="padding:16px;color:var(--muted);font-size:13px">Loading...</div>';
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
  return { refresh, destroy };
}

function historySummaryWidget(container) {
  async function refresh() {
    try {
      const matches = await fetch('/api/matches').then(r => r.json());
      const valid = (matches || []).filter(m => m.Arena && m.Arena !== '-');
      render(valid);
    } catch(_) {
      container.innerHTML = '<div class="ui-widget-error">Failed to load history summary.</div>';
    }
  }

  function render(matches) {
    if (!matches.length) {
      container.innerHTML = '<div class="ui-widget-empty">No match history yet.</div>';
      return;
    }

    const complete = matches.filter(m => !m.Incomplete);
    const blueWins = complete.filter(m => m.WinnerTeamNum === 0).length;
    const orangeWins = complete.filter(m => m.WinnerTeamNum === 1).length;
    const overtime = matches.filter(m => m.Overtime).length;
    const forfeits = matches.filter(m => m.Forfeit).length;
    const latest = matches[0];

    container.innerHTML = `<div style="display:flex;flex-direction:column;gap:10px">
      <div style="display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:8px">
        ${historySummaryPill('Matches', matches.length)}
        ${historySummaryPill('Blue W', blueWins, 'var(--rl-blue)')}
        ${historySummaryPill('Orange W', orangeWins, 'var(--rl-orange)')}
        ${historySummaryPill('OT', overtime)}
      </div>
      <div style="display:flex;align-items:center;justify-content:space-between;gap:10px;padding:9px 10px;background:var(--surface2);border:1px solid var(--line);border-radius:8px">
        <div style="min-width:0">
          <div style="font-size:10px;color:var(--muted);text-transform:uppercase;letter-spacing:.08em">Latest match</div>
          <div style="font-size:12px;font-weight:700;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(friendlyArena(latest.Arena))}</div>
          <div style="font-size:10px;color:var(--muted);margin-top:2px">${esc(formatDate(latest.StartedAt))}</div>
        </div>
        <div style="text-align:right;font-size:18px;font-weight:800;font-variant-numeric:tabular-nums;white-space:nowrap">
          <span style="color:var(--rl-blue)">${latest.team0_goals ?? 0}</span>
          <span style="color:var(--muted)">-</span>
          <span style="color:var(--rl-orange)">${latest.team1_goals ?? 0}</span>
        </div>
      </div>
      <div style="font-size:10px;color:var(--muted)">${forfeits} forfeits recorded across saved matches.</div>
    </div>`;
  }

  function destroy() {
    const i = _historySummaryInstances.indexOf(entry);
    if (i >= 0) _historySummaryInstances.splice(i, 1);
  }

  const entry = { refresh };
  _historySummaryInstances.push(entry);
  return { refresh, destroy };
}

function historySummaryPill(label, value, color = 'var(--text)') {
  return `<div style="background:var(--surface2);border:1px solid var(--line);border-radius:8px;padding:9px 6px;text-align:center;min-width:0">
    <div style="font-size:20px;font-weight:800;font-variant-numeric:tabular-nums;color:${color}">${value}</div>
    <div style="font-size:10px;color:var(--muted);text-transform:uppercase;letter-spacing:.06em;margin-top:2px">${esc(label)}</div>
  </div>`;
}

window.pluginInit_history = function() {
  window.registerWidget?.({
    id: 'history-recent', pluginId: 'history', title: 'Recent Matches',
    defaultW: 6, defaultH: 10, minW: 4, minH: 6,
    factory: historyRecentWidget,
  });
  window.registerWidget?.({
    id: 'history-summary', pluginId: 'history', title: 'History Summary',
    defaultW: 4, defaultH: 5, minW: 3, minH: 4,
    factory: historySummaryWidget,
  });
  const sel = document.getElementById('history-player-filter');
  if (sel) sel.addEventListener('change', e => fetchMatches(e.target.value));
};
