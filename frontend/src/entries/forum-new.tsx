/*
 * 入口：新增貼文頁（/forum/new）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { NewPostPage } from '../forum/NewPostPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /forum/new 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <NewPostPage />
  </I18nProvider>,
);
