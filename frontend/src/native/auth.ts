import { App } from '@capacitor/app';
import { Browser } from '@capacitor/browser';
import { Capacitor } from '@capacitor/core';

const RETURN_KEY = 'forum.nativeLoginReturn';

export function isNativePlatform(): boolean {
  return Capacitor.isNativePlatform();
}

export async function startNativeGoogleLogin(returnPath: string): Promise<void> {
  if (!Capacitor.isNativePlatform()) return;
  localStorage.setItem(RETURN_KEY, returnPath || '/forum');
  const url = `${window.location.origin}/auth/google?return=${encodeURIComponent(returnPath || '/forum')}&native=1`;
  await Browser.open({ url });
}

export function initNativeAuth(): void {
  if (!Capacitor.isNativePlatform()) return;
  App.addListener('appUrlOpen', async (event) => {
    const url = event.url || '';
    if (!url.startsWith('com.forum.app://auth/callback')) return;
    let code = '';
    try {
      const parsed = new URL(url.replace('com.forum.app://', 'http://placeholder/'));
      code = parsed.searchParams.get('code') || '';
    } catch {
      return;
    }
    try {
      await Browser.close();
    } catch {
      // ignore
    }
    if (!code) return;
    try {
      const res = await fetch('/auth/handoff', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code }),
        credentials: 'include',
      });
      if (!res.ok) return;
    } catch {
      return;
    }
    const ret = localStorage.getItem(RETURN_KEY) || '/forum';
    localStorage.removeItem(RETURN_KEY);
    window.location.assign(ret);
  });
}