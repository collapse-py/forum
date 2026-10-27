/*
 * Service Worker（service-worker.ts）
 *
 * 策略只有兩條：document 與動態路徑走 network-first（內容要新），
 * 靜態資產走 cache-first（內容不變且檔名帶 hash）。
 *
 * 這一檔在 WebWorker 環境求值，因此型別檢查跑在獨立的 tsconfig.sw.json
 * （lib 為 WebWorker，不含 DOM）。建置時由 vite.config.ts 單獨編譯並輸出成
 * 根目錄的 service-worker.js —— 位置不能變，因為它的 scope 決定了
 * /forum 底下所有頁面都在控制範圍內。
 */

/// <reference lib="webworker" />

/**
 * lib.webworker.d.ts 把 `self` 宣告成 `WorkerGlobalScope`，因此 skipWaiting()、
 * clients、ExtendableEvent 與 FetchEvent 全部看不到 —— 那些只掛在
 * ServiceWorkerGlobalScope 上。結構上它是 WorkerGlobalScope 的延伸，但宣告檔
 * 沒有把這個關係表達成子型別（intersection 讓直接轉型被視為不夠重疊），
 * 因此先經 unknown 收斂。
 */
const sw = self as unknown as ServiceWorkerGlobalScope;

const CACHE_NAME = 'FORUM-forum-v1';
const PRECACHE_URLS = [
  '/forum',
  '/forum/login',
  '/forum/new',
  '/forum/profile',
  '/forum/others-profile',
  '/forum-manifest.json',
  '/asset/logo.png',
  '/asset/pwa-icon-192.png',
  '/asset/pwa-icon-512.png',
];

sw.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then(async (cache) => {
      await Promise.all(
        PRECACHE_URLS.map(async (url) => {
          try {
            await cache.add(url);
          } catch (error) {
            console.warn('[SW] Unable to precache:', url, error);
          }
        }),
      );
    }),
  );
  void sw.skipWaiting();
});

sw.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) =>
      Promise.all(
        cacheNames.filter((name) => name !== CACHE_NAME && name.startsWith('FORUM-')).map((name) => caches.delete(name)),
      ),
    ),
  );
  void sw.clients.claim();
});

sw.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  /*
   * 只攔 http(s)。頁面裡的請求不一定來自頁面本身：瀏覽器擴充功能的 content
   * script 也會在分頁中發請求（chrome-extension://、moz-extension:// 等），
   * 那些請求同樣經過這個 Service Worker。caches.put 只吃 http(s)，硬要快取
   * 會丟 "Request scheme 'chrome-extension' is unsupported"，而我們對
   * cache.put 的呼叫是 void 的丟進背景 —— 錯誤變成沒有 handle 的 Promise
   * rejection，主控台一行紅字，看了卻完全不知道與自己有關。不 respondWith
   * 就夠了：這類請求本來就該由瀏覽器自己完成。
   */
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    return;
  }

  /*
   * API 與後台的回應**不進快取**。這三個前綴底下的每一份 JSON 都與請求者的
   * 身分有關（自己的貼文、通知、後台的使用者與稽核紀錄），把它存進 Cache API
   * 等於把登入者的資料寫在裝置上，並能在離線時重放給同一裝置上的任何頁面。
   * 後端現在也對所有 JSON 回 `Cache-Control: no-store`，因此這裡的快取嘗試
   * 本來就會被拒；拿掉它是為了讓意圖在這一層也看得出來，而不是靠一個表頭。
   */
  if (
    url.pathname.startsWith('/api/') ||
    url.pathname.startsWith('/auth/') ||
    url.pathname.startsWith('/admin')
  ) {
    event.respondWith(networkOnly(request));
    return;
  }

  /*
   * 論壇頁面（/forum 與其子路徑）走 networkFirst：那些 HTML 殼本身不含使用者
   * 資料，離線時重放舊殼沒有外洩問題，而「先網路、拿到就更新快取」讓頁面在
   * 有網路時永遠拿到最新殼。
   */
  if (url.pathname.startsWith('/forum') || request.destination === 'document') {
    event.respondWith(networkFirst(request));
    return;
  }

  event.respondWith(cacheFirst(request));
});

/** 只走網路、永遠不儲存。用於帶身分的回應（見上方的前綴判斷）。 */
async function networkOnly(request: Request): Promise<Response> {
  try {
    return await fetch(request)
  } catch {
    return new Response(JSON.stringify({ ok: false, error: 'Offline' }), {
      headers: { 'Content-Type': 'application/json' },
      status: 503,
    })
  }
}

async function cacheFirst(request: Request): Promise<Response> {
  if (request.method !== 'GET') {
    return fetch(request);
  }
  const cached = await caches.match(request);
  if (cached) return cached;
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(CACHE_NAME);
      void cache.put(request, response.clone());
    }
    return response;
  } catch {
    return new Response('Offline', { status: 503, statusText: 'Offline' });
  }
}

async function networkFirst(request: Request): Promise<Response> {
  if (request.method !== 'GET') {
    return fetch(request);
  }
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(CACHE_NAME);
      void cache.put(request, response.clone());
    }
    return response;
  } catch {
    const cached = await caches.match(request);
    if (cached) return cached;
    return new Response(JSON.stringify({ error: 'Offline' }), {
      headers: { 'Content-Type': 'application/json' },
      status: 503,
    });
  }
}
