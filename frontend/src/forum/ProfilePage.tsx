/*
 * 個人資料頁（/forum/profile）
 *
 * 職責：讀取自己的暱稱與自我介紹、就地編輯、存回後端。
 *
 * 舊版有五個 hidden 屬性在互相開關（#profile-view / #profile-form /
 * #profile-edit / #profile-login-prompt / 登入提示），任何一處漏切就會出現
 * 「表單和唯讀畫面同時可見」。這裡用一個 phase 決定要渲染哪一種狀態，
 * 那些組合不可能再出現。
 */

import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';

import {
  LoginRequiredError,
  errorText,
  goToLogin,
  readJSONObject,
  requestJSON,
} from '../core';
import type { ForumProfile } from '../types';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import { BottomNav, ForumNav, ForumShell, usePwaInstall, type AuthState } from './shell';

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
      </main>

      <BottomNav active="profile" />
    </ForumShell>
  );
}
