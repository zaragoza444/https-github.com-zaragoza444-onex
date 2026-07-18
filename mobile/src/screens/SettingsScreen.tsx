import { useEffect, useState } from 'react';
import {
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import {
  appName,
  appVersion,
  DEFAULT_WALLET_URL,
  getWalletBaseUrl,
  normalizeWalletBaseUrl,
  setWalletBaseUrl,
} from '../config';
import { ALLTRA_PLUS_CHAIN_ID, PLUS_ENDPOINTS, POUCHPAY_PLUS_TAGLINE } from '../plusConfig';

type Props = {
  onClose: () => void;
  onSaved: () => void;
};

export function SettingsScreen({ onClose, onSaved }: Props) {
  const insets = useSafeAreaInsets();
  const [url, setUrl] = useState('');
  const [current, setCurrent] = useState('');

  useEffect(() => {
    getWalletBaseUrl().then((u) => {
      setCurrent(u);
      setUrl(u);
    });
  }, []);

  const save = async () => {
    const nextUrl = normalizeWalletBaseUrl(url);
    const defaultUrl = normalizeWalletBaseUrl(DEFAULT_WALLET_URL);
    await setWalletBaseUrl(nextUrl === defaultUrl ? '' : nextUrl);
    onSaved();
    onClose();
  };

  const reset = async () => {
    await setWalletBaseUrl('');
    setUrl(DEFAULT_WALLET_URL);
    onSaved();
    onClose();
  };

  return (
    <View style={[styles.root, { paddingTop: insets.top + 16, paddingBottom: insets.bottom + 16 }]}>
      <Text style={styles.title}>{appName()} Settings</Text>
      <Text style={styles.hint}>{POUCHPAY_PLUS_TAGLINE}</Text>
      <Text style={styles.label}>Integrated rails</Text>
      {PLUS_ENDPOINTS.map((endpoint) => (
        <Text key={endpoint.id} style={styles.meta}>{endpoint.label}: {endpoint.value}</Text>
      ))}
      <Text style={styles.meta}>Alltra Plus Chain ID: {ALLTRA_PLUS_CHAIN_ID}</Text>
      <Text style={styles.label}>Wallet URL</Text>
      <Text style={styles.hint}>Production: https://your-pouchpay-plus-domain.com/wallet/</Text>
      <TextInput
        style={styles.input}
        value={url}
        onChangeText={setUrl}
        autoCapitalize="none"
        autoCorrect={false}
        keyboardType="url"
        placeholder={DEFAULT_WALLET_URL}
        placeholderTextColor="#666"
      />
      <Text style={styles.meta}>Active: {current}</Text>
      <Text style={styles.meta}>{appName()} version {appVersion()}</Text>
      <Pressable style={styles.btn} onPress={save}>
        <Text style={styles.btnText}>Save & reload</Text>
      </Pressable>
      <Pressable style={styles.btnSecondary} onPress={reset}>
        <Text style={styles.btnTextSecondary}>Reset to default</Text>
      </Pressable>
      <Pressable style={styles.btnSecondary} onPress={onClose}>
        <Text style={styles.btnTextSecondary}>Close</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#000', paddingHorizontal: 20 },
  title: { color: '#fff', fontSize: 22, fontWeight: '600', marginBottom: 20 },
  label: { color: '#909090', fontSize: 13, marginBottom: 6 },
  hint: { color: '#5c5c5c', fontSize: 12, marginBottom: 10 },
  input: {
    backgroundColor: '#1a1a1a',
    borderWidth: 1,
    borderColor: '#2a2a2a',
    borderRadius: 12,
    color: '#fff',
    padding: 14,
    fontSize: 15,
    marginBottom: 12,
  },
  meta: { color: '#666', fontSize: 12, marginBottom: 6 },
  btn: { backgroundColor: '#fff', padding: 16, borderRadius: 12, alignItems: 'center', marginTop: 16 },
  btnSecondary: { padding: 14, alignItems: 'center', marginTop: 8 },
  btnText: { color: '#000', fontWeight: '600', fontSize: 16 },
  btnTextSecondary: { color: '#909090', fontSize: 15 },
});
