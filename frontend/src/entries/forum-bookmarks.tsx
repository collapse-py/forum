/*
 * 入口：收藏（/forum/bookmarks）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { BookmarksPage } from '../forum/BookmarksPage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /forum/bookmarks 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <BookmarksPage />
  </I18nProvider>,
);
