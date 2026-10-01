/*
 * 入口：站內公告管理（/admin/announcements）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。與另外九個後臺入口相同：
 * AdminProvider 在殼層之外，因為 AdminShell 本身就要用 toast 與 dialog。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { AdminProvider } from '../admin/provider';
import { AnnouncementsPage } from '../admin/AnnouncementsPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin/announcements 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <AnnouncementsPage />
    </AdminProvider>
  </I18nProvider>,
);
