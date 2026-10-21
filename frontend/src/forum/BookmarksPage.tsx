/*
 * 收藏（/forum/bookmarks）
 *
 * 一頁，一個清單：我收藏過的貼文。
 *
 * 為什麼是完整的 PostCard
 * ----------------------
 * 後端保證收藏清單與首頁動態回同一種欄位（httpapi 的 forumPostProjection），
 * 因此可以直接餵給同一支 PostCard —— 按讚、留言、檢舉、追蹤與編輯全部免費
 * 得到。寫一個精簡版只會少掉這些功能，並多一份要維護的渲染邏輯。
 *
 * 取消收藏為什麼要把卡片從列表移除
 * ------------------------------
 * 這一頁的列表就是「我收藏了什麼」。取消收藏之後那張卡不再滿足這個條件，
 * 留在畫面上會讓使用者懷疑自己剛剛那一下沒生效 —— 那正是追蹤頁取消追蹤時
 * 重載列表的同一個判斷（見 FollowingPage 的 handleUnfollow），只是這裡的
 * 條件只在單一筆上，因此就地移除而不是重載整頁。
 *
 * 為什麼不必綁登入狀態
 * ------------------
 * 這一頁的路由在 /forum/ 子樹上（requireLogin）：沒登入的人根本拿不到這份
 * HTML，進來的一定是已登入的 session。因此 useFeed 直接用預設的 enabled=true，
 * 不必像首頁那樣等 auth 翻成 ready —— 那個等待在那裡是必要的，因為未登入的
 * 請求會被 requireLogin 轉到登入頁，而 requestJSON 會把它判成 LoginRequiredError
 * 並導走整頁（見 FeedPage 的 useFollow 說明）。
 *
 * 底部導覽停在「個人」：收藏頁與追蹤頁一樣是從 /forum/profile 進來的子頁，
 * 而底部導覽刻意維持三格（見 shell.tsx 的 bottomTabs 說明）。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { LoginRequiredError, errorMessage, errorText, goToLogin, requestJSON } from '../core';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import type { ForumPost } from '../types';
import { PostCard } from './PostCard';
import { BottomNav, ForumNav, ForumShell, InstallHint, useAuth, usePwaInstall } from './shell';
import { useComments } from './useComments';
import { updateForumPost, useFeed, toggleLike, toggleBookmark, type PostImageChange } from './useFeed';
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

export function BookmarksPage() {
  usePageTitle('title.bookmarks');

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
  const {
    state: feed,
    load: loadFeed,
    patch: patchPost,
    remove: removePost,
    items: postItems,
  } = useFeed({ pageSize: PAGE_SIZE, endpoint: '/api/forum/bookmarks' });

  const findItem = useCallback((postId: number) => postItems.find((item) => item.id === postId), [postItems]);

  const comments = useComments({
    onPosted: (postId) => {
      const post = findItem(postId);
      patchPost(postId, { commentCount: (post?.commentCount || 0) + 1 });
    },
    onError: setFail,
  });

  const report = useReport({ onSent: setOk, onError: setFail });

  /* --- 圖片 token 釋放 ---------------------------------------------------- */

  /*
   * 兩個來源：收藏清單的貼文（含每篇的作者頭像），以及這一頁展開過的留言
   * （每則留言的作者頭像來自 GET /comments 自己簽發的另一把 token，見
   * useMediaTokenRelease 檔頭第 6 點）。少了後者，每展開一次留言就留下一把
   * 沒人釋放的 key —— 而它不會出現在任何錯誤訊息裡。
   */
  const commentItems = useMemo(
    () => Object.values(comments.states).flatMap((state) => [...state.items, ...state.local]),
    [comments.states],
  );
  useMediaTokenRelease(postItems, commentItems);

  /* --- 預載 --------------------------------------------------------------- */

  const sentinelRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const node = sentinelRef.current;
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

  const handleLike = useCallback(
    (post: ForumPost) => {
      void (async () => {
        try {
          const result = await toggleLike(post.id);
          patchPost(post.id, { liked: result.liked, likeCount: result.count });
        } catch (error) {
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

  /*
   * 收藏／取消收藏。
   *
   * 這一頁的取消收藏多做一件事：把卡片從列表移除（理由見檔頭）。而
   * 「伺服器回說現在是已收藏」的分支在這裡其實只有一種情形 —— 連點兩下被
   * 後端合併成一次切換，回到 true。此時卡片還在列表上（移除是上一個分支
   * 做的），補回狀態即可。
   */
  const handleBookmark = useCallback(
    (post: ForumPost) => {
      void (async () => {
        try {
          const bookmarked = await toggleBookmark(post.id);
          if (bookmarked) {
            patchPost(post.id, { bookmarked: true });
            return;
          }
          removePost(post.id);
          setOk(msg('bookmarks.removed'));
        } catch (error) {
          if (error instanceof LoginRequiredError) {
            goToLogin();
            return;
          }
          setFail(errorText(error, msg('error.fallbackBookmark')));
        }
      })();
    },
    [patchPost, removePost, setFail, setOk],
  );

  /*
   * 編輯自己的貼文之後就地換掉列表裡的那一份（理由與 FeedPage 的 handleUpdate
   * 相同：重新載入整頁會把捲動位置打回頂端）。收藏清單裡也可能是自己的舊文 ——
   * 收藏自己的貼文是允許的，而那一張卡與個人頁上的完全相同。
   */
  const handleUpdate = useCallback(
    async (post: ForumPost, content: string, imageChange?: PostImageChange) => {
      try {
        const patch = await updateForumPost(post.id, content, imageChange);
        patchPost(post.id, patch);
        setOk(msg('posts.edited'));
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          throw error;
        }
        setFail(errorText(error, msg('posts.editFailed')));
        // 見 FeedPage 的 handleUpdate：丟回去讓 PostCard 保留輸入框。
        throw error;
      }
    },
    [patchPost, setFail, setOk],
  );

  /*
   * 刪掉自己的貼文之後就地從列表移除（useFeed.remove 會同步調整 offset，
   * 見該函式的說明）。
   */
  const handleDelete = useCallback(
    async (post: ForumPost) => {
      try {
        await requestJSON(`/api/forum/posts/${post.id}`, {
          method: 'DELETE',
          fallback: tr(msg('posts.deleteFailed')),
        });
        removePost(post.id);
        setOk(msg('posts.deleted'));
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return;
        }
        setFail(errorMessage(error, tr(msg('posts.deleteFailed'))));
      }
    },
    [removePost, setFail, setOk],
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
      onBookmark={handleBookmark}
      onDelete={handleDelete}
      onUpdate={handleUpdate}
      onRequireLogin={goToLogin}
    />
  );

  return (
    <ForumShell>
      <ForumNav auth={auth} labels={authLabels()} loginReturn="/forum/bookmarks" install={install} />

      <main className="forum-main bookmarks-main">
        <a className="back-link" href="/forum">
          <span className="material-symbols-outlined" aria-hidden="true">
            arrow_back
          </span>
          {t('common.backToHome')}
        </a>

        {/* 標題沿用個人頁那條入口連結的文案（profile.bookmarksEntry）：使用者
            是從那裡進來的，兩處字面相同才不會讓他懷疑自己到了別的頁面。 */}
        <h1 className="bookmarks-title">{t('profile.bookmarksEntry')}</h1>

        <p className={`status${status.isError ? ' error' : ''}`} role="status" aria-live="polite">
          {status.message === null ? null : tr(status.message)}
        </p>

        <section className="feed" aria-label={t('bookmarks.label')}>
          {feed.phase === 'loading' ? <div className="loading">{t('feed.loadingPosts')}</div> : null}
          {feed.phase === 'failed' ? <div className="empty">{t('bookmarks.failed')}</div> : null}
          {feed.phase === 'ready' && postItems.length === 0 ? (
            <div className="empty">{t('bookmarks.empty')}</div>
          ) : null}

          {postItems.map(renderPost)}

          <div className="feed-status" ref={sentinelRef} role="status" aria-live="polite">
            {tr(feed.status)}
          </div>
        </section>
      </main>

      <BottomNav active="profile" loggedIn={canInteract} />
      <InstallHint hint={install.hint} />
    </ForumShell>
  );
}
