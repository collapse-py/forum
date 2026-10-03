/*
 * 入口：單篇貼文頁（/forum/post/{id}）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { PostPage } from '../forum/PostPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /forum/post/{id} 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <PostPage />
  </I18nProvider>,
);