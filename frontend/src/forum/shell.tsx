/*
 * 論壇公開頁共用的殼層（src/forum/shell.tsx）
 *
 * 頂部導覽列、底部導覽、登入狀態與 PWA 安裝流程。這些在 /forum、/forum/new、
 * /forum/profile、/forum/login、/forum/others-profile 五頁各出現一次，差別只在
 * 「目前頁」與「登入後要回哪裡」。
 *
 * 這一支刻意不引用任何後臺樣式或 admin 的 provider：那兩個檔案拖著深色 rail、
 * <dialog> 樣板與 toast 機制，公開頁為了一個登入連結下載它們並不划算。
 * 舊版 admin-core.ts 的檔頭註解記錄了同一個判斷，這裡把它變成結構上的事實：
 * 公開頁的 import 圖裡不會出現任何 admin 模組，公開頁的樣式也不會引用
 * style.css 裡任何後臺專屬的 class。
 *
 * 導覽列的站名標誌讀自 src/site.ts（後端依設定檔注入的 <meta>），因此換站名
 * 不必重建前端。
 */

import { useCallback, useEffect, useState, type ReactNode } from 'react';

import { logout, readJSONObject } from '../core';
import { LanguageSwitcher } from '../i18n/LanguageSwitcher';
import { t } from '../i18n';
import { SITE_SHORT_NAME } from '../site';

/* ==========================================================================
   登入狀態
   ========================================================================== */

export type AuthPhase = 'checking' | 'ready' | 'error';

export interface AuthState {
  phase: AuthPhase;
  loggedIn: boolean;
}

const CHECKING: AuthState = { phase: 'checking', loggedIn: false };

/**
 * 讀一次 /api/check。
 *
 * 這裡刻意不做重試或輪詢：登入狀態只會在「登入／登出／OAuth 往返」時改變，
 * 而那三種情況的結果都是整頁重新載入。因此一頁一次檢查就足夠，
 * 多輪詢只會白白產生請求。
 */
export function useAuth(): AuthState {
  const [state, setState] = useState<AuthState>(CHECKING);

  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const response = await fetch('/api/check');
        const data = await readJSONObject(response);
        if (!active) return;
        setState({ phase: 'ready', loggedIn: !!data.ok });
      } catch {
        if (active) setState({ phase: 'error', loggedIn: false });
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  return state;
}

export interface AuthLabels {
  loggedIn: string;
  loggedOut: string;
  error: string;
}

/**
 * 導覽列上那句狀態文字。
 *
 * 檢查中的文案與舊版 HTML 的初始節點一致，因此第一幀不會跳動。頁面傳進來的
 * loggedIn / loggedOut / error 三句各不相同（首頁是「可發文」、個人資料頁是
 * 「可設定個人資料」），所以是 props 而不是這裡的常數。
 */
export function authLabel(state: AuthState, labels: AuthLabels): string {
  if (state.phase === 'checking') return t('auth.checking');
  if (state.phase === 'error') return labels.error;
  return state.loggedIn ? labels.loggedIn : labels.loggedOut;
}

/* ==========================================================================
   PWA 安裝
   ========================================================================== */

const SERVICE_WORKER_URL = '/service-worker.js?v=202609190349';

let serviceWorkerRegistered = false;

/**
 * 註冊 Service Worker。
 *
 * 單一旗標讓重複呼叫（多個 hook 實例）只註冊一次；後臺頁面刻意不 import 這個
 * 模組 —— service-worker.js 的預快取清單只涵蓋 /forum/*，從後臺註冊只會
 * 讓管理頁面多一層沒有預存收益的 networkFirst。
 */
function registerServiceWorker(): void {
  if (serviceWorkerRegistered || !('serviceWorker' in navigator)) return;
  serviceWorkerRegistered = true;
  navigator.serviceWorker.register(SERVICE_WORKER_URL).catch((error: unknown) => {
    console.warn('Service worker registration failed:', error);
  });
}

/** BeforeInstallPromptEvent 尚未進入 lib.dom.d.ts，因此在這裡宣告最小介面。 */
interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
}

/**
 * 瀏覽器不提供安裝提示時的說明。
 *
 * 是一個函式而不是模組層的常數：常數會在 import 時求值，那時使用者可能還沒
 * 選定語言（或已經換過），存下來的就是某一種語言的字串。改成在按鈕被按下的
 * 當下才取，翻譯才會跟著目前的語言。
 */
function installFallback(): string {
  return t('install.hint');
}

export interface PwaInstall {
  /** 已以 standalone 模式執行（安裝後的 PWA）：這時不顯示任何安裝按鈕。 */
  standalone: boolean;
  /** 使用者按了安裝鈕、但瀏覽器當下沒有提示時要顯示的說明。null 代表沒這件事。 */
  hint: string | null;
  requestInstall: () => void;
}

export function usePwaInstall(): PwaInstall {
  const [prompt, setPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [hint, setHint] = useState<string | null>(null);
  const [standalone, setStandalone] = useState(false);

  useEffect(() => {
    registerServiceWorker();
    setStandalone(
      window.matchMedia('(display-mode: standalone)').matches ||
        (navigator as Navigator & { standalone?: boolean }).standalone === true,
    );

    const onBeforeInstallPrompt = (event: Event) => {
      // 必須 preventDefault，否則 Chrome 會自己跳出自家的安裝橫幅，
      // 這個提示就永遠不會進來（useState 也就永遠是 null）。
      event.preventDefault();
      setPrompt(event as BeforeInstallPromptEvent);
    };
    const onAppInstalled = () => {
      setPrompt(null);
    };

    window.addEventListener('beforeinstallprompt', onBeforeInstallPrompt);
    window.addEventListener('appinstalled', onAppInstalled);
    return () => {
      window.removeEventListener('beforeinstallprompt', onBeforeInstallPrompt);
      window.removeEventListener('appinstalled', onAppInstalled);
    };
  }, []);

  const requestInstall = useCallback(() => {
    if (!prompt) {
      setHint(installFallback());
      return;
    }
    setHint(null);
    void prompt.prompt().then(() => prompt.userChoice).then(() => setPrompt(null));
  }, [prompt]);

  return { standalone, hint, requestInstall };
}

/** 安裝按鈕。standalone 模式下整顆不渲染，與舊版的 button.hidden 相同。 */
export function InstallButton({ onInstall, standalone, className }: { onInstall: () => void; standalone: boolean; className?: string }) {
  if (standalone) return null;
  return (
    <button className={className ?? 'install-app-button'} type="button" onClick={onInstall}>
      {t('install.button')}
    </button>
  );
}

/** 安裝說明。放在頁尾，只有使用者按過安裝鈕才會出現。 */
export function InstallHint({ hint }: { hint: string | null }) {
  if (!hint) return null;
  return (
    <p className="install-app-hint" id="install-app-hint" role="status">
      {hint}
    </p>
  );
}

/* ==========================================================================
   導覽
   ========================================================================== */

export interface ForumNavProps {
  auth: AuthState;
  /** 登入狀態文字。showAccount=false 時不會用到。 */
  labels?: AuthLabels | undefined;
  /** OAuth 的 return 參數：登入完成後要回到哪一頁。 */
  loginReturn: string;
  install?: PwaInstall | undefined;
  /** 顯示「檢查登入狀態」與登入／登出控制（他人個人頁刻意不顯示）。 */
  showAccount?: boolean | undefined;
  /**
   * 導覽列中央的額外內容，目前只有首頁的搜尋欄。
   *
   * 刻意以「整個節點」傳入而不是把表單邏輯搬進 shell：搜尋欄的狀態（關鍵字、
   * 是否顯示取消鈕）屬於頁面，shell 只負責把它放對位置。桌面版與站名標誌
   * 同一列，寬度不足時（手機）由 CSS 換到第二列 —— 這是版面層的決定，
   * 不該在這裡寫 media query 以外的 JS。
   */
  search?: ReactNode | undefined;
}

const NO_LABELS: AuthLabels = { loggedIn: '', loggedOut: '', error: '' };

export function ForumNav({ auth, labels = NO_LABELS, loginReturn, install, showAccount = true, search }: ForumNavProps) {
  return (
    <header className="forum-nav">
      <a className="brand" href="/forum">
        {SITE_SHORT_NAME}
      </a>
      {search ? <div className="forum-nav__search">{search}</div> : null}
      {/*
        .nav-actions 現在「永遠」渲染，而不再只是 showAccount || install：
        語言切換器對每個頁面都是必要的（他人個人頁刻意不顯示登入狀態與登出鈕，
        但那一頁的使用者仍然需要能換語言）。而 .nav-actions 是 grid 的第 3 欄
        （justify-self: end），少了它這欄就不存在，導覽列的欄數會對不上。
      */}
      <div className="nav-actions">
        {showAccount ? <span className="user-state">{authLabel(auth, labels)}</span> : null}
        {install ? <InstallButton onInstall={install.requestInstall} standalone={install.standalone} /> : null}
        {showAccount && !auth.loggedIn ? (
          <a className="login-link" href={`/auth/google?return=${encodeURIComponent(loginReturn)}`}>
            {t('auth.googleLogin')}
          </a>
        ) : null}
        {showAccount && auth.loggedIn ? (
          // .logout-button 在 style.css 裡預設 display:none，顯示與否由
          // .logout-button--visible 決定 —— 不用 inline style，維持 CSP 的
          // style-src 不含 'unsafe-inline'。
          <button className="logout-button logout-button--visible" type="button" onClick={() => void logout()}>
            {t('auth.logout')}
          </button>
        ) : null}
        <LanguageSwitcher variant="forum" />
      </div>
    </header>
  );
}

export type ForumTab = 'home' | 'new' | 'profile';

/**
 * 底部導覽的三個項目。
 *
 * 標籤是函式裡的 t() 而非模組層的常數陣列，原因同 installFallback：常數會在
 * import 時把當下的語言固定住，切換語言後底部導覽就永遠停在第一種語言。
 */
function bottomTabs(): { key: ForumTab; label: string; icon: string; href: string }[] {
  return [
    { key: 'home', label: t('bottomNav.home'), icon: 'home', href: '/forum' },
    { key: 'new', label: t('bottomNav.new'), icon: 'add', href: '/forum/new' },
    { key: 'profile', label: t('bottomNav.profile'), icon: 'person', href: '/forum/profile' },
  ];
}

export interface BottomNavProps {
  active: ForumTab;
  /** 首頁在舊版是一顆按鈕（捲回頂並重載），這裡保留同樣的行為。 */
  onHome?: (() => void) | undefined;
}

export function BottomNav({ active, onHome }: BottomNavProps) {
  return (
    <nav className="bottom-nav" aria-label={t('bottomNav.label')}>
      {bottomTabs().map((tab) =>
        tab.key === 'home' && onHome ? (
          <button
            key={tab.key}
            className={`bottom-nav-item${tab.key === active ? ' active' : ''}`}
            type="button"
            onClick={onHome}
          >
            <span className="material-symbols-outlined" aria-hidden="true">
              {tab.icon}
            </span>
            <span>{tab.label}</span>
          </button>
        ) : (
          <a
            key={tab.key}
            className={`bottom-nav-item${tab.key === active ? ' active' : ''}${tab.key === 'new' ? ' nav-add' : ''}`}
            href={tab.href}
          >
            <span className="material-symbols-outlined" aria-hidden="true">
              {tab.icon}
            </span>
            <span>{tab.label}</span>
          </a>
        ),
      )}
    </nav>
  );
}

/* ==========================================================================
   版面
   ========================================================================== */

export interface ForumShellProps {
  className?: string | undefined;
  children: ReactNode;
}

/** 公開頁的外框。CSS 依賴 .forum-shell 撐出底部導覽的留白。 */
export function ForumShell({ className, children }: ForumShellProps) {
  return <div className={className ? `forum-shell ${className}` : 'forum-shell'}>{children}</div>;
}
