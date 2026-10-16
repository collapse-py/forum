/*
 * 入口：通知（/forum/notifications）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { NotificationsPage } from '../forum/NotificationsPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /forum/notifications 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <NotificationsPage />
  </I18nProvider>,
);
