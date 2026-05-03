'use strict';

function formatClock(secs) {
  if (secs == null) return '—';
  const m = Math.floor(secs / 60);
  const s = String(secs % 60).padStart(2, '0');
  return `${m}:${s}`;
}

function handleUpdateState(data) {
  document.getElementById('live-waiting').classList.add('hidden');
  document.getElementById('live-match').classList.remove('hidden');

  const g = data.Game || {};
  document.getElementById('live-arena').textContent = friendlyArena(g.Arena);
  document.getElementById('live-clock').textContent = formatClock(g.TimeSeconds);
  document.getElementById('live-overtime').classList.toggle('hidden', !g.bOvertime);
  document.getElementById('live-replay-badge').classList.toggle('hidden', !g.bReplay);

  const bsEl = document.getElementById('live-ball-speed');
  const ball = g.Ball || {};
  if (bsEl) bsEl.textContent = ball.Speed != null ? `${Math.round(ball.Speed)} kph` : '';

  const teams = g.Teams || [];
  const t0 = teams.find(t => t.TeamNum === 0) || {};
  const t1 = teams.find(t => t.TeamNum === 1) || {};

  document.getElementById('team0-name').textContent = t0.Name || 'Blue';
  document.getElementById('team1-name').textContent = t1.Name || 'Orange';
  document.getElementById('score0').textContent     = t0.Score ?? 0;
  document.getElementById('score1').textContent     = t1.Score ?? 0;

  applyTeamColor('team0-name', 'score0', t0.ColorPrimary || null);
  applyTeamColor('team1-name', 'score1', t1.ColorPrimary || null);

  const players = data.Players || [];
  updatePossessionBar(players);
  renderPlayers(players, t0.ColorPrimary || null, t1.ColorPrimary || null);
}

function applyTeamColor(nameId, scoreId, hexColor) {
  const c = hexColor ? `#${hexColor}` : '';
  document.getElementById(nameId).style.color  = c;
  document.getElementById(scoreId).style.color = c;
}

function updatePossessionBar(players) {
  const blue   = players.filter(p => p.TeamNum === 0);
  const orange = players.filter(p => p.TeamNum === 1);
  const bt = blue.reduce((s, p)   => s + (p.Touches || 0), 0);
  const ot = orange.reduce((s, p) => s + (p.Touches || 0), 0);
  const total = bt + ot;

  if (total > 0) {
    const bp = Math.round(bt / total * 100);
    const op = 100 - bp;
    document.getElementById('possession-pct-blue').textContent   = bp + '%';
    document.getElementById('possession-pct-orange').textContent = op + '%';
    document.getElementById('possession-bar-blue').style.width   = bp + '%';
    document.getElementById('possession-bar-orange').style.width = op + '%';
  } else {
    document.getElementById('possession-pct-blue').textContent   = '—';
    document.getElementById('possession-pct-orange').textContent = '—';
    document.getElementById('possession-bar-blue').style.width   = '50%';
    document.getElementById('possession-bar-orange').style.width = '50%';
  }
}

function renderPlayers(players, t0color = null, t1color = null) {
  prefetchTrackerRanks(players);
  const grid = document.getElementById('players-grid');
  const blue   = players.filter(p => p.TeamNum === 0);
  const orange = players.filter(p => p.TeamNum === 1);
  grid.innerHTML =
    teamPanel('Blue',   'blue',   blue,   t0color) +
    teamPanel('Orange', 'orange', orange, t1color);
}

function teamPanel(name, cls, players, teamHexColor = null) {
  const headerStyle = teamHexColor
    ? ` style="background:${hexToRgba(teamHexColor, 0.15)};color:#${teamHexColor}"`
    : '';

  const rows = players.map(p => {
    const boost = p.Boost ?? null;
    const demolished = p.bDemolished ? 'demolished' : '';

    const boostBar = boost != null ? `
      <div class="boost-bar-wrap">
        <div class="boost-bar"><div class="boost-fill ${boostClass(boost)}" style="width:${boost}%"></div></div>
        <span class="boost-num">${boost}</span>
      </div>` : '<span style="color:var(--muted)">—</span>';

    const trnUrl = !isBot(p.PrimaryId) ? trnProfileUrl(p.PrimaryId, p.Name) : '';
    const nameEl = trnUrl
      ? `<a href="${esc(trnUrl)}" target="_blank" rel="noopener" class="player-name player-name-link">${esc(p.Name)}</a>`
      : `<span class="player-name">${esc(p.Name)}</span>`;

    return `
      <div class="player-row ${demolished}">
        <div class="player-name-cell">${nameEl}</div>
        <span>${p.Goals   ?? 0}</span>
        <span>${p.Assists ?? 0}</span>
        <span>${p.Saves   ?? 0}</span>
        <span>${p.Shots   ?? 0}</span>
        <span>${p.Demos   ?? 0}</span>
        <span>${p.Touches ?? 0}</span>
        ${boostBar}
        <span>${p.Score   ?? 0}</span>
      </div>`;
  }).join('');

  const statusRows = players.map(p => {
    const demolished = p.bDemolished ? 'demolished' : '';
    return `
      <div class="team-status-row ${demolished}">
        <span class="team-status-name">${esc(p.Name)}</span>
        <div class="status-wrap">
          ${chip('status-chip-ss',    'SS',    !!p.bSupersonic)}
          ${chip('status-chip-demo',  'DEMO',  !!p.bDemolished)}
          ${chip('status-chip-boost', 'BOOST', !!p.bBoosting)}
          ${chip('',                  'WALL',  !!p.bOnWall)}
          ${chip('',                  'PS',    !!p.bPowerslide)}
        </div>
      </div>`;
  }).join('');

  const rankRows = players.map(p => {
    if (isBot(p.PrimaryId)) return '';
    const rankHTML = trackerRankHTML(p.PrimaryId);
    if (!rankHTML) return '';
    const demolished = p.bDemolished ? 'demolished' : '';
    return `
      <div class="team-rank-row ${demolished}">
        <span class="team-rank-name">${esc(p.Name)}</span>
        ${rankHTML}
      </div>`;
  }).join('');

  const rankSection = rankRows.trim()
    ? `<div class="team-rank-section">${rankRows}</div>`
    : '';

  return `
    <div class="team-panel">
      <div class="team-panel-header ${cls}"${headerStyle}>${esc(name)}</div>
      <div class="player-row header">
        <span>Player</span><span>G</span><span>A</span><span>Sv</span><span>Sh</span><span>Dm</span>
        <span>Tch</span><span>Boost</span><span>Score</span>
      </div>
      ${rows || '<div class="player-row"><span style="color:var(--muted)">No players</span></div>'}
      <div class="team-status-section">${statusRows}</div>
      ${rankSection}
    </div>`;
}

function boostClass(b) {
  if (b < 30) return 'low';
  if (b < 70) return 'mid';
  return 'high';
}

function flashGoal(data) {
  const scorer = data.Scorer ? data.Scorer.Name : '?';
  const banner = document.createElement('div');
  banner.style.cssText = `
    position:fixed;top:60px;left:50%;transform:translateX(-50%);
    background:#7c3aed;color:#fff;padding:10px 24px;border-radius:8px;
    font-weight:700;font-size:16px;z-index:999;pointer-events:none;
    font-family:Inter,ui-sans-serif,system-ui,sans-serif;`;
  banner.textContent = `GOAL — ${scorer}`;
  document.body.appendChild(banner);
  setTimeout(() => banner.remove(), 3000);
}

function clearLive() {
  const waiting = document.getElementById('live-waiting');
  const match   = document.getElementById('live-match');
  if (waiting) waiting.classList.remove('hidden');
  if (match)   match.classList.add('hidden');
  updatePossessionBar([]);
}

window.pluginInit_live = async function() {
  try {
    const s = await fetch('/api/live/state').then(r => r.json());
    if (s.active && s.state) handleUpdateState(s.state);
  } catch (_) {}
};