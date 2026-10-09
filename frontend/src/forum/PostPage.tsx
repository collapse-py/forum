/*
 * 單篇貼文頁（/forum/post/{id}）
 *
 * 職責：以路徑上的文章編號讀取**一篇**貼文，並給它與首頁完全相同的互動
 * （按讚、留言、檢舉、追蹤、編輯、刪除）。
 *
 * 為什麼需要這一頁：貼文原本只存在於列表裡，而列表是分頁的 —— 一篇三個月前
 * 的文章沒有任何辦法被抵達、引用或分享（網址裡沒有它的身分）。連結是論壇
 * 裡最便宜的分享方式，而它需要一個穩定的、可預測的網址：/forum/post/{id}。
 *
 * 刻意不寫一份「精簡版」貼文渲染：後端保證 GET /api/forum/posts/{id} 與
 * 列表回同一種欄位（httpapi 的 forumPostProjection 加上 loadForumPosts 的
 * 共用流程），因此直接餵 PostCard 就得到與首頁一模一樣的卡片與全部互動。
 * 寫精簡版只會少掉這些功能，並多一份要維護的渲染邏輯。
 *
 * 這頁是公開的（後端 /forum/post/ 沒有掛 requireLogin，理由見 server.go）：
 * 分享連結的人不會先跟收連結的人說「請先登入」。
 */

import { useCallback, useEffect, useMemo, useState } from 'react';

import { ApiError, LoginRequiredError, errorMessage, goToLogin, requestJSON } from '../core';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import type { ForumPost, PostDetailResponse } from '../types';
import { PostCard } from './PostCard';
import { BottomNav, ForumNav, ForumShell, InstallHint, useAuth, usePwaInstall } from './shell';
import { useComments } from './useComments';
import { updateForumPost, toggleLike } from './useFeed';
import { useFollow } from './useFollow';
import { useMediaTokenRelease } from './useMediaTokenRelease';
import { useReport } from './useReport';

type Phase = 'loading' | 'ready' | 'missing' | 'failed';

/* 函式而非常數：見 FeedPage 的 authLabels 說明。 */
function authLabels(): { loggedIn: string; loggedOut: string; error: string } {
  return {
    loggedIn: t('auth.feedLoggedIn'),
    loggedOut: t('auth.feedLoggedOut'),
    error: t('auth.statusUnknown'),
  };
}

/**
 * 從網址取出文章編號。
 *
 * 刻意用 lazy initial state（第一次 render 就讀完）而不是放進 useEffect：
 * 後端已經驗過這是個正整數（handleForumPage 的 forumPostPageID），因此這裡
 * 讀不到值只可能是「有人繞過後端直接改網址」，而那一種情況顯示「連結不合法」
 * 就好，不需要等第一次 effect 跑完才決定畫什麼。
 */
function postIDFromPath(): number {
  const match = /^\/forum\/post\/(\d+)\/?$/.exec(window.location.pathname);
  const parsed = match?.[1] ? Number(match[1]) : 0;
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : 0;
}

export function PostPage() {
  usePageTitle('title.post');

  const auth = useAuth();
  const install = usePwaInstall();

  const [postID] = useState(postIDFromPath);
  const [phase, setPhase] = useState<Phase>(postID ? 'loading' : 'missing');
  const [post, setPost] = useState<ForumPost | null>(null);
  const [status, setStatus] = useState<{ message: Message | string | null; isError: boolean }>({
    message: null,
    isError: false,
  });
  const setOk = useCallback((message: Message | string) => setStatus({ message, isError: false }), []);
  const setFail = useCallback((message: Message | string) => setStatus({ message, isError: true }), []);

  const canInteract = auth.phase === 'ready' && auth.loggedIn;
  // 追蹤清單只為 selfKey（PostCard 用它判斷「這是不是我」與「這則留言是不是
  // 我的」）而讀；未登入時不讀，否則 requestJSON 會把 303 轉址判成登入需求
  // 並把整頁導走（詳見 useFollow 的檔頭）。
  const follow = useFollow({ autoLoad: canInteract });

  /* --- 讀取 --------------------------------------------------------------- */

  useEffect(() => {
    if (!postID) return undefined;
    let active = true;
    void (async () => {
      try {
        const data = await requestJSON<PostDetailResponse>(`/api/forum/posts/${postID}`, {
          fallback: t('error.fallbackLoad'),
        });
        if (!active) return;
        const item = data?.item;
        if (!item) {
          setPhase('missing');
          return;
        }
        setPost(item);
        setPhase('ready');
      } catch (error) {
        if (!active) return;
        // 404 與「網路壞掉」在這一頁的意義不同：前者是「這篇文章可能已被刪除」
        // （連結是舊的，是使用者會遇到的正常狀態），後者才是真的失敗。
        // 分流用 ApiError.status 而不是比對訊息字串 —— 後者會在後端改措辭或
        // 換成英文語系時靜默失效，症狀是「已刪除的貼文顯示成網路錯誤」。
        if (error instanceof ApiError && error.status === 404) {
          setPhase('missing');
          return;
        }
        setPhase('failed');
      }
    })();
    return () => {
      active = false;
    };
  }, [postID]);

  /*
   * 貼文清單用 useMemo 讓身分穩定：useMediaTokenRelease 的 effect 依賴就是
   * 傳進去的陣列，而 `post ? [post] : []` 每次 render 都是新的參考，症狀是那支
   * hook 的 effect 每次 render 都重掛（它自己的註解裡寫了這件事）。
   */
  const postList = useMemo(() => (post ? [post] : []), [post]);

  /* --- 互動 --------------------------------------------------------------- */

  const patchPost = useCallback((update: Partial<ForumPost>) => {
    setPost((current) => (current ? { ...current, ...update } : current));
  }, []);

  const comments = useComments({
    onPosted: (id) => {
      if (id === postID) patchPost({ commentCount: (post?.commentCount || 0) + 1 });
    },
    onRemoved: (id) => {
      if (id === postID) patchPost({ commentCount: Math.max(0, (post?.commentCount || 0) - 1) });
    },
    onError: setFail,
  });

  /*
   * 圖片 token 在離開這一頁時釋放（與其他頁面同一支 hook）。
   *
   * 第二批是這一篇的留言：它們的作者頭像來自 GET /comments 自己簽發的 token，
   * 與貼文那把不是同一把（見 useMediaTokenRelease 檔頭第 6 點）。少了它，
   * 每展開一次留言就留下一把沒人刪的 key。
   */
  const commentList = useMemo(() => (post ? comments.get(postID).items : []), [post, postID, comments]);
  useMediaTokenRelease(postList, commentList);

  const report = useReport({ onSent: setOk, onError: setFail });

  const handleLike = useCallback(
    (target: ForumPost) => {
      void (async () => {
        try {
          const result = await toggleLike(target.id);
          patchPost({ liked: result.liked, likeCount: result.count });
        } catch (error) {
          if (error instanceof LoginRequiredError) {
            goToLogin();
            return;
          }
          setFail(errorMessage(error, tr(msg('feed.likeFailed'))));
        }
      })();
    },
    [patchPost, setFail],
  );

  const handleUpdate = useCallback(
    async (target: ForumPost, content: string) => {
      try {
        const next = await updateForumPost(target.id, content);
        patchPost({ content: next, edited: true });
        setOk(msg('posts.edited'));
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          throw error;
        }
        setFail(errorMessage(error, tr(msg('posts.editFailed'))));
        // 見 FeedPage 的 handleUpdate：丟回去讓 PostCard 保留輸入框。
        throw error;
      }
    },
    [patchPost, setFail, setOk],
  );

  /*
   * 從永久連結頁刪掉這一篇之後不能只是「畫面上不見了」：使用者仍然停在
   * /forum/post/{id}，重新整理會再看到一次「這篇文章已被刪除」。整頁導回
   * 首頁是唯一不會讓人卡在一個已經不存在的頁面上的做法。
   */
  const handleDelete = useCallback(
    async (target: ForumPost) => {
      try {
        await requestJSON(`/api/forum/posts/${target.id}`, {
          method: 'DELETE',
          fallback: tr(msg('posts.deleteFailed')),
        });
        window.location.href = '/forum';
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return;
        }
        setFail(errorMessage(error, tr(msg('posts.deleteFailed'))));
      }
    },
    [setFail],
  );

  return (
    <ForumShell>
      <ForumNav auth={auth} labels={authLabels()} loginReturn={window.location.pathname} install={install} />

      <main className="forum-main post-page-main">
        <a className="back-link" href="/forum">
          <span className="material-symbols-outlined" aria-hidden="true">
            arrow_back
          </span>
          {t('common.backToHome')}
        </a>

        <p className={`status${status.isError ? ' error' : ''}`} role="status" aria-live="polite">
          {status.message === null ? null : tr(status.message)}
        </p>

        {phase === 'loading' ? <div className="loading">{t('postPage.loading')}</div> : null}
        {/*
          「連結不合法」與「這篇文章已被刪除」分成兩句，而不是同一句「找不到」。
          前者是使用者自己打錯網址，後者是他收到一個過期的連結 —— 兩者的
          下一步完全不同（前者回到首頁，後者也許該回去看討論串）。
        */}
        {phase === 'missing' ? <div className="empty">{t('postPage.missing')}</div> : null}
        {phase === 'failed' ? <div className="empty">{t('postPage.failed')}</div> : null}

        {phase === 'ready' && post ? (
          <section className="feed" aria-label={t('postPage.label')}>
            <PostCard
              post={post}
              canInteract={canInteract}
              comments={comments}
              report={report}
              follow={follow}
              onLike={handleLike}
              onDelete={handleDelete}
              onUpdate={handleUpdate}
              onRequireLogin={goToLogin}
            />
          </section>
        ) : null}
      </main>

      <BottomNav active="home" />
      <InstallHint hint={install.hint} />
    </ForumShell>
  );
}