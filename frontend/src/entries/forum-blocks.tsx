/*
 * 入口：IP 封鎖名單（/admin/blocks）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。與另外八個後臺入口相同：
 * AdminProvider 在殼層之外，因為 AdminShell 本身就要用 toast 與 dialog。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { BlocksPage } from '../admin/BlocksPage';
import { AdminProvider } from '../admin/provider';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /admin/blocks 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <AdminProvider>
      <BlocksPage />
    </AdminProvider>
  </I18nProvider>,
);
