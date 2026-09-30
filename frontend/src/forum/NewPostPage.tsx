/*
 * 新增貼文頁（/forum/new）
 *
 * 職責：登入閘門、圖片上傳預覽、發佈表單，以及離開頁面時釋放沒被用到的圖片 token。
 *
 * 舊版在這裡自備一份 readAPIResponse()，專門處理「後端把未登入的寫入請求
 * 303 導到登入頁、fetch 跟著轉、最後拿到登入頁 HTML」這件事。React 版把它
 * 收進 core 的 requestJSON（LoginRequiredError），三個公開頁共用同一套判斷。
 */

import { useEffect, useRef, useState, type FormEvent } from 'react';

import { LoginRequiredError, errorText, goToLogin, requestJSON } from '../core';
import type { UploadResponse } from '../types';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import { BottomNav, ForumNav, ForumShell, useAuth, usePwaInstall } from './shell';

/* 函式而非常數：模組層的物件會把 import 時的語言固定住，切換後就不會跟著換。 */
function authLabels(): { loggedIn: string; loggedOut: string; error: string } {
  return {
    loggedIn: t('auth.feedLoggedIn'),
    loggedOut: t('auth.feedLoggedOut'),
    error: t('auth.statusUnknown'),
  };
}

export function NewPostPage() {
  usePageTitle('title.newPost');

  const auth = useAuth();
  const install = usePwaInstall();

  const [content, setContent] = useState('');
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
            <div className="avatar avatar-you">{t('newPost.avatarYou')}</div>
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
              {file ? (
                <div className="image-preview" id="forum-image-preview">
                  {file.name}
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
