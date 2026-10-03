import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import react from '@vitejs/plugin-react';
import { defineConfig, transformWithEsbuild, type Plugin } from 'vite';
import javascriptObfuscator from 'vite-plugin-javascript-obfuscator';

/** 以本檔位置為基準，而不是 process.cwd()，讓 build 從任何目錄執行都得到相同結果。 */
const projectRoot = dirname(fileURLToPath(import.meta.url));

/** 需要混淆的應用程式原始碼，判斷依據同樣是 projectRoot，不受執行目錄影響。 */
const appSourcePrefix = `${resolve(projectRoot, 'src').replace(/\\/g, '/')}/`;

/**
 * 混淆的範圍判斷：src/ 之下、且去掉 Vite 附加的查詢字串後是 JS/TS 檔。
 *
 * 刻意不用插件的 glob 字串比對：字串會被 resolve() 成 process.cwd() 底下的
 * 路徑，等於把「從任何目錄建置都要得到相同結果」這條規則又破壞一次。
 * id 交進來時 anymatch 已把分隔符正規化成 '/'，這裡再轉一次只是求穩。
 */
function isAppSourceModule(id: string): boolean {
  const filePath = id.replace(/\\/g, '/').replace(/\?.*$/, '');
  return filePath.startsWith(appSourcePrefix) && /\.[cm]?[jt]sx?$/.test(filePath);
}

/**
 * 輸出兩個必須「在根層、且不被 hash 改名」的執行期資產。
 *
 * 為什麼不能交給 Vite 的常規入口：Rollup 會把模組入口改名成
 * assets/<name>-<hash>.js，而這兩個檔案的路徑是被外部世界綁死的 ——
 *   - /service-worker.js：瀏覽器用它來註冊，位置改了等於沒有 service worker，
 *     而它的 scope 也決定了 /forum 底下所有頁面是否在控制範圍內。
 *   - /forum-manifest.json：被各頁 <link rel="manifest"> 與後端路由引用，
 *     而它的站名是後端在送出時以設定檔代入的（見 backend/forum/httpapi/site.go）。
 *
 * service-worker.ts 必須單獨編譯（它是 WebWorker 環境，不是頁面模組，
 * 也不該進 shared chunk），因此在這裡直接用 Vite 內建的 esbuild 轉換。
 */
function copyForumRuntimeAssets(): Plugin {
  return {
    name: 'copy-forum-runtime-assets',
    // post 是必要的：Vite 內建的 build-html 插件會在 generateBundle 的最後
    // 用它自己記憶體裡的 HTML 字串重新 emit 一次頁面檔，順序排在我們之後就
    // 會把改寫蓋回去（href 會再次指向雜湊複本，症狀是站名佔位符原樣進到 PWA
    // 名稱）。排在它之後，頁面資產才會以我們改好的內容寫出。
    enforce: 'post',
    async generateBundle(_options, bundle) {
      const worker = await transformWithEsbuild(
        readFileSync(resolve(projectRoot, 'service-worker.ts'), 'utf8'),
        'service-worker.ts',
        {
          // IIFE：service worker 以傳統 script（非 module）註冊與求值。
          format: 'iife',
          target: 'es2022',
          minify: true,
        },
      );

      this.emitFile({ type: 'asset', fileName: 'service-worker.js', source: worker.code });
      this.emitFile({
        type: 'asset',
        fileName: 'forum-manifest.json',
        source: readFileSync(resolve(projectRoot, 'forum-manifest.json'), 'utf8'),
      });

      /*
       * 把 <link rel="manifest"> 拉回根層路徑。
       *
       * 問題：Vite 會把 HTML 裡的 link[href] 當成建置資產，emit 一份
       * assets/forum-manifest-<hash>.json 並把 href 改指向它。那份複本由
       * /assets/ 的靜態檔案伺服器直接送出，不會經過後端的站名取代，PWA 於是
       * 用字面量 "{{FORUM_NAME}}" 當成應用程式名稱 —— 而且帶 immutable 快取，
       * 改設定也不會更新。這與本檔存在的理由（路徑被外部世界綁死）正好相反。
       *
       * 為什麼在 generateBundle 修：Vite 內建的 build-html 插件也是這裡的
       * 呼叫者之一，transformIndexHtml 階段尚未產生最終 HTML；generateBundle
       * 階段的 HTML 資產則已經是完整輸出，且尚未寫進 dist。
       *
       * 為什麼連雜湊複本一起刪掉：留著它等於同一份 manifest 存在兩個位置，
       * 下一個人會不知道哪一份才是權威。刪掉後若 href 改寫失敗，症狀是
       * manifest 404（安裝提示消失），比兩份並存更容易察覺。
       *
       * 刪除是直接從 bundle 物件移除（Rollup 3 的 PluginContext 沒有
       * deleteFile，那是 Rollup 4 才有的 API）：generateBundle 結束後才會寫檔，
       * 因此在這裡拿掉 key 就等於檔案不會被產出。
       */
      const hashedManifests = Object.keys(bundle).filter((name) =>
        /^assets\/forum-manifest-[\w-]+\.json$/.test(name),
      );
      if (hashedManifests.length === 0) return;

      for (const [name, output] of Object.entries(bundle)) {
        if (!name.endsWith('.html') || output.type !== 'asset') continue;
        let source = String(output.source);
        for (const hashed of hashedManifests) {
          source = source.split(`/${hashed}`).join('/forum-manifest.json');
        }
        output.source = source;
      }
      for (const hashed of hashedManifests) delete bundle[hashed];
    },
  };
}

/*
 * 入口仍是十六個 HTML 檔，不是單一 SPA。
 *
 * 為什麼不合成一個：後端 server.go 的 handleForumPage / handleForumLoginPage
 * 逐一把 /forum、/forum/new、/forum/profile、/forum/others-profile、/forum/login
 * 與十個 /admin* 路徑對應到各自的 HTML 檔名，而「未列出的子路徑必須 404」是
 * 那段 switch 的刻意設計。改成 SPA fallback 就要把這層白名單換成「任何路徑都
 * 吐同一份 index.html」，等於放寬了一條刻意的路由規則。
 *
 * 每個 HTML 內只有一個 <div id="root">、一支絕對路徑的入口模組，以及一個
 * <style> 區塊放該頁自己渲染的樣式。頁面程式碼全部在 src/ 之下（TSX），
 * 共用樣式在 /style.css。
 *
 * 兩條 CSP 約束的差別要分清楚：
 *   - script-src 'self' 沒有 'unsafe-inline'，因此掛載點不能靠內聯腳本建立 ——
 *     這也是各頁必須各自有一支入口模組的原因。
 *   - style-src 沒有 'unsafe-inline'，但 <style> 區塊是靠後端在啟動時從磁碟上
 *     的 HTML 算出的 SHA-256 授權的（見 backend/forum/httpapi/securityheaders.go
 *     的 inlineStyleHashes）。雜湊的輸入就是這裡建置出來的 dist/*.html，因此
 *     不論 Vite 有沒有改寫內聯樣式，授權都會自動跟上。
 *
 * frontendShellFiles（後端）列的十六個檔名必須與上面的 input 一致：少一個，那一頁
 * 的 <style> 就沒有授權，症狀是整頁沒有版面。
 *
 * 混淆（vite-plugin-javascript-obfuscator）刻意只做最低限度的一組選項。
 *
 * - apply: 'build'：插件預設是 serve 與 build 都跑，那等於每次 HMR 都要先被
 *   重新生成 AST 才能看到畫面，而 dev 的存在就是為了看原始碼與 stack trace。
 * - include 限定 src/：順手擋掉 Vite 的虛擬模組（HTML proxy、\0vite/…）；
 *   node_modules 走插件預設的排除規則 —— React 混淆沒有防護價值，卻最容易壞掉。
 * - 其餘選項全是預設值，寫出來並註明原因是為了讓「不要打開」變成一個有理由的
 *   決定，而不是留給下一個人去猜：
 *     · controlFlowFlattening / deadCodeInjection：每個模組膨脹數倍。這裡是九個
 *       入口共用同一包程式碼，bundle 成長會直接變成使用者等更久。
 *     · debugProtection / selfDefending：兩者都會插入 debugger 與以 Function
 *       構造式做出來的自我檢查。後端 CSP 的 script-src 只有 'self'（沒有
 *       'unsafe-eval'，見 backend/forum/httpapi/securityheaders.go），一旦需要
 *       eval 就是整頁被擋；debugger 迴圈對開著 devtools 的人也是干擾。
 *     · disableConsoleOutput：程式裡的 console.warn 是真的在回報問題，抹掉沒有
 *       任何好處。
 *
 * 順序不必操心：插件自帶 enforce: 'post'，一定在 React（TSX → JS）之後執行，
 * 而 Vite 預設的 build.minify 仍會在 renderChunk 把輸出壓小，兩者不衝突。
 *
 * 已知沒蓋到：service-worker.ts 由 copyForumRuntimeAssets() 用 esbuild 單獨編譯
 * 成 asset，transform 掛不到它，因此 service-worker.js 維持未混淆。
 */
export default defineConfig({
  base: '/',
  /*
   * 開發模式的後端代理。少了這一組，npm run dev 是開箱即壞的狀態。
   *
   * 原因：dev server 只服務 5173，而前端的 API 呼叫全部是相對路徑（/api/*、
   * /auth/*、/healthz）。這些請求因此會打到 Vite 自己 —— 拿到的是 HTML 或
   * 404，而不是後端的 JSON。症狀是「頁面載得出来但永遠顯示未登入」，而且
   * console 裡只有一串 HTML 內容型別的抱怨，看起來像後端的問題。
   *
   * 目標埠刻意寫成明碼 8088 而不是讀環境變數：後端的 SERVER_PORT 預設就是
   * 8088（見 backend/forum/config 的 applyDefaults），而這個設定存在的目的
   * 是讓「改了後端埠的人」一眼看出這裡也要跟著改。從 process.env 讀取會讓
   * 「proxy 指向哪裡」變成一個需要追查三個檔案才能回答的問題。
   *
   * 刻意**不**代理 /files/*：開發模式下前端取圖走的是 FILES_SERVER_PUBLIC_URL
   *（預設直連 :7070 的跨來源請求），把它轉進 8088 會讓圖片請求多繞一圈，並與
   * 媒體 token 的驗證路徑（後端簽發、檔案伺服器驗證）互動出意料之外的行為：
   * 症狀是 dev 模式看得到圖、生產模式卻 401，而兩邊的差別只是一個 proxy 設定。
   */
  server: {
    proxy: {
      '/api': 'http://localhost:8088',
      '/auth': 'http://localhost:8088',
      '/healthz': 'http://localhost:8088',
    },
  },
  plugins: [
    react(),
    copyForumRuntimeAssets(),
    javascriptObfuscator({
      apply: 'build',
      include: isAppSourceModule,
      options: {
        stringArray: true,
        rotateStringArray: true,
        stringArrayThreshold: 0.75,
        controlFlowFlattening: true,
        deadCodeInjection: false,
        debugProtection: false,
        selfDefending: false,
        disableConsoleOutput: false,
      },
    }),
  ],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        forum: 'forum.html',
        forumLogin: 'forum-login.html',
        forumNew: 'forum-new.html',
        forumProfile: 'forum-profile.html',
        forumOthersProfile: 'forum-others-profile.html',
        forumFollowing: 'forum-following.html',
        admin: 'admin.html',
        forumAdmin: 'forum-admin.html',
        forumReport: 'forum-report.html',
        forumMonitor: 'forum-monitor.html',
        auditLog: 'audit-log.html',
        forumStats: 'forum-stats.html',
        forumExport: 'export.html',
        sessions: 'sessions.html',
        blocks: 'blocks.html',
        announcements: 'announcements.html',
      },
    },
  },
});
