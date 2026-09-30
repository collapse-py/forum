/*
 * 入口：個人資料頁（/forum/profile）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { ProfilePage } from '../forum/ProfilePage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /forum/profile 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <ProfilePage />
  </I18nProvider>,
);
