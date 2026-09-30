/*
 * 圖片 token 釋放（src/forum/useMediaTokenRelease.ts）
 *
 * 貼文的圖片網址帶 ?token=，token 存在 Redis、有 TTL。使用者看過一頁貼文後
 * 直接關掉分頁，那把 token 會一直留到 TTL 到期 —— 那是純粹的浪費：圖片已經
 * 沒有人會再請求了。因此在 pagehide 時把這一頁用過的 token 交給後端刪掉。
 *
 * 為什麼是 hook 而不是各頁自己寫：這段邏輯有三份完全相同的需求（首頁動態 +
 * 搜尋結果、追蹤頁的貼文、他人個人頁的貼文），而它的失效是靜默的 —— 少了
 * 釋放不會有任何錯誤訊息，只有 Redis 裡的 key 一直累積。抄三份的代價正是
 * 「第四頁忘了抄」或「某一頁抄漏了其中一個來源」。集中在一處之後，新增一個
 * 顯示貼文的頁面只要呼叫這支 hook 就自動正確。
 *
 * 三個設計決定：
 *
 *  1. 走 navigator.sendBeacon 而不是 fetch：pagehide 階段的 fetch 會被瀏覽器
 *     直接取消，而 sendBeacon 被保證會把請求送出去（它就是為了「離開頁面時仍
 *     要送出最後一個請求」而生的）。
 *  2. 讀的是 state 裡的 imageUrl 而不是去 DOM 撈 <img class="post-image"> 的
 *     src：結果完全相同，但不必在 pagehide 裡查 DOM（那個時機 DOM 可能已經
 *     被部分釋放）。
 *  3. 只釋放一次（releasedRef）：pagehide 與 unload 都可能觸發，而第二次呼叫
 *     會是一份重複的請求。另外若一頁完全沒有圖片就不送任何請求 —— 那是
 *     最常見的情況（純文字論壇），不該為它產生一次 POST。
 *
 * 傳入的貼文清單可以有任意多個來源（首頁是動態 + 搜尋結果），因為一頁的
 * 圖片集合是它們的聯集。
 */

import { useEffect, useRef } from 'react';

import type { ForumPost } from '../types';

/** 只回傳本次渲染要納入的貼文，刻意不持有 state、不做任何事。 */
export function useMediaTokenRelease(...sources: ForumPost[][]): void {
  const releasedRef = useRef(false);

  useEffect(() => {
    const release = () => {
      if (releasedRef.current) return;
      const tokens = new Set<string>();
      for (const posts of sources) {
        for (const post of posts) {
          if (!post.imageUrl) continue;
          try {
            const token = new URL(post.imageUrl, window.location.href).searchParams.get('token');
            if (token) tokens.add(token);
          } catch {
            // 不是合法網址就當作這張圖沒有 token。
          }
        }
      }
      if (tokens.size === 0) return;
      releasedRef.current = true;
      navigator.sendBeacon(
        '/api/forum/image-tokens/release',
        new Blob([JSON.stringify({ tokens: [...tokens] })], { type: 'application/json' }),
      );
    };
    window.addEventListener('pagehide', release);
    return () => window.removeEventListener('pagehide', release);
    // sources 是每次 render 新建的陣列，因此刻意只依賴「這幾個清單的長度與
    // 內容」而非陣列身分 —— 用 spread 進 useEffect 的依賴會讓 effect 每次
    // render 都重掛，而重掛並不會讓已經送出的釋放請求重送（releasedRef 擋住），
    // 只是白做一次 removeEventListener/addEventListener。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, sources);
}
