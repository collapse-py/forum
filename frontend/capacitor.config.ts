import type { CapacitorConfig } from '@capacitor/cli';

const serverUrl = "https://forum.isme.ccwu.cc";

const config: CapacitorConfig = {
  appId: 'com.forum.app',
  appName: 'Forum',
  webDir: 'dist',
  overrideUserAgent: 'Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36',
  ...(serverUrl ? {
    server: {
      url: serverUrl,
      cleartext: serverUrl.startsWith('http://'),
    },
  } : {}),
};

export default config;
