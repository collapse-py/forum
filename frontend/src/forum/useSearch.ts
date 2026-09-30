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

  const run = useCallback((query: string, reset: boolean) => {
    if (loadingRef.current) return;
    if (!reset && !hasMoreRef.current) return;

    loadingRef.current = true;
    if (reset) {
      offsetRef.current = 0;
      hasMoreRef.current = false;
      setState({ ...idleState(), phase: 'loading', query, status: msg('feed.searching') });
    } else {
      setState((current) => ({ ...current, status: msg('feed.searchMore') }));
    }

    // 換關鍵字時中止前一次請求：否則較慢的舊回應可能在較快的新回應之後抵達，
    // 讓畫面顯示「舊關鍵字的結果」而輸入框已經是新的。
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    void (async () => {
      try {
        const path = `/api/forum/search?q=${encodeURIComponent(query)}&limit=${PAGE_SIZE}&offset=${offsetRef.current}`;
        const data = await requestJSON<SearchResponse<ForumPost>>(path, {
          signal: controller.signal,
          fallback: tr(msg('error.fallbackSearch')),
        });
        const items = Array.isArray(data?.items) ? data.items : [];
        const total = Number(data?.total) || 0;
        offsetRef.current += items.length;
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
        const message = errorText(error, msg('feed.searchFailed'));
        setState((current) =>
          reset
            ? { ...idleState(), query, phase: 'failed', status: message, message }
            : { ...current, status: msg('feed.searchMoreFailed') },
        );
      } finally {
        loadingRef.current = false;
      }
    })();
  }, []);

  const submit = useCallback(
    (query: string) => {
      const next = query.trim();
      // 空白關鍵字等同「回到動態」：不清空畫面，讓使用者回到首頁原本的樣子。
      if (next === '') {
        abortRef.current?.abort();
        queryRef.current = '';
        hasMoreRef.current = false;
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

  useEffect(() => () => abortRef.current?.abort(), []);

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
      const total = Math.max(0, current.total - 1);
      hasMoreRef.current = offsetRef.current < total;
      return {
        ...current,
        items: current.items.filter((post) => post.id !== postId),
        total,
        status: msg('feed.searchTotal', { total }),
      };
    });
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
