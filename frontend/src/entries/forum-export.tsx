/*
 * 入口：匯出與批次操作（/admin/export）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。與另外六個後臺入口相同：
 * AdminProvider 在殼層之外，因為 AdminShell 本身就要用 toast。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { ExportPage } from '../admin/ExportPage';
import { AdminProvider } from '../admin/provider';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin/export 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <ExportPage />
    </AdminProvider>
  </I18nProvider>,
);
