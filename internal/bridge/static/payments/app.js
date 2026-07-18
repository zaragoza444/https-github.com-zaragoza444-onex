(function () {
  'use strict';

  const API = window.location.origin;
  let config = {};
  let page = null;
  let pageConfig = null;

  const $ = (id) => document.getElementById(id);

  const currencySymbols = { USD: '$', EUR: '€', GBP: '£', AUD: 'A$' };

  async function api(path, opts) {
    const res = await fetch(API + path, {
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      ...opts,
    });
    const text = await res.text();
    if (text.trimStart().startsWith('<!') || text.trimStart().startsWith('<html')) {
      throw new Error('Bridge returned HTML instead of JSON — nginx/domain routing is broken on ' + location.hostname);
    }
    try {
      return JSON.parse(text);
    } catch (_) {
      throw new Error('Invalid JSON from ' + path + ' (HTTP ' + res.status + ')');
    }
  }

  async function refreshDashboardStatus() {
    const el = $('dashboard-status');
    if (!el) return;
    try {
      const st = await api('/bridge/payments/status');
      const fw = st.framework || config.framework || 'zbank';
      const on = st.enabled === true || st.enabled === 1 || st.enabled === '1';
      el.textContent =
        location.hostname +
        ' · ' +
        (fw === 'zbank' ? 'Z Bank' : fw) +
        ' · payments ' +
        (on ? 'LIVE' : 'OFF') +
        (st.provider ? ' · ' + st.provider : '');
      el.classList.toggle('live', !!on);
      el.classList.toggle('down', !on);
    } catch (err) {
      el.textContent = location.hostname + ' · dashboard offline — ' + (err.message || 'bridge down');
      el.classList.add('down');
      el.classList.remove('live');
    }
  }

  function qs(name) {
    return new URLSearchParams(window.location.search).get(name);
  }

  function formatMoney(amount, currency) {
    const sym = currencySymbols[currency] || currency + ' ';
    return sym + parseFloat(amount || 0).toFixed(2);
  }

  function hideError() {
    $('error-panel').classList.add('hidden');
    $('payment-form').classList.remove('hidden');
  }

  function showError(msg) {
    $('error-message').textContent = msg;
    $('payment-form').classList.add('hidden');
    $('error-panel').classList.remove('hidden');
  }

  function updateFeePreview() {
    const amount = parseFloat($('amount').value) || 0;
    const currency = (pageConfig && pageConfig.currency) || 'USD';
    let fee = 0;
    const feeCfg = (pageConfig && pageConfig.processingFee) || config.processingFee || {};
    if (feeCfg.enabled && amount > 0) {
      fee = amount * (parseFloat(feeCfg.percent || 0) / 100) + parseFloat(feeCfg.fixed || 0);
    }
    $('fee-subtotal').textContent = formatMoney(amount, currency);
    $('fee-processing').textContent = formatMoney(fee, currency);
    $('fee-total').textContent = formatMoney(amount + fee, currency);
    $('fee-processing-row').classList.toggle('hidden', !feeCfg.enabled || fee <= 0);
  }

  function renderSuggestedAmounts(amounts) {
    const el = $('suggested-amounts');
    el.innerHTML = '';
    if (!amounts || !amounts.length) return;
    amounts.forEach((amt) => {
      const chip = document.createElement('button');
      chip.type = 'button';
      chip.className = 'chip';
      chip.textContent = formatMoney(amt, pageConfig.currency);
      chip.onclick = () => {
        $('amount').value = amt;
        document.querySelectorAll('.chip').forEach((c) => c.classList.remove('active'));
        chip.classList.add('active');
        updateFeePreview();
      };
      el.appendChild(chip);
    });
  }

  async function loadConfig() {
    config = await api('/bridge/payments/config');
    $('gateway-name').textContent = config.displayName || 'Payment Gateway';
    const frameworkLabel =
      config.framework === 'zbank' ? 'Z Bank' : config.framework === 'nova' ? 'Nova Bank' : 'NSB';
    $('gateway-framework').textContent = frameworkLabel + ' · Visa · Mastercard · Amex';
    applyBrand(config);
  }

  function applyBrand(cfg) {
    const logo = $('brand-logo');
    const icon = $('brand-icon');
    const brand = document.querySelector('.brand');
    const footer = $('footer-brand');
    const isZBank = cfg.framework === 'zbank';
    document.body.classList.toggle('framework-zbank', isZBank);
    if (footer) {
      footer.textContent = 'Powered by OneX Bridge · ' + (isZBank ? 'Z Bank' : frameworkLabel(cfg.framework));
    }
    const url = (cfg.logoUrl || (isZBank ? '/payments/assets/zbank-logo.png' : '')).trim();
    if (url && logo) {
      logo.hidden = false;
      logo.src = url;
      logo.alt = (cfg.displayName || 'Z Bank') + ' logo';
      logo.onerror = function () {
        logo.hidden = true;
        if (icon) icon.style.display = '';
        if (brand) brand.classList.remove('zbank');
      };
      if (icon) icon.style.display = 'none';
      if (brand) brand.classList.add('zbank');
    } else if (logo) {
      logo.hidden = true;
      if (icon) icon.style.display = '';
      if (brand) brand.classList.remove('zbank');
    }
  }

  function frameworkLabel(fw) {
    if (fw === 'zbank') return 'Z Bank';
    if (fw === 'nova') return 'Nova Bank';
    return 'NSB';
  }

  async function loadPage(slug) {
    const data = await api('/bridge/payments/page?slug=' + encodeURIComponent(slug));
    if (data.error) throw new Error(data.error);
    pageConfig = data.page;
    page = slug;

    $('page-title').textContent = pageConfig.title;
    $('page-desc').textContent = pageConfig.description || '';
    $('flow-tag').textContent = pageConfig.flow || 'payment';
    $('currency-symbol').textContent = currencySymbols[pageConfig.currency] || pageConfig.currency;

    const labels = { donation: 'Donate securely', payment: 'Pay securely', collection: 'Submit payment' };
    $('submit-label').textContent = labels[pageConfig.flow] || 'Pay securely';

    if (pageConfig.minAmount) $('amount').min = pageConfig.minAmount;
    renderSuggestedAmounts(pageConfig.suggestedAmounts);

    if (data.settlement) {
      $('settlement-info').textContent =
        'Settlement: ' + data.settlement.label + (data.settlement.bank ? ' (' + data.settlement.bank + ')' : '');
    }
    updateFeePreview();
  }

  async function loadPageList() {
    const data = await api('/bridge/payments/pages');
    const list = $('pages-list');
    list.innerHTML = '';
    (data.pages || []).forEach((p) => {
      const li = document.createElement('li');
      li.innerHTML =
        '<a href="?page=' + encodeURIComponent(p.slug) + '">' + escapeHtml(p.title) + '</a>' +
        '<div class="flow">' + escapeHtml(p.flow) + ' · ' + escapeHtml(p.currency) + '</div>';
      list.appendChild(li);
    });
    $('page-list').classList.remove('hidden');
  }

  function escapeHtml(s) {
    const d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
  }

  function detectCardBrand(num) {
    const n = (num || '').replace(/\D/g, '');
    if (/^3[47]/.test(n)) return 'amex';
    if (/^4/.test(n)) return 'visa';
    if (/^5[1-5]/.test(n) || /^2[2-7]/.test(n)) return 'mastercard';
    return $('card-brand').value;
  }

  function showSuccess(sess) {
    $('payment-form').classList.add('hidden');
    $('success-panel').classList.remove('hidden');
    const dl = $('receipt-details');
    dl.innerHTML =
      '<dt>Amount</dt><dd>' + formatMoney(sess.amount, sess.currency) + '</dd>' +
      (sess.processingFee && parseFloat(sess.processingFee) > 0
        ? '<dt>Processing fee</dt><dd>' + formatMoney(sess.processingFee, sess.currency) + '</dd>'
        : '') +
      '<dt>Total charged</dt><dd>' + formatMoney(sess.totalCharged, sess.currency) + '</dd>' +
      '<dt>Reference</dt><dd>' + escapeHtml(sess.reference || sess.id) + '</dd>' +
      '<dt>Settlement</dt><dd>' + escapeHtml(sess.settlementLabel || sess.settlementDestination) + '</dd>';
    $('success-message').textContent =
      pageConfig && pageConfig.flow === 'donation'
        ? 'Thank you for your generous donation.'
        : 'Your payment has been processed successfully.';
  }

  async function handleSubmit(e) {
    e.preventDefault();
    const btn = $('submit-btn');
    btn.disabled = true;
    btn.querySelector('#submit-label').textContent = 'Processing…';

    try {
      const cardNum = ($('card-number') && $('card-number').value) || '';
      const brand = detectCardBrand(cardNum) || $('card-brand').value;

      const body = {
        pageSlug: page || undefined,
        flow: pageConfig ? pageConfig.flow : 'payment',
        amount: $('amount').value,
        currency: pageConfig ? pageConfig.currency : 'USD',
        payerName: $('payer-name').value,
        payerEmail: $('payer-email').value,
        reference: $('reference').value,
        cardBrand: brand,
      };

      const sess = await api('/bridge/payments/session', {
        method: 'POST',
        body: JSON.stringify(body),
      });
      if (sess.error) throw new Error(sess.error);

      if (config.provider === 'stripe' && config.stripePublicKey && sess.clientSecret) {
        await confirmStripe(sess);
      } else {
        const confirmed = await api('/bridge/payments/confirm', {
          method: 'POST',
          body: JSON.stringify({
            sessionId: sess.id,
            providerRef: sess.providerRef,
            cardBrand: brand,
            cardLast4: cardNum.replace(/\D/g, '').slice(-4),
          }),
        });
        if (confirmed.error) throw new Error(confirmed.error);
        showSuccess(confirmed);
      }
    } catch (err) {
      showError(err.message || 'Payment could not be processed.');
      btn.disabled = false;
      btn.querySelector('#submit-label').textContent = 'Try again';
    }
  }

  async function confirmStripe(sess) {
    if (typeof Stripe === 'undefined') {
      const s = document.createElement('script');
      s.src = 'https://js.stripe.com/v3/';
      document.head.appendChild(s);
      await new Promise((r) => (s.onload = r));
    }
    const stripe = Stripe(config.stripePublicKey);
    const { error, paymentIntent } = await stripe.confirmCardPayment(sess.clientSecret, {
      payment_method: {
        card: { token: 'tok_visa' },
        billing_details: { name: $('payer-name').value, email: $('payer-email').value },
      },
    });
    if (error) throw new Error(error.message);
    if (paymentIntent.status === 'succeeded') {
      const confirmed = await api('/bridge/payments/confirm', {
        method: 'POST',
        body: JSON.stringify({ sessionId: sess.id, providerRef: paymentIntent.id }),
      });
      if (confirmed.error) throw new Error(confirmed.error);
      showSuccess(confirmed);
    }
  }

  async function init() {
    try {
      await loadConfig();
      await refreshDashboardStatus();
      const slug = qs('page');
      if (slug) {
        await loadPage(slug);
      } else {
        await loadPageList();
        $('payment-form').classList.add('hidden');
      }
    } catch (err) {
      showError(err.message || 'Payment dashboard could not load. Check domain → VPS and onex-bridge.');
      await refreshDashboardStatus();
    }
    if ($('amount')) $('amount').addEventListener('input', updateFeePreview);
    if ($('pay-form')) $('pay-form').addEventListener('submit', handleSubmit);
  }

  window.hideError = hideError;
  init();
})();
