/*
 * 入口：內容趨勢統計（/admin/stats）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。與另外五個後臺入口相同：
 * AdminProvider 在殼層之外，因為 AdminShell 本身就要用 toast。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { ContentStatsPage } from '../admin/ContentStatsPage';
import { AdminProvider } from '../admin/provider';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin/stats 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <ContentStatsPage />
    </AdminProvider>
  </I18nProvider>,
);
