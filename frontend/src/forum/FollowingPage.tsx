/*
 * 追蹤（/forum/following）
 *
 * 兩個區塊：追蹤中的人（可取消追蹤），以及這些人的貼文動態。
 *
 * 為什麼兩塊放同一頁
 * ------------------
 * 它們是同一件事的兩面。清單是「我在關注誰」，動態是「他們最近說了什麼」。
 * 拆成兩頁的話，使用者想取消追蹤得先找到清單、想看內容得先找到動態，
 * 而取消追蹤的動作通常就是「看到某篇不想再看了」的結果 —— 兩個操作應該
 * 在同一個畫面上。
 *
 * 貼文區為什麼是完整的 PostCard 而不是精簡列表
 * ------------------------------------------
 * 後端保證追蹤動態與首頁動態回同一種欄位（httpapi 的 forumPostProjection），
 * 因此可以直接餵給同一支 PostCard，留言、按讚、檢舉、追蹤鈕全部免費得到。
 * 寫一個精簡版只會少掉這些功能，並多一份要維護的渲染邏輯。
 *
 * autoLoad 為什麼可以直接給預設值（true）
 * ---------------------------------------
 * useFollow 的 autoLoad 在首頁與他人個人頁是綁登入狀態的（那一支端點掛在
 * requireLogin 後面，未登入的請求會 303 到登入頁，requestJSON 會判成
 * LoginRequiredError 而把整頁導走）。但這個頁面的路由本身就在 /forum/ 子樹上，
 * 那條路由掛了 requireLogin —— 沒登入的人根本拿不到這份 HTML，進來的一定是
 * 已登入的 session。
 *
 * 底部導覽停在「個人」：追蹤頁是從 /forum/profile 進來的子頁，而底部導覽
 * 刻意維持三格（見 shell.tsx 的 bottomTabs 說明）。
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { errorText, formatDateTime, goToLogin, LoginRequiredError, text } from '../core';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import type { ForumPost } from '../types';
import { FollowButton } from './FollowButton';
import { PostCard } from './PostCard';
import { BottomNav, ForumNav, ForumShell, InstallHint, useAuth, usePwaInstall } from './shell';
import { useComments } from './useComments';
import { useFeed, toggleLike } from './useFeed';
import { useFollow } from './useFollow';
import { useMediaTokenRelease } from './useMediaTokenRelease';
import { useReport } from './useReport';

const PAGE_SIZE = 25;
const PRELOAD_DISTANCE = '400px';

/* 函式而不是模組層常數：模組層的物件會在 import 時把當下的語言固定住，
   使用者切換語言後整個 session 都不會跟著換。 */
function authLabels(): { loggedIn: string; loggedOut: string; error: string } {
  return {
    loggedIn: t('auth.profileLoggedIn'),
    loggedOut: t('auth.profileLoggedOut'),
    error: t('auth.statusUnknown'),
  };
}

function personURL(userKey: string): string {
  return `/forum/others-profile?user=${encodeURIComponent(userKey)}`;
}

/** 一列的顯示名。沒有暱稱的人顯示「匿名使用者」，而不是後端給的空字串。 */
function personName(nickname: string | undefined): string {
  return text(nickname) || t('publicProfile.anonymous');
}

/**
 * 頭像裡的字：顯示名的第一個字元。
 *
 * 取自「實際會顯示的那個名字」而不是原始 nickname：沒有暱稱者後端回空字串，
 * 若直接對空字串取 charAt(0) 會得到空頭像，而畫面上的名字其實是「匿名使用者」。
 * 因此直接對 personName() 的結果取首字 —— 與貼文卡 PostCard 的
 * `(text(post.author) || t('post.authorAnonymous')).charAt(0)` 同一個做法。
 *
 * 沒設暱稱者會顯示「匿」的首字，與他人公開個人頁那顆 `.profile-avatar` 的
 * t('publicProfile.avatar') 一致。
 */
function personInitial(nickname: string | undefined): string {
  return personName(nickname).charAt(0);
}

export function FollowingPage() {
  usePageTitle('title.following');

  const auth = useAuth();
  const install = usePwaInstall();
  const canInteract = auth.phase === 'ready' && auth.loggedIn;

  // 存 key+參數或後端回的字串，render 時才用 tr() 解析。
  const [status, setStatus] = useState<{ message: Message | string | null; isError: boolean }>({
    message: null,
    isError: false,
  });
  const setOk = useCallback((message: Message | string) => setStatus({ message, isError: false }), []);
  const setFail = useCallback((message: Message | string) => setStatus({ message, isError: true }), []);

  const follow = useFollow();
  const { state: feed, load: loadFeed, patch: patchPost, items: postItems } = useFeed({
    pageSize: PAGE_SIZE,
    endpoint: '/api/forum/following/posts',
  });

  const findItem = useCallback((postId: number) => postItems.find((item) => item.id === postId), [postItems]);

  const comments = useComments({
    onPosted: (postId) => {
      const post = findItem(postId);
      patchPost(postId, { commentCount: (post?.commentCount || 0) + 1 });
    },
    onError: setFail,
  });

  const report = useReport({ onSent: setOk, onError: setFail });
  const { cancel: cancelReport } = report;

  /* --- 圖片 token 釋放 ---------------------------------------------------- */

  useMediaTokenRelease(postItems);

  /* --- 預載 --------------------------------------------------------------- */

  /*
   * 兩個 sentinel 各管一段：清單的尾端補「追蹤中的人」，貼文區的尾端補貼文。
   * 刻意不用同一個 observer —— 兩段的載入條件不同（清單看 useFollow 的
   * hasMore，貼文看 useFeed 的），共用一個會讓「捲到清單底部」也去觸發
   * 貼文分頁。
   */
  const peopleSentinelRef = useRef<HTMLDivElement>(null);
  const { loadMorePeople } = follow;
  useEffect(() => {
    const node = peopleSentinelRef.current;
    if (!node) return undefined;
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries[0]?.isIntersecting) return;
        loadMorePeople();
      },
      { rootMargin: PRELOAD_DISTANCE },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [loadMorePeople]);

  const postsSentinelRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const node = postsSentinelRef.current;
    if (!node) return undefined;
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries[0]?.isIntersecting) return;
        loadFeed(false);
      },
      { rootMargin: PRELOAD_DISTANCE },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [loadFeed]);

  /* --- 動作 --------------------------------------------------------------- */

  /*
   * 取消追蹤之後要重載貼文區：那一區是以追蹤清單為條件查出來的，
   * 被取消的人的貼文不該繼續留在畫面上。
   *
   * 採「重載」而不是「就地濾掉」：篩掉要處理分頁邊界（他可能在第 2 頁，
   * 濾掉之後那一頁少一筆，下一頁的 offset 就不對了），而清單通常很短，
   * 重載的成本可接受。
   */
  const handleUnfollow = useCallback(
    (userKey: string) => {
      cancelReport();
      void follow
        .toggle(userKey)
        .then((following) => {
          // 成功之後清掉錯誤行：否則先前某一次失敗的訊息會一直留著，
          // 即使後續每一次操作都成功。
          follow.clearError();
          if (following) return;
          loadFeed(true);
        })
        .catch(() => undefined);
    },
    [cancelReport, follow, loadFeed],
  );

  /*
   * 貼文卡上的追蹤鈕切換之後，同樣要在「取消」時重載；「追蹤」時則不會
   * 發生（這個頁面上的每個人都已經被追蹤了，除非使用者剛在別處追蹤了一個
   * 追蹤動態裡還沒出現的人 —— 那種情況貼文區本來就沒有他的東西可清）。
   * 兩者合併成同一個判斷，重載是 idempotent 的。
   */
  const handlePostFollowChanged = useCallback(
    (_userKey: string, following: boolean) => {
      if (!following) loadFeed(true);
    },
    [loadFeed],
  );

  const handleLike = useCallback(
    (post: ForumPost) => {
      void (async () => {
        try {
          const result = await toggleLike(post.id);
          patchPost(post.id, { liked: result.liked, likeCount: result.count });
        } catch (error) {
          /*
           * 與 FeedPage 的 handleLike 同一條路徑，包含 errorMessage 的優先序：
           * 後端給了可讀訊息（例如 429 的「請稍後」或 session 失效）就用那一句，
           * 沒給才退回本地文案。追蹤動態的按讚失敗原因與全站動態完全相同，
           * 因此沿用 feed.likeFailed 而不另立一個鍵。
           */
          if (error instanceof LoginRequiredError) {
            goToLogin();
            return;
          }
          setFail(errorText(error, msg('feed.likeFailed')));
        }
      })();
    },
    [patchPost, setFail],
  );

  const renderPost = (post: ForumPost) => (
    <PostCard
      key={post.id}
      post={post}
      canInteract={canInteract}
      comments={comments}
      report={report}
      follow={follow}
      onLike={handleLike}
      onFollowChanged={handlePostFollowChanged}
      onRequireLogin={goToLogin}
    />
  );

  const peoplePhase = follow.state.peoplePhase;
  const peopleError = follow.state.error;

  return (
    <ForumShell>
      <ForumNav auth={auth} labels={authLabels()} loginReturn="/forum/following" install={install} />

      <main className="forum-main">
        <p className={`status${status.isError ? ' error' : ''}`} role="status" aria-live="polite">
          {status.message === null ? null : tr(status.message)}
        </p>

        {/*
          追蹤的錯誤列在狀態列之下而不是取代它：兩者來源不同（前者是追蹤、
          留言與檢舉，後者是按讚），而且追蹤失敗時使用者仍然可能想繼續操作
          其他東西。兩行都是 role="status"，螢幕閱讀器會依序念出。
        */}
        {peopleError !== null ? (
          <p className="status error" role="status" aria-live="polite">
            {tr(peopleError)}
          </p>
        ) : null}

        <section aria-label={t('following.peopleLabel')}>
          <h2 className="following-section-title">{t('following.peopleLabel')}</h2>

          {peoplePhase === 'loading' ? <div className="loading">{t('following.peopleLoading')}</div> : null}
          {peoplePhase === 'failed' ? <div className="empty">{t('following.peopleFailed')}</div> : null}
          {peoplePhase === 'ready' && follow.state.items.length === 0 ? (
            <div className="empty">{t('following.emptyPeople')}</div>
          ) : null}

          <ul className="following-people">
            {follow.state.items.map((person) => (
              <li className="following-person" key={person.key}>
                {/*
                  頭像與名稱共用一個 <a>（見 forum-following.html 的 .following-person__link
                  說明）。頭像標成 aria-hidden：它只是名字的視覺重複，留著會讓讀屏
                  把連結念成「匿 匿名使用者」—— 而這個連結的可及名稱就應該只有名字。
                */}
                <a className="following-person__link" href={personURL(person.key)}>
                  <span className="avatar" aria-hidden="true">
                    {personInitial(person.nickname)}
                  </span>
                  <span className="following-person__name">{personName(person.nickname)}</span>
                </a>
                {person.followedAt ? (
                  <time className="following-person__time" dateTime={person.followedAt}>
                    {formatDateTime(person.followedAt)}
                  </time>
                ) : null}
                <FollowButton
                  userKey={person.key}
                  following
                  busy={follow.pending.has(person.key)}
                  onToggle={() => handleUnfollow(person.key)}
                />
              </li>
            ))}
          </ul>

          <div ref={peopleSentinelRef} aria-hidden="true" />
        </section>

        <section className="feed" aria-label={t('following.postsLabel')}>
          <h2 className="following-section-title">{t('following.postsLabel')}</h2>

          {feed.phase === 'loading' ? <div className="loading">{t('feed.loadingPosts')}</div> : null}
          {feed.phase === 'failed' ? <div className="empty">{t('following.postsFailed')}</div> : null}
          {feed.phase === 'ready' && postItems.length === 0 ? (
            <div className="empty">{t('following.emptyPosts')}</div>
          ) : null}

          {postItems.map(renderPost)}

          <div className="feed-status" ref={postsSentinelRef} role="status" aria-live="polite">
            {tr(feed.status)}
          </div>
        </section>
      </main>

      <BottomNav active="profile" />
      <InstallHint hint={install.hint} />
    </ForumShell>
  );
}
