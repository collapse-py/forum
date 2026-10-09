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
import type { ForumPost, ForumProfile, UploadResponse } from '../types';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import { ForumAvatar } from './ForumAvatar';
import { PostCard } from './PostCard';
import { BottomNav, ForumNav, ForumShell, usePwaInstall, type AuthState } from './shell';
import { useComments } from './useComments';
import { updateForumPost, useFeed, toggleLike } from './useFeed';
import { useFollow } from './useFollow';
import { useMediaTokenRelease } from './useMediaTokenRelease';
import { useReport } from './useReport';

/**
 * 「這次編輯對頭像做了什麼」。
 *
 * 三種狀態必須能分開，而一個字串辦不到：
 *
 *   - null  還沒碰頭像 → 儲存時送回「現在這一張」，取消時什麼都不發生。
 *   - ''    按了移除   → 儲存時送回空字串，資料庫清掉 avatar_url。
 *   - 網址  選了新圖   → 儲存時送回 /api/forum/images 給的那一份。
 *
 * 少了 null 的症狀很具體：讀取資料時把 avatarUrl 初始化進這個 state，取消編輯
 * 就會把頭像清空（因為「取消」只能把它設回空字串，而那正好是「移除」的語意）。
 */
type AvatarEdit = string | null;

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
  /** 「現在存在資料庫裡」的頭像網址。唯讀畫面與編輯中的預設值都用它。 */
  const [avatarUrl, setAvatarUrl] = useState('');
  /** 這次編輯對頭像的決議；null = 還沒碰（見 AvatarEdit）。 */
  const [avatarEdit, setAvatarEdit] = useState<AvatarEdit>(null);
  /** 頭像上傳中。與 saving 分開：上傳發生在按「儲存」之前，兩者不會同時為真。 */
  const [avatarUploading, setAvatarUploading] = useState(false);
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

  /*
   * 貼文列的圖片與作者頭像離開頁面時要釋放 token（實作見 useMediaTokenRelease）。
   *
   * avatarUrl 是第二個來源：它是 GET /api/forum/profile 簽發的 token，不屬於
   * 任何一篇貼文，而它與貼文附圖共用同一種 token、同一個釋放端點。少了它，每一
   * 次進出個人資料頁都留下一把沒人刪的 key。
   */
  useMediaTokenRelease(postItems, [avatarUrl]);

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
      setAvatarUrl(profile.avatarUrl ?? '');
      setAvatarEdit(null);
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
    // 表單的第一顆控件是暱稱，而頭像的決議從「未改動」開始：帶著上一次編輯
    // 留下來的決議進表單，會讓使用者看到一張自己不記得選過的頭像。
    setAvatarEdit(null);
    // 等 React 提交完這次 render：此刻輸入框還不存在，直接 focus 會落在 body。
    requestAnimationFrame(() => nicknameRef.current?.focus());
  };

  const cancelEditing = () => {
    setStatus({ message: '', isError: false });
    // 丟掉這次對頭像的決議：取消就是取消全部，包括已經上傳成功的那一張。
    // （已上傳的檔案會成為孤兒檔，與放棄一張貼文附圖相同，見後端
    // forumProfileRequest 的說明。）
    setAvatarEdit(null);
    setEditing(false);
  };

  /* --- 頭像 --------------------------------------------------------------- */

  /*
   * 表單裡那顆頭像顯示「即將儲存的樣子」。
   *
   * 上傳成功就換成伺服器那一張，而不是另建 URL.createObjectURL：後端回傳的
   * 網址已經帶一把立即可用的 media token，因此預覽看到的就是儲存後每個人都會
   * 看到的那一份。少了 object URL 也少一件要 revoke 的事。
   *
   * 名字用暱稱（與貼文卡同一個取法）：沒有頭像時首字要與其他地方一致。
   */
  const formAvatarURL = avatarEdit === null ? avatarUrl : avatarEdit;
  const formAvatarName = nickname.trim() || t('newPost.avatarYou');
  const hasFormAvatar = formAvatarURL !== '';

  /*
   * 選好圖片就立刻上傳，不等按「儲存」。
   *
   * 理由不是省一次點擊，而是「格式或大小不對」要在這裡就讓使用者知道 ——
   * 等到儲存才一起失敗，錯誤訊息會長得像暱稱或簡介的問題。上傳成功後手上就有
   * 完整網址，表單因此可以顯示真正的預覽。
   *
   * 它只改 avatarEdit，不寫資料庫：真正的寫入還是跟著整份資料一起（見
   * handleSubmit）。因此按「取消」不會留下任何改變。
   */
  const handleAvatarFile = async (file: File) => {
    if (avatarUploading) return;
    setAvatarUploading(true);
    setStatus({ message: t('profile.avatarUploading'), isError: false });
    try {
      const body = new FormData();
      body.append('file', file);
      const uploaded = await requestJSON<UploadResponse>('/api/forum/images', {
        method: 'POST',
        form: body,
        fallback: t('error.fallbackAvatarUpload'),
      });
      setAvatarEdit(uploaded.url ?? '');
      setStatus({ message: '', isError: false });
    } catch (error) {
      if (error instanceof LoginRequiredError) {
        goToLogin();
        return;
      }
      setStatus({ message: errorText(error, msg('error.fallbackAvatarUpload')), isError: true });
    } finally {
      setAvatarUploading(false);
    }
  };

  /** 移除頭像的決議。只是把這次編輯的目標設成空字串，還沒有送出。 */
  const removeAvatar = () => {
    setStatus({ message: '', isError: false });
    setAvatarEdit('');
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (saving || avatarUploading) return;
    setSaving(true);
    setStatus({ message: t('profile.saving'), isError: false });
    try {
      // 頭像跟著暱稱與簡介一起送出，而不是另開一個端點：後端把三者放在同一句
      // upsert 裡（見 handleForumProfile 的說明），因此它們不可能只成功一半。
      const nextAvatar = formAvatarURL;
      const profile = await requestJSON<ForumProfile>('/api/forum/profile', {
        method: 'PUT',
        json: { nickname: nickname.trim(), bio: bio.trim(), avatarUrl: nextAvatar },
        fallback: t('error.fallbackProfileSave'),
      });
      setNickname(profile.nickname ?? '');
      setBio(profile.bio ?? '');
      // 頭像的結果以「自己送出的那一份」為準：PUT 刻意不回 avatarUrl（理由見
      // handleForumProfile），而送出的值就是通過驗證後寫進資料庫的那一個。
      setAvatarUrl(nextAvatar);
      setAvatarEdit(null);
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

  /*
   * 編輯自己的貼文之後就地換掉列表裡的那一份（理由與 FeedPage 的 handleUpdate
   * 相同：重新載入整頁會把捲動位置打回頂端）。
   */
  const handleUpdate = useCallback(
    async (post: ForumPost, content: string) => {
      try {
        const next = await updateForumPost(post.id, content);
        patchPost(post.id, { content: next, edited: true });
        setStatus({ message: msg('posts.edited'), isError: false });
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          throw error;
        }
        setStatus({ message: errorText(error, msg('posts.editFailed')), isError: true });
        // 見 FeedPage 的 handleUpdate：丟回去讓 PostCard 保留輸入框。
        throw error;
      }
    },
    [patchPost],
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
      onUpdate={handleUpdate}
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
            {/*
              唯讀畫面顯示「現在這一張」（avatarUrl），不是編輯中的決議 ——
              這個區塊在 editing 時也照樣渲染，而兩者顯示不同的圖會讓人以為
              已經存進去了。alt 刻意給空字串：旁邊的 <h1> 就是這張圖的說明。
            */}
            <ForumAvatar className="profile-avatar" url={avatarUrl} name={nickname.trim() || t('newPost.avatarYou')} />
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
              {/*
                頭像欄位。與暱稱、簡介不同的地方只有一個：它是「先上傳、再跟著
                整份資料一起儲存」，因此選完圖片的當下就會有一個獨立的成功或失敗
                （見 handleAvatarFile）。表單因此要能同時呈現兩種狀態：上傳中的
                提示在選圖時出現，儲存中的提示在按儲存時出現，兩者不會同時發生。

                 排版刻意與上面兩個「標題 + 輸入框」的欄位不同：它的「值」是一顆
                 圓，橫向排在一起才不會把表單拉得過長。
              */}
              <div className="profile-avatar-field">
                <span className="profile-avatar-field-label">{t('profile.avatarLabel')}</span>
                <div className="profile-avatar-editor">
                  <ForumAvatar
                    className="profile-avatar profile-avatar--editor"
                    url={formAvatarURL}
                    name={formAvatarName}
                    label={t('profile.avatarPreviewAlt')}
                  />
                  <div className="profile-avatar-actions">
                    <label
                      className={`profile-avatar-choose${avatarUploading ? ' is-uploading' : ''}`}
                      htmlFor="profile-avatar"
                    >
                      {avatarUploading ? t('profile.avatarUploading') : t('profile.avatarChoose')}
                    </label>
                    {/*
                      hidden 而不是 display:none：.profile-panel input（0,1,1）
                      贏過 .profile-avatar-input（0,1,0），只有元素上的 hidden
                      帶的 UA !important 留得住它。真正的控制項是上面的
                      <label htmlFor>，與 /forum/new 的圖片選擇同一個寫法。
                    */}
                    <input
                      id="profile-avatar"
                      className="profile-avatar-input"
                      type="file"
                      accept="image/jpeg,image/png,image/gif,image/webp"
                      disabled={avatarUploading}
                      hidden
                      onChange={(event) => {
                        const file = event.target.files?.[0];
                        // 重設 value：同一個檔案再選一次時 onChange 不會觸發，
                        // 而使用者「選了不喜歡的圖、想改回原本那一張」正好需要它。
                        event.target.value = '';
                        if (file) void handleAvatarFile(file);
                      }}
                    />
                    {hasFormAvatar ? (
                      <button
                        className="profile-avatar-remove"
                        id="profile-avatar-remove"
                        type="button"
                        disabled={avatarUploading}
                        onClick={removeAvatar}
                      >
                        {t('profile.avatarRemove')}
                      </button>
                    ) : null}
                  </div>
                </div>
                <p className="profile-hint">{t('profile.avatarHint')}</p>
              </div>
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
