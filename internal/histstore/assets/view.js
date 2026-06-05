'use strict';

let allPlayers = [];
let _expandedMatchId = null;
let _selectedHistoryMatchId = null;
let _historyMatches = [];
let _historyRecentInstances = [];
let _historySummaryInstances = [];
let _historyPlayerID = localStorage.getItem('oof_session_player') || '';
let _historyFilterInitialized = false;

async function loadHistory() {
  _historyRecentInstances.forEach(w => w.refresh());
  _historySummaryInstances.forEach(w => w.refresh());
  allPlayers = await fetch('/api/players').then(r => r.json()) || [];
  const sel = document.getElementById('history-player-filter');
  const cur = _historyFilterInitialized ? _historyPlayerID : (sel.value || _historyPlayerID);
  sel.innerHTML = '<option value="">All Players</option>' +
    allPlayers.map(p => `<option value="${esc(p.PrimaryID)}">${esc(p.Name)}</option>`).join('');
  sel.value = [...sel.options].some(opt => opt.value === cur) ? cur : '';
  _historyPlayerID = sel.value;
  _historyFilterInitialized = true;
  await fetchMatches(_historyPlayerID);
}

async function fetchMatches(playerID) {
  _expandedMatchId = null;
  _historyDetailReRender = null;
  const url = playerID ? `/api/matches?player=${encodeURIComponent(playerID)}` : '/api/matches';
  const matches = await fetch(url).then(r => r.json()) || [];
  _historyMatches = matches.filter(m => m.Arena && m.Arena !== '-');

  const count = document.getElementById('history-match-count');
  if (count) count.textContent = `${_historyMatches.length} match${_historyMatches.length === 1 ? '' : 'es'}`;
  const resultHeading = document.getElementById('history-result-heading');
  if (resultHeading) resultHeading.textContent = playerID ? 'W/L' : 'Winner';

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
      <span class="history-result-token ${result.tokenClass}" title="${esc(result.label)}">${esc(result.token)}</span>
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
  detail.innerHTML = '<div class="ui-widget-loading history-detail-loading">Loading match detail...</div>';

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
    <div class="history-empty-icon">i</div>
    <div>
      <div class="history-empty-title">${esc(title)}</div>
      <div class="history-empty-copy">${esc(copy)}</div>
    </div>`;
}

function renderHistorySelectedSummary(m) {
  const result = historyMatchResult(m);
  const mode = [historyPlaylistKind(m.PlaylistType), matchType(m.player_count)].filter(Boolean).join(' ');
  const duration = historyMatchDuration(m);
  const meta = [
    formatDate(m.StartedAt),
    duration,
  ].filter(Boolean);
  const badges = historyMatchBadges(m);
  const fullGuid = m.MatchGUID ? String(m.MatchGUID).toUpperCase() : '';
  const matchDetails = fullGuid ? `
      <details class="history-hero-details">
        <summary>Match details</summary>
        <div class="history-hero-details-body">
          <span>Game ID</span>
          <code>${esc(fullGuid)}</code>
        </div>
      </details>` : '<div class="history-hero-details-placeholder"></div>';

  return `
    <div class="history-match-hero ${result.matchClass}">
      <div class="history-hero-main">
        <div class="history-hero-mode">${esc(mode || 'Saved Match')}</div>
        <h3>${esc(friendlyArena(m.Arena))}</h3>
        <div class="history-hero-meta">${meta.map(item => `<span class="history-hero-meta-item">${esc(item)}</span>`).join('')}</div>
      </div>
      <div class="history-hero-score">
        <div class="history-hero-scoreline">
          <span class="history-score-team history-score-team-blue">Blue</span>
          <span class="blue">${m.team0_goals ?? 0}</span>
          <span class="sep">-</span>
          <span class="orange">${m.team1_goals ?? 0}</span>
          <span class="history-score-team history-score-team-orange">Orange</span>
        </div>
        <span class="history-hero-result ${result.tokenClass}">${esc(result.label)}</span>
      </div>
      ${matchDetails}
    </div>
    <div class="history-detail-strip">
      ${badges.map(b => `<span class="${esc(b.className)}"${b.style ? ` style="${esc(b.style)}"` : ''}>${esc(b.label)}</span>`).join('')}
    </div>`;
}

function historyMatchResult(m) {
  if (m.Incomplete) {
    return { token: 'INC', label: 'Incomplete', matchClass: 'incomplete', tokenClass: 'neutral' };
  }
  const winnerClass = m.WinnerTeamNum === 0 ? 'blue-win' : m.WinnerTeamNum === 1 ? 'orange-win' : '';
  if (m.player_team === 0 || m.player_team === 1) {
    const won = m.WinnerTeamNum === m.player_team;
    return {
      token: won ? 'W' : 'L',
      label: won ? 'Win' : 'Loss',
      matchClass: `${winnerClass} ${won ? 'player-win' : 'player-loss'}`.trim(),
      tokenClass: won ? 'player-win' : 'player-loss',
    };
  }
  if (m.WinnerTeamNum === 0) {
    return { token: 'BLUE', label: m.Forfeit ? 'Blue FF win' : 'Blue win', matchClass: 'blue-win', tokenClass: 'blue' };
  }
  if (m.WinnerTeamNum === 1) {
    return { token: 'ORNG', label: m.Forfeit ? 'Orange FF win' : 'Orange win', matchClass: 'orange-win', tokenClass: 'orange' };
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
    : '<div class="history-team-empty">No players recorded.</div>';

  return `
    <div class="history-team-panel ${teamClass}">
      <div class="history-team-panel-header">
        <span>${esc(teamName)}</span>
        <span class="history-team-count">${players.length} player${players.length === 1 ? '' : 's'}</span>
      </div>
      <div class="history-team-roster">${rows}</div>
    </div>`;
}

function historyTeamPlayerRow(p) {
  const rankHTML = isBot(p.PrimaryId) ? '' : trackerRankHTML(p.PrimaryId);
  const rankDetails = rankHTML ? `
        <details class="history-rank-details">
          <summary>Ranks</summary>
          <div class="history-rank-panel">${rankHTML}</div>
        </details>` : '';

  return `
    <div class="history-team-player-row">
      <div class="history-player-cell">
        <div class="history-player-name"><span class="history-player-name-text">${esc(p.Name)}</span>${playerPlatformBadge(p.PrimaryId)}</div>
        ${rankDetails}
      </div>
      <div class="history-player-stat-grid">
        ${historyPlayerStat('Score', p.Score ?? 0, 'history-score-cell')}
        ${historyPlayerStat('G', p.Goals ?? 0)}
        ${historyPlayerStat('A', p.Assists ?? 0)}
        ${historyPlayerStat('Sv', p.Saves ?? 0)}
        ${historyPlayerStat('Sh', p.Shots ?? 0)}
        ${historyPlayerStat('Dm', p.Demos ?? 0)}
        ${historyPlayerStat('Tch', p.Touches ?? 0)}
      </div>
    </div>`;
}

function historyPlayerStat(label, value, className = '') {
  return `<span class="history-stat-cell ${esc(className)}"><span>${esc(label)}</span><strong>${value}</strong></span>`;
}

function historyEventFeed(events, goals, nameTeam, matchStart) {
  if (events.length) {
    return `<div class="history-event-feed">${events.map(e => historyEventRow(e, goals, nameTeam, matchStart)).join('')}</div>`;
  }
  if (goals.length) {
    return `<div class="history-event-feed">${goals.map(g => historyGoalRow(g, nameTeam)).join('')}</div>`;
  }
  return '<div class="ui-widget-empty history-events-empty">No match events recorded.</div>';
}

function historyEventRow(e, goals, nameTeam, matchStart) {
  const teamClass = e.team_num === 0 ? 'blue' : e.team_num === 1 ? 'orange' : 'neutral';
  const type = historyEventLabel(e.event_type);
  const actor = historyColoredName(e.player_name, nameTeam);
  const target = e.target_name ? ` -> ${historyColoredName(e.target_name, nameTeam)}` : '';
  const companions = historyEventCompanions(e, goals, nameTeam, matchStart);
  const note = historyEventNote(e.event_type);
  return `
    <div class="history-event-row ${teamClass}">
      ${historyEventTimeCell(e.occurred_at, e)}
      <span class="history-event-type">${esc(type)}</span>
      <span class="history-event-player">${actor}${target}</span>
      <span class="history-event-note">
        ${note ? `<span>${esc(note)}</span>` : ''}
        ${companions}
      </span>
    </div>`;
}

function historyGoalRow(g, nameTeam) {
  const scorer = historyColoredName(g.ScorerName, nameTeam);
  const assist = g.AssisterName ? historyColoredName(g.AssisterName, nameTeam) : 'Unassisted';
  const speed = g.GoalSpeed != null ? `${g.GoalSpeed.toFixed(1)} kph` : '';
  return `
    <div class="history-event-row goal">
      ${historyEventTimeCell(g.scored_at ?? g.ScoredAt, g)}
      <span class="history-event-type">Goal</span>
      <span class="history-event-player">${scorer}</span>
      <span class="history-event-note">
        <span class="history-event-companion"><span>Assist</span>${assist}</span>
        ${speed ? `<span class="history-event-companion"><span>Speed</span>${esc(speed)}</span>` : ''}
      </span>
    </div>`;
}

function historyEventCompanions(e, goals, nameTeam, matchStart) {
  if (!['Goal', 'OwnGoal'].includes(e.event_type) || !goals.length) return '';
  const goal = historyFindGoalForEvent(e, goals, matchStart);
  if (!goal) return '';
  const parts = [];
  if (goal.AssisterName) {
    parts.push(`<span class="history-event-companion"><span>Assist</span>${historyColoredName(goal.AssisterName, nameTeam)}</span>`);
  }
  if (goal.GoalSpeed != null) {
    parts.push(`<span class="history-event-companion"><span>Speed</span>${esc(goal.GoalSpeed.toFixed(1))} kph</span>`);
  }
  return parts.join('');
}

function historyFindGoalForEvent(e, goals, matchStart) {
  const eventClock = historyGameTimeSeconds(e);
  const eventSecs = matchStart && e.occurred_at
    ? Math.max(0, (new Date(e.occurred_at).getTime() - matchStart) / 1000)
    : null;
  const sameScorer = goals.filter(g => !e.player_name || g.ScorerName === e.player_name);
  if (eventClock != null) {
    return sameScorer.find(g => {
      const goalClock = historyGameTimeSeconds(g);
      return goalClock != null && Math.abs(goalClock - eventClock) <= 1;
    })
      || goals.find(g => {
        const goalClock = historyGameTimeSeconds(g);
        return goalClock != null && Math.abs(goalClock - eventClock) <= 1;
      })
      || sameScorer[0]
      || null;
  }
  if (eventSecs != null) {
    return sameScorer.find(g => Math.abs((g.GoalTime ?? -9999) - eventSecs) <= 3)
      || goals.find(g => Math.abs((g.GoalTime ?? -9999) - eventSecs) <= 3)
      || sameScorer[0]
      || null;
  }
  return sameScorer[0] || null;
}

function historyEventTimeCell(pcTime, row) {
  return `<span class="history-event-time history-event-time-pair">
    <span>${esc(historyGameClockLabel(row))}</span>
    <small>${esc(historyPCClock(pcTime))}</small>
  </span>`;
}

function historyPCClock(value) {
  if (!value) return 'PC time unavailable';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'PC time unavailable';
  return `PC ${date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', second: '2-digit' })}`;
}

function historyGameClockLabel(row) {
  const secs = historyGameTimeSeconds(row);
  if (secs == null) return 'Game clock unavailable';
  return `Game ${historyFormatGameClock(secs, historyGameOvertime(row))}`;
}

function historyGameTimeSeconds(row) {
  const value = row?.game_time_seconds ?? row?.GameTimeSeconds;
  if (value == null || value === '') return null;
  const secs = Number(value);
  return Number.isFinite(secs) ? secs : null;
}

function historyGameOvertime(row) {
  return !!(row?.game_overtime ?? row?.GameOvertime);
}

function historyFormatGameClock(secs, overtime) {
  const rounded = Math.max(0, Math.round(Number(secs)));
  const m = Math.floor(rounded / 60);
  const s = String(rounded % 60).padStart(2, '0');
  return overtime ? `OT ${m}:${s}` : `${m}:${s}`;
}

function historyColoredName(name, nameTeam) {
  if (!name) return '<span class="history-name-muted">-</span>';
  const team = nameTeam.get(name);
  const cls = team === 'blue' ? 'blue' : team === 'orange' ? 'orange' : '';
  return cls ? `<span class="${cls}">${esc(name)}</span>` : esc(name);
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
        container.innerHTML = '<div class="ui-widget-empty history-widget-empty">No matches yet.</div>';
        return;
      }
      render(valid);
    } catch(_) {
      container.innerHTML = '<div class="ui-widget-error history-widget-empty">Failed to load matches.</div>';
    }
  }

  function render(matches) {
    const listEl = document.createElement('div');
    listEl.className = 'history-widget-recent-list';
    for (const m of matches) {
      const blue = m.team0_goals ?? 0;
      const orange = m.team1_goals ?? 0;
      const result = historyMatchResult(m);
      const badges = historyMatchBadges(m);
      const card = document.createElement('div');
      card.className = `match-card history-widget-card ${result.matchClass}`;
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

    container.innerHTML = `<div class="history-summary-widget">
      <div class="history-summary-grid">
        ${historySummaryPill('Matches', matches.length)}
        ${historySummaryPill('Blue W', blueWins, 'var(--rl-blue)')}
        ${historySummaryPill('Orange W', orangeWins, 'var(--rl-orange)')}
        ${historySummaryPill('OT', overtime)}
      </div>
      <div class="history-summary-latest">
        <div class="history-summary-latest-main">
          <div class="history-summary-label">Latest match</div>
          <div class="history-summary-arena">${esc(friendlyArena(latest.Arena))}</div>
          <div class="history-summary-date">${esc(formatDate(latest.StartedAt))}</div>
        </div>
        <div class="history-summary-score">
          <span class="blue">${latest.team0_goals ?? 0}</span>
          <span class="sep">-</span>
          <span class="orange">${latest.team1_goals ?? 0}</span>
        </div>
      </div>
      <div class="history-summary-footnote">${forfeits} forfeits recorded across saved matches.</div>
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
  return `<div class="history-summary-pill" style="--history-summary-color:${esc(color)}">
    <div class="history-summary-pill-value">${value}</div>
    <div class="history-summary-pill-label">${esc(label)}</div>
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
  if (sel) sel.addEventListener('change', e => {
    _historyPlayerID = e.target.value;
    fetchMatches(_historyPlayerID);
  });
};
