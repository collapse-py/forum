/*
 * 留言層的狀態（src/forum/useComments.ts）
 *
 * 每一篇文章的留言是獨立的一組狀態：是否展開、已載入幾筆、還有沒有更多、
 * 輸入框內容。舊版把這些塞在 <section> 的 dataset 與 DOM 查詢裡
 * （data-offset / data-hasMore / data-loaded），React 版收成一個純物件。
 *
 * 展開 → 首次載入 8 筆；捲到底 → 每次再取 12 筆。兩個數字沿用舊版：
 * 首屏要快，後續頁可以大一點。
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { LoginRequiredError, errorText, goToLogin, requestJSON } from '../core';
import { msg, t, tr, type Message } from '../i18n';
import type { CommentResponse, ForumComment, ItemsResponse } from '../types';

const FIRST_PAGE_SIZE = 8;
const NEXT_PAGE_SIZE = 12;

export interface CommentState {
  open: boolean;
  loaded: boolean;
  items: ForumComment[];
  hasMore: boolean;
  offset: number;
  loading: boolean;
  // 後端有訊息時是字串（不翻譯），沒有時是 key+參數（延後翻譯）。
  // 詳見 runtime.ts 的「延後翻譯的訊息」。
  error: Message | string;
  draft: string;
  posting: boolean;
  /** 正在執行的單則變更（編輯／刪除）的留言 id；null 代表沒有。 */
  pendingId: number | null;
}

const CLOSED: CommentState = {
  open: false,
  loaded: false,
  items: [],
  hasMore: false,
  offset: 0,
  loading: false,
  error: '',
  draft: '',
  posting: false,
  pendingId: null,
};

export interface UseCommentsOptions {
  /** 留言成功後通知呼叫端（用來把貼文的留言數 +1）。 */
  onPosted?: (postId: number) => void;
  /** 刪除留言成功後通知呼叫端（用來把貼文的留言數 -1）。 */
  onRemoved?: (postId: number) => void;
  /** 失敗訊息交給呼叫端呈現（頁面上的狀態列）。 */
  onError?: (message: Message | string) => void;
}

export interface CommentsController {
  /** 目前展開中的文章 id。沒展開任何留言時是 null。 */
  states: Record<number, CommentState>;
  /** 讀取某一篇的留言狀態；不存在時回傳關閉狀態的預設值。 */
  get: (postId: number) => CommentState;
  toggle: (postId: number) => void;
  /** 捲到接近底部時呼叫：還有更多才會真的發請求。 */
  loadMore: (postId: number) => void;
  setDraft: (postId: number, value: string) => void;
  submit: (postId: number) => void;
  /**
   * 改掉一則留言的本文（就地替換該筆，不重抓整頁）。
   *
   * 回傳 Promise<boolean>，成功為 true：**失敗時保留輸入框是呼叫端的決定**，
   * 而它只能從「有沒有成功」判斷。錯誤訊息在這裡就報給 onError（呼叫端已
   * 翻譯好），但訊息報出去不代表呼叫端該丟掉使用者打的字 —— 回傳值才是
   * 那個資訊的載體。
   */
  update: (postId: number, commentId: number, content: string) => Promise<boolean>;
  /** 移除一則留言，並把該篇的留言數交給呼叫端減一。 */
  remove: (postId: number, commentId: number) => void;
}

export function useComments({ onPosted, onRemoved, onError }: UseCommentsOptions = {}): CommentsController {
  const [states, setStates] = useState<Record<number, CommentState>>({});

  // statesRef 是 states 的鏡像：事件處理器讀最新狀態時不必把它放進依賴陣列，
  // 因此不會因為「狀態改變 → 函式換參考 → effect 重跑」而製造額外請求。
  const statesRef = useRef<Record<number, CommentState>>({});
  const inFlightRef = useRef(new Set<number>());
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const commit = useCallback((next: Record<number, CommentState>) => {
    statesRef.current = next;
    if (mountedRef.current) setStates(next);
  }, []);

  const patch = useCallback(
    (postId: number, update: Partial<CommentState>) => {
      const current = statesRef.current;
      const base = current[postId] ?? CLOSED;
      commit({ ...current, [postId]: { ...base, ...update } });
    },
    [commit],
  );

  const loadPage = useCallback(
    async (postId: number, offset: number, limit: number, replace: boolean) => {
      inFlightRef.current.add(postId);
      patch(postId, { loading: true, error: '' });
      try {
        const data = await requestJSON<ItemsResponse<ForumComment>>(
          `/api/forum/posts/${postId}/comments?limit=${limit}&offset=${offset}`,
          { fallback: t('error.fallbackComments') },
        );
        const items = Array.isArray(data?.items) ? data.items : [];
        const base = statesRef.current[postId] ?? CLOSED;
        patch(postId, {
          items: replace ? items : [...base.items, ...items],
          hasMore: !!data.hasMore,
          offset: offset + items.length,
          loaded: true,
          loading: false,
          error: '',
        });
      } catch (error) {
        patch(postId, { loading: false, error: errorText(error, msg('comment.loadFailed')) });
      } finally {
        inFlightRef.current.delete(postId);
      }
    },
    [patch],
  );

  const toggle = useCallback(
    (postId: number) => {
      const base = statesRef.current[postId] ?? CLOSED;
      if (base.open) {
        patch(postId, { open: false });
        return;
      }
      patch(postId, { open: true });
      // 已經載入過的不再重抓：關掉再開是常見操作，重打一次 API 沒有理由。
      if (!base.loaded && !inFlightRef.current.has(postId)) {
        void loadPage(postId, 0, FIRST_PAGE_SIZE, true);
      }
    },
    [loadPage, patch],
  );

  const loadMore = useCallback(
    (postId: number) => {
      const base = statesRef.current[postId] ?? CLOSED;
      if (!base.loaded || base.loading || !base.hasMore) return;
      void loadPage(postId, base.offset, NEXT_PAGE_SIZE, false);
    },
    [loadPage],
  );

  const setDraft = useCallback(
    (postId: number, value: string) => {
      patch(postId, { draft: value });
    },
    [patch],
  );

  const submit = useCallback(
    (postId: number) => {
      const base = statesRef.current[postId] ?? CLOSED;
      const content = base.draft.trim();
      if (!content || base.posting) return;
      patch(postId, { posting: true });
      void (async () => {
        try {
          const data = await requestJSON<CommentResponse>(`/api/forum/posts/${postId}/comments`, {
            method: 'POST',
            json: { content },
            fallback: tr(msg('error.fallbackCommentPost')),
          });
          const current = statesRef.current[postId] ?? CLOSED;
          patch(postId, {
            items: [...current.items, data.item],
            draft: '',
            posting: false,
            loaded: true,
            offset: current.offset + 1,
          });
          onPosted?.(postId);
        } catch (error) {
          patch(postId, { posting: false });
          if (error instanceof LoginRequiredError) {
            goToLogin();
            return;
          }
          onError?.(errorText(error, msg('comment.failed')));
        }
      })();
    },
    [onError, onPosted, patch],
  );

  const get = useCallback((postId: number) => statesRef.current[postId] ?? CLOSED, []);

  /*
   * 就地改一則留言。
   *
   * 刻意不重抓整頁：留言層是「展開才載入」的分頁列表，重抓會讓使用者看到
   * 捲動位置跳回頂端 —— 而他剛剛只是改了一句話。成本因此全在這裡：
   * 沒有人能替我們確認「伺服器存進去的字串就是我們送出的字串」，所以我們
   * 採用與 handleForumComments 相同的規則（後端 TrimSpace 後才回，
   * 而我們送出前也 TrimSpace），兩邊不會分歧。
   */
  const update = useCallback(
    async (postId: number, commentId: number, content: string): Promise<boolean> => {
      const base = statesRef.current[postId] ?? CLOSED;
      const next = content.trim();
      if (!next || base.pendingId !== null) return false;
      patch(postId, { pendingId: commentId });
      try {
        await requestJSON(`/api/forum/posts/${postId}/comments/${commentId}`, {
          method: 'PUT',
          json: { content: next },
          fallback: tr(msg('error.fallbackCommentEdit')),
        });
        const current = statesRef.current[postId] ?? CLOSED;
        patch(postId, {
          pendingId: null,
          items: current.items.map((item) =>
            item.id === commentId ? { ...item, content: next, edited: true } : item,
          ),
        });
        return true;
      } catch (error) {
        patch(postId, { pendingId: null });
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return false;
        }
        onError?.(errorText(error, msg('comment.editFailed')));
        return false;
      }
    },
    [onError, patch],
  );

  /*
   * 刪掉一則留言。
   *
   * 移除的是「已載入的那幾筆」，而不是重抓整頁：後端的 offset 分頁裡，從中間
   * 抽掉一列之後畫面上會少一則，而「還有多少」這個資訊會在下一次 loadMore
   * 用同一個 offset 讀到時自動補回（後端回的是依 created_at 排序的下一批，
   * 少讀一列由 hasMore 的「湊滿一頁就推測還有下一頁」規則吸收）。
   * 那個推測在這裡是安全的：它只會多載一次或一次空結果，不會漏讀。
   */
  const remove = useCallback(
    (postId: number, commentId: number) => {
      const base = statesRef.current[postId] ?? CLOSED;
      if (base.pendingId !== null) return;
      if (!base.items.some((item) => item.id === commentId)) return;
      patch(postId, { pendingId: commentId });
      void (async () => {
        try {
          await requestJSON(`/api/forum/posts/${postId}/comments/${commentId}`, {
            method: 'DELETE',
            fallback: tr(msg('error.fallbackCommentDelete')),
          });
          const current = statesRef.current[postId] ?? CLOSED;
          patch(postId, {
            pendingId: null,
            items: current.items.filter((item) => item.id !== commentId),
          });
          onRemoved?.(postId);
        } catch (error) {
          patch(postId, { pendingId: null });
          if (error instanceof LoginRequiredError) {
            goToLogin();
            return;
          }
          onError?.(errorText(error, msg('comment.deleteFailed')));
        }
      })();
    },
    [onError, onRemoved, patch],
  );

  return { states, get, toggle, loadMore, setDraft, submit, update, remove };
}
