'use strict';

// --- Arena name mapping ---
// Keys are the internal RL/BC map codes (lowercased). Source: https://ballchasing.com/api/maps
const ARENA_NAMES = {
  // Champions Field
  cs_p:                    'Champions Field',
  cs_day_p:                'Champions Field (Day)',
  cs_hw_p:                 'Rivals Arena',
  swoosh_p:                'Champions Field (Nike FC)',
  bb_p:                    'Champions Field (NFL)',

  // Mannfield
  eurostadium_p:           'Mannfield',
  eurostadium_night_p:     'Mannfield (Night)',
  eurostadium_rainy_p:     'Mannfield (Stormy)',
  eurostadium_dusk_p:      'Mannfield (Dusk)',
  eurostadium_snownight_p: 'Mannfield (Snowy)',

  // Starbase ARC
  arc_p:                   'Starbase ARC',
  arc_standard_p:          'Starbase ARC (Standard)',
  arc_darc_p:              'Starbase ARC (Aftermath)',

  // DFH Stadium
  stadium_p:               'DFH Stadium',
  stadium_day_p:           'DFH Stadium (Day)',
  stadium_winter_p:        'DFH Stadium (Snowy)',
  stadium_foggy_p:         'DFH Stadium (Stormy)',
  stadium_race_day_p:      'DFH Stadium (Circuit)',

  // Urban Central
  trainstation_p:          'Urban Central',
  trainstation_dawn_p:     'Urban Central (Dawn)',
  trainstation_night_p:    'Urban Central (Night)',
  trainstation_spooky_p:   'Urban Central (Spooky)',
  haunted_trainstation_p:  'Urban Central (Haunted)',

  // Beckwith Park
  park_p:                  'Beckwith Park',
  park_night_p:            'Beckwith Park (Midnight)',
  park_bman_p:             'Beckwith Park (Night)',
  park_rainy_p:            'Beckwith Park (Stormy)',
  park_snowy_p:            'Beckwith Park (Snowy)',

  // Wasteland
  wasteland_p:             'Wasteland',
  wasteland_s_p:           'Wasteland (Standard)',
  wasteland_night_p:       'Wasteland (Night)',
  wasteland_night_s_p:     'Wasteland (Standard, Night)',
  wasteland_grs_p:         'Wasteland (Pitched)',

  // Neo Tokyo
  neotokyo_p:              'Neo Tokyo',
  neotokyo_standard_p:     'Neo Tokyo (Standard)',
  neotokyo_arcade_p:       'Neo Tokyo (Arcade)',
  neotokyo_hax_p:          'Neo Tokyo (Hacked)',
  neotokyo_toon_p:         'Neo Tokyo (Comic)',

  // Utopia Coliseum
  utopiastadium_p:         'Utopia Coliseum',
  utopiastadium_dusk_p:    'Utopia Coliseum (Dusk)',
  utopiastadium_snow_p:    'Utopia Coliseum (Snowy)',
  utopiastadium_lux_p:     'Utopia Coliseum (Gilded)',

  // AquaDome
  underwater_p:            'Aquadome',
  underwater_grs_p:        'AquaDome (Salty Shallows)',

  // Salty Shores
  beach_p:                 'Salty Shores',
  beach_night_p:           'Salty Shores (Night)',
  beach_night_grs_p:       'Salty Shores (Salty Fest)',
  beachvolley:             'Salty Shores (Volley)',

  // Forbidden Temple
  chn_stadium_p:           'Forbidden Temple',
  chn_stadium_day_p:       'Forbidden Temple (Day)',
  fni_stadium_p:           'Forbidden Temple (Fire & Ice)',

  // Farmstead
  farm_p:                  'Farmstead',
  farm_night_p:            'Farmstead (Night)',
  farm_grs_p:              'Farmstead (Pitched)',
  farm_hw_p:               'Farmstead (Spooky)',
  farm_upsidedown_p:       'Farmstead (The Upside Down)',

  // Throwback Stadium
  throwbackstadium_p:      'Throwback Stadium',
  throwbackhockey_p:       'Throwback Stadium (Snowy)',

  // Neon Fields
  music_p:                 'Neon Fields',

  // Dunk House / Hoops
  hoopsstadium_p:          'Dunk House',
  hoopsstreet_p:           'The Block (Dusk)',

  // Deadeye Canyon
  outlaw_p:                'Deadeye Canyon',
  outlaw_oasis_p:          'Deadeye Canyon (Oasis)',

  // Knockout arenas
  ko_calavera_p:           'Calavera',
  ko_carbon_p:             'Carbon',
  ko_quadron_p:            'Quadron',

  // Labs
  labs_basin_p:            'Basin',
  labs_circlepillars_p:    'Pillars',
  labs_corridor_p:         'Corridor',
  labs_cosmic_p:           'Cosmic',
  labs_cosmic_v4_p:        'Cosmic',
  labs_doublegoal_p:       'Double Goal',
  labs_doublegoal_v2_p:    'Double Goal',
  labs_galleon_p:          'Galleon',
  labs_galleon_mast_p:     'Galleon Retro',
  labs_holyfield_p:        'Loophole',
  labs_octagon_p:          'Octagon',
  labs_octagon_02_p:       'Octagon',
  labs_pillarglass_p:      'Hourglass',
  labs_pillarheat_p:       'Barricade',
  labs_pillarwings_p:      'Colossus',
  labs_underpass_p:        'Underpass',
  labs_underpass_v0_p:     'Underpass',
  labs_utopia_p:           'Utopia Retro',

  // Other
  shattershot_p:           'Core 707',
  ff_dusk_p:               'Estadio Vida (Dusk)',
  street_p:                'Sovereign Heights (Dusk)',
  woods_p:                 'Drift Woods',
  woods_night_p:           'Drift Woods (Night)',
};

function friendlyArena(code) {
  if (!code) return '—';
  const key = code.toLowerCase().trim();
  return ARENA_NAMES[key] ||
    key.replace(/_p$/, '').replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

const PLAYLIST_NAMES = {
  1:  'Casual 1v1',  2:  'Casual 2v2',  3:  'Casual 3v3',
  4:  'Custom',      6:  'Private',      10: 'Ranked 1v1',
  11: 'Ranked 2v2',  13: 'Ranked 3v3',  14: 'Solo 3v3',
  22: 'Tournament',  27: 'Rocket Labs',  28: 'Rumble',
  29: 'Dropshot',    30: 'Hoops',        31: 'Snow Day',
  34: 'Casual Chaos',35: 'Gridiron',     41: 'Heatseeker',
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

function chip(cls, label, active) {
  const colorCls = cls ? ` ${cls}` : '';
  const dimCls   = active ? '' : ' dim';
  return `<span class="status-chip${colorCls}${dimCls}">${label}</span>`;
}

// --- Tracker.gg rank cache ---
const trackerCache = new Map();

function platformFromId(primaryId) {
  if (!primaryId) return '';
  const sep = String(primaryId).search(/[|:_]/);
  if (sep < 1) return '';
  return primaryId.slice(0, sep).toLowerCase();
}

const _PLAT_NORM = { ps4: 'psn', ps5: 'psn', playstation: 'psn', xboxone: 'xbl', xbox: 'xbl', epicgames: 'epic', nintendo: 'switch' };
function normPlatform(plat) { return _PLAT_NORM[plat] || plat; }

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

function timeAgo(isoStr) {
  if (!isoStr) return '';
  const secs = Math.round((Date.now() - new Date(isoStr).getTime()) / 1000);
  if (secs < 60)   return `${secs}s ago`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
  return `${Math.floor(secs / 3600)}h ago`;
}

function parseRankData(ranks) {
  if (!ranks || !ranks.length) return null;
  const out = {};
  for (const r of ranks) {
    if (!r.PlaylistID) continue;
    out[r.PlaylistID] = {
      playlistName: r.PlaylistName || '',
      tierName:     r.TierName    || '',
      icon:         r.IconURL     || '',
      rating:       r.MMR ? String(Math.round(r.MMR)) : '',
    };
  }
  return Object.keys(out).length ? out : null;
}

function isBot(primaryId) {
  if (!primaryId) return true;
  const plat = platformFromId(primaryId);
  return !plat || plat === 'unknown' || plat === 'bot';
}

function isMaskedName(name) {
  return typeof name === 'string' && name.length > 0 && /^\*+$/.test(name);
}

// --- Tracker fetch queue ---
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
      const ranks    = parseRankData(json?.ranks);
      const fetchedAt = json?.fetched_at || null;
      trackerCache.set(id, { profile: null, ranks, status, fetchedAt });
      onTrackerDataArrived();
    })
    .catch(e => {
      trackerCache.set(id, { profile: null, ranks: null, status: `Error: ${e.message}`, fetchedAt: null });
      onTrackerDataArrived();
    })
    .finally(() => setTimeout(_drainTrkQueue, 2500));
}

function prefetchTrackerRanks(players) {
  let enqueued = false;
  for (const p of players) {
    const id = p.PrimaryId;
    if (!id || trackerCache.has(id)) continue;
    if (isBot(id)) continue;
    const plat = platformFromId(id);
    if (plat !== 'steam' && isMaskedName(p.Name)) {
      trackerCache.set(id, { profile: null, ranks: null, status: 'masked', fetchedAt: null });
      continue;
    }
    trackerCache.set(id, undefined);
    _trkQueue.push({ id, plat, name: p.Name });
    enqueued = true;
  }
  if (enqueued && !_trkDraining) setTimeout(_drainTrkQueue, 2000);
}

// Written by history/view.js; called here when tracker data arrives.
let _historyDetailReRender = null;
// Written by ranks/view.js; called here when tracker data arrives.
let _ranksReRender = null;

function onTrackerDataArrived() {
  if (_historyDetailReRender) _historyDetailReRender();
  if (_ranksReRender) _ranksReRender();
}

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

// --- Widget registry ---
const widgetRegistry = Object.create(null);
window.widgetRegistry = widgetRegistry;
window.registerWidget = function(def) {
  if (!def || typeof def !== 'object') {
    console.warn('[widgets] rejected widget definition: expected an object');
    return false;
  }

  const id = typeof def.id === 'string' ? def.id.trim() : '';
  if (!id) {
    console.warn('[widgets] rejected widget definition: missing or invalid "id"');
    return false;
  }

  const title = typeof def.title === 'string' ? def.title.trim() : '';
  if (!title) {
    console.warn('[widgets] rejected widget "' + id + '": missing or invalid "title"');
    return false;
  }

  if (typeof def.factory !== 'function') {
    console.warn('[widgets] rejected widget "' + id + '": missing or invalid "factory"');
    return false;
  }

  if (id in widgetRegistry) {
    console.warn('[widgets] duplicate id:', id);
    return false;
  }

  widgetRegistry[id] = def;
  return true;
};

// --- WebSocket ---
const dot = document.getElementById('status-dot');

function connectWS() {
  const ws = new WebSocket(`ws://${location.host}/ws`);

  ws.onmessage = e => {
    const msg = JSON.parse(e.data);
    if (msg.Event === '_Status') {
      dot.className = 'dot ' + (msg.Data.connected ? 'connected' : 'disconnected');
      dot.title = msg.Data.connected ? 'Connected to Rocket League' : 'Waiting for Rocket League';
      if (!msg.Data.connected && typeof clearLive === 'function') clearLive();
      return;
    }
    if ((msg.Event === 'MatchCreated' || msg.Event === 'MatchInitialized') && typeof handleSessionMatchStart === 'function') handleSessionMatchStart();
    if (typeof handleDebugAssistantEvent === 'function') handleDebugAssistantEvent(msg);
    if (msg.Event === 'UpdateState'    && typeof handleUpdateState  === 'function') handleUpdateState(msg.Data);
    if (msg.Event === 'UpdateState'    && typeof handleRanksUpdate  === 'function') handleRanksUpdate(msg.Data);
    if (msg.Event === 'UpdateState'    && typeof handleSessionUpdate === 'function') handleSessionUpdate(msg.Data);
    if (msg.Event === 'GoalScored'     && typeof flashGoal          === 'function') flashGoal(msg.Data);
    if ((msg.Event === 'MatchEnded' || msg.Event === 'MatchDestroyed') && typeof refreshPostMatchViews === 'function') refreshPostMatchViews();
    if (msg.Event === 'MatchDestroyed' && typeof clearLive          === 'function') clearLive();
    if (msg.Event === 'MatchDestroyed' && typeof handleRanksClear   === 'function') handleRanksClear();
    if (msg.Event === 'MatchDestroyed' && typeof clearSessionLive   === 'function') clearSessionLive();
    if (msg.Event === 'bc:uploaded'             && typeof handleBCUploaded          === 'function') handleBCUploaded(msg.Data);
    if (msg.Event === 'bc:save-replay-reminder' && typeof handleBCSaveReplayReminder === 'function') handleBCSaveReplayReminder();
    if (msg.Event === 'MatchDestroyed'          && typeof refreshBCMatches           === 'function') refreshBCMatches();
  };

  ws.onclose = () => {
    dot.className = 'dot disconnected';
    setTimeout(connectWS, 3000);
  };
}

let _postMatchRefreshTimer = null;

function refreshPostMatchViews() {
  clearTimeout(_postMatchRefreshTimer);
  _postMatchRefreshTimer = setTimeout(() => {
    if (typeof refreshSession === 'function') refreshSession();
    if (typeof loadSessionHistory === 'function') loadSessionHistory();
    if (typeof loadHistory === 'function') loadHistory();
  }, 150);
}

// --- Navigation ---
let _activeViewName = null;
const _viewScrollPositions = {};

function rememberActiveViewScroll() {
  if (!_activeViewName) return;
  _viewScrollPositions[_activeViewName] = window.scrollY || document.documentElement.scrollTop || 0;
}

function restoreViewScroll(name) {
  const y = _viewScrollPositions[name] || 0;
  window.scrollTo({ top: y, left: 0, behavior: 'auto' });
}

function restoreViewScrollSoon(name) {
  requestAnimationFrame(() => restoreViewScroll(name));
  setTimeout(() => restoreViewScroll(name), 120);
  setTimeout(() => restoreViewScroll(name), 450);
}

function runViewLoader(name, loader) {
  try {
    const result = loader();
    if (result && typeof result.finally === 'function') {
      result.finally(() => restoreViewScrollSoon(name));
    }
  } catch (err) {
    console.error(err);
  }
}

function showView(name) {
  rememberActiveViewScroll();
  document.querySelectorAll('.view').forEach(v => v.classList.toggle('active', v.id === 'view-' + name));
  document.querySelectorAll('.nav-btn').forEach(b => b.classList.toggle('active', b.dataset.view === name));
  document.querySelector('main')?.classList.toggle('dash-active', name === 'dashboard');
  _activeViewName = name;
  window.oofActiveViewName = name;
  if (name === 'history'   && typeof loadHistory      === 'function') runViewLoader(name, loadHistory);
  if (name === 'settings') runViewLoader(name, loadSettings);
  if (name === 'bc'        && typeof loadBC            === 'function') runViewLoader(name, loadBC);
  if (name === 'ranks'     && typeof refreshRanks      === 'function') runViewLoader(name, refreshRanks);
  if (name === 'session'   && typeof refreshSession    === 'function') runViewLoader(name, refreshSession);
  if (name === 'dashboard' && typeof loadDashboard     === 'function') runViewLoader(name, loadDashboard);
  if (name !== 'history') _historyDetailReRender = null;
  if (name !== 'ranks')   _ranksReRender = null;
  restoreViewScrollSoon(name);
}

window.addEventListener('scroll', rememberActiveViewScroll, { passive: true });

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

async function loadSettings() {
  const [cfg, schema] = await Promise.all([
    fetch('/api/config').then(r => r.json()),
    fetch('/api/settings/schema').then(r => r.json()).catch(() => []),
  ]);

  document.getElementById('cfg-ttl').value = cfg.tracker_cache_ttl_minutes ?? 10;
  const hk = cfg.overlay_hotkey || 'F9';
  _hotkeyBtn.textContent   = hk;
  _hotkeyBtn.dataset.value = hk;
  document.getElementById('cfg-hold-mode').checked = !!cfg.overlay_hold_mode;
  const opacityAlpha = Math.round((cfg.overlay_opacity ?? 1.0) * 255);
  document.getElementById('cfg-overlay-opacity').value = opacityAlpha;
  document.getElementById('cfg-overlay-opacity-pct').textContent = Math.round(opacityAlpha / 2.55) + '%';

  // App URL display
  const urlEl = document.getElementById('cfg-app-url-display');
  const urlLink = document.getElementById('cfg-app-url-link');
  if (urlEl) urlEl.textContent = window.location.origin;
  if (urlLink) { urlLink.href = window.location.origin; }

  renderPluginAccordion(schema, cfg);

  // Data dir display
  fetch('/api/data-dir').then(r => r.json()).then(d => {
    const el = document.getElementById('cfg-data-dir-display');
    if (el) el.textContent = d.path || '';
  }).catch(() => {});

  try {
    const ini = await fetch('/api/config/ini').then(r => r.json());
    const enabled = ini.PacketSendRate > 0;
    document.getElementById('ini-enabled').checked = enabled;
    document.getElementById('ini-rate').value = enabled ? ini.PacketSendRate : 60;
    document.getElementById('ini-port').value = ini.Port || 49123;
    if (ini.note) showMsg('ini-msg', ini.note, !ini.error);
  } catch (_) {
    showMsg('ini-msg', 'Could not read INI settings.', false);
  }
}

let _disabledPlugins = [];

function renderSettingRow(s, cfg) {
  const rawVal = s.key.split('.').reduce((o, k) => o?.[k], cfg);
  const val = rawVal ?? s.default ?? '';
  const descHtml = s.description
    ? `<span class="help-icon">?<span class="help-tip">${esc(s.description)}</span></span>`
    : '';
  if (s.type === 'checkbox') {
    const checked = val === true || val === 'true' || val === '1' ? 'checked' : '';
    return `<div class="settings-row">
      <label class="settings-label flex items-center gap-2 cursor-pointer">
        <input type="checkbox" id="pfield-${esc(s.key)}" class="accent-rl-blue" ${checked}>
        ${esc(s.label)}${descHtml}
      </label>
    </div>`;
  }
  const inputType = s.type === 'password' ? 'password' : s.type === 'number' ? 'number' : 'text';
  return `<div class="settings-row">
    <span class="settings-label">${esc(s.label)}${descHtml}</span>
    <input type="${inputType}" id="pfield-${esc(s.key)}" value="${esc(String(val))}"
           placeholder="${esc(s.placeholder || '')}" class="settings-input" style="width:200px" autocomplete="off">
  </div>`;
}

function renderCoreSettings(blob, cfg) {
  const container = document.getElementById('advanced-core-settings');
  if (!container) return;
  container.innerHTML = '';
  if (!blob || !(blob.settings || []).length) return;

  const msgId = 'plugin-msg-core';
  const fieldRows = blob.settings.map(s => renderSettingRow(s, cfg)).join('');

  const panel = document.createElement('div');
  panel.className = 'settings-panel';
  panel.innerHTML = `
    <div class="settings-panel-title">Data Capture</div>
    <p class="text-xs text-gray-500 mb-3">These options can generate significant amounts of data. Only enable them if you have a specific need.</p>
    ${fieldRows}
    <div class="settings-footer">
      <button class="btn-action" id="core-save-btn">Save</button>
      <div id="${msgId}" class="msg hidden"></div>
    </div>`;

  panel.querySelector('#core-save-btn').addEventListener('click', async () => {
    const values = {};
    for (const s of blob.settings) {
      const el = document.getElementById(`pfield-${s.key}`);
      if (el) values[s.key] = s.type === 'checkbox' ? (el.checked ? 'true' : 'false') : el.value.trim();
    }
    const res = await fetch('/api/settings', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(values),
    });
    showMsg(msgId, res.ok ? 'Saved!' : 'Error saving settings.', res.ok);
  });

  container.appendChild(panel);
}

function renderPluginAccordion(blobs, cfg) {
  _disabledPlugins = cfg.disabled_plugins || [];

  const coreBlob = blobs.find(b => b.plugin_id === 'core');
  renderCoreSettings(coreBlob, cfg);

  const container = document.getElementById('plugin-settings-container');
  container.innerHTML = '';

  for (const blob of blobs) {
    if (blob.plugin_id === 'core') continue;

    const msgId    = `plugin-msg-${blob.plugin_id}`;
    const hasFields = (blob.settings || []).length > 0;
    const isEnabled = blob.enabled;

    const normalSettings = (blob.settings || []).filter(s => !s.developer);
    const devSettings    = (blob.settings || []).filter(s => s.developer);

    const fieldRows = normalSettings.map(s => renderSettingRow(s, cfg)).join('');
    const devRows   = devSettings.length > 0
      ? `<details style="margin-top:6px">
          <summary style="cursor:pointer;font-size:0.8rem;opacity:0.55;padding:4px 18px">Developer</summary>
          ${devSettings.map(s => renderSettingRow(s, cfg)).join('')}
        </details>`
      : '';

    const item = document.createElement('div');
    item.className = 'plugin-item';
    item.innerHTML = `<div class="plugin-item-header">
           <span class="plugin-item-dot ${isEnabled ? 'on' : 'off'}"></span>
           <span class="plugin-item-name${isEnabled ? '' : ' disabled'}">${esc(blob.title)}</span>
           <span class="plugin-item-arrow" aria-hidden="true">›</span>
         </div>
         <div class="plugin-item-body" style="display:none">
           <div class="settings-row">
             <label class="settings-label flex items-center gap-2 cursor-pointer">
               <input type="checkbox" class="plugin-enabled-cb accent-rl-blue" ${isEnabled ? 'checked' : ''}>
               Enable ${esc(blob.title)}
             </label>
           </div>
           ${fieldRows}${devRows}
           ${hasFields ? `<div class="settings-footer">
             <button class="btn-action plugin-save-btn">Save</button>
             <div id="${msgId}" class="msg hidden"></div>
           </div>` : `<div id="${msgId}" class="msg hidden" style="padding:6px 18px 10px"></div>`}
         </div>`;

    // Expand / collapse on header click
    const header = item.querySelector('.plugin-item-header');
    const body   = item.querySelector('.plugin-item-body');
    const arrow  = item.querySelector('.plugin-item-arrow');
    header.addEventListener('click', () => {
      const nowOpen = body.style.display === 'none';
      body.style.display = nowOpen ? '' : 'none';
      arrow.classList.toggle('open', nowOpen);
    });

    // Enable / disable — immediately update nav button without reload
    const cb     = item.querySelector('.plugin-enabled-cb');
    const dot    = item.querySelector('.plugin-item-dot');
    const nameEl = item.querySelector('.plugin-item-name');
    cb.addEventListener('change', async () => {
      const enabled = cb.checked;
      _disabledPlugins = enabled
        ? _disabledPlugins.filter(id => id !== blob.plugin_id)
        : [...new Set([..._disabledPlugins, blob.plugin_id])];

      dot.className = `plugin-item-dot ${enabled ? 'on' : 'off'}`;
      nameEl.className = `plugin-item-name${enabled ? '' : ' disabled'}`;

      if (blob.nav_tab_id) {
        const navBtn = document.querySelector(`.nav-btn[data-view="${blob.nav_tab_id}"]`);
        if (navBtn) navBtn.style.display = enabled ? '' : 'none';
        if (!enabled) {
          const active = document.querySelector('.view.active');
          if (active && active.id === 'view-' + blob.nav_tab_id) showView('settings');
        }
      }

      await fetch('/api/config', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({ disabled_plugins: _disabledPlugins }),
      });
      showMsg(msgId, enabled ? 'Enabled.' : 'Disabled.', true);
    });

    // Save plugin-specific settings
    if (hasFields) {
      item.querySelector('.plugin-save-btn').addEventListener('click', async () => {
        const values = {};
        for (const s of blob.settings) {
          const el = document.getElementById(`pfield-${s.key}`);
          if (el) values[s.key] = s.type === 'checkbox' ? (el.checked ? 'true' : 'false') : el.value.trim();
        }
        const res = await fetch('/api/settings', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify(values),
        });
        showMsg(msgId, res.ok ? 'Saved!' : 'Error saving settings.', res.ok);
      });
    }

    container.appendChild(item);
  }
}

document.getElementById('save-cfg').addEventListener('click', async () => {
  const opacityAlpha = parseInt(document.getElementById('cfg-overlay-opacity').value);
  const cfg = {
    tracker_cache_ttl_minutes: Math.max(2, parseInt(document.getElementById('cfg-ttl').value) || 5),
    overlay_hotkey:    _hotkeyBtn.dataset.value || _hotkeyBtn.textContent || 'F9',
    overlay_hold_mode: document.getElementById('cfg-hold-mode').checked,
    overlay_opacity:   opacityAlpha / 255,
  };
  const res = await fetch('/api/config', { method: 'POST', body: JSON.stringify(cfg) });
  showMsg('cfg-msg', res.ok ? 'Saved!' : 'Error saving config.', res.ok);
  if (res.ok) await loadSettings();
});

document.getElementById('cfg-data-dir-open').addEventListener('click', () => {
  fetch('/api/db/open-folder').catch(() => {});
});

document.getElementById('cfg-overlay-opacity').addEventListener('input', e => {
  const alpha = parseInt(e.target.value);
  document.getElementById('cfg-overlay-opacity-pct').textContent = Math.round(alpha / 2.55) + '%';
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

// --- Helpers ---
function showMsg(id, text, ok) {
  const el = document.getElementById(id);
  el.textContent = text;
  el.className   = 'msg ' + (ok ? 'ok' : 'err');
  el.classList.remove('hidden');
  setTimeout(() => el.classList.add('hidden'), 5000);
}

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

// --- App init ---

function loadScript(src) {
  return new Promise((resolve, reject) => {
    const s = document.createElement('script');
    s.src = src;
    s.onload  = resolve;
    s.onerror = reject;
    document.head.appendChild(s);
  });
}

function buildNav(enabledTabs, allSchema) {
  const nav = document.getElementById('plugin-nav');
  const enabledIds = new Set(enabledTabs.map(t => t.id));
  const pluginBtns = allSchema
    .filter(b => b.nav_tab_id)
    .map(b => {
      const visible = enabledIds.has(b.nav_tab_id);
      return `<button class="nav-btn" data-view="${esc(b.nav_tab_id)}"${visible ? '' : ' style="display:none"'}>${esc(b.title)}</button>`;
    })
    .join('');
  nav.innerHTML = pluginBtns + '<button class="nav-btn" data-view="settings">Settings</button>';
  nav.querySelectorAll('.nav-btn').forEach(b => b.addEventListener('click', () => showView(b.dataset.view)));
}

async function injectPluginViews(allSchema) {
  const container = document.getElementById('plugin-views');
  const ids = allSchema.filter(b => b.nav_tab_id).map(b => b.nav_tab_id);
  const htmls = await Promise.all(ids.map(id =>
    fetch(`/api/plugins/${id}/view`)
      .then(r => r.ok ? r.text() : '')
      .catch(() => '')
  ));
  for (let i = 0; i < ids.length; i++) {
    const id = ids[i];
    const section = document.createElement('section');
    section.id = 'view-' + id;
    section.className = 'view';
    section.innerHTML = htmls[i];
    container.appendChild(section);
    try { await loadScript(`/plugins/${id}/view.js`); } catch (_) {}
    const init = window[`pluginInit_${id}`];
    if (typeof init === 'function') init();
  }
}

async function initApp() {
  const [tabs, schema] = await Promise.all([
    fetch('/api/nav').then(r => r.json()).catch(() => []),
    fetch('/api/settings/schema').then(r => r.json()).catch(() => []),
  ]);
  buildNav(tabs, schema);
  await injectPluginViews(schema);
  showView(tabs[0]?.id || 'settings');
  connectWS();
}

// Overlay-mode init: runs only in the overlay window (url has ?overlay=1)
if (new URLSearchParams(location.search).has('overlay')) {
  document.body.classList.add('overlay-mode');
  document.getElementById('overlay-drag-bar').addEventListener('mousedown', e => {
    if (e.target.tagName === 'INPUT') return;
    if (e.button === 0) window.overlayStartDrag?.();
  });
  document.getElementById('overlay-resize-grip').addEventListener('mousedown', e => {
    if (e.button === 0) window.overlayStartResize?.();
  });
  const opacitySlider = document.getElementById('overlay-opacity-slider');
  const opacityPct    = document.getElementById('overlay-opacity-pct');
  opacitySlider.addEventListener('input', () => {
    const alpha = parseInt(opacitySlider.value);
    opacityPct.textContent = Math.round(alpha / 2.55) + '%';
    window.overlaySetOpacity?.(alpha);
  });
  fetch('/api/config').then(r => r.json()).then(cfg => {
    const alpha = Math.round((cfg.overlay_opacity ?? 1.0) * 255);
    opacitySlider.value = alpha;
    opacityPct.textContent = Math.round(alpha / 2.55) + '%';
  }).catch(() => {});
}

initApp();

// Shared match detail renderer used by both history and session plugins.
// data = { players, goals, events }; panel = DOM element to render into; activeMatchId = guard variable.
window.renderMatchDetailPanel = function(data, panel, activeMatchId, matchID) {
  if (activeMatchId !== matchID) return;

  const players    = data.players || [];
  const goals      = data.goals   || [];
  const events     = data.events  || [];
  const matchStart = data.match?.StartedAt ? new Date(data.match.StartedAt).getTime() : null;

  function matchRelTime(occurredAt) {
    if (!matchStart || !occurredAt) return '';
    const secs = Math.max(0, Math.round((new Date(occurredAt).getTime() - matchStart) / 1000));
    const m = Math.floor(secs / 60);
    const s = String(secs % 60).padStart(2, '0');
    return `+${m}:${s}`;
  }

  players.forEach(p => { if (!p.PrimaryId) p.PrimaryId = p.PrimaryID; });
  const rankablePlayers = players.filter(p => !isBot(p.PrimaryId));
  prefetchTrackerRanks(rankablePlayers);

  const blue   = players.filter(p => p.TeamNum === 0);
  const orange = players.filter(p => p.TeamNum === 1);

  const nameTeam = new Map();
  blue.forEach(p => nameTeam.set(p.Name, 'blue'));
  orange.forEach(p => nameTeam.set(p.Name, 'orange'));

  function colorName(name) {
    if (!name) return '<span style="color:var(--muted)">—</span>';
    const t = nameTeam.get(name);
    const style = t === 'blue' ? 'color:var(--rl-blue)' : t === 'orange' ? 'color:var(--rl-orange)' : '';
    return style ? `<span style="${style}">${esc(name)}</span>` : esc(name);
  }

  const EVENT_ICON = {
    Goal: '⚽', OwnGoal: '😬', Save: '🛡️', EpicSave: '✨', Assist: '🤝',
    Demolish: '💥', Shot: '🎯',
  };
  const EVENT_LABEL = {
    Shot: 'Shot on goal', EpicSave: 'Epic save', OwnGoal: 'Own goal', Demolish: 'Demo',
  };

  const statsRows = (list, cls) => list.map(p => `
    <tr class="${cls}">
      <td>
        <div class="font-medium">${esc(p.Name)}${isBot(p.PrimaryId) ? ' <span class="player-platform-badge">BOT</span>' : ''}</div>
        ${isBot(p.PrimaryId) ? '' : trackerRankHTML(p.PrimaryId)}
      </td>
      <td>${p.Goals}</td><td>${p.Assists}</td><td>${p.Saves}</td>
      <td>${p.Shots}</td><td>${p.Demos}</td>
      <td>${p.Touches ?? 0}</td>
      <td>${p.Score}</td>
    </tr>`).join('');

  const goalsHTML = goals.length
    ? goals.map(g => `
        <tr>
          <td>${colorName(g.ScorerName)}</td>
          <td>${colorName(g.AssisterName)}</td>
          <td>${g.GoalSpeed != null ? g.GoalSpeed.toFixed(1) : '—'}</td>
          <td>${formatDuration(g.GoalTime)}</td>
        </tr>`).join('')
    : '<tr><td colspan="4" style="color:var(--muted)">No goals recorded</td></tr>';

  const eventsHTML = events.length
    ? events.map(e => {
        const icon  = EVENT_ICON[e.event_type] || '•';
        const actor = colorName(e.player_name);
        const tgt   = e.target_name ? ` → ${colorName(e.target_name)}` : '';
        const label = EVENT_LABEL[e.event_type] || e.event_type;
        const t     = matchRelTime(e.occurred_at);
        return `<tr>
          <td style="width:28px;text-align:center">${icon}</td>
          <td style="color:var(--muted);font-size:11px">${esc(label)}</td>
          <td>${actor}${tgt}</td>
          <td style="color:var(--muted);font-size:11px;text-align:right;white-space:nowrap">${t}</td>
        </tr>`;
      }).join('')
    : `<tr><td colspan="4" style="color:var(--muted)">No events recorded</td></tr>`;

  const tbodyId = `detail-stats-${matchID}`;

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
      ${events.length ? `
      <h3 class="detail-section-label" style="margin-top:16px">Events</h3>
      <table class="detail-table">
        <thead><tr><th></th><th></th><th>Player</th><th style="text-align:right">Time</th></tr></thead>
        <tbody>${eventsHTML}</tbody>
      </table>` : ''}
    </div>`;

  // Schedule rank re-render for async tracker lookups.
  window._historyDetailReRender = () => {
    const tbody = document.getElementById(tbodyId);
    if (tbody) tbody.innerHTML = statsRows(blue, 'blue-row') + statsRows(orange, 'orange-row');
  };
};
