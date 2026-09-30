/*
 * 貼文動態的資料狀態（src/forum/useFeed.ts）
 *
 * 向下捲動的無限載入、重新載入，以及離開頁面時釋放圖片 token。
 *
 * offset / hasMore / loading 刻意放進 ref 而不是 state：它們不影響任何
 * 渲染結果（畫面上只看得見 feedStatus 那行文字），放進 state 會讓每次捲動
 * 都多跑一次渲染。舊版是模組層的 let 變數，語意相同但會在多實例時互相污染。
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { errorMessage, requestJSON } from '../core';
import { msg, tr, type Message } from '../i18n';
import type { ItemsResponse, LikeResponse, ForumPost } from '../types';

export type FeedPhase = 'loading' | 'ready' | 'failed';

export interface FeedState {
  phase: FeedPhase;
  items: ForumPost[];
  /**
   * 捲動到列表底部時顯示的一行字（載入中 / 到底了 / 失敗）。
   *
   * 存的是 key+參數而不是字串：使用者切換語言時這一欄會跟著換。存字串的話畫面
   * 會停在舊語言，要等到下一次重新載入才變（見 runtime.ts 的「延後翻譯的訊息」）。
   */
  status: Message;
  message: Message;
}

/**
 * 初始狀態是函式而不是模組層常數。
 *
 * status 是要顯示的字串，模組層常數會在 import 時把它固定成「當下的語言」；
 * 使用者選了別的語言之後，進站第一眼看到的就是錯的語言。整份目錄都是同步
 * import，因此這樣取不會有任何延遲成本。
 */
function initialState(): FeedState {
  return { phase: 'loading', items: [], status: msg('feed.loadingPosts'), message: msg('feed.postsFailed') };
}

export interface FeedController {
  state: FeedState;
  /** 往下取一頁；reset=true 時清空重來。 */
  load: (reset: boolean) => void;
  /** 就地更新某一篇（按讚、留言數）。 */
  patch: (postId: number, update: Partial<ForumPost>) => void;
  /** 刪除目前已載入的貼文，並同步調整後續分頁 offset。 */
  remove: (postId: number) => void;
  items: ForumPost[];
}

export interface FeedOptions {
  pageSize?: number;
  /**
   * 列表端點。預設是全站動態；追蹤頁傳 /api/forum/following/posts 取得
   * 「我追蹤中的人」的貼文，他人個人頁傳 /api/forum/public-posts?user=<key>
   * 取得某一位使用者的貼文。
   *
   * 可以自帶查詢字串；useFeed 會依端點是否已有 `?` 選擇 `&` 或 `?` 續接
   * limit / offset。
   *
   * 為什麼是參數而不是三支 hook：三種列表的分頁、預載、錯誤處理與圖片 token
   * 釋放完全相同，後端也保證它們回同一種欄位（見 httpapi 的
   * forumPostProjection）。寫成多支 hook 等於把這幾段抄多次，而它們一定會
   * 漂移 —— 某一頁少釋放一次圖片 token 之類的問題只會在那一頁出現。
   */
  endpoint?: string;
  /**
   * 是否要送出請求。
   *
   * 需要它的只有一種情況：端點的參數來自網址，而該參數可能不存在。個人頁
   * 沒有 `?user=` 時不該對 `/api/forum/public-posts?user=` 發請求（後端會回
   * 400），而此時 phase 停在 'idle'、畫面不渲染任何狀態 —— 正是「這一頁沒有
   * 這一段內容」該有的樣子。
   *
   * 翻成 true 時會自動補上第一次載入，所以呼叫端不必自己再觸發。
   */
  enabled?: boolean;
}

const DEFAULT_ENDPOINT = '/api/forum/posts';

export function useFeed({ pageSize = 25, endpoint = DEFAULT_ENDPOINT, enabled = true }: FeedOptions = {}): FeedController {
  const [state, setState] = useState<FeedState>(initialState);

  const offsetRef = useRef(0);
  const hasMoreRef = useRef(true);
  const loadingRef = useRef(false);
  const abortRef = useRef<AbortController | null>(null);

  const load = useCallback(
    (reset: boolean) => {
      if (loadingRef.current) return;
      if (!reset && !hasMoreRef.current) return;

      if (reset) {
        offsetRef.current = 0;
        hasMoreRef.current = true;
        // 只有重來時才顯示整頁骨架；往下取一頁時保留既有文章，
        // 否則使用者每滑一次畫面就整個清空再填回。
        setState(initialState());
      }

      loadingRef.current = true;
      setState((current) => ({ ...current, status: msg('feed.loadMorePosts') }));

      const controller = new AbortController();
      abortRef.current = controller;

  void (async () => {
    try {
      /*
       * 分隔符要看 endpoint 本身帶不帶查詢字串：帶了（依作者過濾的公開貼文頁會
       * 傳 `…/public-posts?user=<key>`）就得用 `&` 續接，硬寫 `?` 會得到
       * `?user=xxx?limit=25`，後端讀到的是一個 key 尾巴多了一個 `?` 的非法值，
       * 症狀是整頁 400 或查不到人。
       */
      const separator = endpoint.includes('?') ? '&' : '?';
      const data = await requestJSON<ItemsResponse<ForumPost>>(
        `${endpoint}${separator}limit=${pageSize}&offset=${offsetRef.current}`,
        { signal: controller.signal, fallback: tr(msg('error.fallbackLoad')) },
      );
          const items = Array.isArray(data?.items) ? data.items : [];
          const hasMore = !!data.hasMore;
          offsetRef.current += items.length;
          hasMoreRef.current = hasMore;
          setState((current) => ({
            phase: 'ready',
            items: reset ? items : [...current.items, ...items],
            status: hasMore ? msg('feed.scrollMore') : msg('feed.endOfFeed'),
            message: msg('feed.postsFailed'),
          }));
        } catch (error) {
          if (controller.signal.aborted) return;
          const message = msg('feed.postsFailed');
          setState((current) =>
            reset
              ? { phase: 'failed', items: [], status: message, message }
              : { ...current, status: msg('feed.postsFailedShort') },
          );
          void errorMessage(error, tr(message));
        } finally {
          loadingRef.current = false;
        }
      })();
    },
    [endpoint, pageSize],
  );

  /*
   * 依賴 load（它隨 endpoint / pageSize 改變身分）與 enabled。
   * enabled 從 false 翻成 true 時這個 effect 會重跑並補上第一次載入。
   */
  useEffect(() => {
    if (!enabled) return undefined;
    load(true);
    return () => abortRef.current?.abort();
  }, [enabled, load]);

  const patch = useCallback((postId: number, update: Partial<ForumPost>) => {
    setState((current) => ({
      ...current,
      items: current.items.map((post) => (post.id === postId ? { ...post, ...update } : post)),
    }));
  }, []);

  const remove = useCallback((postId: number) => {
    setState((current) => {
      if (!current.items.some((post) => post.id === postId)) return current;
      offsetRef.current = Math.max(0, offsetRef.current - 1);
      return { ...current, items: current.items.filter((post) => post.id !== postId) };
    });
  }, []);

  return { state, load, patch, remove, items: state.items };
}

/* ==========================================================================
   按讚
   ========================================================================== */

/**
 * 切換讚。
 *
 * 回傳更新後的 { liked, count } 給呼叫端套用 —— 這兩個值以伺服器回應為準，
 * 本地不猜：連點兩下的結果（例如第二下其實被合併成一次取消）只有後端知道。
 */
export async function toggleLike(postId: number): Promise<{ liked: boolean; count: number }> {
  const data = await requestJSON<LikeResponse>(`/api/forum/posts/${postId}/like`, {
    method: 'POST',
    fallback: tr(msg('error.fallbackLike')),
  });
  return { liked: !!data.liked, count: data.count || 0 };
}
