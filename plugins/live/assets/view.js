'use strict';

let _liveInstances = [];
let _liveState     = null;
let _liveActive    = false;

function formatClock(secs) {
  if (secs == null) return '—';
  const m = Math.floor(secs / 60);
  const s = String(secs % 60).padStart(2, '0');
  return `${m}:${s}`;
}

function handleUpdateState(data) {
  _liveState  = data;
  _liveActive = true;
  _liveInstances.forEach(w => w.render(data));

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
  if (!data.Scorer || !data.Scorer.Name) return;
  const scorer = data.Scorer.Name;
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
  _liveActive = false;
  _liveInstances.forEach(w => w.renderEmpty());

  const waiting = document.getElementById('live-waiting');
  const match   = document.getElementById('live-match');
  if (waiting) waiting.classList.remove('hidden');
  if (match)   match.classList.add('hidden');
  updatePossessionBar([]);
}

function liveScoreboardWidget(container) {
  container.innerHTML = `
    <div class="ls-waiting" style="text-align:center;color:var(--muted);padding:32px 8px;font-size:13px">Waiting for a match…</div>
    <div class="ls-board hidden" style="padding:8px">
      <div style="display:flex;align-items:center;justify-content:space-between;gap:8px">
        <div style="text-align:left;min-width:0">
          <div class="ls-t0-name" style="font-size:13px;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis"></div>
          <div class="ls-score0 tabular-nums" style="font-size:36px;font-weight:800;line-height:1.1">0</div>
        </div>
        <div style="text-align:center;flex:1">
          <div class="ls-arena" style="font-size:11px;color:var(--muted)"></div>
          <div class="ls-clock tabular-nums" style="font-size:20px;font-weight:700">5:00</div>
          <div style="display:flex;gap:4px;justify-content:center;margin-top:2px">
            <span class="ls-ot hidden" style="background:#7c3aed;color:#fff;font-size:10px;font-weight:700;padding:1px 6px;border-radius:4px">OT</span>
            <span class="ls-replay hidden" style="color:#a78bfa;font-size:10px;font-weight:700;letter-spacing:.05em">REPLAY</span>
          </div>
          <div class="ls-ball-speed" style="font-size:10px;color:var(--muted);margin-top:2px"></div>
        </div>
        <div style="text-align:right;min-width:0">
          <div class="ls-t1-name" style="font-size:13px;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis"></div>
          <div class="ls-score1 tabular-nums" style="font-size:36px;font-weight:800;line-height:1.1">0</div>
        </div>
      </div>
      <div style="display:flex;align-items:center;gap:6px;margin-top:8px">
        <span class="ls-pct-blue tabular-nums" style="font-size:11px;font-weight:600;width:28px;text-align:right;color:var(--rl-blue)">—</span>
        <div style="flex:1;height:4px;background:var(--surface2);border-radius:2px;overflow:hidden;display:flex">
          <div class="ls-bar-blue" style="height:100%;background:var(--rl-blue);width:50%;transition:width .5s"></div>
          <div class="ls-bar-orange" style="height:100%;background:var(--rl-orange);width:50%;transition:width .5s"></div>
        </div>
        <span class="ls-pct-orange tabular-nums" style="font-size:11px;font-weight:600;width:28px;color:var(--rl-orange)">—</span>
      </div>
    </div>
  `;

  const q = sel => container.querySelector(sel);

  function renderEmpty() {
    q('.ls-waiting').classList.remove('hidden');
    q('.ls-board').classList.add('hidden');
  }

  function render(data) {
    const g       = data.Game   || {};
    const teams   = g.Teams     || [];
    const t0      = teams.find(t => t.TeamNum === 0) || {};
    const t1      = teams.find(t => t.TeamNum === 1) || {};
    const players = data.Players || [];

    q('.ls-waiting').classList.add('hidden');
    q('.ls-board').classList.remove('hidden');

    q('.ls-arena').textContent = friendlyArena(g.Arena);
    q('.ls-clock').textContent = formatClock(g.TimeSeconds);
    q('.ls-ot').classList.toggle('hidden', !g.bOvertime);
    q('.ls-replay').classList.toggle('hidden', !g.bReplay);
    const ball = g.Ball || {};
    q('.ls-ball-speed').textContent = ball.Speed != null ? `${Math.round(ball.Speed)} kph` : '';

    const c0 = t0.ColorPrimary ? `#${t0.ColorPrimary}` : 'var(--rl-blue)';
    const c1 = t1.ColorPrimary ? `#${t1.ColorPrimary}` : 'var(--rl-orange)';
    q('.ls-t0-name').textContent = t0.Name || 'Blue';
    q('.ls-t0-name').style.color = c0;
    q('.ls-score0').textContent  = t0.Score ?? 0;
    q('.ls-score0').style.color  = c0;
    q('.ls-t1-name').textContent = t1.Name || 'Orange';
    q('.ls-t1-name').style.color = c1;
    q('.ls-score1').textContent  = t1.Score ?? 0;
    q('.ls-score1').style.color  = c1;

    const blue   = players.filter(p => p.TeamNum === 0);
    const orange = players.filter(p => p.TeamNum === 1);
    const bt     = blue.reduce((s, p)   => s + (p.Touches || 0), 0);
    const ot     = orange.reduce((s, p) => s + (p.Touches || 0), 0);
    const total  = bt + ot;
    if (total > 0) {
      const bp = Math.round(bt / total * 100);
      q('.ls-pct-blue').textContent   = `${bp}%`;
      q('.ls-pct-orange').textContent = `${100 - bp}%`;
      q('.ls-bar-blue').style.width   = `${bp}%`;
      q('.ls-bar-orange').style.width = `${100 - bp}%`;
    } else {
      q('.ls-pct-blue').textContent   = '—';
      q('.ls-pct-orange').textContent = '—';
      q('.ls-bar-blue').style.width   = '50%';
      q('.ls-bar-orange').style.width = '50%';
    }
  }

  function refresh() {
    fetch('/api/live/state').then(r => r.json()).then(s => {
      if (s.active && s.state) render(s.state);
      else renderEmpty();
    }).catch(() => {});
  }

  function destroy() {
    const i = _liveInstances.indexOf(entry);
    if (i >= 0) _liveInstances.splice(i, 1);
  }

  const entry = { render, renderEmpty };
  _liveInstances.push(entry);
  if (_liveActive && _liveState) render(_liveState); else renderEmpty();
  return { refresh, destroy };
}

function livePlayersWidget(container) {
  function renderEmpty() {
    container.innerHTML = '<div style="text-align:center;color:var(--muted);padding:32px 8px;font-size:13px">Waiting for a match…</div>';
  }

  function render(data) {
    const g       = data.Game   || {};
    const teams   = g.Teams     || [];
    const t0      = teams.find(t => t.TeamNum === 0) || {};
    const t1      = teams.find(t => t.TeamNum === 1) || {};
    const players = data.Players || [];

    prefetchTrackerRanks(players);
    const blue   = players.filter(p => p.TeamNum === 0);
    const orange = players.filter(p => p.TeamNum === 1);
    container.innerHTML =
      `<div class="players-grid">` +
        teamPanel('Blue',   'blue',   blue,   t0.ColorPrimary || null) +
        teamPanel('Orange', 'orange', orange, t1.ColorPrimary || null) +
      `</div>`;
  }

  function refresh() {
    fetch('/api/live/state').then(r => r.json()).then(s => {
      if (s.active && s.state) render(s.state);
      else renderEmpty();
    }).catch(() => {});
  }

  function destroy() {
    const i = _liveInstances.indexOf(entry);
    if (i >= 0) _liveInstances.splice(i, 1);
  }

  const entry = { render, renderEmpty };
  _liveInstances.push(entry);
  if (_liveActive && _liveState) render(_liveState); else renderEmpty();
  return { refresh, destroy };
}

window.pluginInit_live = async function() {
  window.registerWidget?.({
    id: 'live-scoreboard', pluginId: 'live', title: 'Live Score',
    defaultW: 6, defaultH: 5, minW: 4, minH: 4,
    factory: liveScoreboardWidget,
  });
  window.registerWidget?.({
    id: 'live-players', pluginId: 'live', title: 'Live Players',
    defaultW: 12, defaultH: 10, minW: 6, minH: 6,
    factory: livePlayersWidget,
  });

  try {
    const s = await fetch('/api/live/state').then(r => r.json());
    if (s.active && s.state) handleUpdateState(s.state);
  } catch (_) {}
};