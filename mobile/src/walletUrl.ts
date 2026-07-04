export function normalizeWalletBaseUrl(base: string): string {
  const trimmed = base.trim();
  if (!trimmed) return '';
  try {
    const url = new URL(trimmed);
    url.hash = '';
    if (!url.pathname.endsWith('/')) url.pathname += '/';
    return url.toString();
  } catch {
    return trimmed.replace(/#.*$/, '').replace(/\/?$/, '/');
  }
}

export function walletUrlWithHash(base: string, hash: string): string {
  const h = hash.replace(/^#/, '');
  try {
    const url = new URL(normalizeWalletBaseUrl(base));
    url.hash = h;
    return url.toString();
  } catch {
    const clean = normalizeWalletBaseUrl(base);
    return h ? `${clean}#${h}` : clean;
  }
}

function isSamePathOrChild(pathname: string, basePathname: string): boolean {
  const base = basePathname.endsWith('/') ? basePathname : `${basePathname}/`;
  const baseWithoutSlash = base.replace(/\/$/, '') || '/';
  return pathname === baseWithoutSlash || pathname.startsWith(base);
}

export function shouldOpenWalletRequestExternally(requestUrl: string, walletBaseUrl: string): boolean {
  try {
    const request = new URL(requestUrl);
    const walletBase = new URL(normalizeWalletBaseUrl(walletBaseUrl));
    if (request.origin !== walletBase.origin) return true;
    if (isSamePathOrChild(request.pathname, walletBase.pathname)) return false;
    if (isSamePathOrChild(request.pathname, '/bridge/')) return false;
  } catch {
    return true;
  }
  return true;
}
