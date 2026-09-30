/*
 * 入口：追蹤（/forum/following）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { FollowingPage } from '../forum/FollowingPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /forum/following 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <FollowingPage />
  </I18nProvider>,
);
