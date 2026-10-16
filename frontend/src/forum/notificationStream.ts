/*
 * 通知的即時串流與未讀數（src/forum/notificationStream.ts）
 *
 * 這一支負責兩件事，而它們其實是同一件事的兩半：
 *   1. 未讀數的共享快照 —— 頂部導覽列鈴鐺與底部導覽列紅點的來源。
 *   2. 與後端 /api/forum/notifications/ws 的 WebSocket 連線 —— 後端在通知產生的
 *      當下推過來（見後端 forum_notification_stream.go），紅點因此不必等到下一次
 *      頁面載入才出現。
 *
 * 為什麼需要連線
 * --------------
 * 沒有它，紅點只能靠「每次頁面載入問一次」更新：使用者停在貼文頁讀留言時，
 * 不會知道有人剛回了他。連線把「有通知」這件事變成即時的，而不是等下一次導覽。
 *
 * 為什麼仍然保留「每次載入問一次」的 GET
 * --------------------------------------
 * 連線可能根本不成立：反向代理沒轉發 Upgrade、瀏覽器或網路限制、伺服器額度已滿、
 * 使用者的瀏覽器太舊。那些情況下的行為必須與加入這個功能之前完全相同，因此
 * GET 是後備，不是多餘的重複。兩者互相補位：
 *   - GET 保證「這一頁至少會更新一次紅點」。
 *   - 連線上的 ready 保證「每一次重新連線都把數字對回後端」。
 *
 * 重連策略
 * --------
 * 指數退避（1s → 2s → 4s → …上限 30s）加抖動，收到任何一則伺服器訊息就重置。
 * 抖動是為了避免伺服器重啟後所有分頁在同一瞬間回來（那會形成一波自己製造的
 * 尖峰）。重試全程靜默：這一條連線只影響「即時性」，斷了不該跳任何提示。
 *
 * 唯一的例外是被同一使用者的較新連線取代（close code 4000）：那時刻意**不**
 * 重連 —— 額度是每個使用者、跨裝置共用的，重連只會把對方踢掉、然後換自己被
 * 踢。被取代的分頁因此退回「每次載入問一次」，紅點照常運作，只是不即時。
 *
 * 為什麼重新連線要重新載入清單
 * ---------------------------
 * 斷線期間產生的通知不會補推（伺服器不保存「誰在線上過」），因此連上之後只知道
 * 最新的未讀數，不知道清單少了哪幾則。通知頁因此在**非第一次**的 ready 之後
 * 重新載入第一頁（見 subscribeNotificationResync）：那個動作同時把清單補齊並
 * 重新對齊未讀數，代價是一次查詢，而它只發生在斷線之後。
 */

import { useEffect, useState } from 'react';

import { requestJSON } from '../core';
import type { ForumNotificationItem, NotificationUnreadResponse } from '../types';

/* ==========================================================================
   未讀數的共享快照
   ==========================================================================
   模組層而不是 React state：導覽列的鈴鐺與底部導覽的紅點在每一頁都同時存在，
   若各自持有 state 就會有兩次 /unread-count 請求，而標記已讀之後還要讓兩份
   state 同步。每個頁面載入是獨立的 JS 環境，因此快照的生命週期就是「這一頁」，
   重新載入即重新查詢（與 useAuth 讀 /api/check 同一個節奏，刻意不輪詢）。 */

/** 目前已知的未讀數；null 代表這一頁還沒問過。 */
let unreadSnapshot: number | null = null;

/** 進行中的查詢。同時有多個元件要未讀數時，全部等這一個 Promise。 */
let unreadPending: Promise<void> | null = null;

/** 訂閱者（頂部鈴鐺與底部導覽的通知紅點）。 */
const unreadSubscribers = new Set<(count: number) => void>();

/**
 * 快照的版本號，每次寫入 +1。
 *
 * 擋的是「較早發出、較晚回來」的回應。導覽列的 /unread-count、通知頁的自動
 * 已讀與連線上的 ready 是各自獨立的來源，回應沒有先後保證：若自動已讀先把未讀數
 * 歸零，那個帶著舊數字的 /unread-count 才回來，紅點就會被過期的值復活。發
 * 請求時記下版本，回來時若版本已被推進就丟棄這次結果。
 */
let unreadRevision = 0;

/** 寫回快照並通知所有訂閱者。 */
export function publishUnreadCount(count: number): void {
  unreadSnapshot = Math.max(0, count);
  unreadRevision += 1;
  for (const subscriber of unreadSubscribers) subscriber(unreadSnapshot);
}

/**
 * 取得未讀數（每頁最多一次）。
 *
 * 失敗一律退化成「維持目前值」（初始為 0）：一個紅點不該讓導覽列出錯，而
 * 失敗時刻意不寫入快照，讓下一個掛載的元件還能再試一次。
 */
function ensureUnreadCount(): Promise<void> {
  if (unreadSnapshot !== null) return Promise.resolve();
  if (unreadPending !== null) return unreadPending;
  const revision = unreadRevision;
  unreadPending = requestJSON<NotificationUnreadResponse>('/api/forum/notifications/unread-count')
    .then((data) => {
      // 已經有更新的來源（列表載入、自動已讀、連線的 ready）寫過快照就不覆蓋它
      // （見 unreadRevision）。
      if (unreadRevision === revision) publishUnreadCount(data.unreadCount ?? 0);
    })
    .catch(() => {
      // 刻意吞掉：鈴鐺與紅點是附加資訊，取不到就當作沒有未讀。
    })
    .finally(() => {
      unreadPending = null;
    });
  return unreadPending;
}

/**
 * 導覽列用的未讀數（共享）。
 *
 * 同時是「這條連線該不該開著」的驅動點：導覽列在每一頁都存在，因此由它決定
 * 連線的生命週期，不必讓每個頁面自己記得開。
 *
 * loggedIn 為 false 時完全不發請求並回 0：/unread-count 與串流都掛在 requireLogin
 * 後面，對未登入者只會換到一次 303。鈴鐺與紅點只是「有沒有新東西」的提示，
 * 不值得為它把整頁導去登入 —— 因此在呼叫端就擋掉，而不是在 catch 裡補救。
 */
export function useUnreadNotifications(loggedIn: boolean): number {
  const [count, setCount] = useState(0);

  useEffect(() => {
    if (!loggedIn) {
      ensureNotificationStream(false);
      setCount(0);
      return undefined;
    }
    ensureNotificationStream(true);
    // 別的元件已經拿過就沿用同一份快照，紅點不必等第二個請求。
    if (unreadSnapshot !== null) setCount(unreadSnapshot);
    void ensureUnreadCount();

    const onCount = (next: number) => setCount(next);
    unreadSubscribers.add(onCount);
    return () => {
      unreadSubscribers.delete(onCount);
    };
  }, [loggedIn]);

  // 登出（或登入狀態還沒揭曉）時不顯示任何數字，即使 state 還留著上一次的值。
  return loggedIn ? count : 0;
}

/* ==========================================================================
   WebSocket 連線
   ========================================================================== */

/** 後端的訊息（見 forum_notification_stream.go 的檔頭）。type 是唯一的判別欄位。 */
interface NotificationStreamMessage {
  type?: string;
  unreadCount?: number;
  item?: ForumNotificationItem;
}

/** 後端 close frame 的應用層 code：被同一使用者的較新連線取代（見檔頭的重連策略）。 */
const STREAM_CLOSE_REPLACED = 4000;

/** 推播訂閱者（通知頁的清單）。只傳 item：未讀數在下面已經先寫進共享快照。 */
const pushSubscribers = new Set<(item: ForumNotificationItem) => void>();

/** 「重新連上了，請重新同步」的訂閱者（通知頁的清單）。 */
const resyncSubscribers = new Set<() => void>();

/** 目前的連線；null 代表沒有連線（未登入、尚未建立、或剛斷線）。 */
let socket: WebSocket | null = null;

/** 是否應該維持連線（登入狀態）。false 時斷線不再重連。 */
let streamDesired = false;

/** 連續失敗次數，決定下一次退避的長度；收到任何伺服器訊息就歸零。 */
let streamAttempt = 0;

/** 這一頁收過幾次 ready。第二次之後代表「這是重新連線」，見檔頭。 */
let streamReadyCount = 0;

/** 待執行的重連計時器（同時只允許一個）。 */
let reconnectTimer: number | null = null;

/**
 * 依登入狀態維持連線。重複呼叫是安全的（導覽列有兩個元件會呼叫它）。
 *
 * 刻意不做「登出時立刻關閉」以外的任何清理：連線是模組層的，一頁一份，
 * 頁面關閉時由瀏覽器負責。
 */
export function ensureNotificationStream(loggedIn: boolean): void {
  streamDesired = loggedIn;
  if (!loggedIn) {
    closeNotificationStream();
    return;
  }
  if (socket !== null) return;
  openNotificationStream();
}

/**
 * 通知頁訂閱「有人推了一則新通知」。回傳取消訂閱的函式。
 *
 * 只傳 item：未讀數是共享快照的事，處理常式在通知訂閱者之前就寫好了（導覽列的
 * 紅點據此更新）；清單只負責把那一列插到最前面。
 */
export function subscribeNotificationPushes(subscriber: (item: ForumNotificationItem) => void): () => void {
  pushSubscribers.add(subscriber);
  return () => {
    pushSubscribers.delete(subscriber);
  };
}

/**
 * 通知頁訂閱「重新連線了，該重新同步清單」。
 *
 * 只有第一次連線不通知（那時頁面自己的載入才剛發生），見檔頭。
 */
export function subscribeNotificationResync(subscriber: () => void): () => void {
  resyncSubscribers.add(subscriber);
  return () => {
    resyncSubscribers.delete(subscriber);
  };
}

/** 串流的網址。同源、wss 跟隨頁面：混合內容（https 頁面連 ws）會被瀏覽器擋下。 */
function notificationStreamURL(): string {
  const scheme = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
  return `${scheme}${window.location.host}/api/forum/notifications/ws`;
}

function openNotificationStream(): void {
  if (!streamDesired || socket !== null) return;

  let connection: WebSocket;
  try {
    connection = new WebSocket(notificationStreamURL());
  } catch {
    // 建構子丟錯只會來自「這個環境沒有 WebSocket」或網址不合法，兩者都不是
    // 重試能解決的。紅點仍然由每次載入的 GET 維護。
    return;
  }
  socket = connection;

  connection.onmessage = (event) => handleStreamMessage(event.data);
  connection.onclose = (event) => {
    if (socket === connection) socket = null;
    // 被取代時不重連（見檔頭的重連策略）：重連只會把對方踢掉，然後換自己被踢。
    if (event.code === STREAM_CLOSE_REPLACED) {
      if (event.reason) console.warn(`[notifications] 即時通知已由另一個連線接手（${event.reason}）`);
      return;
    }
    // 伺服器的 close frame 會說明「為什麼關掉」（額度已滿、session 失效、讀得
    // 太慢）。那是難得一見的診斷資訊，值得留下一行。
    if (event.reason) console.warn(`[notifications] 即時通知連線結束（${event.code}: ${event.reason}）`);
    // 一般的斷線只記第一次：接下來會自動重連，把每一次失敗都印出來只會讓
    // console 變成一串重複訊息。
    else if (streamAttempt === 0) console.warn('[notifications] 即時通知連線中斷，將自動重連', event.code);
    scheduleStreamReconnect();
  };
  // 連線失敗一定會先經過 error 再 close，因此這裡不做事（見 onclose）。
  connection.onerror = () => undefined;
}

function handleStreamMessage(raw: unknown): void {
  // 收到任何訊息就代表「這條連線真的通了」，退避從下一次失敗重新開始數。
  streamAttempt = 0;
  if (typeof raw !== 'string') return;

  let message: NotificationStreamMessage;
  try {
    message = JSON.parse(raw) as NotificationStreamMessage;
  } catch {
    return;
  }
  if (!message || typeof message !== 'object') return;

  if (message.type === 'ready') {
    if (typeof message.unreadCount === 'number') publishUnreadCount(message.unreadCount);
    streamReadyCount += 1;
    // 第二次之後的 ready 都是「斷線後重新連上」：清單可能少了斷線期間的通知。
    if (streamReadyCount > 1) for (const subscriber of resyncSubscribers) subscriber();
    return;
  }

  if (message.type === 'notification') {
    const item = message.item;
    // id 是清單的唯一鍵（去重與就地更新都靠它），沒有它的訊息沒有用處。
    if (item && typeof item.id === 'number' && item.id > 0) {
      // 未讀數先寫進共享快照（導覽列立刻亮），再交給清單訂閱者插入那一列。
      if (typeof message.unreadCount === 'number') publishUnreadCount(message.unreadCount);
      for (const subscriber of pushSubscribers) subscriber(item);
    }
  }
}

function scheduleStreamReconnect(): void {
  if (!streamDesired || reconnectTimer !== null) return;
  // 抖動（0~500ms）讓同時斷線的分頁不會在下一個瞬間一起回來。
  const delay = Math.min(30_000, 1_000 * 2 ** streamAttempt) + Math.random() * 500;
  streamAttempt += 1;
  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = null;
    openNotificationStream();
  }, delay);
}

function closeNotificationStream(): void {
  if (reconnectTimer !== null) {
    window.clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  const connection = socket;
  socket = null;
  if (connection === null) return;
  // 先拆掉 handler 再關：不這樣做的話，那一則 close 事件會走到 onclose，
  // 而那裡又會排一次重連（streamDesired 已經是 false，不會真的排，但讓一條
  // 明知要關的連線走完那條路徑只是多一個沒必要的分支）。
  connection.onmessage = null;
  connection.onclose = null;
  connection.onerror = null;
  connection.close(1000, 'client closing');
}
