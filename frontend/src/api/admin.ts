/*
 * 後臺專用的 fetch 封裝（src/api/admin.ts）
 *
 * 與 core.ts 的 requestJSON 共用訊息解析，但 401 的反應不同：後臺一律導回
 * /admin（登入閘門），公開頁則導回 /forum/login 並帶 return 參數。這兩條路徑
 * 語意不同、位置不同，硬要合成一支函式只會讓呼叫端多一個參數。
 *
 * 另一個差別：後臺的錯誤訊息一律從後端的 message 欄位取出並以 ApiError 包裝，
 * 呼叫端只需要 try/catch + toast，不必各自解析錯誤形狀。
 */

import { ApiError } from '../core';
import { t } from '../i18n';

export { ApiError };

export interface AdminRequest<TBody = unknown> {
  method?: string | undefined;
  body?: TBody | undefined;
  signal?: AbortSignal | undefined;
}

/** 旗標避免多個並行請求（列表 + 標籤同時載入）重複導頁。 */
let redirecting = false;

function bounceToLogin(): void {
  if (redirecting) return;
  redirecting = true;
  // replace 而非 href：登入閘門不該留一筆不該存在的瀏覽紀錄，
  // 否則使用者的「上一頁」會回到同一個閘門。
  window.location.replace('/admin');
}

async function readErrorMessage(response: Response): Promise<string> {
  let payload: { message?: unknown; error?: unknown } = {};
  try {
    payload = (await response.json()) as { message?: unknown; error?: unknown };
  } catch {
    payload = {};
  }
  const message = payload.message ?? payload.error;
  return typeof message === 'string' && message ? message : t('error.requestStatus', { status: response.status });
}

/**
 * 回傳型別由呼叫端以 T 指定：後端回應形狀是這個前端的 API 契約，寫在
 * 呼叫處比 any 更能擋下欄位改名。
 */
export async function adminApi<T = unknown, TBody = unknown>(
  path: string,
  options: AdminRequest<TBody> = {},
): Promise<T> {
  const { method = 'GET', body, signal } = options;

  // headers 一定要是自訂的 Record 物件才能在送出前補上 Content-Type；
  // signal 只在有傳時才掛上去，避免把 undefined 交給 fetch。
  const init: RequestInit & { headers: Record<string, string> } = {
    method,
    credentials: 'same-origin',
    headers: {},
  };
  if (signal) init.signal = signal;

  if (body !== undefined) {
    init.headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify(body);
  }

  const response = await fetch(path, init);

  if (response.status === 401) {
    bounceToLogin();
    throw new ApiError(t('error.adminSessionExpired'), 401);
  }

  if (!response.ok) {
    throw new ApiError(await readErrorMessage(response), response.status);
  }

  // 204 / 空回應主動當作 null，避免 response.json() 在空 body 上拋例外。
  const text = await response.text();
  if (!text) return null as T;
  try {
    return JSON.parse(text) as T;
  } catch {
    return null as T;
  }
}

/** /api/check 永遠回 200 + { ok, isAdmin }，因此不會觸發 401 導頁。 */
export async function checkAdmin(): Promise<boolean> {
  try {
    const data = await adminApi<{ isAdmin?: boolean }>('/api/check');
    return data?.isAdmin === true;
  } catch {
    return false;
  }
}
