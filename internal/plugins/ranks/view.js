'use strict';

let _rankPlayers = [];

window.pluginInit_ranks = async function() {
  _ranksReRender = renderRanks;
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
};

window.handleRanksClear = function() {
  _rankPlayers = [];
  renderRanks();
};

function renderRanks() {
  const empty   = document.getElementById('ranks-empty');
  const content = document.getElementById('ranks-content');
  if (!empty || !content) return;

  if (!_rankPlayers.length) {
    empty.classList.remove('hidden');
    content.classList.add('hidden');
    return;
  }
  empty.classList.add('hidden');
  content.classList.remove('hidden');

  const blue   = _rankPlayers.filter(p => p.team_num === 0);
  const orange = _rankPlayers.filter(p => p.team_num === 1);
  document.getElementById('ranks-teams').innerHTML =
    renderTeamPanel('Blue',   '4a9eff', blue) +
    renderTeamPanel('Orange', 'ff8c2a', orange);
}

function renderTeamPanel(teamName, hex, players) {
  if (!players.length) return '';
  const rgba  = hexToRgba(hex, 0.12) || 'transparent';
  const color = `#${hex}`;

  const rows = players.map(p => {
    const pid    = p.primary_id;
    const trnUrl = !isBot(pid) ? trnProfileUrl(pid, p.name) : '';
    const nameEl = trnUrl
      ? `<a href="${esc(trnUrl)}" target="_blank" rel="noopener" class="font-semibold hover:underline">${esc(p.name)}</a>`
      : `<span class="font-semibold">${esc(p.name)}</span>`;

    const rankHtml = trackerRankHTML(pid);

    return `<div class="py-3 border-b border-line last:border-0">
      <div class="mb-1">${nameEl}</div>
      <div>${rankHtml || (isBot(pid) ? '' : '<span class="text-xs text-gray-500">Fetching…</span>')}</div>
    </div>`;
  }).join('');

  return `<div class="bg-surface border border-line rounded-xl overflow-hidden mb-4">
    <div class="px-4 py-2 text-sm font-bold" style="color:${color};background:${rgba}">${esc(teamName)}</div>
    <div class="px-4">${rows}</div>
  </div>`;
}
