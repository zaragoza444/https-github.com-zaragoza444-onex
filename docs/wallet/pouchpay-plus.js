(function () {
  const cfg = {
    appName: 'PouchPay Plus',
    tagline: 'PouchPay wallet with Alltra Plus rails',
    pouchPayUrl: 'https://pouchpay.io/',
    alltraRpcUrl: 'https://mainnet-rpc.alltra.global/',
    alltraExplorerUrl: 'https://alltra.global/',
    chainId: '651940',
    ...(window.POUCHPAY_PLUS || {}),
  };

  window.POUCHPAY_PLUS = cfg;

  function setText(selector, text) {
    const el = document.querySelector(selector);
    if (el) el.textContent = text;
  }

  function addIntegrationStatus() {
    const settings = document.getElementById('sheet-settings');
    if (!settings || document.getElementById('pouchpay-plus-integrations')) return;
    const panel = document.createElement('div');
    panel.id = 'pouchpay-plus-integrations';
    panel.className = 'settings-row pouchpay-plus-integrations';
    panel.innerHTML = `
      <span>${cfg.appName} integrations</span>
      <div class="pouchpay-plus-stack">
        <strong>${cfg.tagline}</strong>
        <small>PouchPay: ${cfg.pouchPayUrl}</small>
        <small>Alltra Plus RPC: ${cfg.alltraRpcUrl}</small>
        <small>Alltra Plus Explorer: ${cfg.alltraExplorerUrl}</small>
        <small>Alltra Plus Chain ID: ${cfg.chainId}</small>
      </div>
    `;
    const bridgeLabel = settings.querySelector('label[for="bridge-url-input"], #bridge-url-input');
    settings.insertBefore(panel, bridgeLabel ? bridgeLabel.parentNode === settings ? bridgeLabel : null : settings.children[2] || null);
  }

  function applyBranding() {
    document.title = cfg.appName;
    setText('.avatar', 'P+');
    setText('#network-name', cfg.appName);
    setText('#production-platform-banner strong', `${cfg.appName} Platform`);
    setText('#production-platform-detail', 'PouchPay + Alltra Plus connected');
    const external = document.querySelector('#external-banner p');
    if (external) external.innerHTML = '<strong>External wallet.</strong> Set your PouchPay Plus bridge URL for send, swap &amp; portfolio sync.';
    addIntegrationStatus();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', applyBranding);
  } else {
    applyBranding();
  }
})();
