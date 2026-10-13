/*
 * 留言層的狀態（src/forum/useComments.ts）
 *
 * 每一篇文章的留言是獨立的一組狀態：是否展開、已載入幾筆、還有沒有更多、
 * 輸入框內容。舊版把這些塞在 <section> 的 dataset 與 DOM 查詢裡
 * （data-offset / data-hasMore / data-loaded），React 版收成一個純物件。
 *
 * 展開 → 首次載入 8 筆；捲到底 → 每次再取 12 筆。兩個數字沿用舊版：
 * 首屏要快，後續頁可以大一點。
 *
 * 層級：留言可以是另一則留言的回覆（最多三層，見 MAX_COMMENT_DEPTH）。回傳的 items
 * 仍然是**扁平**的一串（後端就是這麼給的），組樹交給渲染端（PostCard 的
 * buildCommentTree，見 commentTree.ts）—— 因為分頁的 offset 語意在扁平列表上才
 * 單純，而「哪些已載入」也是同一件事。
 *
 * items 與 local 分兩個桶，而這是本檔最容易搞錯的一件事：
 *
 *   - items：後端分頁回傳的列，offset 的帳以它為準（「已經從伺服器讀了幾列」）。
 *   - local：這個工作階段本地新增的留言，尚未被任何一頁讀到。
 *
 * 為什麼不能合成一個陣列：後端的分頁是 `ORDER BY created_at ASC LIMIT ? OFFSET ?`，
 * offset 是**從最舊一則算起的絕對位置**。新留言是最新的一則，位於整串末端而不是
 * 陣列中間 —— 把它 append 進 items 再讓 offset 加一，下一次「載入更多」就會從
 * offset+1 開始，正好跳過原本排在 offset 上的那一則既有留言，而它不會再出現
 * （症狀是「明明有 20 則，看到 19 則」）。分桶之後新留言永遠排在 items 之後
 * —— 那也正是 created_at 遞增下它該在的位置 —— 而 offset 只數伺服器讀過的列，
 * 兩個問題同時消失。
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { LoginRequiredError, errorText, goToLogin, requestJSON } from '../core';
import { msg, t, tr, type Message } from '../i18n';
import type { CommentResponse, ForumComment, ItemsResponse } from '../types';

const FIRST_PAGE_SIZE = 8;
const NEXT_PAGE_SIZE = 12;

/**
 * DELETE /api/forum/posts/{id}/comments/{cid} 的回應。
 *
 * removed 是後端真的刪掉的留言數（含被連帶刪掉的回覆子樹），由
 * forumCommentSubtreeDelete 的 RowsAffected 而來。前端需要它來把貼文的留言數
 * 減去正確的筆數：本地只數得到已載入的兩桶，而未載入的子孫也會一起消失。
 */
interface DeleteCommentResponse {
  ok: boolean;
  removed?: number;
}

/**
 * 留言層級上限，與後端 forumCommentMaxDepth 一致。第三層不能再被回覆。
 *
 * 兩處用到它：remove 要走訪子樹（深度以此夾住，壞資料也不會無限迴圈），以及
 * 渲染端決定「這一則還能不能出現回覆鈕」。匯出而不是讓兩個檔案各寫一個 3。
 */
export const MAX_COMMENT_DEPTH = 3;

export interface CommentState {
  open: boolean;
  loaded: boolean;
  /** 後端分頁回傳的留言（依 created_at 遞增）。offset 的帳只算這一桶。 */
  items: ForumComment[];
  /**
   * 這個工作階段本地新增、尚未被任何一頁讀到的留言（依 created_at 遞增的尾端）。
   *
   * 與 items 分桶的理由見檔頭：合成一個陣列會讓 offset 每次新增留言後少讀一列。
   * 渲染時永遠排在 items 之後 —— 新留言的 created_at 晚於一切已載入的留言。
   */
  local: ForumComment[];
  hasMore: boolean;
  offset: number;
  loading: boolean;
  // 後端有訊息時是字串（不翻譯），沒有時是 key+參數（延後翻譯）。
  // 詳見 runtime.ts 的「延後翻譯的訊息」。
  error: Message | string;
  draft: string;
  posting: boolean;
  /** 正在執行的單則變更（編輯／刪除／回覆）的留言 id；null 代表沒有。 */
  pendingId: number | null;
}

const CLOSED: CommentState = {
  open: false,
  loaded: false,
  items: [],
  local: [],
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
  /**
   * 刪除留言成功後通知呼叫端（用來把貼文的留言數減去實際移除的筆數）。
   *
   * 第二個參數是後端回報的 removed，不是本地數的：刪除一則留言會連同它的回覆
   * 一起刪（後端子樹刪除），而未載入的子孫也會一起消失 —— 用本地數字會讓留言數
   * 虛高到下次重載。呼叫端不須自己數，它沒有樹的資訊，而 controller 與後端有。
   */
  onRemoved?: (postId: number, removed: number) => void;
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
   * 回覆某一則留言（parentId 非 0）。
   *
   * 與 submit 分成兩支而不是加一個選填參數：回覆的成功與失敗要讓**那一則留言
   * 底下的輸入框**知道（它靠自己的一個區域狀態決定要不要關掉），而表單的主提交
   * 按鈕已經占用 posting。兩件事的狀態宿主不同，合成一個函式只會讓呼叫端多傳
   * 一個它其實不需要的參數。
   *
   * 回傳 Promise<boolean>，語意與 update 相同：失敗時保留輸入框是呼叫端的決定。
   */
  reply: (postId: number, parentId: number, content: string) => Promise<boolean>;
  /**
   * 改掉一則留言的本文（就地替換該筆，不重抓整頁）。
   *
   * 回傳 Promise<boolean>，成功為 true：**失敗時保留輸入框是呼叫端的決定**，
   * 而它只能從「有沒有成功」判斷。錯誤訊息在這裡就報給 onError（呼叫端已
   * 翻譯好），但訊息報出去不代表呼叫端該丟掉使用者打的字 —— 回傳值才是
   * 那個資訊的載體。
   */
  update: (postId: number, commentId: number, content: string) => Promise<boolean>;
  /** 移除一則留言（含它底下的回覆），並把該篇的留言數交給呼叫端減去移除的筆數。 */
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
            // parentId 固定 0：最下方那個輸入框是「在這篇文章下留言」，回覆某一則
            // 要走 reply()。0 同時是後端「沒有父留言」的表示，送出去不會改變語意。
            json: { content, parentId: 0 },
            fallback: tr(msg('error.fallbackCommentPost')),
          });
          const current = statesRef.current[postId] ?? CLOSED;
          // 進 local 而不是 items，而 offset 不動：見檔頭的兩個桶。新留言是整串
          // 最新的一則，排在最尾端就是它正確的位置，而 offset 繼續指著「下一批
          // 要從伺服器的第幾列開始讀」。
          patch(postId, {
            local: [...current.local, data.item],
            draft: '',
            posting: false,
            loaded: true,
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

  /*
   * 回覆某一則留言。
   *
   * 新留言進 local 桶而不是 items：offset 的帳以「伺服器讀過幾列」為準，而新留言
   * 還沒被任何一頁讀到（它甚至在送出當下才存在）。把它 append 進 items 再讓
   * offset 加一，下一次載入更多會跳過一則既有留言 —— 見檔頭的說明。
   *
   * 組樹只認 parentId，位置由渲染端決定：local 排在 items 之後，而新留言的
   * created_at 晚於一切已載入的留言，那正是 created_at 遞增下它該在的位置。
   *
   * pendingId 記的是**父留言**的 id：這樣父留言自己的編輯／刪除鈕在回覆送出期間
   * 一起停用（同一則留言不該同時有兩個進行中的變更），而回覆表單的送出鈕讀的
   * 正是同一個 pendingId，不需要多帶一個狀態。
   */
  const reply = useCallback(
    async (postId: number, parentId: number, content: string): Promise<boolean> => {
      const base = statesRef.current[postId] ?? CLOSED;
      const text = content.trim();
      if (!text || parentId <= 0) return false;
      // 另一個變更進行中時，這裡必須**出聲**：呼叫端（第二則留言的回覆表單）的
      // 送出鈕沒有停用，靜靜 return false 會讓使用者按了一顆沒有任何反應的按鈕。
      if (base.pendingId !== null) {
        onError?.(msg('comment.pending'));
        return false;
      }
      patch(postId, { pendingId: parentId });
      try {
        const data = await requestJSON<CommentResponse>(`/api/forum/posts/${postId}/comments`, {
          method: 'POST',
          json: { content: text, parentId },
          fallback: tr(msg('error.fallbackCommentPost')),
        });
        const current = statesRef.current[postId] ?? CLOSED;
        patch(postId, {
          pendingId: null,
          local: [...current.local, data.item],
          loaded: true,
        });
        onPosted?.(postId);
        return true;
      } catch (error) {
        patch(postId, { pendingId: null });
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return false;
        }
        onError?.(errorText(error, msg('comment.failed')));
        return false;
      }
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
        // 兩個桶都要替換：編輯可能落在剛po出去的那一則上（local）。
        const replace = (item: ForumComment) =>
          item.id === commentId ? { ...item, content: next, edited: true } : item;
        patch(postId, {
          pendingId: null,
          items: current.items.map(replace),
          local: current.local.map(replace),
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
   * 後端刪的是整棵子樹（forumCommentSubtreeJoin / …Condition 組出的 SQL，見
   * forum_handlers.go），因此這裡移除的也是子樹：只濾掉那一筆的話，它的回覆會
   * 留在畫面上，而重新載入後它們才消失 —— 一個會自己修正的錯態，但沒理由留著。
   *
   * 移除的是「已載入的那幾筆」，而不是重抓整頁：後端的 offset 分頁裡，從中間
   * 抽掉幾列之後畫面上會少幾則，而「還有多少」這個資訊會在下一次 loadMore
   * 用同一個 offset 讀到時自動補回（後端回的是依 created_at 排序的下一批）。
   * 那個推測在這裡是安全的：它只會多載一次或一次空結果，不會漏讀 —— 前提是
   * offset 跟著下修（見下一個說明）。
   *
   * 回傳給呼叫端的筆數是**後端回報的 removed**（RowsAffected），不是本地數出來的
   * 那個數字：本地只看得到已載入的列，而未載入的子孫（使用者沒捲到底就刪）也會
   * 被後端一起刪掉。用本地數字會讓貼文的留言數虛高到下次重載。
   */
  const remove = useCallback(
    (postId: number, commentId: number) => {
      const base = statesRef.current[postId] ?? CLOSED;
      if (base.pendingId !== null) return;
      const loaded = [...base.items, ...base.local];
      if (!loaded.some((item) => item.id === commentId)) return;
      patch(postId, { pendingId: commentId });
      void (async () => {
        try {
          const data = await requestJSON<DeleteCommentResponse>(
            `/api/forum/posts/${postId}/comments/${commentId}`,
            {
              method: 'DELETE',
              fallback: tr(msg('error.fallbackCommentDelete')),
            },
          );
          const current = statesRef.current[postId] ?? CLOSED;
          // 子樹內的 id：從被刪的那一則沿著 parentId 往下走，只走已載入的兩桶。
          // 走訪深度由 MAX_COMMENT_DEPTH 夾住，因此即使資料出了意想不到的結構也不會
          // 無限迴圈（parentId 指向自己之類的壞資料）。
          const doomed = new Set<number>([commentId]);
          for (let level = 0; level < MAX_COMMENT_DEPTH; level += 1) {
            const before = doomed.size;
            for (const item of [...current.items, ...current.local]) {
              const parent = Number(item.parentId) || 0;
              if (parent > 0 && doomed.has(parent)) doomed.add(item.id);
            }
            if (doomed.size === before) break;
          }
          const notDoomed = (item: ForumComment) => !doomed.has(item.id);
          const items = current.items.filter(notDoomed);
          const local = current.local.filter(notDoomed);
          patch(postId, {
            pendingId: null,
            items,
            local,
            // offset 只數「從伺服器讀了幾列」，因此刪掉幾列 server rows 就要下修
            // 幾列：硬留著會讓下一次載入更多剛好跳過一則（刪除讓之後的列全部前移）。
            offset: Math.max(0, current.offset - (current.items.length - items.length)),
          });
          // 後端回報的才是真的少了幾則（含未載入的子孫）；讀不到欄位時退回本地計數。
          const loadedRemoved =
            current.items.length + current.local.length - items.length - local.length;
          const removed = typeof data?.removed === 'number' ? data.removed : loadedRemoved;
          onRemoved?.(postId, removed);
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

  return { states, get, toggle, loadMore, setDraft, submit, reply, update, remove };
}
