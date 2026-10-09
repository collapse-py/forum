/*
 * 新增貼文頁（/forum/new）
 *
 * 職責：登入閘門、圖片上傳預覽、發佈表單，以及離開頁面時釋放沒被用到的圖片 token。
 *
 * 舊版在這裡自備一份 readAPIResponse()，專門處理「後端把未登入的寫入請求
 * 303 導到登入頁、fetch 跟著轉、最後拿到登入頁 HTML」這件事。React 版把它
 * 收進 core 的 requestJSON（LoginRequiredError），三個公開頁共用同一套判斷。
 */

import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react';

import { LoginRequiredError, errorText, goToLogin, requestJSON } from '../core';
import type { ForumProfile, UploadResponse } from '../types';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import { BottomNav, ForumNav, ForumShell, useAuth, usePwaInstall } from './shell';
import { ForumAvatar } from './ForumAvatar';
import { useMediaTokenRelease } from './useMediaTokenRelease';

/* 函式而非常數：模組層的物件會把 import 時的語言固定住，切換後就不會跟著換。 */
function authLabels(): { loggedIn: string; loggedOut: string; error: string } {
  return {
    loggedIn: t('auth.feedLoggedIn'),
    loggedOut: t('auth.feedLoggedOut'),
    error: t('auth.statusUnknown'),
  };
}

/**
 * 草稿在 localStorage 的鍵。
 *
 * 刻意帶站名之外的固定前綴而不是使用者 id：這一支頁面沒有登入身分可用
 * （useAuth 這時可能還在 checking），而「同一台裝置上的上一份未完成貼文」
 * 對單機使用的讀者來說本來就是同一個人。共用電腦的邊界在這個功能裡是
 * 已知且可接受的 —— 它與瀏覽器對整個站點的快取共用同一層信任。
 */
const DRAFT_KEY = 'forum:new-post-draft';

/** 草稿寫入的debounce 毫秒數：夠長到連續打字不會每鍵寫一次，又短到關掉分頁前來得及落盤。 */
const DRAFT_SAVE_DELAY_MS = 400;

/**
 * 讀出草稿。
 *
 * localStorage 在三種情況下會拋出或不可用：隱私模式、瀏覽器設定把它關掉、
 * 配額已滿。因此每一次存取都包在 try/catch 裡，而不是「正常情況不會錯」
 * —— 症狀會是整個發文頁（連輸入框都掛不起來）白畫面，而那是一個使用者
 * 完全無法自行處理的狀態。
 */
function loadDraft(): string {
  try {
    return window.localStorage.getItem(DRAFT_KEY) ?? '';
  } catch {
    return '';
  }
}

function saveDraft(value: string): void {
  try {
    if (value.trim() === '') window.localStorage.removeItem(DRAFT_KEY);
    else window.localStorage.setItem(DRAFT_KEY, value);
  } catch {
    /* 見 loadDraft 的說明：草稿存不下來不該擋住發文。 */
  }
}

function clearDraft(): void {
  try {
    window.localStorage.removeItem(DRAFT_KEY);
  } catch {
    /* 同上。 */
  }
}

export function NewPostPage() {
  usePageTitle('title.newPost');

  const auth = useAuth();
  const install = usePwaInstall();

  const [content, setContent] = useState(loadDraft);
  const [file, setFile] = useState<File | null>(null);
  // 存 key+參數或後端回的字串，render 時才翻譯；存字串的話切換語言後這一行不會跟著
  // 換（見 runtime.ts 的「延後翻譯的訊息」）。
  const [status, setStatus] = useState<{ message: Message | string | null; isError: boolean }>({
    message: null,
    isError: false,
  });
  const [submitting, setSubmitting] = useState(false);
  const contentRef = useRef<HTMLTextAreaElement>(null);

  const loggedIn = auth.phase === 'ready' && auth.loggedIn;

  // 登入確認後才把焦點交給內文框：未登入時表單根本不顯示，focus 會是無效呼叫。
  useEffect(() => {
    if (loggedIn) contentRef.current?.focus();
  }, [loggedIn]);

  /* --- 草稿 --------------------------------------------------------------- */

  /*
   * 草稿自動儲存。
   *
   * 存在的理由是這一頁的內容上限是 10000 字，而沒有任何提示會告訴使用者
   * 「貼上視窗時這段文字已經不見了」—— 使用者只會在按了發佈之後才發現，
   * 而那時已經來不及了。
   *
   * debounce 而不是每次 onChange 都寫：localStorage 是同步 I/O，10,000 個
   * 中文字元是約 30 KB 的 JSON 序列化，而 setItem 會把它寫進磁碟。在中文
   * 注音輸入法打字時每鍵寫一次，會讓輸入法組字的頓挫變得明顯 —— 那是一個
   * 「為了省 0.4 秒而讓打字變難」的取捨，不划算。
   *
   * 只在已登入時存：未登入時表單不顯示，存的會是從上一個 session 還原出來的
   * 內容被覆蓋成空字串。
   *
   * publishedRef 是發佈成功後的閘門：clearDraft() 與 window.location.href
   * 之間還有一個 event loop 的空隙，那段時間裡若 debounce 的計時器正好到期，
   * 它會把剛清掉的草稿寫回去 —— 症狀是使用者發完之後回到這一頁，看到自己
   * 剛剛成功發佈的那篇草稿還躺在輸入框裡。
   */
  const publishedRef = useRef(false);
  useEffect(() => {
    if (!loggedIn || publishedRef.current) return undefined;
    const timer = window.setTimeout(() => saveDraft(content), DRAFT_SAVE_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, [content, loggedIn]);

  /* --- 圖片預覽 ----------------------------------------------------------- */

  /*
   * 預覽用的 object URL 必須在「換掉檔案」與「離開頁面」時 revoke。
   *
   * 不 revoke 的症狀不會出現在畫面上：那些 URL 會一直被這個 File 物件綁住，
   * 而瀏覽器要等整個 document 卸載才會回收 —— 使用者在這一頁連續換掉二十張
   * 圖片，就會有二十份完整的圖片留在記憶體裡（50 MB 上限 × 20 是 1 GB）。
   * 因此這裡用一個 effect 依賴 file：每次換檔時 revoke 掉上一把。
   */
  const [previewURL, setPreviewURL] = useState('');
  useEffect(() => {
    if (!file) {
      setPreviewURL('');
      return undefined;
    }
    const url = URL.createObjectURL(file);
    setPreviewURL(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  /* --- 「我」的公開資料 ---------------------------------------------------- */

  /*
   * composer 那顆頭像要顯示「我自己」，而 useAuth 只回答「登入了沒」—— 暱稱與
   * 頭像住在 /api/forum/profile，因此已登入時多讀一次。
   *
   * 刻意只在 loggedIn 已經是 true 之後才問：未登入的請求會被 requireLogin 轉到
   * 登入頁，requestJSON 會把它判成 LoginRequiredError 並導走整頁 —— 而那會把
   * 「先看看這一頁長什麼樣」的訪客直接踢去登入。
   *
   * 讀失敗刻意不顯示任何錯誤：這一顆頭像不是發文的必要條件，退回顯示「你」
   * 就夠了（與後端 handleForumProfile 簽不到 media token 時退回首字同一個
   * 取捨）。也不存翻譯過的訊息 —— 那就沒有切換語言後殘留舊語言的問題。
   */
  const [myProfile, setMyProfile] = useState<{ nickname: string; avatarUrl: string }>({
    nickname: '',
    avatarUrl: '',
  });

  useEffect(() => {
    if (!loggedIn) {
      setMyProfile({ nickname: '', avatarUrl: '' });
      return undefined;
    }
    let active = true;
    void (async () => {
      try {
        const profile = await requestJSON<ForumProfile>('/api/forum/profile', {
          fallback: t('error.fallbackProfile'),
        });
        if (!active) return;
        setMyProfile({ nickname: profile.nickname ?? '', avatarUrl: profile.avatarUrl ?? '' });
      } catch {
        if (active) setMyProfile({ nickname: '', avatarUrl: '' });
      }
    })();
    return () => {
      active = false;
    };
  }, [loggedIn]);

  /*
   * 那顆頭像的 media token 也要在離開時釋放（理由見 useMediaTokenRelease）。
   *
   * 與下面上傳圖片那把 token 分成兩個 listener：那一把存在 ref 裡（上傳當下就要
   * 用，不經過 state），而這一把天生就是 state —— 硬把兩者塞進同一個機制要嘛
   * 把上傳網址改成 state（多幾次 re-render），要嘛把頭像網址改成 ref（失去這個
   * 專用的去重與去空邏輯）。兩個 pagehide 的花費是兩次 sendBeacon，而那本來就
   * 是兩個不同生命週期的東西。
   */
  const myAvatarURLs = useMemo(() => (myProfile.avatarUrl ? [myProfile.avatarUrl] : []), [myProfile.avatarUrl]);
  useMediaTokenRelease(myAvatarURLs);

  /* --- 圖片 token --------------------------------------------------------- */
  const uploadedURLRef = useRef('');
  const releasedRef = useRef(false);

  useEffect(() => {
    const release = () => {
      if (releasedRef.current || !uploadedURLRef.current) return;
      let token = '';
      try {
        token = new URL(uploadedURLRef.current, window.location.href).searchParams.get('token') ?? '';
      } catch {
        return;
      }
      if (!token) return;
      releasedRef.current = true;
      // 走 sendBeacon 而非 fetch：離頁階段的 fetch 會被瀏覽器取消。
      navigator.sendBeacon(
        '/api/forum/image-tokens/release',
        new Blob([JSON.stringify({ tokens: [token] })], { type: 'application/json' }),
      );
    };
    window.addEventListener('pagehide', release);
    return () => window.removeEventListener('pagehide', release);
  }, []);

  /* --- 發佈 --------------------------------------------------------------- */

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) return;
    setSubmitting(true);
    setStatus({ message: t('newPost.publishing'), isError: false });

    try {
      let imageURL = '';
      if (file) {
        setStatus({ message: t('newPost.uploading'), isError: false });
        const body = new FormData();
        body.append('file', file);
        const uploaded = await requestJSON<UploadResponse>('/api/forum/images', {
          method: 'POST',
          form: body,
          fallback: t('error.fallbackUpload'),
        });
        imageURL = uploaded.url ?? '';
        uploadedURLRef.current = imageURL;
      }

      setStatus({ message: t('newPost.publishing'), isError: false });
      await requestJSON('/api/forum/posts', {
        method: 'POST',
        json: { content: content.trim(), imageUrl: imageURL },
        fallback: t('error.fallbackPublish'),
      });
      // 發佈成功才清草稿：清太早的話，一旦這一步之後發生任何問題（例如導向
      // 失敗），使用者回來會看到一個空的輸入框，而內容其實還在伺服器上。
      publishedRef.current = true;
      clearDraft();
      window.location.href = '/forum';
    } catch (error) {
      if (error instanceof LoginRequiredError) {
        goToLogin();
        return;
      }
      setStatus({ message: errorText(error, msg('newPost.failed')), isError: true });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <ForumShell className="forum-new-shell">
      <ForumNav auth={auth} labels={authLabels()} loginReturn="/forum/new" install={install} />

      <main className="forum-main new-post-main">
        <a className="back-link" href="/forum">
          <span className="material-symbols-outlined" aria-hidden="true">
            arrow_back
          </span>
          {t('common.backToHome')}
        </a>

        <section className="new-post-panel">
          <div className="new-post-heading">
            {/*
              「我」的頭像。沒有暱稱時退回顯示「你」，與 React 版之前的行為一致 ——
              而有暱稱時的首字與貼文卡、個人頁讀同一份資料（/api/forum/profile），
              因此同一顆頭像在各個頁面同一個長相。

              alt 給空字串：旁邊的 <h1> 就是這個頁面的標題，而這顆頭像與它是
              同一個意思（這個頁面是「以你的身分發文」）。
            */}
            <ForumAvatar
              className="avatar avatar-you"
              url={myProfile.avatarUrl}
              name={myProfile.nickname || t('newPost.avatarYou')}
            />
            <div>
              <div className="eyebrow">{t('newPost.eyebrow')}</div>
              <h1>{t('newPost.title')}</h1>
            </div>
          </div>

          {!loggedIn ? (
            <div className="login-prompt" id="forum-login-prompt">
              <a href="/auth/google?return=%2Fforum%2Fnew">{t('newPost.loginFirst')}</a>
            </div>
          ) : null}

          {loggedIn ? (
            <form
              id="forum-form"
              onSubmit={(event) => void handleSubmit(event)}
            >
              <textarea
                id="forum-content"
                ref={contentRef}
                maxLength={10000}
                placeholder={t('newPost.contentPlaceholder')}
                required
                value={content}
                onChange={(event) => setContent(event.target.value)}
              />
              {/*
                草稿說明緊接在輸入框之下，而不是塞進頁尾那一列：頁尾那一列已經
                有「email 不會公開」與送出鈕，而草稿是「關於這個輸入框」的行為。
                刻意講清楚只有文字會被保留 —— 選好的圖片仍在裝置裡，重來時要
                再選一次（File 物件無法序列化進 localStorage，這是不可行的，
                不是沒做）。
              */}
              <p className="new-post-draft-note">{t('newPost.draftNote')}</p>
              <label className="image-upload-label" htmlFor="forum-image">
                {t('newPost.addImage')}
              </label>
              {/*
                hidden 是必要的，不只是裝飾：forum-new.html 的 <style> 裡
                `#forum-form input { display: block }` 的選擇器權重
                （ID + 型別）高於 `.image-upload-input { display: none }`，
                只有 [hidden] 的 !important 擋得住。實際控制項是上面的
                <label htmlFor>，所以把原生控制項藏起來正是預期行為。
              */}
              <input
                id="forum-image"
                className="image-upload-input"
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp"
                hidden
                onChange={(event) => setFile(event.target.files?.[0] ?? null)}
              />
              {file && previewURL ? (
                /*
                  預覽圖的來源是 URL.createObjectURL（見上面的 effect），不是
                  /api/forum/images 的回傳網址：那是檔案伺服器，而圖片此刻還在
                  使用者裝置裡、根本沒上傳。objectURL 在換檔或離頁時會被 revoke。
                */
                <div className="image-preview" id="forum-image-preview">
                  <img src={previewURL} alt={t('newPost.imagePreviewAlt')} />
                  <span className="image-preview__name">{file.name}</span>
                </div>
              ) : null}
              <div className="new-post-footer">
                <span className="composer-note">{t('newPost.emailPrivate')}</span>
                <button className="submit-button" type="submit" disabled={submitting}>
                  {submitting ? t('newPost.publishing') : t('newPost.submit')}
                </button>
              </div>
              <p className={`status${status.isError ? ' error' : ''}`} id="forum-status" role="status">
                {status.message === null ? null : tr(status.message)}
              </p>
            </form>
          ) : null}
        </section>
      </main>

      <BottomNav active="new" />
    </ForumShell>
  );
}
