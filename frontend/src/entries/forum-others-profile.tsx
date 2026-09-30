/*
 * 入口：他人公開個人頁（/forum/others-profile）
 *
 * 見 src/entries/forum.tsx 的 StrictMode 說明。
 */

import { createRoot } from 'react-dom/client';

import { I18nProvider } from '../i18n';

import { OthersProfilePage } from '../forum/OthersProfilePage';

const root = document.getElementById('root');
if (!root) throw new Error('[FORUM] /forum/others-profile 缺少 #root 掛載點。');

createRoot(root).render(
  <I18nProvider>
    <OthersProfilePage />
  </I18nProvider>,
);
