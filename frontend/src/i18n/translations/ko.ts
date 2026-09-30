/*
 * ko catalog (src/i18n/translations/ko.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const ko: Record<MessageKey, string> = {
  'common.cancel':
    '취소',
  'common.save':
    '저장',
  'common.submitting':
    '제출 중...',
  'common.delete':
    '삭제',
  'common.edit':
    '편집',
  'common.search':
    '검색',
  'common.loading':
    '로딩 중…',
  'common.loadFailed':
    '로딩 실패',
  'common.refresh':
    '새로 고침',
  'common.nextStep':
    '다음 단계',
  'common.prevPage':
    '이전 페이지',
  'common.nextPage':
    '다음 페이지',
  'common.create':
    '만들기',
  'common.publish':
    '게시',
  'common.placeholder':
    '—',
  'common.backToHome':
    '홈으로',
  'common.backToForumHome':
    '포럼 홈으로',
  'common.backOnePage':
    '이전 페이지로 돌아가기',
  'error.request':
    '잠시 후 다시 시도해 주세요.',
  'error.requestStatus':
    '요청 실패(HTTP {status})',
  'error.loginRequired':
    '로그인한 후 계속하세요.',
  'error.adminSessionExpired':
    '로그인 세션이 만료되어 로그인 페이지로 돌아가려 합니다.',
  'error.fallbackLoad':
    '로딩 실패',
  'error.fallbackSearch':
    '검색 실패',
  'error.fallbackLike':
    '좋아요 실패',
  'error.fallbackComments':
    '댓글 로딩 실패',
  'error.fallbackCommentPost':
    '댓글 작성 실패',
  'error.fallbackReport':
    '신고 실패',
  'error.fallbackProfile':
    '프로필 정보를 불러오지 못했습니다.',
  'error.fallbackProfileSave':
    '저장 실패',
  'error.fallbackPublish':
    '게시 응답 형식이 올바르지 않습니다',
  'error.fallbackUpload':
    '이미지 업로드 응답 형식이 올바르지 않습니다',
  'error.fallbackNotFound':
    '이 사용자를 찾을 수 없습니다',
  'error.fallbackFollow':
    '팔로우 실패',
  'auth.checking':
    '로그인 상태 확인 중...',
  'auth.statusUnknown':
    '로그인 상태를 확인할 수 없습니다',
  'auth.feedLoggedIn':
    '로그인했으므로 게시할 수 있습니다.',
  'auth.feedLoggedOut':
    '로그인하면 게시할 수 있습니다.',
  'auth.profileLoggedIn':
    '로그인했습니다.',
  'auth.profileLoggedOut':
    '로그인하면 프로필을 설정할 수 있습니다.',
  'auth.googleLogin':
    'Google 로그인',
  'auth.loginWithGoogle':
    'Google 계정으로 로그인',
  'auth.loginWithGoogleAdmin':
    'Google 관리자 계정으로 로그인',
  'auth.logout':
    '로그아웃',
  'install.button':
    '앱 설치',
  'install.hint':
    '현재 브라우저 자동 설치 안내가 제공되지 않습니다. 브라우저 메뉴에서 "앱 설치" 또는 "홈 화면에 추가"를 선택하세요.',
  'bottomNav.label':
    '주요 탐색',
  'bottomNav.home':
    '홈',
  'bottomNav.new':
    '새로 만들기',
  'bottomNav.profile':
    '프로필',
  'i18n.ariaLabel':
    '언어 선택',
  'i18n.current':
    '언어: {name}',
  'feed.searchPlaceholder':
    '게시글 내용 검색',
  'feed.searchAriaLabel':
    '게시글 검색',
  'feed.searchResultsLabel':
    '검색 결과',
  'feed.postsLabel':
    '포럼 게시글',
  'feed.searchFailed':
    '검색 실패, 잠시 후 다시 시도해 주세요.',
  'feed.searching':
    '검색 중...',
  'feed.searchMore':
    '추가 검색 결과 로딩...',
  'feed.searchMoreFailed':
    '추가 로딩 실패',
  'feed.searchFound':
    '{total}개 찾음',
  'feed.searchDegraded':
    '{base}(검색 서비스 활성화 안 됨, 현재 데이터베이스 키워드 대조 중)',
  'feed.searchTotal':
    '총 {total}개 결과',
  'feed.searchNoResults':
    '「{query}」에 해당하는 게시글을 찾지 못했습니다.',
  'feed.loadingPosts':
    '게시글 로딩 중...',
  'feed.loadMorePosts':
    '게시글 더 로딩...',
  'feed.postsFailed':
    '게시글 로딩 실패, 잠시 후 다시 시도해 주세요.',
  'feed.postsFailedShort':
    '로딩 실패, 잠시 후 다시 시도해 주세요',
  'feed.scrollMore':
    '아래로 밀어 더 불러오기',
  'feed.endOfFeed':
    '더 이상 없습니다',
  'feed.noPosts':
    '게시글이 없습니다. 첫 생각을 남겨 보세요.',
  'feed.likeFailed':
    '좋아요를 하거나 취소하는 데 실패했습니다.',
  'post.authorAnonymous':
    '익명',
  'post.report':
    '게시글 신고',
  'post.imageAlt':
    '게시글 이미지',
  'post.unlike':
    '좋아요 취소',
  'post.like':
    '좋아요',
  'post.reply':
    '답글',
  'comment.loading':
    '댓글 로딩 중...',
  'comment.none':
    '댓글 없음',
  'comment.loadFailed':
    '댓글 로딩 실패, 잠시 후 다시 시도해 주세요',
  'comment.placeholder':
    '댓글 작성...',
  'comment.max':
    '최대 2000자',
  'comment.submit':
    '댓글',
  'comment.failed':
    '댓글 작성 실패, 잠시 후 다시 시도해 주세요.',
  'comment.report':
    '신고',
  'comment.more':
    '댓글 더 로딩...',
  'report.reasonPlaceholder':
    '신고 사유 입력(최대 500자)',
  'report.note':
    '신고는 사이트 관리자에게 전송됩니다.',
  'report.formLabel':
    '신고 입력란',
  'report.submit':
    '신고 제출',
  'report.failed':
    '신고 실패, 잠시 후 다시 시도해 주세요.',
  'report.sent':
    '신고가 전송되었습니다. 신고해 주셔서 감사합니다.',
  'newPost.avatarYou':
    '너',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    '게시글 새로 만들기',
  'newPost.loginFirst':
    '게시하려면 먼저 Google 로 로그인하세요',
  'newPost.contentPlaceholder':
    '생각 공유...',
  'newPost.addImage':
    '이미지 추가',
  'newPost.emailPrivate':
    '당신의 Email은 공개되지 않습니다.',
  'newPost.submit':
    '게시',
  'newPost.publishing':
    '게시 중...',
  'newPost.uploading':
    '업로드 중...',
  'newPost.failed':
    '게시하지 못했습니다. 나중에 다시 시도해 주세요.',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    '프로필',
  'profile.edit':
    '편집',
  'profile.loginPrompt':
    '로그인한 뒤 포럼 닉네임과 자기소개를 설정할 수 있습니다.',
  'profile.nicknameLabel':
    '포럼 닉네임',
  'profile.notSet':
    '미설정',
  'profile.notSetBio':
    '자기소개를 아직 설정하지 않았습니다.',
  'profile.nicknameInput':
    '닉네임',
  'profile.nicknamePlaceholder':
    '닉네임 입력',
  'profile.nicknameHint':
    '닉네임은 게시글에 표시됩니다. 최대 30자입니다.',
  'profile.bioLabel':
    '자기소개',
  'profile.bioPlaceholder':
    '자기소개를 간략히 소개해 주세요(선택 사항)',
  'profile.bioHint':
    '최대 500자입니다.',
  'profile.updated':
    '프로필이 업데이트되었습니다.',
  'profile.saving':
    '저장 중...',
  'profile.saveFailed':
    '저장에 실패했습니다. 나중에 다시 시도해 주세요.',
  'profile.loadFailed':
    '불러오기에 실패했습니다. 나중에 다시 시도해 주세요.',
  'profile.followingEntry':
    '내 팔로우',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    '공개 프로필',
  'publicProfile.avatar':
    '익명',
  'publicProfile.loading':
    '불러오는 중...',
  'publicProfile.invalidLinkName':
    '잘못된 공개 프로필 링크입니다.',
  'publicProfile.invalidLinkBio':
    '공개 프로필은 포럼 게시글의 작성자 이름에서 이동해 주세요.',
  'publicProfile.notFound':
    '이 사용자를 찾을 수 없습니다.',
  'publicProfile.anonymous':
    '익명 사용자',
  'publicProfile.noBio':
    '이 사용자는 공개 프로필을 설정하지 않았습니다.',
  'publicProfile.loadFailed':
    '공개 프로필을 불러오지 못했습니다.',
  'publicProfile.postsLabel':
    '글',
  'publicProfile.emptyPosts':
    '이 사용자는 아직 글을 쓰지 않았습니다.',

  'follow.label':
    '이 사용자 팔로우',
  'follow.action':
    '팔로우',
  'follow.actionDone':
    '팔로잉 중',
  'follow.unfollow':
    '팔로우 취소',
  'follow.done':
    '팔로우했습니다.',
  'follow.failed':
    '팔로우하지 못했습니다. 나중에 다시 시도해 주세요.',

  'following.peopleLabel':
    '팔로우하는 사용자',
  'following.postsLabel':
    '팔로우하는 사용자의 글',
  'following.peopleLoading':
    '팔로우 목록을 불러오는 중...',
  'following.emptyPeople':
    '아직 아무도 팔로우하지 않았습니다. 글에서 ‘팔로우’를 누르거나 사용자의 공개 프로필에서 팔로우할 수 있습니다.',
  'following.emptyPosts':
    '팔로우한 사용자가 아직 글을 쓰지 않았습니다.',
  'following.peopleFailed':
    '팔로우 목록을 불러오지 못했습니다.',
  'following.postsFailed':
    '팔로우한 사용자의 글을 불러오지 못했습니다.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    '다시 오신 것을 환영합니다',
  'login.body':
    '{site}은 생각을 글로 남기는 모든 사람을 위한 공간입니다. 회원가입 양식이 필요하지 않습니다. Google 계정으로 게시글을 작성할 수 있습니다.',
  'login.browseFirst':
    '먼저 홈 보기',
  'admin.skipToMain':
    '주요 내용으로 이동',
  'admin.railLabel':
    '관리 메뉴',
  'admin.railBrandAria':
    '{site} 홈',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    '주요 기능',
  'admin.railGovernance':
    '거버넌스',
  'admin.railMode':
    '관리자 모드',
  'admin.railExit':
    '포럼으로 돌아가기',
  'admin.topbarMenu':
    '관리자 메뉴 전환',
  'admin.statusOnline':
    '정상 연결됨',
  'admin.topbarForum':
    '포럼',
  'admin.logoutFailed':
    '로그아웃에 실패했습니다. 나중에 다시 시도해 주세요.',
  'admin.navUsers':
    '사용자 관리',
  'admin.navPosts':
    '포럼 게시글',
  'admin.navReports':
    '신고 관리',
  'admin.listLoadFailed':
    '불러오기 실패',
  'admin.dlgClose':
    '닫기',
  'admin.dlgConfirm':
    '확인',
  'admin.dlgSave':
    '저장',
  'admin.dlgApplyTags':
    '태그 적용',
  'admin.dlgNoTags':
    '현재 적용할 수 있는 태그가 없으므로 아래 "태그 관리"에서 먼저 추가해 주세요.',
  'users.title':
    '사용자 관리',
  'users.contentAction':
    '내용 작업',
  'users.updateContentFailed':
    '내용 작업에 실패했습니다.',
  'users.updated':
    '내용이 업데이트되었습니다.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    '사용자의 게시 활동 통계, 태그 및 계정 상태를 확인하고 정지 및 게시글별 내용 관리를 처리합니다.',
  'users.refresh':
    '새로 고침',
  'users.statTotal':
    '사용자 총 수',
  'users.statActive':
    '활성',
  'users.statSuspended':
    '정지됨',
  'users.statContent':
    '게시글/댓글 총량',
  'users.count':
    '{count}명',
  'users.tagsCount':
    '{count}개 태그',
  'users.loadFailed':
    '사용자 데이터를 불러오지 못했습니다.',
  'users.tagsLoadFailed':
    '태그 데이터를 불러오지 못했습니다.',
  'users.panelTitle':
    '포럼 사용자',
  'users.tagsPanelTitle':
    '사용자 태그',
  'users.colUser':
    '사용자',
  'users.colTags':
    '태그',
  'users.colStatus':
    '상태',
  'users.colPosts':
    '게시글',
  'users.colComments':
    '댓글',
  'users.colLikes':
    '좋아요',
  'users.colLastActivity':
    '마지막 활동',
  'users.colActions':
    '작업',
  'users.nicknameUnset':
    '닉네임 미설정',
  'users.notSet':
    '미설정',
  'users.statusActive':
    '활성',
  'users.statusSuspended':
    '정지됨',
  'users.emptyTitle':
    '현재 사용자 데이터가 없습니다.',
  'users.emptyBody':
    'Google로 로그인한 사용자가 없을 때만 여기가 비어 있습니다.',
  'users.tagsEmptyTitle':
    '현재 태그가 없습니다.',
  'users.tagsEmptyBody':
    '태그를 먼저 만든 다음 사용자 목록에서 대상에 적용할 수 있습니다.',
  'users.addTag':
    '태그 추가',
  'users.colName':
    '이름',
  'users.colCreated':
    '생성 시간',
  'users.colUpdated':
    '업데이트 시간',
  'users.renameTag':
    '태그 이름 바꾸기',
  'users.suspend':
    '정지',
  'users.restore':
    '복구',
  'users.statusDialogTitle':
    '{action}사용자',
  'users.statusSuspendMessage':
    '{email} 사용자가 더 이상 포럼에 로그인할 수 없으며 기존 게시글과 댓글은 유지됩니다. 계속 진행하시겠습니까?',
  'users.statusRestoreMessage':
    '{email} 사용자가 로그인하고 게시글을 작성할 수 있는 권한을 회복합니다. 계속 진행하시겠습니까?',
  'users.userSuspended':
    '사용자가 정지되었습니다.',
  'users.userRestored':
    '사용자를 복구했습니다.',
  'users.updateStatusFailed':
    '사용자 상태 업데이트에 실패했습니다.',
  'users.editTagsTitle':
    '태그 편집 · {user}',
  'users.editTagsMessage':
    '적용할 태그를 선택하세요. 모두 선택 해제는 해당 사용자의 모든 태그 제거와 같습니다.',
  'users.tagsUpdated':
    '사용자 태그가 업데이트되었습니다.',
  'users.updateTagsFailed':
    '사용자 태그 업데이트에 실패했습니다.',
  'users.contentLoadFailed':
    '콘텐츠를 불러오지 못했습니다.',
  'users.contentLoadFailedShort':
    '콘텐츠를 불러오지 못했습니다.',
  'users.contentPanelTitle':
    '사용자 콘텐츠',
  'users.contentCount':
    '{posts}개 게시글 · {comments}개 댓글',
  'users.contentLoading':
    '게시글과 댓글을 불러오는 중…',
  'users.addPost':
    '게시글 추가',
  'users.addComment':
    '댓글 추가',
  'users.postsColumn':
    '게시글',
  'users.commentsColumn':
    '댓글',
  'users.noPosts':
    '게시글이 없습니다',
  'users.noComments':
    '댓글이 없습니다',
  'users.postRef':
    '게시글 #{id}',
  'users.editRecordTitle':
    '{kind} 편집 #{id}',
  'users.deleteRecordTitle':
    '{kind} 삭제 #{id}',
  'users.deleteRecordMessage':
    '삭제하면 되돌릴 수 없으며 관련 좋아요와 연결된 데이터도 함께 제거됩니다. 계속 진행하시겠습니까?',
  'users.contentLabel':
    '콘텐츠',
  'users.addPostTitle':
    '게시글 추가',
  'users.addPostMessage':
    '이 콘텐츠는 해당 사용자의 신원으로 게시되며 작성자 필드는 조작할 수 없습니다.',
  'users.postContentLabel':
    '게시글 내용',
  'users.postContentPlaceholder':
    '게시글 내용 입력',
  'users.pickPostTitle':
    '게시글 선택',
  'users.postIdLabel':
    '게시글 ID',
  'users.postIdPlaceholder':
    '댓글을 달 게시글 번호',
  'users.addCommentTitle':
    '댓글 추가',
  'users.commentContentLabel':
    '댓글 내용',
  'users.commentContentPlaceholder':
    '댓글 내용 입력',
  'users.createTagTitle':
    '태그 추가',
  'users.createTagMessage':
    '태그는 사용자를 분류하는 데 사용할 수 있으며 예: \'모데레이터\' \'활발한 사용자\' \'차단된 사용자\'.',
  'users.tagNameLabel':
    '태그 이름',
  'users.tagNamePlaceholder':
    '최대 50자',
  'users.renameTagTitle':
    '태그 이름 변경',
  'users.renameTagMessage':
    '이 태그를 사용하는 모든 사용자가 새 이름을 보게 됩니다.',
  'users.deleteTagTitle':
    '\'{name}\' 태그 삭제',
  'users.deleteTagMessage':
    '삭제하면 모든 사용자가 가진 이 태그가 함께 제거되며 되돌릴 수 없습니다. 계속 진행하시겠습니까?',
  'users.deleteTagConfirm':
    '태그 삭제',
  'users.tagCreated':
    '태그가 생성되었습니다.',
  'users.createTagFailed':
    '태그 생성에 실패했습니다.',
  'users.tagUpdated':
    '태그가 업데이트되었습니다.',
  'users.updateTagFailed':
    '태그 업데이트에 실패했습니다.',
  'users.tagDeleted':
    '태그가 삭제되었습니다.',
  'users.deleteTagFailed':
    '태그 삭제에 실패했습니다.',
  'users.refreshDone':
    '최신 데이터로 업데이트되었습니다.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    '관리자 화면',
  'users.signinBody':
    '중앙집중식 콘텐츠 통제로 모든 검사를 명확하고 빠르게 추적할 수 있습니다.',
  'users.signinStep1':
    '안전 확인',
  'users.signinStep2':
    '사용자 통제',
  'users.signinStep3':
    '콘텐츠 검사',
  'users.signinPanelTitle':
    'Admin Console 로그인',
  'users.signinPanelBody':
    'Admin Console은 권한이 있는 Google 관리자 계정만 사용할 수 있으며 관리자 자격으로 로그인하세요.',
  'posts.title':
    '포럼 게시글',
  'posts.searching':
    '검색 중…',
  'posts.searchDegraded':
    '（검색 서비스를 사용할 수 없어 데이터베이스 키워드로 검색합니다）',
  'posts.searchSummary':
    '\'{query}\'에서 {total}건을 찾았고 이 페이지에는 {shown}건을 표시합니다',
  'posts.pendingCount':
    '{count}건 처리 대기',
  'posts.pageSummary':
    '{page}/{pages}페이지, 이 페이지 {count}개',
  'posts.listLoadFailed':
    '게시글 목록을 불러오지 못했습니다.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    '포럼 게시글을 만들고 편집하고 삭제하며 댓글 단계에서 콘텐츠를 관리합니다.',
  'posts.toReports':
    '신고 관리',
  'posts.editorTitleNew':
    '게시글 추가',
  'posts.editorTitleEdit':
    '게시글 #{id} 편집',
  'posts.editorNote':
    '관리자 자격으로 게시합니다. 작성자는 로그인한 신원이며 요청으로 조작할 수 없습니다.',
  'posts.cancelEdit':
    '편집 취소',
  'posts.contentLabel':
    '게시글 내용',
  'posts.contentPlaceholder':
    '게시글 내용 입력',
  'posts.saveChanges':
    '변경 사항 저장',
  'posts.emptyContent':
    '게시글 내용은 비워둘 수 없습니다.',
  'posts.saving':
    '저장 중…',
  'posts.saved':
    '게시글이 업데이트되었습니다.',
  'posts.published':
    '게시글이 게시되었습니다.',
  'posts.saveFailed':
    '저장에 실패했습니다.',
  'posts.deleteTitle':
    '게시글 #{id} 삭제',
  'posts.deleteMessage':
    '삭제하면 되돌릴 수 없으며 해당 게시글 아래의 모든 댓글도 함께 제거됩니다. 계속 진행하시겠습니까?',
  'posts.deleteConfirm':
    '게시글 삭제',
  'posts.deleted':
    '게시글이 삭제되었습니다.',
  'posts.deleteFailed':
    '게시글 삭제에 실패했습니다.',
  'posts.commentUpdated':
    '댓글이 업데이트되었습니다.',
  'posts.commentActionFailed':
    '댓글 작업에 실패했습니다.',
  'posts.pickPostTitle':
    '댓글이 속한 게시글 선택',
  'posts.pickPostMessage':
    '기본값은 현재 행의 게시글입니다. 다른 게시글에 연결하려면 게시글 번호를 바꾸세요.',
  'posts.postIdLabel':
    '게시글 ID',
  'posts.addCommentAtTitle':
    '게시글 #{id}에 댓글 추가',
  'posts.addCommentMessage':
    '이 댓글은 관리자 자격으로 게시됩니다.',
  'posts.commentContentLabel':
    '댓글 내용',
  'posts.commentContentPlaceholder':
    '댓글 내용 입력',
  'posts.add':
    '추가',
  'posts.editCommentTitle':
    '댓글 #{id} 편집',
  'posts.deleteCommentTitle':
    '댓글 #{id} 삭제',
  'posts.deleteCommentMessage':
    '삭제하면 되돌릴 수 없습니다. 계속 진행하시겠습니까?',
  'posts.listTitle':
    '게시글 목록',
  'posts.clearSearch':
    '검색 지우기',
  'posts.searchLabel':
    '키워드 검색',
  'posts.searchPlaceholder':
    '글 내용 또는 작성자의 전체 Email',
  'posts.searchHint':
    '관련성순으로 정렬됩니다. 전체 Email을 입력하면 해당 사용자의 모든 글을 찾을 수 있습니다. 검색 기능으로 탭이 대체되며 결과는 최대 25건까지 표시됩니다.',
  'posts.searchTotal':
    '검색 결과 {total}건',
  'posts.searchFailed':
    '검색에 실패했습니다.',
  'posts.searchStatusFailed':
    '검색에 실패했습니다',
  'posts.colContentImage':
    '콘텐츠 및 이미지',
  'posts.colEngagement':
    '상호작용',
  'posts.colComments':
    '댓글',
  'posts.colAuthor':
    '작성자',
  'posts.imageAlt':
    '게시글 이미지 대체 텍스트',
  'posts.likes':
    '{count} 좋아요',
  'posts.author':
    '작성자: {name}',
  'posts.emptyTitle':
    '현재 포럼 게시글이 없습니다',
  'posts.emptyBody':
    '위의 편집기로 첫 게시글을 작성할 수 있습니다.',
  'posts.emptySearchTitle':
    '일치하는 게시글이 없습니다',
  'posts.emptySearchBody':
    '「{query}」에 일치하는 게시글이 없습니다. 다른 키워드를 시도해 보세요.',
  'posts.commentCount':
    '댓글 {count}개',
  'posts.noComments':
    '댓글이 아직 없습니다',
  'posts.reportsTitle':
    '처리 대기 중인 신고',
  'posts.allReports':
    '모든 신고',
  'posts.colReportedContent':
    '신고된 콘텐츠',
  'posts.colReason':
    '신고 사유',
  'posts.colReporter':
    '신고자',
  'posts.colTime':
    '시간',
  'posts.colVerdict':
    '처리 결과',
  'posts.emptyReportsTitle':
    '처리 대기 중인 신고가 없습니다',
  'posts.emptyReportsBody':
    '모든 신고가 처리되었습니다.',
  'posts.verdictResolved':
    '처리됨',
  'posts.verdictRejected':
    '사실 아님',
  'posts.verdictDialogTitle':
    '신고 #{id}를「{label}」으로 표시',
  'posts.verdictDialogMessage':
    '표시하면 신고가 처리 대기 목록에서 사라지지만 데이터는 신고 관리 페이지에 유지됩니다. 계속 진행하시겠습니까?',
  'posts.verdictConfirm':
    '{label}으로 표시',
  'posts.verdictDone':
    '신고가 {label}으로 처리되었습니다.',
  'posts.verdictFailed':
    '신고 상태 업데이트에 실패했습니다.',
  'reports.title':
    '신고 관리',
  'reports.listSummary':
    '{count}건 · {filter}',
  'reports.listLoadFailed':
    '신고 목록을 불러오지 못했습니다.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    '신고를 건별로 처리합니다.「처리됨」은 신고된 콘텐츠를 삭제하고「사실 아님」은 원문을 유지합니다.「처리 대기」로 되돌리면 기존 처리 시간이 지워집니다.',
  'reports.backToPosts':
    '게시글로 돌아가',
  'reports.editorTitle':
    '신고 #{id} 편집',
  'reports.editorNote':
    '신고 사유와 상태를 보완할 수 있습니다. 대상과 신고자는 기존 기록이며 이 페이지에서 변경하지 않습니다.',
  'reports.targetTypeLabel':
    '대상 유형',
  'reports.targetIdLabel':
    '대상 ID',
  'reports.reporterEmailLabel':
    '신고자 Email',
  'reports.statusLabel':
    '상태',
  'reports.reasonLabel':
    '신고 사유',
  'reports.reasonHint':
    '최대 500자. 다른 관리자 처리 근거로 직접 표시됩니다.',
  'reports.filterLabel':
    '상태로 필터링',
  'reports.filterAll':
    '전체',
  'reports.listTitle':
    '신고 목록',
  'reports.colTarget':
    '대상 콘텐츠',
  'reports.colReason':
    '신고 사유',
  'reports.colReporter':
    '신고자',
  'reports.colStatus':
    '상태',
  'reports.targetGone':
    '(콘텐츠 삭제됨)',
  'reports.author':
    '작성자: {name}',
  'reports.deleteTitle':
    '신고 #{id} 삭제',
  'reports.deleteMessage':
    '삭제 대상은 이 신고 기록 자체이며 신고된 콘텐츠에는 영향을 주지 않습니다. 또한 되돌릴 수 없습니다. 계속 진행하시겠습니까?',
  'reports.deleteConfirm':
    '신고 삭제',
  'reports.deleted':
    '신고가 삭제되었습니다.',
  'reports.deleteFailed':
    '신고 삭제에 실패했습니다.',
  'reports.updated':
    '신고가 업데이트되었습니다.',
  'reports.saveFailed':
    '신고 저장에 실패했습니다.',
  'reports.approveTitle':
    '신고 #{id} 처리',
  'reports.approveGoneMessage':
    '신고된 {kind} #{id}가 더 이상 존재하지 않으므로 이 신고만 처리된 것으로 표시됩니다.',
  'reports.approveMessage':
    '신고된 {kind} #{id}를 영구적으로 삭제하고(게시글인 경우 해당 게시글의 모든 댓글도 함께 삭제) 이 신고를 처리된 것으로 표시합니다. 계속 진행하시겠습니까?',
  'reports.approveConfirm':
    '처리하고 게시글 삭제',
  'reports.approveGoneDone':
    '콘텐츠가 존재하지 않으므로 신고가 처리된 것으로 표시되었습니다.',
  'reports.approveDone':
    '게시글을 삭제하고 신고를 처리된 것으로 표시했습니다.',
  'reports.approveFailed':
    '신고 처리에 실패했습니다.',
  'reports.approveTitleGone':
    '콘텐츠가 삭제되었으므로 신고만 표시됩니다',
  'reports.approveTitleFull':
    '신고된 콘텐츠를 삭제하고 처리됨으로 표시',
  'reports.rejectTitle':
    '신고 #{id} 사실 아님',
  'reports.rejectMessage':
    '사실 아님으로 표시하면 신고된 콘텐츠에 대한 조치가 필요하지 않으며 원문 그대로 유지됩니다. 계속 진행하시겠습니까?',
  'reports.rejectConfirm':
    '사실 아님으로 표시',
  'reports.rejectDone':
    '신고가 사실 아님으로 표시되었습니다.',
  'reports.statusFailed':
    '신고 상태 업데이트에 실패했습니다.',
  'reports.emptyTitle':
    '현재 신고가 없습니다',
  'reports.emptyBody':
    '이 필터 조건에 따른 기록이 없습니다.',
  'reports.rejectTitleAttr':
    '콘텐츠 유지 및 신고 사실 아님 표시',
  'kind.post':
    '게시글',
  'kind.comment':
    '댓글',
  'reports.statusPending':
    '처리 대기',
  'reports.statusResolved':
    '처리됨',
  'reports.statusRejected':
    '사실 아님',
  'title.forum':
    '{site}',
  'title.login':
    '로그인｜{site}',
  'title.newPost':
    '새 게시글 작성｜{site}',
  'title.profile':
    '프로필｜{site}',
  'title.publicProfile':
    '공개 프로필｜{site}',
  'title.following':
    '팔로우｜{site}',
  'title.adminUsers':
    '사용자 관리｜{brand} 관리자 화면',
  'title.adminLogin':
    '로그인｜{brand} 관리자 화면',
  'title.adminPosts':
    '포럼 게시글｜{brand} 관리자 화면',
  'title.adminReports':
    '신고 관리｜{brand} 관리자 화면',
};
