/*
 * 繁體中文訊息目錄（src/i18n/messages.ts）
 *
 * 這一檔是整個 i18n 的**型別基準**，不是「預設語言的備份」：
 * MessageKey 就是由它的鍵推導出來的聯集，其他語言的目錄必須是
 * `Record<MessageKey, string>`。因此漏翻一個字、或某個語言多了一個沒人用的
 * 鍵，都是編譯期錯誤 —— 不會有人到介面上才發現某顆按鈕還是中文。
 *
 * 新增一條文案的順序：
 *   1. 在這裡加一個鍵（繁體中文是原文，抄現有介面上的字串，不重新措辭）。
 *   2. 執行 npm run typecheck —— 會列出三個還沒翻的語言。
 *   3. 在 translations/{zh-CN,en,ja}.ts 各補一條。
 *
 * 命名規則：
 *   - 點分命名空間，前綴對應檔案：common / auth / feed / post / comment /
 *     report / newPost / profile / publicProfile / follow / following / login /
 *     admin / users / posts / reports / i18n / title。
 *   - 帶插值的值用 {name}，name 用底線小寫；值裡的單引號、雙引號都不會被
 *     當成佔位符，插值是整個字串取代（見 runtime.ts 的 interpolate）。
 *   - 同一句話在不同頁面語境下意思不同時才拆鍵（例如「發佈」既是按鈕也是
 *     確認框的按鈕，兩者共用 common.publish）；不要為了湊鍵而把同一句話複製
 *     成三個鍵 —— 那會讓三處措辭各自漂移。
 *
 * 站名不是譯文的一部分：
 *   含站名的 13 條文案寫成 {site}（完整站名）或 {brand}（短名），值來自設定檔
 *   的 FORUM_NAME / FORUM_SHORT_NAME（見 src/site.ts 與 README 的「站名」一節）。
 *   換站名因此不必動任何語言檔，也不會讓十七種語言各自的用詞漂移。
 *   {site} 由 usePageTitle 與 login.body 的呼叫端注入，{brand} 由後臺 rail 與
 *   登入閘門注入；兩種參數都不能省略 —— 少了就是畫面上殘留字面量，而
 *   tools/i18n/verify-catalogs.mjs 只會檢查各語言之間的佔位符是否一致。
 *
 * 兩條硬性約束：
 *   1. 這裡的繁體中文就是介面目前的原文。任何改動都是「改介面文案」而不是
 *      「加翻譯」，不該在這個檔案裡順便修字。
 *   2. 值的長度會影響版面：.btn 與 .status-pill 有 nowrap 的地方，英文與日文
 *      通常比中文長 1.5~2 倍，寬度不足時應該調整 CSS 而不是把文案縮短。
 */

/** 繁體中文原文。鍵的聯集就是 MessageKey。 */
export const zhTW = {
  /* ==========================================================================
     common
     ========================================================================== */

  'common.cancel': '取消',
  'common.save': '儲存',
  'common.submitting': '送出中...',
  'common.delete': '刪除',
  'common.edit': '編輯',
  'common.search': '搜尋',
  'common.loading': '載入中…',
  'common.loadFailed': '載入失敗',
  'common.refresh': '重新整理',
  'common.nextStep': '下一步',
  'common.prevPage': '上一頁',
  'common.nextPage': '下一頁',
  'common.create': '建立',
  'common.publish': '發佈',
  'common.placeholder': '—',
  'common.backToHome': '回到首頁',
  'common.backToForumHome': '回到論壇首頁',
  'common.backOnePage': '返回上一頁',

  /* ==========================================================================
     errors
     ==========================================================================
     這些是「後端沒給可讀訊息」時的本地文案。它們會被 errorMessage() 優先
     取用：後端真的回了一句話（例如「暱稱已存在」）時，顯示的是後端那句。
     ========================================================================== */

  'error.request': '請稍後再試。',
  'error.requestStatus': '請求失敗（HTTP {status}）',
  'error.loginRequired': '請先登入後再繼續。',
  'error.adminSessionExpired': '登入狀態已失效，即將返回登入頁。',

  /* requestJSON 的 fallback：回應不是 JSON 時顯示。 */
  'error.fallbackLoad': '載入失敗',
  'error.fallbackSearch': '搜尋失敗',
  'error.fallbackLike': '按讚失敗',
  'error.fallbackComments': '留言載入失敗',
  'error.fallbackCommentPost': '留言失敗',
  'error.fallbackReport': '檢舉失敗',
  'error.fallbackProfile': '讀取個人資料失敗',
  'error.fallbackProfileSave': '儲存失敗',
  'error.fallbackPublish': '發佈回應格式錯誤',
  'error.fallbackUpload': '圖片上傳回應格式錯誤',
  'error.fallbackNotFound': '找不到這個使用者',
  'error.fallbackFollow': '追蹤失敗',

  /* ==========================================================================
     auth（公開頁導覽列的登入狀態與登入按鈕）
     ========================================================================== */

  'auth.checking': '檢查登入狀態中...',
  'auth.statusUnknown': '登入狀態無法確認',
  'auth.feedLoggedIn': '已登入，可發文',
  'auth.feedLoggedOut': '登入後即可發文',
  'auth.profileLoggedIn': '已登入',
  'auth.profileLoggedOut': '登入後即可設定個人資料',
  'auth.googleLogin': 'Google 登入',
  'auth.loginWithGoogle': '使用 Google 帳號登入',
  'auth.loginWithGoogleAdmin': '使用 Google 管理員帳號登入',
  'auth.logout': '登出',

  /* ==========================================================================
     PWA 安裝
     ========================================================================== */

  'install.button': '安裝 App',
  'install.hint':
    '目前瀏覽器未提供自動安裝提示，請開啟瀏覽器選單，選擇「安裝應用程式」或「加入主畫面」。',

  /* ==========================================================================
     底部導覽
     ========================================================================== */

  'bottomNav.label': '主要導覽',
  'bottomNav.home': '首頁',
  'bottomNav.new': '新增',
  'bottomNav.profile': '個人',

  /* ==========================================================================
     i18n 切換器
     ========================================================================== */

  'i18n.ariaLabel': '選擇語言',
  'i18n.current': '語言：{name}',

  /* ==========================================================================
     feed：搜尋、狀態列、清單
     ========================================================================== */

  'feed.searchPlaceholder': '搜尋貼文內容',
  'feed.searchAriaLabel': '搜尋貼文',
  'feed.searchResultsLabel': '搜尋結果',
  'feed.postsLabel': '論壇文章',
  'feed.searchFailed': '搜尋失敗，請稍後再試。',
  'feed.searching': '搜尋中...',
  'feed.searchMore': '載入更多搜尋結果...',
  'feed.searchMoreFailed': '載入更多失敗',
  'feed.searchFound': '找到 {total} 筆',
  'feed.searchDegraded': '{base}（搜尋服務未啟用，目前以資料庫關鍵字比對）',
  'feed.searchTotal': '共 {total} 筆結果',
  'feed.searchNoResults': '找不到包含「{query}」的貼文。',
  'feed.loadingPosts': '正在載入文章...',
  'feed.loadMorePosts': '載入更多文章...',
  'feed.postsFailed': '文章載入失敗，請稍後再試。',
  'feed.postsFailedShort': '載入失敗，請稍後再試',
  'feed.scrollMore': '向下滑動載入更多',
  'feed.endOfFeed': '已經到底了',
  'feed.noPosts': '還沒有文章，先留下第一個想法吧。',
  'feed.likeFailed': '讚或收回讚失敗，請稍後再試。',

  /* ==========================================================================
     post：貼文卡
     ========================================================================== */

  'post.authorAnonymous': '匿',
  'post.report': '檢舉文章',
  'post.imageAlt': '貼文圖片',
  'post.unlike': '收回讚',
  'post.like': '讚',
  'post.reply': '回應',

  /* ==========================================================================
     comment：留言串
     ========================================================================== */

  'comment.loading': '留言載入中...',
  'comment.none': '尚無留言',
  'comment.loadFailed': '留言載入失敗，請稍後再試',
  'comment.placeholder': '寫下留言...',
  'comment.max': '最多 2000 字',
  'comment.submit': '留言',
  'comment.failed': '留言失敗，請稍後再試。',
  'comment.report': '檢舉',
  'comment.more': '載入更多留言...',

  /* ==========================================================================
     report：公開頁的檢舉表單
     ========================================================================== */

  'report.reasonPlaceholder': '請輸入檢舉原因（最多 500 字）',
  'report.note': '檢舉會送出給站方管理員',
  'report.formLabel': '檢舉輸入框',
  'report.submit': '送出檢舉',
  'report.failed': '檢舉失敗，請稍後再試。',
  'report.sent': '檢舉已送出，感謝你的回報。',

  /* ==========================================================================
     newPost：新增貼文頁
     ========================================================================== */

  'newPost.avatarYou': '你',
  'newPost.eyebrow': 'NEW POST',
  'newPost.title': '新增貼文',
  'newPost.loginFirst': '請先使用 Google 登入後發文',
  'newPost.contentPlaceholder': '分享你的想法...',
  'newPost.addImage': '新增圖片',
  'newPost.emailPrivate': '你的 email 不會公開',
  'newPost.submit': '發佈貼文',
  'newPost.publishing': '發佈中...',
  'newPost.uploading': '圖片上傳中...',
  'newPost.failed': '發佈失敗，請稍後再試。',

  /* ==========================================================================
     profile：自己的個人資料
     ========================================================================== */

  'profile.eyebrow': 'YOUR PROFILE',
  'profile.title': '個人資料',
  'profile.edit': '編輯',
  'profile.loginPrompt': '登入後即可設定你的論壇暱稱與個人簡介。',
  'profile.nicknameLabel': '論壇暱稱',
  'profile.notSet': '尚未設定',
  'profile.notSetBio': '尚未設定個人簡介。',
  'profile.nicknameInput': '暱稱',
  'profile.nicknamePlaceholder': '輸入暱稱',
  'profile.nicknameHint': '暱稱會顯示在你發布的文章上，最多 30 個字。',
  'profile.bioLabel': '個人簡介',
  'profile.bioPlaceholder': '介紹一下自己（選填）',
  'profile.bioHint': '最多 500 個字。',
  'profile.updated': '個人資料已更新。',
  'profile.saving': '儲存中...',
  'profile.saveFailed': '儲存失敗，請稍後再試。',
  'profile.loadFailed': '讀取失敗，請稍後再試。',
  'profile.followingEntry': '我的追蹤',

  /* ==========================================================================
     publicProfile：他人的公開個人頁
     ========================================================================== */

  'publicProfile.eyebrow': 'PUBLIC PROFILE',
  'publicProfile.title': '公開個人資料',
  'publicProfile.avatar': '匿',
  'publicProfile.loading': '載入中...',
  'publicProfile.invalidLinkName': '無效的公開個人頁連結',
  'publicProfile.invalidLinkBio': '請從論壇貼文中的作者名稱進入個人資料。',
  'publicProfile.notFound': '找不到這個使用者',
  'publicProfile.anonymous': '匿名使用者',
  'publicProfile.noBio': '這位使用者尚未設定公開資料。',
  'publicProfile.loadFailed': '公開資料載入失敗。',
  'publicProfile.postsLabel': '貼文',
  'publicProfile.emptyPosts': '這位使用者還沒有發表貼文。',

  /* ==========================================================================
     follow：追蹤鈕（三個頁面共用同一顆按鈕）
     ==========================================================================
     label 是 aria-label（未追蹤時的動作說明），action 是按鈕上顯示的文字。
     兩者刻意分開：actionDone（追蹤中）與 unfollow（取消追蹤）也一樣 ——
     視覺上顯示「追蹤中」讓按鈕保持穩定的寬度與位置，但讀螢幕軟體時
     「取消追蹤 X 的帳號」才是使用者需要聽到的資訊。
     ========================================================================== */

  'follow.label': '追蹤這位使用者',
  'follow.action': '追蹤',
  'follow.actionDone': '追蹤中',
  'follow.unfollow': '取消追蹤',
  'follow.done': '已追蹤。',
  'follow.failed': '追蹤失敗，請稍後再試。',

  /* ==========================================================================
     following：追蹤頁（/forum/following）
     ==========================================================================
     peopleLabel / postsLabel 同時是區塊標題與 section 的 aria-label，
     因此寫成「看得懂的片語」而不是「追蹤中的人數」這種帶計數的字串。
     ========================================================================== */

  'following.peopleLabel': '追蹤中的人',
  'following.postsLabel': '追蹤者的貼文',
  'following.peopleLoading': '載入追蹤清單中...',
  'following.emptyPeople': '你還沒有追蹤任何人。在貼文上按「追蹤」，或從他人的公開個人頁追蹤。',
  'following.emptyPosts': '追蹤的人還沒有發表貼文。',
  'following.peopleFailed': '追蹤清單載入失敗。',
  'following.postsFailed': '追蹤者的貼文載入失敗。',

  /* ==========================================================================
     login：登入閘門
     ========================================================================== */

  'login.eyebrow': 'MEMBER ACCESS',
  'login.title': '歡迎回來',
  'login.body': '{site} 是給所有會寫下想法的人的空間。不需要註冊表單 —— 一個 Google 帳號就能開始發文。',
  'login.browseFirst': '先看看首頁',

  /* ==========================================================================
     admin：後臺殼層與共用元件
     ========================================================================== */

  'admin.skipToMain': '跳至主要內容',
  'admin.railLabel': '管理選單',
  'admin.railBrandAria': '{site}首頁',
  'admin.consoleName': 'Admin Console',
  'admin.railNavLabel': '主要功能',
  'admin.railGovernance': '治理',
  'admin.railMode': '管理員模式',
  'admin.railExit': '回論壇',
  'admin.topbarMenu': '切換管理選單',
  'admin.statusOnline': '連線正常',
  'admin.topbarForum': '論壇',
  'admin.logoutFailed': '登出失敗，請稍後再試。',
  'admin.navUsers': '用戶管理',
  'admin.navPosts': '論壇文章',
  'admin.navReports': '檢舉管理',
  'admin.listLoadFailed': '載入失敗',
  'admin.dlgClose': '關閉視窗',
  'admin.dlgConfirm': '確認',
  'admin.dlgSave': '儲存',
  'admin.dlgApplyTags': '套用標籤',
  'admin.dlgNoTags': '目前沒有可套用的標籤，請先在下方「標籤管理」新增。',

  /* ==========================================================================
     users：用戶管理頁
     ========================================================================== */

  'users.title': '用戶管理',
  'users.contentAction': '內容',
  'users.updateContentFailed': '內容操作失敗。',
  'users.updated': '內容已更新。',
  'users.eyebrow': 'USER MANAGEMENT',
  'users.copy': '檢視論壇用戶的活動統計、標籤與帳號狀態，並處理停權與逐篇內容治理。',
  'users.refresh': '更新資料',
  'users.statTotal': '用戶總數',
  'users.statActive': '啟用中',
  'users.statSuspended': '已停權',
  'users.statContent': '文章／留言總量',
  'users.count': '{count} 位用戶',
  'users.tagsCount': '{count} 個標籤',
  'users.loadFailed': '用戶資料載入失敗。',
  'users.tagsLoadFailed': '標籤資料載入失敗。',
  'users.panelTitle': '論壇用戶',
  'users.tagsPanelTitle': '用戶標籤',
  'users.colUser': '用戶',
  'users.colTags': '標籤',
  'users.colStatus': '狀態',
  'users.colPosts': '文章',
  'users.colComments': '留言',
  'users.colLikes': '按讚',
  'users.colLastActivity': '最後活動',
  'users.colActions': '操作',
  'users.nicknameUnset': '未設定暱稱',
  'users.notSet': '未設定',
  'users.statusActive': '啟用中',
  'users.statusSuspended': '已停權',
  'users.emptyTitle': '目前沒有用戶資料',
  'users.emptyBody': '沒有用戶透過 Google 登入論壇時，這裡才會是空的。',
  'users.tagsEmptyTitle': '目前沒有標籤',
  'users.tagsEmptyBody': '先建立標籤，才能在用戶清單裡替對象套用。',
  'users.addTag': '新增標籤',
  'users.colName': '名稱',
  'users.colCreated': '建立時間',
  'users.colUpdated': '更新時間',
  'users.renameTag': '重新命名',
  'users.suspend': '停權',
  'users.restore': '恢復',
  'users.statusDialogTitle': '{action}此用戶',
  'users.statusSuspendMessage': '{email} 將無法再登入論壇，既有文章與留言會保留。要繼續嗎？',
  'users.statusRestoreMessage': '{email} 將恢復登入與發文權限。要繼續嗎？',
  'users.userSuspended': '用戶已停權。',
  'users.userRestored': '用戶已恢復。',
  'users.updateStatusFailed': '更新用戶狀態失敗。',
  'users.editTagsTitle': '編輯標籤 · {user}',
  'users.editTagsMessage': '勾選要套用的標籤；全部取消等同移除此用戶的所有標籤。',
  'users.tagsUpdated': '用戶標籤已更新。',
  'users.updateTagsFailed': '更新用戶標籤失敗。',
  'users.contentLoadFailed': '內容載入失敗。',
  'users.contentLoadFailedShort': '內容載入失敗。',
  'users.contentPanelTitle': '用戶內容',
  'users.contentCount': '{posts} 篇文章 · {comments} 則留言',
  'users.contentLoading': '正在載入文章與留言…',
  'users.addPost': '新增文章',
  'users.addComment': '新增留言',
  'users.postsColumn': '文章',
  'users.commentsColumn': '留言',
  'users.noPosts': '尚無文章',
  'users.noComments': '尚無留言',
  'users.postRef': '文章 #{id}',
  'users.editRecordTitle': '編輯{kind} #{id}',
  'users.deleteRecordTitle': '刪除{kind} #{id}',
  'users.deleteRecordMessage': '刪除後無法復原，相關的按讚與關聯資料也會一併移除。要繼續嗎？',
  'users.contentLabel': '內容',
  'users.addPostTitle': '新增文章',
  'users.addPostMessage': '此內容會以該用戶的身分發佈，作者欄位無法偽造。',
  'users.postContentLabel': '文章內容',
  'users.postContentPlaceholder': '輸入文章內容',
  'users.pickPostTitle': '選擇文章',
  'users.postIdLabel': '文章 ID',
  'users.postIdPlaceholder': '要留言的文章編號',
  'users.addCommentTitle': '新增留言',
  'users.commentContentLabel': '留言內容',
  'users.commentContentPlaceholder': '輸入留言內容',
  'users.createTagTitle': '新增標籤',
  'users.createTagMessage': '標籤可用於替用戶分類，例如「版主」「活躍」或「已封鎖」。',
  'users.tagNameLabel': '標籤名稱',
  'users.tagNamePlaceholder': '最多 50 個字',
  'users.renameTagTitle': '重新命名標籤',
  'users.renameTagMessage': '所有使用此標籤的用戶都會看到新名稱。',
  'users.deleteTagTitle': '刪除標籤「{name}」',
  'users.deleteTagMessage': '刪除後，所有用戶身上的此標籤都會一併移除，且無法復原。要繼續嗎？',
  'users.deleteTagConfirm': '刪除標籤',
  'users.tagCreated': '標籤已建立。',
  'users.createTagFailed': '建立標籤失敗。',
  'users.tagUpdated': '標籤已更新。',
  'users.updateTagFailed': '更新標籤失敗。',
  'users.tagDeleted': '標籤已刪除。',
  'users.deleteTagFailed': '刪除標籤失敗。',
  'users.refreshDone': '已更新至最新資料。',
  'users.signinEyebrow': '{brand} ADMIN',
  'users.signinTitle': '後臺管理',
  'users.signinBody': '集中式內容治理，讓每一次審查都清楚、快速且可追蹤。',
  'users.signinStep1': '安全驗證',
  'users.signinStep2': '用戶治理',
  'users.signinStep3': '內容審查',
  'users.signinPanelTitle': '登入管理控制台',
  'users.signinPanelBody': '管理控制台僅開放給已授權的 Google 管理員帳號，請使用管理員身分登入。',

  /* ==========================================================================
     posts：論壇文章頁
     ========================================================================== */

  'posts.title': '論壇文章',
  'posts.searching': '搜尋中…',
  'posts.searchDegraded': '（搜尋服務未啟用，以資料庫關鍵字比對）',
  'posts.searchSummary': '「{query}」找到 {total} 筆，本頁顯示 {shown} 筆',
  'posts.pendingCount': '{count} 筆待處理',
  'posts.pageSummary': '第 {page} / {pages} 頁，本頁 {count} 篇',
  'posts.listLoadFailed': '文章列表載入失敗。',
  'posts.eyebrow': 'POST MODERATION',
  'posts.copy': '建立、編輯與刪除論壇文章，並在留言層級處理內容治理。',
  'posts.toReports': '檢舉管理',
  'posts.editorTitleNew': '新增文章',
  'posts.editorTitleEdit': '編輯文章 #{id}',
  'posts.editorNote': '以管理員身分發佈；作者取自登入身分，請求內容無法偽造作者。',
  'posts.cancelEdit': '取消編輯',
  'posts.contentLabel': '文章內容',
  'posts.contentPlaceholder': '輸入文章內容',
  'posts.saveChanges': '儲存修改',
  'posts.emptyContent': '文章內容不可為空白。',
  'posts.saving': '儲存中…',
  'posts.saved': '文章已更新。',
  'posts.published': '文章已發佈。',
  'posts.saveFailed': '儲存失敗。',
  'posts.deleteTitle': '刪除文章 #{id}',
  'posts.deleteMessage': '刪除後無法復原，該文章底下的所有留言也會一併移除。要繼續嗎？',
  'posts.deleteConfirm': '刪除文章',
  'posts.deleted': '文章已刪除。',
  'posts.deleteFailed': '刪除文章失敗。',
  'posts.commentUpdated': '留言已更新。',
  'posts.commentActionFailed': '留言操作失敗。',
  'posts.pickPostTitle': '選擇留言所屬文章',
  'posts.pickPostMessage': '預設是這一列的文章；若要掛到別篇，請改成對的文章編號。',
  'posts.postIdLabel': '文章 ID',
  'posts.addCommentAtTitle': '在文章 #{id} 新增留言',
  'posts.addCommentMessage': '此留言會以管理員身分發佈。',
  'posts.commentContentLabel': '留言內容',
  'posts.commentContentPlaceholder': '輸入留言內容',
  'posts.add': '新增',
  'posts.editCommentTitle': '編輯留言 #{id}',
  'posts.deleteCommentTitle': '刪除留言 #{id}',
  'posts.deleteCommentMessage': '刪除後無法復原。要繼續嗎？',
  'posts.listTitle': '文章列表',
  'posts.clearSearch': '清除搜尋',
  'posts.searchLabel': '關鍵字搜尋',
  'posts.searchPlaceholder': '貼文內容，或貼文作者的完整 Email',
  'posts.searchHint': '依相關性排序；填入完整 Email 可找出該使用者的所有貼文。搜尋取代分頁，結果最多顯示 25 筆。',
  'posts.searchTotal': '搜尋結果共 {total} 筆',
  'posts.searchFailed': '搜尋失敗。',
  'posts.searchStatusFailed': '搜尋失敗',
  'posts.colContentImage': '內容與圖片',
  'posts.colEngagement': '互動',
  'posts.colComments': '留言',
  'posts.colAuthor': '作者',
  'posts.imageAlt': '文章附圖',
  'posts.likes': '{count} 按讚',
  'posts.author': '作者：{name}',
  'posts.emptyTitle': '目前沒有論壇文章',
  'posts.emptyBody': '可以先用上方的編輯器建立第一篇文章。',
  'posts.emptySearchTitle': '沒有符合的貼文',
  'posts.emptySearchBody': '「{query}」沒有比對到任何貼文，換個關鍵字試試。',
  'posts.commentCount': '{count} 則留言',
  'posts.noComments': '尚無留言',
  'posts.reportsTitle': '待處理檢舉',
  'posts.allReports': '全部檢舉',
  'posts.colReportedContent': '被檢舉內容',
  'posts.colReason': '檢舉原因',
  'posts.colReporter': '檢舉人',
  'posts.colTime': '時間',
  'posts.colVerdict': '裁定',
  'posts.emptyReportsTitle': '沒有待處理的檢舉',
  'posts.emptyReportsBody': '所有檢舉都已裁定完成。',
  'posts.verdictResolved': '已處理',
  'posts.verdictRejected': '不成立',
  'posts.verdictDialogTitle': '將檢舉 #{id} 標為「{label}」',
  'posts.verdictDialogMessage': '標記後檢舉會離開待處理清單，但資料仍保留在檢舉管理頁。要繼續嗎？',
  'posts.verdictConfirm': '標為{label}',
  'posts.verdictDone': '檢舉已標為{label}。',
  'posts.verdictFailed': '檢舉狀態更新失敗。',

  /* ==========================================================================
     reports：檢舉管理頁
     ========================================================================== */

  'reports.title': '檢舉管理',
  'reports.listSummary': '{count} 筆 · {filter}',
  'reports.listLoadFailed': '檢舉列表載入失敗。',
  'reports.eyebrow': 'REPORT MODERATION',
  'reports.copy': '逐筆裁定檢舉：「通過」會刪除被檢舉的內容，「不成立」則保留原文。改回「待處理」會清除既有裁定時間。',
  'reports.backToPosts': '回到文章',
  'reports.editorTitle': '編輯檢舉 #{id}',
  'reports.editorNote': '可補正檢舉原因與狀態；目標與檢舉人是既有紀錄，不在此更改。',
  'reports.targetTypeLabel': '目標類型',
  'reports.targetIdLabel': '目標 ID',
  'reports.reporterEmailLabel': '檢舉人 Email',
  'reports.statusLabel': '狀態',
  'reports.reasonLabel': '檢舉原因',
  'reports.reasonHint': '最多 500 字，會直接顯示給其他管理員作為裁定依據。',
  'reports.filterLabel': '依狀態篩選',
  'reports.filterAll': '全部',
  'reports.listTitle': '檢舉列表',
  'reports.colTarget': '目標內容',
  'reports.colReason': '檢舉原因',
  'reports.colReporter': '檢舉人',
  'reports.colStatus': '狀態',
  'reports.targetGone': '（內容已刪除）',
  'reports.author': '作者：{name}',
  'reports.deleteTitle': '刪除檢舉 #{id}',
  'reports.deleteMessage': '刪除的是這筆檢舉紀錄本身，被檢舉的內容不受影響，且無法復原。要繼續嗎？',
  'reports.deleteConfirm': '刪除檢舉',
  'reports.deleted': '檢舉已刪除。',
  'reports.deleteFailed': '刪除檢舉失敗。',
  'reports.updated': '檢舉已更新。',
  'reports.saveFailed': '儲存檢舉失敗。',
  'reports.approveTitle': '通過檢舉 #{id}',
  'reports.approveGoneMessage': '被檢舉的{kind} #{id} 已經不存在，這一筆只會被標為已處理。',
  'reports.approveMessage':
    '將永久刪除被檢舉的{kind} #{id}（若為文章，該文章底下的所有留言也一併移除），並把這筆檢舉標為已處理。要繼續嗎？',
  'reports.approveConfirm': '通過並刪文',
  'reports.approveGoneDone': '內容已不存在，檢舉已標為已處理。',
  'reports.approveDone': '已刪文並將檢舉標為已處理。',
  'reports.approveFailed': '通過檢舉失敗。',
  'reports.approveTitleGone': '內容已刪除，只會標記檢舉',
  'reports.approveTitleFull': '刪除被檢舉的內容並標為已處理',
  'reports.rejectTitle': '檢舉 #{id} 不成立',
  'reports.rejectMessage': '不成立代表被檢舉的內容無須處理，內容會原樣保留。要繼續嗎？',
  'reports.rejectConfirm': '標為不成立',
  'reports.rejectDone': '檢舉已標為不成立。',
  'reports.statusFailed': '檢舉狀態更新失敗。',
  'reports.emptyTitle': '目前沒有檢舉',
  'reports.emptyBody': '這個篩選條件下沒有任何紀錄。',
  'reports.rejectTitleAttr': '保留內容，只將檢舉標為不成立',
  'kind.post': '文章',
  'kind.comment': '留言',
  'reports.statusPending': '待處理',
  'reports.statusResolved': '已處理',
  'reports.statusRejected': '不成立',

  /* ==========================================================================
     title：分頁標題
     ==========================================================================
     靜態 HTML 的 <title> 只能是繁體中文（那是送出 HTML 時的內容，React 還沒
     跑）。這些鍵讓每個頁面在掛載後把標題換成目前語言，切換語言時也會跟著換。
     ========================================================================== */

  'title.forum': '{site}',
  'title.login': '登入｜{site}',
  'title.newPost': '新增貼文｜{site}',
  'title.profile': '個人資料｜{site}',
  'title.publicProfile': '公開個人資料｜{site}',
  'title.following': '追蹤｜{site}',
  'title.adminUsers': '用戶管理｜{brand} 後臺',
  'title.adminLogin': '登入｜{brand} 後臺',
  'title.adminPosts': '論壇文章｜{brand} 後臺',
  'title.adminReports': '檢舉管理｜{brand} 後臺',
} as const;

/**
 * 訊息鍵的聯集。
 *
 * 由 zhTW 推導，而不是手寫一份 union —— 手寫的版本一定會在某次新增訊息時
 * 忘了同步，而漏掉的鍵會在翻譯檔少一個欄位、編譯才發現。
 */
export type MessageKey = keyof typeof zhTW;

/** 佔位符的參數型別。值只允許字串或數字：數字會在插值時自動轉字串。 */
export type MessageParams = Readonly<Record<string, string | number>>;
