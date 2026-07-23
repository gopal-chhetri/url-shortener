/**
 * ShortURL: Precision App Logic
 * Real API integration with backend services
 */

'use strict';

/* ── Theme ── */
function getPreferredTheme() {
  const stored = localStorage.getItem('su_theme');
  if (stored === 'light' || stored === 'dark') return stored;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function setTheme(theme) {
  document.documentElement.classList.toggle('dark', theme === 'dark');
  localStorage.setItem('su_theme', theme);
}

function toggleTheme() {
  const isDark = document.documentElement.classList.contains('dark');
  setTheme(isDark ? 'light' : 'dark');
}

/* ── State ── */
const state = {
  token: localStorage.getItem('su_token') || null,
  user:  JSON.parse(localStorage.getItem('su_user') || 'null'),
  currentView: 'dashboard',
  urlFilter: 'active',
  userUrlSort: 'date',
  adminUrlFilter: 'active',
  adminUrlSearch: '',
  adminUrlSort: 'date',
  adminUrlUserId: null,
  charts: {},
};

const API_BASE = (window.location.protocol === 'file:' || window.location.port === '5500' || window.location.port === '3000') 
  ? 'http://localhost:8080/api/v1' 
  : '/api/v1';

/* ─────────────────────────────────────────
   API WRAPPER
──────────────────────────────────────── */
async function apiFetch(endpoint, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  if (state.token) headers['Authorization'] = `Bearer ${state.token}`;

  try {
    const res = await fetch(`${API_BASE}${endpoint}`, { ...options, headers });
    const data = await res.json();
    
    if (!res.ok) {
      if (res.status === 401 && endpoint !== '/auth/login') {
        logout(false);
        throw new Error('Session expired');
      }
      throw new Error(data.message || data.error || 'API Request Failed');
    }
    return data;
  } catch (err) {
    throw err;
  }
}

/* ─────────────────────────────────────────
   UTILITIES
──────────────────────────────────────── */
const $ = id => document.getElementById(id);

function fmt(n) {
  if (n === undefined || n === null) return '0';
  if (n >= 1_000_000) return (n/1_000_000).toFixed(1)+'M';
  if (n >= 1_000)     return (n/1_000).toFixed(1)+'K';
  return String(n);
}

function showToast(msg, type = 'success') {
  const t = $('toast');
  t.innerHTML = `<i class="ph ph-${type === 'success' ? 'check-circle' : 'warning-circle'}"></i> <span>${msg}</span>`;
  t.className = `toast toast-${type}`;
  clearTimeout(t._timer);
  t._timer = setTimeout(() => { t.className = 'toast hidden'; }, 4000);
}

function setLoading(btn, on) {
  const txt = btn.querySelector('.btn-text');
  const sp  = btn.querySelector('.btn-spinner');
  if (!txt || !sp) return;
  btn.disabled = on;
  txt.classList.toggle('hidden', on);
  sp.classList.toggle('hidden', !on);
}

function css(token) {
  return getComputedStyle(document.documentElement).getPropertyValue(token).trim();
}

function hsla(token, alpha) {
  const val = css(token);
  const m = val.match(/hsl\(([^)]+)\)/);
  return m ? `hsla(${m[1]}, ${alpha})` : val;
}

function refreshIcons() {
  if (window.PhosphorIcons) {
    PhosphorIcons.render();
  }
}

function updateNavbar() {
  const signIn = $('nav-sign-in');
  const cta = $('nav-cta');
  if (!signIn || !cta) return;

  if (state.token && state.user) {
    // Logged in: hide Sign In, change CTA to Dashboard
    signIn.classList.add('hidden');
    cta.textContent = 'Dashboard';
    cta.href = '#';
    cta.onclick = (e) => {
      e.preventDefault();
      if (!$('view-app').classList.contains('hidden')) {
        switchView('dashboard');
      } else {
        // If on login screen, enter app
        enterApp(state.user, state.token);
      }
    };
  } else {
    // Logged out: show Sign In, change CTA to Start Building
    signIn.classList.remove('hidden');
    cta.textContent = 'Start Building';
    cta.href = '/app/index.html';
    cta.onclick = null;
  }
}

/* ─────────────────────────────────────────
   AUTH
──────────────────────────────────────── */
function enterApp(user, token) {
  state.token = token;
  state.user = user;
  localStorage.setItem('su_token', token);
  localStorage.setItem('su_user', JSON.stringify(user));

  /* update sidebar */
  const initials = (user.first_name[0] + (user.last_name?.[0]||'')).toUpperCase();
  $('sidebar-avatar').textContent = initials;
  $('sidebar-name').textContent   = `${user.first_name} ${user.last_name||''}`.trim();
  $('sidebar-role').textContent   = user.role;

  /* show admin nav if needed */
  if (user.role === 'admin') {
    $('nav-admin-users').classList.remove('hidden');
    $('nav-admin-links').classList.remove('hidden');
    $('nav-my-urls').classList.add('hidden');
  } else {
    $('nav-admin-users').classList.add('hidden');
    $('nav-admin-links').classList.add('hidden');
    $('nav-my-urls').classList.remove('hidden');
  }

  $('view-login').classList.add('hidden');
  $('view-app').classList.remove('hidden');
  updateNavbar();
  refreshIcons();
  switchView('dashboard');
}

async function logout(callApi = true) {
  if (callApi && state.token) {
    try { await apiFetch('/auth/logout', { method: 'POST' }); } catch(e) {}
  }
  
  localStorage.removeItem('su_token');
  localStorage.removeItem('su_user');
  state.token = null;
  state.user  = null;
  $('view-app').classList.add('hidden');
  $('view-login').classList.remove('hidden');
  updateNavbar();
  
  Object.values(state.charts).forEach(c => c?.destroy());
  state.charts = {};
}

/* Login */
$('form-login').addEventListener('submit', async e => {
  e.preventDefault();
  const btn = $('btn-login');
  setLoading(btn, true);
  $('auth-error').classList.add('hidden');

  const email    = $('login-email').value;
  const password = $('login-password').value;

  try {
    const res = await apiFetch('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password })
    });
    // Response wrapper has { data: { token, user } }
    const { token, user } = res.data;
    enterApp(user, token);
    showToast(`Signed in as ${user.first_name}`);
  } catch (err) {
    const errEl = $('auth-error');
    errEl.innerHTML = `<i class="ph ph-warning"></i> ${err.message}`;
    errEl.classList.remove('hidden');
    refreshIcons();
  } finally {
    setLoading(btn, false);
  }
});

/* Register */
$('form-register').addEventListener('submit', async e => {
  e.preventDefault();
  const btn = $('btn-register');
  setLoading(btn, true);
  
  const payload = {
    first_name: $('reg-first').value,
    last_name:  $('reg-last').value,
    email:      $('reg-email').value,
    password:   $('reg-password').value,
  };

  try {
    await apiFetch('/auth/register', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
    showToast('Account created. Sign in to continue.');
    toggleForms(false);
  } catch (err) {
    showToast(err.message, 'error');
  } finally {
    setLoading(btn, false);
  }
});

function toggleForms(showRegister) {
  $('form-login').classList.toggle('active', !showRegister);
  $('form-login').classList.toggle('hidden',  showRegister);
  $('form-register').classList.toggle('hidden', !showRegister);
  $('auth-error').classList.add('hidden');
}
$('go-register').addEventListener('click', e => { e.preventDefault(); toggleForms(true); });
$('go-login').addEventListener('click',    e => { e.preventDefault(); toggleForms(false); });
$('btn-logout').addEventListener('click', () => logout(true));

/* ─────────────────────────────────────────
   VIEW NAVIGATION
──────────────────────────────────────── */
function switchView(name) {
  document.querySelectorAll('.page').forEach(p => {
    p.classList.remove('active');
    p.classList.add('hidden');
  });
  document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));

  const page = document.getElementById(`page-${name}`);
  const nav  = document.getElementById(`nav-${name}`);
  if (page) {
    page.classList.remove('hidden');
    page.classList.add('active');
  }
  if (nav) nav.classList.add('active');

  state.currentView = name;

  if (name === 'dashboard') loadDashboard();
  if (name === 'my-urls')   loadMyUrls();
  if (name === 'admin-users') loadAdminUsers();
  if (name === 'admin-links') loadAdminLinks();
}

document.querySelectorAll('.nav-item').forEach(item => {
  item.addEventListener('click', e => {
    e.preventDefault();
    switchView(item.dataset.view);
  });
});

/* ─────────────────────────────────────────
   DASHBOARD
──────────────────────────────────────── */
function skeletonRows(containerId, rows = 5, cols = 5) {
  const tbody = $(containerId);
  if (!tbody) return;
  tbody.innerHTML = Array.from({ length: rows }, () =>
    `<tr>${Array.from({ length: cols }, () =>
      `<td><div class="skeleton skeleton-text"></div></td>`
    ).join('')}</tr>`
  ).join('');
}

async function loadDashboard() {
  skeletonRows('tbody-recent-urls', 4, 5);

  try {
    const res = await apiFetch('/urls?limit=5');
    const urls = res.data || [];

    animateCount($('stat-my-urls'), res.metadata?.total || urls.length);
    animateCount($('stat-clicks'),  urls.reduce((acc, u) => acc + (u.click_count||0), 0));
    animateCount($('stat-active'),  urls.filter(u => u.is_active !== false).length);

    renderRecentUrls(urls);
  } catch (err) {
    showToast('Failed to load dashboard data: ' + err.message, 'error');
  }
}

function animateCount(el, target) {
  if (!el || target === undefined) return;
  const duration = 800;
  const start = performance.now();
  const from = 0;
  function update(now) {
    const p = Math.min((now - start) / duration, 1);
    const ease = 1 - Math.pow(1 - p, 3);
    el.textContent = fmt(Math.round(from + (target - from) * ease));
    if (p < 1) requestAnimationFrame(update);
  }
  requestAnimationFrame(update);
}

function renderRecentUrls(urls) {
  const tbody = $('tbody-recent-urls');
  if (!urls.length) {
    tbody.innerHTML = `<tr><td colspan="5"><div class="empty-state"><i class="ph ph-link ph-lg"></i><h3>No routes yet</h3><p>Create your first shortened link above.</p></div></td></tr>`;
    return;
  }
  tbody.innerHTML = urls.map(u => `
    <tr>
      <td><a href="${u.short_url}" target="_blank" class="short-link data-font">${u.short_url.split('/').pop()}</a></td>
      <td class="url-orig-cell hide-mobile" title="${u.original_url}">${u.original_url}</td>
      <td class="data-font text-data">${fmt(u.click_count || 0)}</td>
      <td class="hide-mobile">
        <span class="status-badge ${u.is_active !== false ? 'active' : 'inactive'}">
          <span class="status-dot"></span>${u.is_active !== false ? 'Live' : 'Offline'}
        </span>
      </td>
      <td>
        <div class="table-actions">
          <label class="toggle-switch" title="Toggle Route State">
            <input type="checkbox" ${u.is_active !== false ? 'checked' : ''} onchange="toggleUrl('${u.id}', this.checked)">
            <span class="toggle-track"><span class="toggle-thumb"></span></span>
          </label>
          <button class="btn btn-ghost btn-xs" onclick="showAnalytics('${u.id}', '${u.short_url}')">
            <i class="ph ph-chart-bar ph-sm"></i>
          </button>
          <button class="btn btn-danger btn-xs" onclick="deleteUrl('${u.id}')">
            <i class="ph ph-trash ph-sm"></i>
          </button>
        </div>
      </td>
    </tr>
  `).join('');
  refreshIcons();
}

/* Create URL */
$('form-create-url').addEventListener('submit', async e => {
  e.preventDefault();
  const btn = $('btn-shorten');
  const original_url = $('input-original-url').value;
  const custom_slug = $('input-custom-slug').value;
  setLoading(btn, true);

  try {
    const res = await apiFetch('/urls', {
      method: 'POST',
      body: JSON.stringify({ original_url, custom_slug: custom_slug || undefined })
    });
    
    const urlData = res.data;
    
    const resultEl = $('shorten-result');
    const linkEl   = $('result-short-url');
    linkEl.textContent = urlData.short_url.split('/').pop();
    linkEl.href = urlData.short_url;
    resultEl.classList.remove('hidden');

    showToast('Link created.');
    $('form-create-url').reset();
    
    if (state.currentView === 'dashboard') loadDashboard();
  } catch(err) {
    showToast(err.message, 'error');
  } finally {
    setLoading(btn, false);
  }
});

$('btn-copy').addEventListener('click', () => {
  navigator.clipboard.writeText($('result-short-url').href)
    .then(() => showToast('Copied to clipboard!'))
    .catch(() => showToast('Copy failed', 'error'));
});

$('btn-view-all-urls').addEventListener('click', () => switchView('my-urls'));

/* ─────────────────────────────────────────
   ANALYTICS PANEL
──────────────────────────────────────── */
let dailyChart = null, deviceChart = null, browserChart = null, geoChart = null;

async function showAnalytics(id, shortUrl) {
  const panel = $('analytics-panel');
  $('analytics-url-label').textContent = shortUrl.split('/').pop();
  panel.classList.remove('hidden');
  panel.scrollIntoView({ behavior: 'smooth', block: 'nearest' });

  try {
    const res = await apiFetch(`/urls/${id}/analytics`);
    const d = res.data;
    
    const dailyData = {
      labels: d.daily_clicks ? d.daily_clicks.map(x => x.date.split('T')[0]) : ['No Data'],
      data: d.daily_clicks ? d.daily_clicks.map(x => x.clicks) : [0]
    };
    
    const deviceData = {
      labels: d.device_stats ? d.device_stats.map(x => x.device_type || 'Unknown') : ['No Data'],
      data: d.device_stats ? d.device_stats.map(x => x.clicks) : [0]
    };
    
    const browserData = {
      labels: d.browser_stats ? d.browser_stats.map(x => x.browser || 'Unknown') : ['No Data'],
      data: d.browser_stats ? d.browser_stats.map(x => x.clicks) : [0]
    };

    const geoData = {
      labels: d.geo_stats ? d.geo_stats.map(x => x.country || 'Unknown') : ['No Data'],
      data: d.geo_stats ? d.geo_stats.map(x => x.count) : [0]
    };

    renderLineChart(dailyData);
    renderDonutChart('chart-device',  deviceData);
    renderDonutChart('chart-browser', browserData);
    renderDonutChart('chart-geo', geoData);
  } catch (err) {
    showToast('Failed to load analytics: ' + err.message, 'error');
  }
}

$('btn-close-analytics').addEventListener('click', () => {
  $('analytics-panel').classList.add('hidden');
});

function renderLineChart(data) {
  const ctx = $('chart-daily').getContext('2d');
  if (state.charts.daily) state.charts.daily.destroy();

  const trace = css('--trace');
  const surface = css('--surface');

  const gradient = ctx.createLinearGradient(0, 0, 0, 200);
  gradient.addColorStop(0, hsla('--trace', 0.35));
  gradient.addColorStop(1, hsla('--trace', 0.01));

  state.charts.daily = new Chart(ctx, {
    type: 'line',
    data: {
      labels: data.labels,
      datasets: [{
        label: 'Clicks',
        data: data.data,
        borderColor: trace,
        backgroundColor: gradient,
        borderWidth: 2,
        pointBackgroundColor: trace,
        pointBorderColor: surface,
        pointBorderWidth: 2,
        pointRadius: 4,
        tension: 0.3,
        fill: true,
      }],
    },
    options: chartDefaults({ legend: false }),
  });
}

function renderDonutChart(canvasId, data) {
  const ctx = $(canvasId).getContext('2d');
  const key = canvasId.replace('chart-', '');
  if (state.charts[key]) state.charts[key].destroy();

  state.charts[key] = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: data.labels,
      datasets: [{
        data: data.data,
        backgroundColor: [css('--trace'), css('--mark'), css('--data'), css('--success'), css('--danger')],
        borderColor: css('--surface'),
        borderWidth: 2,
        hoverOffset: 4,
      }],
    },
    options: {
      ...chartDefaults({ legend: true }),
      cutout: '68%',
    },
  });
}

function chartDefaults({ legend }) {
    return {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: !!legend,
        position: 'bottom',
        labels: {
          color: css('--muted'),
          font: { family: 'JetBrains Mono', size: 10 },
          boxWidth: 10, padding: 12,
        },
      },
      tooltip: {
        backgroundColor: hsla('--surface', 0.95),
        titleColor: css('--text'),
        bodyColor: css('--muted'),
        borderColor: hsla('--trace', 0.3),
        borderWidth: 1, padding: 10, cornerRadius: 8,
        titleFont: { family: 'Inter', weight: '600' },
        bodyFont:  { family: 'Inter' },
      },
    },
    scales: legend ? undefined : {
      x: {
        ticks: { color: css('--muted'), font: { family: 'JetBrains Mono', size: 10 } },
        grid:  { color: hsla('--trace', 0.06) },
        border:{ color: 'transparent' },
      },
      y: {
        ticks: { color: css('--muted'), font: { family: 'JetBrains Mono', size: 10 } },
        grid:  { color: hsla('--trace', 0.06) },
        border:{ color: 'transparent' },
      },
    },
  };
}

/* ─────────────────────────────────────────
   MY URLS VIEW
──────────────────────────────────────── */
async function loadMyUrls(page = 1) {
  skeletonRows('tbody-all-urls', 6, 6);

  try {
    const res = await apiFetch(`/urls?page=${page}&limit=10&sort=${state.userUrlSort}`);
    let urls = res.data || [];
    
    if (state.urlFilter === 'active') {
      urls = urls.filter(u => u.is_active !== false);
    }
    
    const tbody = $('tbody-all-urls');
    if (!urls.length) {
      tbody.innerHTML = `<tr><td colspan="6"><div class="empty-state"><i class="ph ph-link ph-lg"></i><h3>No routes found</h3><p>${state.urlFilter === 'active' ? 'No active routes.' : 'No routes found.'}</p></div></td></tr>`;
      $('pagination-urls').innerHTML = '';
      return;
    }
    
    tbody.innerHTML = urls.map(u => `
      <tr>
        <td><a href="${u.short_url}" target="_blank" class="short-link data-font">${u.short_url.split('/').pop()}</a></td>
        <td class="url-orig-cell hide-mobile" title="${u.original_url}">${u.original_url}</td>
        <td class="data-font text-muted hide-tablet">${new Date(u.created_at).toLocaleDateString()}</td>
        <td class="data-font text-data">${fmt(u.click_count || 0)}</td>
        <td class="hide-mobile">
          <span class="status-badge ${u.is_active !== false ? 'active' : 'inactive'}">
            <span class="status-dot"></span>${u.is_active !== false ? 'Live' : 'Offline'}
          </span>
        </td>
        <td>
          <div class="table-actions">
            <label class="toggle-switch" title="Toggle State">
              <input type="checkbox" ${u.is_active !== false ? 'checked' : ''} onchange="toggleUrl('${u.id}', this.checked)">
              <span class="toggle-track"><span class="toggle-thumb"></span></span>
            </label>
            <button class="btn btn-ghost btn-xs" onclick="switchView('dashboard'); setTimeout(()=>showAnalytics('${u.id}', '${u.short_url}'), 300)">
              <i class="ph ph-chart-bar ph-sm"></i>
            </button>
            <button class="btn btn-danger btn-xs" onclick="deleteUrl('${u.id}')">
              <i class="ph ph-trash ph-sm"></i>
            </button>
          </div>
        </td>
      </tr>
    `).join('');
    
    refreshIcons();
    renderPagination('pagination-urls', res.metadata.page, res.metadata.total_pages, loadMyUrls);
  } catch (err) {
    showToast('Failed to load links: ' + err.message, 'error');
  }
}

/* ─────────────────────────────────────────
   ADMIN USERS VIEW
──────────────────────────────────────── */
async function loadAdminUsers(page = 1) {
  skeletonRows('tbody-admin-users', 6, 5);

  try {
    // Stats
    const statsRes = await apiFetch('/admin/stats');
    const s = statsRes.data;
    animateCount($('admin-stat-users'),        s.total_users);
    animateCount($('admin-stat-active-urls'),  s.active_urls);
    animateCount($('admin-stat-inactive-urls'),s.inactive_urls);
    animateCount($('admin-stat-clicks'),       s.total_clicks);

    // Users
    const usersRes = await apiFetch(`/admin/users?page=${page}&limit=10`);
    const users = usersRes.data || [];

    if (!users.length) {
      $('tbody-admin-users').innerHTML = `<tr><td colspan="5"><div class="empty-state"><i class="ph ph-users ph-lg"></i><h3>No users</h3><p>No users found.</p></div></td></tr>`;
      $('pagination-admin-users').innerHTML = '';
      return;
    }

    $('tbody-admin-users').innerHTML = users.map(u => `
      <tr>
        <td>
          <div style="display:flex;align-items:center;gap:0.75rem">
            <div class="user-avatar" style="width:28px;height:28px;font-size:0.75rem">
              ${(u.first_name[0]+(u.last_name?.[0]||'')).toUpperCase()}
            </div>
            <span class="data-font">${u.first_name} ${u.last_name||''}</span>
          </div>
        </td>
        <td class="text-muted hide-mobile">${u.email}</td>
        <td><span class="role-badge ${u.role}">${u.role}</span></td>
        <td class="hide-tablet">
          <span class="status-badge ${u.is_active ? 'active' : 'inactive'}">
            <span class="status-dot"></span>${u.is_active ? 'Active' : 'Suspended'}
          </span>
        </td>
        <td>
          <div class="table-actions">
            <label class="toggle-switch" title="Toggle Account Status">
              <input type="checkbox" ${u.is_active ? 'checked' : ''} onchange="toggleUser('${u.id}', this.checked)">
              <span class="toggle-track"><span class="toggle-thumb"></span></span>
            </label>
            <button class="btn btn-ghost btn-xs data-font" onclick="toggleRole('${u.id}', '${u.role}')">
              ${u.role === 'admin' ? 'Demote' : 'Promote'}
            </button>
          </div>
        </td>
      </tr>
    `).join('');
    
    refreshIcons();
    renderPagination('pagination-admin-users', usersRes.metadata.page, usersRes.metadata.total_pages, loadAdminUsers);
  } catch(err) {
    showToast('Admin error: ' + err.message, 'error');
  }
}

/* ─────────────────────────────────────────
   ADMIN LINKS VIEW
──────────────────────────────────────── */
async function loadAdminLinks(page = 1) {
  skeletonRows('tbody-admin-urls', 6, 6);

  try {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: '10',
    });
    
    if (state.adminUrlSearch) {
      params.append('search', state.adminUrlSearch);
    }
    
    if (state.adminUrlSort) {
      params.append('sort', state.adminUrlSort);
    }
    
    const res = await apiFetch(`/admin/urls?${params.toString()}`);
    let urls = res.data || [];
    
    // Filter URLs based on selected tab
    if (state.adminUrlFilter === 'active') {
      urls = urls.filter(u => u.is_active !== false);
    }
    
    // Filter by specific user if selected
    if (state.adminUrlUserId) {
      urls = urls.filter(u => u.user_id === state.adminUrlUserId);
    }
    
    const tbody = $('tbody-admin-urls');
    if (!urls.length) {
      const emptyMsg = state.adminUrlUserId 
        ? 'No routes found for this user.' 
        : (state.adminUrlFilter === 'active' ? 'No active routes.' : 'No routes found.');
      tbody.innerHTML = `<tr><td colspan="6"><div class="empty-state"><i class="ph ph-link ph-lg"></i><h3>No routes</h3><p>${emptyMsg}</p></div></td></tr>`;
      $('pagination-admin-urls').innerHTML = '';
      return;
    }
    
    tbody.innerHTML = urls.map(u => {
      const ownerName = u.user_first_name && u.user_last_name 
        ? `${u.user_first_name} ${u.user_last_name}` 
        : (u.user_id || 'N/A');
      const ownerDisplay = u.user_id 
        ? `<span class="user-link" onclick="filterByUser('${u.user_id}', '${ownerName.replace(/'/g, "\\'")}')">${ownerName}</span>`
        : 'N/A';
      
      return `
      <tr>
        <td><a href="${u.short_url}" target="_blank" class="short-link data-font">${u.short_url.split('/').pop()}</a></td>
        <td class="url-orig-cell hide-mobile" title="${u.original_url}">${u.original_url}</td>
        <td class="data-font text-muted hide-tablet">${ownerDisplay}</td>
        <td class="hide-mobile">
          <span class="status-badge ${u.is_active !== false ? 'active' : 'inactive'}">
            <span class="status-dot"></span>${u.is_active !== false ? 'Live' : 'Offline'}
          </span>
        </td>
        <td class="data-font text-muted">${u.click_count || 0}</td>
        <td>
          <div class="table-actions">
            <label class="toggle-switch" title="Toggle State">
              <input type="checkbox" ${u.is_active !== false ? 'checked' : ''} onchange="toggleAdminUrl('${u.id}', this.checked)">
              <span class="toggle-track"><span class="toggle-thumb"></span></span>
            </label>
            <button class="btn btn-danger btn-xs" onclick="deleteAdminUrl('${u.id}')">
              <i class="ph ph-trash ph-sm"></i>
            </button>
          </div>
        </td>
      </tr>
    `;
    }).join('');
    
    refreshIcons();
    renderPagination('pagination-admin-urls', res.metadata.page, res.metadata.total_pages, loadAdminLinks);
  } catch (err) {
    showToast('Failed to load admin URLs: ' + err.message, 'error');
  }
}

function filterByUser(userId, userName) {
  state.adminUrlUserId = userId;
  loadAdminLinks(1);
  
  // Show filter indicator
  const header = document.querySelector('#page-admin-links .page-header');
  const existingIndicator = document.getElementById('user-filter-indicator');
  if (existingIndicator) existingIndicator.remove();
  
  const indicator = document.createElement('div');
  indicator.id = 'user-filter-indicator';
  indicator.className = 'filter-indicator';
  indicator.innerHTML = `
    <span class="filter-text">Filtering by: <strong>${userName}</strong></span>
    <button class="btn btn-ghost btn-xs" onclick="clearUserFilter()">
      <i class="ph ph-x ph-sm"></i> Clear
    </button>
  `;
  header.appendChild(indicator);
  refreshIcons();
}

function clearUserFilter() {
  state.adminUrlUserId = null;
  const indicator = document.getElementById('user-filter-indicator');
  if (indicator) indicator.remove();
  loadAdminLinks(1);
}

async function toggleAdminUrl(id, active) {
  try {
    await apiFetch(`/admin/urls/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ is_active: active })
    });
    showToast(active ? 'Link is live.' : 'Link disabled.');
    loadAdminLinks(1);
  } catch(err) {
    showToast(err.message, 'error');
  }
}

async function deleteAdminUrl(id) {
  if (!confirm('Are you sure you want to permanently delete this route?')) return;
  try {
    await apiFetch(`/admin/urls/${id}`, { method: 'DELETE' });
    showToast('Link deleted.', 'success');
    loadAdminLinks(1);
  } catch(err) {
    showToast(err.message, 'error');
  }
}

/* ─────────────────────────────────────────
   ACTIONS
──────────────────────────────────────── */
async function toggleUrl(id, active) {
  try {
    await apiFetch(`/urls/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ is_active: active })
    });
    showToast(active ? 'Link is live.' : 'Link disabled.');
    
    // Refresh current view softly
    if (state.currentView === 'dashboard') loadDashboard();
    if (state.currentView === 'my-urls') loadMyUrls();
  } catch(err) {
    showToast(err.message, 'error');
    // revert toggle visually (not implemented for simplicity, reloading instead)
    if (state.currentView === 'dashboard') loadDashboard();
  }
}

async function deleteUrl(id) {
  if (!confirm('Are you sure you want to permanently delete this route?')) return;
  try {
    await apiFetch(`/urls/${id}`, { method: 'DELETE' });
    showToast('Link deleted.', 'success');
    if (state.currentView === 'dashboard') loadDashboard();
    if (state.currentView === 'my-urls')   loadMyUrls();
  } catch(err) {
    showToast(err.message, 'error');
  }
}

async function toggleUser(id, active) {
  try {
    await apiFetch(`/admin/users/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ is_active: active })
    });
    showToast(active ? 'User activated.' : 'User suspended.');
    loadAdminUsers();
  } catch(err) {
    showToast(err.message, 'error');
  }
}

async function toggleRole(id, currentRole) {
  try {
    const newRole = currentRole === 'admin' ? 'user' : 'admin';
    await apiFetch(`/admin/users/${id}/role`, {
      method: 'PUT',
      body: JSON.stringify({ role: newRole })
    });
    showToast('Role updated.');
    loadAdminUsers();
  } catch(err) {
    showToast(err.message, 'error');
  }
}

/* ─────────────────────────────────────────
   PAGINATION
──────────────────────────────────────── */
function renderPagination(containerId, current, total, loadFn) {
  const el = $(containerId);
  if (!el || total <= 1) {
    if (el) el.innerHTML = '';
    return;
  }
  
  let html = '';
  for (let i = 1; i <= total; i++) {
    html += `<button class="page-btn${i === current ? ' active' : ''}" data-page="${i}">${i}</button>`;
  }
  el.innerHTML = html;
  
  el.querySelectorAll('.page-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      loadFn(parseInt(btn.dataset.page));
    });
  });
}

/* ─────────────────────────────────────────
   TAB HANDLING
──────────────────────────────────────── */
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    state.urlFilter = btn.dataset.tab;
    loadMyUrls(1);
  });
});

// Admin tabs
document.querySelectorAll('[data-admin-tab]').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('[data-admin-tab]').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    state.adminUrlFilter = btn.dataset.adminTab;
    loadAdminLinks(1);
  });
});

// User URL sort
$('user-url-sort').addEventListener('change', () => {
  state.userUrlSort = $('user-url-sort').value;
  loadMyUrls(1);
});

// Admin URL sort
$('admin-url-sort').addEventListener('change', () => {
  state.adminUrlSort = $('admin-url-sort').value;
  loadAdminLinks(1);
});

// Admin search
const adminSearchInput = $('admin-url-search');
if (adminSearchInput) {
  let searchTimeout;
  adminSearchInput.addEventListener('input', (e) => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      state.adminUrlSearch = e.target.value;
      loadAdminLinks(1);
    }, 300);
  });
}

// Admin sort
const adminSortSelect = $('admin-url-sort');
if (adminSortSelect) {
  adminSortSelect.addEventListener('change', (e) => {
    state.adminUrlSort = e.target.value;
    loadAdminLinks(1);
  });
}

/* ─────────────────────────────────────────
   MOBILE MENU TOGGLE
──────────────────────────────────────── */
const mobileMenuToggle = $('mobile-menu-toggle');
const sidebar = $('sidebar');
const sidebarOverlay = $('sidebar-overlay');

if (mobileMenuToggle && sidebar && sidebarOverlay) {
  mobileMenuToggle.addEventListener('click', () => {
    sidebar.classList.toggle('open');
    sidebarOverlay.classList.toggle('active');
  });

  sidebarOverlay.addEventListener('click', () => {
    sidebar.classList.remove('open');
    sidebarOverlay.classList.remove('active');
  });

  // Close sidebar when clicking a nav item on mobile
  document.querySelectorAll('.nav-item').forEach(item => {
    item.addEventListener('click', () => {
      if (window.innerWidth < 768) {
        sidebar.classList.remove('open');
        sidebarOverlay.classList.remove('active');
      }
    });
  });
}

/* ─────────────────────────────────────────
   BOOT
──────────────────────────────────────── */
(function init() {
  setTheme(getPreferredTheme());
  document.getElementById('theme-toggle-app')?.addEventListener('click', toggleTheme);
  refreshIcons();
  updateNavbar();
  
  // Handle hash fragments for login/register
  const hash = window.location.hash;
  if (hash === '#register') {
    $('form-login').classList.remove('active');
    $('form-login').classList.add('hidden');
    $('form-register').classList.remove('hidden');
    $('form-register').classList.add('active');
  } else if (hash === '#login') {
    $('form-register').classList.remove('active');
    $('form-register').classList.add('hidden');
    $('form-login').classList.remove('hidden');
    $('form-login').classList.add('active');
  }
  
  if (state.token && state.user) {
    enterApp(state.user, state.token);
  } else {
    // Show login screen implicitly by HTML default
  }
})();
