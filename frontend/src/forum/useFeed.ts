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
import type { ItemsResponse, LikeResponse, BookmarkResponse, UploadResponse, ForumPost } from '../types';

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

/**
 * 這一頁要從第一頁重新載入時，把捲動位置放回頂端，並關掉瀏覽器的「重新整理後
 * 還原捲動位置」。
 *
 * 為什麼必須關掉還原（而不是只在載入之後回頂端）：重新整理後瀏覽器會把上一次
 * 讀到的捲動位置還原回去，而那個還原是在內容一頁一頁變長的過程中反覆套用的 ——
 * 內容還沒長到那個位置時它被夾到列表底部，於是列表底部的 sentinel 一直待在
 * 觀察帶裡，IntersectionObserver 每被重新建立一次就載入一頁，一路把整份動態
 * 拉完才停。那正是「往下捲之後重新整理會一直請求到最底」的成因：使用者看到
 * 的是頁面停在一個沒有內容的位置，而背後送出的是幾十次分頁請求。
 *
 * 為什麼選擇「回到頂端」而不是「把那些頁補齊」：這一頁是倒序動態，要還原一個
 * 深捲動位置就必須先把前面所有頁載回來，而那正是我們要省掉的成本。代價是重新整理
 * 後會回到頂端 —— 對一個「一直往下讀」的動態來說，這比十幾次請求更便宜。
 */
function resetScrollForFreshFeed(): void {
  // 舊版瀏覽器（舊 Safari）沒有這個屬性，因此只能先確認它存在再寫。
  if (typeof history.scrollRestoration === 'string' && history.scrollRestoration !== 'manual') {
    history.scrollRestoration = 'manual';
  }
  // 還原可能已經發生在 React 掛載之前（那時它被夾到舊的列表底部），這一項
  // 把那份夾出來的位置放回頂端。
  window.scrollTo(0, 0);
}

export function useFeed({ pageSize = 25, endpoint = DEFAULT_ENDPOINT, enabled = true }: FeedOptions = {}): FeedController {
  const [state, setState] = useState<FeedState>(initialState);

  const offsetRef = useRef(0);
  const hasMoreRef = useRef(true);
  const loadingRef = useRef(false);
  const abortRef = useRef<AbortController | null>(null);
  /*
   * requestIdRef 認證「哪一個請求才是最新的那一個」。
   *
   * 為什麼 abort 不夠：中止一個請求不會讓它的 await 立刻返回 —— 它還會在
   * 稍後走完 catch 與 finally。因此「我這一份結果還算不算數」不能只看
   * aborted 旗標，要看自己的序號是不是仍然是最新的一個。少了這個判斷，
   * 被中止的舊請求會在**新請求已經開始之後**才回來，而它會把自己的
   * offset 累加回去、把列表覆蓋掉。
   */
  const requestIdRef = useRef(0);
  /*
   * itemsRef 是「目前列表」的鏡像。
   *
   * 為什麼需要它：remove() 必須在 **updater 之外**判斷「這一篇在不在列表
   * 裡」，才能把 offset 的調整也放在 updater 之外 —— updater 內的副作用不
   * 具冪等性（React 可能呼叫 updater 一次以上：重繪重試、concurrent render
   * 被丟棄），扣兩次會讓分頁位置漂移 1~2，下一頁就會跳過或重複一篇。
   *
   * 它是 ref 而不是 state：remove() 由事件處理器呼叫（不在 render 期間），
   * 因此讀到的是最近一次 render 的結果 —— 也就是當下的列表。
   */
  const itemsRef = useRef<ForumPost[]>([]);
  itemsRef.current = state.items;

  const load = useCallback(
    (reset: boolean) => {
      if (loadingRef.current) {
        /*
         * reset 一定要執行：它的語意是「回到第一頁重新載入」，而被丟棄時
         * 呼叫端會看到「按了沒反應」—— 最常見的是「取消追蹤之後重整清單」
         * 與「首頁鈕」。那兩個呼叫點都傳 true，而它們被丟棄的症狀是
         * 已經取消追蹤的人的貼文還留在畫面上。
         *
         * 因此中止進行中的那一個、從頭開始。非 reset 的載入則維持
         * 「已在載入中就忽略」—— 那個保護是為了避免重複拉同一頁。
         */
        if (!reset) return;
        abortRef.current?.abort();
      }
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
      const requestId = ++requestIdRef.current;
      // 送出請求時的當下位置：必須在這裡讀，不能等到回應時才讀（那時列表
      // 可能已經被更晚的 reset 清空）。
      const offset = offsetRef.current;

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
        `${endpoint}${separator}limit=${pageSize}&offset=${offset}`,
        { signal: controller.signal, fallback: tr(msg('error.fallbackLoad')) },
      );
          const items = Array.isArray(data?.items) ? data.items : [];
          const hasMore = !!data.hasMore;
          /*
           * 自己已經不是最新那一個請求了（被中止、或被更晚的 reset 取代）：
           * 這一頁的結果與它的 offset 累加都不該生效 —— 否則舊的一頁會蓋掉
           * 新的一頁，並讓分頁位置往前跳。
           */
          if (requestId !== requestIdRef.current) return;
          offsetRef.current = reset ? items.length : offsetRef.current + items.length;
          hasMoreRef.current = hasMore;
          setState((current) => ({
            phase: 'ready',
            items: reset ? items : [...current.items, ...items],
            status: hasMore ? msg('feed.scrollMore') : msg('feed.endOfFeed'),
            message: msg('feed.postsFailed'),
          }));
        } catch (error) {
          if (controller.signal.aborted) return;
          if (requestId !== requestIdRef.current) return;
          const message = msg('feed.postsFailed');
          setState((current) =>
            reset
              ? { phase: 'failed', items: [], status: message, message }
              : { ...current, status: msg('feed.postsFailedShort') },
          );
          void errorMessage(error, tr(message));
        } finally {
          /*
           * 只讓「自己這一個請求」清掉載入中的標記。被中止的舊請求的 finally
           * 會在新請求已經開始之後才執行 —— 那時它若無條件清掉標記，
           * 第三個請求就會被當成「不在載入中」而疊上來。
           */
          if (requestId === requestIdRef.current) {
            loadingRef.current = false;
          }
        }
      })();
    },
    [endpoint, pageSize],
  );

  /*
   * 依賴 load（它隨 endpoint / pageSize 改變身分）與 enabled。
   * enabled 從 false 翻成 true 時這個 effect 會重跑並補上第一次載入。
   *
   * 載入之前先 resetScrollForFreshFeed()：重新整理後瀏覽器還原的捲動位置會讓
   * 列表底部的 sentinel 直接落在觀察帶裡，那會把「第一次載入」變成「載入到最底」
   * （理由見該函式的說明）。放在這個 effect 而不是 load() 裡，因為它屬於
   * 「這一頁剛出現」而不是「要取一頁」—— load(true) 也會被「首頁鈕」與
   * 「取消追蹤後重整」呼叫，而那兩種情況不該搶走使用者正在看的捲動位置。
   */
  useEffect(() => {
    if (!enabled) return undefined;
    resetScrollForFreshFeed();
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
    /*
     * 分頁位置的調整刻意放在 updater **之外**。
     *
     * updater 可能在一次 render 裡被呼叫多次（React 的重繪重試、
     * concurrent render 被丟棄），而 updater 內的副作用不具冪等性：
     * 扣兩次會讓分頁位置漂移 1~2，症狀是下一頁跳過或重複貼文。
     *
     * 「這篇還在列表裡嗎」的判斷也放在外面（itemsRef 的說明見上），而且
     * 這個判斷**不可省**：一個頁面可能同時有兩份列表（動態與搜尋結果），
     * 而刪掉「只在搜尋結果裡」的那一篇時，動態那份的 offset 不該被扣 ——
     * 扣了它，動態的下一頁就會跳過一篇。
     */
    if (!itemsRef.current.some((post) => post.id === postId)) return;
    offsetRef.current = Math.max(0, offsetRef.current - 1);
    setState((current) => ({
      ...current,
      items: current.items.filter((post) => post.id !== postId),
    }));
  }, []);

  return { state, load, patch, remove, items: state.items };
}

/* ==========================================================================
   按讚與收藏
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

/**
 * 切換收藏。
 *
 * 回傳伺服器確認的狀態而不是本地翻轉：連點兩下的合併結果只有後端知道，
 * 而呼叫端要拿它去 patch 列表 —— 猜錯會讓卡片上的圖示與資料庫不一致，
 * 直到下一次重新載入才修正。
 *
 * 沒有第二個回傳值（例如收藏數）：收藏是私人的，後端刻意不回公開的收藏數
 * （見 forum_bookmark_handlers.go 的說明）。
 */
export async function toggleBookmark(postId: number): Promise<boolean> {
  const data = await requestJSON<BookmarkResponse>(`/api/forum/posts/${postId}/bookmark`, {
    method: 'POST',
    fallback: tr(msg('error.fallbackBookmark')),
  });
  return !!data.bookmarked;
}

/* ==========================================================================
   圖片
   ========================================================================== */

/**
 * 上傳一張圖片，回傳檔案伺服器給的網址（含 media token）。
 *
 * 發文頁與貼文編輯共用：兩者的差別只在拿到網址之後做什麼（前者放進發佈
 * 請求，後者放進 PUT 的 imageUrl），而「怎麼上傳」完全相同 —— 表單欄位名、
 * 端點與錯誤文案都不該有兩份。
 *
 * 缺 url 時**拋錯**，不傳空字串。
 *
 * 這個三態契約裡 '' 的意思是「移除圖片」（見 updateForumPost 的 imageUrl
 * 契約）：把「上傳的回應沒有 url」當成「移除」，會讓呼叫端送出
 * `imageUrl: ""` 而把使用者既有的附圖或頭像刪掉 —— 而且沒有任何錯誤訊息。
 * 症狀是「上傳成功、貼文存得下、圖片不見了」。
 */
export async function uploadForumImage(file: File): Promise<string> {
  const body = new FormData();
  body.append('file', file);
  const uploaded = await requestJSON<UploadResponse>('/api/forum/images', {
    method: 'POST',
    form: body,
    fallback: tr(msg('error.fallbackUpload')),
  });
  const url = uploaded?.url;
  if (!url) {
    throw new Error(tr(msg('error.fallbackUpload')));
  }
  return url;
}

/**
 * 刪除自己上傳、且已不被任何貼文或頭像引用的圖片。
 *
 * 只在兩條路徑上呼叫：使用者主動丟掉一張圖（編輯時按「移除附圖」），以及
 * 上傳成功但貼文沒存成（見 updateForumPost 與 NewPostPage.handleSubmit）。
 *
 * 刻意不做「先問引用狀態再決定刪不刪」：那個檢查必然在刪除前過期，而後端
 * 端點本來就是原子地判斷（仍被引用時回 409，見 forum_media_handlers.go）。
 * 因此呼叫端一律把失敗當成 best-effort —— 它清不掉的最壞情況是留下一個
 * 孤兒檔，而不是讓使用者正在做的事失敗。
 */
export async function deleteForumImage(url: string): Promise<void> {
  await requestJSON('/api/forum/images/delete', {
    method: 'POST',
    json: { url },
    fallback: tr(msg('error.fallbackImageDelete')),
  });
}

/* ==========================================================================
   編輯貼文
   ========================================================================== */

/**
 * 編輯時附圖的變更意圖。
 *
 *   undefined —— 不動附圖（表單沒被碰過）
 *   null      —— 移除附圖
 *   File      —— 換成這張新圖
 *
 * 用 null 而不是空字串代表「移除」：undefined 與 null 在這裡必須是兩個
 * 不同的狀態（前者是「沒碰」、後者是「拿掉」），而空字串在型別上與「有沒有
 * 選檔案」無關，呼叫端很容易用 `||` 之類的寫法把兩者混在一起。
 */
export type PostImageChange = undefined | null | File;

/**
 * 存檔後要套用到列表那一筆貼文的變更。
 *
 * imageUrl 刻意允許 undefined 且**保留這個欄位**：useFeed 的 patch 是淺合併，
 * 欄位不存在代表「不動附圖」，而移除附圖要表達的正是「把它設成沒有」。
 * 多出來的 undefined 因此不是雜訊，是這個三態的第三個值 —— 這也是這個
 * 介面存在的理由（exactOptionalPropertyTypes 下必須明確寫成
 * `imageUrl?: string | undefined`）。
 */
export interface PostUpdatePatch {
  content: string;
  edited: true;
  imageUrl?: string | undefined;
  /**
   * 索引簽章讓它可以直接餵給 useFeed 的 patch()（參數型別是 Partial<ForumPost>，
   * 而 ForumPost 帶著同樣的索引簽章）。
   *
   * TypeScript 的規則：來源型別**沒有**索引簽章時，它不能指派給帶索引簽章的
   * 目標型別 —— 物件字面量可以（推導出的隱含索引簽章），但這個 patch 是介面，
   * 因此少了這一行會在六個呼叫端各噴一次「Index signature is missing」。
   * 六處都改寫成物件展開只會製造六份一樣的轉換。
   */
  [key: string]: unknown;
}

/**
 * 改掉自己貼文的本文與（可選的）附圖。
 *
 * 與 toggleLike 放在一起的理由相同：兩者都是「對某一篇貼文的寫入」，而五個
 * 渲染 PostCard 的頁面都需要它。寫成共用函式而不是讓每個頁面各寫一次
 * requestJSON，否則那五處的 fallback 文案與錯誤處理會各自漂移 —— 而它們
 * 漂移之後不會出現在任何 console 裡。
 *
 * 回傳的是「要 patch 進列表的變更」而不是只有修剪後的內容：後端從
 * 圖片網址到 edited 旗標都與請求一一對應，但呼叫端自己並不知道後端怎麼
 * 存那張圖（例如 media token 是不是同一個字串），因此由這裡把請求中真的
 * 送出去的值原樣帶回來，呼叫端只負責套用。
 *
 * 換圖採用「先上傳、再 PUT」：上傳成功而 PUT 失敗時，那張新圖不會被任何
 * 貼文引用 —— 不主動清掉就是一個孤兒檔，而一般使用者沒有任何手段清掉它
 * （刪圖端點要求「我上傳的、且未被引用」兩個條件都成立）。清理是
 * best-effort，且不掩蓋原本的錯誤：使用者要看到的是「存檔失敗」。
 */
export async function updateForumPost(
  postId: number,
  content: string,
  imageChange?: PostImageChange,
): Promise<PostUpdatePatch> {
  const next = content.trim();
  const json: { content: string; imageUrl?: string } = { content: next };
  const patch: PostUpdatePatch = { content: next, edited: true };
  let uploadedURL = '';
  if (imageChange === null) {
    json.imageUrl = '';
    patch.imageUrl = undefined;
  } else if (imageChange) {
    uploadedURL = await uploadForumImage(imageChange);
    json.imageUrl = uploadedURL;
    patch.imageUrl = uploadedURL;
  }
  try {
    await requestJSON(`/api/forum/posts/${postId}`, {
      method: 'PUT',
      json,
      fallback: tr(msg('posts.editFailed')),
    });
  } catch (error) {
    if (uploadedURL) void deleteForumImage(uploadedURL).catch(() => undefined);
    throw error;
  }
  return patch;
}
