let config = null;
let provider = null;
let signer = null;

const API_KEY_STORAGE = 'bsc-launcher-api-key';

function getApiKey() {
  return localStorage.getItem(API_KEY_STORAGE) || '';
}

function setApiKey(key) {
  if (key) localStorage.setItem(API_KEY_STORAGE, key);
  else localStorage.removeItem(API_KEY_STORAGE);
}

async function api(path, opts = {}) {
  const headers = { ...(opts.headers || {}) };
  const key = getApiKey();
  if (key) headers['X-API-Key'] = key;
  const res = await fetch(path, { ...opts, headers });
  const data = await res.json().catch(() => ({}));
  if (!res.ok && !data.error) {
    data.error = res.status === 401 ? 'API key required — open Settings' : `HTTP ${res.status}`;
  }
  return data;
}

function setMsg(el, text, type = '') {
  el.textContent = text;
  el.className = 'msg' + (type ? ' ' + type : '');
}

function shortAddr(a) {
  if (!a) return '';
  return a.slice(0, 6) + '…' + a.slice(-4);
}

function fmtUsd(n) {
  if (n == null || isNaN(n)) return '$0.00';
  if (n < 0.000001) return '< $0.000001';
  return '$' + Number(n).toLocaleString(undefined, { maximumFractionDigits: 6 });
}

function fmtPct(n) {
  if (n == null || isNaN(n)) return '—';
  const sign = n >= 0 ? '+' : '';
  return sign + Number(n).toFixed(2) + '%';
}

async function loadConfig() {
  config = await api('/api/config');
  if (config.error) throw new Error(config.error);
  const backendOpt = document.querySelector('#deploy-method option[value="backend"]');
  if (backendOpt && !config.backendDeployEnabled) {
    backendOpt.disabled = true;
    backendOpt.textContent = 'Platform wallet (not configured)';
  }
  const envEl = document.getElementById('env-badge');
  if (envEl && config.env) {
    envEl.textContent = config.env;
    envEl.classList.toggle('prod', config.env === 'production');
  }
  if (config.apiKeyRequired && !getApiKey()) {
    setMsg(document.getElementById('create-msg'), 'Production: set API key in Settings before deploying.', 'error');
  }
}

function openSettings() {
  document.getElementById('settings-modal').classList.remove('hidden');
  document.getElementById('settings-api-key').value = getApiKey();
}

function closeSettings() {
  document.getElementById('settings-modal').classList.add('hidden');
}

function saveSettings() {
  setApiKey(document.getElementById('settings-api-key').value.trim());
  closeSettings();
  setMsg(document.getElementById('create-msg'), 'Settings saved.', 'ok');
}

async function connectWallet() {
  if (!window.ethereum) {
    alert('MetaMask not detected. Install MetaMask to deploy with your wallet.');
    return;
  }
  provider = new ethers.BrowserProvider(window.ethereum);
  await provider.send('eth_requestAccounts', []);
  const net = await provider.getNetwork();
  if (Number(net.chainId) !== 56) {
    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: '0x38' }],
      });
    } catch (e) {
      if (e.code === 4902) {
        await window.ethereum.request({
          method: 'wallet_addEthereumChain',
          params: [{
            chainId: '0x38',
            chainName: 'BNB Smart Chain',
            nativeCurrency: { name: 'BNB', symbol: 'BNB', decimals: 18 },
            rpcUrls: [config.rpcUrl],
            blockExplorerUrls: [config.explorer],
          }],
        });
      } else {
        throw e;
      }
    }
    provider = new ethers.BrowserProvider(window.ethereum);
  }
  signer = await provider.getSigner();
  const addr = await signer.getAddress();
  document.getElementById('wallet-addr').textContent = shortAddr(addr);
  document.getElementById('btn-connect').textContent = 'Connected';
}

async function deployMetaMask(form) {
  if (!signer) await connectWallet();
  const name = form.name.value.trim();
  const symbol = form.symbol.value.trim().toUpperCase();
  const decimals = parseInt(form.decimals.value, 10) || 18;
  const supplyHuman = form.supply.value.trim();
  const supplyRaw = ethers.parseUnits(supplyHuman, decimals);

  const factory = new ethers.ContractFactory(
    config.contractAbi,
    config.contractBytecode,
    signer
  );

  setMsg(document.getElementById('create-msg'), 'Confirm deploy transaction in MetaMask…');
  const contract = await factory.deploy(name, symbol, decimals, supplyRaw);
  const deployTx = contract.deploymentTransaction();
  const txHash = deployTx.hash;
  setMsg(document.getElementById('create-msg'), 'Waiting for BSC confirmation…');
  await deployTx.wait();
  const address = await contract.getAddress();
  const creator = await signer.getAddress();

  const reg = await api('/api/tokens/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      contractAddress: address,
      name,
      symbol,
      decimals,
      supply: supplyHuman,
      txHash,
      creator,
    }),
  });

  if (reg.error) throw new Error(reg.error);
  return reg;
}

async function deployBackend(form) {
  const body = {
    name: form.name.value.trim(),
    symbol: form.symbol.value.trim(),
    decimals: parseInt(form.decimals.value, 10) || 18,
    supply: form.supply.value.trim(),
  };
  setMsg(document.getElementById('create-msg'), 'Deploying via platform wallet…');
  const j = await api('/api/deploy', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (j.error) throw new Error(j.error);
  return j;
}

function showDeployResult(data) {
  const el = document.getElementById('deploy-result');
  const token = data.token || data;
  const tokenUrl = data.explorerTokenUrl || (config.explorer + '/token/' + token.contractAddress);
  const txUrl = data.explorerTxUrl || (config.explorer + '/tx/' + token.txHash);
  el.innerHTML = `
    <p class="msg ok">Token deployed successfully!</p>
    <p><strong>${token.symbol}</strong> — ${token.name}</p>
    <p class="token-meta">${token.contractAddress}</p>
    <p class="token-links">
      <a href="${tokenUrl}" target="_blank" rel="noopener">View on BSCScan</a>
      <a href="${txUrl}" target="_blank" rel="noopener">View transaction</a>
    </p>`;
  el.classList.remove('hidden');
}

async function handleDeploy(e) {
  e.preventDefault();
  const form = e.target;
  const msg = document.getElementById('create-msg');
  const result = document.getElementById('deploy-result');
  result.classList.add('hidden');
  const method = document.getElementById('deploy-method').value;
  const btn = document.getElementById('btn-deploy');
  btn.disabled = true;

  try {
    let data;
    if (method === 'metamask') {
      data = await deployMetaMask(form);
    } else {
      data = await deployBackend(form);
    }
    setMsg(msg, 'Deployed on BSC mainnet.', 'ok');
    showDeployResult(data);
    renderDashboard();
  } catch (err) {
    setMsg(msg, err.message || String(err), 'error');
  } finally {
    btn.disabled = false;
  }
}

async function enrichToken(token) {
  const addr = token.contractAddress;
  const [bscscan, price] = await Promise.all([
    api('/api/bscscan/' + addr).catch((e) => ({ error: e.message })),
    api('/api/price/' + addr).catch(() => ({})),
  ]);
  if (bscscan?.error && !bscscan.symbol) {
    bscscan._lookupError = bscscan.error;
  }
  return { token, bscscan, price };
}

async function renderDashboard() {
  const el = document.getElementById('token-list');
  const list = await api('/api/tokens');
  if (list.error) {
    el.innerHTML = `<p class="msg error">${list.error}</p>`;
    return;
  }
  if (!list.length) {
    el.innerHTML = '<p class="msg">No tokens yet. Create one in the Create tab.</p>';
    return;
  }

  el.innerHTML = '<p class="msg">Loading on-chain data…</p>';
  const enriched = await Promise.all(list.map(enrichToken));
  el.innerHTML = enriched.map(({ token, bscscan, price }) => {
    const tokenUrl = config.explorer + '/token/' + token.contractAddress;
    const txUrl = config.explorer + '/tx/' + token.txHash;
    const hasLiq = price && price.hasLiquidity;
    const priceNote = hasLiq ? '' : '<p class="msg">No DEX listing yet — price shows $0.00</p>';
    return `
      <div class="token-card">
        <h3>${token.symbol} <span class="badge">${token.deployMethod || 'deployed'}</span></h3>
        <p class="token-meta">${token.name} · ${token.chainId || 'bsc'} · supply ${token.supply}</p>
        <p class="token-meta">${token.contractAddress}</p>
        <div class="token-stats">
          <div class="stat"><strong>${fmtUsd(price?.priceUsd || 0)}</strong><span>Price USD</span></div>
          <div class="stat"><strong>${fmtPct(price?.priceChange24h)}</strong><span>24h change</span></div>
          <div class="stat"><strong>${bscscan?.holders || '—'}</strong><span>Holders</span></div>
          <div class="stat"><strong>${bscscan?.txCount || '—'}</strong><span>Transfers</span></div>
          <div class="stat"><strong>${price?.liquidityUsd ? fmtUsd(price.liquidityUsd) : '—'}</strong><span>Liquidity</span></div>
        </div>
        ${priceNote}
        <div class="token-links">
          <a href="${tokenUrl}" target="_blank" rel="noopener">View on BSCScan</a>
          <a href="${txUrl}" target="_blank" rel="noopener">View TX</a>
        </div>
      </div>`;
  }).join('');
}

function setTab(tab) {
  document.querySelectorAll('.tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
  document.getElementById('pane-create').classList.toggle('hidden', tab !== 'create');
  document.getElementById('pane-dashboard').classList.toggle('hidden', tab !== 'dashboard');
  if (tab === 'dashboard') renderDashboard();
}

document.querySelectorAll('.tab').forEach(btn => {
  btn.addEventListener('click', () => setTab(btn.dataset.tab));
});
document.getElementById('create-form').addEventListener('submit', handleDeploy);
document.getElementById('btn-connect').addEventListener('click', connectWallet);
document.getElementById('btn-refresh').addEventListener('click', renderDashboard);
document.getElementById('btn-lookup').addEventListener('click', lookupAddress);
document.getElementById('btn-settings').addEventListener('click', openSettings);
document.getElementById('btn-settings-save').addEventListener('click', saveSettings);
document.getElementById('btn-settings-close').addEventListener('click', closeSettings);

async function lookupAddress() {
  const input = document.getElementById('lookup-addr');
  const msg = document.getElementById('lookup-msg');
  const addr = (input.value || '').trim();
  if (!addr.startsWith('0x') || addr.length < 10) {
    setMsg(msg, 'Enter a valid BSC address (0x…)', 'error');
    return;
  }
  setMsg(msg, 'Looking up on BSC…');
  const data = await api('/api/tokens/' + encodeURIComponent(addr));
  if (data.error) {
    setMsg(msg, data.error, 'error');
    return;
  }
  const info = data.bscscan || {};
  if (info.isWallet) {
    setMsg(msg, 'This is a wallet address, not a token contract. Deploy a token from the Create tab, then use the contract address shown after deploy.', 'error');
    return;
  }
  const price = data.price || {};
  setMsg(msg, `${info.symbol || info.tokenName || 'Token'} — ${fmtUsd(price.priceUsd || 0)} · holders ${info.holders || '—'}`, 'ok');
}

loadConfig().catch(err => {
  setMsg(document.getElementById('create-msg'), err.message, 'error');
});
