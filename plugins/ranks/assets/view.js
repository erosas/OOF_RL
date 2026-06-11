'use strict';

let _rankPlayers    = [];
let _ranksInstances = [];

window.pluginInit_ranks = async function() {
  _ranksReRender = renderRanks;

  window.registerView?.('ranks', { onShow: refreshRanks });

  window.registerWidget?.({
    id: 'ranks-display', pluginId: 'ranks', title: 'Player Ranks',
    defaultW: 4, defaultH: 8, minW: 3, minH: 4,
    factory: ranksDisplayWidget,
  });

  try {
    const players = await fetch('/api/ranks/players').then(r => r.json());
    _rankPlayers = players || [];
    renderRanks();
    if (_rankPlayers.length) {
      prefetchTrackerRanks(_rankPlayers.map(p => ({ PrimaryId: p.primary_id, Name: p.name })));
    }
  } catch(_) {}
};

function refreshRanks() {
  fetch('/api/ranks/players')
    .then(r => r.json())
    .then(players => {
      _rankPlayers = players || [];
      renderRanks();
      if (_rankPlayers.length) {
        prefetchTrackerRanks(_rankPlayers.map(p => ({ PrimaryId: p.primary_id, Name: p.name })));
      }
    })
    .catch(() => {});
}

window.handleRanksUpdate = function(data) {
  _rankPlayers = (data.Players || []).map(p => ({
    primary_id: p.PrimaryId,
    name:       p.Name,
    team_num:   p.TeamNum,
  }));
  prefetchTrackerRanks(data.Players || []);
  renderRanks();
  _ranksInstances.forEach(w => w.render());
};

window.handleRanksClear = function() {
  _rankPlayers = [];
  renderRanks();
  _ranksInstances.forEach(w => w.render());
};

function renderRanks() {
  const empty   = document.getElementById('ranks-empty');
  const content = document.getElementById('ranks-content');
  const teams   = document.getElementById('ranks-teams');
  if (!empty || !content || !teams) return;

  if (!_rankPlayers.length) {
    empty.classList.remove('hidden');
    content.classList.add('hidden');
    return;
  }
  empty.classList.add('hidden');
  content.classList.remove('hidden');

  const blue   = _rankPlayers.filter(p => p.team_num === 0);
  const orange = _rankPlayers.filter(p => p.team_num === 1);
  teams.innerHTML = renderTeams(blue, orange);
}

function ranksDisplayWidget(container) {
  function render() {
    if (!_rankPlayers.length) {
      container.innerHTML = `<div class="ranks-widget">
        <div class="ranks-widget-empty">Waiting for a match - ranks will appear here.</div>
      </div>`;
      return;
    }
    const blue   = _rankPlayers.filter(p => p.team_num === 0);
    const orange = _rankPlayers.filter(p => p.team_num === 1);
    container.innerHTML = `<div class="ranks-widget">${renderTeams(blue, orange, true)}</div>`;
  }

  function destroy() {
    const i = _ranksInstances.indexOf(entry);
    if (i >= 0) _ranksInstances.splice(i, 1);
  }

  const entry = { render };
  _ranksInstances.push(entry);
  render();
  return { refresh: render, destroy };
}

function renderTeams(blue, orange, compact = false) {
  return renderTeamPanel('Blue', 'blue', '4a9eff', blue, compact) +
    renderTeamPanel('Orange', 'orange', 'ff8c2a', orange, compact);
}

function renderTeamPanel(teamName, cls, hex, players, compact = false) {
  const rgba = hexToRgba(hex, 0.12) || 'transparent';
  const border = hexToRgba(hex, 0.34) || 'var(--oof-color-border)';
  const glow = hexToRgba(hex, 0.32) || 'transparent';
  const panelClass = compact ? 'ranks-team-panel ranks-team-panel-compact' : 'ranks-team-panel';

  const rows = players.map(p => {
    const pid        = p.primary_id;
    const playerName = p.name || 'Unknown player';
    const trnUrl     = !isBot(pid) ? trnProfileUrl(pid, playerName) : '';
    const nameEl = trnUrl
      ? `<a href="${esc(trnUrl)}" target="_blank" rel="noopener" class="ranks-player-name ranks-player-link">${esc(playerName)}</a>`
      : `<span class="ranks-player-name">${esc(playerName)}</span>`;

    const rankHtml = trackerRankHTML(pid);
    const rankState = rankHtml || (isBot(pid)
      ? '<span class="ranks-rank-note">Bot or local player</span>'
      : '<span class="ranks-rank-note">Fetching...</span>');

    return `<div class="ranks-player-row">
      <div class="ranks-player-main">
        ${nameEl}
        <span class="ranks-player-meta">${isBot(pid) ? 'Rank unavailable' : 'Tracker profile'}</span>
      </div>
      <div class="ranks-rank-surface">${rankState}</div>
    </div>`;
  }).join('');

  const emptyRow = `<div class="ranks-player-row ranks-player-row-empty">
    <div class="ranks-player-main">
      <span class="ranks-player-name">No players detected</span>
      <span class="ranks-player-meta">Waiting for ${esc(teamName.toLowerCase())} team data</span>
    </div>
  </div>`;

  return `<section class="${panelClass} ranks-team-panel-${cls}" style="--ranks-team-color:#${hex};--ranks-team-bg:${rgba};--ranks-team-border:${border};--ranks-team-glow:${glow}">
    <div class="ranks-team-header">
      <div>
        <span class="ranks-team-label">${esc(teamName)} team</span>
        <strong>${players.length}</strong>
      </div>
      <span class="ranks-team-chip">Ranks</span>
    </div>
    <div class="ranks-playlist-stack">${rows || emptyRow}</div>
  </section>`;
}
