/*
 * 語言的執行期狀態（src/i18n/runtime.ts）
 *
 * 這個檔案刻意不含任何 React。理由是 core.ts 的 formatNumber / formatDateTime
 * 也需要知道「目前是哪一種語言」，而 core.ts 同樣刻意不含 React（它被非 React
 * 的模組共用）。把「目前語言」放在一個純模組裡，兩邊就不必互相依賴。
 *
 * 一個 document 只有一種語言，因此用模組層變數記錄「目前生效的語言」是安全的：
 * 它與 Provider 的 state 是同一個值，而 Provider 才是觸發重繪的那一端。
 * 這也是為什麼 t() 可以在自訂 hook 與純函式（useFeed、searchStatusText）裡
 * 直接呼叫 —— 它不需要讀到 React 的 context。
 */

import { DEFAULT_LOCALE, localeMeta, matchFirstLocale, type LocaleCode } from './locales';
import { zhTW, type MessageKey } from './messages';
import { ar } from './translations/ar';
import { de } from './translations/de';
import { en } from './translations/en';
import { es } from './translations/es';
import { fr } from './translations/fr';
import { hi } from './translations/hi';
import { id } from './translations/id';
import { ja } from './translations/ja';
import { ko } from './translations/ko';
import { ptBR } from './translations/pt-BR';
import { ru } from './translations/ru';
import { th } from './translations/th';
import { vi } from './translations/vi';
import { zhCN } from './translations/zh-CN';
import { zhHK } from './translations/zh-HK';
import { zhMO } from './translations/zh-MO';

/**
 * 每個語言一份目錄。型別讓「漏翻一個 key」變成編譯期錯誤。
 *
 * 這裡刻意不用 import.meta.glob 之類的動態匯入：全部目錄都是靜態 import，
 * 因此第一次 render 就有完整的翻譯，切換語言時不會有任何非同步等待或「先用
 * 預設語言畫一帧再換掉」。代價是十七種語言的目錄一起進 bundle（約 90KB gzip
 * 前的原始碼），對這個規模的站台來說是划算的。
 */
const CATALOGS: Readonly<Record<LocaleCode, Record<MessageKey, string>>> = {
  'zh-TW': zhTW,
  'zh-HK': zhHK,
  'zh-MO': zhMO,
  'zh-CN': zhCN,
  en,
  ja,
  ko,
  vi,
  th,
  id,
  fr,
  de,
  es,
  'pt-BR': ptBR,
  ru,
  ar,
  hi,
};

/**
 * 語言選擇的 cookie 名稱。
 *
 * 前綴沿用站名（FORUM_）是為了不與同一個主機上的其他 cookie 撞名。
 * 不設 HttpOnly：這個 cookie 的唯一用途就是讓 JS 讀回使用者的選擇，
 * 加上 HttpOnly 只會讓讀不到，然後每次都要重新問瀏覽器語言。
 */
export const LANG_COOKIE = 'FORUM_lang';

/** 一年。語言偏好不是會過期的資料，設成 session cookie 會在關掉瀏覽器後失效。 */
const COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

/* ==========================================================================
   cookie
   ========================================================================== */

/** 從 document.cookie 讀出語言 cookie；不存在或格式不符時回 null。 */
export function readLangCookie(): LocaleCode | null {
  if (typeof document === 'undefined') return null;
  // 一次 split 處理完所有 cookie，而不是每個名稱都重掃一次 document.cookie ——
  // 這個屬性每次讀取都要解析整個字串。
  for (const pair of document.cookie.split(';')) {
    const separator = pair.indexOf('=');
    if (separator < 0) continue;
    const name = pair.slice(0, separator).trim();
    if (name !== LANG_COOKIE) continue;
    // decodeURIComponent 對畸形輸入會丟例外，而 cookie 的值是使用者可控的
    // （同網域上的其他頁面就能寫），因此包起來。
    let value = '';
    try {
      value = decodeURIComponent(pair.slice(separator + 1).trim());
    } catch {
      return null;
    }
    return matchFirstLocale([value]);
  }
  return null;
}

/**
 * 寫入語言 cookie。
 *
 * 值不變時直接返回：每次載入都送一次 Set-Cookie 沒有任何好處，而且會讓
 * 「這個頁面有沒有動過 cookie」變成一個不可靠的訊號（偵測錯誤時唯一的線索）。
 *
 * Secure 只在 https 下加：localhost 是 http，硬加 Secure 會讓 cookie 被丟棄，
 * 開發時的語言選擇就存不住。SameSite=Lax 是預設值但寫出來是為了讓意圖明確 ——
 * 這個 cookie 不需要跟任何跨站請求綁在一起。
 */
export function writeLangCookie(locale: LocaleCode): void {
  if (typeof document === 'undefined') return;
  if (readLangCookie() === locale) return;
  const secure = window.location.protocol === 'https:' ? '; Secure' : '';
  document.cookie = `${LANG_COOKIE}=${encodeURIComponent(locale)}; Path=/; Max-Age=${COOKIE_MAX_AGE}; SameSite=Lax${secure}`;
}

/* ==========================================================================
   目前語言
   ========================================================================== */

/** 目前生效的語言。Provider 掛載時由 resolveInitialLocale() 設定。 */
let activeLocale: LocaleCode = DEFAULT_LOCALE;

export function currentLocale(): LocaleCode {
  return activeLocale;
}

/** 給 Intl.NumberFormat / Intl.DateTimeFormat 用的標籤。 */
export function currentIntlTag(): string {
  return localeMeta(activeLocale).htmlLang;
}
/**
 * 決定這個 document 一開始該用哪一種語言。
 *
 * 線索的優先順序代表「誰的意圖比較明確」：
 *   1. cookie        使用者在本站明確選過，這是最強的訊號。
 *   2. navigator.languages  使用者設定的偏好排序；[0] 是第一選擇。
 *   3. html lang     樣板的 lang 屬性（靜態 HTML 一定是 zh-Hant）。它排在
 *                    navigator 後面是刻意的：樣板值是站台預設，不是使用者的
 *                    表達，瀏覽器語言才是。
 *   4. DEFAULT_LOCALE 連上面三個都沒有。
 */
export function resolveInitialLocale(): LocaleCode {
  const fromCookie = readLangCookie();
  if (fromCookie) return fromCookie;

  if (typeof navigator !== 'undefined') {
    // languages 是排序過的偏好清單；language 是第一項的別名。某些舊瀏覽器只有
    // 後者，所以兩個都讀。
    const candidates = [...(navigator.languages ?? []), navigator.language].filter(
      (tag): tag is string => typeof tag === 'string' && tag !== '',
    );
    const fromBrowser = matchFirstLocale(candidates);
    if (fromBrowser) return fromBrowser;
  }

  if (typeof document !== 'undefined') {
    const fromHTML = matchFirstLocale([document.documentElement.lang]);
    if (fromHTML) return fromHTML;
  }

  return DEFAULT_LOCALE;
}

/**
 * 只改模組狀態，不碰 DOM 與 cookie。
 *
 * 存在的理由是「t() 必須在第一次 render 之前就拿到正確的語言」。Provider 把
 * 偵測結果放進 useState 的初始化函式，那個時機比任何 effect 都早；把
 * <html lang> 與 cookie 寫在 effect 裡（早於繪製）就好，不必讓偵測本身
 * 變成一個 render 期間的副作用。
 */
export function setActiveLocale(locale: LocaleCode): void {
  activeLocale = locale;
}

/**
 * 套用語言：更新模組狀態、<html lang> / <html dir> 與 cookie。四件事缺一不可。
 *
 * dir 與 lang 分開寫：lang 影響朗讀與翻譯提示，dir 影響整頁鏡像。阿拉伯文的
 * lang 是 `ar`、dir 是 `rtl`；兩者都設了瀏覽器才知道既要用阿拉伯文的發音引擎，
 * 也要把版面鏡像過來（見 locales.ts 的 LocaleMeta.dir）。
 */
export function applyLocale(locale: LocaleCode): void {
  setActiveLocale(locale);
  if (typeof document !== 'undefined') {
    const meta = localeMeta(locale);
    // <html lang> 是螢幕閱讀器朗讀語言與瀏覽器翻譯提示的依據。lang 寫錯不會
    // 看得出來，但會讓中文介面被日文發音引擎念出來。
    document.documentElement.lang = meta.htmlLang;
    // dir 一定要在 lang 之後寫：style.css 裡有 [dir="rtl"] 的覆寫，兩者同時
    // 改動時順序不影響結果，但保持一致比較好讀。
    document.documentElement.dir = meta.dir;
  }
  writeLangCookie(locale);
}

/* ==========================================================================
   翻譯
   ========================================================================== */

export type TranslateParams = Readonly<Record<string, string | number>>;

/**
 * 把訊息裡的 {name} 換成實際值。
 *
 * 用字串取代而不是 Intl.MessageFormat / 類似的函式庫：這個站的文案沒有複數、
 * 沒有性別變化、沒有巢狀複數，唯一的插值需求就是數字與識別碼。
 *
 * 替換完還殘留 {…} 代表翻譯裡的佔位符名稱寫錯了（多半是從原文複製時手誤）。
 * 這種錯誤在介面上看起來像 "{countt} 筆" —— 會被使用者看到，所以值得在開發
 * 模式下叫一聲，而不是默默顯示。
 */
function interpolate(message: string, params: TranslateParams | undefined): string {
  if (!params) return message;
  const filled = message.replace(/\{(\w+)\}/g, (match, name: string) => {
    const value = params[name];
    return value === undefined ? match : String(value);
  });
  if (process.env.NODE_ENV !== 'production' && /\{\w+\}/.test(filled)) {
    console.warn(`[FORUM] 翻譯佔位符沒被填滿：${JSON.stringify(filled)}（參數：${JSON.stringify(params)}）`);
  }
  return filled;
}

/**
 * 取一行譯文。
 *
 * 找不到 key 時的三段式處置：先回退到繁體中文（翻譯檔漏了一個 key 的情況），
 * 再回退到 key 本身。三段都拿不到時回傳 key 字串而不是丟例外，因為一個漏翻的
 * 按鈕不該讓整頁白畫面。
 */
export function t(key: MessageKey, params?: TranslateParams): string {
  const active = CATALOGS[activeLocale] as Record<MessageKey, string>;
  const message = active[key] ?? zhTW[key] ?? key;
  return interpolate(message, params);
}

/* ==========================================================================
   延後翻譯的訊息
   ==========================================================================
   狀態列、摘要、錯誤訊息這些東西的來源有兩種，混在一起會讓其中一種壞掉：

     - 「我們自己的文案」：例如「向下滑動載入更多」「12 位用戶」。它是已載入
       資料的純函式結果。
     - 「後端回的訊息」：例如「暱稱已存在」。我們沒有譯文，也不該譯。

   過去兩種都存成字串，於是在 useState 裡就呼叫了 t()。這會造成一個明確的
   bug：使用者切換語言時，畫面上那些「不隨著重新 render 改變」的狀態列會停在
   舊語言，直到下一次重新載入才變 —— 而摘要列之類的東西常常就是使用者切換
   語言後第一眼會去看的地方。

   解法是不要在事件處理器裡就翻譯：把 key 與參數存進 state，render 時才用
   tr() 解析。後端來的訊息是字串，tr() 會原樣通過（不該替它翻譯），所以同一
   個欄位可以同時放兩種來源。
   */

/**
 * 一則「尚未翻譯」的訊息。
 *
 * params 刻意宣告成 `| undefined` 而不是 optional：這個專案開了
 * exactOptionalPropertyTypes，而 msg() 一定會把 params 這個鍵放進物件（值可能是
 * undefined）。宣告成 optional 會讓回傳型別與 Message 對不上 —— 兩種寫法在
 * 沒有 exactOptionalPropertyTypes 時都過，開了之後只有現在這樣過得了。
 */
export interface Message {
  readonly key: MessageKey;
  readonly params?: TranslateParams | undefined;
}

/** 建立一則延後翻譯的訊息。存進 state 時用這個，不要在 setter 裡呼叫 t()。 */
export function msg(key: MessageKey, params?: TranslateParams): Message {
  return { key, params };
}

/**
 * 把 state 裡的訊息解析成字串。
 *
 * 接受 Message 或字串：字串代表「這是後端回來的訊息，或已經是使用者輸入的
 * 內容」，原樣輸出。這個分支是整個設計能成立的原因 —— 沒有它，一個混合了兩種
 * 來源的欄位就只能選一種儲存方式，而另一種必然出錯。
 */
export function tr(message: Message | string): string {
  return typeof message === 'string' ? message : t(message.key, message.params);
}
