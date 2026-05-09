'use strict';

const BC_MATCH_PAGE_SIZE    = 20;
const BC_UPLOADED_PAGE_SIZE = 10;
const BC_GROUPS_PAGE_SIZE   = 10;

let _bcReminderTimer = null;

// Tab-level controller refs (set by pluginInit_bc, used for purge/sync and loadBC)
let _matchWidget    = null;
let _uploadedWidget = null;
let _groupsWidget   = null;

// All instances (tab + dashboard) for WS event routing
let _bcMatchInstances    = [];
let _bcUploadedInstances = [];

// ── Tab-level load / WS-triggered refresh ──────────────────────

async function loadBC() {
  loadBCStatus();
  await Promise.all([
    _matchWidget?.refresh(),
    _uploadedWidget?.refresh(),
    _groupsWidget?.refresh(),
  ].filter(Boolean));
}

async function refreshBCMatches() {
  await (_matchWidget?.refresh() ?? Promise.resolve());
}

// ── Status bar ─────────────────────────────────────────────────

async function loadBCStatus() {
  const el = document.getElementById('bc-status');
  if (!el) return;
  el.innerHTML = '<span class="bc-status-dot bc-dot-pending"></span> Checking…';
  try {
    const r = await fetch('/api/ballchasing/ping');
    if (!r.ok) {
      const j = await r.json().catch(() => ({}));
      el.innerHTML = `<span class="bc-status-dot bc-dot-err"></span> ${esc(j.error || 'Ballchasing API key not set — configure it in Settings.')}`;
      return;
    }
    const j = await r.json();
    el.innerHTML = `<span class="bc-status-dot bc-dot-ok"></span> Connected as <strong>${esc(j.name || '(connected)')}</strong> · Replays auto-upload on match end.`;
  } catch (e) {
    el.innerHTML = `<span class="bc-status-dot bc-dot-err"></span> ${esc(e.message)}`;
  }
}

// ── Widget factories ───────────────────────────────────────────

function bcMatchReplaysWidget(container) {
  let list = [];
  let page = 0;

  function render() {
    if (container.closest('#view-bc')) {
      const purge = document.getElementById('bc-purge-btn');
      if (purge) purge.classList.toggle('hidden', !list.some(m => m.uploaded));
    }

    if (!list.length) {
      container.innerHTML = '<div class="bc-empty">No matches found yet — play a game first.</div>';
      return;
    }

    const start = page * BC_MATCH_PAGE_SIZE;
    container.innerHTML = list.slice(start, start + BC_MATCH_PAGE_SIZE).map(m => bcMatchRow(m)).join('');
    container.querySelectorAll('.bc-upload-btn[data-replay]').forEach(btn => {
      btn.addEventListener('click', () => uploadMatchReplay(btn.dataset.replay, btn));
    });

    const pagerEl = document.createElement('div');
    pagerEl.className = 'flex items-center gap-2 mt-2';
    renderPager(pagerEl, list.length, BC_MATCH_PAGE_SIZE, page,
      p => { page = p; render(); });
    if (Math.ceil(list.length / BC_MATCH_PAGE_SIZE) > 1) container.appendChild(pagerEl);
  }

  async function refresh() {
    const data = await fetch('/api/ballchasing/matches').then(r => r.json()).catch(() => null);
    list = Array.isArray(data) ? data : [];
    page = 0;
    render();
  }

  function getUploadedCount() {
    return list.filter(m => m.uploaded).length;
  }

  function destroy() {
    const i = _bcMatchInstances.indexOf(entry);
    if (i >= 0) _bcMatchInstances.splice(i, 1);
  }

  const entry = { refresh, getUploadedCount };
  _bcMatchInstances.push(entry);
  return { refresh, getUploadedCount, destroy };
}

function bcUploadedReplaysWidget(container) {
  let list   = [];
  let page   = 0;
  const newIds = new Set();

  function render() {
    if (!list.length) {
      container.innerHTML = '<div class="bc-empty">No replays found on Ballchasing.</div>';
      return;
    }
    const start = page * BC_UPLOADED_PAGE_SIZE;
    container.innerHTML = list.slice(start, start + BC_UPLOADED_PAGE_SIZE).map(rp => bcReplayCard(rp, newIds)).join('');

    const pagerEl = document.createElement('div');
    pagerEl.className = 'flex items-center gap-2 mt-2';
    renderPager(pagerEl, list.length, BC_UPLOADED_PAGE_SIZE, page,
      p => { page = p; render(); });
    if (Math.ceil(list.length / BC_UPLOADED_PAGE_SIZE) > 1) container.appendChild(pagerEl);
  }

  async function refresh() {
    const data = await fetch('/api/ballchasing/replays').then(r => r.json()).catch(() => null);
    list = data?.list || [];
    page = 0;
    render();
  }

  function onNewUploads(replays) {
    for (const r of replays) {
      newIds.add(r.bc_id);
      list.unshift({ id: r.bc_id, name: r.name, date: new Date().toISOString() });
    }
    page = 0;
    render();
  }

  function destroy() {
    const i = _bcUploadedInstances.indexOf(entry);
    if (i >= 0) _bcUploadedInstances.splice(i, 1);
  }

  const entry = { refresh, onNewUploads };
  _bcUploadedInstances.push(entry);
  return { refresh, onNewUploads, destroy };
}

function bcGroupsWidget(container) {
  let list = [];
  let page = 0;

  function render() {
    if (!list.length) {
      container.innerHTML = '<div class="bc-empty">No groups found.</div>';
      return;
    }
    const start = page * BC_GROUPS_PAGE_SIZE;
    container.innerHTML = list.slice(start, start + BC_GROUPS_PAGE_SIZE).map(bcGroupCard).join('');

    const pagerEl = document.createElement('div');
    pagerEl.className = 'flex items-center gap-2 mt-2';
    renderPager(pagerEl, list.length, BC_GROUPS_PAGE_SIZE, page,
      p => { page = p; render(); });
    if (Math.ceil(list.length / BC_GROUPS_PAGE_SIZE) > 1) container.appendChild(pagerEl);
  }

  async function refresh() {
    const data = await fetch('/api/ballchasing/groups').then(r => r.json()).catch(() => null);
    list = data?.list || [];
    page = 0;
    render();
  }

  return { refresh };
}

// ── Match row ──────────────────────────────────────────────────

function bcMatchRow(m) {
  const arena = (m.arena && friendlyArena(m.arena)) || 'Unknown Arena';
  const date  = m.started_at ? formatDate(m.started_at) : '—';

  let actions;
  if (m.uploaded) {
    actions = `<span class="bc-uploaded-badge">Uploaded</span>
               <a href="${esc(m.bc_url)}" target="_blank" rel="noopener" class="bc-link">↗ View</a>`;
  } else if (m.replay_exists) {
    actions = `<button class="bc-upload-btn" data-replay="${esc(m.replay_name)}">Upload</button>`;
  } else {
    actions = `<span class="bc-no-replay">No replay saved</span>`;
  }

  return `
    <div class="bc-replay-row${m.replay_exists || m.uploaded ? '' : ' bc-row-dim'}">
      <div class="bc-replay-name">${esc(arena)}</div>
      <div class="bc-replay-meta">${date}</div>
      <div class="bc-replay-actions">${actions}</div>
    </div>`;
}

// ── Upload action ──────────────────────────────────────────────

async function uploadMatchReplay(replayName, btn) {
  if (btn) { btn.disabled = true; btn.textContent = '…'; }
  try {
    const res = await fetch('/api/ballchasing/upload', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ replay_name: replayName, visibility: 'unlisted' }),
    });
    const j = await res.json().catch(() => ({}));
    if (!res.ok) {
      alert(j.error || `Upload failed (${res.status})`);
      if (btn) { btn.disabled = false; btn.textContent = 'Upload'; }
      return;
    }
    _matchWidget?.refresh();
  } catch (e) {
    alert(e.message);
    if (btn) { btn.disabled = false; btn.textContent = 'Upload'; }
  }
}

// ── Sync & purge (page-level actions) ─────────────────────────

async function syncFromBC() {
  const btn = document.getElementById('bc-sync-btn');
  if (btn) { btn.disabled = true; btn.textContent = 'Syncing…'; }
  try {
    const res = await fetch('/api/ballchasing/sync', { method: 'POST' });
    const j   = await res.json().catch(() => ({}));
    if (!res.ok) {
      alert(j.error || `Sync failed (${res.status})`);
      return;
    }
    _matchWidget?.refresh();
    const notify = document.getElementById('bc-notify');
    if (notify) {
      const n = j.synced ?? 0;
      notify.textContent = n > 0
        ? `Synced ${n} replay${n !== 1 ? 's' : ''} from Ballchasing.`
        : 'Already up to date — no new uploads found.';
      notify.classList.remove('hidden');
      setTimeout(() => notify.classList.add('hidden'), 5000);
    }
  } catch (e) {
    alert(e.message);
  } finally {
    if (btn) { btn.disabled = false; btn.textContent = 'Sync from Ballchasing'; }
  }
}

function handleBCSaveReplayReminder() {
  const notify = document.getElementById('bc-notify');
  if (!notify) return;
  notify.textContent = 'Match ended — save your replay from the post-match screen before leaving!';
  notify.classList.remove('hidden');
  clearTimeout(_bcReminderTimer);
  _bcReminderTimer = setTimeout(() => notify.classList.add('hidden'), 5000);
}

async function purgeUploaded() {
  const count = _matchWidget?.getUploadedCount?.() ?? 0;
  if (!confirm(`Delete ${count} uploaded replay file${count !== 1 ? 's' : ''} from disk? Only replays already on Ballchasing will be removed.`)) return;
  const purgeBtn = document.getElementById('bc-purge-btn');
  if (purgeBtn) { purgeBtn.disabled = true; purgeBtn.textContent = '…'; }
  try {
    const res = await fetch('/api/ballchasing/local-replays/purge', { method: 'POST' });
    const j   = await res.json().catch(() => ({}));
    _matchWidget?.refresh();
    if (purgeBtn) { purgeBtn.disabled = false; purgeBtn.textContent = 'Delete Uploaded'; }
    const notify = document.getElementById('bc-notify');
    if (notify && j.deleted != null) {
      notify.textContent = `Deleted ${j.deleted} local replay file${j.deleted !== 1 ? 's' : ''} from disk.`;
      notify.classList.remove('hidden');
      setTimeout(() => notify.classList.add('hidden'), 5000);
    }
  } catch (e) {
    alert(e.message);
    if (purgeBtn) { purgeBtn.disabled = false; purgeBtn.textContent = 'Delete Uploaded'; }
  }
}

// ── WS handler ─────────────────────────────────────────────────

function handleBCUploaded(data) {
  const replays = data?.replays || [];
  if (!replays.length) return;

  _bcUploadedInstances.forEach(w => w.onNewUploads(replays));
  _bcMatchInstances.forEach(w => w.refresh());

  const notify = document.getElementById('bc-notify');
  if (notify) {
    notify.textContent = `${replays.length} replay${replays.length > 1 ? 's' : ''} auto-uploaded to Ballchasing.`;
    notify.classList.remove('hidden');
    setTimeout(() => notify.classList.add('hidden'), 8000);
  }
}

// ── Card renderers ─────────────────────────────────────────────

function bcReplayCard(rp, newIds) {
  const mapName  = rp.map_name || friendlyArena(rp.map_code) || (rp.name ? rp.name.replace(/\.replay$/i,'') : '—');
  const playlist = rp.playlist_name || '';
  const date     = rp.date ? formatDate(rp.date) : '—';
  const blueGoals = rp.blue  != null ? (rp.blue.goals  ?? 0) : null;
  const orgGoals  = rp.orange != null ? (rp.orange.goals ?? 0) : null;
  const score = blueGoals != null && orgGoals != null
    ? `<span class="bc-score"><span style="color:var(--rl-blue)">${blueGoals}</span> — <span style="color:var(--rl-orange)">${orgGoals}</span></span>`
    : '';
  const link  = rp.id ? `https://ballchasing.com/replay/${rp.id}` : '';
  const isNew = rp.id && newIds?.has(rp.id);

  return `
    <div class="bc-card${isNew ? ' bc-card-new' : ''}">
      <div class="bc-card-top">
        <span class="bc-card-map">${esc(mapName)}</span>
        ${playlist ? `<span class="bc-card-playlist">${esc(playlist)}</span>` : ''}
        ${isNew ? '<span class="bc-new-badge">New</span>' : ''}
      </div>
      <div class="bc-card-meta">
        ${score}
        <span>${date}</span>
        ${link ? `<a href="${esc(link)}" target="_blank" rel="noopener" class="bc-link">↗ View</a>` : ''}
      </div>
    </div>`;
}

function bcGroupCard(g) {
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
}

// ── Shared pager ───────────────────────────────────────────────

function renderPager(el, total, pageSize, currentPage, onPageChange) {
  if (!el) return;
  const totalPages = Math.ceil(total / pageSize);
  if (totalPages <= 1) { el.innerHTML = ''; return; }
  el.innerHTML = `
    <span class="text-xs text-gray-500">${currentPage + 1} / ${totalPages}</span>
    <button class="bc-page-btn" ${currentPage > 0 ? '' : 'disabled'} data-dir="-1">‹</button>
    <button class="bc-page-btn" ${currentPage < totalPages - 1 ? '' : 'disabled'} data-dir="1">›</button>
  `;
  el.querySelectorAll('.bc-page-btn').forEach(btn =>
    btn.addEventListener('click', () => onPageChange(currentPage + parseInt(btn.dataset.dir)))
  );
}

// ── Plugin init ────────────────────────────────────────────────

window.pluginInit_bc = function() {
  const matchesContainer  = document.getElementById('bc-matches-widget');
  const uploadedContainer = document.getElementById('bc-uploaded-widget');
  const groupsContainer   = document.getElementById('bc-groups-widget');

  _matchWidget    = matchesContainer ? bcMatchReplaysWidget(matchesContainer) : null;
  _uploadedWidget = uploadedContainer ? bcUploadedReplaysWidget(uploadedContainer) : null;
  _groupsWidget   = groupsContainer ? bcGroupsWidget(groupsContainer) : null;

  document.getElementById('bc-purge-btn')?.addEventListener('click', purgeUploaded);
  document.getElementById('bc-sync-btn')?.addEventListener('click', syncFromBC);

  window.registerWidget?.({
    id: 'bc-match-replays', pluginId: 'bc', title: 'Match Replays',
    defaultW: 6, defaultH: 8, minW: 3, minH: 4,
    factory: bcMatchReplaysWidget,
  });
  window.registerWidget?.({
    id: 'bc-uploaded-replays', pluginId: 'bc', title: 'Ballchasing Replays',
    defaultW: 6, defaultH: 6, minW: 3, minH: 4,
    factory: bcUploadedReplaysWidget,
  });
  window.registerWidget?.({
    id: 'bc-groups', pluginId: 'bc', title: 'Ballchasing Groups',
    defaultW: 4, defaultH: 6, minW: 2, minH: 4,
    factory: bcGroupsWidget,
  });
};
