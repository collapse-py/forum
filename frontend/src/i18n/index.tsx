/*
 * 語言的 React 層（src/i18n/index.tsx）
 *
 * runtime.ts 是純邏輯（偵測、cookie、t()），這個檔案只負責把它接到 React 上。
 * 分成兩半的理由是 core.ts 的數字／日期格式化也需要知道目前語言，而 core.ts
 * 刻意不含 React —— 把「目前語言」放在純模組裡，兩邊就不會互相依賴。
 *
 * 一個 document 只有一種語言，因此這裡不需要多語言並存、不需要載入非同步的
 * 語言檔，也沒有「先用預設語言畫一帧再換掉」的問題：所有目錄都是同步 import，
 * t() 在第一次 render 就已經是正確的語言。
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

import { SITE_NAME, SITE_SHORT_NAME } from '../site';
import { localeMeta, LOCALES, type LocaleCode, type LocaleMeta } from './locales';
import type { MessageKey, MessageParams } from './messages';
import { applyLocale, resolveInitialLocale, setActiveLocale, t } from './runtime';

export { t, msg, tr } from './runtime';
export type { Message, TranslateParams } from './runtime';
export { LOCALES, localeMeta, DEFAULT_LOCALE, matchLocale, matchFirstLocale } from './locales';
export type { LocaleCode, LocaleMeta } from './locales';
export type { MessageKey, MessageParams } from './messages';

/**
 * 模組求值時就跑一次偵測。
 *
 * 位置是刻意的：這是唯一能保證「早於任何 render」的地方。若改在 Provider 的
 * useState 初始化函式裡設定模組狀態，React 仍可能在第一次 render 的某個更早
 * 位置呼叫 t()（例如 render 一個 sibling 時），那時 activeLocale 還是預設值。
 * 模組層的 const 在任何元件開始 render 之前就已經算好。
 *
 * 副作用只有「設定一個模組變數」，不碰 DOM、不寫 cookie。
 */
const INITIAL_LOCALE: LocaleCode = (() => {
  const resolved = resolveInitialLocale();
  setActiveLocale(resolved);
  return resolved;
})();

export interface I18nContextValue {
  locale: LocaleCode;
  meta: LocaleMeta;
  /** 可選的語言清單。給切換器 render 選項用。 */
  locales: readonly LocaleMeta[];
  setLocale: (locale: LocaleCode) => void;
  t: (key: MessageKey, params?: MessageParams) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<LocaleCode>(INITIAL_LOCALE);

  const setLocale = useCallback((next: LocaleCode) => {
    // 先改模組狀態再 setState：t() 在 setState 觸發的重繪之前就會被下一個
    // 畫面讀到，順序反過來會有一個中間狀態是「state 已換、翻譯還是舊的」。
    applyLocale(next);
    setLocaleState(next);
  }, []);

  /*
   * <html lang> 與 <html dir> 寫在 useLayoutEffect 而不是 useEffect：讀螢幕軟體
   * 在繪製那一刻就決定用哪個發音引擎，遲一個 frame 會讓它先讀出錯誤的語言；
   * dir 晚一個 frame 則會讓阿拉伯文先以 LTR 版面畫一帧再鏡像。useEffect 聽起來
   * 無害，但那是給「cookie 這種沒有畫面依賴的東西」用的。
   */
  useLayoutEffect(() => {
    const meta = localeMeta(locale);
    document.documentElement.lang = meta.htmlLang;
    document.documentElement.dir = meta.dir;
  }, [locale]);

  /*
   * 語言選擇在每次載入都寫回 cookie，包含「第一次由瀏覽器語言推導出來」的那一次。
   *
   * 這是刻意的：需求是「自動切換到瀏覽器語言」加上「保存 cookie」，兩者合起來
   * 最好的行為是 —— 使用者第一次進站看到自己瀏覽器的語言，此後不論他怎麼調整
   * 瀏覽器偏好，站內介面都維持他看到的那一種（除非他自己在這裡改）。只在使用者
   * 手動切換時寫入的話，會出現另一種結果：改了瀏覽器語言，隔天整個介面就自己
   * 變成另一種語言，而且沒有任何提示說是為什麼。
   *
   * writeLangCookie 內部會比對既有值，相同就不重複寫。
   */
  useEffect(() => {
    applyLocale(locale);
  }, [locale]);

  const value = useMemo<I18nContextValue>(
    () => ({ locale, meta: localeMeta(locale), locales: LOCALES, setLocale, t }),
    [locale, setLocale],
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

/**
 * 取用語言功能。必須在 I18nProvider 內使用。
 *
 * 為什麼大部分元件其實不需要呼叫它：t() 是純函式，讀模組狀態，因此在任何地方
 * （包含自訂 hook 與純函式，例如 useFeed 與 searchStatusText）都能直接用。
 * useI18n 只有在需要「目前是哪一種語言」或「切換」時才必要 —— 語言切換器、
 * 頁面標題，以及依語言決定版面（本站沒有）的情況。
 */
export function useI18n(): I18nContextValue {
  const value = useContext(I18nContext);
  if (!value) throw new Error('[FORUM] useI18n 必須在 I18nProvider 內使用。');
  return value;
}

/**
 * 把分頁標題設成指定的一條訊息，並在語言切換時跟著換。
 *
 * 為什麼需要這個：靜態 HTML 的 <title> 只能是繁體中文（送出 HTML 時 React
 * 還沒跑），所以切換語言後分頁籤不換會是一個很明顯的不一致。依賴陣列裡放
 * locale 而不是只放 key，正是為了讓語言改變時重新執行。
 *
 * key 傳 null 代表「現在還不知道該用哪個標題」，這一次執行什麼都不做。
 * 用戶管理頁需要它：通過身分檢查與否決定標題是「用戶管理」還是「登入」，
 * 而在那之前連是哪一頁都還不確定。
 *
 * 站名一律在這裡注入（{site} 與 {brand}）：title.* 這組鍵的譯文只寫頁面
 * 名稱與連接符，站名來自設定檔（見 src/site.ts）。集中在一個函式裡傳參，
 * 各頁就不必記得哪幾條標題含有站名 —— 少傳一次參數的症狀是標題裡殘留
 * 字面量 "{site}"，那不是編譯期錯誤。
 */
export function usePageTitle(key: MessageKey | null): void {
  const { locale } = useI18n();
  useEffect(() => {
    if (key === null) return;
    document.title = t(key, { site: SITE_NAME, brand: SITE_SHORT_NAME });
  }, [key, locale]);
}
