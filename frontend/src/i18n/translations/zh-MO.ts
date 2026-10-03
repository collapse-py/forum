/*
 * zh-MO catalog (src/i18n/translations/zh-MO.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const zhMO: Record<MessageKey, string> = {
  'common.cancel':
    '取消',
  'common.save':
    '儲存',
  'common.submitting':
    '送出中...',
  'common.delete':
    '刪除',
  'common.edit':
    '編輯',
  'common.search':
    '搜尋',
  'common.loading':
    '載入中…',
  'common.loadFailed':
    '載入失敗',
  'common.refresh':
    '重新載入',
  'common.nextStep':
    '下一步',
  'common.prevPage':
    '上一頁',
  'common.nextPage':
    '下一頁',
  'common.create':
    '建立',
  'common.publish':
    '發佈',
  'common.placeholder':
    '—',
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 小時',
  'common.oneDay': '1 天',
  'common.sevenDays': '7 天',
  'common.thirtyDays': '30 天',
  'common.oneYear': '1 年',
  'common.backToHome':
    '回到首頁',
  'common.backToForumHome':
    '回到論壇首頁',
  'common.backOnePage':
    '返回上一頁',
  'error.request':
    '請稍後再試。',
  'error.requestStatus':
    '請求失敗（HTTP {status}）',
  'error.loginRequired':
    '請先登入後再繼續。',
  'error.adminSessionExpired':
    '登入狀態已失效，即將返回登入頁。',
  'error.fallbackLoad':
    '載入失敗',
  'error.fallbackSearch':
    '搜尋失敗',
  'error.fallbackLike':
    '讚好失敗',
  'error.fallbackComments':
    '留言載入失敗',
  'error.fallbackCommentPost':
    '留言失敗',
  'error.fallbackReport':
    '舉報失敗',
  'error.fallbackProfile':
    '讀取個人資料失敗',
  'error.fallbackProfileSave':
    '儲存失敗',
  'error.fallbackPublish':
    '發佈回應格式錯誤',
  'error.fallbackUpload':
    '圖片上載回應格式錯誤',
  'error.fallbackNotFound':
    '找不到這個使用者',
  'error.fallbackFollow':
    '追蹤失敗',
  'auth.checking':
    '檢查登入狀態中...',
  'auth.statusUnknown':
    '登入狀態無法確認',
  'auth.feedLoggedIn':
    '已登入，可發文',
  'auth.feedLoggedOut':
    '登入後即可發文',
  'auth.profileLoggedIn':
    '已登入',
  'auth.profileLoggedOut':
    '登入後即可設定個人資料',
  'auth.googleLogin':
    'Google 登入',
  'auth.loginWithGoogle':
    '使用 Google 帳戶登入',
  'auth.loginWithGoogleAdmin':
    '使用 Google 管理員帳戶登入',
  'auth.logout':
    '登出',
  'install.button':
    '安裝程式',
  'install.hint':
    '目前瀏覽器未提供自動安裝提示，請開啟瀏覽器選單，選擇「安裝程式」或「加入主畫面」。',
  'bottomNav.label':
    '主要導覽',
  'bottomNav.home':
    '首頁',
  'bottomNav.new':
    '新增',
  'bottomNav.profile':
    '個人',
  'i18n.ariaLabel':
    '選擇語言',
  'i18n.current':
    '語言：{name}',
  'feed.searchPlaceholder':
    '搜尋帖文內容',
  'feed.searchAriaLabel':
    '搜尋帖文',
  'feed.searchResultsLabel':
    '搜尋結果',
  'feed.postsLabel':
    '論壇文章',
  'feed.searchFailed':
    '搜尋失敗，請稍後再試。',
  'feed.searching':
    '搜尋中...',
  'feed.searchMore':
    '載入更多搜尋結果...',
  'feed.searchMoreFailed':
    '載入更多失敗',
  'feed.searchFound':
    '找到 {total} 筆',
  'feed.searchDegraded':
    '{base}（搜尋服務未啟用，目前以資料庫關鍵字比對）',
  'feed.searchTotal':
    '共 {total} 筆結果',
  'feed.searchNoResults':
    '找不到包含「{query}」的帖文。',
  'feed.loadingPosts':
    '正在載入文章...',
  'feed.loadMorePosts':
    '載入更多文章...',
  'feed.postsFailed':
    '文章載入失敗，請稍後再試。',
  'feed.postsFailedShort':
    '載入失敗，請稍後再試',
  'feed.scrollMore':
    '向下滑動載入更多',
  'feed.endOfFeed':
    '已經到底了',
  'feed.noPosts':
    '還沒有文章，先留下第一個想法吧。',
  'feed.likeFailed':
    '讚好或取消讚好失敗，請稍後再試。',
  'post.authorAnonymous':
    '匿',
  'post.report':
    '舉報帖文',
  'post.imageAlt':
    '帖文圖片',
  'post.unlike':
    '取消讚好',
  'post.like':
    '讚好',
  'post.reply':
    '回應',
  'comment.loading':
    '留言載入中...',
  'comment.none':
    '尚無留言',
  'comment.loadFailed':
    '留言載入失敗，請稍後再試',
  'comment.placeholder':
    '寫下留言...',
  'comment.max':
    '最多 2000 字',
  'comment.submit':
    '留言',
  'comment.failed':
    '留言失敗，請稍後再試。',
  'comment.report':
    '舉報',
  'comment.more':
    '載入更多留言...',
  'report.reasonPlaceholder':
    '請輸入舉報原因（最多 500 字）',
  'report.note':
    '舉報會送出給站方管理員',
  'report.formLabel':
    '舉報輸入框',
  'report.submit':
    '送出舉報',
  'report.failed':
    '舉報失敗，請稍後再試。',
  'report.sent':
    '舉報已送出，感謝你的回報。',
  'newPost.avatarYou':
    '你',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    '新增帖文',
  'newPost.loginFirst':
    '請先使用 Google 登入後發文',
  'newPost.contentPlaceholder':
    '分享你的想法...',
  'newPost.addImage':
    '新增圖片',
  'newPost.emailPrivate':
    '你的 email 不會公開',
  'newPost.submit':
    '發佈帖文',
  'newPost.publishing':
    '發佈中...',
  'newPost.uploading':
    '圖片上載中...',
  'newPost.failed':
    '發佈失敗，請稍後再試。',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    '個人資料',
  'profile.edit':
    '編輯',
  'profile.loginPrompt':
    '登入後即可設定你的論壇暱稱與個人簡介。',
  'profile.nicknameLabel':
    '論壇暱稱',
  'profile.notSet':
    '尚未設定',
  'profile.notSetBio':
    '尚未設定個人簡介。',
  'profile.nicknameInput':
    '暱稱',
  'profile.nicknamePlaceholder':
    '輸入暱稱',
  'profile.nicknameHint':
    '暱稱會顯示在你發布的帖文上，最多 30 個字。',
  'profile.bioLabel':
    '個人簡介',
  'profile.bioPlaceholder':
    '介紹一下自己（選填）',
  'profile.bioHint':
    '最多 500 個字。',
  'profile.updated':
    '個人資料已更新。',
  'profile.saving':
    '儲存中...',
  'profile.saveFailed':
    '儲存失敗，請稍後再試。',
  'profile.loadFailed':
    '讀取失敗，請稍後再試。',
  'profile.followingEntry':
    '我的追蹤',
  'profile.postsLabel':
    '我的貼文',
  'profile.emptyPosts':
    '你還沒有發表貼文。',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    '公開個人資料',
  'publicProfile.avatar':
    '匿',
  'publicProfile.loading':
    '載入中...',
  'publicProfile.invalidLinkName':
    '無效的公開個人頁連結',
  'publicProfile.invalidLinkBio':
    '請從論壇帖文中的作者名稱進入個人資料。',
  'publicProfile.notFound':
    '找不到這個使用者',
  'publicProfile.anonymous':
    '匿名使用者',
  'publicProfile.noBio':
    '這位使用者尚未設定公開資料。',
  'publicProfile.loadFailed':
    '公開資料載入失敗。',
  'publicProfile.postsLabel':
    '貼文',
  'publicProfile.emptyPosts':
    '這位使用者還沒有發表貼文。',

  'follow.label':
    '追蹤這位使用者',
  'follow.action':
    '追蹤',
  'follow.actionDone':
    '追蹤中',
  'follow.unfollow':
    '取消追蹤',
  'follow.done':
    '已追蹤。',
  'follow.failed':
    '追蹤失敗，請稍後再試。',

  'following.peopleLabel':
    '追蹤中的人',
  'following.postsLabel':
    '追蹤者的貼文',
  'following.peopleLoading':
    '載入追蹤清單中...',
  'following.emptyPeople':
    '你還沒有追蹤任何人。在貼文上按「追蹤」，或從他人的公開個人頁追蹤。',
  'following.emptyPosts':
    '追蹤的人還沒有發表貼文。',
  'following.peopleFailed':
    '追蹤清單載入失敗。',
  'following.postsFailed':
    '追蹤者的貼文載入失敗。',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    '歡迎回來',
  'login.body':
    '{site} 是給所有會寫下想法的人的空間。不需要註冊表單 —— 一個 Google 帳戶就能開始發文。',
  'login.browseFirst':
    '先看看首頁',
  'admin.skipToMain':
    '跳至主要內容',
  'admin.railLabel':
    '管理選單',
  'admin.railBrandAria':
    '{site}首頁',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    '主要功能',
  'admin.railGovernance':
    '治理',
  'admin.railMode':
    '管理員模式',
  'admin.railExit':
    '回論壇',
  'admin.topbarMenu':
    '切換管理選單',
  'admin.statusOnline':
    '連線正常',
  'admin.topbarForum':
    '論壇',
  'admin.logoutFailed':
    '登出失敗，請稍後再試。',
  'admin.navUsers':
    '用戶管理',
  'admin.navPosts':
    '論壇文章',
  'admin.navReports':
    '舉報管理',
  'admin.navMonitor': '系統監控',
  'admin.navLog': '操作紀錄',
  'admin.navStats': '內容趨勢',
  'admin.navExport': '匯出與批次',
  'admin.navSessions': '登入與 Session',
  'admin.navBlocks': 'IP 封鎖',
  'admin.navAnnouncements': '站內公告',
  'admin.listLoadFailed':
    '載入失敗',
  'admin.dlgClose':
    '關閉視窗',
  'admin.dlgConfirm':
    '確認',
  'admin.dlgSave':
    '儲存',
  'admin.dlgApplyTags':
    '套用標籤',
  'admin.dlgNoTags':
    '目前沒有可套用的標籤，請先在下方「標籤管理」新增。',

  'monitor.title': '系統監控',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': '即時查看服務狀態、請求流量、延遲分佈與限流器計數。',
  'monitor.refresh': '重新整理',
  'monitor.refreshing': '讀取中…',
  'monitor.autoRefresh': '自動更新',
  'monitor.autoRefreshOn': '每 {seconds} 秒自動更新',
  'monitor.autoRefreshOff': '自動更新已暫停',
  'monitor.nextUpdate': '{seconds} 秒後更新',
  'monitor.loadFailed': '監控資料載入失敗。',
  'monitor.loadFailedHint': '請確認已登入管理員帳號，且後端服務仍在運行。',
  'monitor.pausedHint': '自動更新已暫停，畫面顯示的是最後一次讀取的結果。',
  'monitor.visibilityPaused': '分頁在背景中，自動更新已暫停。',
  'monitor.lastUpdated': '更新於 {time}',
  'monitor.probeTook': '依賴探測耗時 {ms} 毫秒',
  'monitor.unreachable': '後端沒有回應。畫面停留在最後一次成功讀取的資料。',
  'monitor.depsTitle': '服務狀態',
  'monitor.depsNote': '每次讀取都會實際探測一次；單一依賴逾時為 2 秒，三者並行執行。',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': '搜尋引擎',
  'monitor.stateOk': '正常',
  'monitor.stateDown': '無法連線',
  'monitor.stateDisabled': '未啟用',
  'monitor.depSearchFallback': '未設定 ES_URL，搜尋使用 MySQL 關鍵字比對。',
  'monitor.depDisabled': '未注入 Redis 用戶端，媒體功能未啟用。',
  'monitor.depLatency': '回應 {ms} 毫秒',
  'monitor.depKeys': '{count} 個 key',
  'monitor.depMemory': '記憶體 {size}',
  'monitor.depPoolUsage': '連線 {inUse}／{open}（上限 {max}）',
  'monitor.depPoolWait': '等待 {count} 次，共 {ms} 毫秒',
  'monitor.depRedisPool': '命中 {hits}／未命中 {misses}',
  'monitor.depEngineMysql': 'MySQL 關鍵字比對',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': '請求總覽',
  'monitor.statUptime': '運行時間',
  'monitor.statRequests': '請求總數',
  'monitor.statErrorRate': '錯誤率',
  'monitor.statP95': 'P95 延遲',
  'monitor.statInFlight': '進行中請求',
  'monitor.statGoroutines': 'Goroutine',
  'monitor.statHeap': '堆積記憶體',
  'monitor.statDbPool': '資料庫連線',
  'monitor.statRateLimited': '限流阻擋',
  'monitor.statCountWithPeak': '峰值 {peak}',
  'monitor.statCountWithInUse': '使用中 {inUse}，閒置 {idle}',
  'monitor.statBlockedSplit': '4xx {client}／5xx {server}',
  'monitor.goVersion': 'Go {version}・{cpu} 邏輯核心・GC {gc} 次',
  'monitor.noData': '尚未收到請求。',
  'monitor.noDataBody': '服務啟動後的請求會出現在這裡；目前這一欄是空的。',
  'monitor.timelineTitle': '近 {minutes} 分鐘流量',
  'monitor.timelineNote': '總量是本次啟動以來的累計值，重啟後歸零；每格的分位數是延遲直方圖的桶上界，因此只會落在離散的刻度上。圖上沒有資料的分鐘代表服務當時沒有流量。',
  'monitor.timelineLive': '本次啟動',
  'monitor.timelineHistory': '重啟前',
  'monitor.timelineLegendVolume': '請求量',
  'monitor.timelineLegendError': '5xx 錯誤',
  'monitor.timelinePeak': '峰值 {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': '資料庫裡沒有可用的歷史彙總；啟動時讀不到（表尚未建立或權限不足）時只有本次啟動的資料。',
  'monitor.routesTitle': '按路由統計',
  'monitor.routesNote': '路徑已正規化（數字與電郵換成 :id），因此同一路由的不同 id 會合併計算。',
  'monitor.colRoute': '路由',
  'monitor.colCount': '請求',
  'monitor.colAvg': '平均',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': '最慢',
  'monitor.colErrors': '錯誤',
  'monitor.routeOther': '其他（已達路由數上限）',
  'monitor.clientsTitle': '來源位址',
  'monitor.clientsNote': '依請求量由多到少列出目前活躍的來源。取自 X-Forwarded-For 或 X-Real-IP 的位址未經可信代理驗證，請先確認這兩道代理會覆寫這些標頭，再決定要不要封鎖。',
  'monitor.noClients': '尚未追蹤到任何來源。',
  'monitor.noClientsBody': '每個請求都會記到它的來源位址上。目前這一欄是空的。',
  'monitor.clientsDropped': '來源數量已達上限 {limit}，最久未再出現的 {count} 個位址已被移出追蹤。看到這一行的意思是這份清單不完整，而不是「只有這麼多人來過」。',
  'monitor.colIp': '位址',
  'monitor.colSource': '來源',
  'monitor.colRateLimited': '限流阻擋',
  'monitor.colBanned': '封鎖擋下',
  'monitor.colLastRoute': '最近打到',
  'monitor.colActions': '動作',
  'monitor.colBlock': '封鎖',
  'monitor.blocking': '封鎖中…',
  'monitor.sourcePeer': '連線對端',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.trustLegacy': '未設定 TRUSTED_PROXY_CIDRS：程式沿用「X-Forwarded-For 最左項優先」的舊行為。在確認前面確實有一道會覆寫這些標頭的代理、且使用者無法繞過它直連之前，限流與 IP 封鎖都可以被單一偽造標頭繞過，稽核紀錄的來源位址也不宜當成證據。',
  'monitor.trustConfigured': '已設定可信代理：只有連線對端落在下列位址段時才採信 X-Forwarded-For／X-Real-IP，其餘一律以連線對端為準。目前生效：{cidrs}。',
  'monitor.trustBroken': '宣告了 TRUSTED_PROXY_CIDRS，但沒有任何一項能被解析成位址段（{declared}），因此實際上仍在使用未設定時的舊行為。',
  'monitor.trustPartial': 'TRUSTED_PROXY_CIDRS 有下列項目無法解析成位址段（{invalid}），因此它們的轉送標頭永遠不會被採信。來自這些位址段的請求會以連線對端的位址分桶，也就是共用同一組限流額度與封鎖查詢。',
  'monitor.blockTitle': '封鎖 {ip}',
  'monitor.blockMessage': '這個位址的寫入型請求（發文、留言、按讚、檢舉、上傳圖片、登入跳轉）會被拒絕 {duration}，閱讀不受影響。確定要封鎖嗎？',
  'monitor.blockReason': '從監控頁封鎖',
  'monitor.blocked': '已封鎖 {ip}',
  'monitor.blockFailed': '封鎖操作失敗。',
  'monitor.limitsTitle': '限流器',
  'monitor.limitsNote': '每組額度按端點成本分開計算；阻擋次數是本次啟動以來回 429 的總量。',
  'monitor.colLimiter': '限流器',
  'monitor.colBudget': '額度',
  'monitor.colAllowed': '放行',
  'monitor.colBlocked': '阻擋',
  'monitor.colTracked': '追蹤中的來源',
  'monitor.colBlockedRate': '阻擋率',
  'monitor.limitContent': '內容寫入',
  'monitor.limitUpload': '圖片上載',
  'monitor.limitAuth': 'OAuth 登入',
  'monitor.limitBudget': '{window} 秒內 {limit} 次',
  'monitor.limitUnknown': '（未知）',
  'monitor.noLimits': '沒有可用的限流器。',
  'monitor.noLimitsBody': '限流器尚未建立，統計資料不可用。',

  'log.title': '操作紀錄',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': '查詢後臺的每一項變更：誰、什麼時候、對哪個對象、哪些欄位從什麼變成什麼。',
  'log.refresh': '重新整理',
  'log.loadFailed': '稽核紀錄載入失敗。',
  'log.empty': '沒有符合條件的操作紀錄。',
  'log.emptyBody': '放寬篩選條件，或確認這個時間範圍內真的沒有任何後臺變更。',
  'log.count': '第 {from}–{to} 筆，共 {total} 筆',
  'log.retention': '紀錄保留 {days} 天，過期由背景作業刪除。這一頁沒有任何刪除紀錄的按鈕 —— 能刪掉自己紀錄的稽核日誌不算稽核日誌。',
  'log.filterActor': '操作者',
  'log.filterAction': '動作',
  'log.filterTargetType': '資源類別',
  'log.filterFrom': '起始日期',
  'log.filterTo': '截止日期',
  'log.filterAll': '全部',
  'log.filterApply': '套用篩選',
  'log.filterReset': '清除篩選',
  'log.filterTargetHint': '在某一筆的「對象」上按一下，即可只看與它有關的操作。',
  'log.colTime': '時間',
  'log.colActor': '操作者',
  'log.colAction': '動作',
  'log.colTarget': '對象',
  'log.colChanges': '變更內容',
  'log.colOrigin': '來源',
  'log.noChanges': '（無欄位變更）',
  'log.changedTo': '變更為',
  'log.removed': '（已刪除）',
  'log.created': '（新增）',
  'log.requestId': 'request {id}',
  'log.page': '第 {page} 頁',
  'log.targetUser': '使用者',
  'log.targetPost': '文章',
  'log.targetComment': '留言',
  'log.targetReport': '檢舉',
  'log.targetTag': '標籤',
  'log.targetSystem': '系統',
  'log.actionUserSuspend': '停權',
  'log.actionUserReinstate': '恢復',
  'log.actionUserTags': '改標籤',
  'log.actionUserPost': '代發文',
  'log.actionUserComment': '代留言',
  'log.actionUserContent': '刪除其內容',
  'log.actionPostCreate': '發文',
  'log.actionPostUpdate': '改文',
  'log.actionPostDelete': '刪文',
  'log.actionCommentCreate': '留言',
  'log.actionCommentUpdate': '改留言',
  'log.actionCommentDelete': '刪留言',
  'log.actionReportCreate': '建立檢舉',
  'log.actionReportResolve': '檢舉成立',
  'log.actionReportReject': '檢舉不成立',
  'log.actionReportUpdate': '改檢舉',
  'log.actionReportDelete': '刪檢舉',
  'log.actionTagCreate': '新增標籤',
  'log.actionTagUpdate': '改標籤名',
  'log.actionTagDelete': '刪標籤',
  'log.fieldStatus': '帳號狀態',
  'log.fieldContent': '內容',
  'log.fieldName': '名稱',
  'log.fieldTags': '標籤',
  'log.fieldReason': '原因',
  'log.fieldAuthorEmail': '作者',
  'log.fieldReporterEmail': '檢舉人',
  'log.fieldTargetType': '資源類別',
  'log.fieldTargetId': '資源編號',
  'log.fieldPostId': '文章編號',
  'log.fieldCommentId': '留言編號',
  'log.fieldPostIdShort': '文章',
  'log.fieldCommentIdShort': '留言',
  'log.fieldTarget': '對象',
  'log.fieldAssignmentsRemoved': '解除綁定',
  'log.truncated': '已截斷',

  'stats.title': '內容趨勢',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': '每天有多少新使用者、文章與留言，以及目前最熱門的文章、標籤與作者。',
  'stats.refresh': '重新整理',
  'stats.loadFailed': '內容統計載入失敗。',
  'stats.window': '顯示最近幾天',
  'stats.windowDays': '{days} 天',
  'stats.windowClamped': '（最多顯示 90 天）',
  'stats.windowNote': '每天的份量以伺服器主機的本地時區切分。若站台部署在 UTC 而管理員在另一個時區，「今天」的數字看起來會少一截 —— 那是時區差，不是流量掉了。',
  'stats.generatedAt': '統計產生於 {time}',
  'stats.seriesTitle': '每日新增',
  'stats.seriesNote': '三條線各自獨立的量級，因此分開顯示而不是疊在一起。',
  'stats.seriesUsers': '新使用者',
  'stats.seriesPosts': '新文章',
  'stats.seriesComments': '新留言',
  'stats.seriesEmpty': '這個視窗內沒有任何資料。',
  'stats.totalsTitle': '視窗內合計',
  'stats.totalsNote': '這是這段期間「新增」的量，不是站台目前的總數。',
  'stats.totalUsers': '新使用者',
  'stats.totalPosts': '新文章',
  'stats.totalComments': '新留言',
  'stats.totalLikes': '新按讚',
  'stats.topPostsTitle': '熱門文章',
  'stats.topPostsNote': '依「留言數 + 按讚數」排序，只算視窗內發表的文章。',
  'stats.topTagsTitle': '熱門標籤',
  'stats.topTagsNote': '依綁定人數排序，不限時間 —— 標籤是身分分類而不是事件。',
  'stats.topAuthorsTitle': '活躍作者',
  'stats.topAuthorsNote': '依視窗內的發文數排序，留言數另外列出。',
  'stats.colExcerpt': '內容摘要',
  'stats.colEngagement': '互動',
  'stats.colPosts': '文章',
  'stats.colComments': '留言',
  'stats.colUsers': '人數',
  'stats.colAuthor': '作者',
  'stats.empty': '這個視窗內沒有資料。',
  'stats.emptyBody': '把天數拉長，或確認這段期間確實沒有新的內容。',
  'stats.engagement': '{comments} 留言・{likes} 讚',
  'stats.rank': '第 {rank} 名',

  'export.title': '匯出與批次操作',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': '把站內資料匯出成 CSV 對帳，或一次處理多個帳號。',
  'export.download': '下載 CSV',
  'export.downloading': '準備中…',
  'export.exportTitle': '匯出',
  'export.exportNote': '每份匯出最多 5 萬列，超過時只匯出最新的部分。三份匯出的欄位都與後臺頁面上看到的欄位一致，可以直接對帳。',
  'export.exportUsers': '使用者清單',
  'export.exportUsersNote': 'email、狀態、建立與更新時間、文章數、留言數。',
  'export.exportPosts': '文章清單',
  'export.exportPostsNote': '編號、作者、內容前 200 字、建立時間、留言數、按讚數。',
  'export.exportReports': '檢舉工單',
  'export.exportReportsNote': '編號、被檢舉對象、檢舉人、原因、狀態與審核軌跡。',
  'export.safety': '匯出檔以 UTF-8 開頭（含 BOM），Excel 開啟不會亂碼。',
  'export.safetyPrefix': '開頭是 = + - @ 或不可見空白的欄位值，會被加上單引號前綴 —— 那是為了讓試算表把它當文字而不是公式執行。這個前綴是刻意保留的，不要要求移除它。',
  'export.batchTitle': '批次操作',
  'export.batchNote': '在「用戶管理」頁勾選多個帳號後，批次按鈕才會啟用。批次是整批生效或整批不動，不會有部分套用的結果。',
  'export.batchSuspend': '批次停權',
  'export.batchReinstate': '批次恢復',
  'export.batchTags': '批次套用標籤',
  'export.batchTagsNote': '覆寫語意：送出去的清單就是結果，傳空清單等同清除所有標籤。',
  'export.batchConfirm': '確定要對 {count} 個帳號執行「{action}」嗎？',
  'export.batchConfirmTags': '確定要對 {count} 個帳號覆寫標籤為 {tags} 嗎？',
  'export.batchTagsPicker': '選擇標籤',
  'export.batchTagsNone': '不選任何標籤（清除全部）',
  'export.batchRunning': '處理中…',
  'export.batchDone': '已更新 {updated} 個帳號',
  'export.batchDoneUnchanged': '其中 {unchanged} 個原本就是目標狀態，未變更',
  'export.batchSkipped': '跳過 {count} 個',
  'export.batchMax': '一次最多處理 200 個帳號',
  'export.gotoUsers': '前往用戶管理',
  'export.noSelection': '請先在用戶管理頁勾選帳號。',
  'export.selected': '已選 {count} 個帳號',
  'export.clearSelection': '清除選取',
  'export.selectionHint': '選取會保留在這個分頁，關閉頁面後消失。',

  'session.title': '登入與 Session',
  'session.eyebrow': 'SESSIONS',
  'session.copy': '查看目前還有效的登入狀態，必要時強制登出某個帳號的所有裝置。',
  'session.refresh': '重新整理',
  'session.loadFailed': 'Session 清單載入失敗。',
  'session.privacyTitle': '為什麼看不到完整 token',
  'session.privacyNote': 'token 就是登入憑證本身。只顯示前 8 個字元是為了分辨「是不是同一支 session」，而那不足以讓任何人登入你的站 —— 包括拿到這張截圖的人。這是刻意的限制，不是尚未完成的功能。',
  'session.expireNote': 'Session 在連續 {hours} 小時沒有活動後才會失效；只要還在操作就會自動延續。',
  'session.filterEmail': '只看某個帳號',
  'session.filterPlaceholder': '完整 email',
  'session.search': '查詢',
  'session.clearFilter': '清除',
  'session.summary': '全站 {total} 支 session，已檢查 {scanned} 個 key',
  'session.truncated': '掃描達到上限 {scanned} 個 key 而提前結束，因此這份清單不完整。',
  'session.empty': '目前沒有任何 session。',
  'session.emptyBody': '沒有任何人在線，或所有 session 都已過期。',
  'session.colUser': '帳號',
  'session.colToken': 'Session',
  'session.colCreated': '建立時間',
  'session.colExpires': '到期時間',
  'session.colRemaining': '剩餘',
  'session.colActions': '動作',
  'session.unknown': '未知',
  'session.adminBadge': '管理員',
  'session.revoke': '強制登出',
  'session.revokeTitle': '強制登出 {email}',
  'session.revokeMessage': '這個帳號目前的 {count} 筆 session 會立刻全部失效，任何裝置上的登入狀態都會被清掉。使用者必須重新登入才能繼續使用。確定要執行嗎？',
  'session.revokeRunning': '撤銷中…',
  'session.revokeDone': '已撤銷 {count} 筆 session',
  'session.revokeNone': '這個帳號目前沒有任何 session',
  'session.revokeFailed': '撤銷失敗。',
  'session.revokeUnavailable': '無法確認是否已全部撤銷，請稍後再試。掃描達到上限時會出現這種情況 —— 它不表示撤銷失敗，而是「可能還有 session 沒被掃到」。',
  'session.titleColumnNote': '可分辨用的前綴，不能拿來登入',

  'block.title': 'IP 封鎖',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': '把確認為濫用的來源位址加進封鎖名單。名單存在 Redis，重啟與部署都不會解除。',
  'block.refresh': '重新整理',
  'block.loadFailed': '封鎖名單載入失敗。',
  'block.unavailable': '這台站沒有接 Redis，因此封鎖功能未啟用。',
  'block.unavailableNote': '封鎖名單需要與 session、媒體 token 共用的那個 Redis。接上之後這一頁就會開始作用；在那之前，所有防護只有限流（行程內，重啟即失效）。',
  'block.add': '封鎖',
  'block.addTitle': '封鎖一個 IP',
  'block.addMessage': '被封鎖的來源位址會被拒絕所有寫入型請求（發文、留言、按讚、檢舉、上傳圖片、登入跳轉），直到時間到期。閱讀不受影響。',
  'block.ipLabel': 'IP 位址',
  'block.ipPlaceholder': '203.0.113.9 或 2001:db8::1',
  'block.durationLabel': '封鎖多久',
  'block.reasonLabel': '原因',
  'block.reasonPlaceholder': '為什麼封這個位址（會記進稽核紀錄）',
  'block.reasonHint': '原因只寫進稽核紀錄，不會顯示給被封鎖的人，也不會出現在公開的錯誤訊息裡。',
  'block.blocking': '封鎖中…',
  'block.done': '已封鎖 {ip}',
  'block.removed': '已解除封鎖 {ip}',
  'block.removedNone': '{ip} 本來就沒有被封鎖',
  'block.failed': '封鎖操作失敗。',
  'block.unavailableService': '封鎖名單不可用（缺少 Redis）',
  'block.colIp': 'IP',
  'block.colExpires': '到期',
  'block.colRemaining': '剩餘',
  'block.colActions': '動作',
  'block.unblock': '解除',
  'block.unblockTitle': '解除封鎖 {ip}',
  'block.unblockMessage': '這個位址會立刻恢復正常的寫入權限。確定要解除嗎？',
  'block.empty': '封鎖名單是空的。',
  'block.emptyBody': '沒有任何來源位址被封鎖。這是正常狀態 —— 封鎖永遠是管理員的決定，系統不會自動封鎖任何人。',
  'block.notAutoNote': '這個名單**不會**自動填滿。超過限流額度的來源位址只會收到 429，不會被自動加進來 —— 因為同一個出口後面可能有一整間辦公室或一整個 NAT，而自動封鎖會誤傷他們。',
  'block.scopeNoteLabel': '範圍',
  'block.notAutoNoteLabel': '不會自動封鎖',
  'block.maxNoteLabel': '長度上限',
  'block.scopeNote': '封鎖只擋寫入型請求。閱讀貼文、留言與靜態資產不受影響，被封鎖的人仍然可以登入與看內容 —— 這是刻意的：閱讀端點刻意不掛限流（否則匿名訪客無法使用），封鎖沿用同一個範圍。',
  'block.maxNote': '單次封鎖最長 365 天。輸入更長的時間會被收斂到一年 —— 因為到期時間是存成一個數字，「永久」會變成一筆沒有人記得、也永遠不會自己解除的封鎖。',
  'block.count': '共 {count} 筆封鎖',
  'block.ipInvalid': 'IP 格式不正確。請填一個 IPv4 或 IPv6 位址；不支援 CIDR 範圍。',
  'announce.label': '站內公告',
  'announce.publicNote': '站方公告',
  'announce.closeAria': '關閉這則站內公告',
  'announce.publishedOn': '發佈於 {date}',
  'announce.expiresOn': '將於 {date} 自動失效',
  'announce.neverExpires': '長期有效',
  'announce.pinnedBadge': '置頂',
  'announce.title': '站內公告',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': '在全站最上方顯示一則公告。同一時間只有一則生效 —— 發佈新公告會自動停用舊的。',
  'announce.refresh': '重新整理',
  'announce.loadFailed': '公告清單載入失敗。',
  'announce.new': '發佈新公告',
  'announce.edit': '修改這則',
  'announce.deactivate': '停用',
  'announce.reactivate': '重新啟用',
  'announce.deleteNote': '公告不會被刪除，只會被停用 —— 保留歷史是為了回答「這則公告是什麼時候、經誰發布的」。',
  'announce.bodyLabel': '公告內容',
  'announce.bodyPlaceholder': '例如：系統將於週四 02:00–04:00 進行維護。',
  'announce.bodyHint': '最多 300 字。純文字，換行會保留。',
  'announce.activeLabel': '立即顯示',
  'announce.expiryLabel': '有效時間',
  'announce.expiryNever': '永不自動過期',
  'announce.expiryHours': '{hours} 小時後',
  'announce.expiryDays': '{days} 天後',
  'announce.saving': '儲存中…',
  'announce.published': '已發佈公告',
  'announce.updated': '已更新公告',
  'announce.deactivated': '已停用公告',
  'announce.reactivated': '已重新啟用公告',
  'announce.saveFailed': '公告操作失敗。',
  'announce.empty': '還沒有任何公告。',
  'announce.emptyBody': '發佈一則之後，它會出現在所有訪客的頁面最上方。',
  'announce.colBody': '內容',
  'announce.colState': '狀態',
  'announce.colAuthor': '發佈者',
  'announce.colCreated': '發佈時間',
  'announce.colActions': '動作',
  'announce.stateActive': '顯示中',
  'announce.stateInactive': '已停用',
  'announce.stateExpired': '已過期',
  'announce.confirmDeactivate': '確定要停用這則公告嗎？停用後所有訪客立刻就看不到它了。',
  'announce.confirmEdit': '修改這則公告的內容或有效時間？',
  'announce.count': '共 {count} 則',
  'users.title':
    '用戶管理',
  'users.contentAction':
    '內容',
  'users.updateContentFailed':
    '內容操作失敗。',
  'users.updated':
    '內容已更新。',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    '檢視論壇用戶的活動統計、標籤與帳戶狀態，並處理停權與逐篇內容治理。',
  'users.refresh':
    '更新資料',
  'users.statTotal':
    '用戶總數',
  'users.statActive':
    '啟用中',
  'users.statSuspended':
    '已停權',
  'users.statContent':
    '文章／留言總量',
  'users.count':
    '{count} 位用戶',
  'users.tagsCount':
    '{count} 個標籤',
  'users.loadFailed':
    '用戶資料載入失敗。',
  'users.tagsLoadFailed':
    '標籤資料載入失敗。',
  'users.panelTitle':
    '論壇用戶',
  'users.tagsPanelTitle':
    '用戶標籤',
  'users.colUser':
    '用戶',
  'users.colTags':
    '標籤',
  'users.colStatus':
    '狀態',
  'users.colPosts':
    '文章',
  'users.colComments':
    '留言',
  'users.colLikes':
    '讚好',
  'users.colLastActivity':
    '最後活動',
  'users.colActions':
    '操作',
  'users.nicknameUnset':
    '未設定暱稱',
  'users.notSet':
    '未設定',
  'users.statusActive':
    '啟用中',
  'users.statusSuspended':
    '已停權',
  'users.emptyTitle':
    '目前沒有用戶資料',
  'users.emptyBody':
    '沒有用戶透過 Google 登入論壇時，這裡才會是空的。',
  'users.tagsEmptyTitle':
    '目前沒有標籤',
  'users.tagsEmptyBody':
    '先建立標籤，才能在用戶清單裡替對象套用。',
  'users.addTag':
    '新增標籤',
  'users.colName':
    '名稱',
  'users.colCreated':
    '建立時間',
  'users.colUpdated':
    '更新時間',
  'users.renameTag':
    '重新命名',
  'users.suspend':
    '停權',
  'users.restore':
    '恢復',
  'users.statusDialogTitle':
    '{action}此用戶',
  'users.statusSuspendMessage':
    '{email} 將無法再登入論壇，既有文章與留言會保留。要繼續嗎？',
  'users.statusRestoreMessage':
    '{email} 將恢復登入與發文權限。要繼續嗎？',
  'users.userSuspended':
    '用戶已停權。',
  'users.userRestored':
    '用戶已恢復。',
  'users.updateStatusFailed':
    '更新用戶狀態失敗。',
  'users.editTagsTitle':
    '編輯標籤 · {user}',
  'users.editTagsMessage':
    '勾選要套用的標籤；全部取消選取等同移除此用戶的所有標籤。',
  'users.tagsUpdated':
    '用戶標籤已更新。',
  'users.updateTagsFailed':
    '更新用戶標籤失敗。',
  'users.contentLoadFailed':
    '內容載入失敗。',
  'users.contentLoadFailedShort':
    '內容載入失敗。',
  'users.contentPanelTitle':
    '用戶內容',
  'users.contentCount':
    '{posts} 篇文章 · {comments} 則留言',
  'users.contentLoading':
    '正在載入文章與留言…',
  'users.addPost':
    '新增文章',
  'users.addComment':
    '新增留言',
  'users.postsColumn':
    '文章',
  'users.commentsColumn':
    '留言',
  'users.noPosts':
    '尚無文章',
  'users.noComments':
    '尚無留言',
  'users.postRef':
    '文章 #{id}',
  'users.editRecordTitle':
    '編輯{kind} #{id}',
  'users.deleteRecordTitle':
    '刪除{kind} #{id}',
  'users.deleteRecordMessage':
    '刪除後無法復原，相關的讚好紀錄與關聯資料也會一併移除。要繼續嗎？',
  'users.contentLabel':
    '內容',
  'users.addPostTitle':
    '新增文章',
  'users.addPostMessage':
    '此內容會以該用戶嘅身分發佈，作者欄位無法偽造。',
  'users.postContentLabel':
    '文章內容',
  'users.postContentPlaceholder':
    '輸入文章內容',
  'users.pickPostTitle':
    '選擇文章',
  'users.postIdLabel':
    '文章 ID',
  'users.postIdPlaceholder':
    '要留言的文章編號',
  'users.addCommentTitle':
    '新增留言',
  'users.commentContentLabel':
    '留言內容',
  'users.commentContentPlaceholder':
    '輸入留言內容',
  'users.createTagTitle':
    '新增標籤',
  'users.createTagMessage':
    '標籤可用於為用戶分類，例如「版主」「活躍」或「已封鎖」。',
  'users.tagNameLabel':
    '標籤名稱',
  'users.tagNamePlaceholder':
    '最多 50 個字',
  'users.renameTagTitle':
    '重新命名標籤',
  'users.renameTagMessage':
    '所有使用呢個標籤嘅用戶都會看到新名稱。',
  'users.deleteTagTitle':
    '刪除標籤「{name}」',
  'users.deleteTagMessage':
    '刪除後，所有用戶身上嘅此標籤都會一併移除，且無法復原。要繼續嗎？',
  'users.deleteTagConfirm':
    '刪除標籤',
  'users.tagCreated':
    '標籤已建立。',
  'users.createTagFailed':
    '建立標籤失敗。',
  'users.tagUpdated':
    '標籤已更新。',
  'users.updateTagFailed':
    '更新標籤失敗。',
  'users.tagDeleted':
    '標籤已刪除。',
  'users.deleteTagFailed':
    '刪除標籤失敗。',
  'users.refreshDone':
    '已更新至最新資料。',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    '後台管理',
  'users.signinBody':
    '集中式內容治理，讓每一次審查都清楚、快速且可追蹤。',
  'users.signinStep1':
    '安全驗證',
  'users.signinStep2':
    '用戶治理',
  'users.signinStep3':
    '內容審查',
  'users.signinPanelTitle':
    '登入管理控制台',
  'users.signinPanelBody':
    '管理控制台僅開放給已授權的 Google 管理員帳戶，請使用管理員身分登入。',
  'posts.title':
    '論壇文章',
  'posts.searching':
    '搜尋中…',
  'posts.searchDegraded':
    '（搜尋服務未啟用，以資料庫關鍵字比對）',
  'posts.searchSummary':
    '「{query}」找到 {total} 筆，本頁顯示 {shown} 筆',
  'posts.pendingCount':
    '{count} 筆待處理',
  'posts.pageSummary':
    '第 {page} / {pages} 頁，本頁 {count} 篇',
  'posts.listLoadFailed':
    '文章列表載入失敗。',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    '建立、編輯與刪除論壇文章，並在留言層級處理內容治理。',
  'posts.toReports':
    '舉報管理',
  'posts.editorTitleNew':
    '新增文章',
  'posts.editorTitleEdit':
    '編輯文章 #{id}',
  'posts.editorNote':
    '以管理員身分發佈；作者取自登入身分，請求內容無法偽造作者。',
  'posts.cancelEdit':
    '取消編輯',
  'posts.contentLabel':
    '文章內容',
  'posts.contentPlaceholder':
    '輸入文章內容',
  'posts.saveChanges':
    '儲存修改',
  'posts.emptyContent':
    '文章內容不可為空白。',
  'posts.saving':
    '儲存中…',
  'posts.saved':
    '文章已更新。',
  'posts.published':
    '文章已發佈。',
  'posts.saveFailed':
    '儲存失敗。',
  'posts.deleteTitle':
    '刪除文章 #{id}',
  'posts.deleteMessage':
    '刪除後無法復原，該文章下面嘅所有留言也會一併移除。要繼續嗎？',
  'posts.deleteConfirm':
    '刪除文章',
  'posts.deleted':
    '文章已刪除。',
  'posts.deleteFailed':
    '刪除文章失敗。',
  'posts.commentUpdated':
    '留言已更新。',
  'posts.commentActionFailed':
    '留言操作失敗。',
  'posts.pickPostTitle':
    '選擇留言所屬文章',
  'posts.pickPostMessage':
    '預設係呢列嘅文章；若要掛到另一篇，請改成正確的文章編號。',
  'posts.postIdLabel':
    '文章 ID',
  'posts.addCommentAtTitle':
    '在文章 #{id} 新增留言',
  'posts.addCommentMessage':
    '此留言會以管理員身分發佈。',
  'posts.commentContentLabel':
    '留言內容',
  'posts.commentContentPlaceholder':
    '輸入留言內容',
  'posts.add':
    '新增',
  'posts.editCommentTitle':
    '編輯留言 #{id}',
  'posts.deleteCommentTitle':
    '刪除留言 #{id}',
  'posts.deleteCommentMessage':
    '刪除後無法復原。要繼續嗎？',
  'posts.listTitle':
    '文章列表',
  'posts.clearSearch':
    '清除搜尋',
  'posts.searchLabel':
    '關鍵字搜尋',
  'posts.searchPlaceholder':
    '帖文內容，或帖文作者的完整 Email',
  'posts.searchHint':
    '按相關性排序；輸入完整 Email 可找出該用家的所有帖文。搜尋功能取代分頁，結果最多顯示 25 筆。',
  'posts.searchTotal':
    '搜尋結果共 {total} 筆',
  'posts.searchFailed':
    '搜尋失敗。',
  'posts.searchStatusFailed':
    '搜尋失敗',
  'posts.colContentImage':
    '內容與圖片',
  'posts.colEngagement':
    '互動',
  'posts.colComments':
    '留言',
  'posts.colAuthor':
    '作者',
  'posts.imageAlt':
    '文章附圖',
  'posts.likes':
    '{count} 讚好',
  'posts.author':
    '作者：{name}',
  'posts.emptyTitle':
    '現時冇論壇文章',
  'posts.emptyBody':
    '可以而家用上方的編輯器建立第一篇文章。',
  'posts.emptySearchTitle':
    '冇符合的帖文',
  'posts.emptySearchBody':
    '「{query}」冇比對到任何帖文，換個關鍵字再試下。',
  'posts.commentCount':
    '{count} 則留言',
  'posts.noComments':
    '尚無留言',
  'posts.reportsTitle':
    '待處理舉報',
  'posts.allReports':
    '全部舉報',
  'posts.colReportedContent':
    '被舉報內容',
  'posts.colReason':
    '舉報原因',
  'posts.colReporter':
    '舉報人',
  'posts.colTime':
    '時間',
  'posts.colVerdict':
    '裁定',
  'posts.emptyReportsTitle':
    '冇待處理的舉報',
  'posts.emptyReportsBody':
    '所有舉報都已裁定完成。',
  'posts.verdictResolved':
    '已處理',
  'posts.verdictRejected':
    '不成立',
  'posts.verdictDialogTitle':
    '將舉報 #{id} 標為「{label}」',
  'posts.verdictDialogMessage':
    '標記後，舉報會離開待處理清單，但資料仍保留在舉報管理頁。要繼續嗎？',
  'posts.verdictConfirm':
    '標為{label}',
  'posts.verdictDone':
    '舉報已標為{label}。',
  'posts.verdictFailed':
    '舉報狀態更新失敗。',
  'posts.pin': '置頂',
  'posts.unpin': '取消置頂',
  'posts.pinTitle': '置頂這篇文章',
  'posts.unpinTitle': '取消置頂這篇文章',
  'posts.pinMessage': '置頂後這篇文章會固定顯示在所有訪客的動態最前面，不會被新文章擠下去。',
  'posts.unpinMessage': '取消置頂後這篇文章會回到依時間排序的位置。',
  'posts.pinDone': '已置頂',
  'posts.unpinDone': '已取消置頂',
  'posts.pinFailed': '置頂操作失敗。',
  'reports.title':
    '舉報管理',
  'reports.listSummary':
    '{count} 筆 · {filter}',
  'reports.listLoadFailed':
    '舉報列表載入失敗。',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    '逐筆裁定舉報：「通過」會刪除被舉報的內容，「不成立」則保留原文。改回「待處理」會清除既有裁定時間。',
  'reports.backToPosts':
    '返回文章',
  'reports.editorTitle':
    '編輯舉報 #{id}',
  'reports.editorNote':
    '可修改舉報原因與狀態；目標與舉報人是既有紀錄，不在此更改。',
  'reports.targetTypeLabel':
    '目標類型',
  'reports.targetIdLabel':
    '目標 ID',
  'reports.reporterEmailLabel':
    '舉報人 Email',
  'reports.statusLabel':
    '狀態',
  'reports.reasonLabel':
    '舉報原因',
  'reports.reasonHint':
    '最多 500 字，會直接顯示俾其他管理員作裁定依據。',
  'reports.filterLabel':
    '按狀態篩選',
  'reports.filterAll':
    '全部',
  'reports.listTitle':
    '舉報列表',
  'reports.colTarget':
    '目標內容',
  'reports.colReason':
    '舉報原因',
  'reports.colReporter':
    '舉報人',
  'reports.colStatus':
    '狀態',
  'reports.targetGone':
    '（內容已刪除）',
  'reports.author':
    '作者：{name}',
  'reports.deleteTitle':
    '刪除舉報 #{id}',
  'reports.deleteMessage':
    '刪除的是呢筆舉報紀錄本身，被舉報的內容不受影響，而且無法回復。要繼續嗎？',
  'reports.deleteConfirm':
    '刪除舉報',
  'reports.deleted':
    '舉報已刪除。',
  'reports.deleteFailed':
    '刪除舉報失敗。',
  'reports.updated':
    '舉報已更新。',
  'reports.saveFailed':
    '儲存舉報失敗。',
  'reports.approveTitle':
    '通過舉報 #{id}',
  'reports.approveGoneMessage':
    '被舉報的{kind} #{id} 已經唔存在，呢一筆只會被標為已處理。',
  'reports.approveMessage':
    '將永久刪除被舉報的{kind} #{id}（若係文章，該文章底下的所有留言亦一併移除），並把呢筆舉報標為已處理。要繼續嗎？',
  'reports.approveConfirm':
    '通過並刪帖文',
  'reports.approveGoneDone':
    '內容已唔存在，舉報已標為已處理。',
  'reports.approveDone':
    '已刪帖文並將舉報標為已處理。',
  'reports.approveFailed':
    '通過舉報失敗。',
  'reports.approveTitleGone':
    '內容已刪除，只會標記舉報',
  'reports.approveTitleFull':
    '刪除被舉報的內容並標為已處理',
  'reports.rejectTitle':
    '舉報 #{id} 不成立',
  'reports.rejectMessage':
    '不成立代表被舉報的內容唔需要處理，內容會原樣保留。要繼續嗎？',
  'reports.rejectConfirm':
    '標為不成立',
  'reports.rejectDone':
    '舉報已標為不成立。',
  'reports.statusFailed':
    '舉報狀態更新失敗。',
  'reports.emptyTitle':
    '現時冇舉報',
  'reports.emptyBody':
    '呢個篩選條件下冇任何紀錄。',
  'reports.rejectTitleAttr':
    '保留內容，只將舉報標為不成立',
  'kind.post':
    '文章',
  'kind.comment':
    '留言',
  'reports.statusPending':
    '待處理',
  'reports.statusResolved':
    '已處理',
  'reports.statusRejected':
    '不成立',
  'title.forum':
    '{site}',
  'title.login':
    '登入｜{site}',
  'title.newPost':
    '新增帖文｜{site}',
  'title.profile':
    '個人資料｜{site}',
  'title.publicProfile':
    '公開個人資料｜{site}',
  'title.following':
    '追蹤｜{site}',
  'title.adminUsers':
    '用戶管理｜{brand} 後台',
  'title.adminLogin':
    '登入｜{brand} 後台',
  'title.adminPosts':
    '論壇文章｜{brand} 後台',
  'title.adminReports':
    '舉報管理｜{brand} 後台',
  'title.adminMonitor': '系統監控｜{brand} 後台',
  'title.adminLog': '操作紀錄｜{brand} 後臺',
  'title.adminStats': '內容趨勢｜{brand} 後臺',
  'title.adminExport': '匯出與批次｜{brand} 後臺',
  'title.adminSessions': '登入與 Session｜{brand} 後臺',
  'title.adminBlocks': 'IP 封鎖｜{brand} 後臺',
  'title.adminAnnouncements': '站內公告｜{brand} 後臺',
};
