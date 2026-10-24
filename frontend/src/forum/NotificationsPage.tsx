/*
 * 通知（/forum/notifications）
 *
 * 一頁一份的通知清單：每一列是一則事件（有人在你的貼文留言／回覆你的留言／
 * 追蹤你），點一列會跳到那則事件對應的位置，並把它標成已讀。
 *
 * 為什麼是獨立一頁而不是下拉面板
 * -----------------------------
 * 與追蹤頁同一個取捨：通知需要「載入更多」、時間戳與未讀狀態，塞進導覽列的
 * 下拉面板只會讓一個小元件同時負責定位、捲動與焦點管理。導覽列的鈴鐺因此只
 * 是入口（帶紅點），內容集中在這一頁。
 *
 * 未讀數的來源
 * ------------
 * 這一頁的未讀數來自 GET /api/forum/notifications 的回應（與列表一起回），
 * useNotifications 把它寫進共享快照給導覽列 —— 因此標記已讀之後鈴鐺與底部
 * 導覽的紅點會立刻消失，不必等重新載入。這一頁自己刻意不留一份：畫面上沒有
 * 任何地方需要那個數字（見下）。
 *
 * 進入頁面就自動已讀
 * ----------------
 * 清單一載入完、後端也回報還有未讀時，useNotifications 就直接把全部標成已讀
 * （見該檔檔頭），所以「進來這一頁」就等於把全部標成已讀：紅點立刻消失，
 * 使用者不必動手。
 *
 * 頁面上因此沒有「全部標為已讀」按鈕：它與自動已讀做的是同一件事，而自動已讀
 * 成功時它永遠是按不動的 —— 一個常駐卻幾乎永遠停用的按鈕。它唯一還有作用的
 * 時刻是自動已讀失敗，而那種情況的重試路徑有兩條，都不需要它：重新載入這一頁，
 * 或斷線後重新連上時的重新同步（兩者都會再送一次自動已讀）。
 *
 * 清單上保留的是「進來時哪些是新的」樣式（item.read）；重新載入之後才回到後端
 * 的權威狀態。自動已讀失敗仍然是看得見的：訊息會出現在下方的錯誤列。
 *
 * 為什麼整列是一個連結
 * ------------------
 * 一列就是一則事件，而對應的動作只有一個（去看那則留言、或去看那個人的頁面）。
 * 把頭像與文字包進同一個 <a>（而不是各自一個連結）的理由與追蹤清單相同：一個
 * 連結給一列，Tab 順序才與畫面上可點的範圍一致。
 */

import { useCallback, useEffect, useMemo, useRef, type MouseEvent } from 'react';

import { formatDateTime, text } from '../core';
import { ForumGlyph } from '../icons';
import { t, tr, usePageTitle } from '../i18n';
import type { ForumNotificationItem } from '../types';
import { ForumAvatar } from './ForumAvatar';
import { BottomNav, ForumNav, ForumShell, InstallHint, useAuth, usePwaInstall } from './shell';
import { useMediaTokenRelease } from './useMediaTokenRelease';
import { useNotifications } from './useNotifications';

const PRELOAD_DISTANCE = '400px';

/* 函式而不是模組層常數：模組層的物件會在 import 時把當下的語言固定住，
   使用者切換語言後整個 session 都不會跟著換（與 FollowingPage 相同）。 */
function authLabels(): { loggedIn: string; loggedOut: string; error: string } {
  return {
    loggedIn: t('auth.profileLoggedIn'),
    loggedOut: t('auth.profileLoggedOut'),
    error: t('auth.statusUnknown'),
  };
}

/** 這一則通知要跳到哪裡；沒有可跳的目的地時回 null（unknown kind）。 */
function notificationHref(item: ForumNotificationItem): string | null {
  if (item.kind === 'follow') {
    return item.actorKey ? `/forum/others-profile?user=${encodeURIComponent(item.actorKey)}` : null;
  }
  if (item.kind === 'comment' || item.kind === 'reply') {
    return typeof item.postId === 'number' && item.postId > 0 ? `/forum/post/${item.postId}` : null;
  }
  return null;
}

/** 這一列的句子。actor 由後端算好，空字串只可能來自壞資料，退回匿名。 */
function notificationText(item: ForumNotificationItem): string {
  const name = text(item.actor) || t('publicProfile.anonymous');
  switch (item.kind) {
    case 'comment':
      return t('notification.comment', { name });
    case 'reply':
      return t('notification.reply', { name });
    case 'follow':
      return t('notification.follow', { name });
    default:
      return t('notification.unknown');
  }
}

export function NotificationsPage() {
  usePageTitle('title.notifications');

  const auth = useAuth();
  const install = usePwaInstall();
  const { state, loadMore, markRead } = useNotifications();

  // 底部導覽的通知紅點（見 shell.tsx 的 BottomNav）。與 ForumNav 用同一個
  // 「已確定登入」判斷：checking 期間 loggedIn 恆為 false，紅點不需要在
  // 身分揭曉前搶先出現。
  const loggedIn = auth.phase === 'ready' && auth.loggedIn;

  /*
   * 頭像 token 釋放。通知的頭像來自 GET /api/forum/notifications 逐請求簽發的
   * 一把 token，不屬於任何貼文，因此用裸網址那一種來源傳入（見
   * useMediaTokenRelease 的 MediaTokenSource）。
   *
   * 用 useMemo 而不是每次 render 現組：這個陣列每次 render 都是新物件，而
   * hook 的 effect 依賴它 —— 每次都換身分會讓 effect 反覆重掛。
   */
  const avatarURLs = useMemo(() => state.items.map((item) => item.actorAvatar), [state.items]);
  useMediaTokenRelease(avatarURLs);

  /* --- 預載 --------------------------------------------------------------- */

  const sentinelRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const node = sentinelRef.current;
    if (!node) return undefined;
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries[0]?.isIntersecting) return;
        loadMore();
      },
      { rootMargin: PRELOAD_DISTANCE },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [loadMore]);

  /* --- 動作 --------------------------------------------------------------- */

  /*
   * 點一列。已讀的直接讓瀏覽器正常導覽；未讀的攔下導覽，先把「標記已讀」送出
   * 再跳頁 —— 若不等它完成，導覽會取消那個 POST，於是使用者點過的紅點下次
   * 回來還在。自動已讀之後這一列其實多半已經是已讀，這個 POST 是重複的；留著
   * 它是因為自動已讀失敗時，補上狀態的就是這一條路徑。標記本身失敗也照樣導覽
   * （使用者想看的內容比紅點重要）。
   */
  const handleOpen = useCallback(
    (item: ForumNotificationItem, href: string) =>
      (event: MouseEvent<HTMLAnchorElement>) => {
        if (item.read) return;
        event.preventDefault();
        void markRead(item.id)
          .catch(() => undefined)
          .finally(() => {
            window.location.assign(href);
          });
      },
    [markRead],
  );

  return (
    <ForumShell>
      <ForumNav auth={auth} labels={authLabels()} loginReturn="/forum/notifications" install={install} />

      <main className="forum-main notification-main">
        <a className="back-link" href="/forum">
          <ForumGlyph name="arrow_back" />
          {t('common.backToHome')}
        </a>

        <p className="eyebrow">{t('notification.eyebrow')}</p>
        <h1 className="notification-title">{t('notification.title')}</h1>

        {/* 伺服器／網路的錯誤（載入或標記）獨立成一行，理由與追蹤頁相同。 */}
        {state.error !== null ? (
          <p className="status error" role="status" aria-live="polite">
            {tr(state.error)}
          </p>
        ) : null}

        <section aria-label={t('notification.listLabel')}>
          {state.phase === 'loading' ? <div className="loading">{t('notification.loading')}</div> : null}
          {state.phase === 'failed' ? <div className="empty">{t('notification.loadFailed')}</div> : null}
          {state.phase === 'ready' && state.items.length === 0 ? (
            <div className="empty">{t('notification.empty')}</div>
          ) : null}

          <ul className="notification-list">
            {state.items.map((item) => {
              const href = notificationHref(item);
              const line = notificationText(item);
              const name = text(item.actor) || t('publicProfile.anonymous');
              const inner = (
                <>
                  <ForumAvatar className="avatar" url={item.actorAvatar} name={name} />
                  <span className="notification-item__text">{line}</span>
                </>
              );
              return (
                <li
                  className={`notification-item${item.read ? '' : ' notification-item--unread'}`}
                  key={item.id}
                >
                  {href ? (
                    <a className="notification-item__link" href={href} onClick={handleOpen(item, href)}>
                      {inner}
                    </a>
                  ) : (
                    <div className="notification-item__link">{inner}</div>
                  )}
                  {item.createdAt ? (
                    <time className="notification-item__time" dateTime={item.createdAt}>
                      {formatDateTime(item.createdAt)}
                    </time>
                  ) : null}
                </li>
              );
            })}
          </ul>

          <div className="feed-status" ref={sentinelRef} role="status" aria-live="polite">
            {state.phase === 'ready' && state.hasMore ? t('notification.loadMore') : null}
          </div>
        </section>
      </main>

      <BottomNav active="notifications" loggedIn={loggedIn} />
      <InstallHint hint={install.hint} />
    </ForumShell>
  );
}
