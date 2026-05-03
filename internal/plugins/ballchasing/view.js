'use strict';

const BC_PAGE_SIZE = 10;

let _bcUploadedPage = 0;
let _bcGroupsPage   = 0;
let _bcUploadedList = [];
let _bcGroupsList   = [];
const _bcNewIds     = new Set();

async function loadBC() {
  loadBCStatus();
  const [bcData, bcGroups] = await Promise.all([
    fetch('/api/ballchasing/replays').then(r => r.json()).catch(() => null),
    fetch('/api/ballchasing/groups').then(r => r.json()).catch(() => null),
  ]);

  _bcUploadedList = bcData?.list  || [];
  _bcGroupsList   = bcGroups?.list || [];
  _bcUploadedPage = 0;
  _bcGroupsPage   = 0;

  renderBCUploaded();
  renderBCGroups();
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

function renderBCUploaded() {
  const el    = document.getElementById('bc-uploaded');
  const pager = document.getElementById('bc-uploaded-pager');
  if (!_bcUploadedList.length) {
    el.innerHTML    = '<div class="bc-empty">No uploaded replays found.</div>';
    pager.innerHTML = '';
    return;
  }
  const start = _bcUploadedPage * BC_PAGE_SIZE;
  el.innerHTML = _bcUploadedList.slice(start, start + BC_PAGE_SIZE).map(bcReplayCard).join('');
  renderPager(pager, _bcUploadedList.length, _bcUploadedPage, p => { _bcUploadedPage = p; renderBCUploaded(); });
}

function renderBCGroups() {
  const el    = document.getElementById('bc-groups');
  const pager = document.getElementById('bc-groups-pager');
  if (!_bcGroupsList.length) {
    el.innerHTML    = '<div class="bc-empty">No groups found.</div>';
    pager.innerHTML = '';
    return;
  }
  const start = _bcGroupsPage * BC_PAGE_SIZE;
  el.innerHTML = _bcGroupsList.slice(start, start + BC_PAGE_SIZE).map(bcGroupCard).join('');
  renderPager(pager, _bcGroupsList.length, _bcGroupsPage, p => { _bcGroupsPage = p; renderBCGroups(); });
}

function renderPager(el, total, currentPage, onPageChange) {
  const totalPages = Math.ceil(total / BC_PAGE_SIZE);
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

function bcReplayCard(rp) {
  const mapName  = rp.map_name || friendlyArena(rp.map_code) || (rp.name ? rp.name.replace(/\.replay$/i,'') : '—');
  const playlist = rp.playlist_name || '';
  const date     = rp.date ? formatDate(rp.date) : '—';
  const blueGoals = rp.blue?.goals ?? null;
  const orgGoals  = rp.orange?.goals ?? null;
  const score = blueGoals !== null
    ? `<span class="bc-score"><span style="color:var(--rl-blue)">${blueGoals}</span> — <span style="color:var(--rl-orange)">${orgGoals}</span></span>`
    : '';
  const link  = rp.id ? `https://ballchasing.com/replay/${rp.id}` : '';
  const isNew = rp.id && _bcNewIds.has(rp.id);
  const newBadge = isNew ? '<span class="bc-new-badge">New</span>' : '';

  return `
    <div class="bc-card${isNew ? ' bc-card-new' : ''}">
      <div class="bc-card-top">
        <span class="bc-card-map">${esc(mapName)}</span>
        ${playlist ? `<span class="bc-card-playlist">${esc(playlist)}</span>` : ''}
        ${newBadge}
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

// Called from app.js WS handler when a bc:uploaded event arrives.
function handleBCUploaded(data) {
  const replays = data?.replays || [];
  if (!replays.length) return;

  for (const r of replays) {
    _bcNewIds.add(r.bc_id);
    _bcUploadedList.unshift({ id: r.bc_id, name: r.name, date: new Date().toISOString() });
  }
  _bcUploadedPage = 0;
  renderBCUploaded();

  const notify = document.getElementById('bc-notify');
  if (notify) {
    notify.textContent = `${replays.length} new replay${replays.length > 1 ? 's' : ''} auto-uploaded.`;
    notify.classList.remove('hidden');
    setTimeout(() => notify.classList.add('hidden'), 8000);
  }
}

window.pluginInit_bc = function() {};