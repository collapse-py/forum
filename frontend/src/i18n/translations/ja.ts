/*
 * Japanese catalog (src/i18n/translations/ja.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 */

import type { MessageKey } from '../messages';

export const ja: Record<MessageKey, string> = {
  /* ==========================================================================
     common
     ========================================================================== */

  'common.cancel': 'キャンセル',
  'common.save': '保存',
  'common.submitting': '送信中...',
  'common.delete': '削除',
  'common.edit': '編集',
  'common.search': '検索',
  'common.loading': '読み込み中…',
  'common.loadFailed': '読み込みに失敗しました',
  'common.refresh': '更新',
  'common.nextStep': '次へ',
  'common.prevPage': '前へ',
  'common.nextPage': '次へ',
  'common.create': '作成',
  'common.publish': '投稿',
  'common.placeholder': '—',
  'common.backToHome': 'ホームに戻る',
  'common.backToForumHome': 'フォーラムへ戻る',
  'common.backOnePage': '前のページへ',

  /* ==========================================================================
     errors
     ========================================================================== */

  'error.request': '時間をおいて、もう一度お試しください。',
  'error.requestStatus': 'リクエストに失敗しました（HTTP {status}）',
  'error.loginRequired': '先にログインしてください。',
  'error.adminSessionExpired': 'ログインの有効期限が切れました。ログイン画面に戻ります。',
  'error.fallbackLoad': '読み込みに失敗しました',
  'error.fallbackSearch': '検索に失敗しました',
  'error.fallbackLike': 'いいねに失敗しました',
  'error.fallbackComments': 'コメントの読み込みに失敗しました',
  'error.fallbackCommentPost': 'コメントの送信に失敗しました',
  'error.fallbackReport': '通報に失敗しました',
  'error.fallbackProfile': 'プロフィールの読み込みに失敗しました',
  'error.fallbackProfileSave': '保存に失敗しました',
  'error.fallbackPublish': '投稿の形式が不正です',
  'error.fallbackUpload': '画像のアップロード形式が不正です',
  'error.fallbackNotFound': 'ユーザーが見つかりません',
  'error.fallbackFollow': 'フォローに失敗しました',

  /* ==========================================================================
     auth
     ========================================================================== */

  'auth.checking': 'ログイン状態を確認中...',
  'auth.statusUnknown': 'ログイン状態を確認できません',
  'auth.feedLoggedIn': 'ログイン済み・投稿できます',
  'auth.feedLoggedOut': 'ログインすると投稿できます',
  'auth.profileLoggedIn': 'ログイン済み',
  'auth.profileLoggedOut': 'ログインするとプロフィールを設定できます',
  'auth.googleLogin': 'Google でログイン',
  'auth.loginWithGoogle': 'Google アカウントでログイン',
  'auth.loginWithGoogleAdmin': 'Google 管理者アカウントでログイン',
  'auth.logout': 'ログアウト',

  /* ==========================================================================
     PWA インストール
     ========================================================================== */

  'install.button': 'アプリをインストール',
  'install.hint': '現在、ブラウザで自動インストールは提示されていません。ブラウザのメニューから「アプリをインストール」または「ホーム画面に追加」を選択してください。',

  /* ==========================================================================
     ボトムナビゲーション
     ========================================================================== */

  'bottomNav.label': 'メインナビゲーション',
  'bottomNav.home': 'ホーム',
  'bottomNav.new': '新規',
  'bottomNav.profile': 'プロフィール',

  /* ==========================================================================
     i18n
     ========================================================================== */

  'i18n.ariaLabel': '言語を選択',
  'i18n.current': '言語：{name}',

  /* ==========================================================================
     feed
     ========================================================================== */

  'feed.searchPlaceholder': '投稿内容を検索',
  'feed.searchAriaLabel': '投稿を検索',
  'feed.searchResultsLabel': '検索結果',
  'feed.postsLabel': '投稿',
  'feed.searchFailed': '検索に失敗しました。時間をおいてお試しください。',
  'feed.searching': '検索中...',
  'feed.searchMore': '検索結果をさらに読み込み中...',
  'feed.searchMoreFailed': '追加の読み込みに失敗しました',
  'feed.searchFound': '{total} 件',
  'feed.searchDegraded': '{base}（検索機能が無効なため、現在データベースのキーワード検索を使用しています）',
  'feed.searchTotal': '検索結果 {total} 件',
  'feed.searchNoResults': '「{query}」に一致する投稿はありません。',
  'feed.loadingPosts': '投稿を読み込み中...',
  'feed.loadMorePosts': '投稿をさらに読み込み中...',
  'feed.postsFailed': '投稿の読み込みに失敗しました。時間をおいてお試しください。',
  'feed.postsFailedShort': '読み込み失敗',
  'feed.scrollMore': '下へスクロールして読み込み',
  'feed.endOfFeed': 'すべて表示しました',
  'feed.noPosts': 'まだ投稿がありません。最初の思いを書いてみませんか。',
  'feed.likeFailed': 'いいねまたは取り消しが失敗しました。時間をおいてお試しください。',

  /* ==========================================================================
     post
     ========================================================================== */

  'post.authorAnonymous': '匿',
  'post.report': '投稿を通報',
  'post.imageAlt': '投稿の画像',
  'post.unlike': 'いいね取消',
  'post.like': 'いいね',
  'post.reply': '返信',

  /* ==========================================================================
     comment
     ========================================================================== */

  'comment.loading': 'コメントを読み込み中...',
  'comment.none': 'まだコメントはありません',
  'comment.loadFailed': 'コメントの読み込みに失敗しました。時間をおいてお試しください',
  'comment.placeholder': 'コメントを書く...',
  'comment.max': '最大 2000 文字',
  'comment.submit': 'コメントする',
  'comment.failed': 'コメントの送信に失敗しました。時間をおいてお試しください。',
  'comment.report': '通報',
  'comment.more': 'コメントをさらに読み込み中...',

  /* ==========================================================================
     report
     ========================================================================== */

  'report.reasonPlaceholder': '通報理由を入力（最大 500 文字）',
  'report.note': '通報は管理者に送信されます',
  'report.formLabel': '通報フォーム',
  'report.submit': '通報を送信',
  'report.failed': '通報に失敗しました。時間をおいてお試しください。',
  'report.sent': '通報を送信しました。ご報告ありがとうございます。',

  /* ==========================================================================
     newPost
     ========================================================================== */

  'newPost.avatarYou': 'あなた',
  'newPost.eyebrow': 'NEW POST',
  'newPost.title': '新規投稿',
  'newPost.loginFirst': 'Google でログインすると投稿できます',
  'newPost.contentPlaceholder': 'あなたの思いを書いてください...',
  'newPost.addImage': '画像を追加',
  'newPost.emailPrivate': 'あなたのメールアドレスは公開されません',
  'newPost.submit': '投稿する',
  'newPost.publishing': '投稿中...',
  'newPost.uploading': '画像をアップロード中...',
  'newPost.failed': '投稿に失敗しました。時間をおいてお試しください。',

  /* ==========================================================================
     profile
     ========================================================================== */

  'profile.eyebrow': 'YOUR PROFILE',
  'profile.title': 'プロフィール',
  'profile.edit': '編集',
  'profile.loginPrompt': 'ログインするとフォーラムのニックネームと自己紹介を設定できます。',
  'profile.nicknameLabel': 'フォーラムのニックネーム',
  'profile.notSet': '未設定',
  'profile.notSetBio': '自己紹介は未設定です。',
  'profile.nicknameInput': 'ニックネーム',
  'profile.nicknamePlaceholder': 'ニックネームを入力',
  'profile.nicknameHint': 'ニックネームは投稿と一緒に表示されます。最大 30 文字です。',
  'profile.bioLabel': '自己紹介',
  'profile.bioPlaceholder': '自己紹介を入力（任意）',
  'profile.bioHint': '最大 500 文字です。',
  'profile.updated': 'プロフィールを更新しました。',
  'profile.saving': '保存中...',
  'profile.saveFailed': '保存に失敗しました。時間をおいてお試しください。',
  'profile.loadFailed': '読み込みに失敗しました。時間をおいてお試しください。',
  'profile.followingEntry': 'マイフォロー',

  /* ==========================================================================
     publicProfile
     ========================================================================== */

  'publicProfile.eyebrow': 'PUBLIC PROFILE',
  'publicProfile.title': '公開プロフィール',
  'publicProfile.avatar': '匿',
  'publicProfile.loading': '読み込み中...',
  'publicProfile.invalidLinkName': '無効な公開プロフィールのリンクです',
  'publicProfile.invalidLinkBio': 'フォーラムの投稿に表示された作者名からプロフィールを開いてください。',
  'publicProfile.notFound': 'ユーザーが見つかりません',
  'publicProfile.anonymous': '匿名のユーザー',
  'publicProfile.noBio': 'このユーザーは公開プロフィールを設定していません。',
  'publicProfile.loadFailed': '公開プロフィールの読み込みに失敗しました。',
  'publicProfile.postsLabel': '投稿',
  'publicProfile.emptyPosts': 'このユーザーはまだ投稿していません。',

  'follow.label': 'このユーザーをフォロー',
  'follow.action': 'フォロー',
  'follow.actionDone': 'フォロー中',
  'follow.unfollow': 'フォロー解除',
  'follow.done': 'フォローしました。',
  'follow.failed': 'フォローに失敗しました。時間をおいてお試しください。',

  'following.peopleLabel': 'フォロー中のユーザー',
  'following.postsLabel': 'フォロー中のユーザーの投稿',
  'following.peopleLoading': 'フォロー一覧を読み込み中...',
  'following.emptyPeople': 'まだ誰もフォローしていません。投稿の「フォロー」ボタン、またはユーザーの公開プロフィールからフォローできます。',
  'following.emptyPosts': 'フォロー中のユーザーの投稿はまだありません。',
  'following.peopleFailed': 'フォロー一覧の読み込みに失敗しました。',
  'following.postsFailed': 'フォロー中のユーザーの投稿を読み込めませんでした。',

  /* ==========================================================================
     login
     ========================================================================== */

  'login.eyebrow': 'MEMBER ACCESS',
  'login.title': 'おかえりなさい',
  'login.body': '{site} は、思いを書き留めるすべての人のための場所です。登録フォームは不要です。Google アカウントがあれば、すぐに投稿を始められます。',
  'login.browseFirst': 'ホームを見る',

  /* ==========================================================================
     admin
     ========================================================================== */

  'admin.skipToMain': 'メインコンテンツへスキップ',
  'admin.railLabel': '管理メニュー',
  'admin.railBrandAria': '{site}のホーム',
  'admin.consoleName': 'Admin Console',
  'admin.railNavLabel': '主な機能',
  'admin.railGovernance': 'モデレーション',
  'admin.railMode': '管理者モード',
  'admin.railExit': 'フォーラムへ',
  'admin.topbarMenu': '管理メニューを切り替える',
  'admin.statusOnline': '接続良好',
  'admin.topbarForum': 'フォーラム',
  'admin.logoutFailed': 'ログアウトに失敗しました。時間をおいてお試しください。',
  'admin.navUsers': 'ユーザー管理',
  'admin.navPosts': '投稿',
  'admin.navReports': '通報管理',
  'admin.listLoadFailed': '読み込み失敗',
  'admin.dlgClose': 'ウィンドウを閉じる',
  'admin.dlgConfirm': '確認',
  'admin.dlgSave': '保存',
  'admin.dlgApplyTags': 'タグを適用',
  'admin.dlgNoTags': '適用できるタグがありません。下の「タグ管理」で先に追加してください。',

  /* ==========================================================================
     users
     ========================================================================== */

  'users.title': 'ユーザー管理',
  'users.contentAction': 'コンテンツ',
  'users.updateContentFailed': 'コンテンツの操作に失敗しました。',
  'users.updated': 'コンテンツを更新しました。',
  'users.eyebrow': 'USER MANAGEMENT',
  'users.copy': 'フォーラムのユーザーの活動統計、タグ、アカウント状態を確認し、停止や投稿ごとのモデレーションを行います。',
  'users.refresh': 'データ更新',
  'users.statTotal': 'ユーザー総数',
  'users.statActive': '有効',
  'users.statSuspended': '停止中',
  'users.statContent': '投稿／コメント総数',
  'users.count': '{count} 人',
  'users.tagsCount': '{count} 個のタグ',
  'users.loadFailed': 'ユーザーデータの読み込みに失敗しました。',
  'users.tagsLoadFailed': 'タグデータの読み込みに失敗しました。',
  'users.panelTitle': 'フォーラムユーザー',
  'users.tagsPanelTitle': 'ユーザータグ',
  'users.colUser': 'ユーザー',
  'users.colTags': 'タグ',
  'users.colStatus': '状態',
  'users.colPosts': '投稿',
  'users.colComments': 'コメント',
  'users.colLikes': 'いいね',
  'users.colLastActivity': '最終利用',
  'users.colActions': '操作',
  'users.nicknameUnset': 'ニックネーム未設定',
  'users.notSet': '未設定',
  'users.statusActive': '有効',
  'users.statusSuspended': '停止中',
  'users.emptyTitle': 'ユーザーデータがありません',
  'users.emptyBody': 'Google でログインしてフォーラムを利用したユーザーがいないため、この表示になります。',
  'users.tagsEmptyTitle': '現在タグがありません',
  'users.tagsEmptyBody': '先にタグを作成すると、ユーザーリストで対象者に適用できます。',
  'users.addTag': 'タグを追加',
  'users.colName': '名前',
  'users.colCreated': '作成日時',
  'users.colUpdated': '更新日時',
  'users.renameTag': '名前を変更',
  'users.suspend': '停止',
  'users.restore': '復帰',
  'users.statusDialogTitle': 'このユーザーを{action}します',
  'users.statusSuspendMessage': '{email} はフォーラムにログインできなくなります。既存の投稿とコメントは残ります。続行しますか？',
  'users.statusRestoreMessage': '{email} はログインと投稿を再開できます。続行しますか？',
  'users.userSuspended': 'ユーザーを停止しました。',
  'users.userRestored': 'ユーザーを復帰しました。',
  'users.updateStatusFailed': 'ユーザー状態の更新に失敗しました。',
  'users.editTagsTitle': 'タグを編集 · {user}',
  'users.editTagsMessage': '適用するタグにチェックを入れると、チェックを外したタグはこのユーザーから削除されます。',
  'users.tagsUpdated': 'ユーザータグを更新しました。',
  'users.updateTagsFailed': 'ユーザータグの更新に失敗しました。',
  'users.contentLoadFailed': 'コンテンツの読み込みに失敗しました。',
  'users.contentLoadFailedShort': 'コンテンツ読み込み失敗',
  'users.contentPanelTitle': 'ユーザーのコンテンツ',
  'users.contentCount': '{posts} 件の投稿 · {comments} 件のコメント',
  'users.contentLoading': '投稿とコメントを読み込み中…',
  'users.addPost': '投稿を追加',
  'users.addComment': 'コメントを追加',
  'users.postsColumn': '投稿',
  'users.commentsColumn': 'コメント',
  'users.noPosts': 'まだ投稿はありません',
  'users.noComments': 'まだコメントはありません',
  'users.postRef': '投稿 #{id}',
  'users.editRecordTitle': '{kind} #{id} を編集',
  'users.deleteRecordTitle': '{kind} #{id} を削除',
  'users.deleteRecordMessage': '削除すると元に戻せません。関連するいいねや関連データも削除されます。続行しますか？',
  'users.contentLabel': 'コンテンツ',
  'users.addPostTitle': '投稿を追加',
  'users.addPostMessage': 'このコンテンツはユーザーの名義で公開されます。作者欄は偽造できません。',
  'users.postContentLabel': '投稿内容',
  'users.postContentPlaceholder': '投稿内容を入力',
  'users.pickPostTitle': '投稿を選択',
  'users.postIdLabel': '投稿 ID',
  'users.postIdPlaceholder': 'コメントする投稿の番号',
  'users.addCommentTitle': 'コメントを追加',
  'users.commentContentLabel': 'コメント内容',
  'users.commentContentPlaceholder': 'コメント内容を入力',
  'users.createTagTitle': 'タグを追加',
  'users.createTagMessage': 'タグはユーザーを分類するために使えます。たとえば「モデレーター」「アクティブ」「ブロック済み」です。',
  'users.tagNameLabel': 'タグ名',
  'users.tagNamePlaceholder': '最大 50 文字',
  'users.renameTagTitle': 'タグ名を変更',
  'users.renameTagMessage': 'このタグを使用中のすべてのユーザーに新しい名前が表示されます。',
  'users.deleteTagTitle': 'タグ「{name}」を削除',
  'users.deleteTagMessage': '削除すると、このタグは全ユーザーから削除され、元に戻せません。続行しますか？',
  'users.deleteTagConfirm': 'タグを削除',
  'users.tagCreated': 'タグを作成しました。',
  'users.createTagFailed': 'タグの作成に失敗しました。',
  'users.tagUpdated': 'タグを更新しました。',
  'users.updateTagFailed': 'タグの更新に失敗しました。',
  'users.tagDeleted': 'タグを削除しました。',
  'users.deleteTagFailed': 'タグの削除に失敗しました。',
  'users.refreshDone': '最新のデータに更新しました。',
  'users.signinEyebrow': '{brand} ADMIN',
  'users.signinTitle': '管理画面',
  'users.signinBody': '一元的なモデレーションで、すべての審査をはっきり速く記録として残せます。',
  'users.signinStep1': '認証',
  'users.signinStep2': 'ユーザー',
  'users.signinStep3': '審査',
  'users.signinPanelTitle': '管理コンソールにログイン',
  'users.signinPanelBody': '管理コンソールは承認済みの Google 管理者アカウントのみ利用できます。管理者としてログインしてください。',

  /* ==========================================================================
     posts
     ========================================================================== */

  'posts.title': '投稿',
  'posts.searching': '検索中…',
  'posts.searchDegraded': '（検索サービスが無効のため、データベースのキーワード検索で代用しています）',
  'posts.searchSummary': '「{query}」で {total} 件該当、このページ {shown} 件',
  'posts.pendingCount': '未処理 {count} 件',
  'posts.pageSummary': '{pages} ページ中 {page} ページ目、このページ {count} 件',
  'posts.listLoadFailed': '投稿一覧の読み込みに失敗しました。',
  'posts.eyebrow': 'POST MODERATION',
  'posts.copy': 'フォーラムの投稿を作成・編集・削除し、コメントのモデレーションを行います。',
  'posts.toReports': '通報管理',
  'posts.editorTitleNew': '投稿を追加',
  'posts.editorTitleEdit': '投稿 #{id} を編集',
  'posts.editorNote': '管理者として公開されます。作者はログイン情報から取得されるため、作者名を偽造することはできません。',
  'posts.cancelEdit': '編集をキャンセル',
  'posts.contentLabel': '投稿内容',
  'posts.contentPlaceholder': '投稿内容を入力',
  'posts.saveChanges': '変更を保存',
  'posts.emptyContent': '投稿内容を空にすることはできません。',
  'posts.saving': '保存中…',
  'posts.saved': '投稿を更新しました。',
  'posts.published': '投稿を公開しました。',
  'posts.saveFailed': '保存に失敗しました。',
  'posts.deleteTitle': '投稿 #{id} を削除',
  'posts.deleteMessage': '削除すると元に戻せず、この投稿へのコメントもすべて削除されます。続行しますか？',
  'posts.deleteConfirm': '投稿を削除',
  'posts.deleted': '投稿を削除しました。',
  'posts.deleteFailed': '投稿の削除に失敗しました。',
  'posts.commentUpdated': 'コメントを更新しました。',
  'posts.commentActionFailed': 'コメントの操作に失敗しました。',
  'posts.pickPostTitle': 'コメントする投稿を選択',
  'posts.pickPostMessage': '既定ではこの行の投稿です。別の投稿に置く場合は、正しい投稿番号を入力してください。',
  'posts.postIdLabel': '投稿 ID',
  'posts.addCommentAtTitle': '投稿 #{id} にコメントを追加',
  'posts.addCommentMessage': 'このコメントは管理者として公開されます。',
  'posts.commentContentLabel': 'コメント内容',
  'posts.commentContentPlaceholder': 'コメント内容を入力',
  'posts.add': '追加',
  'posts.editCommentTitle': 'コメント #{id} を編集',
  'posts.deleteCommentTitle': 'コメント #{id} を削除',
  'posts.deleteCommentMessage': '削除すると元に戻せません。続行しますか？',
  'posts.listTitle': '投稿一覧',
  'posts.clearSearch': '検索をクリア',
  'posts.searchLabel': 'キーワード検索',
  'posts.searchPlaceholder': '投稿内容、または投稿者の完全なメールアドレス',
  'posts.searchHint': '関連度順に並びます。完全なメールアドレスを入力すると、そのユーザーの投稿をすべて見つけられます。検索はページングを置き換え、結果は最大 25 件です。',
  'posts.searchTotal': '検索 {total} 件',
  'posts.searchFailed': '検索に失敗しました。',
  'posts.searchStatusFailed': '検索失敗',
  'posts.colContentImage': '内容と画像',
  'posts.colEngagement': '反応',
  'posts.colComments': 'コメント',
  'posts.colAuthor': '作者',
  'posts.imageAlt': '投稿の添付画像',
  'posts.likes': '{count} いいね',
  'posts.author': '作者：{name}',
  'posts.emptyTitle': '現在、フォーラムに投稿はありません。',
  'posts.emptyBody': '上のエディターから最初の投稿を作成できます。',
  'posts.emptySearchTitle': '該当する投稿がありません',
  'posts.emptySearchBody': '「{query}」に一致する投稿はありません。別のキーワードでお試しください。',
  'posts.commentCount': '{count} 件のコメント',
  'posts.noComments': 'まだコメントはありません',
  'posts.reportsTitle': '未処理の通報',
  'posts.allReports': 'すべての通報',
  'posts.colReportedContent': '通報された内容',
  'posts.colReason': '通報理由',
  'posts.colReporter': '通報者',
  'posts.colTime': '日時',
  'posts.colVerdict': '判定',
  'posts.emptyReportsTitle': '未処理の通報はありません',
  'posts.emptyReportsBody': 'すべての通報は判定済みです。',
  'posts.verdictResolved': '処理済み',
  'posts.verdictRejected': '却下',
  'posts.verdictDialogTitle': '通報 #{id} を「{label}」にする',
  'posts.verdictDialogMessage': '設定すると、この通報は未処理リストから外れますが、データは通報管理ページにそのまま残ります。続行しますか？',
  'posts.verdictConfirm': '{label}にする',
  'posts.verdictDone': '通報を{label}にしました。',
  'posts.verdictFailed': '通報の状態更新に失敗しました。',

  /* ==========================================================================
     reports
     ========================================================================== */

  'reports.title': '通報管理',
  'reports.listSummary': '{count} 件 · {filter}',
  'reports.listLoadFailed': '通報一覧の読み込みに失敗しました。',
  'reports.eyebrow': 'REPORT MODERATION',
  'reports.copy': '通報を 1 件ずつ判定します。「承認」は通報された内容を削除し、「却下」は内容をそのまま残します。「未処理」に戻すと、確定した判定時刻が消去されます。',
  'reports.backToPosts': '投稿に戻る',
  'reports.editorTitle': '通報 #{id} を編集',
  'reports.editorNote': '通報理由と状態は修正できますが、対象と通報者は既存の記録のため変更できません。',
  'reports.targetTypeLabel': '対象の種類',
  'reports.targetIdLabel': '対象 ID',
  'reports.reporterEmailLabel': '通報者 Email',
  'reports.statusLabel': '状態',
  'reports.reasonLabel': '通報理由',
  'reports.reasonHint': '最大 500 文字です。他の管理者の判定根拠として直接表示されます。',
  'reports.filterLabel': '状態で絞り込む',
  'reports.filterAll': 'すべて',
  'reports.listTitle': '通報一覧',
  'reports.colTarget': '対象の内容',
  'reports.colReason': '通報理由',
  'reports.colReporter': '通報者',
  'reports.colStatus': '状態',
  'reports.targetGone': '（内容は削除済み）',
  'reports.author': '作者：{name}',
  'reports.deleteTitle': '通報 #{id} を削除',
  'reports.deleteMessage': '削除されるのはこの通報の記録自体で、通報された内容は影響を受けず、元にも戻せません。続行しますか？',
  'reports.deleteConfirm': '通報を削除',
  'reports.deleted': '通報を削除しました。',
  'reports.deleteFailed': '通報の削除に失敗しました。',
  'reports.updated': '通報を更新しました。',
  'reports.saveFailed': '通報の保存に失敗しました。',
  'reports.approveTitle': '通報 #{id} を承認',
  'reports.approveGoneMessage': '通報された{kind} #{id} はすでに存在しません。この通報は処理済みとしてのみ記録されます。',
  'reports.approveMessage': '通報された{kind} #{id} を完全に削除し（投稿の場合、その投稿へのコメントもすべて削除されます）、この通報を処理済みにします。続行しますか？',
  'reports.approveConfirm': '承認して削除',
  'reports.approveGoneDone': '内容はすでに存在しません。通報を処理済みにしました。',
  'reports.approveDone': '内容を削除し、通報を処理済みにしました。',
  'reports.approveFailed': '通報の承認に失敗しました。',
  'reports.approveTitleGone': '内容は削除済みのため、通報のみマーク',
  'reports.approveTitleFull': '通報された内容を削除して処理済みにする',
  'reports.rejectTitle': '通報 #{id} を却下',
  'reports.rejectMessage': '却下は、通報された内容を処理する必要がないことを意味し、内容はそのまま残ります。続行しますか？',
  'reports.rejectConfirm': '却下にする',
  'reports.rejectDone': '通報を却下にしました。',
  'reports.statusFailed': '通報の状態更新に失敗しました。',
  'reports.emptyTitle': '現在、通報はありません',
  'reports.emptyBody': 'この絞り込み条件に一致する記録はありません。',
  'reports.rejectTitleAttr': '内容を残し、通報のみ却下にする',
  'kind.post': '投稿',
  'kind.comment': 'コメント',
  'reports.statusPending': '未処理',
  'reports.statusResolved': '処理済み',
  'reports.statusRejected': '却下',

  /* ==========================================================================
     title
     ========================================================================== */

  'title.forum': '{site}',
  'title.login': 'ログイン｜{site}',
  'title.newPost': '新規投稿｜{site}',
  'title.profile': 'プロフィール｜{site}',
  'title.publicProfile': '公開プロフィール｜{site}',
  'title.following': 'フォロー｜{site}',
  'title.adminUsers': 'ユーザー管理｜{brand} 管理画面',
  'title.adminLogin': 'ログイン｜{brand} 管理画面',
  'title.adminPosts': '投稿｜{brand} 管理画面',
  'title.adminReports': '通報管理｜{brand} 管理画面',
};
