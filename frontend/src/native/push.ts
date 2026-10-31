import { Capacitor } from '@capacitor/core';
import { PushNotifications } from '@capacitor/push-notifications';

export async function initNativePush(): Promise<void> {
  if (!Capacitor.isNativePlatform()) return;
  const perm = await PushNotifications.requestPermissions();
  if (perm.receive !== 'granted') return;
  await PushNotifications.register();
  PushNotifications.addListener('registration', (token) => {
    void fetch('/api/forum/push/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token.value, platform: 'android' }),
    }).catch(() => undefined);
  });
  PushNotifications.addListener('pushNotificationReceived', (notification) => {
    console.info('push received', notification);
  });
}
