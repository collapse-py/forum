/*
 * 站名（src/site.ts）
 *
 * 站名是部署者的設定（backend/config/config.conf 的 FORUM_NAME 與
 * FORUM_SHORT_NAME），不是寫死在程式裡的字串。九個頁面殼的 <head> 帶著
 * {{FORUM_NAME}} / {{FORUM_SHORT_NAME}} 佔位符，後端在送出時換成設定值
 * （backend/forum/httpapi/site.go），並把結果放進兩個 <meta>；這個模組就是
 * 讀它們的地方。
 *
 * 為什麼是模組層的 const 而不是 hook 或 context
 *   第一次 render 就必須是正確的站名。頁面標題、導覽列標誌與後臺 rail 都會
 *   讀它們，而 React 的第一個 render 早於任何 effect；用 hook 或 context 會
 *   讓這三處在掛載前先畫出預設值（值相同時使用者看不出來，站名改過時就是
 *   一幀閃爍）。meta 標記在模組求值時就已經在 DOM 裡 —— 頁面碼是以
 *   type="module" 載入的，模組求值一定發生在 HTML 解析完成之後。
 *
 * 為什麼走 <meta> 而不是內嵌 JSON 或全域變數
 *   CSP 的 script-src 只有 'self' 沒有 'unsafe-inline'，內聯 <script> 與
 *   衍生寫法都不該出現在樣板裡；<meta> 不受 script 規則約束，也不需要多一
 *   次請求就能在第一次繪製前拿到值。走 API 的話，站名會晚一個 RTT 才出現，
 *   而且沒有 JavaScript 的爬蟲連 <title> 都拿不到。
 *
 * 站名不隨語言切換
 *   它是設定檔裡的一個值，不是譯文：所有語言共用同一個名字，語言只改寫
 *   它周圍的介面文字（"登入｜{site}" 的「登入」）。因此這裡不需要任何
 *   重算，也沒有「切換語言後站名變成英文」的問題。
 */

/** 頁面殼 <meta> 的 name，值必須與後端的樣板一致。 */
const META_NAME = 'forum-name';
const META_SHORT_NAME = 'forum-short-name';

/**
 * 樣板沒有被後端取代時的站名。
 *
 * 只有兩種情況會走到這裡：直接用瀏覽器打開原始碼目錄（沒有後端送出），
 * 或設定檔的 FORUM_NAME 是空字串。後端已有 "FORUM 論壇" 的預設值，因此這
 * 兩個常數只是「別讓標題變空白」的保險，不是第二個真相來源 —— 真要換站名
 * 請改設定檔，改這裡只會換掉佔位符失效時看到的樣子。
 */
const FALLBACK_NAME = 'FORUM 論壇';

/**
 * 讀出一個 <meta> 的 content，找不到或空白時回空字串。
 *
 * querySelector 而非 getElementsByName：getElementsByName 會撈到表單控件的
 * name 屬性，這裡要的是明確的 meta 標記。content 一律 trim，設定檔的行尾
 * 空白不該變成畫面上看不見、卻讓版面偏移的字元。
 */
function readMeta(name: string): string {
  if (typeof document === 'undefined') return '';
  const meta = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`);
  return meta?.content.trim() ?? '';
}

/** 完整站名，例如 "FORUM 論壇"。出現在頁面標題、登入頁說明與 logo 的 aria 標籤。 */
export const SITE_NAME = readMeta(META_NAME) || FALLBACK_NAME;

/**
 * 標誌用的短名，例如 "FORUM"。
 *
 * 與完整站名分開是因為版面：短名要塞進導覽列的膠囊按鈕與後臺 rail 的方塊，
 * 長度必須可預期。設定檔沒給（或給了空字串）時沿用完整站名。
 */
export const SITE_SHORT_NAME = readMeta(META_SHORT_NAME) || SITE_NAME;

/**
 * 標誌方塊裡的兩字縮寫（例如 "HP"）。
 *
 * 用 Array.from 取的是字元碼點而不是 UTF-16 單位，因此不會把 surrogate pair
 * 從中間切開（emoji 或部分語系會出現那種情況，切開就是亂碼）。取兩字而不是
 * 一字：單字在方塊裡太小，兩個以上又會撐破版面 —— 這是 CSS 與內容的分工，
 * 這裡只負責決定「哪兩個字」。
 */
export const SITE_MARK = Array.from(SITE_SHORT_NAME).slice(0, 2).join('');
