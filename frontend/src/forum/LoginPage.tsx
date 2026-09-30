/*
 * 登入閘門（/forum/login）
 *
 * 職責：說明登入方式、帶著 return 參數導向 Google OAuth，以及安裝 App。
 *
 * 舊版這個頁面的 return 參數處理是一段內聯 <script type="module">。
 * CSP 的 script-src 'self' 沒有 'unsafe-inline'，內聯模組腳本會被直接擋下 ——
 * 也就是說從 ?return=/forum/profile 進來時，兩顆登入按鈕會指回 /forum，
 * 登入完成後被丟回首頁。改成 React 後這條路徑才真的能運作。
 */

import { useEffect, useState } from 'react';

import { t, usePageTitle } from '../i18n';
import { SITE_MARK, SITE_NAME, SITE_SHORT_NAME } from '../site';
import { InstallHint, InstallButton, usePwaInstall } from './shell';

export function LoginPage() {
  usePageTitle('title.login');

  const install = usePwaInstall();
  const [loginReturn, setLoginReturn] = useState('/forum');

  useEffect(() => {
    // 參數在 effect 裡讀取而不是在模組層：入口模組求值的時間早於 URL 就緒的
    // 假設並不成立，而用 state 渲染也讓「未知參數」的預設值只有一份。
    const requested = new URLSearchParams(window.location.search).get('return');
    if (requested) setLoginReturn(requested);
  }, []);

  const oauthHref = `/auth/google?return=${encodeURIComponent(loginReturn)}`;

  return (
    <div className="forum-shell">
      <header className="forum-nav">
        <a className="brand" href="/forum">
          {SITE_SHORT_NAME}
        </a>
        <div className="nav-actions">
          <InstallButton onInstall={install.requestInstall} standalone={install.standalone} />
          <a className="login-link" href={oauthHref}>
            {t('auth.googleLogin')}
          </a>
        </div>
      </header>

      <main className="login-gate-main">
        <section className="login-gate-panel">
          <div className="login-gate-avatar" aria-hidden="true">
            {SITE_MARK}
          </div>
          <p className="eyebrow">{t('login.eyebrow')}</p>
          <h1 id="login-title">{t('login.title')}</h1>
          <p>{t('login.body', { site: SITE_NAME })}</p>
          <div className="login-gate-actions">
            <a className="google-btn login-gate-button" href={oauthHref}>
              <span className="google-btn__mark" aria-hidden="true">
                G
              </span>
              {t('auth.loginWithGoogle')}
            </a>
            <a className="login-link login-gate-button login-gate-install" href="/forum">
              {t('login.browseFirst')}
            </a>
          </div>
        </section>
      </main>

      <button
        className="back-link plain-link"
        type="button"
        onClick={() => {
          // 有上一頁才往回走；直接開這頁的訪客（history 只有這一筆）
          // 回退只會離開站台。
          if (window.history.length > 1) {
            window.history.back();
            return;
          }
          window.location.assign('/forum');
        }}
      >
        <span className="material-symbols-outlined" aria-hidden="true">
          arrow_back
        </span>
        {t('common.backOnePage')}
      </button>

      <footer>
        <a href="/forum">{t('common.backToForumHome')}</a>
      </footer>
      <InstallHint hint={install.hint} />
    </div>
  );
}
