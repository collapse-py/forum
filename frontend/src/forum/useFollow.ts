/*
 * 追蹤（src/forum/useFollow.ts）
 *
 * 一頁一份的追蹤狀態：我追蹤了哪些人（authorKey 的集合）、那份清單本身，
 * 以及切換一個對象的動作。
 *
 * 為什麼追蹤狀態是「頁面層的一個集合」而不是每篇貼文的一個欄位
 * ------------------------------------------------------------
 * 動態、追蹤動態與搜尋結果三種來源都渲染同一支 PostCard。如果把
 * 「我是否追蹤了這位作者」做成 forumPost.authorFollowed，那三個後端查詢
 * 都得各加一個相關子查詢，而且只要漏改一處，就會出現「首頁的追蹤鈕是對的、
 * 搜尋結果裡的永遠顯示未追蹤」這種只出現在單一頁面的錯誤，而且沒有任何
 * 編譯期或測試期的訊號。
 *
 * 改成頁面層讀一次清單之後：後端的 forumPost 形狀完全不必動（也讓三種來源
 * 不會漂移），前端只要拿 post.authorKey 去 Set 裡查 —— 兩者本來就是同一個
 * 函式（後端 publicForumKey）算出來的值，不需要任何對照表。
 *
 * 唯一的代價是這一頁要多一次請求，而渲染 PostCard 或顯示追蹤鈕的頁面
 * 本來就需要知道自己的追蹤狀態，因此那一次請求不是額外成本。
 *
 * autoLoad 為什麼是參數而不是永遠自動載入
 * --------------------------------------
 * GET /api/forum/follows 掛在 requireLogin 後面，而 requireLogin 對 API 路徑
 * 也回 303 轉址到登入頁。因此匿名訪客只要無條件發出這一次請求，就會被
 * requestJSON 判成 LoginRequiredError 而整頁導去登入 —— 未登入者連首頁都
 * 開不了。呼叫端必須等 useAuth() 確認已登入才把 autoLoad 打開。
 * （追蹤頁 /forum/following 本身就需要登入，頁面路由已經擋過，因此那一頁
 * 可以直接 autoLoad。）
 *
 * 單一真相來源
 * -----------
 * keys 不是獨立的一份 state，而是由 items（伺服器給的清單）與最近一次
 * toggle 的伺服器回應共同推導出來。把「目前追蹤了誰」只存在一個地方，
 * 就不會出現按鈕顯示已追蹤但清單沒有（或反過來）的中間狀態。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { LoginRequiredError, errorText, goToLogin, requestJSON } from '../core';
import { msg, tr, type Message } from '../i18n';
import type { FollowListResponse, FollowTarget, FollowToggleResponse } from '../types';

/** 清單單頁筆數，與後端 handleForumFollows 的 pageSize 一致。 */
const PAGE_SIZE = 50;

export type PeoplePhase = 'loading' | 'ready' | 'failed';

export interface FollowState {
  peoplePhase: PeoplePhase;
  /** 已載入的追蹤對象。完整清單（可能不只一頁）。 */
  items: FollowTarget[];
  /** 還有沒有下一頁。 */
  hasMore: boolean;
  /** 我自己的 publicKey；空字串代表還沒載入完成。 */
  selfKey: string;
  /**
   * 伺服器或網路的錯誤訊息。null 代表沒有。
   * 存 key+參數而不是字串，render 時才用 tr() 解析 —— 存字串的話使用者
   * 切換語言後這一行會停在舊語言（見 i18n/runtime.ts 的「延後翻譯的訊息」）。
   */
  error: Message | string | null;
}

export interface FollowController {
  state: FollowState;
  /**
   * 我追蹤中的 authorKey 集合。PostCard 與個人頁據此決定按鈕樣態。
   *
   * 由 items 推導（useMemo），切換時因為 items 換了參考，React 才會重繪。
   * 不可就地 mutate —— 那不會觸發任何重繪。
   */
  keys: ReadonlySet<string>;
  /** 該對象目前是否被追蹤。 */
  isFollowing: (key: string) => boolean;
  /** 該對象是不是我自己（自我追蹤被後端拒絕，按鈕應該藏起來）。 */
  isSelf: (key: string) => boolean;
  /**
   * 切換追蹤狀態。回傳伺服器確認後的新狀態（不是本地猜的）。
   *
   * 需要登入而未登入時會導去登入頁並回傳「原來的狀態」，不丟出例外 ——
   * 呼叫端是事件處理器，丟出來只會變成一個沒人接的 rejection。
   * 其他錯誤會丟出（並寫進 state.error），讓呼叫端決定要不要另外回報。
   */
  toggle: (key: string) => Promise<boolean>;
  /**
   * 目前有請求在途的對象 key。PostCard 據此把該顆按鈕標成 busy。
   *
   * 與 togglingRef 分開的原因同 useReport 的 inFlightRef / sending：
   * 守衛需要「在送出之前就已經知道」的同步值（不能等 render），而 busy 樣態
   * 需要觸發重繪。兩份值在 toggle 的同一段裡一起更新，因此不會不同步。
   */
  pending: ReadonlySet<string>;
  /** 清單分頁（追蹤頁的「載入更多」）。 */
  loadMorePeople: () => void;
  /**
   * 清掉錯誤訊息。
   *
   * 呼叫端在一次成功的操作之後呼叫：否則「追蹤失敗」那一行會一直留著，
   * 即使後續的操作都成功了也沒人把它收掉。
   */
  clearError: () => void;
}

function initialState(): FollowState {
  return { peoplePhase: 'loading', items: [], hasMore: false, selfKey: '', error: null };
}

export function useFollow({ autoLoad = true }: { autoLoad?: boolean } = {}): FollowController {
  const [state, setState] = useState<FollowState>(initialState);
  const [pending, setPending] = useState<ReadonlySet<string>>(() => new Set<string>());

  /*
   * offset / hasMore / loading 刻意放 ref 而不是 state：它們不影響任何渲染
   * 結果（畫面上只看得到 peoplePhase 那一行），放進 state 會讓每次切換追蹤
   * 都多跑一次渲染。語意與 useFeed 的 offsetRef 相同。
   *
   * hasMore 特別重要：load() 若從 state 讀它，load 的身分就會隨每次載入改變，
   * 而下面那個 effect 依賴 load —— 那會變成「載入 → hasMore 改變 → effect 重跑
   * → 又一次載入」的無限迴圈。
   */
  const offsetRef = useRef(0);
  const hasMoreRef = useRef(true);
  const loadingRef = useRef(false);
  const abortRef = useRef<AbortController | null>(null);

  /** in-flight 保護：連點兩下時第二次直接忽略，不送第二個請求。 */
  const togglingRef = useRef<ReadonlySet<string>>(new Set<string>());

  const load = useCallback((reset: boolean) => {
    if (loadingRef.current) return;
    if (!reset && !hasMoreRef.current) return;

    if (reset) {
      offsetRef.current = 0;
      hasMoreRef.current = true;
      setState(initialState());
    }

    loadingRef.current = true;
    /*
     * 重新載入時回到 loading（initialState 已經是 loading，不需再設一次）；
     * 翻頁（load(false)）則刻意不動 peoplePhase —— 否則每次「載入更多」都會讓
     * 整份清單閃成載入中的樣子，而那一段清單其實完整可見，只是底部多了幾列。
     */

    const controller = new AbortController();
    abortRef.current = controller;

    void (async () => {
      try {
        const data = await requestJSON<FollowListResponse>(
          `/api/forum/follows?limit=${PAGE_SIZE}&offset=${offsetRef.current}`,
          { signal: controller.signal, fallback: tr(msg('following.peopleFailed')) },
        );
        if (controller.signal.aborted) return;
        const items = Array.isArray(data?.items) ? data.items : [];
        const hasMore = !!data.hasMore;
        offsetRef.current += items.length;
        hasMoreRef.current = hasMore;
        setState((current) => ({
          peoplePhase: 'ready',
          items: reset ? items : [...current.items, ...items],
          hasMore,
          // selfKey 只在第一頁載入時取得；後續頁沿用已知的值。
          selfKey: current.selfKey || data.selfKey || '',
          error: null,
        }));
      } catch (error) {
        if (controller.signal.aborted) return;
        // 未登入不該在這裡導頁：autoLoad 已經讓呼叫端避免發出這一次請求，
        // 走到這裡代表「原本已登入、session 中途失效」，照常導去登入頁。
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return;
        }
        const message = errorText(error, msg('following.peopleFailed'));
        setState((current) =>
          reset ? { ...initialState(), peoplePhase: 'failed', error: message } : { ...current, error: message },
        );
      } finally {
        loadingRef.current = false;
      }
    })();
  }, []);

  /*
   * 依賴只有 autoLoad 與 load（後者穩定，無依賴）。autoLoad 從 false 翻成
   * true 時這個 effect 會重跑一次並載入 —— 這正是匿名訪客在 useAuth()
   * 確認登入後才開始載入的機制。
   */
  useEffect(() => {
    if (!autoLoad) return undefined;
    load(true);
    return () => abortRef.current?.abort();
  }, [autoLoad, load]);

  const keys = useMemo<ReadonlySet<string>>(() => new Set(state.items.map((item) => item.key)), [state.items]);

  const isFollowing = useCallback((key: string) => keys.has(key), [keys]);
  const isSelf = useCallback((key: string) => state.selfKey !== '' && key === state.selfKey, [state.selfKey]);

  const toggle = useCallback(
    async (key: string): Promise<boolean> => {
      if (!key) return false;
      // 連點兩下時第二次直接回傳「目前已知的狀態」：送出第二個請求會讓按鈕
      // 的樣態和使用者的直覺脫節（後端可能把兩次合併成一次切換）。
      if (togglingRef.current.has(key)) return keys.has(key);
      togglingRef.current = new Set(togglingRef.current).add(key);
      setPending(togglingRef.current);

      try {
        const data = await requestJSON<FollowToggleResponse>('/api/forum/follows', {
          method: 'POST',
          json: { key },
          fallback: tr(msg('error.fallbackFollow')),
        });
        const following = !!data.following;
        setState((current) => {
          // 以伺服器回應為準（與 toggleLike 同一條規則）：連點被合併時，
          // 本地猜的結果會是錯的。
          if (following) {
            if (current.items.some((item) => item.key === key)) return { ...current, error: null };
            /*
             * 從貼文卡加入追蹤時清單裡還沒有這個人，因此補一列讓按鈕與
             * 清單一致。nickname 留空而不是抄貼文上的顯示名：貼文上顯示的
             * 可能是那個每次請求都不同的匿名代號（forumAuthor），把它寫進
             * 清單會讓兩處顯示不同，而且下一次載入時又換成真的名字。
             * 追蹤頁下次載入時由後端補齊。
             */
            return { ...current, items: [{ key, nickname: '' }, ...current.items], error: null };
          }
          return { ...current, items: current.items.filter((item) => item.key !== key), error: null };
        });
        return following;
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return keys.has(key);
        }
        /*
         * 失敗時不動 items：按鈕維持原狀，使用者再按一次就好。這比「先樂觀
         * 更新、失敗再回滾」誠實 —— 回滾會讓按鈕閃一下，而它可能是在 400
         * （追蹤自己）這種再按一百次也不會成功的情況下發生。
         */
        setState((current) => ({ ...current, error: errorText(error, msg('follow.failed')) }));
        throw error;
      } finally {
        const next = new Set(togglingRef.current);
        next.delete(key);
        togglingRef.current = next;
        setPending(togglingRef.current);
      }
    },
    [keys],
  );

  const loadMorePeople = useCallback(() => load(false), [load]);

  const clearError = useCallback(() => setState((current) => ({ ...current, error: null })), []);

  return { state, keys, isFollowing, isSelf, toggle, pending, loadMorePeople, clearError };
}
