'use strict';

let _dashGrid        = null;
let _dashReady       = false;
let _dashLayoutLoaded = false;
let _dashEditing     = false;
let _dashPreEdit     = null; // layout snapshot taken on Edit click

// Called by showView('dashboard') in app.js every time the tab is opened.
// Initialises the grid on first call (requires the container to be visible),
// then loads the saved layout exactly once per page load.
async function loadDashboard() {
  if (!_dashReady) return;

  if (!_dashGrid) {
    _dashGrid = GridStack.init({
      column:     12,
      cellHeight: 60,
      margin:     6,
      handle:     '.widget-header',
      staticGrid: true,
      float:      true,
      resizable:  { handles: 's,e,se' },
      columnOpts: {
        breakpointForWindow: false,
        breakpoints: [
          { w: 1100, c: 12 },
          { w: 700,  c: 8  },
          { w: 480,  c: 4  },
        ],
      },
    }, document.getElementById('dash-grid'));
  }

  if (_dashLayoutLoaded) return;
  _dashLayoutLoaded = true;

  const layout = await fetch('/api/dashboard/layout').then(r => r.json()).catch(() => []);

  if (Array.isArray(layout)) {
    for (const item of layout) {
      _dashAddWidget(item.id, item.x, item.y, item.w, item.h);
    }
  }

  _dashUpdateEmpty();
}

// ── Widget management ──────────────────────────────────────────

function _dashAddWidget(widgetId, x, y, w, h) {
  const def = window.widgetRegistry?.[widgetId];
  if (!def) {
    console.warn('[dashboard] unknown widget:', widgetId);
    return;
  }

  const fromLayout = x != null && y != null;
  const itemEl = _dashGrid.addWidget({
    ...(fromLayout ? { x, y } : { autoPosition: true }),
    w:    w    ?? def.defaultW ?? 6,
    h:    h    ?? def.defaultH ?? 6,
    minW: def.minW,
    minH: def.minH,
  });

  itemEl.dataset.widgetId = widgetId;

  const contentEl = itemEl.querySelector('.grid-stack-item-content');
  contentEl.innerHTML = `
    <div class="widget-header">
      <span class="widget-drag-icon">⣿</span>
      <span class="widget-title-text">${esc(def.title)}</span>
      <button class="widget-remove-btn" title="Remove">✕</button>
    </div>
    <div class="widget-body"></div>
  `;

  contentEl.querySelector('.widget-remove-btn').addEventListener('click', () => {
    _dashGrid.removeWidget(itemEl);
    _dashUpdateEmpty();
  });

  const ctrl = def.factory(contentEl.querySelector('.widget-body'));
  ctrl?.refresh?.();
}

function _dashUpdateEmpty() {
  const hasItems = _dashGrid.el.querySelectorAll('.grid-stack-item').length > 0;
  document.getElementById('dash-empty')?.classList.toggle('hidden', hasItems);
}

function _dashGetLayout() {
  const items = [];
  _dashGrid.el.querySelectorAll('.grid-stack-item').forEach(el => {
    const node = el.gridstackNode;
    if (!node || !el.dataset.widgetId) return;
    items.push({ id: el.dataset.widgetId, x: node.x, y: node.y, w: node.w, h: node.h });
  });
  return items;
}

// ── Edit mode ──────────────────────────────────────────────────

function _dashEnterEdit() {
  _dashEditing  = true;
  _dashPreEdit  = _dashGetLayout();

  _dashGrid.setStatic(false);
  _dashGrid.el.classList.add('dash-editing');

  document.getElementById('dash-edit-btn').classList.add('hidden');
  document.getElementById('dash-save-btn').classList.remove('hidden');
  document.getElementById('dash-cancel-btn').classList.remove('hidden');
  document.getElementById('dash-add-widget-btn').classList.remove('hidden');
}

function _dashExitEdit() {
  _dashEditing = false;
  _dashGrid.setStatic(true);
  _dashGrid.el.classList.remove('dash-editing');
  _closePicker();

  document.getElementById('dash-edit-btn').classList.remove('hidden');
  document.getElementById('dash-save-btn').classList.add('hidden');
  document.getElementById('dash-cancel-btn').classList.add('hidden');
  document.getElementById('dash-add-widget-btn').classList.add('hidden');
}

async function _dashSave() {
  const layout = _dashGetLayout();
  const res = await fetch('/api/dashboard/layout', {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify(layout),
  });
  if (!res.ok) { alert('Failed to save layout.'); return; }
  _dashPreEdit = null;
  _dashExitEdit();
}

function _dashCancel() {
  if (_dashPreEdit !== null) {
    _dashGrid.removeAll(true);
    for (const item of _dashPreEdit) {
      _dashAddWidget(item.id, item.x, item.y, item.w, item.h);
    }
    _dashUpdateEmpty();
  }
  _dashExitEdit();
}

// ── Widget picker ──────────────────────────────────────────────

function _openPicker() {
  const list = document.getElementById('dash-picker-list');
  const widgets = Object.values(window.widgetRegistry || {});

  list.innerHTML = '';
  if (!widgets.length) {
    const msg = document.createElement('div');
    msg.style.cssText = 'color:var(--muted);font-size:0.85rem;text-align:center;padding:24px';
    msg.textContent = 'No widgets registered yet.';
    list.appendChild(msg);
  } else {
    for (const def of widgets) {
      const item = document.createElement('div');
      item.className = 'dash-picker-item';

      const info = document.createElement('div');
      const name = document.createElement('div');
      name.className = 'dash-picker-item-name';
      name.textContent = def.title;
      const plugin = document.createElement('div');
      plugin.className = 'dash-picker-item-plugin';
      plugin.textContent = def.pluginId;
      info.appendChild(name);
      info.appendChild(plugin);

      const btn = document.createElement('button');
      btn.className = 'bc-sync-btn';
      btn.textContent = 'Add';
      btn.addEventListener('click', () => {
        _dashAddWidget(def.id);
        _dashUpdateEmpty();
        _closePicker();
      });

      item.appendChild(info);
      item.appendChild(btn);
      list.appendChild(item);
    }
  }

  document.getElementById('dash-picker').classList.remove('hidden');
}

function _closePicker() {
  document.getElementById('dash-picker')?.classList.add('hidden');
}

// ── Plugin init ────────────────────────────────────────────────

window.pluginInit_dashboard = async function() {
  if (!window.GridStack) {
    await loadScript('/plugins/dashboard/gridstack-all.js');
  }

  document.getElementById('dash-edit-btn')?.addEventListener('click', _dashEnterEdit);
  document.getElementById('dash-save-btn')?.addEventListener('click', _dashSave);
  document.getElementById('dash-cancel-btn')?.addEventListener('click', _dashCancel);
  document.getElementById('dash-add-widget-btn')?.addEventListener('click', _openPicker);
  document.getElementById('dash-picker-close')?.addEventListener('click', _closePicker);

  _dashReady = true;

  // If the dashboard tab was already shown before async init finished, load now.
  const section = document.getElementById('view-dashboard');
  if (section?.classList.contains('active')) loadDashboard();
};