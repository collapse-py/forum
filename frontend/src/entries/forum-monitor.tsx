/*
 * 入口：系統監控（/admin/monitor）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。與另外三個後臺入口相同：
 * AdminProvider 在殼層之外，因為 AdminShell 本身就要用 toast。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { AdminProvider } from '../admin/provider';
import { MonitorPage } from '../admin/MonitorPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin/monitor 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <MonitorPage />
    </AdminProvider>
  </I18nProvider>,
);
