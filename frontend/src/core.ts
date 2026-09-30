/*
 * 全站共用的底層（src/core.ts）
 *
 * 這裡只放「沒有任何 UI 語意」的東西：顯示格式化、錯誤訊息解析、fetch 封裝。
 * 純函式與 I/O 邊界，刻意不放任何 React 元件或 hook —— 它要能被
 * admin 的 provider、論壇公開頁的 hook 與 service worker 之外的任何模組引用，
 * 而不把 React 或後臺樣式拖進來。
 *
 * 與舊版 forum-core.ts / admin-core.ts 的差異只有一處值得說明：
 * 舊版的 escapeHTML 在 React 版裡沒有對應物，也不需要。React 在建立文字節點
 * 時本來就會跳脫，`dangerouslySetInnerHTML` 只留給「純常數字串」這一個用途
 * （見 icons.tsx 的說明）。因此這裡改提供 text()：它處理「後端欄位可能是
 * undefined 而舊版要顯示空字串」這個規則，讓兩邊的顯示語意維持一致。
 *
 * 這裡引用 i18n/ 並不會把 React 拖進來：runtime.ts 是純模組（沒有元件、沒有
 * hook），而 core.ts 同樣刻意維持這個性質 —— 它要能被 provider、公開頁的
 * hook 與任何非 UI 的模組引用。數字與日期要跟著語言走，唯一的代價就是這兩行
 * 匯入。
 */

import { currentIntlTag, t } from './i18n/runtime';
import type { Message } from './i18n/runtime';

/* ==========================================================================
   顯示格式化
   ==========================================================================
   數字與日期都跟著目前語言走，因此 formatter 必須在語言改變時重建。
   重建的代價不低（Intl 的格式器建構會去讀 ICU 資料），所以用快取：鍵是
   locale，語言沒換就沿用同一個實例。
 */

/*
 * 建 Intl formatter 的 try/catch 與 Map 快取看似多餘，但少了它壞掉的代價不對稱：
 * Intl.NumberFormat 會在標籤無法解析時擲 RangeError，而這兩支函式被表格裡的
 * 每一格呼叫 —— 一個例外就讓整頁白畫面，連「錯誤訊息」都來不及顯示。快取同時
 * 讓正常路徑不必重複建構。快取無上限是刻意的：一個 document 最多四種語言。
 */
const numberFormats = new Map<string, Intl.NumberFormat>();
const dateTimeFormats = new Map<string, Intl.DateTimeFormat>();

function numberFormatFor(tag: string): Intl.NumberFormat {
  const cached = numberFormats.get(tag);
  if (cached) return cached;
  let format: Intl.NumberFormat;
  try {
    format = new Intl.NumberFormat(tag);
  } catch {
    // 標籤不合法時退回不分化的數字格式：少一點排版美化，但不能壞掉。
    format = new Intl.NumberFormat();
  }
  numberFormats.set(tag, format);
  return format;
}

function dateTimeFormatFor(tag: string): Intl.DateTimeFormat {
  const cached = dateTimeFormats.get(tag);
  if (cached) return cached;
  const options: Intl.DateTimeFormatOptions = { dateStyle: 'medium', timeStyle: 'short' };
  let format: Intl.DateTimeFormat;
  try {
    format = new Intl.DateTimeFormat(tag, options);
  } catch {
    // dateStyle / timeStyle 是 ES2020 的選項。缺這兩項的舊引擎不支援它們，
    // 會擲 TypeError 而 RangeError —— 因此需要明確的逐項展開替代。
    format = new Intl.DateTimeFormat(tag, {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
  }
  dateTimeFormats.set(tag, format);
  return format;
}

/** 空值顯示空字串。這是 React 版取代 escapeHTML(String(value ?? '')) 的唯一入口。 */
export function text(value: unknown): string {
  return String(value ?? '');
}

export function formatNumber(value: unknown): string {
  return numberFormatFor(currentIntlTag()).format(Number(value) || 0);
}

/** 無法解析成日期時原樣輸出字串；空值輸出全形破折號（與 common.placeholder 同義）。 */
export function formatDateTime(value: unknown): string {
  if (!value) return t('common.placeholder');
  const date = new Date(value as string | number);
  return Number.isNaN(date.getTime()) ? text(value) : dateTimeFormatFor(currentIntlTag()).format(date);
}

/* ==========================================================================
   錯誤
   ========================================================================== */

/**
 * 從 catch 綁定值取出可顯示的訊息。useUnknownInCatchVariables 下 catch 進來的是
 * unknown；非 Error 的拋出（字串、物件）也確實發生過，因此統一收斂。
 */
export function errorMessage(error: unknown, fallback: string): string {
  const server = serverErrorMessage(error);
  return server === null ? fallback : server;
}

/**
 * 只取出「後端／被丟出的錯誤自己帶的訊息」，沒有就回 null。
 *
 * 這是 errorMessage 的第一步，拆出來是為了讓 i18n 能分辨來源。一段錯誤訊息可能
 * 是後端回的中文字串（我們沒有譯文，也不該譯），也可能是我們自己的文案
 * （該跟著語言走）。兩者混成一個字串之後就再也分不開，因此需要存在狀態裡的
 * 錯誤訊息要走這裡：fallback 存成 msg()、後端字串存成 string，render 時由
 * tr() 決定要不要翻譯。
 */
function serverErrorMessage(error: unknown): string | null {
  if (error instanceof Error && error.message) return error.message;
  if (typeof error === 'string' && error) return error;
  return null;
}

/**
 * 同時處理兩種來源的錯誤訊息，回一個可以直接放進 state 的值。
 *
 * 後端有訊息 → 回那個字串（不翻譯）；沒有 → 回 fallback（延後翻譯）。
 * 搭配 i18n 的 tr() 使用：{tr(errorText(error, msg('feed.postsFailed')))}
 */
export function errorText(error: unknown, fallback: Message): Message | string {
  const server = serverErrorMessage(error);
  return server === null ? fallback : server;
}

/* ==========================================================================
   Fetch
   ========================================================================== */

/** 後端錯誤回應的 message 欄位形狀。 */
export interface ErrorPayload {
  message?: string;
  error?: string;
}

/** 後端回錯時的 Error 子類，保留 HTTP status 供上層分流。 */
export class ApiError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

/**
 * 「這次請求其實需要登入」訊號。
 *
 * 判斷依據不只 401：後端對未登入的寫入請求回 303 轉址到登入頁，而 fetch 會
 * 自動跟著轉，最後拿到的是 200 + text/html 的登入頁 HTML。那時
 * response.status 是 200，只有 redirected 與 content-type 還看得出真相，
 * 因此兩個條件都必須檢查。
 */
export class LoginRequiredError extends Error {
  constructor() {
    super(t('error.loginRequired'));
    this.name = 'LoginRequiredError';
  }
}

export interface RequestOptions {
  method?: string | undefined;
  /** JSON body。給定時自動補上 Content-Type，不與 form 同時使用。 */
  json?: unknown;
  /** multipart body（圖片上傳）。此時不設定 Content-Type，讓瀏覽器帶 boundary。 */
  form?: FormData | undefined;
  signal?: AbortSignal | undefined;
  /** 後端沒給訊息時要顯示的文案。 */
  fallback?: string | undefined;
}

function isLoginRedirect(response: Response): boolean {
  if (response.status === 401 || response.status === 303) return true;
  if (!response.redirected) return false;
  // redirected 且仍回 JSON：正常的重新導向（例如 OAuth 成功後回 /forum），
  // 不算登入需求。redirected 卻回 HTML 才是被導到登入閘門。
  return !(response.headers.get('content-type') ?? '').includes('application/json');
}

async function readErrorMessage(response: Response, fallback: string): Promise<string> {
  let payload: ErrorPayload = {};
  try {
    payload = (await response.json()) as ErrorPayload;
  } catch {
    payload = {};
  }
  return payload.message || payload.error || fallback;
}

/**
 * 公開頁（/forum/*）唯一的後端呼叫出口。
 *
 * 與後臺 adminApi 的分界不是「哪裡比較常用」，而是 401 該送去哪裡：公開頁
 * 導回 /forum/login 並帶 return 參數，後臺導回 /admin。兩者共用這裡的
 * 訊息解析與 JSON 解析，但跳轉目標不同，因此維持兩支函式。
 */
export async function requestJSON<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', json, form, signal, fallback = t('error.request') } = options;

  const init: RequestInit = { method, credentials: 'same-origin' };
  if (signal) init.signal = signal;
  if (form !== undefined) {
    init.body = form;
  } else if (json !== undefined) {
    init.headers = { 'Content-Type': 'application/json' };
    init.body = JSON.stringify(json);
  }

  const response = await fetch(path, init);
  if (isLoginRedirect(response)) throw new LoginRequiredError();
  if (!response.ok) throw new ApiError(await readErrorMessage(response, fallback), response.status);

  // 204 / 空回應主動當作 null，避免 response.json() 在空 body 上拋例外。
  const text = await response.text();
  if (!text) return null as T;
  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(fallback);
  }
}

/** 只取後端 JSON 物件、不因非 2xx 丟錯（/api/check 永遠回 200）。 */
export async function readJSONObject(response: Response): Promise<ErrorPayload & { ok?: boolean }> {
  try {
    return (await response.json()) as ErrorPayload & { ok?: boolean };
  } catch {
    return {};
  }
}

/* ==========================================================================
   登入與導覽
   ========================================================================== */

/** 帶 return 參數導回登入頁，讓 OAuth 完成後回到原本想去的路徑。 */
export function goToLogin(): void {
  window.location.href = '/forum/login?return=' + encodeURIComponent(window.location.pathname);
}

/** 登出。成功後整頁重載，讓所有衍生狀態（頭像、計數、按鈕）一併歸零。 */
export async function logout(): Promise<void> {
  await fetch('/api/logout', { method: 'POST' });
  window.location.reload();
}
