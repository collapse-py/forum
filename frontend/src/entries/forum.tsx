/*
 * 入口：論壇首頁（/forum）
 *
 * 每個 HTML 只放一個掛載點，這裡是它的啟動檔。刻意不放 StrictMode：
 * 本站的頁面都是 effect 驅動的資料載入，StrictMode 會在開發模式把 mount 跑兩次，
 * 等於每個頁面第一次開啟都發兩輪請求（而查詢端點是公開的、寫入端點會吃額度）。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { FeedPage } from '../forum/FeedPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /forum 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <FeedPage />
  </I18nProvider>,
);
