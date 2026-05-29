// EIP-1193-style provider for Shiva (Ed25519). Use with Shiva Wallet extension or in-page wallet.
(function () {
  if (window.shiva) return;

  const listeners = new Map();
  let selectedAddress = null;

  function emit(event, data) {
    (listeners.get(event) || []).forEach((fn) => {
      try { fn(data); } catch (_) {}
    });
  }

  async function rpc(method, params) {
    const res = await fetch('/rpc', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ jsonrpc: '2.0', id: 1, method, params }),
    });
    const j = await res.json();
    if (j.error) throw new Error(j.error.message || 'rpc error');
    return j.result;
  }

  const provider = {
    isShiva: true,
    isMetaMask: false,
    selectedAddress: null,

    on(event, fn) {
      if (!listeners.has(event)) listeners.set(event, []);
      listeners.get(event).push(fn);
      return provider;
    },
    removeListener(event, fn) {
      const arr = listeners.get(event) || [];
      const i = arr.indexOf(fn);
      if (i >= 0) arr.splice(i, 1);
    },

    async request({ method, params = [] }) {
      switch (method) {
        case 'shiva_requestAccounts':
        case 'eth_requestAccounts': {
          const addr = window.__shivaWalletAddress;
          if (!addr) throw new Error('No Shiva wallet connected — open Wallet tab or install Shiva Wallet extension');
          selectedAddress = addr;
          provider.selectedAddress = addr;
          emit('accountsChanged', [addr]);
          return [addr];
        }
        case 'shiva_accounts':
        case 'eth_accounts':
          return selectedAddress || window.__shivaWalletAddress ? [selectedAddress || window.__shivaWalletAddress] : [];
        case 'shiva_getBalance':
        case 'eth_getBalance': {
          const addr = params[0] || selectedAddress;
          const r = await rpc('shiva_getBalance', [addr]);
          return method.startsWith('eth_') ? '0x' + (r.balance || 0).toString(16) : r;
        }
        case 'shiva_getTransactionCount':
        case 'eth_getTransactionCount': {
          const addr = params[0] || selectedAddress;
          return rpc(method, [addr]);
        }
        case 'eth_chainId':
          return rpc('eth_chainId', []);
        case 'shiva_chainId':
          return rpc('shiva_chainId', []);
        case 'shiva_sendTransaction': {
          const tx = params[0];
          if (!tx.signature && window.__shivaSignTransaction) {
            await window.__shivaSignTransaction(tx);
          }
          return rpc('shiva_sendTransaction', [tx]);
        }
        case 'wallet_addEthereumChain': {
          const chain = params[0];
          if (!window.ethereum || !window.ethereum.request) {
            throw new Error('MetaMask not detected. Shiva uses Ed25519 — use the built-in Shiva Wallet instead.');
          }
          return window.ethereum.request({ method: 'wallet_addEthereumChain', params: [chain] });
        }
        default:
          throw new Error('Unsupported method: ' + method);
      }
    },
  };

  window.shiva = provider;
})();
