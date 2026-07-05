import AsyncStorage from '@react-native-async-storage/async-storage';
import Constants from 'expo-constants';
import { POUCHPAY_PLUS_APP_NAME } from './plusConfig';
export {
  normalizeWalletBaseUrl,
  shouldOpenWalletRequestExternally,
  walletUrlWithHash,
} from './walletUrl';
import { normalizeWalletBaseUrl } from './walletUrl';

const STORAGE_KEY = 'pouchpay_plus_wallet_url_override';
const LEGACY_STORAGE_KEYS = ['pouchpay_wallet_url_override', 'onex_wallet_url_override', 'shiva_wallet_url_override'];

export const DEFAULT_WALLET_URL =
  process.env.EXPO_PUBLIC_WALLET_URL?.trim() || 'http://127.0.0.1:9338/wallet/';

export async function getWalletBaseUrl(): Promise<string> {
  let override = await AsyncStorage.getItem(STORAGE_KEY);
  for (const legacyKey of LEGACY_STORAGE_KEYS) {
    if (override) break;
    override = await AsyncStorage.getItem(legacyKey);
    if (override) await AsyncStorage.setItem(STORAGE_KEY, override);
  }
  return normalizeWalletBaseUrl(override || DEFAULT_WALLET_URL);
}

export async function setWalletBaseUrl(url: string): Promise<void> {
  const normalized = normalizeWalletBaseUrl(url);
  if (!normalized) {
    await AsyncStorage.removeItem(STORAGE_KEY);
    return;
  }
  await AsyncStorage.setItem(STORAGE_KEY, normalized);
}

export function resolveDeepLink(path: string | null): string | null {
  if (!path) return null;
  const p = path.replace(/^\/+/, '').toLowerCase();
  const map: Record<string, string> = {
    swap: 'swap',
    trade: 'swap',
    ai: 'ai',
    earn: 'earn',
    stake: 'earn',
    discover: 'discover',
    web3: 'web3',
    wallet: '',
    pouchpay: '',
    plus: '',
    home: '',
  };
  if (p in map) return map[p];
  return null;
}

export function appVersion(): string {
  return Constants.expoConfig?.version ?? '1.0.0';
}

export function appName(): string {
  return Constants.expoConfig?.name ?? POUCHPAY_PLUS_APP_NAME;
}
