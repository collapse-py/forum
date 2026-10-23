/*
 * 貼文搜尋的資料狀態（src/forum/useSearch.ts）
 *
 * 公開頁的搜尋框：送出關鍵字、翻頁，以及把結果組成「和動態一樣的卡片列表」。
 *
 * 與 useFeed 的關係是分工而不是取代：
 *   - useFeed  負責「依時間新到舊的動態」，含無限捲動與 token 釋放。
 *   - useSearch 負責「依關鍵字」，結果是分頁的，且有引擎（ES / MySQL）這個
 *     useFeed 不存在的狀態。
 * FeedPage 依「有沒有輸入關鍵字」決定畫哪一個，因此兩者可以各自保持簡單，
 * 不必在同一個 hook 裡塞兩種互斥的分頁語意。
 *
 * 後端在 ES 不可用時會降級成 MySQL 的逐字比對，並以 engine 欄位告知；
 * 這裡原樣保留這個值，讓頁面能把「結果是降級模式」講出來，而不是讓使用者
 * 誤以為站上只有這幾篇符合。
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { errorText, requestJSON } from '../core';
import { msg, t, tr, type Message } from '../i18n';
import type { ForumPost, SearchResponse } from '../types';

export type SearchPhase = 'idle' | 'loading' | 'ready' | 'failed';

export interface SearchState {
  phase: SearchPhase;
  /** 後端回傳的關鍵字（已去除前後空白），可能與輸入框目前內容不同。 */
  query: string;
  items: ForumPost[];
  total: number;
  /** 'elasticsearch' | 'mysql'；phase 為 idle 時為空字串。 */
  engine: string;
  // 兩者都是「key+參數」或後端回的字串，render 時才用 tr() 解析；存字串的話
  // 切換語言後這兩行會停在舊語言（見 runtime.ts 的「延後翻譯的訊息」）。
  /**
   * idle 時是空字串：沒有任何訊息要顯示，與「有一則訊息但尚未翻譯」是兩件事，
   * 因此這裡是 union 而不是一律 Message。render 時交給 tr() 兩種都收。
   */
  status: Message | string;
  message: Message | string;
}

/** 函式而非常數：status 是顯示字串，常數會把 import 時的語言固定住。 */
function idleState(): SearchState {
  return { phase: 'idle', query: '', items: [], total: 0, engine: '', status: '', message: msg('feed.searchFailed') };
}

export interface SearchController {
  state: SearchState;
  /** 送出關鍵字（第一頁）；空白字串等同「清除搜尋」。 */
  submit: (query: string) => void;
  /** 往後取一頁結果。 */
  loadMore: () => void;
  /** 就地更新某一篇（按讚、留言數），與動態列表共用同一個 PostCard。 */
  patch: (postId: number, update: Partial<ForumPost>) => void;
  /** 刪除搜尋結果中的貼文，並同步調整總數與分頁 offset。 */
  remove: (postId: number) => void;
}

const PAGE_SIZE = 25;

/** 後端的 engine 值：'mysql' 代表這是降級後的逐字比對。 */
const MYSQL_ENGINE = 'mysql';

export function useSearch(): SearchController {
  const [state, setState] = useState<SearchState>(idleState);

  const offsetRef = useRef(0);
  const loadingRef = useRef(false);
  const hasMoreRef = useRef(false);
  // 目前送出的關鍵字存在 ref：它同時是「下一頁要帶的 query」與「回應是否仍屬於
  // 這個關鍵字」的判斷依據。放進 state 會讓 loadMore 的依賴陣列每換一個字
  // 就重建一次，而它其實不需要觸發重新渲染。
  const queryRef = useRef('');
  const abortRef = useRef<AbortController | null>(null);
  /*
   * requestIdRef 認證「哪一個請求才是最新的那一個」。
   *
   * abort() 只會讓 fetch 的 promise 稍後以 AbortError 結束，並不會讓
   * await 立刻返回；因此被中止的舊請求仍可能在**新請求已經開始之後**才
   * 走完 catch/finally。少了序號判斷，舊請求會清掉新請求的 loading 旗標，
   * 或把舊關鍵字的結果蓋回畫面上。
   */
  const requestIdRef = useRef(0);
  /*
   * itemsRef / totalRef 是 state 的鏡像，供 remove() 在 updater **之外**
   * 判斷與扣減。updater 內的副作用不具冪等性（React 可能重試或丟棄
   * updater），扣兩次會讓 offset 與總數漂移。
   */
  const itemsRef = useRef<ForumPost[]>([]);
  itemsRef.current = state.items;
  const totalRef = useRef(0);
  totalRef.current = state.total;

  const run = useCallback((query: string, reset: boolean) => {
    if (loadingRef.current) {
      /*
       * reset 一定要執行：它的語意是「用新關鍵字從第一頁重來」，若因為
       * 正在載入而丟棄，使用者會看到輸入框已更新、結果卻還是舊關鍵字。
       * 非 reset（下一頁）則維持「已在載入中就忽略」，避免重複拉同一頁。
       */
      if (!reset) return;
      abortRef.current?.abort();
    }
    if (!reset && !hasMoreRef.current) return;

    if (reset) {
      offsetRef.current = 0;
      hasMoreRef.current = false;
      setState({ ...idleState(), phase: 'loading', query, status: msg('feed.searching') });
    } else {
      setState((current) => ({ ...current, status: msg('feed.searchMore') }));
    }

    loadingRef.current = true;
    const controller = new AbortController();
    abortRef.current = controller;
    const requestId = ++requestIdRef.current;
    // 送出當下的位置必須現在讀；等到回應時 offsetRef 可能已被 reset 歸零。
    const offset = offsetRef.current;

    void (async () => {
      try {
        const path = `/api/forum/search?q=${encodeURIComponent(query)}&limit=${PAGE_SIZE}&offset=${offset}`;
        const data = await requestJSON<SearchResponse<ForumPost>>(path, {
          signal: controller.signal,
          fallback: tr(msg('error.fallbackSearch')),
        });
        // 被中止、或被更晚的 reset 取代：這一頁的結果與 offset 累加都不算數。
        if (requestId !== requestIdRef.current) return;
        const items = Array.isArray(data?.items) ? data.items : [];
        const total = Number(data?.total) || 0;
        offsetRef.current = reset ? items.length : offset + items.length;
        // 還有下一頁的判斷用 total 而不是 hasMore：搜尋端點回的是總數，
        // 「已取筆數 < total」比「這一頁有沒有塞滿」精確（尾端那一頁通常不滿）。
        hasMoreRef.current = offsetRef.current < total;
        setState((current) => ({
          phase: 'ready',
          query,
          items: reset ? items : [...current.items, ...items],
          total,
          engine: String(data?.engine ?? ''),
          status: hasMoreRef.current ? msg('feed.searchMore') : msg('feed.searchTotal', { total }),
          message: msg('feed.searchFailed'),
        }));
      } catch (error) {
        if (controller.signal.aborted) return;
        if (requestId !== requestIdRef.current) return;
        const message = errorText(error, msg('feed.searchFailed'));
        setState((current) =>
          reset
            ? { ...idleState(), query, phase: 'failed', status: message, message }
            : { ...current, status: msg('feed.searchMoreFailed') },
        );
      } finally {
        // 只讓最新請求清掉 loading；舊請求的 finally 會晚到，不能影響新請求。
        if (requestId === requestIdRef.current) {
          loadingRef.current = false;
        }
      }
    })();
  }, []);

  const submit = useCallback(
    (query: string) => {
      const next = query.trim();
      // 空白關鍵字等同「回到動態」：清空搜尋狀態，並讓任何在途請求失效。
      if (next === '') {
        abortRef.current?.abort();
        requestIdRef.current++;
        loadingRef.current = false;
        queryRef.current = '';
        hasMoreRef.current = false;
        offsetRef.current = 0;
        totalRef.current = 0;
        setState(idleState());
        return;
      }
      queryRef.current = next;
      run(next, true);
    },
    [run],
  );

  const loadMore = useCallback(() => {
    if (queryRef.current === '') return;
    run(queryRef.current, false);
  }, [run]);

  useEffect(
    () => () => {
      // 卸載後不該再有任何 setState：先讓請求失效，再中止它。
      requestIdRef.current++;
      abortRef.current?.abort();
    },
    [],
  );

  const patch = useCallback((postId: number, update: Partial<ForumPost>) => {
    setState((current) => ({
      ...current,
      items: current.items.map((post) => (post.id === postId ? { ...post, ...update } : post)),
    }));
  }, []);

  const remove = useCallback((postId: number) => {
    /*
     * 分頁位置與總數的調整放在 updater 之外（理由見 itemsRef 的說明）。
     * 「這篇還在結果裡嗎」也在外面判斷：同一個頁面可能同時掛著動態與搜尋
     * 兩份列表，刪掉只在搜尋結果裡的那一篇時，動態的 offset 不該被扣。
     */
    if (!itemsRef.current.some((post) => post.id === postId)) return;
    // 立刻從鏡像移除：同一個 tick 內重複呼叫（例如連點）才不會扣兩次。
    itemsRef.current = itemsRef.current.filter((post) => post.id !== postId);
    offsetRef.current = Math.max(0, offsetRef.current - 1);
    totalRef.current = Math.max(0, totalRef.current - 1);
    hasMoreRef.current = offsetRef.current < totalRef.current;
    const total = totalRef.current;
    setState((current) => ({
      ...current,
      items: current.items.filter((post) => post.id !== postId),
      total,
      status: hasMoreRef.current ? msg('feed.searchMore') : msg('feed.searchTotal', { total }),
    }));
  }, []);

  return { state, submit, loadMore, patch, remove };
}

/**
 * 結果狀態列的文案。engine 為 mysql 時附上降級說明。
 *
 * 這是純函式而不是元件，因此直接呼叫 t()/tr() 而不需要 useI18n() —— 它們讀的是
 * 模組層的目前語言，在任何地方都成立（見 src/i18n/runtime.ts 的說明）。
 */
export function searchStatusText(state: SearchState): string {
  if (state.phase === 'idle') return '';
  if (state.phase === 'loading') return t('feed.searching');
  if (state.phase === 'failed') return tr(state.message);
  const base = t('feed.searchFound', { total: state.total });
  if (state.engine === MYSQL_ENGINE) return t('feed.searchDegraded', { base });
  return base;
}
