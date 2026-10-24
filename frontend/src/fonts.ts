/*
 * 網路字型的非阻斷載入（src/fonts.ts）
 *
 * 為什麼要由 JS 注入而不是放在 <head> 的 <link rel="stylesheet">：
 * 瀏覽器對 <head> 裡的 stylesheet 是「算繪阻斷」的 —— 在它下載並解析完成之前
 * 一個像素都不會畫。Google Fonts 的這份 CSS 有 136 KiB（Inter 六個字重 +
 * Noto Sans TC 的 unicode-range 切片），Lighthouse 在慢速 4G 上實測它讓手機版
 * FCP 卡了約 2,290 毫秒，是整份報告裡最大的單一項目。
 *
 * 改成在首次繪製之後才注入，瀏覽器會立刻以系統字體堆疊（style.css 的 --font：
 * Inter → system-ui → -apple-system → Segoe UI → Noto Sans TC）畫出整頁，字型
 * CSS 與 woff2 檔在背景下載，到齊後以 font-display: swap 換上 —— 也就是說
 * swap 本來就發生在首次繪製之後（原本的 <link> 只阻擋 CSS，字型檔仍是非同步
 * 的），因此這個改動對 CLS 沒有額外代價，卻把 2.3 秒的白屏從關鍵路徑上拿掉。
 *
 * 為什麼不用常見的 <link rel="preload" as="style" onload="..."> 手法：
 * 那需要一個內聯 onload 處理器，而本站 CSP 的 script-src 只有 'self' 與
 * Cloudflare Insights，沒有 'unsafe-inline' 也不允許 'unsafe-hashes'
 * （見 backend/forum/httpapi/securityheaders.go）。同源的 JS 模組是唯一既合規
 * 又不阻斷算繪的做法 —— 本站自己的 bundle 已在 script-src 的 'self' 裡。
 *
 * Material Symbols 字型已完全移除：論壇只用到六個圖示，改用內聯 SVG
 * （src/icons.tsx 的 ForumGlyph）之後不再需要它，因此這裡的 URL 不含
 * family=Material+Symbols+Outlined，那個 493 KiB 的第三方字檔不再有任何請求。
 *
 * Noto Sans TC 也已移除（中文改用系統字型）：Lighthouse 的「減少無用的 CSS」
 * 實測這份樣式表有 135.5 KiB，其中 91%（432/474 條 @font-face）是 Noto Sans TC
 * 的 unicode-range 切片 —— Google 把中文字型切成每字重 108 片，而任何單一頁面
 * 只會用到其中幾片，因此那些規則對 Lighthouse 來說恆為「無用」，**只要還用
 * Google Fonts 載入中文，這項稽核就不可能通過**（裁字重只會讓數字變小，消不掉）。
 *
 * 改用系統中文字型的取捨：macOS 拿到 PingFang TC、Windows 拿到微軟正黑體、
 * Linux 拿到 Noto Sans CJK，都是高品質原生字型，且不佔任何下載。中文樣式因而
 * 從 Noto Sans TC Web 字型換成 OS 原生字型（視覺會有一點差異）。這與本站其他
 * 非拉丁語系的現況一致 —— 日文假名、韓文、泰文、阿拉伯文、天城文本來就走
 * 系統字型（見 style.css 的 --font 堆疊），因為它們從未被列進這個 URL。
 *
 * --font 堆疊刻意保留 "Noto Sans TC" 這個名稱：系統若本身有裝，仍然會命中；
 * 沒裝就順序落到 PingFang TC / 微軟正黑體。這裡移除的是 Web 字型的下載，不是
 * 字體名稱的候選。
 */

const FONT_CSS_URL = 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap';

let requested = false;

/**
 * 在首次繪製之後注入字型樣式表。
 *
 * 重複呼叫只會注入一次（requested 旗標）：這個函式會被掛在 ForumShell 的
 * effect 上，而那一層在單頁應用的切頁時不會重掛，但未來若有別的呼叫端也不該
 * 產生第二個 <link>。
 *
 * 刻意不用模組層的立即執行：模組求值發生在 HTML 解析完成之後、React 渲染之前，
 * 那時注入仍會延後首次繪製。真正的延後由呼叫端用 requestIdleCallback /
 * setTimeout 保證 —— 兩者都排在瀏覽器完成首次算繪之後。
 */
export function loadWebFonts(): void {
  if (requested || typeof document === 'undefined') return;
  requested = true;
  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = FONT_CSS_URL;
  document.head.appendChild(link);
}

/**
 * 排定一次「畫完再載字型」。
 *
 * requestIdleCallback 優先（瀏覽器空閒時才做，不會與首屏的算繪/互動搶時間）；
 * Safari 與部分瀏覽器沒有它，因此退回 setTimeout 0 —— 那同樣排在目前這個
 * task 與首次算繪之後，只是沒有「等到空閒」那麼禮貌。
 */
export function scheduleWebFonts(): void {
  if (typeof window === 'undefined') return;
  const ric = window.requestIdleCallback;
  if (typeof ric === 'function') ric(() => loadWebFonts());
  else window.setTimeout(loadWebFonts, 1);
}
