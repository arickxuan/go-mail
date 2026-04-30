// === Utilities ===
const API = '/api';

function $(sel) { return document.querySelector(sel); }
function $$(sel) { return document.querySelectorAll(sel); }

async function api(path, opts = {}) {
  const url = API + path;
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json','X-ADMIN-TOKEN': getAdminToken(), ...(opts.headers || {}) },
    ...opts
  });
  let data;
  const text = await res.text();
  try { data = JSON.parse(text); } catch { data = text; }
  if (!res.ok) {
    DelAdminToken();
    const msg = (data && data.error) || (typeof data === 'string' ? data : res.statusText);
    throw new Error(msg);
  }
  return data;
}

function showToast(message, type = 'success') {
  const el = document.createElement('div');
  el.className = `toast toast-${type}`;
  el.textContent = message;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), 3000);
}

function formatDate(d) {
  if (!d) return '';
  const dt = new Date(d);
  const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
  return `${months[dt.getMonth()]} ${String(dt.getDate()).padStart(2,'0')} ${String(dt.getHours()).padStart(2,'0')}:${String(dt.getMinutes()).padStart(2,'0')}`;
}

function formatDateFull(d) {
  if (!d) return '';
  const dt = new Date(d);
  const days = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'];
  const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
  const pad = n => String(n).padStart(2,'0');
  const off = -dt.getTimezoneOffset();
  const sign = off >= 0 ? '+' : '-';
  const h = pad(Math.floor(Math.abs(off)/60));
  const m = pad(Math.abs(off)%60);
  return `${days[dt.getDay()]}, ${pad(dt.getDate())} ${months[dt.getMonth()]} ${dt.getFullYear()} ${pad(dt.getHours())}:${pad(dt.getMinutes())}:${pad(dt.getSeconds())} ${sign}${h}${m}`;
}

function normalizePassword(provider, password) {
  password = (password || '').trim();
  if (provider === 'yahoo' || provider === 'gmail') {
    return password.split(/\s+/).join('');
  }
  return password;
}

function escapeHtml(str) {
  return String(str).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

// === Routing ===
const routes = [
  { path: /^\/$/, render: renderDashboard },
  { path: /^\/accounts$/, render: renderAccounts },
  { path: /^\/accounts\/add$/, render: renderAddAccount },
  { path: /^\/accounts\/([^\/]+)\/edit$/, render: (_, id) => renderEditAccount(id) },
  { path: /^\/accounts\/([^\/]+)\/inbox$/, render: (_, id) => renderInbox(id) },
  { path: /^\/accounts\/([^\/]+)\/mail\/([^\/]+)$/, render: (_, id, uid) => renderEmailDetail(id, uid) },
  { path: /^\/import$/, render: renderImport },
];

function matchRoute(path) {
  for (const r of routes) {
    const m = path.match(r.path);
    if (m) return () => r.render(m, ...m.slice(1));
  }
  return renderDashboard;
}

function navigate(path, push = true) {
  if (push) history.pushState({}, '', path);
  updateNavActive(path);
  const app = $('#app');
  app.innerHTML = '<div class="loading-state"><div class="spinner"></div></div>';
  matchRoute(path)();
}

function updateNavActive(path) {
  $$('[data-nav]').forEach(el => {
    el.classList.toggle('active', el.getAttribute('href') === path);
  });
}

window.addEventListener('popstate', () => {
  navigate(location.pathname, false);
});

document.addEventListener('click', e => {
  const a = e.target.closest('a[href]');
  if (!a) return;
  const href = a.getAttribute('href');
  if (href && href.startsWith('/') && !href.startsWith('//')) {
    e.preventDefault();
    navigate(href);
  }
});

// === Page: Dashboard ===
async function renderDashboard() {
  try {
    const accounts = await api('/accounts');
    const total = accounts.length;
    const imap = accounts.filter(a => a.account_type === 'imap').length;
    const graph = accounts.filter(a => a.account_type === 'graph').length;
    const pop3 = accounts.filter(a => a.account_type === 'pop3').length;

    $('#app').innerHTML = `
      <div>
        <h2>Dashboard</h2>
        <div class="stats-grid">
          <div class="stat-card"><div class="stat-number">${total}</div><div class="stat-label">Total Accounts</div></div>
          <div class="stat-card"><div class="stat-number">${imap}</div><div class="stat-label">IMAP</div></div>
          <div class="stat-card"><div class="stat-number">${graph}</div><div class="stat-label">Graph API</div></div>
          <div class="stat-card"><div class="stat-number">${pop3}</div><div class="stat-label">POP3</div></div>
        </div>
        <div class="card">
          <div class="card-header"><h3>Connected Accounts</h3><a href="/accounts/add" class="btn btn-primary btn-sm">+ Add Account</a></div>
          ${accounts.length ? `
          <div class="table-wrapper"><table><thead><tr><th>Account</th><th>Type</th><th>Provider</th></tr></thead><tbody>
          ${accounts.map(a => `
            <tr class="email-row" onclick="navigate('/accounts/${a.id}/inbox')">
              <td><div style="font-weight:500">${escapeHtml(a.label)}</div><div style="font-size:12px;color:var(--text-muted)">${escapeHtml(a.email)}</div></td>
              <td><span class="badge badge-${a.account_type}">${a.account_type}</span></td>
              <td>${a.provider_type || '-'}</td>
            </tr>
          `).join('')}
          </tbody></table></div>` : `
          <div style="text-align:center;padding:40px 0;color:var(--text-muted)">
            <div style="font-size:48px;margin-bottom:12px;opacity:0.3">📭</div>
            <p>No accounts configured yet.</p>
            <p style="margin-top:8px"><a href="/accounts/add" class="btn btn-primary btn-sm">Add your first account</a></p>
          </div>`}
        </div>
      </div>`;
  } catch (err) {
    $('#app').innerHTML = `<div class="alert alert-error"><span class="alert-icon">⚠</span><div>${escapeHtml(err.message)}</div></div>`;
  }
}

// === Page: Accounts ===
async function renderAccounts() {
  try {
    const accounts = await api('/accounts');
    $('#app').innerHTML = `
      <div>
        <div class="card-header" style="padding:0 0 16px 0; margin-bottom:20px"><h2>Accounts</h2><a href="/accounts/add" class="btn btn-primary">+ Add Account</a></div>
        <div class="card">
          ${accounts.length ? `
          <div class="table-wrapper"><table><thead><tr><th>Account</th><th>Type</th><th>Provider</th><th>Status</th><th>Actions</th></tr></thead><tbody>
          ${accounts.map(a => `
            <tr>
              <td><div style="font-weight:500">${escapeHtml(a.label)}</div><div style="font-size:12px;color:var(--text-muted)">${escapeHtml(a.email)}</div></td>
              <td><span class="badge badge-${a.account_type}">${a.account_type}</span></td>
              <td><span class="badge badge-neutral">${a.provider_type || '-'}</span></td>
              <td id="check-${a.id}"><button class="btn btn-outline btn-sm" onclick="checkAccount('${a.id}')">Check</button></td>
              <td>
                <div class="btn-group">
                  <a href="/accounts/${a.id}/inbox" class="btn btn-primary btn-sm">Inbox</a>
                  <a href="/accounts/${a.id}/edit" class="btn btn-outline btn-sm">Edit</a>
                  <button class="btn btn-danger btn-sm" onclick="deleteAccount('${a.id}', '${escapeHtml(a.email)}')">Delete</button>
                </div>
              </td>
            </tr>
          `).join('')}
          </tbody></table></div>` : `
          <div style="text-align:center;padding:40px 0;color:var(--text-muted)">
            <div style="font-size:48px;margin-bottom:12px;opacity:0.3">📭</div>
            <p>No accounts yet.</p>
            <p style="margin-top:8px"><a href="/accounts/add" class="btn btn-primary">Add your first account</a></p>
          </div>`}
        </div>
      </div>`;
  } catch (err) {
    $('#app').innerHTML = `<div class="alert alert-error"><span class="alert-icon">⚠</span><div>${escapeHtml(err.message)}</div></div>`;
  }
}

window.checkAccount = async function(id) {
  const cell = $(`#check-${id}`);
  cell.innerHTML = '<div class="spinner"></div>';
  try {
    const res = await api(`/accounts/${id}/check`, { method: 'POST' });
    if (res.status === 'ok') {
      cell.innerHTML = `<span class="badge badge-success">✓ Connected</span> <button class="btn btn-outline btn-sm" style="margin-left:6px" onclick="checkAccount('${id}')">Retry</button>`;
    } else {
      cell.innerHTML = `<span class="badge badge-danger" title="${escapeHtml(res.error || '')}">✗ Failed</span>
        <span style="font-size:11px;color:var(--text-muted);margin-left:4px;max-width:140px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;display:inline-block;vertical-align:middle">${escapeHtml(res.error || '')}</span>
        <button class="btn btn-outline btn-sm" style="margin-left:6px" onclick="checkAccount('${id}')">Retry</button>`;
    }
  } catch (err) {
    cell.innerHTML = `<span class="badge badge-danger">✗ Failed</span> <button class="btn btn-outline btn-sm" style="margin-left:6px" onclick="checkAccount('${id}')">Retry</button>`;
  }
};

window.deleteAccount = async function(id, email) {
  if (!confirm(`Delete account ${email}?`)) return;
  try {
    await api(`/accounts/${id}`, { method: 'DELETE' });
    showToast('Account deleted', 'success');
    navigate('/accounts');
  } catch (err) {
    showToast(err.message, 'error');
  }
};

// === Page: Add Account ===
function renderAddAccount() {
  $('#app').innerHTML = `
    <div>
      <h2>Add Account</h2>
      <div class="card">
        <form id="add-form">
          <div class="form-row">
            <div class="form-group">
              <label>Account Type</label>
              <select name="account_type" id="accType" onchange="toggleTypeFields(this.value)">
                <option value="imap">IMAP</option>
                <option value="pop3">POP3</option>
                <option value="graph">Microsoft Graph API</option>
              </select>
            </div>
            <div class="form-group">
              <label>Provider</label>
              <select name="provider_type">
                <option value="gmail">Gmail</option>
                <option value="yahoo">Yahoo Mail</option>
                <option value="qq">QQ Mail</option>
                <option value="163">163 Mail</option>
                <option value="outlook">Outlook</option>
                <option value="custom">Custom</option>
              </select>
            </div>
          </div>
          <div class="form-row">
            <div class="form-group"><label>Label</label><input type="text" name="label" placeholder="My Account" required></div>
            <div class="form-group"><label>Email</label><input type="email" name="email" placeholder="user@example.com" required></div>
          </div>
          <div id="imapFields" class="type-fields" style="display:block">
            <div class="form-row"><div class="form-group"><label>Password <small>(App Password)</small></label><input type="password" name="password" placeholder="Password or app-specific password"></div></div>
            <div class="form-row">
              <div class="form-group"><label>IMAP Server</label><input type="text" name="imap_server" placeholder="Leave empty for auto-detect"></div>
              <div class="form-group"><label>Port</label><input type="number" name="imap_port" placeholder="993"></div>
            </div>
            <div class="form-group"><label class="form-checkbox"><input type="checkbox" name="imap_tls" checked> Use TLS</label></div>
          </div>
          <div id="pop3Fields" class="type-fields" style="display:none">
            <div class="form-row"><div class="form-group"><label>Password</label><input type="password" name="password" placeholder="Password"></div></div>
            <div class="form-row">
              <div class="form-group"><label>POP3 Server</label><input type="text" name="pop3_server" placeholder="Leave empty for auto-detect"></div>
              <div class="form-group"><label>Port</label><input type="number" name="pop3_port" placeholder="995"></div>
            </div>
            <div class="form-group"><label class="form-checkbox"><input type="checkbox" name="pop3_tls" checked> Use TLS</label></div>
          </div>
          <div id="graphFields" class="type-fields" style="display:none">
            <div class="form-row">
              <div class="form-group"><label>Client ID</label><input type="text" name="client_id" placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"></div>
              <div class="form-group"><label>Tenant ID</label><input type="text" name="tenant_id" placeholder="common" value="common"></div>
            </div>
            <div class="form-group"><label>Refresh Token</label><input type="password" name="refresh_token" placeholder="Refresh token from OAuth flow"></div>
          </div>
          <div class="btn-group" style="margin-top:4px">
            <button type="submit" class="btn btn-primary btn-lg">Add Account</button>
            <a href="/accounts" class="btn btn-outline btn-lg">Cancel</a>
          </div>
        </form>
      </div>
    </div>`;

  $('#add-form').addEventListener('submit', async e => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const type = fd.get('account_type');
    const provider = fd.get('provider_type');
    const body = {
      label: fd.get('label'),
      email: fd.get('email'),
      account_type: type,
      provider_type: provider,
    };
    if (type === 'imap') {
      body.password = normalizePassword(provider, fd.get('password'));
      body.imap_server = fd.get('imap_server') || undefined;
      body.imap_port = fd.get('imap_port') ? parseInt(fd.get('imap_port')) : undefined;
      body.imap_tls = fd.get('imap_tls') === 'on';
    } else if (type === 'pop3') {
      body.password = normalizePassword(provider, fd.get('password'));
      body.pop3_server = fd.get('pop3_server') || undefined;
      body.pop3_port = fd.get('pop3_port') ? parseInt(fd.get('pop3_port')) : undefined;
      body.pop3_tls = fd.get('pop3_tls') === 'on';
    } else if (type === 'graph') {
      body.client_id = fd.get('client_id');
      body.tenant_id = fd.get('tenant_id');
      body.refresh_token = fd.get('refresh_token');
    }
    try {
      await api('/accounts', { method: 'POST', body: JSON.stringify(body) });
      showToast('Account added');
      navigate('/accounts');
    } catch (err) {
      showToast(err.message, 'error');
    }
  });
}

window.toggleTypeFields = function(type) {
  $$('.type-fields').forEach(el => el.style.display = 'none');
  const el = $(`#${type}Fields`);
  if (el) el.style.display = 'block';
};

// === Page: Edit Account ===
async function renderEditAccount(id) {
  $('#app').innerHTML = '<div class="loading-state"><div class="spinner"></div></div>';
  try {
    const a = await api(`/accounts/${id}`);
    let fields = '';
    if (a.account_type === 'imap') {
      fields = `
        <div class="form-row"><div class="form-group"><label>Password <small>(leave empty to keep)</small></label><input type="password" name="password" placeholder="New password"></div></div>
        <div class="form-row">
          <div class="form-group"><label>IMAP Server</label><input type="text" name="imap_server" value="${escapeHtml(a.imap_server || '')}"></div>
          <div class="form-group"><label>Port</label><input type="number" name="imap_port" value="${a.imap_port || ''}"></div>
        </div>
        <div class="form-group"><label class="form-checkbox"><input type="checkbox" name="imap_tls" ${a.imap_tls ? 'checked' : ''}> Use TLS</label></div>`;
    } else if (a.account_type === 'graph') {
      fields = `
        <div class="form-row">
          <div class="form-group"><label>Client ID</label><input type="text" name="client_id" value="${escapeHtml(a.client_id || '')}"></div>
          <div class="form-group"><label>Tenant ID</label><input type="text" name="tenant_id" value="${escapeHtml(a.tenant_id || '')}"></div>
        </div>
        <div class="form-group"><label>Refresh Token <small>(leave empty to keep)</small></label><input type="password" name="refresh_token" placeholder="New refresh token"></div>`;
    } else if (a.account_type === 'pop3') {
      fields = `
        <div class="form-row"><div class="form-group"><label>Password <small>(leave empty to keep)</small></label><input type="password" name="password" placeholder="New password"></div></div>
        <div class="form-row">
          <div class="form-group"><label>POP3 Server</label><input type="text" name="pop3_server" value="${escapeHtml(a.pop3_server || '')}"></div>
          <div class="form-group"><label>Port</label><input type="number" name="pop3_port" value="${a.pop3_port || ''}"></div>
        </div>
        <div class="form-group"><label class="form-checkbox"><input type="checkbox" name="pop3_tls" ${a.pop3_tls ? 'checked' : ''}> Use TLS</label></div>`;
    }

    $('#app').innerHTML = `
      <div>
        <h2>Edit Account</h2>
        <div class="card">
          <form id="edit-form">
            <div class="form-row">
              <div class="form-group"><label>Label</label><input type="text" name="label" value="${escapeHtml(a.label)}" required></div>
              <div class="form-group"><label>Email</label><input type="email" name="email" value="${escapeHtml(a.email)}" required></div>
            </div>
            ${fields}
            <div class="btn-group" style="margin-top:4px">
              <button type="submit" class="btn btn-primary btn-lg">Save Changes</button>
              <a href="/accounts" class="btn btn-outline btn-lg">Cancel</a>
            </div>
          </form>
        </div>
      </div>`;

    $('#edit-form').addEventListener('submit', async e => {
      e.preventDefault();
      const fd = new FormData(e.target);
      const body = {
        label: fd.get('label'),
        email: fd.get('email'),
        account_type: a.account_type,
        provider_type: a.provider_type,
      };
      if (a.account_type === 'imap') {
        const pw = fd.get('password');
        if (pw) body.password = normalizePassword(a.provider_type, pw);
        body.imap_server = fd.get('imap_server') || undefined;
        body.imap_port = fd.get('imap_port') ? parseInt(fd.get('imap_port')) : undefined;
        body.imap_tls = fd.get('imap_tls') === 'on';
      } else if (a.account_type === 'graph') {
        body.client_id = fd.get('client_id');
        body.tenant_id = fd.get('tenant_id');
        const rt = fd.get('refresh_token');
        if (rt) body.refresh_token = rt;
      } else if (a.account_type === 'pop3') {
        const pw = fd.get('password');
        if (pw) body.password = normalizePassword(a.provider_type, pw);
        body.pop3_server = fd.get('pop3_server') || undefined;
        body.pop3_port = fd.get('pop3_port') ? parseInt(fd.get('pop3_port')) : undefined;
        body.pop3_tls = fd.get('pop3_tls') === 'on';
      }
      try {
        await api(`/accounts/${id}`, { method: 'PUT', body: JSON.stringify(body) });
        showToast('Account updated');
        navigate('/accounts');
      } catch (err) {
        showToast(err.message, 'error');
      }
    });
  } catch (err) {
    $('#app').innerHTML = `<div class="alert alert-error"><span class="alert-icon">⚠</span><div>${escapeHtml(err.message)}</div></div>`;
  }
}

// === Page: Inbox ===
async function renderInbox(id) {
  const params = new URLSearchParams(location.search);
  const page = parseInt(params.get('page')) || 1;
  $('#app').innerHTML = '<div class="loading-state"><div class="spinner"></div></div>';
  try {
    const data = await api(`/accounts/${id}/inbox?page=${page}`);
    const a = data.account || {};
    const emails = data.emails || [];
    const total = data.total || 0;

    $('#app').innerHTML = `
      <div>
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:20px">
          <div>
            <a href="/accounts" class="btn btn-outline btn-sm" style="margin-bottom:8px;display:inline-block">← Back</a>
            <h2 style="margin-bottom:0">${escapeHtml(a.email || '')}</h2>
            <div style="font-size:13px;color:var(--text-muted);margin-top:2px">${emails.length} messages</div>
          </div>
          <div style="display:flex;align-items:center;gap:8px">
            <span class="badge badge-${a.account_type}">${a.account_type}</span>
            <span class="badge badge-neutral">${a.provider_type || '-'}</span>
          </div>
        </div>
        <div class="card" style="padding:0">
          ${emails.length ? `
          <div class="table-wrapper" style="border:none"><table><thead><tr><th style="width:40px"></th><th>From</th><th>Subject</th><th style="width:130px">Date</th></tr></thead><tbody>
          ${emails.map(e => `
            <tr class="email-row" onclick="navigate('/accounts/${id}/mail/${e.open_ref || e.uid}')">
              <td>${e.seen ? '' : '<span style="display:inline-block;width:8px;height:8px;background:var(--primary);border-radius:50%"></span>'}</td>
              <td class="email-from">${escapeHtml(e.from || '')}</td>
              <td class="email-subject">${escapeHtml(e.subject || '(no subject)')}</td>
              <td class="email-date">${formatDate(e.date)}</td>
            </tr>
          `).join('')}
          </tbody></table></div>` : `
          <div style="text-align:center;padding:48px 0;color:var(--text-muted)">
            <div style="font-size:48px;margin-bottom:12px;opacity:0.3">📭</div>
            <p>Inbox is empty.</p>
          </div>`}
        </div>
        ${total > 50 ? `
        <div class="pagination">
          <button class="btn btn-outline btn-sm" ${page <= 1 ? 'disabled' : ''} onclick="navigate('/accounts/${id}/inbox?page=${page-1}')">Previous</button>
          <span class="page-info">Page ${page}</span>
          <button class="btn btn-outline btn-sm" ${emails.length < 50 ? 'disabled' : ''} onclick="navigate('/accounts/${id}/inbox?page=${page+1}')">Next</button>
        </div>` : ''}
      </div>`;
  } catch (err) {
    $('#app').innerHTML = `
      <div>
        <a href="/accounts" class="btn btn-outline btn-sm" style="margin-bottom:16px">← Back to Accounts</a>
        <div class="alert alert-error">
          <span class="alert-icon">⚠</span>
          <div>
            <strong>Failed to load inbox</strong><br>
            <span style="font-size:12px;opacity:0.8">${escapeHtml(err.message)}</span>
            <div style="font-size:12px;margin-top:8px;opacity:0.8">Check the account email, provider, and app password settings, then update the account and try again.</div>
          </div>
        </div>
      </div>`;
  }
}

// === Page: Email Detail ===
async function renderEmailDetail(id, uid) {
  $('#app').innerHTML = '<div class="loading-state"><div class="spinner"></div></div>';
  try {
    const e = await api(`/accounts/${id}/mail/${uid}`);
    const attachments = (e.attachments || []).map(att => `
      <li><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/></svg>
      ${escapeHtml(att.filename)} (${att.size} bytes, ${att.mime_type || '-'})</li>
    `).join('');

    $('#app').innerHTML = `
      <div>
        <div style="display:flex; align-items:flex-start; justify-content:space-between; margin-bottom:20px">
          <div>
            <h2 style="margin-bottom:4px">${escapeHtml(e.subject || '')}</h2>
            <div style="font-size:13px;color:var(--text-muted)">${escapeHtml(e.from || '')}</div>
          </div>
          <a href="/accounts/${id}/inbox" class="btn btn-outline">← Back to Inbox</a>
        </div>
        <div class="card">
          <dl class="email-meta">
            <dt>From</dt><dd>${escapeHtml(e.from || '')}</dd>
            <dt>To</dt><dd>${escapeHtml(e.to || '')}</dd>
            ${e.cc ? `<dt>Cc</dt><dd>${escapeHtml(e.cc)}</dd>` : ''}
            <dt>Date</dt><dd>${formatDateFull(e.date)}</dd>
          </dl>
          ${attachments ? `<div class="attachments"><strong style="font-size:13px">Attachments (${e.attachments.length})</strong><ul>${attachments}</ul></div>` : ''}
          <div class="email-body">${e.html_body ? e.html_body : `<pre>${escapeHtml(e.text_body || '')}</pre>`}</div>
        </div>
      </div>`;
  } catch (err) {
    $('#app').innerHTML = `<div class="alert alert-error"><span class="alert-icon">⚠</span><div>${escapeHtml(err.message)}</div></div>`;
  }
}

// === Page: Import ===
function renderImport() {
  $('#app').innerHTML = `
    <div>
      <h2>Import Accounts</h2>
      <div class="import-format">
        <strong>Format Guide</strong>
        <div style="margin-top:8px;display:flex;flex-direction:column;gap:6px">
          <div><strong>Graph API:</strong> <code>email----password----client_id----token</code></div>
          <div><strong>IMAP:</strong> <code>email----password----refresh_token----client_id----provider</code> (outlook/yahoo/gmail/qq/163)</div>
          <div style="margin-top:4px;color:var(--text-muted);font-size:12px">One account per line · Fields separated by <code>----</code> (4 dashes)</div>
        </div>
      </div>
      <div class="card">
        <form id="import-form">
          <div class="form-row" style="margin-bottom:20px">
            <div class="form-group">
              <label>Import Type</label>
              <select name="import_type">
                <option value="imap">IMAP</option>
                <option value="graph">Microsoft Graph API</option>
              </select>
            </div>
          </div>
          <div class="form-group">
            <label>Data <small>(paste or type account data)</small></label>
            <textarea name="data" rows="10" placeholder="email@example.com----password----refresh_token----client_id----gmail"></textarea>
          </div>
          <div style="display:flex;align-items:center;gap:12px">
            <button type="submit" class="btn btn-primary btn-lg">Import Accounts</button>
          </div>
        </form>
        <div id="import-result" style="margin-top:20px"></div>
      </div>
    </div>`;

  $('#import-form').addEventListener('submit', async e => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      const res = await api('/import', {
        method: 'POST',
        body: JSON.stringify({ type: fd.get('import_type'), data: fd.get('data') })
      });
      const el = $('#import-result');
      el.innerHTML = `
        <div style="display:flex;align-items:center;gap:16px;margin-bottom:12px">
          <div class="stat-card" style="flex:1"><div class="stat-number">${res.total}</div><div class="stat-label">Total</div></div>
          <div class="stat-card" style="flex:1"><div class="stat-number" style="color:var(--success)">${res.success}</div><div class="stat-label">Succeeded</div></div>
          <div class="stat-card" style="flex:1"><div class="stat-number" style="color:var(--danger)">${res.failed}</div><div class="stat-label">Failed</div></div>
        </div>
        ${res.errors.length ? `<div class="alert alert-error"><span class="alert-icon">⚠</span><div><ul style="margin:4px 0 0 16px;font-size:13px;line-height:1.8">${res.errors.map(err => `<li>${escapeHtml(err)}</li>`).join('')}</ul></div></div>` : ''}
        ${res.success > 0 ? `<a href="/accounts" class="btn btn-primary btn-sm">View Accounts</a>` : ''}`;
      if (res.success > 0) showToast(`Imported ${res.success} accounts`);
    } catch (err) {
      showToast(err.message, 'error');
    }
  });
}

function getAdminToken(){
  let token = localStorage.getItem('x_admin_token')
  if (!toek){
    // 302 to /login
    window.location.href = '/login';
    return null;
  }
  return token;
}



function setAdminToken(token){
  localStorage.setItem('x_admin_token', token);
}

function DelAdminToken(){
  localStorage.removeItem('x_admin_token');
}

// === Init ===
navigate(location.pathname, false);
