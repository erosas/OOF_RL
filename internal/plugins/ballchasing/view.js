'use strict';

const BC_MATCH_PAGE_SIZE    = 20;
const BC_UPLOADED_PAGE_SIZE = 10;
const BC_GROUPS_PAGE_SIZE   = 10;

let _bcMatchList    = [];
let _bcMatchPage    = 0;
let _bcUploadedList = [];
let _bcUploadedPage = 0;
let _bcGroupsList   = [];
let _bcGroupsPage   = 0;
const _bcNewIds     = new Set();
let _bcReminderTimer = null;

async function loadBC() {
  loadBCStatus();
  const [matchData, bcData, bcGroups] = await Promise.all([
    fetch('/api/ballchasing/matches').then(r => r.json()).catch(() => null),
    fetch('/api/ballchasing/replays').then(r => r.json()).catch(() => null),
    fetch('/api/ballchasing/groups').then(r => r.json()).catch(() => null),
  ]);

  _bcMatchList    = Array.isArray(matchData) ? matchData : [];
  _bcUploadedList = bcData?.list            || [];
  _bcGroupsList   = bcGroups?.list          || [];
  _bcMatchPage    = 0;
  _bcUploadedPage = 0;
  _bcGroupsPage   = 0;

  renderMatchReplays();
  renderBCUploaded();
  renderBCGroups();
}

async function refreshBCMatches() {
  const data = await fetch('/api/ballchasing/matches').then(r => r.json()).catch(() => null);
  if (!data) return;
  _bcMatchList = Array.isArray(data) ? data : [];
  renderMatchReplays();
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
    el.innerHTML = `<span class="bc-status-dot bc-dot-ok"></span> Connected as <strong>${esc(j.name || '(connected)')}</strong> · Replays auto-upload on match end.`;
  } catch (e) {
    el.innerHTML = `<span class="bc-status-dot bc-dot-err"></span> ${esc(e.message)}`;
  }
}

// ── Match replays (history-driven) ────────────────────────────

function renderMatchReplays() {
  const el    = document.getElementById('bc-matches');
  const pager = document.getElementById('bc-matches-pager');
  const purge = document.getElementById('bc-purge-btn');
  if (!el) return;

  const hasUploaded = _bcMatchList.some(m => m.uploaded);
  if (purge) purge.classList.toggle('hidden', !hasUploaded);

  if (!_bcMatchList.length) {
    el.innerHTML = '<div class="bc-empty">No matches found yet — play a game first.</div>';
    if (pager) pager.innerHTML = '';
    return;
  }

  const start = _bcMatchPage * BC_MATCH_PAGE_SIZE;
  const page  = _bcMatchList.slice(start, start + BC_MATCH_PAGE_SIZE);

  el.innerHTML = page.map(m => bcMatchRow(m)).join('');

  el.querySelectorAll('.bc-upload-btn[data-replay]').forEach(btn => {
    btn.addEventListener('click', () => uploadMatchReplay(btn.dataset.replay, btn));
  });

  renderPager(pager, _bcMatchList.length, BC_MATCH_PAGE_SIZE, _bcMatchPage,
    p => { _bcMatchPage = p; renderMatchReplays(); });
}

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

async function uploadMatchReplay(replayName, btn) {
  if (btn) { btn.disabled = true; btn.textContent = '…'; }
  try {
    const res = await fetch('/api/ballchasing/upload', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ replay_name: replayName, visibility: 'public-team' }),
    });
    const j = await res.json().catch(() => ({}));
    if (!res.ok) {
      alert(j.error || `Upload failed (${res.status})`);
      if (btn) { btn.disabled = false; btn.textContent = 'Upload'; }
      return;
    }
    await refreshBCMatches();
  } catch (e) {
    alert(e.message);
    if (btn) { btn.disabled = false; btn.textContent = 'Upload'; }
  }
}

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
    await refreshBCMatches();
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
  const count = _bcMatchList.filter(m => m.uploaded).length;
  if (!confirm(`Delete ${count} uploaded replay file${count !== 1 ? 's' : ''} from disk? Only replays already on Ballchasing will be removed.`)) return;
  const purgeBtn = document.getElementById('bc-purge-btn');
  if (purgeBtn) { purgeBtn.disabled = true; purgeBtn.textContent = '…'; }
  try {
    const res = await fetch('/api/ballchasing/local-replays/purge', { method: 'POST' });
    const j   = await res.json().catch(() => ({}));
    await refreshBCMatches();
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

// ── BC uploaded replays ────────────────────────────────────────

function renderBCUploaded() {
  const el    = document.getElementById('bc-uploaded');
  const pager = document.getElementById('bc-uploaded-pager');
  if (!_bcUploadedList.length) {
    el.innerHTML    = '<div class="bc-empty">No replays found on Ballchasing.</div>';
    if (pager) pager.innerHTML = '';
    return;
  }
  const start = _bcUploadedPage * BC_UPLOADED_PAGE_SIZE;
  el.innerHTML = _bcUploadedList.slice(start, start + BC_UPLOADED_PAGE_SIZE).map(bcReplayCard).join('');
  renderPager(pager, _bcUploadedList.length, BC_UPLOADED_PAGE_SIZE, _bcUploadedPage,
    p => { _bcUploadedPage = p; renderBCUploaded(); });
}

// ── Groups ─────────────────────────────────────────────────────

function renderBCGroups() {
  const el    = document.getElementById('bc-groups');
  const pager = document.getElementById('bc-groups-pager');
  if (!_bcGroupsList.length) {
    el.innerHTML    = '<div class="bc-empty">No groups found.</div>';
    if (pager) pager.innerHTML = '';
    return;
  }
  const start = _bcGroupsPage * BC_GROUPS_PAGE_SIZE;
  el.innerHTML = _bcGroupsList.slice(start, start + BC_GROUPS_PAGE_SIZE).map(bcGroupCard).join('');
  renderPager(pager, _bcGroupsList.length, BC_GROUPS_PAGE_SIZE, _bcGroupsPage,
    p => { _bcGroupsPage = p; renderBCGroups(); });
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

// ── Card renderers ─────────────────────────────────────────────

function bcReplayCard(rp) {
  const mapName  = rp.map_name || friendlyArena(rp.map_code) || (rp.name ? rp.name.replace(/\.replay$/i,'') : '—');
  const playlist = rp.playlist_name || '';
  const date     = rp.date ? formatDate(rp.date) : '—';
  const blueGoals = rp.blue  != null ? (rp.blue.goals  ?? 0) : null;
  const orgGoals  = rp.orange != null ? (rp.orange.goals ?? 0) : null;
  const score = blueGoals != null && orgGoals != null
    ? `<span class="bc-score"><span style="color:var(--rl-blue)">${blueGoals}</span> — <span style="color:var(--rl-orange)">${orgGoals}</span></span>`
    : '';
  const link  = rp.id ? `https://ballchasing.com/replay/${rp.id}` : '';
  const isNew = rp.id && _bcNewIds.has(rp.id);

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

// ── WS handler ─────────────────────────────────────────────────

function handleBCUploaded(data) {
  const replays = data?.replays || [];
  if (!replays.length) return;

  for (const r of replays) {
    _bcNewIds.add(r.bc_id);
    _bcUploadedList.unshift({ id: r.bc_id, name: r.name, date: new Date().toISOString() });
  }
  _bcUploadedPage = 0;
  renderBCUploaded();

  // Refresh match list so newly auto-uploaded rows flip to "Uploaded".
  refreshBCMatches();

  const notify = document.getElementById('bc-notify');
  if (notify) {
    notify.textContent = `${replays.length} replay${replays.length > 1 ? 's' : ''} auto-uploaded to Ballchasing.`;
    notify.classList.remove('hidden');
    setTimeout(() => notify.classList.add('hidden'), 8000);
  }
}

window.pluginInit_bc = function() {
  document.getElementById('bc-purge-btn')?.addEventListener('click', purgeUploaded);
  document.getElementById('bc-sync-btn')?.addEventListener('click', syncFromBC);
};