/*
 * 單篇貼文（src/forum/PostCard.tsx）
 *
 * 作者列、內文、附圖、按讚／回應，以及展開後的留言層。
 *
 * 這裡沒有任何 fetch：所有狀態都在 useFeed / useComments / useReport 裡，
 * 元件只負責把狀態畫成標記並把事件往上送。舊版靠 .posts 上的兩組事件代理
 * + closest() 逐一對應 data-id 來達成同一件事，React 版不需要那些 data 屬性，
 * 元件的 key 就足以識別是哪一篇。
 *
 * 記得：style-src 沒有 'unsafe-inline'，所以這支檔案不能有任何 style 屬性。
 * 顯示與隱藏一律用 hidden 或 class。
 */

import { useEffect, useRef, useState, type FormEvent } from 'react';

import { formatDateTime, text } from '../core';
import { tr, t } from '../i18n';
import type { ForumComment, ForumPost } from '../types';
import { FollowButton } from './FollowButton';
import type { CommentsController } from './useComments';
import type { FollowController } from './useFollow';
import type { ReportController, ReportTarget } from './useReport';

/** 距離底部多少 px 就觸發「載入更多留言」。 */
const NEAR_BOTTOM_PX = 80;

function authorURL(authorKey: string | undefined): string {
  return `/forum/others-profile?user=${encodeURIComponent(authorKey || '')}`;
}

export interface PostCardProps {
  post: ForumPost;
  canInteract: boolean;
  comments: CommentsController;
  report: ReportController;
  follow: FollowController;
  onLike: (post: ForumPost) => void;
  onDelete?: ((post: ForumPost) => Promise<void>) | undefined;
  onRequireLogin: () => void;
  /**
   * 切換追蹤成功後的頁面層副作用（可選）。
   *
   * 追蹤頁需要它：那一頁的「追蹤者的貼文」是以追蹤清單為條件查出來的，
   * 取消追蹤之後那些貼文就不該再留在畫面上。用回呼而不是讓 PostCard 知道
   * 這些，是因為「哪些貼文該顯示」屬於頁面的資料來源問題，PostCard 只負責
   * 一張卡片。
   */
  onFollowChanged?: ((userKey: string, following: boolean) => void) | undefined;
}

export function PostCard({ post, canInteract, comments, report, follow, onLike, onDelete, onRequireLogin, onFollowChanged }: PostCardProps) {
  const state = comments.get(post.id);
  const listRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const [deleting, setDeleting] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  /*
   * useComments() 回傳的 controller 每次渲染都是新的物件字面值，因此不能整包
   * 放進依賴陣列 —— 那會讓下面這個 effect 在每次父層渲染時都重跑，量捲動位置
   * 的版面計算等於白做。這裡只取出 loadMore（useCallback 參考穩定）。
   */
  const { loadMore } = comments;

  const nearBottom = () => {
    const list = listRef.current;
    if (!list) return false;
    return list.scrollTop + list.clientHeight >= list.scrollHeight - NEAR_BOTTOM_PX;
  };

  /*
   * 自動補頁。
   *
   * 舊版在每次載入成功後直接量一次捲動位置，還沒填滿可見範圍就再取一頁 ——
   * 否則留言只有兩三則、容器高度小於視窗時，使用者永遠不會觸發捲動事件，
   * 「還有更多」的訊息就會永遠掛在那裡。
   *
   * 依賴項刻意只放留言狀態：nearBottom() 讀的是 scrollHeight，會強制版面
   * 計算，25 張卡片就是 25 次，不能每次渲染都做。
   */
  useEffect(() => {
    if (!state.open) return;
    if (nearBottom()) loadMore(post.id);
  }, [state.open, state.loaded, state.loading, state.hasMore, state.items, post.id, loadMore]);

  useEffect(() => {
    if (!menuOpen) return undefined;
    const onPointerDown = (event: PointerEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) setMenuOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      setMenuOpen(false);
      menuButtonRef.current?.focus();
    };
    document.addEventListener('pointerdown', onPointerDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [menuOpen]);

  const url = authorURL(post.authorKey);
  const initial = (text(post.author) || t('post.authorAnonymous')).charAt(0);
  const authorTags = Array.isArray(post.authorTags) ? post.authorTags : [];
  const postTarget: ReportTarget = { kind: 'post', postId: post.id, commentId: 0 };

  /*
   * 追蹤鈕要不要出現。
   *
   * 自己的貼文不顯示：後端會以 400 拒絕自我追蹤，讓使用者按一顆注定失敗的
   * 按鈕只會得到一個看不懂的錯誤訊息。判斷用 authorKey（而不是暱稱）——
   * post.authorKey 是後端 publicForumKey(email) 算出來的，與
   * useFollow.state.selfKey 是同一個函式的輸出，可以直接比字串。
   *
   * 已知取捨：追蹤清單還沒載入完成時（useFollow.isSelf 因 selfKey 為空而回傳
   * false），自己的貼文也會短暫顯示追蹤鈕，等清單回來後才消失。反過來的做法
   * ——「狀態還不知道就不渲染」—— 會讓按鈕在每個頁面載入時閃進閃出，作者列因此
   * 重新排版。一個會自我修正的短暫錯態比版面跳動更可以接受，理由寫在這裡
   * 以免日後有人「修正」成反過來那種。
   */
  const authorKey = text(post.authorKey);
  const followable = authorKey !== '' && !follow.isSelf(authorKey);
  const isOwnPost = authorKey !== '' && follow.isSelf(authorKey);

  const handlePostAction = (value: string) => {
    setMenuOpen(false);
    menuButtonRef.current?.focus();
    if (value === 'report') {
      if (canInteract) report.start(postTarget);
      else onRequireLogin();
      return;
    }
    if (value !== 'delete' || !onDelete || !isOwnPost) return;
    if (!canInteract) {
      onRequireLogin();
      return;
    }
    const confirmed = window.confirm(`${t('posts.deleteTitle', { id: post.id })}\n\n${t('posts.deleteMessage')}`);
    if (!confirmed) return;
    void (async () => {
      setDeleting(true);
      try {
        await onDelete(post);
      } finally {
        setDeleting(false);
      }
    })();
  };

  return (
    <article className="post">
      <div className="post-head">
        <a className="avatar avatar-link" href={url}>
          {initial}
        </a>
        <div className="post-identity">
          <div className="author-line">
            <a className="author-link" href={url}>
              <strong>{text(post.author)}</strong>
            </a>
            {authorTags.map((tag) => (
              <span className="author-tag" key={tag}>
                {tag}
              </span>
            ))}
            {/*
              追蹤鈕是 .author-link 的同層 sibling，不可放進那個 <a> 裡：
              巢狀在錨點內的按鈕在部分瀏覽器與讀螢幕軟體下仍會觸發導覽，
              使用者按了「追蹤」卻被帶去個人頁。這也是為什麼它必須是
              <button> 而不是 <a>：它的行為是改變伺服器端狀態。
            */}
            {followable ? (
              <FollowButton
                userKey={authorKey}
                following={follow.isFollowing(authorKey)}
                busy={follow.pending.has(authorKey)}
                /*
                 * 未登入時刻意不 disabled：按下去會導去登入頁（onRequireLogin），
                 * 與同一張卡上的 ⋯ 檢舉鈕行為一致。一顆按不動的按鈕不會告訴
                 * 使用者為什麼，而「追蹤需要登入」是值得說明的一件事。
                 */
                onToggle={() => {
                  if (!canInteract) {
                    onRequireLogin();
                    return;
                  }
                  // toggle 失敗時把錯誤寫進 useFollow 的 state（由頁面的狀態列
                  // 顯示），因此在這裡吞掉 rejection —— 事件處理器丟出來只會
                  // 變成一個沒有人接的 unhandled rejection。
                  void follow
                    .toggle(authorKey)
                    .then((following) => onFollowChanged?.(authorKey, following))
                    .catch(() => undefined);
                }}
              />
            ) : null}
          </div>
          <time>{formatDateTime(post.createdAt)}</time>
        </div>
        <div className="post-menu" ref={menuRef}>
          <button
            className="more-button"
            type="button"
            aria-label={isOwnPost && onDelete ? `${t('post.report')}, ${t('posts.deleteConfirm')}` : t('post.report')}
            aria-expanded={menuOpen}
            aria-controls={`post-actions-${post.id}`}
            disabled={deleting}
            ref={menuButtonRef}
            onClick={() => setMenuOpen((open) => !open)}
          >
            ⋯
          </button>
          {menuOpen ? (
            <div className="post-menu__panel" id={`post-actions-${post.id}`}>
              <button className="post-menu__item" type="button" onClick={() => handlePostAction('report')}>
                {t('post.report')}
              </button>
              {isOwnPost && onDelete ? (
                <button
                  className="post-menu__item post-menu__item--danger"
                  type="button"
                  onClick={() => handlePostAction('delete')}
                >
                  {t('posts.deleteConfirm')}
                </button>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>

      <p>{text(post.content)}</p>

      {post.imageUrl ? (
        <img className="post-image" src={post.imageUrl} alt={t('post.imageAlt')} loading="lazy" />
      ) : null}

      <div className="post-actions">
        <button
          className={`post-like-button${post.liked ? ' liked' : ''}`}
          type="button"
          aria-label={post.liked ? t('post.unlike') : t('post.like')}
          aria-pressed={!!post.liked}
          disabled={!canInteract}
          onClick={() => onLike(post)}
        >
          <LikeIcon liked={!!post.liked} />
          <span className="post-like-count">{post.likeCount || 0}</span>
        </button>
        <button
          className="post-comment-button"
          type="button"
          aria-label={t('post.reply')}
          aria-expanded={state.open}
          onClick={() => comments.toggle(post.id)}
        >
          <CommentIcon />
          <span className="post-comment-count">{post.commentCount || 0}</span>
        </button>
      </div>

      {report.isTarget(postTarget) ? <ReportComposer report={report} /> : null}

      <section className="post-comments" hidden={!state.open}>
        <div
          className="comments-list"
          ref={listRef}
          onScroll={() => {
            if (nearBottom()) comments.loadMore(post.id);
          }}
        >
          {state.loading && !state.loaded ? <div className="loading">{t('comment.loading')}</div> : null}
          {!state.loaded && state.error ? <div className="comments-empty">{tr(state.error)}</div> : null}
          {state.loaded && state.items.length === 0 && !state.error ? (
            <div className="comments-empty">{t('comment.none')}</div>
          ) : null}
          {state.items.map((comment) => (
            <Comment
              key={comment.id}
              comment={comment}
              postId={post.id}
              canInteract={canInteract}
              report={report}
              onRequireLogin={onRequireLogin}
            />
          ))}
          {state.hasMore ? (
            <div className="comments-load-sentinel" aria-hidden="true">
              {state.loading ? t('comment.loading') : state.error ? tr(state.error) : t('comment.more')}
            </div>
          ) : null}
        </div>

        {canInteract ? (
          <form
            className="comment-form"
            onSubmit={(event: FormEvent<HTMLFormElement>) => {
              event.preventDefault();
              comments.submit(post.id);
            }}
          >
            <textarea
              maxLength={2000}
              placeholder={t('comment.placeholder')}
              required
              value={state.draft}
              onChange={(event) => comments.setDraft(post.id, event.target.value)}
            />
            <div className="comment-form-footer">
              <span className="composer-note">{t('comment.max')}</span>
              <button className="submit-button" type="submit" disabled={state.posting}>
                {state.posting ? t('common.submitting') : t('comment.submit')}
              </button>
            </div>
          </form>
        ) : null}
      </section>
    </article>
  );
}

/* ==========================================================================
   單則留言
   ========================================================================== */

interface CommentProps {
  comment: ForumComment;
  /** 檢舉留言的端點是 /api/forum/posts/{postId}/comments/report，因此需要父層的 id。 */
  postId: number;
  canInteract: boolean;
  report: ReportController;
  onRequireLogin: () => void;
}

function Comment({ comment, postId, canInteract, report, onRequireLogin }: CommentProps) {
  const target: ReportTarget = { kind: 'comment', postId, commentId: comment.id };
  return (
    <div className="comment">
      <div className="comment-head">
        <a className="comment-author" href={authorURL(comment.authorKey)}>
          {text(comment.author)}
        </a>
        <time>{formatDateTime(comment.createdAt)}</time>
        <button
          className="comment-report-button"
          type="button"
          onClick={() => (canInteract ? report.start(target) : onRequireLogin())}
        >
          {t('comment.report')}
        </button>
      </div>
      <p>{text(comment.content)}</p>
      {report.isTarget(target) ? <ReportComposer report={report} /> : null}
    </div>
  );
}

/* ==========================================================================
   {t('report.formLabel')}
   ========================================================================== */

function ReportComposer({ report }: { report: ReportController }) {
  return (
    <form
      className="comment-form"
      onSubmit={(event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (report.sending) return;
        report.submit();
      }}
    >
      <textarea
        maxLength={500}
        placeholder={t('report.reasonPlaceholder')}
        required
        value={report.reason}
        onChange={(event) => report.setReason(event.target.value)}
      />
      <div className="comment-form-footer">
        <span className="composer-note">{t('report.note')}</span>
        <button className="ghost-button" type="button" onClick={report.cancel}>
          {t('common.cancel')}
        </button>
        <button className="submit-button" type="submit" disabled={report.sending}>
          {report.sending ? t('common.submitting') : t('report.submit')}
        </button>
      </div>
    </form>
  );
}

/* ==========================================================================
   圖示
   ========================================================================== */

/*
 * 兩顆讚圖其實是同一個 Material Symbols 愛心，只是原始碼分別給了兩種
 * 路徑與 viewBox。舊版把它們當成字串在按讚後整段替換（iconNode.outerHTML），
 * 這裡改成交互分支：React 只重繪按鈕的子節點，替換節點的副作用自然消失。
 */
function LikeIcon({ liked }: { liked: boolean }) {
  return liked ? (
    <svg
      className="post-like-icon"
      aria-label={t('post.unlike')}
      fill="currentColor"
      height={24}
      role="img"
      viewBox="0 0 48 48"
      width={24}
    >
      <title>{t('post.unlike')}</title>
      <path d="M34.6 3.1c-4.5 0-7.9 1.8-10.6 5.6-2.7-3.7-6.1-5.5-10.6-5.5C6 3.1 0 9.6 0 17.6c0 7.3 5.4 12 10.6 16.5.6.5 1.3 1.1 1.9 1.7l2.3 2c4.4 3.9 6.6 5.9 7.6 6.5.5.3 1.1.5 1.6.5s1.1-.2 1.6-.5c1-.6 2.8-2.2 7.8-6.8l2-1.8c.7-.6 1.3-1.2 2-1.7C42.7 29.6 48 25 48 17.6c0-8-6-14.5-13.4-14.5z" />
    </svg>
  ) : (
    <svg
      className="post-like-icon"
      aria-label={t('post.like')}
      fill="currentColor"
      height={24}
      role="img"
      viewBox="0 0 24 24"
      width={24}
    >
      <title>{t('post.like')}</title>
      <path d="M16.792 3.904A4.989 4.989 0 0 1 21.5 9.122c0 3.072-2.652 4.959-5.197 7.222-2.512 2.243-3.865 3.469-4.303 3.752-.477-.309-2.143-1.823-4.303-3.752C5.141 14.167 2.5 12.167 2.5 9.122a4.989 4.989 0 0 1 4.708-5.218 4.21 4.21 0 0 1 3.675 1.941c.84 1.175.98 1.763 1.12 1.763s.278-.588 1.11-1.766a4.17 4.17 0 0 1 3.679-1.938m0-2a6.04 6.04 0 0 0-4.797 2.127 6.052 6.052 0 0 0-4.787-2.127A6.985 6.985 0 0 0 .5 9.122c0 3.61 2.55 5.827 5.015 7.97.283.246.569.494.853.747l1.027.918a44.998 44.998 0 0 0 3.518 3.018 2 2 0 0 0 2.174 0 45.263 45.263 0 0 0 3.626-3.115l.922-.824c.293-.26.59-.519.885-.774 2.334-2.025 4.98-4.32 4.98-7.94a6.985 6.985 0 0 0-6.708-7.218Z" />
    </svg>
  );
}

function CommentIcon() {
  return (
    <svg
      className="post-comment-icon"
      aria-label={t('post.reply')}
      fill="none"
      height={24}
      role="img"
      stroke="currentColor"
      strokeLinejoin="round"
      strokeWidth={2}
      viewBox="0 0 24 24"
      width={24}
    >
      <title>{t('post.reply')}</title>
      <path d="M20.656 17.008a9.993 9.993 0 1 0-3.59 3.615L22 22Z" />
    </svg>
  );
}
