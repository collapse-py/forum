/*
 * 他人公開個人頁（/forum/others-profile?user=）
 *
 * 職責：以加密金鑰讀取公開的暱稱與自我介紹、追蹤這位使用者，以及顯示這位使用者的
 * 貼文。
 *
 * 這頁刻意不顯示登入／登出控制，也不掛 Service Worker 安裝按鈕
 * （舊版的 forum-others-profile.html 的 nav-actions 是空的）：公開頁的對象
 * 可能是任何人，讓訪客看到「登出」是誤導；這頁通常從貼文作者名稱點進來，
 * 使用者已經在 PWA 裡，不需要再給一次安裝入口。
 *
 * 「不顯示登入控制」不等於「不知道有沒有登入」：追蹤鈕需要後者（未登入按下去
 * 導去登入頁），因此這頁掛 useAuth()，只是不把結果餵給 ForumNav 的 showAccount。
 *
 * 追蹤狀態取自 /api/forum/public-profile 回應的 following，而不是這一頁自己
 * 另外查一次追蹤清單：這個頁面只有一個對象，為了一顆按鈕多打一次請求不划算。
 * 後端刻意在「這個金鑰沒有對應的公開個人資料」時不放 following 欄位，因此
 * key 為 undefined 與 false 是兩件事 —— 前者代表沒有可追蹤的對象，按鈕不渲染。
 *
 * 貼文區直接餵 PostCard（不是自己寫一個精簡版）：後端保證
 * /api/forum/public-posts 與首頁動態回同一種欄位（httpapi 的
 * forumPostProjection），因此按讚、留言、檢舉、追蹤鈕全部免費得到。寫精簡版
 * 只會少掉這些功能，並多一份要維護的渲染邏輯。
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { errorMessage, errorText, goToLogin, requestJSON } from '../core';
import type { ForumPost, ForumProfile } from '../types';
import { msg, t, tr, usePageTitle, type Message } from '../i18n';
import { FollowButton } from './FollowButton';
import { PostCard } from './PostCard';
import { BottomNav, ForumNav, ForumShell, useAuth } from './shell';
import { useComments } from './useComments';
import { updateForumPost, useFeed, toggleLike } from './useFeed';
import { useFollow } from './useFollow';
import { useMediaTokenRelease } from './useMediaTokenRelease';
import { useReport } from './useReport';

interface PublicProfile {
  name: string;
  bio: string;
  /** 後端有沒有回 following：undefined 代表「沒有可追蹤的對象」。 */
  following: boolean | undefined;
  /** 網址上的金鑰。空字串代表連結不合法。 */
  selfKey: string;
}

const PAGE_SIZE = 25;
const PRELOAD_DISTANCE = '400px';

/* 函式而非常數：name 是要顯示的字串，常數會把 import 時的語言固定住。 */
function loading(): PublicProfile {
  return { name: t('publicProfile.loading'), bio: '', following: undefined, selfKey: '' };
}

function invalidLink(): PublicProfile {
  return {
    name: t('publicProfile.invalidLinkName'),
    bio: t('publicProfile.invalidLinkBio'),
    following: undefined,
    selfKey: '',
  };
}

export function OthersProfilePage() {
  usePageTitle('title.publicProfile');

  // 只用來判斷「能不能追蹤」；不餵給 ForumNav（見檔頭說明）。
  const auth = useAuth();
  const loggedIn = auth.phase === 'ready' && auth.loggedIn;

  /*
   * 金鑰在第一次 render 就從網址讀出來（lazy initial state），而不是放進
   * useEffect：貼文列表的端點需要它，放進 effect 就得再存一份 state 並處理
   * 「effect 還沒跑、端點已經組出來」的那一瞬間。網址在這一頁的生命週期內不會
   * 變（換 key 一定是新的頁面載入），因此讀一次就夠。
   */
  const [userKey] = useState(() => new URLSearchParams(window.location.search).get('user') ?? '');

  const [profile, setProfile] = useState<PublicProfile>(userKey ? loading() : invalidLink());

  /*
   * 追蹤清單在這一頁只需要 selfKey（用來判斷「這是不是我自己」）—— 按鈕的
   * 狀態已經由 public-profile 的 following 給了。仍然讀一次是因為 useFollow
   * 是這顆按鈕唯一能拿到 selfKey 的地方，而這一支請求對已登入者很便宜。
   *
   * 未登入時 autoLoad=false：那一支端點掛 requireLogin，未登入請求會 303
   * 到登入頁並把整頁導走（詳見 useFollow 的檔頭說明）。
   */
  const follow = useFollow({ autoLoad: loggedIn });

  // 存 key+參數或後端回的字串，render 時才用 tr() 解析。
  const [status, setStatus] = useState<{ message: Message | string | null; isError: boolean }>({
    message: null,
    isError: false,
  });
  const setOk = useCallback((message: Message | string) => setStatus({ message, isError: false }), []);
  const setFail = useCallback((message: Message | string) => setStatus({ message, isError: true }), []);

  const { state: feed, load: loadFeed, patch: patchPost, items: postItems } = useFeed({
    pageSize: PAGE_SIZE,
    // 端點自帶查詢字串是刻意的：user 是一個「這頁唯一的輸入」，把它放進 URL
    // 而不是路徑，才會讓這支請求在 devtools 的網域分組裡與 public-profile
    // 歸在一起。useFeed 會依端點是否已有 `?` 選擇續接符號。
    endpoint: `/api/forum/public-posts?user=${encodeURIComponent(userKey)}`,
    // 沒有金鑰時不發請求：那一頁顯示的是「連結不合法」，不是「這個人沒貼文」。
    enabled: userKey !== '',
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

  useMediaTokenRelease(postItems);

  /* --- 公開資料 ----------------------------------------------------------- */

  useEffect(() => {
    if (!userKey) return undefined;
    let active = true;
    void (async () => {
      try {
        const data = await requestJSON<ForumProfile>(`/api/forum/public-profile?key=${encodeURIComponent(userKey)}`, {
          fallback: t('error.fallbackNotFound'),
        });
        if (!active) return;
        setProfile({
          name: data.nickname || t('publicProfile.anonymous'),
          bio: data.bio || t('publicProfile.noBio'),
          following: data.following,
          selfKey: userKey,
        });
      } catch (error) {
        if (!active) return;
        setProfile({
          name: t('publicProfile.notFound'),
          // 這裡用 errorMessage 而非 errorText：bio 欄位是純字串（要直接顯示，
          // 不是延後翻譯的訊息），而 errorMessage 會優先取後端那句可讀訊息。
          bio: errorMessage(error, t('publicProfile.loadFailed')),
          following: undefined,
          selfKey: userKey,
        });
      }
    })();
    return () => {
      active = false;
    };
  }, [userKey]);

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

  // 自己的頁面（自己點自己的名字進來）不顯示追蹤鈕：後端會以 400 拒絕。
  const isSelf = follow.isSelf(profile.selfKey);
  const canFollow = loggedIn && profile.following !== undefined && !isSelf;

  /*
   * 這裡是個人頁的追蹤鈕唯一需要自己更新狀態的地方：按鈕的樣態來自
   * profile.following（useFollow 不知道它），而 toggle 成功後要讓它翻轉，
   * 否則畫面會停在舊樣態。toggle 回傳的是伺服器確認後的狀態 —— 不是本地
   * 反轉，因此連點被合併時結果仍然正確。
   */
  const handleToggle = useCallback(() => {
    if (!loggedIn) {
      goToLogin();
      return;
    }
    void follow
      .toggle(profile.selfKey)
      .then((following) => {
        setProfile((current) => ({ ...current, following }));
        follow.clearError();
      })
      .catch(() => undefined);
  }, [follow, loggedIn, profile.selfKey]);

  const handleLike = useCallback(
    (post: ForumPost) => {
      void (async () => {
        try {
          const result = await toggleLike(post.id);
          patchPost(post.id, { liked: result.liked, likeCount: result.count });
        } catch (error) {
          setFail(errorText(error, msg('feed.likeFailed')));
        }
      })();
    },
    [patchPost, setFail],
  );

  /*
   * 在這張卡上追蹤／取消追蹤之後，貼文區不需要重載 —— 這不是追蹤動態，
   * 貼文的篩選條件是「作者是這個人」，與我有沒有追蹤他無關。取消追蹤只影響
   * 頂部那顆面板按鈕與每張卡上的小鈕，而兩者的狀態都由同一個 follow
   * controller 與 profile.following 供應。
   */
  /*
   * 他人個人頁上理論上看不到自己的貼文（自己的在 /forum/profile），但這一頁
   * 沒有「不顯示編輯鈕」的分支：isOwnPost 由 authorKey 與自己的金鑰比對決定，
   * 而那一頁拿得到 selfKey（見上方 useFollow 的說明）。多加一個「這不是我的
   * 頁面所以不給編輯」的條件只會是一個永遠為真的分支 —— 若真的出現自己的
   * 貼文，讓它可編輯才是正確的行為。
   */
  const handleUpdate = useCallback(
    async (post: ForumPost, content: string) => {
      try {
        const next = await updateForumPost(post.id, content);
        patchPost(post.id, { content: next, edited: true });
        setOk(msg('posts.edited'));
      } catch (error) {
        setFail(errorText(error, msg('posts.editFailed')));
        // 見 FeedPage 的 handleUpdate：丟回去讓 PostCard 保留輸入框。
        throw error;
      }
    },
    [patchPost, setFail, setOk],
  );

  const renderPost = (post: ForumPost) => (
    <PostCard
      key={post.id}
      post={post}
      canInteract={loggedIn}
      comments={comments}
      report={report}
      follow={follow}
      onLike={handleLike}
      onUpdate={handleUpdate}
      onRequireLogin={goToLogin}
    />
  );

  return (
    <ForumShell className="forum-profile-shell">
      {/* showAccount=false：公開頁不顯示登入狀態與登出鈕。 */}
      <ForumNav auth={{ phase: 'ready', loggedIn: false }} loginReturn="/forum" showAccount={false} />

      <main className="forum-main profile-main">
        <a className="back-link" href="/forum">
          <span className="material-symbols-outlined" aria-hidden="true">
            arrow_back
          </span>
          {t('common.backToHome')}
        </a>

        <section className="profile-panel">
          <div className="profile-heading">
            <div className="profile-avatar">{t('publicProfile.avatar')}</div>
            <div>
              <div className="eyebrow">{t('publicProfile.eyebrow')}</div>
              <h1>{t('publicProfile.title')}</h1>
            </div>
            {canFollow ? (
              <FollowButton
                userKey={profile.selfKey}
                variant="panel"
                following={!!profile.following}
                busy={follow.pending.has(profile.selfKey)}
                onToggle={handleToggle}
              />
            ) : null}
          </div>
          <div className="public-profile" id="public-profile">
            <div className="public-profile-name" id="public-profile-name">
              {profile.name}
            </div>
            <p id="public-profile-bio">{profile.bio}</p>
            {/* 追蹤失敗的訊息放在內容下方而不是導覽列：這一頁沒有其他狀態列。 */}
            {follow.state.error !== null ? (
              <p className="status error" role="status" aria-live="polite">
                {tr(follow.state.error)}
              </p>
            ) : null}
          </div>
        </section>

        {/*
          貼文區獨立成一個 section 而非塞進上面的 .profile-panel：那一段的
          定位是「一張個人資料卡」，裡面每一格都是這個人的欄位；貼文是另一種
          內容，混進去會讓「哪些是欄位、哪些是列表」在畫面上分不出來。
        */}
        {userKey ? (
          <section className="feed public-profile-posts" aria-label={t('publicProfile.postsLabel')}>
            <h2 className="eyebrow">{t('publicProfile.postsLabel')}</h2>

            {feed.phase === 'loading' ? <div className="loading">{t('feed.loadingPosts')}</div> : null}
            {feed.phase === 'failed' ? <div className="empty">{t('feed.postsFailed')}</div> : null}
            {feed.phase === 'ready' && postItems.length === 0 ? (
              <div className="empty">{t('publicProfile.emptyPosts')}</div>
            ) : null}

            {postItems.map(renderPost)}

            <div className="feed-status" ref={sentinelRef} role="status" aria-live="polite">
              {tr(feed.status)}
            </div>
          </section>
        ) : null}

        {/* 追蹤／留言／檢舉的失敗訊息放在貼文區之後：它是全域的操作結果，
            與上面「這個人的資料」是兩件事。 */}
        <p className={`status${status.isError ? ' error' : ''}`} role="status" aria-live="polite">
          {status.message === null ? null : tr(status.message)}
        </p>
      </main>

      <BottomNav active="home" />
    </ForumShell>
  );
}
