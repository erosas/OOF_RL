'use strict';

// --- Navigation ---
const views   = document.querySelectorAll('.view');
const navBtns = document.querySelectorAll('.nav-btn');

function showView(name) {
  views.forEach(v => v.classList.toggle('active', v.id === 'view-' + name));
  navBtns.forEach(b => b.classList.toggle('active', b.dataset.view === name));
  if (name === 'history') loadHistory();
  if (name === 'players') loadPlayers();
  if (name === 'replay')  loadReplayCaptures();
  if (name !== 'replay')  _replayReRender = null;
  if (name !== 'history') _historyDetailReRender = null;
  if (name === 'settings') loadSettings();
  if (name === 'bc') loadBC();
}

navBtns.forEach(b => b.addEventListener('click', () => showView(b.dataset.view)));

// --- WebSocket ---
const dot = document.getElementById('status-dot');

function connectWS() {
  const ws = new WebSocket(`ws://${location.host}/ws`);

  ws.onmessage = e => {
    const msg = JSON.parse(e.data);
    if (msg.Event === '_Status') {
      dot.className = 'dot ' + (msg.Data.connected ? 'connected' : 'disconnected');
      dot.title = msg.Data.connected ? 'Connected to Rocket League' : 'Waiting for Rocket League';
      if (!msg.Data.connected) clearLive();
      return;
    }
    if (msg.Event === 'UpdateState')    handleUpdateState(msg.Data);
    if (msg.Event === 'GoalScored')     flashGoal(msg.Data);
    if (msg.Event === 'MatchDestroyed') clearLive();
  };

  ws.onclose = () => {
    dot.className = 'dot disconnected';
    setTimeout(connectWS, 3000);
  };
}

connectWS();

// --- Arena name mapping ---
const ARENA_NAMES = {
  cs_day_p:               'Champions Field (Day)',
  cs_p:                   'Champions Field',
  cs_night_p:             'Champions Field (Night)',
  eurostadium_p:          'Mannfield',
  eurostadium_night_p:    'Mannfield (Night)',
  eurostadium_rainy_p:    'Mannfield (Stormy)',
  arc_p:                  'Starbase ARC',
  arc_standard_p:         'Starbase ARC (Standard)',
  stadium_p:              'DFH Stadium',
  stadium_winter_p:       'DFH Stadium (Snowy)',
  stadium_day_p:          'DFH Stadium (Day)',
  stadium_race_day_p:     'DFH Stadium',
  trainstation_p:         'Urban Central',
  trainstation_dawn_p:    'Urban Central (Dawn)',
  trainstation_night_p:   'Urban Central (Night)',
  park_p:                 'Beckwith Park',
  park_night_p:           'Beckwith Park (Midnight)',
  park_rainy_p:           'Beckwith Park (Stormy)',
  wasteland_s_p:          'Wasteland (Standard)',
  wasteland_p:            'Wasteland',
  neotokyo_p:             'Tokyo Underpass',
  neotokyo_standard_p:    'Tokyo Underpass (Standard)',
  utopiastadium_p:        'Utopia Coliseum',
  utopiastadium_dusk_p:   'Utopia Coliseum (Dusk)',
  utopiastadium_snow_p:   'Utopia Coliseum (Snowy)',
  underwater_p:           'AquaDome',
  hoopsstadium_p:         'Dunk House',
  throwbackstadium_p:     'Throwback Stadium',
  farmstead_p:            'Farmstead',
  farmstead_night_p:      'Farmstead (Night)',
  salty_shores_p:         'Salty Shores',
  salty_shores_night_p:   'Salty Shores (Night)',
  haunted_trainstation_p: 'Forbidden Temple',
  neon_fields_p:          'Neon Fields',
  bb_p:                   'Rivals Arena',
  beach_p:                'Beachside',
  cosmic_p:               'Cosmic',
  deadeye_canyon_p:       'Deadeye Canyon',
  forest_night_p:         'Forest',
};

function friendlyArena(code) {
  if (!code) return '—';
  return ARENA_NAMES[code.toLowerCase().trim()] || code;
}

// Rocket League playlist IDs → friendly names.
const PLAYLIST_NAMES = {
  1:  'Casual 1v1',
  2:  'Casual 2v2',
  3:  'Casual 3v3',
  4:  'Custom',
  6:  'Private',
  10: 'Ranked 1v1',
  11: 'Ranked 2v2',
  13: 'Ranked 3v3',
  14: 'Solo 3v3',
  22: 'Tournament',
  27: 'Rocket Labs',
  28: 'Rumble',
  29: 'Dropshot',
  30: 'Hoops',
  31: 'Snow Day',
  34: 'Casual Chaos',
  35: 'Gridiron',
  41: 'Heatseeker',
  43: 'Spike Rush',
};

function friendlyPlaylist(id) {
  if (id == null) return '';
  return PLAYLIST_NAMES[id] || '';
}

function matchType(playerCount) {
  if (!playerCount) return '';
  const n = Math.ceil(playerCount / 2);
  return `${n}v${n}`;
}

function hexToRgba(hex, alpha) {
  if (!hex) return null;
  const h = hex.replace('#', '');
  if (h.length !== 6) return null;
  const r = parseInt(h.slice(0, 2), 16);
  const g = parseInt(h.slice(2, 4), 16);
  const b = parseInt(h.slice(4, 6), 16);
  return `rgba(${r},${g},${b},${alpha})`;
}

// Always rendered; add class "dim" when inactive so the row height never shifts.
function chip(cls, label, active) {
  const colorCls = cls ? ` ${cls}` : '';
  const dimCls   = active ? '' : ' dim';
  return `<span class="status-chip${colorCls}${dimCls}">${label}</span>`;
}

// --- Tracker.gg rank cache ---
// Stores { profile, ranks, status, fetchedAt } per player,
// where `profile` is the full json.data from the API response.
// null on failure, undefined while in-flight.
const trackerCache = new Map();

function platformFromId(primaryId) {
  if (!primaryId) return '';
  const sep = String(primaryId).search(/[|:_]/);
  if (sep < 1) return '';
  return primaryId.slice(0, sep).toLowerCase();
}

// Normalize RL Stats API platform slugs to TRN/tracker.gg slugs.
const _PLAT_NORM = { ps4: 'psn', ps5: 'psn', playstation: 'psn', xboxone: 'xbl', xbox: 'xbl', epicgames: 'epic', nintendo: 'switch' };
function normPlatform(plat) { return _PLAT_NORM[plat] || plat; }

// Builds a tracker.gg profile URL from a primary ID + display name.
// Non-Steam platforms use the display name as the identifier.
function trnProfileUrl(primaryId, playerName) {
  if (!primaryId) return '';
  const sep = String(primaryId).search(/[|:_]/);
  if (sep < 1) return '';
  const plat = normPlatform(primaryId.slice(0, sep).toLowerCase());
  const rawId = primaryId.slice(sep + 1);
  const identifier = (plat !== 'steam' && playerName) ? playerName : rawId;
  return `https://tracker.gg/rocket-league/profile/${encodeURIComponent(plat)}/${encodeURIComponent(identifier)}`;
}

function shortPlaylistName(name) {
  return name.replace(/^Ranked\s+/i, '');
}

// Returns a human-readable "X ago" string from an ISO timestamp.
function timeAgo(isoStr) {
  if (!isoStr) return '';
  const secs = Math.round((Date.now() - new Date(isoStr).getTime()) / 1000);
  if (secs < 60)   return `${secs}s ago`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
  return `${Math.floor(secs / 3600)}h ago`;
}

// Parses TRN segments into a { [playlistId]: { playlistName, tierName, icon, rating } } map.
// Uses metadata.name from each segment — no hardcoded playlist lookup needed.
function parseTrackerSegments(segments) {
  const ranks = {};
  for (const seg of segments) {
    if (seg.type !== 'playlist') continue;
    const pid  = seg.attributes?.playlistId;
    if (pid == null) continue;
    const tier = seg.stats?.tier?.metadata || {};
    ranks[pid] = {
      playlistName: seg.metadata?.name || String(pid),
      tierName: tier.name || 'Unranked',
      icon: tier.iconUrl || '',
      rating: seg.stats?.rating?.displayValue || '',
    };
  }
  return Object.keys(ranks).length ? ranks : null;
}

// Returns true for bot/unknown players that have no real platform ID.
function isBot(primaryId) {
  if (!primaryId) return true;
  const plat = platformFromId(primaryId);
  return !plat || plat === 'unknown';
}

// Returns true for cross-platform Switch players whose identity RL masks as "**…**".
function isMaskedName(name) {
  return typeof name === 'string' && name.length > 0 && /^\*+$/.test(name);
}

// Staggered tracker fetch queue — requests drain one at a time with a delay
// so the server's rate limiter isn't flooded with simultaneous goroutines.
const _trkQueue = [];
let   _trkDraining = false;

function _drainTrkQueue() {
  if (!_trkQueue.length) { _trkDraining = false; return; }
  _trkDraining = true;
  const { id, plat, name } = _trkQueue.shift();
  const nameParam = plat && plat !== 'steam' && name
    ? `&name=${encodeURIComponent(name)}` : '';
  fetch(`/api/tracker/profile?id=${encodeURIComponent(id)}${nameParam}`)
    .then(r => {
      const status = `HTTP ${r.status}`;
      return r.ok ? r.json().then(j => ({ json: j, status })) : Promise.resolve({ json: null, status });
    })
    .then(({ json, status }) => {
      const ranks    = parseTrackerSegments(json?.data?.segments || []);
      const fetchedAt = json?.fetched_at || null;
      const profile  = json?.data || null;
      trackerCache.set(id, { profile, ranks, status, fetchedAt });
      onTrackerDataArrived();
    })
    .catch(e => {
      trackerCache.set(id, { profile: null, ranks: null, status: `Error: ${e.message}`, fetchedAt: null });
      onTrackerDataArrived();
    })
    .finally(() => setTimeout(_drainTrkQueue, 2500)); // 2.5 s between dispatches
}

// Kicks off tracker fetches for any players not yet in the in-memory cache.
// Routes through the server so the DB cache is checked/written there.
function prefetchTrackerRanks(players) {
  let enqueued = false;
  for (const p of players) {
    const id = p.PrimaryId;
    if (!id || trackerCache.has(id)) continue;
    if (isBot(id)) continue;
    const plat = platformFromId(id);
    if (plat !== 'steam' && isMaskedName(p.Name)) {
      // Cross-platform privacy mask — no TRN profile exists.
      trackerCache.set(id, { profile: null, ranks: null, status: 'masked', fetchedAt: null });
      continue;
    }
    trackerCache.set(id, undefined); // mark in-flight
    _trkQueue.push({ id, plat, name: p.Name });
    enqueued = true;
  }
  // 2 s startup delay before the first request so the queue doesn't fire
  // immediately when a match starts or a page loads.
  if (enqueued && !_trkDraining) setTimeout(_drainTrkQueue, 2000);
}

// Called when any tracker data lands — re-renders replay (paused) and history detail.
let _replayReRender = null;
let _historyDetailReRender = null;
function onTrackerDataArrived() {
  if (_replayReRender) _replayReRender();
  if (_historyDetailReRender) _historyDetailReRender();
}

// Returns a compact multi-mode rank line for use inside a player-name-cell.
// Shows every playlist the API returned with mode label, platform badge, and update time.
function trackerRankHTML(primaryId) {
  if (!primaryId) return '';
  if (!trackerCache.has(primaryId)) return '';
  const entry = trackerCache.get(primaryId);
  const plat = platformFromId(primaryId);
  const platformBadge = plat ? `<span class="player-platform-badge">${esc(plat)}</span>` : '';
  if (entry === undefined) {
    return `<div class="player-trk-rank">${platformBadge}<span class="player-trk-name">···</span></div>`;
  }
  const { ranks, status, fetchedAt } = entry;
  if (!ranks) {
    return `<div class="player-trk-rank">${platformBadge}<span class="player-trk-name">${esc(status || 'err')}</span></div>`;
  }
  const pills = Object.values(ranks).map(r =>
    `<span class="player-trk-pill" title="${esc(r.playlistName)}: ${esc(r.tierName)} ${esc(r.rating)}">${
      r.icon ? `<img src="${esc(r.icon)}" class="player-trk-icon" alt="">` : ''
    }<span class="trk-pill-mode">${esc(shortPlaylistName(r.playlistName))}</span>${esc(r.tierName)}${
      r.rating ? ` <span class="player-trk-mmr">${esc(r.rating)}</span>` : ''
    }</span>`
  ).join('');
  const updatedStr = fetchedAt
    ? `<span class="player-trk-updated" title="${esc(fetchedAt)}">${timeAgo(fetchedAt)}</span>`
    : '';
  return `<div class="player-trk-rank">${platformBadge}${pills}${updatedStr}</div>`;
}

// --- Live view ---
function handleUpdateState(data) {
  document.getElementById('live-waiting').classList.add('hidden');
  document.getElementById('live-match').classList.remove('hidden');

  const g = data.Game || {};
  document.getElementById('live-arena').textContent = friendlyArena(g.Arena);
  document.getElementById('live-clock').textContent = formatClock(g.TimeSeconds);
  document.getElementById('live-overtime').classList.toggle('hidden', !g.bOvertime);
  document.getElementById('live-replay-badge').classList.toggle('hidden', !g.bReplay);

  const ball = g.Ball || {};
  const bsEl = document.getElementById('live-ball-speed');
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

function formatClock(secs) {
  if (secs == null) return '—';
  const m = Math.floor(secs / 60);
  const s = String(secs % 60).padStart(2, '0');
  return `${m}:${s}`;
}

function renderPlayers(players, t0color = null, t1color = null) {
  prefetchTrackerRanks(players);
  const grid = document.getElementById('players-grid');
  const blue   = players.filter(p => p.TeamNum === 0);
  const orange = players.filter(p => p.TeamNum === 1);
  grid.innerHTML =
    teamPanel('Blue',   'blue',   blue,   null, t0color) +
    teamPanel('Orange', 'orange', orange, null, t1color);
}

// Unified player panel — live and replay share this component.
// Pass `insights` for the TOUCH chip (replay only); `teamHexColor` to override header color.
function teamPanel(name, cls, players, insights = null, teamHexColor = null) {
  const headerStyle = teamHexColor
    ? ` style="background:${hexToRgba(teamHexColor, 0.15)};color:#${teamHexColor}"`
    : '';

  // Pre-compute touch insight per player
  const playerData = players.map(p => {
    let touchActive = false;
    if (insights) {
      const key = replayPlayerKey(p);
      const ins = insights.byKey.get(key) ||
                  insights.byName.get((p.Name || '').toLowerCase()) ||
                  emptyInsight();
      touchActive = ins.recentTouch;
    }
    return { p, touchActive };
  });

  // Stat rows — name + numbers only, no chips or rank pills
  const rows = playerData.map(({ p }) => {
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
        <div class="player-name-cell">
          ${nameEl}
        </div>
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

  // Status section — one row per player showing all chips
  const statusRows = playerData.map(({ p, touchActive }) => {
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
          ${insights != null ? chip('status-chip-touch', 'TOUCH', touchActive) : ''}
        </div>
      </div>`;
  }).join('');

  // Rank section — one row per player showing rank pills with MMR (skip bots)
  const rankRows = playerData.map(({ p }) => {
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
  document.getElementById('live-waiting').classList.remove('hidden');
  document.getElementById('live-match').classList.add('hidden');
  updatePossessionBar([]);
}

// --- History ---
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

document.getElementById('history-player-filter').addEventListener('change', e => fetchMatches(e.target.value));

async function fetchMatches(playerID) {
  _expandedMatchId = null;
  _historyDetailReRender = null;
  const url = playerID ? `/api/matches?player=${encodeURIComponent(playerID)}` : '/api/matches';
  const matches = await fetch(url).then(r => r.json()) || [];
  const list = document.getElementById('matches-list');

  // Filter matches with no meaningful arena (RL sometimes reports "-" for untracked modes)
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

  // Normalise PrimaryId — DB model uses PrimaryID, live view uses PrimaryId
  players.forEach(p => { if (!p.PrimaryId) p.PrimaryId = p.PrimaryID; });
  const realPlayers = players.filter(p => !isBot(p.PrimaryId));
  prefetchTrackerRanks(realPlayers);

  const blue   = realPlayers.filter(p => p.TeamNum === 0);
  const orange = realPlayers.filter(p => p.TeamNum === 1);

  // Map player name → team for coloring goal rows
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

// --- Players ---
let _expandedPlayerId = null;

async function loadPlayers() {
  const players = await fetch('/api/players').then(r => r.json()) || [];
  const list = document.getElementById('players-list');
  _expandedPlayerId = null;

  if (!players.length) {
    list.innerHTML = '<p class="text-gray-600 py-5">No players recorded yet.</p>';
    return;
  }

  list.innerHTML = players.map(p => `
    <div class="player-card" data-id="${esc(p.PrimaryID)}" data-name="${esc(p.Name)}">
      <div>
        <div class="player-card-name">${esc(p.Name)}</div>
        <div class="player-card-id">${esc(p.PrimaryID)}</div>
      </div>
      <span class="player-expand-chevron">›</span>
    </div>`).join('');

  list.querySelectorAll('.player-card').forEach(card => {
    card.addEventListener('click', () => togglePlayerInline(card));
  });
}

async function togglePlayerInline(card) {
  const id   = card.dataset.id;
  const name = card.dataset.name;

  // Clicking an already-open card collapses it
  const existing = card.nextElementSibling;
  if (existing && existing.classList.contains('player-inline-panel')) {
    existing.remove();
    card.classList.remove('expanded');
    _expandedPlayerId = null;
    return;
  }

  // Collapse any other open panel
  document.querySelectorAll('.player-inline-panel').forEach(el => el.remove());
  document.querySelectorAll('.player-card.expanded').forEach(el => el.classList.remove('expanded'));

  card.classList.add('expanded');
  _expandedPlayerId = id;

  const panel = document.createElement('div');
  panel.className = 'player-inline-panel';
  panel.innerHTML = '<div style="padding:16px;color:var(--muted);font-size:13px">Loading…</div>';
  card.insertAdjacentElement('afterend', panel);

  const html = await buildPlayerDetailHTML(id, name);
  if (_expandedPlayerId === id) panel.innerHTML = html;
}

async function buildPlayerDetailHTML(primaryID, displayName) {
  let trackerResp = null;
  let trackerStatus = '';
  try {
    const plat = platformFromId(primaryID);
    const nameParam = plat && plat !== 'steam' && displayName
      ? `&name=${encodeURIComponent(displayName)}` : '';
    const r = await fetch(`/api/tracker/profile?id=${encodeURIComponent(primaryID)}${nameParam}`);
    trackerStatus = `HTTP ${r.status}`;
    if (r.ok) trackerResp = await r.json();
  } catch (e) {
    trackerStatus = `Network error — ${e.message}`;
  }

  const data    = await fetch(`/api/players/${encodeURIComponent(primaryID)}`).then(r => r.json()).catch(() => ({}));
  const agg     = data.aggregate || {};
  const matches = (data.matches || []).filter(m => m.Arena && m.Arena !== '-');

  const matchRows = matches.slice(0, 20).map(m => `
    <tr>
      <td>${esc(friendlyArena(m.Arena) || '—')}</td>
      <td>${formatDate(m.StartedAt)}</td>
      <td>${m.WinnerTeamNum === 0 ? '<span class="text-rl-blue">Blue</span>' : m.WinnerTeamNum === 1 ? '<span class="text-rl-orange">Orange</span>' : '—'}</td>
    </tr>`).join('');

  const trackerHTML = trackerResp?.data
    ? renderTrackerProfile(trackerResp.data, trackerResp.fetched_at)
    : `<div class="trn-unavailable">tracker.gg — ${esc(trackerStatus || 'unavailable')}</div>`;

  return `
    <div class="player-stats-row mb-4">
      ${statPill(agg.Matches,  'Matches')}
      ${statPill(agg.Goals,    'Goals')}
      ${statPill(agg.Assists,  'Assists')}
      ${statPill(agg.Saves,    'Saves')}
      ${statPill(agg.Shots,    'Shots')}
      ${statPill(agg.Demos,    'Demos')}
      ${statPill(agg.Touches,  'Touches')}
    </div>
    ${trackerHTML}
    <h3 class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">Recent Matches</h3>
    <table class="detail-table">
      <thead><tr><th>Arena</th><th>Date</th><th>Winner</th></tr></thead>
      <tbody>${matchRows || '<tr><td colspan="3" class="text-gray-600">No matches</td></tr>'}</tbody>
    </table>`;
}

function renderTrackerProfile(profile, fetchedAt) {
  const platform = profile.platformInfo || {};
  const segments = profile.segments || [];

  const overview = segments.find(s => s.type === 'overview');
  const os = overview?.stats || {};

  // Build rank cards from every playlist segment using their API-provided names.
  const ranks = parseTrackerSegments(segments);
  const rankCards = ranks
    ? Object.values(ranks).map(r => `
        <div class="trn-rank-card">
          ${r.icon ? `<img src="${esc(r.icon)}" class="trn-rank-icon" alt="">` : '<div class="trn-rank-icon-placeholder"></div>'}
          <div class="trn-rank-name">${esc(r.tierName)}</div>
          <div class="trn-rank-playlist">${esc(r.playlistName)}</div>
          <div class="trn-rank-rating">${esc(r.rating)}</div>
        </div>`).join('')
    : '<div class="trn-unavailable">No ranked data</div>';

  const trnUrl = `https://tracker.gg/rocket-league/profile/${esc(platform.platformSlug || 'steam')}/${esc(platform.platformUserIdentifier || '')}`;
  const season = profile.metadata?.currentSeason ? `Season ${profile.metadata.currentSeason}` : '';
  const updatedStr = fetchedAt
    ? `<span class="trn-updated" title="${esc(fetchedAt)}">Updated ${timeAgo(fetchedAt)}</span>`
    : '';

  return `
    <div class="trn-section">
      <div class="trn-header">
        ${platform.avatarUrl ? `<img src="${esc(platform.avatarUrl)}" class="trn-avatar" alt="">` : ''}
        <div class="trn-handle-wrap">
          <div class="trn-handle">${esc(platform.platformUserHandle || '—')}</div>
          <a href="${trnUrl}" target="_blank" rel="noopener" class="trn-link">tracker.gg ↗</a>
        </div>
        <div class="trn-header-meta">
          ${season ? `<div class="trn-season-badge">${esc(season)}</div>` : ''}
          ${updatedStr}
        </div>
      </div>
      <div class="trn-ranks">${rankCards}</div>
      <div class="trn-lifetime-label">Lifetime</div>
      <div class="player-stats-row">
        ${statPill(os.wins?.displayValue,    'Wins')}
        ${statPill(os.goals?.displayValue,   'Goals')}
        ${statPill(os.saves?.displayValue,   'Saves')}
        ${statPill(os.assists?.displayValue, 'Assists')}
        ${statPill(os.shots?.displayValue,   'Shots')}
        ${statPill(os.mVPs?.displayValue,    'MVPs')}
      </div>
    </div>`;
}

function statPill(val, lbl) {
  return `<div class="stat-pill"><div class="val">${val ?? 0}</div><div class="lbl">${lbl}</div></div>`;
}

// --- Replay ---
let replayPackets = [];
let replayCursor  = 0;
let replayTimer   = null;
let replaySpeed   = 1;

async function loadReplayCaptures() {
  const sel = document.getElementById('replay-capture');
  await loadLocalReplayFiles();
  let captures = [];
  try { captures = await fetch('/api/captures').then(r => r.json()); } catch (_) {}

  if (!captures.length) {
    sel.innerHTML = '<option value="">No captures found</option>';
    replayPackets = [];
    replayCursor  = 0;
    document.getElementById('replay-seek').value = 0;
    document.getElementById('replay-seek').max   = 0;
    document.getElementById('replay-players-grid').innerHTML = '';
    document.getElementById('replay-event').textContent      = '';
    return;
  }

  const prev = sel.value;
  sel.innerHTML = captures.map(c => {
    const ts = c.started_at_utc ? new Date(c.started_at_utc).toLocaleString() : c.id;
    return `<option value="${esc(c.id)}">${esc(ts)} · ${esc(c.match_guid || 'unknown')} · ${c.packet_count || 0} pkts</option>`;
  }).join('');
  if (prev && captures.find(c => c.id === prev)) sel.value = prev;
}

document.getElementById('replay-load').addEventListener('click', async () => {
  const id = document.getElementById('replay-capture').value;
  if (!id) return;
  stopReplay();
  try {
    const txt = await fetch(`/api/captures/${encodeURIComponent(id)}/events`).then(r => r.text());
    replayPackets = txt.split('\n').map(l => l.trim()).filter(Boolean).map(l => {
      try { return JSON.parse(l); } catch (_) { return null; }
    }).filter(Boolean);
    replayCursor = 0;
    const seek = document.getElementById('replay-seek');
    seek.max   = Math.max(0, replayPackets.length - 1);
    seek.value = 0;
    showMsg('replay-msg', `Loaded ${replayPackets.length} packets`, true);
    renderReplayPacketAt(0);
  } catch (_) {
    showMsg('replay-msg', 'Failed to load capture.', false);
    document.getElementById('replay-players-grid').innerHTML = '';
  }
});

document.getElementById('replay-play').addEventListener('click', () => {
  if (!replayPackets.length || replayTimer) return;
  replayTimer = setInterval(stepReplay, Math.max(10, Math.round(16 / replaySpeed)));
});

document.getElementById('replay-pause').addEventListener('click', stopReplay);

document.getElementById('replay-speed').addEventListener('change', e => {
  replaySpeed = parseFloat(e.target.value) || 1;
  if (replayTimer) { stopReplay(); replayTimer = setInterval(stepReplay, Math.max(10, Math.round(16 / replaySpeed))); }
});

document.getElementById('replay-seek').addEventListener('input', e => {
  replayCursor = parseInt(e.target.value || '0', 10);
  renderReplayPacketAt(replayCursor);
});

function stepReplay() {
  if (replayCursor >= replayPackets.length) { stopReplay(); return; }
  renderReplayPacketAt(replayCursor);
  replayCursor++;
  document.getElementById('replay-seek').value = Math.min(replayCursor, Math.max(0, replayPackets.length - 1));
}

function stopReplay() {
  if (replayTimer) { clearInterval(replayTimer); replayTimer = null; }
}

function renderReplayPacketAt(i) {
  if (!replayPackets.length || i < 0 || i >= replayPackets.length) return;
  const pkt = replayPackets[i];
  document.getElementById('replay-event').textContent = JSON.stringify({ index: i, ...pkt }, null, 2);
  const snap = replayStateAtCursor(i);
  if (snap) renderReplayState(snap.data, snap.packetIdx);
  else document.getElementById('replay-players-grid').innerHTML = '';
}

function replayStateAtCursor(idx) {
  for (let i = idx; i >= 0; i--) {
    const pkt = replayPackets[i];
    if (pkt && pkt.Event === 'UpdateState' && pkt.Data) return { data: pkt.Data, packetIdx: i };
  }
  return null;
}

function renderReplayState(data, packetIdx) {
  _replayReRender = () => renderReplayState(data, packetIdx);
  const g = data.Game || {};
  document.getElementById('replay-arena').textContent  = friendlyArena(g.Arena);
  document.getElementById('replay-clock').textContent  = formatClock(g.TimeSeconds);

  const teams = g.Teams || [];
  const t0 = teams.find(t => t.TeamNum === 0) || {};
  const t1 = teams.find(t => t.TeamNum === 1) || {};
  document.getElementById('replay-team0').textContent  = t0.Name || 'Blue';
  document.getElementById('replay-team1').textContent  = t1.Name || 'Orange';
  document.getElementById('replay-score0').textContent = t0.Score ?? 0;
  document.getElementById('replay-score1').textContent = t1.Score ?? 0;

  const mtEl = document.getElementById('replay-match-type');
  if (mtEl) mtEl.textContent = matchType((data.Players || []).length);

  const insights = replayPlayerInsights(packetIdx);
  renderReplayPlayers(data.Players || [], insights, t0.ColorPrimary || null, t1.ColorPrimary || null);
}

async function loadLocalReplayFiles() {
  const list = document.getElementById('replay-files');
  if (!list) return;
  let files = [];
  try { files = await fetch('/api/replays').then(r => r.json()); } catch (_) {}

  if (!Array.isArray(files) || !files.length) {
    list.innerHTML = '<p class="replay-subtle" style="padding:8px 10px">No local replay files found.</p>';
    return;
  }

  list.innerHTML = files.slice(0, 25).map(f => `
    <div class="replay-file-row">
      <span class="replay-file-name">${esc(f.name)}</span>
      <span class="replay-subtle">${formatBytes(f.size_bytes)} · ${f.modified_at ? formatDate(f.modified_at) : '—'}</span>
    </div>`).join('');
}

function renderReplayPlayers(players, insights, t0color = null, t1color = null) {
  prefetchTrackerRanks(players);
  const grid = document.getElementById('replay-players-grid');
  if (!grid) return;
  const blue   = players.filter(p => p.TeamNum === 0);
  const orange = players.filter(p => p.TeamNum === 1);
  grid.innerHTML =
    teamPanel('Blue',   'blue',   blue,   insights, t0color) +
    teamPanel('Orange', 'orange', orange, insights, t1color);
}

function replayPlayerKey(p) {
  return p.PrimaryId || p.Name || `${p.TeamNum || 0}-${p.Shortcut || 0}`;
}

function emptyInsight() { return { ballHits: 0, recentTouch: false, lastEvent: '' }; }

function replayPlayerInsights(packetIdx) {
  const byKey  = new Map();
  const byName = new Map();
  const touchWindow = 120;

  function ensure(key, name) {
    let s = byKey.get(key);
    if (!s) {
      s = { ballHits: 0, recentTouch: false, lastTouchPacket: -999999, lastEvent: '' };
      byKey.set(key, s);
    }
    if (name) byName.set(name.toLowerCase(), s);
    return s;
  }

  for (let i = 0; i <= packetIdx && i < replayPackets.length; i++) {
    const pkt = replayPackets[i];
    if (!pkt || !pkt.Data) continue;

    if (pkt.Event === 'BallHit') {
      const p = pkt.Data.Players?.[0] || {};
      const s = ensure(playerRefKey(p), p.Name);
      s.ballHits++;
      s.lastTouchPacket = i;
      s.lastEvent = 'BallHit';
    }

    if (pkt.Event === 'GoalScored') {
      ensure(playerRefKey(pkt.Data.Scorer || {}), (pkt.Data.Scorer || {}).Name).lastEvent = 'GoalScored';
      const assister = pkt.Data.Assister || {};
      if (assister.Name || assister.PrimaryId)
        ensure(playerRefKey(assister), assister.Name).lastEvent = 'GoalScored (assist)';
      const toucher = pkt.Data.BallLastTouch?.Player || {};
      const ts = ensure(playerRefKey(toucher), toucher.Name);
      ts.lastTouchPacket = i;
      ts.lastEvent = 'GoalScored (last touch)';
    }

    if (pkt.Event === 'StatfeedEvent') {
      const t = pkt.Data.MainTarget || {};
      if (t.Name || t.PrimaryId) {
        const label = pkt.Data.Type || pkt.Data.EventName || 'StatfeedEvent';
        ensure(playerRefKey(t), t.Name).lastEvent = `Stat: ${label}`;
      }
    }
  }

  byKey.forEach(v => { v.recentTouch = (packetIdx - v.lastTouchPacket) <= touchWindow; });
  return { byKey, byName };
}

function playerRefKey(ref) {
  if (!ref || typeof ref !== 'object') return 'unknown';
  return ref.PrimaryId || ref.Name || `${ref.TeamNum || 0}-${ref.Shortcut || 0}`;
}

// --- Settings / overlay hotkey capture ---
const _hotkeyBtn = document.getElementById('cfg-hotkey-btn');
const _allowedHotkeys = new Set([
  'F1','F2','F3','F4','F5','F6','F7','F8','F9','F10','F11','F12',
  'Insert','Delete','Home','End','PageUp','PageDown','Pause','ScrollLock',
]);
let _capturingHotkey = false;

_hotkeyBtn.addEventListener('click', () => {
  _capturingHotkey = true;
  _hotkeyBtn.textContent = 'Press a key…';
  _hotkeyBtn.classList.add('ring-2', 'ring-rl-blue');
});

document.addEventListener('keydown', e => {
  if (!_capturingHotkey) return;
  e.preventDefault();
  e.stopPropagation();
  if (_allowedHotkeys.has(e.key)) {
    _hotkeyBtn.textContent = e.key;
    _hotkeyBtn.dataset.value = e.key;
  } else {
    _hotkeyBtn.textContent = _hotkeyBtn.dataset.value || 'F9';
  }
  _capturingHotkey = false;
  _hotkeyBtn.classList.remove('ring-2', 'ring-rl-blue');
}, true);

// --- Settings ---
async function loadSettings() {
  const cfg = await fetch('/api/config').then(r => r.json());
  document.getElementById('cfg-port').value    = cfg.app_port;
  document.getElementById('cfg-rl-path').value = cfg.rl_install_path;
  document.getElementById('cfg-db-path').value = cfg.db_path;
  document.getElementById('cfg-match-meta').checked      = cfg.storage.match_metadata;
  document.getElementById('cfg-player-stats').checked    = cfg.storage.player_match_stats;
  document.getElementById('cfg-goals').checked           = cfg.storage.goal_events;
  document.getElementById('cfg-ball-hits').checked       = cfg.storage.ball_hit_events;
  document.getElementById('cfg-ticks').checked           = cfg.storage.tick_snapshots;
  document.getElementById('cfg-tick-rate').value         = cfg.storage.tick_snapshot_rate;
  document.getElementById('cfg-other-events').checked    = cfg.storage.other_events;
  document.getElementById('cfg-raw-packets').checked     = !!cfg.storage.raw_packets;
  document.getElementById('cfg-raw-packets-dir').value   = cfg.storage.raw_packets_dir || 'captures';
  document.getElementById('cfg-open-browser').checked    = !!cfg.open_in_browser;
  document.getElementById('cfg-tracker-ttl').value       = cfg.tracker_cache_ttl_minutes ?? 60;
  document.getElementById('cfg-bc-key').value            = cfg.ballchasing_api_key || '';
  const hk = cfg.overlay_hotkey || 'F9';
  _hotkeyBtn.textContent   = hk;
  _hotkeyBtn.dataset.value = hk;

  try {
    const ini = await fetch('/api/config/ini').then(r => r.json());
    const enabled = ini.PacketSendRate > 0;
    document.getElementById('ini-enabled').checked = enabled;
    document.getElementById('ini-rate').value = enabled ? ini.PacketSendRate : 60;
    document.getElementById('ini-port').value = ini.Port || 49123;
    if (ini.note) showMsg('ini-msg', ini.note, !ini.error);
  } catch (_) {
    showMsg('ini-msg', 'Could not read INI settings. Check RL install path above.', false);
  }
}

document.getElementById('save-cfg').addEventListener('click', async () => {
  const cfg = {
    app_port:                    parseInt(document.getElementById('cfg-port').value),
    rl_install_path:             document.getElementById('cfg-rl-path').value,
    db_path:                     document.getElementById('cfg-db-path').value,
    open_in_browser:             document.getElementById('cfg-open-browser').checked,
    tracker_cache_ttl_minutes:   parseInt(document.getElementById('cfg-tracker-ttl').value) || 0,
    ballchasing_api_key:         document.getElementById('cfg-bc-key').value.trim(),
    overlay_hotkey:              _hotkeyBtn.dataset.value || _hotkeyBtn.textContent || 'F9',
    storage: {
      match_metadata:     document.getElementById('cfg-match-meta').checked,
      player_match_stats: document.getElementById('cfg-player-stats').checked,
      goal_events:        document.getElementById('cfg-goals').checked,
      ball_hit_events:    document.getElementById('cfg-ball-hits').checked,
      tick_snapshots:     document.getElementById('cfg-ticks').checked,
      tick_snapshot_rate: parseFloat(document.getElementById('cfg-tick-rate').value),
      other_events:       document.getElementById('cfg-other-events').checked,
      raw_packets:        document.getElementById('cfg-raw-packets').checked,
      raw_packets_dir:    document.getElementById('cfg-raw-packets-dir').value.trim() || 'captures',
    },
  };
  const res = await fetch('/api/config', { method: 'POST', body: JSON.stringify(cfg) });
  showMsg('cfg-msg', res.ok ? 'Saved! App port changes require a restart.' : 'Error saving config.', res.ok);
  if (res.ok) await loadSettings();
});

document.getElementById('save-ini').addEventListener('click', async () => {
  const enabled = document.getElementById('ini-enabled').checked;
  const ini = {
    PacketSendRate: enabled ? parseFloat(document.getElementById('ini-rate').value) : 0,
    Port: parseInt(document.getElementById('ini-port').value),
  };
  const res = await fetch('/api/config/ini', { method: 'POST', body: JSON.stringify(ini) });
  let data = {};
  try { data = await res.json(); } catch (_) {}
  showMsg('ini-msg', res.ok ? (data.note || 'Saved!') : ('Error: ' + (data.error || 'unknown')), res.ok);
});

// --- Ballchasing tab ---

async function loadBC() {
  loadBCStatus();
  const [files, uploads, bcData] = await Promise.all([
    fetch('/api/replays').then(r => r.json()).catch(() => []),
    fetch('/api/ballchasing/uploads').then(r => r.json()).catch(() => ({})),
    fetch('/api/ballchasing/replays').then(r => r.json()).catch(() => null),
  ]);

  // Build a set of BC IDs we have locally
  const localById = {};
  Object.values(uploads).forEach(u => { localById[u.ballchasing_id] = u; });

  renderBCUploaded(bcData, uploads, localById);
  renderBCNotUploaded(files, uploads);
  loadBCGroups();
}

async function loadBCStatus() {
  const el = document.getElementById('bc-status');
  el.innerHTML = '<span class="bc-status-dot bc-dot-pending"></span> Checking…';
  try {
    const r = await fetch('/api/ballchasing/ping');
    if (!r.ok) {
      const j = await r.json().catch(() => ({}));
      el.innerHTML = `<span class="bc-status-dot bc-dot-err"></span> ${esc(j.error || 'Ballchasing API key not set — configure it in Settings.')}`;
      return;
    }
    const j = await r.json();
    const name = j.name || j.steam_name || '(connected)';
    el.innerHTML = `<span class="bc-status-dot bc-dot-ok"></span> Connected as <strong>${esc(name)}</strong>`;
  } catch (e) {
    el.innerHTML = `<span class="bc-status-dot bc-dot-err"></span> ${esc(e.message)}`;
  }
}

function renderBCUploaded(bcData, uploads, localById) {
  const el = document.getElementById('bc-uploaded');
  const list = bcData?.list || [];

  // Also include locally-tracked uploads that might not be in the BC API result
  // (e.g. uploaded long ago, not in the last 50)
  const bcIds = new Set(list.map(r => r.id));
  const localOnlyUploads = Object.values(uploads).filter(u => !bcIds.has(u.ballchasing_id));

  if (!list.length && !localOnlyUploads.length) {
    el.innerHTML = bcData === null
      ? '<div class="bc-empty">Configure your Ballchasing API key in Settings to load replays.</div>'
      : '<div class="bc-empty">No uploaded replays found.</div>';
    return;
  }

  const cards = list.map(rp => bcReplayCard(rp, localById[rp.id] || null)).join('');

  // Soft-linked local-only uploads (not returned by BC API but in our DB)
  const localCards = localOnlyUploads.map(u => `
    <div class="bc-card">
      <div class="bc-card-top">
        <span class="bc-card-map">${esc(u.replay_name)}</span>
        <span class="bc-local-badge">Local</span>
      </div>
      <div class="bc-card-meta">
        <span>${formatDate(u.uploaded_at)}</span>
        <a href="${esc(u.bc_url)}" target="_blank" rel="noopener" class="bc-link">↗ View on BC</a>
      </div>
    </div>`).join('');

  el.innerHTML = cards + localCards;
}

function bcReplayCard(rp, localUpload) {
  const mapName   = rp.map_name || friendlyArena(rp.map_code) || '—';
  const playlist  = rp.playlist_name || '';
  const date      = rp.date ? formatDate(rp.date) : '—';
  const blueGoals = rp.blue?.goals ?? null;
  const orgGoals  = rp.orange?.goals ?? null;
  const score     = blueGoals !== null
    ? `<span class="bc-score"><span style="color:var(--rl-blue)">${blueGoals}</span> — <span style="color:var(--rl-orange)">${orgGoals}</span></span>`
    : '';
  const link = `https://ballchasing.com/replay/${rp.id}`;
  const localBadge = localUpload ? `<span class="bc-local-badge" title="${esc(localUpload.replay_name)}">Local</span>` : '';

  return `
    <div class="bc-card">
      <div class="bc-card-top">
        <span class="bc-card-map">${esc(mapName)}</span>
        ${playlist ? `<span class="bc-card-playlist">${esc(playlist)}</span>` : ''}
        ${localBadge}
      </div>
      <div class="bc-card-meta">
        ${score}
        <span>${date}</span>
        <a href="${esc(link)}" target="_blank" rel="noopener" class="bc-link">↗ View</a>
      </div>
    </div>`;
}

function renderBCNotUploaded(files, uploads) {
  const el = document.getElementById('bc-not-uploaded');
  const notUploaded = files.filter(f => !uploads[f.name]);

  if (!notUploaded.length) {
    el.innerHTML = '<div class="bc-empty">All local replays have been uploaded.</div>';
    return;
  }

  el.innerHTML = notUploaded.slice(0, 100).map(f => `
    <div class="bc-replay-row" data-replay="${esc(f.name)}">
      <div class="bc-replay-name" title="${esc(f.name)}">${esc(f.name)}</div>
      <div class="bc-replay-meta">${formatBytes(f.size_bytes)} · ${f.modified_at ? formatDate(f.modified_at) : '—'}</div>
      <div class="bc-replay-actions">
        <button class="bc-upload-btn" data-name="${esc(f.name)}">↑ Upload</button>
      </div>
    </div>`).join('');

  el.querySelectorAll('.bc-upload-btn').forEach(btn => {
    btn.addEventListener('click', () => uploadReplay(btn.dataset.name, btn));
  });
}

async function uploadReplay(replayName, btn) {
  btn.disabled = true;
  btn.textContent = '…';
  try {
    const r = await fetch('/api/ballchasing/upload', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ replay_name: replayName, visibility: 'public' }),
    });
    const j = await r.json().catch(() => ({}));
    if (r.ok && !j.error) {
      // Move the row out of "not uploaded" and reload BC tab
      btn.closest('.bc-replay-row')?.remove();
      // Add a soft link in the uploaded section immediately
      const bcId  = j.id || '';
      const bcURL = j.link || (bcId ? `https://ballchasing.com/replay/${bcId}` : '');
      const uploadedEl = document.getElementById('bc-uploaded');
      if (uploadedEl && bcURL) {
        uploadedEl.insertAdjacentHTML('afterbegin', `
          <div class="bc-card">
            <div class="bc-card-top">
              <span class="bc-card-map">${esc(replayName)}</span>
              <span class="bc-local-badge">Local</span>
            </div>
            <div class="bc-card-meta">
              <span>Just uploaded</span>
              <a href="${esc(bcURL)}" target="_blank" rel="noopener" class="bc-link">↗ View</a>
            </div>
          </div>`);
      }
    } else {
      btn.disabled = false;
      btn.textContent = '↑ Retry';
      btn.title = j.error || `HTTP ${r.status}`;
    }
  } catch (e) {
    btn.disabled = false;
    btn.textContent = '↑ Retry';
    btn.title = e.message;
  }
}

async function loadBCGroups() {
  const el = document.getElementById('bc-groups');
  el.innerHTML = '<div class="bc-loading">Loading…</div>';
  try {
    const r = await fetch('/api/ballchasing/groups');
    if (!r.ok) {
      const j = await r.json().catch(() => ({}));
      el.innerHTML = `<div class="bc-empty">${esc(j.error || 'Not available')}</div>`;
      return;
    }
    const data = await r.json();
    const list = data.list || [];
    if (!list.length) {
      el.innerHTML = '<div class="bc-empty">No groups found.</div>';
      return;
    }
    el.innerHTML = list.map(g => {
      const link  = g.id ? `https://ballchasing.com/group/${g.id}` : '';
      const count = g.direct_replays != null ? `${g.direct_replays} replays` : '';
      return `
        <div class="bc-card">
          <div class="bc-card-top">
            <span class="bc-card-map">${esc(g.name || g.id || '—')}</span>
          </div>
          <div class="bc-card-meta">
            ${count ? `<span>${count}</span>` : ''}
            ${g.created ? `<span>${formatDate(g.created)}</span>` : ''}
            ${link ? `<a href="${esc(link)}" target="_blank" rel="noopener" class="bc-link">↗ View</a>` : ''}
          </div>
        </div>`;
    }).join('');
  } catch (e) {
    el.innerHTML = `<div class="bc-empty">${esc(e.message)}</div>`;
  }
}

function showMsg(id, text, ok) {
  const el = document.getElementById(id);
  el.textContent = text;
  el.className   = 'msg ' + (ok ? 'ok' : 'err');
  el.classList.remove('hidden');
  setTimeout(() => el.classList.add('hidden'), 5000);
}

// --- Helpers ---
function esc(s) {
  return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function formatDate(d) {
  if (!d) return '—';
  return new Date(d).toLocaleString();
}

function formatDuration(secs) {
  if (secs == null || isNaN(secs)) return '—';
  const m = Math.floor(secs / 60);
  const s = (secs % 60).toFixed(0).padStart(2, '0');
  return `${m}:${s}`;
}

function formatBytes(n) {
  const v = Number(n || 0);
  if (v < 1024)         return `${v} B`;
  if (v < 1024 * 1024)  return `${(v / 1024).toFixed(1)} KB`;
  return `${(v / (1024 * 1024)).toFixed(1)} MB`;
}
