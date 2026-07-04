import * as Linking from 'expo-linking';
import * as SplashScreen from 'expo-splash-screen';
import { StatusBar } from 'expo-status-bar';
import { useCallback, useEffect, useState } from 'react';
import { Modal, StyleSheet, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { resolveDeepLink } from './src/config';
import { SettingsScreen } from './src/screens/SettingsScreen';
import { WebWalletScreen } from './src/screens/WebWalletScreen';

SplashScreen.preventAutoHideAsync().catch(() => {});

export default function App() {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [deepLinkHash, setDeepLinkHash] = useState<string | null>(null);
  const [reloadKey, setReloadKey] = useState(0);

  const handleUrl = useCallback((url: string | null) => {
    if (!url) return;
    const parsed = Linking.parse(url);
    const path = parsed.path ?? parsed.hostname ?? '';
    const hash = resolveDeepLink(path);
    if (hash !== null) setDeepLinkHash(hash);
  }, []);

  useEffect(() => {
    Linking.getInitialURL().then(handleUrl);
    const sub = Linking.addEventListener('url', (e) => handleUrl(e.url));
    SplashScreen.hideAsync().catch(() => {});
    return () => sub.remove();
  }, [handleUrl]);

  return (
    <SafeAreaProvider>
      <View style={styles.root}>
        <StatusBar style="light" />
        <WebWalletScreen
          key={reloadKey}
          deepLinkHash={deepLinkHash}
          onOpenSettings={() => setSettingsOpen(true)}
        />
        <Modal visible={settingsOpen} animationType="slide" presentationStyle="pageSheet">
          <SettingsScreen
            onClose={() => setSettingsOpen(false)}
            onSaved={() => setReloadKey((k) => k + 1)}
          />
        </Modal>
      </View>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#000' },
});
