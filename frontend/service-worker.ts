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

  if (
    url.pathname.startsWith('/api/') ||
    url.pathname.startsWith('/auth/') ||
    url.pathname.startsWith('/forum') ||
    url.pathname.startsWith('/admin')
  ) {
    event.respondWith(networkFirst(request));
    return;
  }

  if (request.destination === 'document') {
    event.respondWith(networkFirst(request));
    return;
  }

  event.respondWith(cacheFirst(request));
});

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
