/*
 * 通知（src/forum/useNotifications.ts）
 *
 * 一頁一份的通知狀態：清單、未讀數，以及「標記已讀」的兩個動作
 * （單則與全部）。兩條導覽列（頂部鈴鐺、底部通知紅點）的未讀數也從這裡讀，
 * 理由見下。
 *
 * 為什麼通知沒有「一頁一份」以外的選擇
 * ------------------------------------
 * 與 useFollow 的追蹤狀態不同：追蹤狀態會被貼文卡共用（動態、追蹤動態、搜尋
 * 三種來源都渲染同一支 PostCard），因此必須是頁面層的一個集合。通知只有一個
 * 消費端（/forum/notifications 這一頁），因此沒有那種「同一份資料被多處渲染」
 * 的問題，直接一支 hook 即可。
 *
 * 導覽列的未讀數住在 notificationStream.ts
 * -----------------------------------------
 * 未讀數的共享快照原本就在這支檔案，後來與即時推播的連線一起搬到
 * notificationStream.ts：未讀數正是那條連線最先送來的東西（ready 訊息），而
 * 紅點要同時滿足「頁面載入時正確」與「有人回覆時立刻亮」，兩者本來就是同一個
 * 問題的兩半。這裡只留下清單，並在收到推播時把它插到最前面。
 *
 * 即時推播與清單的關係
 * --------------------
 * 推播帶來的 item 與清單端點回傳的項目形狀完全相同（後端共用同一份組裝程式碼，
 * 見 forumNotificationItemFromRow），因此收到時直接插到列表最前面，而不是整份
 * 重新載入 —— 使用者正在讀的那一頁不該因為別人留了言而跳動。斷線之後重新連上
 * 才重新載入（見 subscribeNotificationResync 的說明）：那時清單可能少了斷線期間
 * 的通知，而推播不補送。
 *
 * 推播來的通知一定是未讀的，也不受這一趟的自動已讀影響：使用者正在看著通知頁
 * 時又來了一則，它不該被吞掉，否則使用者永遠不會知道多了一則。紅點因此會重新
 * 亮起來，「全部標為已讀」也重新可以按。
 *
 * 進入頁面就自動已讀（而清單仍看得到「新」的樣式）
 * ----------------------------------------------
 * 第一頁載入完成、後端也回報還有未讀時，就直接送出「全部已讀」：鈴鐺與底部
 * 導覽的紅點立刻消失，使用者不必再按「全部標為已讀」。
 *
 * 但清單上的列不跟著變成已讀樣式。這裡的 item.read 是「這一趟進來時哪些是
 * 新的」的快照，留著它使用者才看得出這次進來有哪些新通知；重新載入後才回到
 * 後端的權威狀態（全部已讀）。也因此「全部標為已讀」按鈕在自動已讀之後就是
 * 停用狀態（見 NotificationsPage 的 hasUnread）。
 *
 * 代價是翻頁載入的較舊通知不會有這個樣式：那些列是自動已讀之後才向後端拿
 * 的，後端已經把它們標成已讀（要留住「當時是否未讀」得在後端另存一份）。
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { LoginRequiredError, errorText, goToLogin, requestJSON } from '../core';
import { msg, tr, type Message } from '../i18n';
import type { ForumNotificationItem, NotificationListResponse, NotificationUnreadResponse } from '../types';
import { publishUnreadCount, subscribeNotificationPushes, subscribeNotificationResync } from './notificationStream';

/** 清單單頁筆數，與後端 listForumNotifications 的 initialPageSize 一致。 */
const PAGE_SIZE = 20;

/**
 * 送出「全部已讀」，並把後端回報的未讀數寫回共享快照；回傳剩下的未讀數。
 *
 * 兩個呼叫端共用：進入通知頁時的自動已讀與使用者按下的「全部標為已讀」。差別
 * 只在事後怎麼處理清單 —— 自動已讀刻意不動列的 read 旗標（見檔頭），按鈕則把
 * 已載入的列一起標成已讀。
 */
async function postMarkAllRead(): Promise<number> {
  const data = await requestJSON<NotificationUnreadResponse>('/api/forum/notifications', {
    method: 'POST',
    // id 0／省略都代表「全部已讀」；這裡送 0 讓請求主體形狀固定。
    json: { id: 0 },
    fallback: tr(msg('notification.markReadFailed')),
  });
  const unread = Math.max(0, data.unreadCount ?? 0);
  publishUnreadCount(unread);
  return unread;
}

export type NotificationsPhase = 'loading' | 'ready' | 'failed';

export interface NotificationsState {
  phase: NotificationsPhase;
  /** 已載入的通知，最新的在最前面（後端以 id DESC 排序）。 */
  items: ForumNotificationItem[];
  /** 還有沒有下一頁。 */
  hasMore: boolean;
  /** 未讀數。與列表一起由後端回，也是導覽列紅點的來源（進頁自動已讀後即為 0）。 */
  unreadCount: number;
  /** 伺服器或網路的錯誤訊息；null 代表沒有。 */
  error: Message | string | null;
}

export interface NotificationsController {
  state: NotificationsState;
  /** 分頁（通知頁的「載入更多」）。 */
  loadMore: () => void;
  /**
   * 重新載入第一頁。清單上「進來時未讀」的樣式也會一併回到後端的權威狀態
   * （全部已讀），見檔頭。
   */
  reload: () => void;
  /** 把一則標成已讀。成功後就地更新那一列與未讀數。 */
  markRead: (id: number) => Promise<void>;
  /** 把全部未讀標成已讀，並清掉清單上的「新」樣式。 */
  markAllRead: () => Promise<void>;
  /** 是否有一則「全部已讀」請求在途（按鈕據此顯示忙碌狀態）。 */
  markingAll: boolean;
}

function initialState(): NotificationsState {
  return { phase: 'loading', items: [], hasMore: false, unreadCount: 0, error: null };
}

export function useNotifications({ autoLoad = true }: { autoLoad?: boolean } = {}): NotificationsController {
  const [state, setState] = useState<NotificationsState>(initialState);
  const [markingAll, setMarkingAll] = useState(false);

  /*
   * offset / hasMore / loading 放 ref 而不是 state，理由與 useFollow 完全相同：
   * 它們不影響渲染結果，放進 state 只會讓每次分頁都多跑一次渲染；而 hasMore
   * 若從 state 讀，load 的身分會隨載入改變，effect 就會無限重跑。
   */
  const offsetRef = useRef(0);
  const hasMoreRef = useRef(true);
  const loadingRef = useRef(false);
  const abortRef = useRef<AbortController | null>(null);

  /**
   * 清單裡出現過的最大 id（最新的那一則）。因為列表是 id 遞減、而新通知的 id 必定
   * 比現有的都大，這個數字就足以回答「推播來的這一則是不是已經在清單裡了」——
   * 不需要為此維護一整個 id 集合。
   */
  const maxSeenIdRef = useRef(0);

  /** 防止連點「全部已讀」送出兩次（第二次沒有可更新的列，但仍是多餘請求）。 */
  const markingAllRef = useRef(false);

  /**
   * 進入通知頁的自動已讀（見檔頭）。
   *
   * 寫後端、共享快照與頁面自己的未讀數，但刻意不動 items 的 read 旗標：那些旗標
   * 是「這一趟進來時哪些是新的」的快照，清單要留著才看得出新通知。失敗時把錯誤
   * 放進 state.error 而不是吞掉 —— 後果是可見的（紅點不會消失），而且使用者可以
   * 直接按「全部標為已讀」重試。
   */
  const autoMarkAllRead = useCallback((): void => {
    void postMarkAllRead()
      .then((unread) => {
        // 頁面自己的未讀數也要歸零：按鈕的停用狀態看它（見 NotificationsPage
        // 的 hasUnread）。清單的 read 旗標刻意不動，見上。
        setState((current) => ({ ...current, unreadCount: unread }));
      })
      .catch((error: unknown) => {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return;
        }
        setState((current) => ({ ...current, error: errorText(error, msg('notification.markReadFailed')) }));
      });
  }, []);

  const load = useCallback((reset: boolean) => {
    if (loadingRef.current) return;
    if (!reset && !hasMoreRef.current) return;

    if (reset) {
      offsetRef.current = 0;
      hasMoreRef.current = true;
      // 清單即將被整份換掉，因此「看過的最大 id」也一起歸零：重載之後的推播
      // 要以新的清單為基準判斷新舊。
      maxSeenIdRef.current = 0;
      setState(initialState());
    }

    loadingRef.current = true;
    const controller = new AbortController();
    abortRef.current = controller;

    void (async () => {
      try {
        const data = await requestJSON<NotificationListResponse>(
          `/api/forum/notifications?limit=${PAGE_SIZE}&offset=${offsetRef.current}`,
          { signal: controller.signal, fallback: tr(msg('notification.loadFailed')) },
        );
        if (controller.signal.aborted) return;
        const items = Array.isArray(data?.items) ? data.items : [];
        const hasMore = !!data.hasMore;
        offsetRef.current += items.length;
        hasMoreRef.current = hasMore;
        for (const item of items) {
          if (item.id > maxSeenIdRef.current) maxSeenIdRef.current = item.id;
        }
        setState((current) => {
          // 翻頁時濾掉已經在畫面上的列。正常情況不會有重複（推播插入時會把
          // offset 一起推進一格，見訂閱推播的 effect），但「載入更多的請求正好
          // 與一則推播交錯」時後端可能把同一列再回一次。濾掉的成本是一行程式，
          // 而重複的一列在畫面上非常明顯。
          const known = new Set(current.items.map((item) => item.id));
          const appended = items.filter((item) => !known.has(item.id));
          return {
            phase: 'ready',
            items: reset ? items : [...current.items, ...appended],
            hasMore,
            // 未讀數只在第一頁（reset）時採用：翻頁時後端回的是同一個數字，
            // 但列表新增的列不會改變未讀數，沿用目前的即可。
            unreadCount: reset ? Math.max(0, data.unreadCount ?? 0) : current.unreadCount,
            error: null,
          };
        });
        // 列表與導覽列的未讀數同步：寫回共享快照，鈴鐺與底部紅點一起更新；
        // 還有未讀的話就順手把它們全部標成已讀（見檔頭的自動已讀）。
        if (reset) {
          const unread = Math.max(0, data.unreadCount ?? 0);
          publishUnreadCount(unread);
          if (unread > 0) autoMarkAllRead();
        }
      } catch (error) {
        if (controller.signal.aborted) return;
        // 這一頁的路由掛在 requireLogin 之後，正常不會走到未登入；但 session
        // 中途失效仍應導去登入頁（與 useFollow 同一條路徑）。
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return;
        }
        const message = errorText(error, msg('notification.loadFailed'));
        setState((current) =>
          reset ? { ...initialState(), phase: 'failed', error: message } : { ...current, error: message },
        );
      } finally {
        loadingRef.current = false;
      }
    })();
  }, [autoMarkAllRead]);

  const loadMore = useCallback(() => load(false), [load]);
  const reload = useCallback(() => load(true), [load]);

  useEffect(() => {
    const unsubscribePush = subscribeNotificationPushes((item, unreadCount) => {
      // 這一則已經在清單裡（例如清單請求比推播晚到）：只要更新未讀數，不要
      // 插入第二列。id 遞增且列表遞減，因此「大於看過的最大 id」就等價於「新的一則」。
      if (item.id <= maxSeenIdRef.current) {
        setState((current) => ({ ...current, unreadCount }));
        return;
      }
      maxSeenIdRef.current = item.id;
      // 清單頂端多了一列，接下來翻頁的起點也要跟著 +1：後端以 offset（位置）
      // 分頁，而新的通知會把它後面每一則的位置往後推一格。少了這一行，
      // 「載入更多」會把已經在畫面上的那一列再拿回來一次。
      offsetRef.current += 1;
      setState((current) => ({ ...current, items: [item, ...current.items], unreadCount }));
    });

    return unsubscribePush;
  }, []);

  useEffect(() => {
    if (!autoLoad) return undefined;
    load(true);
    // 重新連上（非第一次的 ready）時補一次清單：斷線期間的通知沒有被推播，
    // 只有重新查詢才拿得到（見 notificationStream.ts 的檔頭）。
    const unsubscribeResync = subscribeNotificationResync(() => reload());
    return () => {
      unsubscribeResync();
      abortRef.current?.abort();
    };
  }, [autoLoad, load, reload]);

  const markRead = useCallback(async (id: number): Promise<void> => {
    if (id <= 0) return;
    try {
      const data = await requestJSON<NotificationUnreadResponse>('/api/forum/notifications', {
        method: 'POST',
        json: { id },
        fallback: tr(msg('notification.markReadFailed')),
      });
      const unread = Math.max(0, data.unreadCount ?? 0);
      setState((current) => ({
        ...current,
        items: current.items.map((item) => (item.id === id ? { ...item, read: true } : item)),
        unreadCount: unread,
      }));
      publishUnreadCount(unread);
    } catch (error) {
      if (error instanceof LoginRequiredError) {
        goToLogin();
        return;
      }
      setState((current) => ({ ...current, error: errorText(error, msg('notification.markReadFailed')) }));
      throw error;
    }
  }, []);

  const markAllRead = useCallback(async (): Promise<void> => {
    if (markingAllRef.current) return;
    markingAllRef.current = true;
    setMarkingAll(true);
    try {
      const unread = await postMarkAllRead();
      setState((current) => ({
        ...current,
        items: current.items.map((item) => (item.read ? item : { ...item, read: true })),
        unreadCount: unread,
        error: null,
      }));
    } catch (error) {
      if (error instanceof LoginRequiredError) {
        goToLogin();
        return;
      }
      setState((current) => ({ ...current, error: errorText(error, msg('notification.markReadFailed')) }));
      throw error;
    } finally {
      markingAllRef.current = false;
      setMarkingAll(false);
    }
  }, []);

  return { state, loadMore, reload, markRead, markAllRead, markingAll };
}
