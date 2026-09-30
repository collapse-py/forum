/*
 * 簡體中文翻譯（src/i18n/translations/zh-CN.ts）
 *
 * 繁體中文原文在 ../messages.ts，那一檔是型別基準；這一檔只放字串。
 * 規則見 ../messages.ts 檔頭的說明。
 */

import type { MessageKey } from '../messages';

export const zhCN: Record<MessageKey, string> = {
  /* ==========================================================================
     common
     ========================================================================== */

  'common.cancel': '取消',
  'common.save': '保存',
  'common.submitting': '发送中...',
  'common.delete': '删除',
  'common.edit': '编辑',
  'common.search': '搜索',
  'common.loading': '加载中…',
  'common.loadFailed': '加载失败',
  'common.refresh': '刷新',
  'common.nextStep': '下一步',
  'common.prevPage': '上一页',
  'common.nextPage': '下一页',
  'common.create': '创建',
  'common.publish': '发布',
  'common.placeholder': '—',
  'common.backToHome': '返回首页',
  'common.backToForumHome': '返回论坛首页',
  'common.backOnePage': '返回上一页',

  /* ==========================================================================
     errors
     ========================================================================== */

  'error.request': '请稍后再试。',
  'error.requestStatus': '请求失败（HTTP {status}）',
  'error.loginRequired': '请先登录后再继续。',
  'error.adminSessionExpired': '登录状态已失效，即将返回登录页。',
  'error.fallbackLoad': '加载失败',
  'error.fallbackSearch': '搜索失败',
  'error.fallbackLike': '点赞失败',
  'error.fallbackComments': '评论加载失败',
  'error.fallbackCommentPost': '评论失败',
  'error.fallbackReport': '举报失败',
  'error.fallbackProfile': '读取个人资料失败',
  'error.fallbackProfileSave': '保存失败',
  'error.fallbackPublish': '发布响应格式错误',
  'error.fallbackUpload': '图片上传响应格式错误',
  'error.fallbackNotFound': '找不到该用户',
  'error.fallbackFollow': '关注失败',

  /* ==========================================================================
     auth
     ========================================================================== */

  'auth.checking': '正在检查登录状态...',
  'auth.statusUnknown': '无法确认登录状态',
  'auth.feedLoggedIn': '已登录，可发帖',
  'auth.feedLoggedOut': '登录后即可发帖',
  'auth.profileLoggedIn': '已登录',
  'auth.profileLoggedOut': '登录后即可设置个人资料',
  'auth.googleLogin': 'Google 登录',
  'auth.loginWithGoogle': '使用 Google 账号登录',
  'auth.loginWithGoogleAdmin': '使用 Google 管理员账号登录',
  'auth.logout': '退出登录',

  /* ==========================================================================
     install
     ========================================================================== */

  'install.button': '安装 App',
  'install.hint':
    '目前浏览器未提供自动安装提示，请打开浏览器菜单，选择「安装应用」或「添加到主屏幕」。',

  /* ==========================================================================
     bottomNav
     ========================================================================== */

  'bottomNav.label': '主要导航',
  'bottomNav.home': '首页',
  'bottomNav.new': '新增',
  'bottomNav.profile': '我的',

  /* ==========================================================================
     i18n
     ========================================================================== */

  'i18n.ariaLabel': '选择语言',
  'i18n.current': '语言：{name}',

  /* ==========================================================================
     feed
     ========================================================================== */

  'feed.searchPlaceholder': '搜索帖子内容',
  'feed.searchAriaLabel': '搜索帖子',
  'feed.searchResultsLabel': '搜索结果',
  'feed.postsLabel': '论坛帖子',
  'feed.searchFailed': '搜索失败，请稍后再试。',
  'feed.searching': '搜索中...',
  'feed.searchMore': '加载更多搜索结果...',
  'feed.searchMoreFailed': '加载更多失败',
  'feed.searchFound': '找到 {total} 条',
  'feed.searchDegraded': '{base}（搜索服务未启用，目前使用数据库关键词匹配）',
  'feed.searchTotal': '共 {total} 条结果',
  'feed.searchNoResults': '找不到包含「{query}」的帖子。',
  'feed.loadingPosts': '正在加载帖子...',
  'feed.loadMorePosts': '加载更多帖子...',
  'feed.postsFailed': '帖子加载失败，请稍后再试。',
  'feed.postsFailedShort': '加载失败，请稍后再试',
  'feed.scrollMore': '向下滑动加载更多',
  'feed.endOfFeed': '已经到底了',
  'feed.noPosts': '还没有帖子，先留下第一个想法吧。',
  'feed.likeFailed': '点赞或取消点赞失败，请稍后再试。',

  /* ==========================================================================
     post
     ========================================================================== */

  'post.authorAnonymous': '匿',
  'post.report': '举报帖子',
  'post.imageAlt': '帖子图片',
  'post.unlike': '取消点赞',
  'post.like': '点赞',
  'post.reply': '回复',

  /* ==========================================================================
     comment
     ========================================================================== */

  'comment.loading': '评论加载中...',
  'comment.none': '暂无评论',
  'comment.loadFailed': '评论加载失败，请稍后再试',
  'comment.placeholder': '写下评论...',
  'comment.max': '最多 2000 字',
  'comment.submit': '评论',
  'comment.failed': '评论失败，请稍后再试。',
  'comment.report': '举报',
  'comment.more': '加载更多评论...',

  /* ==========================================================================
     report
     ========================================================================== */

  'report.reasonPlaceholder': '请输入举报原因（最多 500 字）',
  'report.note': '举报会发送给站方管理员',
  'report.formLabel': '举报输入框',
  'report.submit': '提交举报',
  'report.failed': '举报失败，请稍后再试。',
  'report.sent': '举报已提交，感谢你的反馈。',

  /* ==========================================================================
     newPost
     ========================================================================== */

  'newPost.avatarYou': '你',
  'newPost.eyebrow': 'NEW POST',
  'newPost.title': '新增帖子',
  'newPost.loginFirst': '请先使用 Google 登录后发帖',
  'newPost.contentPlaceholder': '分享你的想法...',
  'newPost.addImage': '添加图片',
  'newPost.emailPrivate': '你的 email 不会公开',
  'newPost.submit': '发布帖子',
  'newPost.publishing': '发布中...',
  'newPost.uploading': '图片上传中...',
  'newPost.failed': '发布失败，请稍后再试。',

  /* ==========================================================================
     profile
     ========================================================================== */

  'profile.eyebrow': 'YOUR PROFILE',
  'profile.title': '个人资料',
  'profile.edit': '编辑',
  'profile.loginPrompt': '登录后即可设置你的论坛昵称和个人简介。',
  'profile.nicknameLabel': '论坛昵称',
  'profile.notSet': '尚未设置',
  'profile.notSetBio': '尚未设置个人简介。',
  'profile.nicknameInput': '昵称',
  'profile.nicknamePlaceholder': '输入昵称',
  'profile.nicknameHint': '昵称会显示在你发布的帖子内容上，最多 30 个字。',
  'profile.bioLabel': '个人简介',
  'profile.bioPlaceholder': '介绍一下自己（选填）',
  'profile.bioHint': '最多 500 个字。',
  'profile.updated': '个人资料已更新。',
  'profile.saving': '保存中...',
  'profile.saveFailed': '保存失败，请稍后再试。',
  'profile.loadFailed': '读取失败，请稍后再试。',
  'profile.followingEntry': '我的关注',

  /* ==========================================================================
     publicProfile
     ========================================================================== */

  'publicProfile.eyebrow': 'PUBLIC PROFILE',
  'publicProfile.title': '公开个人资料',
  'publicProfile.avatar': '匿',
  'publicProfile.loading': '加载中...',
  'publicProfile.invalidLinkName': '无效的公开个人主页链接',
  'publicProfile.invalidLinkBio': '请从论坛帖子中的作者名称进入个人资料。',
  'publicProfile.notFound': '找不到该用户',
  'publicProfile.anonymous': '匿名用户',
  'publicProfile.noBio': '该用户尚未设置公开资料。',
  'publicProfile.loadFailed': '公开资料加载失败。',
  'publicProfile.postsLabel': '帖子',
  'publicProfile.emptyPosts': '这位用户还没有发表帖子。',

  'follow.label': '关注这位用户',
  'follow.action': '关注',
  'follow.actionDone': '已关注',
  'follow.unfollow': '取消关注',
  'follow.done': '已关注。',
  'follow.failed': '关注失败，请稍后再试。',

  'following.peopleLabel': '关注的人',
  'following.postsLabel': '关注的人的帖子',
  'following.peopleLoading': '正在加载关注列表...',
  'following.emptyPeople': '你还没有关注任何人。在帖子上点「关注」，或从他人的公开个人资料页关注。',
  'following.emptyPosts': '关注的人还没有发表帖子。',
  'following.peopleFailed': '关注列表加载失败。',
  'following.postsFailed': '关注的人的帖子加载失败。',

  /* ==========================================================================
     login
     ========================================================================== */

  'login.eyebrow': 'MEMBER ACCESS',
  'login.title': '欢迎回来',
  'login.body': '{site} 是给所有愿意写下想法的人的空间。不需要注册表单 —— 一个 Google 账号就能开始发帖。',
  'login.browseFirst': '先看看首页',

  /* ==========================================================================
     admin
     ========================================================================== */

  'admin.skipToMain': '跳至主要内容',
  'admin.railLabel': '管理菜单',
  'admin.railBrandAria': '{site}首页',
  'admin.consoleName': 'Admin Console',
  'admin.railNavLabel': '主要功能',
  'admin.railGovernance': '治理',
  'admin.railMode': '管理员模式',
  'admin.railExit': '返回论坛',
  'admin.topbarMenu': '切换管理菜单',
  'admin.statusOnline': '连接正常',
  'admin.topbarForum': '论坛',
  'admin.logoutFailed': '退出登录失败，请稍后再试。',
  'admin.navUsers': '用户管理',
  'admin.navPosts': '论坛帖子',
  'admin.navReports': '举报管理',
  'admin.listLoadFailed': '加载失败',
  'admin.dlgClose': '关闭窗口',
  'admin.dlgConfirm': '确认',
  'admin.dlgSave': '保存',
  'admin.dlgApplyTags': '应用标签',
  'admin.dlgNoTags': '目前没有可应用的标签，请先在下方「标签管理」中新增。',

  /* ==========================================================================
     users
     ========================================================================== */

  'users.title': '用户管理',
  'users.contentAction': '内容',
  'users.updateContentFailed': '内容操作失败。',
  'users.updated': '内容已更新。',
  'users.eyebrow': 'USER MANAGEMENT',
  'users.copy': '查看论坛用户的活动统计、标签与账号状态，并处理封禁与逐条内容治理。',
  'users.refresh': '更新数据',
  'users.statTotal': '用户总数',
  'users.statActive': '启用中',
  'users.statSuspended': '已封禁',
  'users.statContent': '帖子／评论总量',
  'users.count': '{count} 位用户',
  'users.tagsCount': '{count} 个标签',
  'users.loadFailed': '用户数据加载失败。',
  'users.tagsLoadFailed': '标签数据加载失败。',
  'users.panelTitle': '论坛用户',
  'users.tagsPanelTitle': '用户标签',
  'users.colUser': '用户',
  'users.colTags': '标签',
  'users.colStatus': '状态',
  'users.colPosts': '帖子',
  'users.colComments': '评论',
  'users.colLikes': '点赞',
  'users.colLastActivity': '最后活动',
  'users.colActions': '操作',
  'users.nicknameUnset': '未设置昵称',
  'users.notSet': '未设置',
  'users.statusActive': '启用中',
  'users.statusSuspended': '已封禁',
  'users.emptyTitle': '目前没有用户数据',
  'users.emptyBody': '没有用户通过 Google 登录论坛时，这里才会是空的。',
  'users.tagsEmptyTitle': '目前没有标签',
  'users.tagsEmptyBody': '先创建标签，才能在用户列表里给对象应用。',
  'users.addTag': '新增标签',
  'users.colName': '名称',
  'users.colCreated': '创建时间',
  'users.colUpdated': '更新时间',
  'users.renameTag': '重命名',
  'users.suspend': '封禁',
  'users.restore': '恢复',
  'users.statusDialogTitle': '{action}该用户',
  'users.statusSuspendMessage': '{email} 将无法再登录论坛，已有帖子与评论会保留。要继续吗？',
  'users.statusRestoreMessage': '{email} 将恢复登录与发帖权限。要继续吗？',
  'users.userSuspended': '用户已封禁。',
  'users.userRestored': '用户已恢复。',
  'users.updateStatusFailed': '更新用户状态失败。',
  'users.editTagsTitle': '编辑标签 · {user}',
  'users.editTagsMessage': '勾选要应用的标签；全部取消等同于移除该用户的所有标签。',
  'users.tagsUpdated': '用户标签已更新。',
  'users.updateTagsFailed': '更新用户标签失败。',
  'users.contentLoadFailed': '内容加载失败。',
  'users.contentLoadFailedShort': '内容加载失败。',
  'users.contentPanelTitle': '用户内容',
  'users.contentCount': '{posts} 篇帖子 · {comments} 条评论',
  'users.contentLoading': '正在加载帖子与评论…',
  'users.addPost': '新增帖子',
  'users.addComment': '新增评论',
  'users.postsColumn': '帖子',
  'users.commentsColumn': '评论',
  'users.noPosts': '暂无帖子',
  'users.noComments': '暂无评论',
  'users.postRef': '帖子 #{id}',
  'users.editRecordTitle': '编辑{kind} #{id}',
  'users.deleteRecordTitle': '删除{kind} #{id}',
  'users.deleteRecordMessage': '删除后无法恢复，相关的点赞与关联数据也会一并移除。要继续吗？',
  'users.contentLabel': '内容',
  'users.addPostTitle': '新增帖子',
  'users.addPostMessage': '此内容将以该用户的身份发布，作者字段无法伪造。',
  'users.postContentLabel': '帖子内容',
  'users.postContentPlaceholder': '输入帖子内容',
  'users.pickPostTitle': '选择帖子',
  'users.postIdLabel': '帖子 ID',
  'users.postIdPlaceholder': '要评论的帖子编号',
  'users.addCommentTitle': '新增评论',
  'users.commentContentLabel': '评论内容',
  'users.commentContentPlaceholder': '输入评论内容',
  'users.createTagTitle': '新增标签',
  'users.createTagMessage': '标签可用于给用户分类，例如「版主」「活跃」或「已封禁」。',
  'users.tagNameLabel': '标签名称',
  'users.tagNamePlaceholder': '最多 50 个字',
  'users.renameTagTitle': '重命名标签',
  'users.renameTagMessage': '所有使用该标签的用户都会看到新名称。',
  'users.deleteTagTitle': '删除标签「{name}」',
  'users.deleteTagMessage': '删除后，所有用户身上的该标签都会一并移除，且无法恢复。要继续吗？',
  'users.deleteTagConfirm': '删除标签',
  'users.tagCreated': '标签已创建。',
  'users.createTagFailed': '创建标签失败。',
  'users.tagUpdated': '标签已更新。',
  'users.updateTagFailed': '更新标签失败。',
  'users.tagDeleted': '标签已删除。',
  'users.deleteTagFailed': '删除标签失败。',
  'users.refreshDone': '已更新至最新数据。',
  'users.signinEyebrow': '{brand} ADMIN',
  'users.signinTitle': '后台管理',
  'users.signinBody': '集中式内容治理，让每一次审核都清晰、快速且可追踪。',
  'users.signinStep1': '身份验证',
  'users.signinStep2': '用户治理',
  'users.signinStep3': '内容审核',
  'users.signinPanelTitle': '登录管理控制台',
  'users.signinPanelBody': '管理控制台仅对已授权的 Google 管理员账号开放，请使用管理员身份登录。',

  /* ==========================================================================
     posts
     ========================================================================== */

  'posts.title': '论坛帖子',
  'posts.searching': '搜索中…',
  'posts.searchDegraded': '（搜索服务未启用，以数据库关键词比对）',
  'posts.searchSummary': '「{query}」找到 {total} 条，本页显示 {shown} 条',
  'posts.pendingCount': '{count} 条待处理',
  'posts.pageSummary': '第 {page} / {pages} 页，本页 {count} 篇',
  'posts.listLoadFailed': '文章列表加载失败。',
  'posts.eyebrow': 'POST MODERATION',
  'posts.copy': '创建、编辑与删除论坛帖子，并在评论层级处理内容治理。',
  'posts.toReports': '举报管理',
  'posts.editorTitleNew': '新增帖子',
  'posts.editorTitleEdit': '编辑帖子 #{id}',
  'posts.editorNote': '以管理员身份发布；作者取自登录身份，请求内容无法伪造作者。',
  'posts.cancelEdit': '取消编辑',
  'posts.contentLabel': '帖子内容',
  'posts.contentPlaceholder': '输入帖子内容',
  'posts.saveChanges': '保存修改',
  'posts.emptyContent': '帖子内容不能为空。',
  'posts.saving': '保存中…',
  'posts.saved': '帖子已更新。',
  'posts.published': '帖子已发布。',
  'posts.saveFailed': '保存失败。',
  'posts.deleteTitle': '删除帖子 #{id}',
  'posts.deleteMessage': '删除后无法恢复，该帖子下的所有评论也会一并移除。要继续吗？',
  'posts.deleteConfirm': '删除帖子',
  'posts.deleted': '帖子已删除。',
  'posts.deleteFailed': '删除帖子失败。',
  'posts.commentUpdated': '评论已更新。',
  'posts.commentActionFailed': '评论操作失败。',
  'posts.pickPostTitle': '选择评论所属的帖子',
  'posts.pickPostMessage': '默认是这一行的帖子；如果要挂到别的帖子，请改成对应的帖子编号。',
  'posts.postIdLabel': '帖子 ID',
  'posts.addCommentAtTitle': '在帖子 #{id} 新增评论',
  'posts.addCommentMessage': '此评论将以管理员身份发布。',
  'posts.commentContentLabel': '评论内容',
  'posts.commentContentPlaceholder': '输入评论内容',
  'posts.add': '新增',
  'posts.editCommentTitle': '编辑评论 #{id}',
  'posts.deleteCommentTitle': '删除评论 #{id}',
  'posts.deleteCommentMessage': '删除后无法恢复。要继续吗？',
  'posts.listTitle': '帖子列表',
  'posts.clearSearch': '清除搜索',
  'posts.searchLabel': '关键词搜索',
  'posts.searchPlaceholder': '帖子内容，或帖子作者的完整 Email',
  'posts.searchHint':
    '按相关度排序；填入完整 Email 可以找出该用户的所有帖子。搜索会取代分页，结果最多显示 25 条。',
  'posts.searchTotal': '搜索结果共 {total} 条',
  'posts.searchFailed': '搜索失败。',
  'posts.searchStatusFailed': '搜索失败',
  'posts.colContentImage': '内容与图片',
  'posts.colEngagement': '互动',
  'posts.colComments': '评论',
  'posts.colAuthor': '作者',
  'posts.imageAlt': '帖子附图',
  'posts.likes': '{count} 个赞',
  'posts.author': '作者：{name}',
  'posts.emptyTitle': '目前没有论坛帖子',
  'posts.emptyBody': '可以先用上方的编辑器创建第一个帖子。',
  'posts.emptySearchTitle': '没有匹配的帖子',
  'posts.emptySearchBody': '「{query}」没有匹配到任何帖子，换个关键词试试。',
  'posts.commentCount': '{count} 条评论',
  'posts.noComments': '暂无评论',
  'posts.reportsTitle': '待处理的举报',
  'posts.allReports': '全部举报',
  'posts.colReportedContent': '被举报内容',
  'posts.colReason': '举报原因',
  'posts.colReporter': '举报人',
  'posts.colTime': '时间',
  'posts.colVerdict': '裁定',
  'posts.emptyReportsTitle': '没有待处理的举报',
  'posts.emptyReportsBody': '所有举报都已裁定完成。',
  'posts.verdictResolved': '已处理',
  'posts.verdictRejected': '不成立',
  'posts.verdictDialogTitle': '将举报 #{id} 标为「{label}」',
  'posts.verdictDialogMessage': '标记后举报会离开待处理列表，但数据仍保留在举报管理页。要继续吗？',
  'posts.verdictConfirm': '标为{label}',
  'posts.verdictDone': '举报已标为{label}。',
  'posts.verdictFailed': '举报状态更新失败。',

  /* ==========================================================================
     reports
     ========================================================================== */

  'reports.title': '举报管理',
  'reports.listSummary': '{count} 条 · {filter}',
  'reports.listLoadFailed': '举报列表加载失败。',
  'reports.eyebrow': 'REPORT MODERATION',
  'reports.copy': '逐条裁定举报：「通过」会删除被举报的内容，「不成立」则保留原文。改回「待处理」会清除已有的裁定时间。',
  'reports.backToPosts': '返回帖子',
  'reports.editorTitle': '编辑举报 #{id}',
  'reports.editorNote': '可以补正举报原因与状态；目标与举报人是既有记录，不在这里更改。',
  'reports.targetTypeLabel': '目标类型',
  'reports.targetIdLabel': '目标 ID',
  'reports.reporterEmailLabel': '举报人 Email',
  'reports.statusLabel': '状态',
  'reports.reasonLabel': '举报原因',
  'reports.reasonHint': '最多 500 字，会直接显示给其他管理员作为裁定依据。',
  'reports.filterLabel': '按状态筛选',
  'reports.filterAll': '全部',
  'reports.listTitle': '举报列表',
  'reports.colTarget': '目标内容',
  'reports.colReason': '举报原因',
  'reports.colReporter': '举报人',
  'reports.colStatus': '状态',
  'reports.targetGone': '（内容已删除）',
  'reports.author': '作者：{name}',
  'reports.deleteTitle': '删除举报 #{id}',
  'reports.deleteMessage': '删除的是这条举报记录本身，被举报的内容不受影响，且无法恢复。要继续吗？',
  'reports.deleteConfirm': '删除举报',
  'reports.deleted': '举报已删除。',
  'reports.deleteFailed': '删除举报失败。',
  'reports.updated': '举报已更新。',
  'reports.saveFailed': '保存举报失败。',
  'reports.approveTitle': '通过举报 #{id}',
  'reports.approveGoneMessage': '被举报的{kind} #{id} 已经不存在，这一条只会被标为已处理。',
  'reports.approveMessage':
    '将永久删除被举报的{kind} #{id}（若为帖子，该帖子下的所有评论也会一并移除），并把这条举报标为已处理。要继续吗？',
  'reports.approveConfirm': '通过并删帖',
  'reports.approveGoneDone': '内容已不存在，举报已标为已处理。',
  'reports.approveDone': '已删帖并将举报标为已处理。',
  'reports.approveFailed': '通过举报失败。',
  'reports.approveTitleGone': '内容已删除，只标记举报',
  'reports.approveTitleFull': '删除被举报的内容并标为已处理',
  'reports.rejectTitle': '举报 #{id} 不成立',
  'reports.rejectMessage': '不成立代表被举报的内容无需处理，内容会原样保留。要继续吗？',
  'reports.rejectConfirm': '标为不成立',
  'reports.rejectDone': '举报已标为不成立。',
  'reports.statusFailed': '举报状态更新失败。',
  'reports.emptyTitle': '目前没有举报',
  'reports.emptyBody': '这个筛选条件下没有任何记录。',
  'reports.rejectTitleAttr': '保留内容，只将举报标为不成立',
  'kind.post': '帖子',
  'kind.comment': '评论',
  'reports.statusPending': '待处理',
  'reports.statusResolved': '已处理',
  'reports.statusRejected': '不成立',

  /* ==========================================================================
     title
     ========================================================================== */

  'title.forum': '{site}',
  'title.login': '登录｜{site}',
  'title.newPost': '新增帖子｜{site}',
  'title.profile': '个人资料｜{site}',
  'title.publicProfile': '公开个人资料｜{site}',
  'title.following': '关注｜{site}',
  'title.adminUsers': '用户管理｜{brand} 后台',
  'title.adminLogin': '登录｜{brand} 后台',
  'title.adminPosts': '论坛帖子｜{brand} 后台',
  'title.adminReports': '举报管理｜{brand} 后台',
};
