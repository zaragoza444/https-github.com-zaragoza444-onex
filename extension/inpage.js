(function () {
  if (window.shiva) return;
  const listeners = new Map();
  let selectedAddress = null;

  function emit(event, data) {
    (listeners.get(event) || []).forEach((fn) => {
      try { fn(data); } catch (_) {}
    });
  }

  window.addEventListener('message', (event) => {
    if (event.source !== window || !event.data || event.data.target !== 'shiva-inpage') return;
    const { type, payload, id } = event.data;
    if (type === 'accounts') {
      selectedAddress = payload[0] || null;
      emit('accountsChanged', payload);
    }
    if (type === 'rpcResult' && window.__shivaPending && window.__shivaPending[id]) {
      const { resolve, reject } = window.__shivaPending[id];
      delete window.__shivaPending[id];
      if (payload.error) reject(new Error(payload.error));
      else resolve(payload.result);
    }
  });

  const provider = {
    isShiva: true,
    isMetaMask: false,
    request({ method, params = [] }) {
      return new Promise((resolve, reject) => {
        const id = Math.random().toString(36).slice(2);
        window.__shivaPending = window.__shivaPending || {};
        window.__shivaPending[id] = { resolve, reject };
        window.postMessage({ target: 'shiva-content', type: 'rpc', id, method, params }, '*');
      });
    },
    on(event, fn) {
      if (!listeners.has(event)) listeners.set(event, []);
      listeners.get(event).push(fn);
    },
  };
  window.shiva = provider;
})();
