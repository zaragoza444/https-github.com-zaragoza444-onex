type PlusEndpoint = {
  id: 'pouchpay' | 'alltra-plus-rpc' | 'alltra-plus-explorer';
  label: string;
  value: string;
};

function envOrDefault(key: string, fallback: string): string {
  const raw = process.env[key]?.trim();
  return raw || fallback;
}

export const POUCHPAY_PLUS_APP_NAME = envOrDefault('EXPO_PUBLIC_APP_NAME', 'PouchPay Plus');
export const POUCHPAY_PLUS_TAGLINE = 'PouchPay wallet with Alltra Plus rails';
export const POUCHPAY_BASE_URL = envOrDefault('EXPO_PUBLIC_POUCHPAY_URL', 'https://pouchpay.io/');
export const ALLTRA_PLUS_RPC_URL = envOrDefault(
  'EXPO_PUBLIC_ALLTRA_PLUS_RPC_URL',
  'https://mainnet-rpc.alltra.global/',
);
export const ALLTRA_PLUS_EXPLORER_URL = envOrDefault(
  'EXPO_PUBLIC_ALLTRA_PLUS_EXPLORER_URL',
  'https://alltra.global/',
);
export const ALLTRA_PLUS_CHAIN_ID = 651940;

export const PLUS_ENDPOINTS: PlusEndpoint[] = [
  { id: 'pouchpay', label: 'PouchPay', value: POUCHPAY_BASE_URL },
  { id: 'alltra-plus-rpc', label: 'Alltra Plus RPC', value: ALLTRA_PLUS_RPC_URL },
  { id: 'alltra-plus-explorer', label: 'Alltra Plus Explorer', value: ALLTRA_PLUS_EXPLORER_URL },
];
