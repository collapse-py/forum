/*
 * 個人資料頁（/forum/profile）
 *
 * 職責：讀取自己的暱稱與自我介紹、就地編輯、存回後端，以及列出自己發過的貼文。
 *
 * 舊版有五個 hidden 屬性在互相開關（#profile-view / #profile-form /
 * #profile-edit / #profile-login-prompt / 登入提示），任何一處漏切就會出現
 * 「表單和唯讀畫面同時可見」。這裡用一個 phase 決定要渲染哪一種狀態，
 * 那些組合不可能再出現。
 *
 * 「我的貼文」為什麼是 section 而不是 profile-panel 裡的一格
 * ---------------------------------------------------------
 * .profile-panel 裡的每一格都是「這個欄位的標題與值」，塞進第三個欄位會讓
 * 「哪些是資料、哪些是列表」在畫面上分不出來。貼文是另一種內容，因此與
 * /forum/others-profile 對稱地獨立成一段（那頁的分隔理由見該檔的註解）。
 *
 * 貼文區直接餵 PostCard 而不自己寫精簡版：後端保證 /api/forum/my-posts 與首頁
 * 動態回同一種欄位（httpapi 的 forumPostProjection），因此按讚、留言、檢舉與
 * 刪除全部免費得到 —— 而「刪掉自己某篇舊文」正是這一頁存在的主要理由。
 */

import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';

import {
  LoginRequiredError,
  errorText,
  goToLogin,
  readJSONObject,
  requestJSON,
} from '../core';
import type { ForumPost, ForumProfile } from '../types';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import { PostCard } from './PostCard';
import { BottomNav, ForumNav, ForumShell, usePwaInstall, type AuthState } from './shell';
import { useComments } from './useComments';
import { useFeed, toggleLike } from './useFeed';
import { useFollow } from './useFollow';
import { useMediaTokenRelease } from './useMediaTokenRelease';
import { useReport } from './useReport';

/** 單頁筆數與另外三個貼文列表一致（後端 handleForumMyPosts 的 pageSize）。 */
const PAGE_SIZE = 25;
const PRELOAD_DISTANCE = '400px';

/* 函式而非常數：模組層的物件會把 import 時的語言固定住，切換後就不會跟著換。 */
function authLabels(): { loggedIn: string; loggedOut: string; error: string } {
  return {
    loggedIn: t('auth.profileLoggedIn'),
    loggedOut: t('auth.profileLoggedOut'),
    error: t('auth.statusUnknown'),
  };
}

type Phase = 'checking' | 'anonymous' | 'ready' | 'error';

export function ProfilePage() {
  usePageTitle('title.profile');

  const install = usePwaInstall();
  const [phase, setPhase] = useState<Phase>('checking');
  const [editing, setEditing] = useState(false);
  const [nickname, setNickname] = useState('');
  const [bio, setBio] = useState('');
  // 存 key+參數或後端回的字串，render 時才翻譯（見 runtime.ts 的「延後翻譯的訊息」）。
  const [status, setStatus] = useState<{ message: Message | string | null; isError: boolean }>({
    message: null,
    isError: false,
  });
  const [saving, setSaving] = useState(false);
  const nicknameRef = useRef<HTMLInputElement>(null);

  // 這頁的「是否登入」不是用 useAuth()：讀不到個人資料時要顯示的是
  // 「登入後即可設定個人資料」而不是「登入後即可發文」，而且 /api/check
  // 與 /api/forum/profile 是一次連續的兩步，任一步失敗都導向同一個畫面。
  const navAuth: AuthState =
    phase === 'ready'
      ? { phase: 'ready', loggedIn: true }
      : phase === 'checking'
        ? { phase: 'checking', loggedIn: false }
        : { phase: 'error', loggedIn: false };

  /*
   * phase === 'ready' 才等同「已登入」：這頁的身分判斷是上面那兩步連續請求的
   * 結果，checking 期間不能當成已登入 —— /api/forum/my-posts 掛在 requireLogin
   * 之後，未登入時會 303 到登入頁，而 requestJSON 把它判成 LoginRequiredError，
   * 於是還沒載完個人資料就先把整頁導走。
   */
  const loggedIn = phase === 'ready';

  /*
   * 追蹤清單在這一頁不是「為了追蹤」：useFollow 的 selfKey 是 PostCard 判斷
   * 「這篇是不是我自己的」唯一來源，而那一個判斷同時控制兩件事 ——
   * 自己的貼文不渲染追蹤鈕（自我追蹤被後端以 400 拒絕），以及 ⋯ 選單裡
   * 「刪除」這一項才會出現（見 PostCard 的 isOwnPost）。
   * 因此少了這一次請求，這一頁會既不能刪除自己的貼文、每一張卡又都多一顆
   * 注定失敗的追蹤鈕。autoLoad 綁在 loggedIn 上是為了不讓未登入的請求被 303
   * 導走（與 /forum/others-profile 同一組判斷）。
   */
  const follow = useFollow({ autoLoad: loggedIn });

  const { state: feed, load: loadFeed, patch: patchPost, remove: removePost, items: postItems } = useFeed({
    pageSize: PAGE_SIZE,
    // 必須顯式指定 endpoint：useFeed 的預設是全站動態 /api/forum/posts，而這一頁
    // 要的是「我自己的貼文」。後端回同一組欄位（forumPostProjection），因此
    // 分頁、預載、圖片 token 釋放與錯誤處理全部沿用同一支流程。
    endpoint: '/api/forum/my-posts',
    // 身分確認之前不發請求：端點掛在 requireLogin 之後，未登入時會 303 到登入頁
    // 而被 requestJSON 判成 LoginRequiredError（見上面 loggedIn 的說明）。
    // enabled 翻成 true 時 useFeed 會自動補上第一次載入。
    enabled: loggedIn,
  });

  const findItem = useCallback((postId: number) => postItems.find((item) => item.id === postId), [postItems]);

  const comments = useComments({
    onPosted: (postId) => {
      const post = findItem(postId);
      patchPost(postId, { commentCount: (post?.commentCount || 0) + 1 });
    },
    onError: (message) => setStatus({ message, isError: true }),
  });

  const report = useReport({
    onSent: (message) => setStatus({ message, isError: false }),
    onError: (message) => setStatus({ message, isError: true }),
  });

  // 貼文列的圖片離開頁面時要釋放 token（實作見 useMediaTokenRelease）。
  useMediaTokenRelease(postItems);

  const load = useCallback(async () => {
    try {
      const check = await readJSONObject(await fetch('/api/check'));
      if (!check.ok) {
        setPhase('anonymous');
        return;
      }
      const profile = await requestJSON<ForumProfile>('/api/forum/profile', { fallback: t('error.fallbackProfile') });
      setNickname(profile.nickname ?? '');
      setBio(profile.bio ?? '');
      setPhase('ready');
      setEditing(false);
    } catch (error) {
      if (error instanceof LoginRequiredError) {
        goToLogin();
        return;
      }
      setStatus({ message: errorText(error, msg('profile.loadFailed')), isError: true });
      setPhase('error');
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const startEditing = () => {
    setStatus({ message: '', isError: false });
    setEditing(true);
    // 等 React 提交完這次 render：此刻輸入框還不存在，直接 focus 會落在 body。
    requestAnimationFrame(() => nicknameRef.current?.focus());
  };

  const cancelEditing = () => {
    setStatus({ message: '', isError: false });
    setEditing(false);
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (saving) return;
    setSaving(true);
    setStatus({ message: t('profile.saving'), isError: false });
    try {
      const profile = await requestJSON<ForumProfile>('/api/forum/profile', {
        method: 'PUT',
        json: { nickname: nickname.trim(), bio: bio.trim() },
        fallback: t('error.fallbackProfileSave'),
      });
      setNickname(profile.nickname ?? '');
      setBio(profile.bio ?? '');
      setEditing(false);
      setStatus({ message: t('profile.updated'), isError: false });
    } catch (error) {
      if (error instanceof LoginRequiredError) {
        goToLogin();
        return;
      }
      setStatus({ message: errorText(error, msg('profile.saveFailed')), isError: true });
    } finally {
      setSaving(false);
    }
  };

  /* --- 貼文區的動作 ------------------------------------------------------- */

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
          setStatus({ message: errorText(error, msg('feed.likeFailed')), isError: true });
        }
      })();
    },
    [patchPost],
  );

  /*
   * 刪掉自己的貼文之後就地從列表移除，而不是重新載入整頁：useFeed 的 offset 會
   * 跟著減一，因此往下的分頁不會漏讀或重讀（見 useFeed.remove 的說明）。
   */
  const handleDelete = useCallback(
    async (post: ForumPost) => {
      try {
        await requestJSON(`/api/forum/posts/${post.id}`, {
          method: 'DELETE',
          fallback: tr(msg('posts.deleteFailed')),
        });
        removePost(post.id);
        setStatus({ message: msg('posts.deleted'), isError: false });
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return;
        }
        setStatus({ message: errorText(error, msg('posts.deleteFailed')), isError: true });
      }
    },
    [removePost],
  );

  /* --- 預載 --------------------------------------------------------------- */

  /*
   * 與另外三個貼文列表同一組 sentinel。rootMargin 刻意比首頁小：這一頁的貼文
   * 區在個人資料卡「下面」，使用者通常要捲過卡片才開始讀列表，提前載入的
   * 命中率因此比首頁低得多。
   *
   * 依賴必須帶 loggedIn：這個 section 是在 phase 翻成 'ready' 之後才出現的，
   * 而掛載時那一次 effect 執行時 sentinelRef.current 還是 null（拿不到節點就
   * 直接結束）。若只依賴 loadFeed（它的身分不變），observer 就永遠不會接上，
   * 症狀是「第一頁 25 則永遠載不到下一頁」—— 不會有任何錯誤訊息。
   */
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
  }, [loggedIn, loadFeed]);

  const renderPost = (post: ForumPost) => (
    <PostCard
      key={post.id}
      post={post}
      // 貼文區只在 loggedIn 時渲染，因此這裡恆為 true；仍寫成 loggedIn 而不是
      // 硬編 true —— canInteract 的意思是「這張卡的按鈕能不能用」，它不該由
      // 「這個 section 會不會出現」間接決定。
      canInteract={loggedIn}
      comments={comments}
      report={report}
      follow={follow}
      onLike={handleLike}
      onDelete={handleDelete}
      onRequireLogin={goToLogin}
    />
  );

  return (
    <ForumShell className="forum-profile-shell">
      <ForumNav auth={navAuth} labels={authLabels()} loginReturn="/forum/profile" install={install} />

      <main className="forum-main profile-main">
        <a className="back-link" href="/forum">
          <span className="material-symbols-outlined" aria-hidden="true">
            arrow_back
          </span>
          {t('common.backToHome')}
        </a>

        <section className="profile-panel">
          <div className="profile-heading">
            <div className="profile-avatar">{t('newPost.avatarYou')}</div>
            <div>
              <div className="eyebrow" id="profile-eyebrow">
                {t('profile.eyebrow')}
              </div>
              <h1 id="profile-title">{t('profile.title')}</h1>
            </div>
            {phase === 'ready' && !editing ? (
              <button className="profile-edit-button" id="profile-edit" type="button" onClick={startEditing}>
                {t('profile.edit')}
              </button>
            ) : null}
          </div>

          {phase === 'anonymous' ? (
            <div className="profile-login-prompt" id="profile-login-prompt">
              <p>{t('profile.loginPrompt')}</p>
              <a className="login-link" href="/auth/google?return=%2Fforum%2Fprofile">
                {t('auth.loginWithGoogle')}
              </a>
            </div>
          ) : null}

          {phase === 'error' ? (
            <p className={`status${status.isError ? ' error' : ''}`} role="status">
              {status.message === null ? null : tr(status.message)}
            </p>
          ) : null}

          {phase === 'ready' && !editing ? (
            <div className="profile-view" id="profile-view">
              <div className="profile-view-row">
                <span className="profile-view-label">{t('profile.nicknameLabel')}</span>
                <strong id="profile-nickname-view">{nickname.trim() || t('profile.notSet')}</strong>
              </div>
              <div className="profile-view-row">
                <span className="profile-view-label">{t('profile.bioLabel')}</span>
                <p id="profile-bio-view">{bio.trim() || t('profile.notSetBio')}</p>
              </div>
              {/*
                追蹤的入口放在個人資料頁而不是底部導覽：底部那三格是最高頻的
                三件事（看動態／發文／改資料），追蹤是次要瀏覽，而個人資料頁
                本來就是「我的帳號」容器。

                純連結，因此不需要任何資料 —— 刻意不在這裡讀追蹤清單來顯示
                人數：那一頁的分頁是推測式的（有下一頁只能說「N+」），一個
                會與實際不符的數字比沒有數字更糟。

                版面刻意不做成上面兩欄那種「標題在上、值在下」：沒有值可填的
                欄位看起來只會像填錯了。箭頭是裝飾，標成 aria-hidden，讓這個
                連結的可及名稱就是「我的追蹤」四個字。
              */}
              <a className="profile-following-link" href="/forum/following">
                <span>{t('profile.followingEntry')}</span>
                <span className="material-symbols-outlined" aria-hidden="true">
                  chevron_right
                </span>
              </a>
            </div>
          ) : null}

          {phase === 'ready' && editing ? (
            <form id="profile-form" onSubmit={(event) => void handleSubmit(event)}>
              <label htmlFor="profile-nickname">{t('profile.nicknameInput')}</label>
              <input
                id="profile-nickname"
                ref={nicknameRef}
                maxLength={30}
                placeholder={t('profile.nicknamePlaceholder')}
                required
                value={nickname}
                onChange={(event) => setNickname(event.target.value)}
              />
              <p className="profile-hint">{t('profile.nicknameHint')}</p>
              <label htmlFor="profile-bio">{t('profile.bioLabel')}</label>
              <textarea
                id="profile-bio"
                maxLength={500}
                placeholder={t('profile.bioPlaceholder')}
                value={bio}
                onChange={(event) => setBio(event.target.value)}
              />
              <p className="profile-hint">{t('profile.bioHint')}</p>
              <div className="profile-form-actions">
                <button className="submit-button" type="submit" disabled={saving}>
                  {saving ? t('profile.saving') : t('common.save')}
                </button>
                <button className="logout-button profile-cancel-button" id="profile-cancel" type="button" onClick={cancelEditing}>
                  {t('common.cancel')}
                </button>
              </div>
              <p className={`status${status.isError ? ' error' : ''}`} id="profile-status" role="status">
                {status.message === null ? null : tr(status.message)}
              </p>
            </form>
          ) : null}
        </section>

        {/* 貼文區獨立成一個 section：理由見檔頭。 */}
        {loggedIn ? (
          <section className="feed profile-posts" aria-label={t('profile.postsLabel')}>
            <h2 className="eyebrow">{t('profile.postsLabel')}</h2>

            {feed.phase === 'loading' ? <div className="loading">{t('feed.loadingPosts')}</div> : null}
            {feed.phase === 'failed' ? <div className="empty">{t('feed.postsFailed')}</div> : null}
            {feed.phase === 'ready' && postItems.length === 0 ? (
              <div className="empty">{t('profile.emptyPosts')}</div>
            ) : null}

            {postItems.map(renderPost)}

            <div className="feed-status" ref={sentinelRef} role="status" aria-live="polite">
              {tr(feed.status)}
            </div>
          </section>
        ) : null}

        {/*
          貼文區的操作結果（按讚／刪除／檢舉／留言失敗）放在卡片之後，且只在
          唯讀狀態出現：編輯中時同一則訊息由表單內的 #profile-status 顯示
          （它在欄位旁邊才有意義）。刻意不放進 .profile-panel —— 那一段的定位
          是「一張個人資料卡」，而這是全域的操作結果，與「這個人的資料」無關。
        */}
        {loggedIn && !editing ? (
          <p className={`status${status.isError ? ' error' : ''}`} role="status" aria-live="polite">
            {status.message === null ? null : tr(status.message)}
          </p>
        ) : null}
      </main>

      <BottomNav active="profile" />
    </ForumShell>
  );
}
