/*
 * 入口：用戶管理（/admin）
 *
 * AdminProvider 在殼層之外：AdminShell 本身就要用 toast 與對話框，
 * 因此 provider 必須是整棵樹最外層。
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { AdminProvider } from '../admin/provider';
import { UsersPage } from '../admin/UsersPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <UsersPage />
    </AdminProvider>
  </I18nProvider>,
);
