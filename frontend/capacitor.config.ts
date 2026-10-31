import type { CapacitorConfig } from '@capacitor/cli';

const serverUrl = process.env.CAPACITOR_SERVER_URL;

const config: CapacitorConfig = {
  appId: 'com.forum.app',
  appName: 'Forum',
  webDir: 'dist',
  ...(serverUrl ? { server: { url: serverUrl, cleartext: true } } : {}),
};

export default config;
