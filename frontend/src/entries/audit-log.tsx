/*
 * 入口：管理員操作稽核紀錄（/admin/log）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。與另外四個後臺入口相同：
 * AdminProvider 在殼層之外，因為 AdminShell 本身就要用 toast。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { AdminLogPage } from '../admin/AdminLogPage';
import { AdminProvider } from '../admin/provider';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin/log 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <AdminLogPage />
    </AdminProvider>
  </I18nProvider>,
);
