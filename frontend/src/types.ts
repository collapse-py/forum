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
