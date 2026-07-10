(function () {
  'use strict';

  const API = window.location.origin;
  let dashboard = null;
  let allSessions = [];

  const $ = (id) => document.getElementById(id);

  async function api(path) {
    const res = await fetch(API + path);
    if (!res.ok) throw new Error('Request failed: ' + res.status);
    return res.json();
  }

  function escapeHtml(s) {
    const d = document.createElement('div');
    d.textContent = s == null ? '' : String(s);
    return d.innerHTML;
  }

  function formatMoney(amount, currency) {
    const sym = { USD: '$', EUR: '€', GBP: '£', AUD: 'A$' }[currency] || currency + ' ';
    return sym + parseFloat(amount || 0).toFixed(2);
  }

  function formatTime(ts) {
    if (!ts) return '—';
    return new Date(ts * 1000).toLocaleString();
  }

  function frameworkLabel(fw) {
    if (fw === 'zbank') return 'Z Bank';
    if (fw === 'nova') return 'Nova Bank';
    return fw || 'NSB';
  }

  function renderStatusMeta(status) {
    const items = [
      ['Framework', frameworkLabel(status.framework)],
      ['Provider', status.provider || '—'],
      ['Stripe configured', status.stripeConfigured ? 'Yes' : 'No'],
      ['Stripe live ready', status.stripeLiveReady ? 'Yes' : 'No'],
      ['Active pages', status.activePages],
      ['Settlement destinations', status.settlementDestinations],
      ['Default settlement', status.defaultSettlement || '—'],
      ['Cards', (status.acceptedCards || []).join(', ') || '—'],
    ];
    const fee = status.processingFee || {};
    if (fee.enabled) {
      items.push(['Processing fee', fee.percent + '% + ' + fee.fixed + ' ' + (fee.currency || 'USD')]);
    } else {
      items.push(['Processing fee', 'Disabled']);
    }
    $('status-meta').innerHTML = items
      .map(([k, v]) => '<dt>' + escapeHtml(k) + '</dt><dd>' + escapeHtml(String(v)) + '</dd>')
      .join('');
  }

  function renderVolume(stats) {
    const vol = stats.volumeByCurrency || {};
    const fees = stats.feesByCurrency || {};
    const keys = new Set([...Object.keys(vol), ...Object.keys(fees)]);
    const el = $('volume-summary');
    if (!keys.size) {
      el.innerHTML = '<div class="summary-row"><span class="label">No settled volume yet</span></div>';
      return;
    }
    el.innerHTML = [...keys]
      .map((cur) => {
        const v = vol[cur] || '0.00';
        const f = fees[cur] || '0.00';
        return (
          '<div class="summary-row">' +
          '<span class="label">' + escapeHtml(cur) + ' volume</span>' +
          '<span class="value">' + escapeHtml(formatMoney(v, cur)) +
          (parseFloat(f) > 0 ? ' <span style="color:var(--muted);font-weight:400">(fees ' + escapeHtml(formatMoney(f, cur)) + ')</span>' : '') +
          '</span></div>'
        );
      })
      .join('');
  }

  function renderFlows(stats) {
    const flows = stats.byFlow || {};
    const el = $('flow-summary');
    const keys = Object.keys(flows);
    if (!keys.length) {
      el.innerHTML = '<span class="chip">No sessions yet</span>';
      return;
    }
    el.innerHTML = keys
      .map((f) => '<span class="chip">' + escapeHtml(f) + ': ' + flows[f] + '</span>')
      .join('');
  }

  function renderSettlements(stats, destinations) {
    const bySettle = stats.bySettlement || {};
    const el = $('settlement-summary');
    const destMap = {};
    (destinations || []).forEach((d) => { destMap[d.id] = d; });

    const ids = new Set([
      ...Object.keys(bySettle),
      ...(destinations || []).map((d) => d.id),
    ]);

    if (!ids.size) {
      el.innerHTML = '<div class="summary-row"><span class="label">No destinations configured</span></div>';
      return;
    }

    el.innerHTML = [...ids]
      .map((id) => {
        const s = bySettle[id];
        const d = destMap[id];
        const label = (s && s.label) || (d && d.label) || id;
        const type = d ? d.type : '';
        const bank = d && d.bankName ? ' · ' + d.bankName : '';
        const count = s ? s.count : 0;
        const vol = s ? formatMoney(s.volume, s.currency) : '—';
        return (
          '<div class="summary-row">' +
          '<span class="label">' + escapeHtml(label) +
          (type ? ' <span style="opacity:0.7">(' + escapeHtml(type) + bank + ')</span>' : '') +
          '</span>' +
          '<span class="value">' + count + ' · ' + escapeHtml(vol) + '</span></div>'
        );
      })
      .join('');
  }

  function renderPages(pages) {
    const el = $('pages-list');
    if (!pages || !pages.length) {
      el.innerHTML = '<li>No active pages</li>';
      return;
    }
    el.innerHTML = pages
      .map(
        (p) =>
          '<li><a href="../?page=' + encodeURIComponent(p.slug) + '">' +
          escapeHtml(p.title) + '</a> · ' + escapeHtml(p.flow) + ' · ' + escapeHtml(p.currency) + '</li>'
      )
      .join('');
  }

  function renderSessions(sessions) {
    const tbody = $('sessions-body');
    if (!sessions.length) {
      tbody.innerHTML = '<tr><td colspan="7" class="empty">No sessions match filters</td></tr>';
      return;
    }
    tbody.innerHTML = sessions
      .map((s) => {
        const ref = s.reference || s.id;
        const statusClass = (s.status || '').toLowerCase();
        return (
          '<tr>' +
          '<td>' + escapeHtml(formatTime(s.createdAt)) + '</td>' +
          '<td title="' + escapeHtml(s.id) + '">' + escapeHtml(ref) + '</td>' +
          '<td>' + escapeHtml(s.flow || '—') + '</td>' +
          '<td>' + escapeHtml(formatMoney(s.totalCharged || s.amount, s.currency)) + '</td>' +
          '<td>' + escapeHtml(s.processingFee && parseFloat(s.processingFee) > 0 ? formatMoney(s.processingFee, s.currency) : '—') + '</td>' +
          '<td>' + escapeHtml(s.settlementLabel || s.settlementDestination || '—') + '</td>' +
          '<td><span class="badge ' + escapeHtml(statusClass) + '">' + escapeHtml(s.status || '—') + '</span></td>' +
          '</tr>'
        );
      })
      .join('');
  }

  function applyFilters() {
    const flow = $('filter-flow').value;
    const status = $('filter-status').value;
    const filtered = allSessions.filter((s) => {
      if (flow && s.flow !== flow) return false;
      if (status && s.status !== status) return false;
      return true;
    });
    renderSessions(filtered);
  }

  function renderDashboard(data) {
    dashboard = data;
    const status = data.status || {};
    const stats = data.stats || {};

    $('gateway-name').textContent = status.displayName || 'Payment Gateway';
    $('gateway-subtitle').textContent = frameworkLabel(status.framework) + ' · Admin dashboard';

    const pill = $('gateway-status');
    if (status.enabled) {
      pill.textContent = 'Live';
      pill.classList.remove('off');
    } else {
      pill.textContent = 'Disabled';
      pill.classList.add('off');
    }

    $('stat-total').textContent = stats.totalSessions || 0;
    const byStatus = stats.byStatus || {};
    $('stat-succeeded').textContent = byStatus.succeeded || 0;
    $('stat-pending').textContent = (byStatus.pending || 0) + (byStatus.processing || 0);
    $('stat-failed').textContent = byStatus.failed || 0;

    renderStatusMeta(status);
    renderVolume(stats);
    renderFlows(stats);
    renderSettlements(stats, data.destinations);
    renderPages(data.pages);

    allSessions = stats.recentSessions || [];
    applyFilters();

    $('last-updated').textContent = stats.lastUpdated
      ? 'Updated ' + formatTime(stats.lastUpdated)
      : 'No session data yet';
  }

  async function load() {
    try {
      const data = await api('/bridge/payments/dashboard');
      if (data.error) throw new Error(data.error);
      renderDashboard(data);
    } catch (err) {
      $('sessions-body').innerHTML =
        '<tr><td colspan="7" class="empty">Could not load dashboard: ' + escapeHtml(err.message) + '</td></tr>';
      $('gateway-status').textContent = 'Error';
      $('gateway-status').classList.add('off');
    }
  }

  $('refresh-btn').addEventListener('click', load);
  $('filter-flow').addEventListener('change', applyFilters);
  $('filter-status').addEventListener('change', applyFilters);

  load();
  setInterval(load, 30000);
})();
