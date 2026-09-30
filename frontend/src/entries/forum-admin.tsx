/*
 * 入口：論壇文章（/admin/forum）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { ForumAdminPage } from '../admin/ForumAdminPage';
import { AdminProvider } from '../admin/provider';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin/forum 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <ForumAdminPage />
    </AdminProvider>
  </I18nProvider>,
);
