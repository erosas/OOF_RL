'use strict';

let _dejavuTimer = null;
let _dejavuInFlight = false;
let _dejavuPlayerNamesPromise = null;
let _dejavuPlayerNamesLoaded = false;
let _dejavuPlayerNameByID = new Map();

window.pluginInit_dejavu = function() {
  window.registerView?.('dejavu', {
    onShow: () => refreshDejavu(true),
    fullWidth: true,
    densePadding: true,
  });
  refreshDejavu(true);
  if (!_dejavuTimer) {
    _dejavuTimer = setInterval(() => {
      if (isDejavuActive()) refreshDejavu(false);
    }, 2500);
  }
};

function isDejavuActive() {
  return document.getElementById('view-dejavu')?.classList.contains('active');
}

function dejavuSelectedPlayerID() {
  return (localStorage.getItem('oof_session_player') || '').trim();
}

function loadDejavuPlayerNames() {
  if (_dejavuPlayerNamesLoaded) return Promise.resolve();
  if (_dejavuPlayerNamesPromise) return _dejavuPlayerNamesPromise;
  _dejavuPlayerNamesPromise = fetch('/api/players')
    .then(res => res.ok ? res.json() : [])
    .then(players => {
      const next = new Map();
      for (const p of players || []) {
        const id = (p?.PrimaryID || '').trim();
        const name = (p?.Name || '').trim();
        if (id && name) next.set(id, name);
      }
      _dejavuPlayerNameByID = next;
      _dejavuPlayerNamesLoaded = true;
    })
    .catch(() => {})
    .finally(() => {
      _dejavuPlayerNamesPromise = null;
    });
  return _dejavuPlayerNamesPromise;
}

async function refreshDejavu(showLoading = false, opts = {}) {
  if (_dejavuInFlight) return;
  const selectedID = dejavuSelectedPlayerID();
  updateDejavuAnchor(selectedID);
  loadDejavuPlayerNames().then(() => updateDejavuAnchor(dejavuSelectedPlayerID()));
  if (showLoading) {
    renderDejavuState('loading');
  }
  _dejavuInFlight = true;
  try {
    const params = new URLSearchParams();
    params.set('player', selectedID);
    if (opts.retry) params.set('retry', '1');
    const res = await fetch(`/api/dejavu/recall?${params.toString()}`);
    const data = await res.json();
    if (!res.ok) {
      renderDejavuState('history_error', { error: data.error || 'History unavailable.' });
    } else {
      renderDejavu(data);
    }
  } catch (_) {
    renderDejavuState('history_error', { error: 'History unavailable.' });
  } finally {
    _dejavuInFlight = false;
  }
}

function updateDejavuAnchor(selectedID, selected) {
  const el = document.getElementById('dejavu-anchor-value');
  if (!el) return;
  const id = selected?.primary_id || selectedID || '';
  const knownName = id ? _dejavuPlayerNameByID.get(id) : '';
  if (knownName || selected?.name || selected?.primary_id) {
    el.textContent = knownName || selected.name || selected.primary_id;
  } else {
    el.textContent = selectedID || 'Not selected';
  }
}

function renderDejavu(data) {
  updateDejavuAnchor(data.selected_id || dejavuSelectedPlayerID(), data.selected);
  if (data.status !== 'ok' && data.status !== 'history_error') {
    renderDejavuState(data.status, data);
    return;
  }

  const rows = data.players || [];
  const empty = document.getElementById('dejavu-empty');
  const content = document.getElementById('dejavu-content');
  const error = document.getElementById('dejavu-error');
  const summary = document.getElementById('dejavu-summary');
  const body = document.getElementById('dejavu-rows');
  if (!empty || !content || !error || !summary || !body) return;

  empty.classList.add('hidden');
  content.classList.remove('hidden');
  if (data.status === 'history_error') {
    renderDejavuError(error, data);
    error.classList.remove('hidden');
  } else {
    error.replaceChildren();
    error.classList.add('hidden');
  }

  summary.innerHTML = renderDejavuSummary(rows);
  body.innerHTML = rows.length
    ? rows.map(renderDejavuRow).join('')
    : `<tr><td colspan="6"><span class="dejavu-unavailable">No other current roster players detected.</span></td></tr>`;
}

function renderDejavuError(errorEl, data) {
  errorEl.replaceChildren();
  const message = document.createElement('span');
  message.textContent = data.error || 'History unavailable.';
  errorEl.appendChild(message);

  if (data.retryable !== true) return;

  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'dejavu-retry-btn';
  btn.textContent = 'Retry';
  btn.addEventListener('click', async () => {
    if (_dejavuInFlight || btn.disabled) return;
    btn.disabled = true;
    btn.textContent = 'Retrying...';
    await refreshDejavu(false, { retry: true });
  });
  errorEl.appendChild(btn);
}

function renderDejavuState(status, data = {}) {
  const empty = document.getElementById('dejavu-empty');
  const content = document.getElementById('dejavu-content');
  const title = document.getElementById('dejavu-empty-title');
  const copy = document.getElementById('dejavu-empty-copy');
  if (!empty || !content || !title || !copy) return;

  const msg = dejavuStateMessage(status, data);
  title.textContent = msg.title;
  copy.textContent = msg.copy;
  empty.classList.remove('hidden');
  content.classList.add('hidden');
}

function dejavuStateMessage(status, data) {
  switch (status) {
    case 'loading':
      return {
        title: 'Loading history',
        copy: 'Checking current roster and saved match history.',
      };
    case 'no_active_match':
      return {
        title: 'No active match',
        copy: 'Prior saved-match recall appears when current roster data is available.',
      };
    case 'no_session_selected_player':
      return {
        title: 'No Session-selected tracked player',
        copy: 'Choose a tracked player in Session first. Deja Vu only reads that existing selection.',
      };
    case 'selected_not_in_roster':
      return {
        title: 'Selected tracked player not in current roster',
        copy: 'The current Session selection does not match any stable player ID in this live roster.',
      };
    case 'selected_no_stable_id':
      return {
        title: 'History unavailable for selected player',
        copy: 'The Session-selected tracked player does not have a stable player ID in this roster.',
      };
    case 'selected_team_unclassified':
      return {
        title: 'Waiting for valid tracked-player team assignment',
        copy: 'Deja Vu waits until the Session-selected tracked player is assigned to blue or orange.',
      };
    case 'history_error':
      return {
        title: 'History unavailable',
        copy: data.error || 'The history read failed. Other OOF RL features remain unaffected.',
      };
    default:
      return {
        title: 'History unavailable',
        copy: 'Deja Vu could not read the current roster state.',
      };
  }
}

function renderDejavuSummary(rows) {
  const stableRows = rows.filter(r => r.stable_id);
  const known = stableRows.filter(r => (r.prior_count || 0) > 0);
  const withMatches = sum(rows, 'with_count');
  const againstMatches = sum(rows, 'against_count');
  return [
    summaryItem('Roster players', String(rows.length)),
    summaryItem('Known from saved history', String(known.length)),
    summaryItem('Prior with', String(withMatches)),
    summaryItem('Prior against', String(againstMatches)),
  ].join('');
}

function summaryItem(label, value) {
  return `<div class="dejavu-summary-item">
    <span class="dejavu-summary-label">${esc(label)}</span>
    <span class="dejavu-summary-value">${esc(value)}</span>
  </div>`;
}

function renderDejavuRow(row) {
  const stable = !!row.stable_id;
  const name = row.name || 'Unknown player';
  const badge = stable && typeof playerPlatformBadge === 'function' ? playerPlatformBadge(row.primary_id) : '';
  return `<tr>
    <td>
      <div class="dejavu-player-name"><span>${esc(name)}</span>${badge}</div>
      <div class="dejavu-player-sub">${stable ? esc(row.primary_id) : 'History unavailable - no stable player ID'}</div>
    </td>
    <td>${sideChip(row.current_side)}</td>
    <td>${priorCell(row)}</td>
    <td>${relationCell(row.with_count, row.with_wins, row.with_losses, row.with_no_result, stable, 'with')}</td>
    <td>${relationCell(row.against_count, row.against_wins, row.against_losses, row.against_no_result, stable, 'against')}</td>
    <td>${stable && row.last_seen ? esc(formatDate(row.last_seen)) : '<span class="dejavu-unavailable">-</span>'}</td>
  </tr>`;
}

function sideChip(side) {
  const labels = {
    teammate: 'Teammate',
    opponent: 'Opponent',
    unclassified: 'Assigning team',
  };
  const cls = side === 'teammate' || side === 'opponent' ? side : '';
  return `<span class="dejavu-side-chip ${cls}">${esc(labels[side] || 'Assigning team')}</span>`;
}

function priorCell(row) {
  if (!row.stable_id) {
    return `<div class="dejavu-unavailable">History unavailable</div>
      <div class="dejavu-metric-sub">No stable player ID</div>`;
  }
  const count = row.prior_count || 0;
  if (!count) {
    return `<div class="dejavu-metric-main">No prior saved matches</div>`;
  }
  return `<div class="dejavu-metric-main">${count} prior saved match${count === 1 ? '' : 'es'}</div>`;
}

function relationCell(count, wins, losses, noResult, stable, label) {
  if (!stable) {
    return `<span class="dejavu-unavailable">Unavailable</span>`;
  }
  const c = count || 0;
  const nr = noResult ? `<div class="dejavu-metric-sub">${noResult} no-result saved match${noResult === 1 ? '' : 'es'}</div>` : '';
  if (!c) {
    return `<div class="dejavu-metric-main">0 prior ${esc(label)}</div>${nr}`;
  }
  return `<div class="dejavu-metric-main">${c} - ${wins || 0}-${losses || 0}</div>${nr}`;
}

function sum(rows, key) {
  return rows.reduce((acc, row) => acc + Number(row[key] || 0), 0);
}
