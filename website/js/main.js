(function () {
  const cfg = window.ONEX_SITE || {};
  const production = cfg.productionUrl || 'https://onexproduction.com';
  const walletPath = cfg.walletPath || '/wallet/';
  const walletUrl = cfg.walletUrl || (production.replace(/\/$/, '') + walletPath);

  document.querySelectorAll('[data-wallet]').forEach(el => {
    el.href = walletUrl;
  });
  document.querySelectorAll('[data-explorer]').forEach(el => {
    el.href = production.replace(/\/$/, '') + '/explorer/';
  });

  const navToggle = document.querySelector('.nav-toggle');
  const navLinks = document.querySelector('.nav-links');
  if (navToggle && navLinks) {
    navToggle.addEventListener('click', () => navLinks.classList.toggle('open'));
  }

  const toast = document.getElementById('toast');
  function showToast(msg) {
    if (!toast) return;
    toast.textContent = msg;
    toast.classList.add('show');
    setTimeout(() => toast.classList.remove('show'), 2200);
  }

  document.querySelectorAll('[data-copy]').forEach(btn => {
    btn.addEventListener('click', () => {
      const text = btn.dataset.copy;
      navigator.clipboard?.writeText(text).then(() => showToast('Copied to clipboard'));
    });
  });

  const form = document.getElementById('contact-form');
  if (form) {
    form.addEventListener('submit', e => {
      e.preventDefault();
      const fd = new FormData(form);
      const to = fd.get('department') || 'hello@onexproduction.com';
      const subject = encodeURIComponent('[OneX] ' + (fd.get('subject') || 'Website inquiry'));
      const body = encodeURIComponent(
        'Name: ' + fd.get('name') + '\nEmail: ' + fd.get('email') + '\n\n' + fd.get('message')
      );
      window.location.href = 'mailto:' + to + '?subject=' + subject + '&body=' + body;
    });
  }
})();
