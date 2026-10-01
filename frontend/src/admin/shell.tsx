/*
 * 後臺殼層（src/admin/shell.tsx）
 *
 * rail / topbar 的骨架在十個後臺頁面完全一致，唯一差異是「哪一頁」與
 * 「頁面標題」。舊版靠多份 HTML 各自寫死 class="active" 與 data-page-title，
 * 再由 initShell() 依路徑事後覆寫 —— 兩處不同步時不會有人發現。現在這兩項
 * 直接是 props，沒有第二個真相來源可以漂移。
 *
 * 站名（rail 標記、rail 文字與 topbar 麵包屑）不是 props 也不是常數，而是
 * 讀自頁面殼的 <meta>（src/site.ts）；換站名只改設定檔，不必動這些後臺
 * 頁面中的任何一個。
 */

import { useEffect, useState, type ReactNode } from 'react';

import { adminApi } from '../api/admin';
import { errorMessage, formatNumber } from '../core';
import { Icon } from '../icons';
import { LanguageSwitcher } from '../i18n/LanguageSwitcher';
import { t } from '../i18n';
import { SITE_MARK, SITE_NAME, SITE_SHORT_NAME } from '../site';
import { useAdmin } from './provider';

export type AdminRoute =
  | '/admin'
  | '/admin/forum'
  | '/admin/forum-report'
  | '/admin/monitor'
  | '/admin/log'
  | '/admin/stats'
  | '/admin/export'
  | '/admin/sessions'
  | '/admin/blocks'
  | '/admin/announcements';

/**
 * rail 上的十個項目。
 *
 * 標籤是函式裡的 t() 而非模組層的常數陣列，原因同公開頁的 bottomTabs()：常數
 * 會在 import 時把當下的語言固定住，切換語言之後 rail 就會永遠停在第一種語言。
 * 路由與圖示則是純資料，與語言無關，因此留在模組層。
 *
 * 分組標籤（railGovernance）刻意不含後七項：它們是「站台狀態」（monitor）、
 * 「操作紀錄」（log）、「內容走向」（stats）、「資料進出」（export）、
 * 「登入狀態」（sessions）、「防禦措施」（blocks）與「站方發言」
 * （announcements），與治理（誰能發文、哪些內容違規）不同性質。把它們混在
 * 治理組會讓 rail 讀起來像「監控也是治理工具」，而實際上權限需求不同 ——
 * 治理需要「會不會」判斷與刪除，後七項只需要讀。
 *
 * 最後兩項刻意排在最後：它們是最少被日常使用的頁面（只有真的在處理濫用或
 * 要發布站方公告時才需要），而 rail 是有限的高度 —— 把罕用的頁面放最後，
 * 管理員在前四個之內不會需要捲動。
 */
function navItems(): {
  route: AdminRoute;
  label: string;
  icon: 'users' | 'posts' | 'flag' | 'monitor' | 'log' | 'chart' | 'export' | 'sessions' | 'shield' | 'megaphone';
}[] {
  return [
    { route: '/admin', label: t('admin.navUsers'), icon: 'users' },
    { route: '/admin/forum', label: t('admin.navPosts'), icon: 'posts' },
    { route: '/admin/forum-report', label: t('admin.navReports'), icon: 'flag' },
    { route: '/admin/monitor', label: t('admin.navMonitor'), icon: 'monitor' },
    { route: '/admin/log', label: t('admin.navLog'), icon: 'log' },
    { route: '/admin/stats', label: t('admin.navStats'), icon: 'chart' },
    { route: '/admin/export', label: t('admin.navExport'), icon: 'export' },
    { route: '/admin/sessions', label: t('admin.navSessions'), icon: 'sessions' },
    { route: '/admin/blocks', label: t('admin.navBlocks'), icon: 'shield' },
    { route: '/admin/announcements', label: t('admin.navAnnouncements'), icon: 'megaphone' },
  ];
}

export interface AdminShellProps {
  active: AdminRoute;
  pageTitle: string;
  children: ReactNode;
}

export function AdminShell({ active, pageTitle, children }: AdminShellProps) {
  const { toast, userCount } = useAdmin();
  const [railOpen, setRailOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  // body.is-rail-open 是 .rail 抽屜鎖住背景捲動的開關（style.css 的
  // 900px 斷點）。掛在 body 上而不是 rail 本身，因此要在 effect 裡同步。
  useEffect(() => {
    document.body.classList.toggle('is-rail-open', railOpen);
    return () => document.body.classList.remove('is-rail-open');
  }, [railOpen]);

  useEffect(() => {
    if (!railOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setRailOpen(false);
    };
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [railOpen]);

  const handleLogout = async () => {
    setLoggingOut(true);
    try {
      await adminApi('/api/logout', { method: 'POST' });
      // 登出後整頁重載，讓 rail 側的用戶數等狀態一併歸零。
      window.location.assign('/admin');
    } catch (error) {
      toast(errorMessage(error, t('admin.logoutFailed')), 'error');
      setLoggingOut(false);
    }
  };

  return (
    <>
      <a className="skip-link" href="#main">
        {t('admin.skipToMain')}
      </a>

      <div className="admin-app">
        <aside className={`rail${railOpen ? ' is-open' : ''}`} id="rail" aria-label={t('admin.railLabel')}>
          <a className="rail__brand" href="/forum" aria-label={t('admin.railBrandAria', { site: SITE_NAME })}>
            <span className="rail__mark" aria-hidden="true">
              {SITE_MARK}
            </span>
            <span className="rail__brand-text">
              <strong>{SITE_SHORT_NAME}</strong>
              <small>{t('admin.consoleName')}</small>
            </span>
          </a>

          <nav className="rail__nav" aria-label={t('admin.railNavLabel')}>
            <p className="rail__label">{t('admin.railGovernance')}</p>
            {navItems().map((item) => {
              const isActive = item.route === active;
              return (
                <a
                  key={item.route}
                  className={`rail__link${isActive ? ' is-active' : ''}`}
                  data-nav={item.route}
                  href={item.route}
                  aria-current={isActive ? 'page' : undefined}
                  onClick={() => setRailOpen(false)}
                >
                  <Icon name={item.icon} />
                  <span>{item.label}</span>
                  {/*
                    用戶數只有 /admin 會算，這裡一律渲染節點、由 userCount 決定
                    內容；條件式渲染會讓它在兩種狀態間移動 DOM 節點。
                  */}
                  {item.route === '/admin' ? (
                    <span className="rail__count" hidden={userCount === null}>
                      {formatNumber(userCount)}
                    </span>
                  ) : null}
                </a>
              );
            })}
          </nav>

          <div className="rail__foot">
            <p className="rail__session">
              <span className="rail__pulse" aria-hidden="true" />
              {t('admin.railMode')}
            </p>
            <a className="rail__exit" href="/forum">
              <Icon name="home" />
              {t('admin.railExit')}
            </a>
            <button className="rail__exit" type="button" disabled={loggingOut} onClick={() => void handleLogout()}>
              <Icon name="logout" />
              {t('auth.logout')}
            </button>
          </div>
        </aside>
        {/*
          scrim 在 900px 以上是 display:none（style.css），因此 closed 時
          直接用 hidden 拿掉節點；open 時加 is-open 讓它變成可點擊的遮罩。
        */}
        <div
          className={`rail__scrim${railOpen ? ' is-open' : ''}`}
          id="rail-scrim"
          hidden={!railOpen}
          onClick={() => setRailOpen(false)}
        />

        <div className="viewport">
          <header className="topbar">
            <button
              className="topbar__menu"
              type="button"
              aria-label={t('admin.topbarMenu')}
              aria-controls="rail"
              aria-expanded={railOpen}
              onClick={() => setRailOpen((open) => !open)}
            >
              <Icon name="menu" />
            </button>
            <p className="topbar__crumb">
              <span>{SITE_SHORT_NAME}</span>
              <i>/</i>
              <strong>{pageTitle}</strong>
            </p>
            <div className="topbar__actions">
              <span className="status-pill">
                <i aria-hidden="true" />
                {t('admin.statusOnline')}
              </span>
              <LanguageSwitcher variant="admin" />
              <a className="topbar__link" href="/forum">
                {t('admin.topbarForum')}
              </a>
            </div>
          </header>

          <main className="canvas" id="main">
            {children}
          </main>
        </div>
      </div>
    </>
  );
}
