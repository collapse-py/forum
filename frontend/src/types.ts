/*
 * 前端可見的後端 API 契約（src/types.ts）
 *
 * 這些型別描述的是「這個前端的後端保證會回什麼」，集中放在一處的理由：
 * 公開頁與三支後臺頁面讀同一批資源（用戶、標籤、文章、留言、檢舉），各自重寫
 * 一份介面會在後端改欄位時出現多種不同的錯誤訊息，而其中兩種不會被編譯器抓到。
 *
 * 欄位一律寬鬆（索引簽章 + unknown）留給邊緣情境，但結構本身帶型別：
 * 拿到的 item 是有 id 與 content 的物件這件事，編譯器應該管。
 *
 * 公開頁（/forum/*）只用到 ItemsResponse；其餘是後臺三頁的契約。
 */

/** GET /api/check */
export interface CheckResponse {
  ok: boolean;
  isAdmin?: boolean;
}

/** 列表端點共用的 { items, hasMore } 包裝。 */
export interface ItemsResponse<T> {
  items: T[];
  hasMore?: boolean;
  [key: string]: unknown;
}

/** 分頁端點的 { items, pages } 包裝。 */
export interface PagedResponse<T> {
  items: T[];
  pages?: number;
  [key: string]: unknown;
}

/**
 * 搜尋端點的回應（GET /api/forum/search、GET /api/admin/forum/search）。
 *
 * 刻意沿用 items 這個鍵而不是 results：兩種來源的結果要能被同一段渲染
 * 程式處理，而渲染程式讀的就是 items。名稱不同等於強迫呼叫端多寫一次對應。
 *
 *	Total  符合條件的總筆數（ES 以 track_total_hits 取真實值，不是「還有沒有下一頁」）。
 *	Query  後端實際使用的關鍵字（已去除前後空白）。前端用它顯示「正在搜尋 X」，
 *	      避免輸入框與結果不同步時使用者以為搜錯了。
 *	Engine 這批結果的來源：'elasticsearch' 或 'mysql'。後者在 ES 未設定或故障時
 *	      出現，代表結果是資料庫的逐字比對而非全文檢索；前端據此顯示說明。
 */
export interface SearchResponse<T> {
  items: T[];
  total?: number;
  query?: string;
  engine?: 'elasticsearch' | 'mysql' | (string & {});
  [key: string]: unknown;
}

/* ==========================================================================
   標籤
   ========================================================================== */

export interface AdminTag {
  id: number;
  name: string;
  createdAt?: string;
  updatedAt?: string;
  [key: string]: unknown;
}

/* ==========================================================================
   用戶
   ========================================================================== */

export type UserStatus = 'ACTIVE' | 'SUSPENDED' | (string & {});

export interface AdminUser {
  email: string;
  nickname?: string;
  status?: UserStatus;
  postCount?: number;
  commentCount?: number;
  likeCount?: number;
  updatedAt?: string;
  tags?: AdminTag[];
  [key: string]: unknown;
}

/** GET /api/admin/users/:email/content */
export interface AdminUserPost {
  id: number;
  content: string;
  createdAt?: string;
  [key: string]: unknown;
}

export interface AdminUserComment {
  id: number;
  postId: number;
  content: string;
  createdAt?: string;
  [key: string]: unknown;
}

export interface AdminUserContent {
  posts: AdminUserPost[];
  comments: AdminUserComment[];
  [key: string]: unknown;
}

/* ==========================================================================
   文章與留言
   ========================================================================== */

export interface AdminComment {
  id: number;
  postId?: number;
  content: string;
  authorEmail?: string;
  createdAt?: string;
  [key: string]: unknown;
}

export interface AdminPost {
  id: number;
  content: string;
  imageUrl?: string;
  likeCount?: number;
  commentCount?: number;
  authorEmail?: string;
  createdAt?: string;
  /** 是否為管理員置頂。用它決定那一列的按鈕是「取消置頂」還是「置頂」。 */
  pinned?: boolean;
  comments?: AdminComment[];
  [key: string]: unknown;
}

/* ==========================================================================
   檢舉
   ========================================================================== */

export type ReportTargetType = 'post' | 'comment' | (string & {});
export type ReportStatus = 'PENDING' | 'RESOLVED' | 'REJECTED' | (string & {});

export interface AdminReport {
  id: number;
  targetType: ReportTargetType;
  targetId: number;
  targetContent?: string;
  targetAuthor?: string;
  reason: string;
  reporterEmail: string;
  status: ReportStatus;
  createdAt?: string;
  [key: string]: unknown;
}

/* ==========================================================================
   監控
   ========================================================================== */

/**
 * GET /api/admin/monitor 的回應。
 *
 * 整個型別分成三塊，界線是「誰需要 I/O」：
 *
 *   - Stats  純記憶體，由 metrics 套件的快照組成。欄位名稱刻意與後端的
 *            JSON 標籤一致（駝峰），因此這個介面同時就是「後端欄位改名時
 *            編譯期會報錯」的那個東西。
 *   - Dependencies  每次讀取都會真的送出 ping，因此帶 latencyMs 與可能為
 *            空的 detail。state 有三種而不是兩種：disabled 不是健康的其中
 *            一種，而是根本沒啟用，維運上必須能與「查得到但連不上」分開。
 *   - Timeline 每格帶 source，用來標示這一分鐘是本次啟動的即時資料還是從
 *            資料庫讀回的重啟前紀錄。兩者的可信度不同，混在一起會讓「重啟
 *            之後數字變得很小」被誤讀成「服務沒有流量」。
 *
 * 延遲相關的欄位一律命名 MS／Ms：它們是毫秒，不是秒。p50/p95/p99 是延遲
 * 直方圖的「桶上界」而不是精確值（見 backend/forum/metrics 的說明）。
 */
export interface MonitorResponse {
  ok?: boolean;
  now?: string;
  forum?: string;
  probesMs?: number;
  stats: MonitorSnapshot;
  dependencies: Record<'mysql' | 'redis' | 'search', MonitorDependency>;
}

export type MonitorDependencyState = 'ok' | 'down' | 'disabled' | (string & {});

/** 單一依賴的探測結果。detail 的形狀因依賴而異，因此是索引簽章。 */
export interface MonitorDependency {
  state: MonitorDependencyState;
  latencyMs: number;
  error?: string;
  detail?: Record<string, unknown>;
}

export interface MonitorSnapshot {
  startedAt?: string;
  now?: string;
  uptimeSeconds?: number;
  windowMinutes?: number;
  retentionHours?: number;
  maxRoutes?: number;
  runtime: MonitorRuntime;
  requests: MonitorRequests;
  timeline: MonitorTimelinePoint[];
  rateLimits: MonitorRateLimit[];
  historyLoaded?: boolean;
}

/** Go 執行期的使用量。記憶體數值是位元組，不是 KiB。 */
export interface MonitorRuntime {
  version: string;
  goroutines: number;
  numCpu: number;
  gomaxprocs: number;
  gcCycles: number;
  allocBytes: number;
  sysBytes: number;
  heapAllocBytes: number;
  heapInUseBytes: number;
  heapObjects: number;
  stackInUseBytes: number;
  lastGcPauseMs: number;
}

export interface MonitorRequests {
  total: number;
  clientErrors: number;
  serverErrors: number;
  inFlight: number;
  maxInFlight: number;
  avgDurationMs: number;
  p50Ms: number;
  p95Ms: number;
  p99Ms: number;
  maxMs: number;
  routes: MonitorRoute[];
}

export interface MonitorRoute {
  method: string;
  /** 已正規化的路由樣式（數字與 email 換成 :id）；上限用盡時為 "__other__"。 */
  route: string;
  total: number;
  clientErrors: number;
  serverErrors: number;
  avgMs: number;
  p50Ms: number;
  p95Ms: number;
  maxMs: number;
}

export interface MonitorTimelinePoint {
  /** 分鐘起點，UTC 的 RFC 3339。 */
  minute: string;
  total: number;
  clientErrors: number;
  serverErrors: number;
  avgDurationMs: number;
  /** "live" = 本次啟動、"history" = 資料庫讀回、空字串 = 該分鐘沒有資料。 */
  source: string;
}

export interface MonitorRateLimit {
  /** "content" / "upload" / "auth"；後端只會回這三個。 */
  name: string;
  limit: number;
  windowSeconds: number;
  allowed: number;
  blocked: number;
  trackedKeys: number;
}

/* ==========================================================================
   IP 封鎖名單
   ========================================================================== */

/**
 * GET /api/admin/blocks 的回應。
 *
 * `available` 為 false 代表這台站沒有接 Redis，因此沒有封鎖功能。介面要說明
 * 那件事，而不是假裝名單是空的 —— 空清單會讓管理員以為「沒有人被封」。
 */
export interface AdminBlocksResponse {
  ok?: boolean;
  items: AdminBlockView[];
  available: boolean;
}

export interface AdminBlockView {
  ip: string;
  /** 到期時間（RFC 3339）。 */
  expiresAt: string;
  /** 剩餘秒數。與 expiresAt 一起給，前端不必自己做時區換算。 */
  remainingSeconds: number;
}

/** POST /api/admin/blocks 的回應（封鎖）。 */
export interface AdminBlockAddResponse {
  ok?: boolean;
  ip: string;
  expiresAt: string;
}

/**
 * POST /api/admin/blocks 的回應（解封，minutes <= 0）。
 *
 * `existed` 為 false 代表那個 IP 本來就沒被封 —— 那不是錯誤，介面應該說
 * 「它本來就沒被封」而不是「解封失敗」。
 */
export interface AdminBlockRemoveResponse {
  ok?: boolean;
  ip: string;
  existed: boolean;
}

/* ==========================================================================
   Session 管理
   ========================================================================== */

/** GET /api/admin/sessions 的回應。 */
export interface AdminSessionsResponse {
  ok?: boolean;
  items: AdminSessionView[];
  /** 掃描期間數到的 session 總數（可能大於 items，差異來自篩選與 limit）。 */
  totalActive: number;
  /** 實際檢查過的 key 數。 */
  scanned: number;
  /**
   * 為 true 代表掃描達到 key 數上限而提前放棄 —— `items` **不完整**。
   *
   * 這個欄位存在的理由：「沒列出來」若沒有被明確標成「沒掃完」，看起來就會
   * 像「這個帳號只有這些 session」，而那是一個不實的結論。
   */
  truncated: boolean;
  /** Session 的存續時間（小時）。配合滑動續期即「連續 N 小時沒活動才登出」。 */
  expireInHours: number;
}

/**
 * 一支 session 對外的樣子。
 *
 * 刻意沒有 token 欄位：token 就是憑證本身。`tokenPrefix` 是前 8 個字元，
 * 足以讓管理員分辨「是不是同一支」，不足以還原。
 */
export interface AdminSessionView {
  email: string;
  isAdmin: boolean;
  tokenPrefix: string;
  /** 建立時間（RFC 3339）。空字串代表「既有 session，沒有記錄」。 */
  createdAt: string;
  /** 真正會失效的時間點（now + TTL）。空字串代表查不到。 */
  expiresAt: string;
  /** 剩餘秒數；負值代表 Redis 沒有回報 TTL。 */
  ttlSeconds: number;
}

/** POST /api/admin/sessions/revoke 的回應。 */
export interface AdminSessionRevokeResponse {
  ok?: boolean;
  email: string;
  revoked: number;
  scanned: number;
}

/* ==========================================================================
   站內公告
   ========================================================================== */

/**
 * GET /api/forum/announcement 的回應。
 *
 * `announcement` 為 null 是**壓倒性的常見情況**（沒有公告），因此端點回 200
 * 而不是 404：前端每個頁面載入都會問一次，若「沒有公告」是 404，前端就得
 * 區分「正常的沒有」與「端點壞了」兩種 404。
 */
export interface ForumAnnouncementResponse {
  ok?: boolean;
  announcement: ForumAnnouncement | null;
}

/**
 * 一則生效中的公告。
 *
 * `body` 是管理員寫的**純文字**，不經過翻譯 —— 翻譯它需要一個翻譯資料庫，
 * 而這個專案的多語系是介面層的（見 src/i18n）。圍著它的那些文字（標題、
 * 關閉鈕、發佈時間）才走 i18n。
 *
 * 新��以 \n 分隔（本站沒有富文字編輯器，textarea 的換行會原樣送出）。
 */
export interface ForumAnnouncement {
  body: string;
  publishedAt: string;
  /** 空字串代表永不自動過期。 */
  expiresAt: string;
}

/** GET /api/admin/announcements 的回應。 */
export interface AdminAnnouncementsResponse {
  ok?: boolean;
  items: AdminAnnouncementView[];
}

export interface AdminAnnouncementView {
  id: number;
  body: string;
  /** 後臺的開關狀態。它可能與 effective 不同（見下）。 */
  active: boolean;
  /**
   * 實際上會不會顯示在公開頁上。
   *
   * 刻意與 `active` 分開：一則 active 但已過期的公告在後臺看起來是「開著的」，
   * 而它實際上什麼都不顯示。合成一個欄位會讓管理員以為橫幅還在。
   */
  effective: boolean;
  createdBy: string;
  createdAt: string;
  updatedBy?: string;
  updatedAt?: string;
  /** 空字串代表永不自動過期。 */
  expiresAt?: string;
}

/* ==========================================================================
   匯出與批次操作
   ========================================================================== */

/** 匯出的三種資源。對應後端的三條 CSV 路由。 */
export type ExportKind = 'users' | 'posts' | 'reports';

/**
 * 批次操作的回應。
 *
 * `counts` 刻意是一個字典而不是固定欄位（`updated` / `unchanged`）：不同操作
 * 有不同的計數維度，而讓兩支端點共用一個型別比加一個永遠是 0 的欄位乾淨。
 *
 * `skipped` 逐項帶原因而不是只給總數 —— 管理員需要知道跳過的是「帳號不存在」
 * 還是「格式錯誤」，兩者的下一步完全不同。
 */
export interface BatchResult {
  ok?: boolean;
  /** 送來的 email 數（去重之後）。與 counts 之和不一定要相等。 */
  requested: number;
  counts: Record<string, number>;
  skipped: { email: string; reason: string }[];
}

/* ==========================================================================
   內容趨勢統計
   ========================================================================== */

/**
 * GET /api/admin/stats 的回應。
 *
 * 這是「站正在長成什麼樣子」的資料，與既有的三個後臺頁互補：那三頁都是
 * 「現在是什麼」（總數、清單、待裁決的佇列），這裡是「變成這樣多久了」。
 *
 * `series` 的三個數字陣列與 `dates` **等長且同序**：連沒有資料的日子都在
 * 陣列裡（值為 0）。刻意不做成稀疏 —��� 圖上缺一格和「那天真的是零」是兩件
 * 不同的事，而稀疏的表示會讓前者看起來像後者。
 */
export interface ContentStatsResponse {
  ok?: boolean;
  now?: string;
  forum?: string;
  /** 視窗長度（天）。實際生效的值 —— 請求超過 90 天會被收斂到 90。 */
  days: number;
  series: ContentStatsSeries;
  /** 視窗內的合計，不是目前的總數。 */
  totals: ContentStatsTotals;
  topPosts: ContentStatsPost[];
  topTags: ContentStatsTag[];
  topAuthors: ContentStatsAuthor[];
}

export interface ContentStatsSeries {
  /** "YYYY-MM-DD"，由舊到新。 */
  dates: string[];
  users: number[];
  posts: number[];
  comments: number[];
}

export interface ContentStatsTotals {
  users: number;
  posts: number;
  comments: number;
  /** 視窗內**新按**的讚，與 topPosts 裡的累計讚數不同義。 */
  likes: number;
}

export interface ContentStatsPost {
  id: number;
  authorEmail: string;
  /** 貼文前 80 字（後端截斷，附省略號）。 */
  excerpt: string;
  createdAt: string;
  comments: number;
  likes: number;
}

export interface ContentStatsTag {
  id: number;
  name: string;
  /** 綁定這個標籤的人數。 */
  users: number;
}

export interface ContentStatsAuthor {
  email: string;
  posts: number;
  comments: number;
}

/* ==========================================================================
   管理員操作稽核紀錄
   ========================================================================== */

/**
 * GET /api/admin/log 的回應。
 *
 * `items` 的每筆是一條「誰對哪個對象做了什麼」；`total` 是符合篩選的總筆數
 * （不受分頁影響），前端用它算分頁與顯示「第 x-y 筆，共 n 筆」。
 *
 * `actions` 與 `actors` 是**從資料庫實際出現過的值**取出來的清單，不是前端
 * 寫死的。理由見 MonitorPage 對 timeline source 的同樣考量：寫死會讓新增
 * 的動作或管理員在篩選器裡選不到，而「選不到」看起來像「沒發生過」——
 * 那正是稽核紀錄最不能被誤解的地方。
 */
export interface AdminActionLogResponse {
  ok?: boolean;
  items: AdminActionEntry[];
  total: number;
  pageSize: number;
  actions: string[];
  actors: string[];
}

/** 稽核紀錄的資源類別。與後端 audit.TargetType 一致。 */
export type AdminActionTarget = 'user' | 'post' | 'comment' | 'report' | 'tag' | (string & {});

/**
 * 一筆稽核紀錄。
 *
 * `changes` 是欄位級 diff。刻意保留「改動前」與「改動後」兩個欄位而不是
 * 只記最終狀態：資料庫裡永遠只有「現在是什麼」，而稽核紀錄的全部價值就在
 * 於那個「原本是什麼」。
 */
export interface AdminActionEntry {
  id: number;
  /** 操作者的管理員 email。 */
  actorEmail: string;
  /** 機器可讀的動作名稱，例如 "post.delete"。 */
  action: string;
  targetType: AdminActionTarget;
  /** 對象識別值：user 與 target label 用 email，其餘用數字字串。 */
  targetId: string;
  /** 給人看的對象摘要（貼文內容前綴、標籤名…）。 */
  targetLabel?: string;
  changes?: AdminActionChange[];
  /** 操作來源的 IP 與 request ID；可用來和存取日誌對照。 */
  clientIp?: string;
  requestId?: string;
  /** UTC 的 "YYYY-MM-DD HH:mm:ss"，由後端固定時區後傳出字串。 */
  createdAt: string;
}

export interface AdminActionChange {
  field: string;
  /** 空字串代表「原本不存在這個值」或「已被清空」—— 兩者在介面上分開呈現。 */
  before: string;
  after: string;
  /** 值被截到 200 字元（後端行為）。 */
  truncated?: boolean;
}

/* ==========================================================================
   論壇公開頁
   ========================================================================== */

/**
 * 公開頁的貼文。欄位刻意比 AdminPost 少：公開 API 不回 authorEmail，
 * 只回 publicForumAuthor 產生的化名與 publicForumKey 產生的金鑰。
 * 兩者都不可用來還原 email —— 前端不該、也做不到拿 email 當識別依據。
 */
export interface ForumPost {
  id: number;
  author?: string;
  authorKey?: string;
  authorTags?: string[];
  content: string;
  createdAt?: string;
  imageUrl?: string;
  liked?: boolean;
  /** 管理員是否置頂這一篇。省略未置頂的（見後端 forumPost 的說明）。 */
  pinned?: boolean;
  likeCount?: number;
  commentCount?: number;
  [key: string]: unknown;
}

export interface ForumComment {
  id: number;
  author?: string;
  authorKey?: string;
  content: string;
  createdAt?: string;
  [key: string]: unknown;
}

/** GET /api/forum/profile、GET /api/forum/public-profile */
export interface ForumProfile {
  nickname?: string;
  bio?: string;
  /**
   * 只有 GET /api/forum/public-profile 會帶這個欄位，值是「目前的請求者是否
   * 追蹤了這個人」。
   *
   * 三種狀態刻意區分開來，不是一個 undefined 就是 false：
   *   - key 為 undefined → 這個金鑰沒有對應的公開個人資料，不存在可追蹤的對象
   *     （後端在查無此人時刻意不放這個欄位），前端因此不渲染追蹤鈕
   *   - false           → 有對象，但沒追蹤（或尚未登入）
   *   - true            → 有對象且已追蹤
   *
   * 它代表請求者的狀態，不是這位使用者的追蹤者數 —— 追蹤關係完全私有，
   * 沒有任何端點回傳「誰追蹤了誰」。
   */
  following?: boolean;
}

/* ==========================================================================
   追蹤
   ========================================================================== */

/**
 * GET /api/forum/follows 的單一對象。
 *
 * Key 是被追蹤者的 publicForumKey（email 的 SHA-256），也是 PostCard 上
 * authorKey 的同一個值，因此兩處可以用字串直接比對，不需要任何對照表。
 * Nickname 可能是空字串（對方從未設定暱稱），此時顯示 publicProfile.anonymous；
 * 後端刻意不代填匿名代號，因為那會在每次重新載入時換一個名字。
 */
export interface FollowTarget {
  key: string;
  nickname: string;
  followedAt?: string;
  [key: string]: unknown;
}

/** GET /api/forum/follows */
export interface FollowListResponse extends ItemsResponse<FollowTarget> {
  /**
   * 我自己的 publicKey。前端用它做兩件事：貼文卡上遇到自己的文章時不渲染
   * 追蹤鈕，以及個人頁判斷「這個對象是不是我」。由這支端點順便回傳而不動
   * /api/check，是因為 /api/check 是所有殼層共用的端點。
   */
  selfKey?: string;
}

/** POST /api/forum/follows */
export interface FollowToggleResponse {
  key: string;
  /** 切換「後」的狀態。連點兩下的合併結果只有後端知道，前端不該自己猜。 */
  following: boolean;
}

/** POST /api/forum/images */
export interface UploadResponse {
  url?: string;
  message?: string;
}

/** POST /api/forum/posts/:id/like */
export interface LikeResponse {
  liked?: boolean;
  count?: number;
}

/** POST /api/forum/posts/:id/comments */
export interface CommentResponse {
  item: ForumComment;
}
