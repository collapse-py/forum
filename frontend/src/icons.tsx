/*
 * 圖示（src/icons.tsx）
 *
 * 路徑資料是純常數（寫死在這個檔案裡的 SVG path），因此直接以 JSX 元素表達，
 * 不需要 dangerouslySetInnerHTML —— 也因此這裡不可能意外把後端回來的字串
 * 當成標記解析。這是 React 版相對舊 admin-core.icon() 的實際安全收益：
 * 舊版回傳 HTML 字串給呼叫端丟進 innerHTML，那個字串必須永遠是常數。
 *
 * 兩條硬性約束（沿用 style.css 的說明）：
 *   1. style-src 'self' 沒有 'unsafe-inline'：不得產生 style="..." 屬性。
 *      這支檔案沒有任何 inline style。
 *   2. .rail__icon wrapper 是 style.css 的版面契約，寬高與 stroke 都從
 *      那個 class 取得，因此 Icon 必須保留它，而不是直接輸出 <svg>。
 *
 * SearchGlyph 為什麼不共用 Icon：公開頁（/forum）不載入後臺樣式，.rail__icon
 * 在那裡沒有任何定義，包上去只會得到一個沒有尺寸的 <svg>。因此搜尋放大鏡以
 * 「單一來源的路徑 + 兩種外殼」的方式共用：Icon 給後臺，SearchGlyph 給公開頁
 * 與表單內的輸入框（外殼由呼叫處的 class 決定，兩邊各有一份同名不同義的
 * .search-field__icon 規則）。
 */

import type { ReactNode } from 'react';

export type IconName =
  | 'users'
  | 'posts'
  | 'flag'
  | 'home'
  | 'logout'
  | 'menu'
  | 'close'
  | 'refresh'
  | 'search'
  | 'inbox'
  | 'monitor'
  | 'log'
  | 'chart'
  | 'export'
  | 'sessions'
  | 'shield'
  | 'megaphone';

/*
 * 搜尋放大鏡。形狀來自設計稿指定的 SVG（單一 path，圓 + 握把一起），
 * 取代先前自繪的 circle + line 版本：圓心與半徑一致、握把角度與圓周相接，
 * 在 20~24px 的實際顯示尺寸下不會看成「兩個分開的形狀」。
 *
 * stroke 刻意用 currentColor 而非原始檔的 #000000：這個檔案所有圖示都是
 * currentColor（由所在位置的 color 決定），寫死色碼會讓它在深色底或
 * 灰階按鈕上突然變成一個黑點。粗細則照設計稿的 2，比本檔其他圖示的 1.7
 * 略粗，讓它在輸入框裡有存在感而不致與鄰近的圓角邊框打架。
 */
const SEARCH_PATH = (
  <path d="M15.7955 15.8111L21 21M18 10.5C18 14.6421 14.6421 18 10.5 18C6.35786 18 3 14.6421 3 10.5C3 6.35786 6.35786 3 10.5 3C14.6421 3 18 6.35786 18 10.5Z" />
);

const SEARCH_STROKE_WIDTH = 2;


const ICON_PATHS: Record<IconName, ReactNode> = {
  users: (
    <>
      <path d="M15.5 19.5v-1.4a3.4 3.4 0 0 0-3.4-3.4H7.4A3.4 3.4 0 0 0 4 18.1v1.4" />
      <circle cx="9.75" cy="7.75" r="3.25" />
      <path d="M20 19.5v-1.4a3.4 3.4 0 0 0-2.6-3.3" />
      <path d="M15.4 4.6a3.25 3.25 0 0 1 0 6.3" />
    </>
  ),
  posts: <path d="M20 13.4a2.4 2.4 0 0 1-2.4 2.4H8.4L4.4 19.4V6.6A2.4 2.4 0 0 1 6.8 4.2h10.8A2.4 2.4 0 0 1 20 6.6z" />,
  flag: (
    <>
      <path d="M5 21V3.8" />
      <path d="M5 4.6h11.4l-1.7 3.4 1.7 3.4H5z" />
    </>
  ),
  home: (
    <>
      <path d="M4 10.6 12 4l8 6.6" />
      <path d="M6 9.6V20h12V9.6" />
    </>
  ),
  logout: (
    <>
      <path d="M14.5 8.2V5.6H5.4v12.8h9.1v-2.6" />
      <path d="M11 12h9" />
      <path d="M17 9l3 3-3 3" />
    </>
  ),
  menu: <path d="M4 7h16M4 12h16M4 17h16" />,
  close: <path d="M6 6l12 12M18 6L6 18" />,
  refresh: (
    <>
      <path d="M20 11.5a8 8 0 1 0-.9 4.6" />
      <path d="M20 4.5V11h-6.2" />
    </>
  ),
  search: SEARCH_PATH,
  inbox: (
    <>
      <path d="M3.6 13.4h4.2l1.4 2.6h5.6l1.4-2.6h4.2" />
      <path d="M5.6 5.2h12.8l2.2 8.2v4.4a2 2 0 0 1-2 2H5.4a2 2 0 0 1-2-2v-4.4z" />
    </>
  ),
  /*
   * 監控：螢幕外框 + 一條上升的折線。
   *
   * 刻意畫成「螢幕上的折線」而不是「心跳/ECG」或「齒輪」。三個候選裡，心跳
   * 會被讀成醫療用途，齒輪在這套圖示裡已經隱含「設定」；而監控頁展示的就是
   * 流量隨時間的曲線，螢幕加折線是最接近字面的一個。
   *
   * 折線的末端刻意伸出螢幕右緣：那個「超出邊界」的點就是流量尖峰的意思，
   * 與 .monitor-bar 的語意一致。圓點用 cx/cy 寫在最後一個資料點上，讓折線
   * 不會看起來是斷在邊界的。
   */
  monitor: (
    <>
      <rect x="3" y="4.2" width="18" height="12.6" rx="1.8" />
      <path d="M6.6 13.4l2.6-3.1 2.2 2 2.5-3.6 3.5 3.3" />
      <circle cx="17.4" cy="12" r="0.95" fill="currentColor" stroke="none" />
      <path d="M9 20.2h6" />
    </>
  ),
  /*
   * 稽核紀錄：捲軸 + 幾條橫線（每一列一次操作）。
   *
   * 刻意不畫「放大鏡」或「眼睛」—— 那兩個符號在這套圖示裡一個屬於搜尋、
   * 一個屬於「公開可見性」，都會把「查帳」誤導成「看東西」。捲軸加橫線是
   * 「一份按時間排列的紀錄」最直接的形狀。
   *
   * 橫線的長度刻意不一：稽核紀錄的每一列長度本來就不一（有的帶長文字、
   * 有的只有狀態），把它畫成等長會讓整個圖示看起來像一份試算表。
   */
  log: (
    <>
      <path d="M6.4 3.4h11.2a2 2 0 0 1 2 2v13.2a2 2 0 0 1-2 2H6.4a2 2 0 0 1-2-2V5.4a2 2 0 0 1 2-2z" />
      <path d="M8.6 8h6.8M8.6 11.4h6.8M8.6 14.8h4.2" />
    </>
  ),
  /*
   * 內容趨勢：上升的面板（長條圖 + 上升折線 + 右上角箭頭）。
   *
   * 刻意不畫「圓餅圖」或「齒輪」：這一頁是時間序列，最通用的符號是帶趨勢
   * 的圖形；而「pie」在介面裡還有「圓餅圖」的字面歧義，用在 rail 上會被
   * 讀成分類而不是趨勢。
   *
   * 箭頭的末端刻意伸出面板外緣：那個「超出邊界」的點就是成長的意思，
   * 與 .stats-chart 的語意一致。
   */
  chart: (
    <>
      <path d="M3.4 20.2h17.2" />
      <path d="M5.4 20.2V13h3.2v7.2M11.4 20.2V8.6h3.2v11.6M17.4 20.2V4.4h3.2v15.8" />
      <path d="M5.6 10.4l4.4-3.6 3.6 2.6 4.8-4" />
    </>
  ),
  /*
   * 匯出：向下進入托盤的箭頭 + 一條水平分隔線。
   *
   * 刻意不畫「上傳箭頭」：那一個在介面裡的語意是「把檔案送進系統」，
   * 而這裡的動作是「把資料拿出來」—— 方向相反，而箭頭方向在這套圖示裡
   * 是唯一有語意的部分。托盤的底部那條線讓它讀成「檔案落在這裡」而不是
   * 「東西飛出去」。
   */
  export: (
    <>
      <path d="M12 3.4v10.2" />
      <path d="M8.2 10.2L12 14l3.8-3.8" />
      <path d="M4.4 15.4v3.4a2 2 0 0 0 2 2h11.2a2 2 0 0 0 2-2v-3.4" />
    </>
  ),
  /*
   * 登入 session：圓頭的「門框」+ 從框內伸出的小圓點（那顆點就是憑證）。
   *
   * 刻意不畫「人」或「頭像」：那一組符號在這套圖示裡代表「帳號」（users），
   * 而這個頁面談的是「裝置上的登入狀態」而不是「這個人」。用「門 + 憑證」
   * 讓 rail 上的兩個帳號相關項目（用戶管理、登入與 Session）在形狀上可區分。
   *
   * 框的右上方刻意開一個缺口：那是「可以從那裡進來」的意思，同時避免整個
   * 圖示變成一個填滿的方塊。
   */
  sessions: (
    <>
      <path d="M15.4 3.6H6.6a2 2 0 0 0-2 2v12.8a2 2 0 0 0 2 2h12.8a2 2 0 0 0 2-2V7.4" />
      <path d="M18.6 3.4v5.2M16 6h5.2" />
      <circle cx="11.4" cy="12.4" r="2.2" />
    </>
  ),
  /*
   * IP 封鎖：盾牌 + 內側的一條勾。
   *
   * 刻意不畫「禁止符號」（圓圈加斜線）：那一組符號在介面裡是「關閉／停用」
   * 的通用符號，而這一頁做的事是**啟用**一項防護 —— 用禁止符號會讓管理員
   * 以為它是「把封鎖功能關掉」的開關，方向完全相反。
   *
   * 勾號而非斜線：勾號是「這一項已經生效」的形狀，而防護的啟用狀態正是這一頁
   * 要表達的事。
   */
  shield: (
    <>
      <path d="M12 3.2l7 2.8v5.4c0 4.4-2.9 7.8-7 9.4-4.1-1.6-7-5-7-9.4V6z" />
      <path d="M8.6 12.1l2.5 2.5 4.3-4.8" />
    </>
  ),
  /*
   * 站內公告：喇叭（發聲）而不是信封或報紙。
   *
   * 這幾個形狀在介面裡很容易撞：信封是「登入」的圖示，而報紙會被讀成
   * 「新聞／RSS」。公告在這個專案裡的語意是「站方現在要跟所有訪客說一句
   * 話」，喇叭正是那個動作的形狀。
   *
   * 喇叭的開口刻意朝右上（而不是正上方）：右上是一個「往外說」的方向，
   * 而正上方會讓它看起來像一個向上的箭頭 —— 而 rail 裡已經有 chart（上升
   * 折線）與 export（向下箭頭）兩個含箭頭的符號了。
   */
  megaphone: (
    <>
      <path d="M4 10.4v3.2a1.6 1.6 0 0 0 1.6 1.6h1.9l7.3 4.1V6.7L7.5 10.8H5.6A1.6 1.6 0 0 0 4 12.4z" />
      <path d="M17.7 9.4a3.4 3.4 0 0 1 0 5.2" />
      <path d="M7.5 15.2v3.2a1 1 0 0 0 1 1h1.2" />
    </>
  ),
};

export interface IconProps {
  name: IconName;
  /** 附加在 .rail__icon 之外的 class（例如狀態頁的 state__icon 內縮放）。 */
  className?: string | undefined;
}

// 少數圖示的粗細要跟設計稿走（例如搜尋放大鏡是 2，比本檔預設的 1.7 粗）。
// 沒列在這裡的一律用預設值，維持後臺圖示整體的視覺一致。
const ICON_STROKE_WIDTHS: Partial<Record<IconName, number>> = {
  search: SEARCH_STROKE_WIDTH,
};

export function Icon({ name, className }: IconProps) {
  return (
    <span className={className ? `rail__icon ${className}` : 'rail__icon'}>
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={ICON_STROKE_WIDTHS[name] ?? 1.7}
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
        focusable="false"
      >
        {ICON_PATHS[name]}
      </svg>
    </span>
  );
}

export interface SearchGlyphProps {
  /** 尺寸的 class（不含 .search-glyph 本身）；由呼叫處的 CSS 決定大小與顏色。 */
  className?: string | undefined;
}

/**
 * 搜尋放大鏡（不帶 .rail__icon 外殼）。
 *
 * 用在公開頁與表單輸入框內，因此不包任何 admin 專屬的 class：
 * 尺寸與位置交給呼叫處的 `.search-glyph`（或 `search-field__icon`）規則，
 * 這裡只負責把那個放大鏡畫出來。aria-hidden 與 focusable 與 Icon 一致：
 * 圖示是裝飾，語意由旁邊的 label 或按鈕文字承擔。
 */
export function SearchGlyph({ className }: SearchGlyphProps) {
  return (
    <svg
      className={className ? `search-glyph ${className}` : 'search-glyph'}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={SEARCH_STROKE_WIDTH}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {SEARCH_PATH}
    </svg>
  );
}
