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

export type IconName = 'users' | 'posts' | 'flag' | 'home' | 'logout' | 'menu' | 'close' | 'refresh' | 'search' | 'inbox';

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
