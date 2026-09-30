/*
 * 入口：檢舉管理（/admin/forum-report）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { AdminProvider } from '../admin/provider';
import { ReportAdminPage } from '../admin/ReportAdminPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin/forum-report 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <ReportAdminPage />
    </AdminProvider>
  </I18nProvider>,
);
