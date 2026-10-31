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
import { Capacitor } from '@capacitor/core';

import { startNativeGoogleLogin } from '../native/auth';

import { logout, readJSONObject } from '../core';
import { scheduleWebFonts } from '../fonts';
import { ForumGlyph, type ForumGlyphName } from '../icons';
import { LanguageSwitcher } from '../i18n/LanguageSwitcher';
import { t } from '../i18n';
import { SITE_SHORT_NAME } from '../site';
import { useUnreadNotifications } from './notificationStream';

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

const SERVICE_WORKER_URL = '/service-worker.js?v=202610100148';

let serviceWorkerRegistered = false;

/**
 * 註冊 Service Worker。
 *
 * 單一旗標讓重複呼叫（多個 hook 實例）只註冊一次；後臺頁面刻意不 import 這個
 * 模組 —— service-worker.js 的預快取清單只涵蓋 /forum/*，從後臺註冊只會
 * 讓管理頁面多一層沒有預存收益的 networkFirst。
 */
function registerServiceWorker(): void {
  if (Capacitor.isNativePlatform() || serviceWorkerRegistered || !('serviceWorker' in navigator)) return;
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
      Capacitor.isNativePlatform() ||
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
  /*
   * 未讀通知數。只在「這一頁會顯示帳號控制」且已登入時才發請求：未登入者打
   * /api/forum/notifications/unread-count 會拿到 303，而鈴鐺不值得為它把整頁
   * 導去登入（OthersProfilePage 刻意傳 showAccount=false，且它的 auth 恆為
   * 未登入，因此那裡永遠不會出現鈴鐺）。
   *
   * auth.phase 必須是 ready：checking 時 auth.loggedIn 恆為 false，直接沿用
   * 它會讓 hook 在真正的登入狀態揭曉前先跑一次「未登入」的分支，等 phase 變成
   * ready 才發請求 —— 結果一樣，但多一次沒有意義的狀態切換。
   */
  const unread = useUnreadNotifications(showAccount && auth.phase === 'ready' && auth.loggedIn);

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
        {/* 鈴鐺放在最前面：它是「有東西在等你」的提示，應該比登入狀態文字與
            動作按鈕更靠近品牌標誌一側，視覺上先被掃到。 */}
        {showAccount && auth.loggedIn ? (
          <a
            className="notification-bell"
            href="/forum/notifications"
            aria-label={
              unread > 0 ? t('notification.badgeLabel', { count: unread }) : t('notification.bellLabel')
            }
          >
            <ForumGlyph name="notifications" />
            {/* 數字本身不打給讀屏看：可及名稱已經由上面的 aria-label 說明。
                超過 99 顯示 99+，避免三位數把紅點撐成一個長方形。 */}
            {unread > 0 ? (
              <span className="notification-badge" aria-hidden="true">
                {unread > 99 ? '99+' : unread}
              </span>
            ) : null}
          </a>
        ) : null}
        {showAccount ? <span className="user-state">{authLabel(auth, labels)}</span> : null}
        {install ? <InstallButton onInstall={install.requestInstall} standalone={install.standalone} /> : null}
        {showAccount && !auth.loggedIn ? (
          <a className="login-link" href={`/auth/google?return=${encodeURIComponent(loginReturn)}`} onClick={(e) => {
          if (Capacitor.isNativePlatform()) {
            e.preventDefault();
            void startNativeGoogleLogin(loginReturn);
          }
        }}>
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

export type ForumTab = 'home' | 'new' | 'notifications' | 'profile';

/**
 * 底部導覽的四個項目。
 *
 * 標籤是函式裡的 t() 而非模組層的常數陣列，原因同 installFallback：常數會在
 * import 時把當下的語言固定住，切換語言後底部導覽就永遠停在第一種語言。
 *
 * 順序：首頁、新增、通知、個人。新增排在第二、通知插在個人之前，是為了讓
 * 既有的首頁與「新增」維持原位（既有使用者的肌肉記憶），而個人頁仍然收在
 * 最右緣（手機上最遠、最少誤觸，也符合「帳號放最後」的慣例）。
 */
function bottomTabs(): { key: ForumTab; label: string; icon: ForumGlyphName; href: string }[] {
  return [
    { key: 'home', label: t('bottomNav.home'), icon: 'home', href: '/forum' },
    { key: 'new', label: t('bottomNav.new'), icon: 'add', href: '/forum/new' },
    { key: 'notifications', label: t('bottomNav.notifications'), icon: 'notifications', href: '/forum/notifications' },
    { key: 'profile', label: t('bottomNav.profile'), icon: 'person', href: '/forum/profile' },
  ];
}

export interface BottomNavProps {
  active: ForumTab;
  /** 首頁在舊版是一顆按鈕（捲回頂並重載），這裡保留同樣的行為。 */
  onHome?: (() => void) | undefined;
  /**
   * 是否已登入。通知紅點只在登入後出現（未登入時那一格是登入頁的入口），
   * 而未讀數端點掛在 requireLogin 之後 —— 傳入而不是在 shell 內重新讀取登入
   * 狀態：每一頁都已經有一份 useAuth 的結果，再讀一次會多打一次 /api/check。
   */
  loggedIn: boolean;
}

export function BottomNav({ active, onHome, loggedIn }: BottomNavProps) {
  /*
   * 未讀紅點。未讀數與頂部鈴鐺共用同一份模組層快照（見 useNotifications），
   * 因此這一頁只會有一次 /unread-count 請求，標記已讀時兩邊也一起更新。
   *
   * 這裡只放一個點而不是數字：底部導覽的字級小，塞進「99+」會讓四個項目的
   * 寬度不一致。數字由頂部鈴鐺負責。
   */
  const unread = useUnreadNotifications(loggedIn);
  const hasUnread = unread > 0;

  /* 圖示外多一層 .bottom-nav-icon 是紅點的定位基準：點要貼在圖示的右上角，
     而不是格子（比圖示寬得多）的角落。 */
  const icon = (tab: { key: ForumTab; icon: ForumGlyphName }) => (
    <span className="bottom-nav-icon">
      {/*
        filled 由「是不是目前頁」決定：這是四個項目裡唯一的啟用提示，
        舊版靠 Material 的 font-variation-settings: "FILL" 1 達成同一件事。
      */}
      <ForumGlyph name={tab.icon} filled={tab.key === active} />
      {tab.key === 'notifications' && hasUnread ? <span className="bottom-nav-dot" aria-hidden="true" /> : null}
    </span>
  );

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
            {icon(tab)}
            <span className="bottom-nav-label">{tab.label}</span>
          </button>
        ) : (
          <a
            key={tab.key}
            className={`bottom-nav-item${tab.key === active ? ' active' : ''}${tab.key === 'new' ? ' nav-add' : ''}`}
            href={tab.href}
            /* 有未讀時把可及名稱換成「N 則未讀通知」（與頂部鈴鐺同一個說法）：
               紅點本身是裝飾，讀屏使用者只從這裡知道「有幾則」。 */
            aria-label={
              tab.key === 'notifications' && hasUnread
                ? t('notification.badgeLabel', { count: unread })
                : undefined
            }
          >
            {icon(tab)}
            <span className="bottom-nav-label">{tab.label}</span>
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

/**
 * 公開頁的外框。CSS 依賴 .forum-shell 撐出底部導覽的留白。
 *
 * 也是「首次繪製之後才載入網路字型」的觸發點（見 src/fonts.ts）：這一層是九個
 * 論壇頁唯一的共用容器，把副作用放在這裡等於九頁都會拿到非阻斷的字型載入，
 * 而不必在每一個入口檔案各寫一次。放在 effect（而非模組求值）是為了確保它排在
 * 首次算繪之後 —— 那正是把 Google Fonts 從關鍵路徑上拿掉的關鍵。
 *
 * 「跳至主要內容」的連結同樣放在這一層：它是九頁唯一都拿得到的位置，而且必須排在
 * 任何導覽連結**之前**，鍵盤使用者按 Tab 才會先看到它而不是先跳進導覽列。各頁的
 * <main> 都帶 id="main"，因此這條連結在九頁都落到對的地方。
 */
export function ForumShell({ className, children }: ForumShellProps) {
  useEffect(() => {
    scheduleWebFonts();
  }, []);
  return (
    <div className={className ? `forum-shell ${className}` : 'forum-shell'}>
      <a className="skip-link" href="#main">
        {t('common.skipToMain')}
      </a>
      {children}
    </div>
  );
}
