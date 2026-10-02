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
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1시간',
  'common.oneDay': '1일',
  'common.sevenDays': '7일',
  'common.thirtyDays': '30일',
  'common.oneYear': '1년',
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
  'admin.navMonitor': '시스템 모니터링',
  'admin.navLog': '작업 기록',
  'admin.navStats': '콘텐츠 추이',
  'admin.navExport': '내보내기와 일괄 작업',
  'admin.navSessions': '로그인과 세션',
  'admin.navBlocks': 'IP 차단 목록',
  'admin.navAnnouncements': '공지',
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

  'monitor.title': '시스템 모니터링',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': '서비스 상태, 요청량, 지연 시간 분포, 속도 제한 카운터를 실시간으로 확인합니다.',
  'monitor.refresh': '새로 고침',
  'monitor.refreshing': '읽는 중…',
  'monitor.autoRefresh': '자동 새로 고침',
  'monitor.autoRefreshOn': '{seconds}초마다 자동 새로 고침',
  'monitor.autoRefreshOff': '자동 새로 고침이 일시 중지되었습니다',
  'monitor.nextUpdate': '{seconds}초 후 갱신',
  'monitor.loadFailed': '모니터링 데이터를 불러오지 못했습니다.',
  'monitor.loadFailedHint': '관리자로 로그인되어 있고 백엔드가 계속 실행 중인지 확인하세요.',
  'monitor.pausedHint': '자동 새로 고침이 일시 중지되어 마지막으로 읽은 결과가 표시됩니다.',
  'monitor.visibilityPaused': '탭이 백그라운드에 있어 자동 새로 고침이 일시 중지되었습니다.',
  'monitor.lastUpdated': '{time} 업데이트',
  'monitor.probeTook': '의존성 탐지 {ms}밀리초',
  'monitor.unreachable': '백엔드가 응답하지 않습니다. 마지막으로 성공한 읽기 결과가 표시됩니다.',
  'monitor.depsTitle': '서비스 상태',
  'monitor.depsNote': '읽을 때마다 각 의존성을 실제로 한 번 탐지합니다. 의존성별 제한 시간은 2초이며 세 개를 병렬로 실행합니다.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': '검색 엔진',
  'monitor.stateOk': '정상',
  'monitor.stateDown': '연결할 수 없음',
  'monitor.stateDisabled': '사용 안 함',
  'monitor.depSearchFallback': 'ES_URL이 없어 검색은 MySQL 키워드 일치로 대체됩니다.',
  'monitor.depDisabled': 'Redis 클라이언트가 주입되지 않아 미디어 기능이 꺼져 있습니다.',
  'monitor.depLatency': '응답 {ms}밀리초',
  'monitor.depKeys': '키 {count}개',
  'monitor.depMemory': '메모리 {size}',
  'monitor.depPoolUsage': '연결 {inUse}/{open}(최대 {max})',
  'monitor.depPoolWait': '{count}번 대기, 총 {ms}밀리초',
  'monitor.depRedisPool': '적중 {hits}／미스 {misses}',
  'monitor.depEngineMysql': 'MySQL 키워드 일치',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': '요청 개요',
  'monitor.statUptime': '가동 시간',
  'monitor.statRequests': '전체 요청',
  'monitor.statErrorRate': '오류율',
  'monitor.statP95': 'P95 지연 시간',
  'monitor.statInFlight': '진행 중 요청',
  'monitor.statGoroutines': 'Goroutine',
  'monitor.statHeap': '힙 메모리',
  'monitor.statDbPool': '데이터베이스 연결',
  'monitor.statRateLimited': '속도 제한 차단',
  'monitor.statCountWithPeak': '최대 {peak}',
  'monitor.statCountWithInUse': '사용 중 {inUse}, 유휴 {idle}',
  'monitor.statBlockedSplit': '4xx {client}／5xx {server}',
  'monitor.goVersion': 'Go {version}・논리 {cpu}코어・GC {gc}회',
  'monitor.noData': '아직 요청이 없습니다.',
  'monitor.noDataBody': '서비스 시작 이후의 요청이 여기에 표시됩니다. 지금 이 영역은 비어 있습니다.',
  'monitor.timelineTitle': '최근 {minutes}분 트래픽',
  'monitor.timelineNote': '합계는 이번 프로세스 시작부터의 누적치이며 재시작하면 0이 됩니다. 지연 시간의 각 백분위수는 히스토그램 버킷의 상한값이라 떨어진 값만 나옵니다. 막대가 없는 분은 그 시간대에 트래픽이 없었다는 뜻입니다.',
  'monitor.timelineLive': '이번 프로세스',
  'monitor.timelineHistory': '재시작 이전',
  'monitor.timelineLegendVolume': '요청량',
  'monitor.timelineLegendError': '5xx 오류',
  'monitor.timelinePeak': '최대 {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': '데이터베이스에 사용할 수 있는 과거 집계가 없습니다. 시작 시 읽지 못했으면(테이블 없음 또는 권한 없음) 이번 프로세스만 표시됩니다.',
  'monitor.routesTitle': '경로별',
  'monitor.routesNote': '경로는 정규화됩니다(숫자와 이메일 주소가 :id로 바뀜). 따라서 같은 경로의 서로 다른 id는 합쳐서 계산합니다.',
  'monitor.colRoute': '경로',
  'monitor.colCount': '요청',
  'monitor.colAvg': '평균',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': '가장 느림',
  'monitor.colErrors': '오류',
  'monitor.routeOther': '기타(경로 수 상한 도달)',
  'monitor.clientsTitle': '출발지 주소',
  'monitor.clientsNote': '요청이 많은 순으로 정렬했습니다. X-Forwarded-For나 X-Real-IP에서 가져온 주소는 신뢰할 수 있는 프록시로 검증되지 않았습니다. 조치하기 전에 프록시가 이 헤더들을 덮어쓰도록 설정되어 있는지 확인하세요.',
  'monitor.noClients': '아직 추적된 출발지가 없습니다.',
  'monitor.noClientsBody': '모든 요청은 출발지 주소별로 기록됩니다. 지금 이 표는 비어 있습니다.',
  'monitor.clientsDropped': '출발지 수가 상한 {limit}에 도달해 가장 오래 관측되지 않은 주소 {count}개를 추적에서 제외했습니다. 이 문구는 “이 사람들만 왔다”는 뜻이 아니라 이 목록이 불완전하다는 뜻입니다.',
  'monitor.colIp': '주소',
  'monitor.colSource': '출처',
  'monitor.colRateLimited': '속도 제한',
  'monitor.colBanned': '차단으로 거부',
  'monitor.colLastRoute': '마지막 경로',
  'monitor.colActions': '작업',
  'monitor.colBlock': '차단',
  'monitor.blocking': '차단 중…',
  'monitor.sourcePeer': '연결 상대',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.blockTitle': '{ip} 차단',
  'monitor.blockMessage': '이 주소의 쓰기 요청(글·댓글·좋아요·신고·이미지 업로드·로그인 이동)이 {duration} 동안 거부됩니다. 읽기는 영향받지 않습니다. 차단할까요?',
  'monitor.blockReason': '모니터 페이지에서 차단',
  'monitor.blocked': '{ip} 차단했습니다',
  'monitor.blockFailed': '차단 작업에 실패했습니다.',
  'monitor.limitsTitle': '속도 제한기',
  'monitor.limitsNote': '각 그룹은 엔드포인트 비용에 맞춰 별도 할당량을 가집니다. 차단 횟수는 이번 프로세스 시작 이후의 429 총합입니다.',
  'monitor.colLimiter': '제한기',
  'monitor.colBudget': '할당량',
  'monitor.colAllowed': '허용',
  'monitor.colBlocked': '차단',
  'monitor.colTracked': '추적 중인 출발지',
  'monitor.colBlockedRate': '차단률',
  'monitor.limitContent': '콘텐츠 쓰기',
  'monitor.limitUpload': '이미지 업로드',
  'monitor.limitAuth': 'OAuth 로그인',
  'monitor.limitBudget': '{window}초당 {limit}회',
  'monitor.limitUnknown': '(알 수 없음)',
  'monitor.noLimits': '사용할 수 있는 속도 제한기가 없습니다.',
  'monitor.noLimitsBody': '속도 제한기가 아직 생성되지 않아 카운터를 가져올 수 없습니다.',

  'log.title': '작업 기록',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': '관리 화면에서 일어난 모든 변경을 확인합니다. 누가, 언제, 어떤 대상에, 어떤 항목을 무엇에서 무엇으로 바꿨는지.',
  'log.refresh': '새로 고침',
  'log.loadFailed': '작업 기록을 불러오지 못했습니다.',
  'log.empty': '조건에 맞는 작업이 없습니다.',
  'log.emptyBody': '조건을 넓히거나 이 기간에 관리 화면 작업이 정말 없었다는지 확인하세요.',
  'log.count': '총 {total}건 중 {from}–{to}건',
  'log.retention': '기록은 {days}일간 보관되며 백그라운드 작업으로 삭제됩니다. 이 페이지에는 기록을 삭제하는 버튼이 없습니다 — 자기 기록을 지울 수 있는 감사 로그는 감사 로그가 아닙니다.',
  'log.filterActor': '작업자',
  'log.filterAction': '작업',
  'log.filterTargetType': '리소스 종류',
  'log.filterFrom': '시작일',
  'log.filterTo': '종료일',
  'log.filterAll': '전체',
  'log.filterApply': '필터 적용',
  'log.filterReset': '필터 초기화',
  'log.filterTargetHint': '아무 행의 「대상」 을 클릭하면 그 대상에만 관련된 작업만 볼 수 있습니다.',
  'log.colTime': '시각',
  'log.colActor': '작업자',
  'log.colAction': '작업',
  'log.colTarget': '대상',
  'log.colChanges': '변경 내용',
  'log.colOrigin': '출처',
  'log.noChanges': '(항목 변경 없음)',
  'log.changedTo': '로 변경',
  'log.removed': '(삭제됨)',
  'log.created': '(신규)',
  'log.requestId': 'request {id}',
  'log.page': '{page} 페이지',
  'log.targetUser': '사용자',
  'log.targetPost': '글',
  'log.targetComment': '댓글',
  'log.targetReport': '신고',
  'log.targetTag': '태그',
  'log.targetSystem': '시스템',
  'log.actionUserSuspend': '차단',
  'log.actionUserReinstate': '복원',
  'log.actionUserTags': '태그 변경',
  'log.actionUserPost': '사용자 이름으로 글 작성',
  'log.actionUserComment': '사용자 이름으로 댓글 작성',
  'log.actionUserContent': '해당 사용자 글 삭제',
  'log.actionPostCreate': '글 작성',
  'log.actionPostUpdate': '글 수정',
  'log.actionPostDelete': '글 삭제',
  'log.actionCommentCreate': '댓글 작성',
  'log.actionCommentUpdate': '댓글 수정',
  'log.actionCommentDelete': '댓글 삭제',
  'log.actionReportCreate': '신고 작성',
  'log.actionReportResolve': '신고 인정',
  'log.actionReportReject': '신고 기각',
  'log.actionReportUpdate': '신고 수정',
  'log.actionReportDelete': '신고 삭제',
  'log.actionTagCreate': '태그 추가',
  'log.actionTagUpdate': '태그 이름 변경',
  'log.actionTagDelete': '태그 삭제',
  'log.fieldStatus': '계정 상태',
  'log.fieldContent': '내용',
  'log.fieldName': '이름',
  'log.fieldTags': '태그',
  'log.fieldReason': '사유',
  'log.fieldAuthorEmail': '작성자',
  'log.fieldReporterEmail': '신고자',
  'log.fieldTargetType': '리소스 종류',
  'log.fieldTargetId': '리소스 번호',
  'log.fieldPostId': '글 번호',
  'log.fieldCommentId': '댓글 번호',
  'log.fieldPostIdShort': '글',
  'log.fieldCommentIdShort': '댓글',
  'log.fieldTarget': '대상',
  'log.fieldAssignmentsRemoved': '연결 해제 수',
  'log.truncated': '잘림',

  'stats.title': '콘텐츠 추이',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': '매일 새로 생기는 사용자·글·댓글 수와, 지금 가장 활발한 글·태그·작성자를 확인합니다.',
  'stats.refresh': '새로 고침',
  'stats.loadFailed': '콘텐츠 통계를 불러오지 못했습니다.',
  'stats.window': '표시 기간',
  'stats.windowDays': '{days}일',
  'stats.windowClamped': '(최대 90일까지)',
  'stats.windowNote': '하루의 구분은 서버가 실행되는 환경의 로컬 시간대 기준입니다. 사이트가 UTC에 있고 관리자는 다른 시간대에 있다면 오늘 숫자가 짧아 보일 수 있습니다 — 그것은 시간대 차이이지 트래픽이 줄었다는 뜻이 아닙니다.',
  'stats.generatedAt': '통계 생성 시각 {time}',
  'stats.seriesTitle': '일별 신규',
  'stats.seriesNote': '세 그래프의 규모가 서로 다르므로 겹치지 않고 따로 표시합니다.',
  'stats.seriesUsers': '신규 사용자',
  'stats.seriesPosts': '신규 글',
  'stats.seriesComments': '신규 댓글',
  'stats.seriesEmpty': '이 기간에 데이터가 없습니다.',
  'stats.totalsTitle': '기간 내 합계',
  'stats.totalsNote': '이 기간에 추가된 양이며 사이트의 현재 합계가 아닙니다.',
  'stats.totalUsers': '신규 사용자',
  'stats.totalPosts': '신규 글',
  'stats.totalComments': '신규 댓글',
  'stats.totalLikes': '신규 좋아요',
  'stats.topPostsTitle': '인기 글',
  'stats.topPostsNote': '댓글 수와 좋아요 수의 합으로 정렬하며, 이 기간에 작성된 글만 집계합니다.',
  'stats.topTagsTitle': '인기 태그',
  'stats.topTagsNote': '부여된 사람 수로 정렬하며 기간 제한을 두지 않습니다. 태그는 사건이 아니라 신분 분류입니다.',
  'stats.topAuthorsTitle': '활발한 작성자',
  'stats.topAuthorsNote': '이 기간의 글 수로 정렬하며 댓글 수는 따로 표시합니다.',
  'stats.colExcerpt': '본문 요약',
  'stats.colEngagement': '반응',
  'stats.colPosts': '글',
  'stats.colComments': '댓글',
  'stats.colUsers': '인원',
  'stats.colAuthor': '작성자',
  'stats.empty': '이 기간에 데이터가 없습니다.',
  'stats.emptyBody': '기간을 늘리거나, 이 기간에 실제로 새 내용이 없었는지 확인하세요.',
  'stats.engagement': '댓글 {comments}·좋아요 {likes}',
  'stats.rank': '{rank}위',

  'export.title': '내보내기와 일괄 작업',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': '사이트 데이터를 CSV로 내보내어 대조하거나, 여러 계정을 한 번에 처리합니다.',
  'export.download': 'CSV 내려받기',
  'export.downloading': '준비 중…',
  'export.exportTitle': '내보내기',
  'export.exportNote': '각 내보내기는 최대 5만 행까지이며, 그를 넘는 부분은 최신 행만 포함됩니다. 세 가지 내보내기의 열은 관리 페이지의 열과 같아 바로 대조할 수 있습니다.',
  'export.exportUsers': '사용자 목록',
  'export.exportUsersNote': 'email, 상태, 생성·수정 시각, 글 수, 댓글 수.',
  'export.exportPosts': '글 목록',
  'export.exportPostsNote': '번호, 작성자, 본문 앞 200자, 생성 시각, 댓글 수, 좋아요 수.',
  'export.exportReports': '신고 목록',
  'export.exportReportsNote': '번호, 신고 대상, 신고자, 사유, 상태와 검토 기록.',
  'export.safety': '파일은 UTF-8 바이트 순서 표시로 시작하므로 Excel에서 열어도 글자가 깨지지 않습니다.',
  'export.safetyPrefix': '= + - @ 또는 보이지 않는 공백으로 시작하는 값 앞에는 작은따옴표가 붙습니다. 스프레드시트가 수식이 아니라 텍스트로 처리하도록 하기 위한 것이며, 의도적으로 남겨 둔 것입니다. 제거를 요청하면 안 됩니다.',
  'export.batchTitle': '일괄 작업',
  'export.batchNote': '‘사용자 관리’ 페이지에서 계정을 선택하면 일괄 작업 버튼이 활성화됩니다. 일괄 작업은 전체가 반영되거나 전체가 반영되지 않으며, 일부만 반영되는 결과는 없습니다.',
  'export.batchSuspend': '선택 계정 정지',
  'export.batchReinstate': '선택 계정 복구',
  'export.batchTags': '태그 일괄 지정',
  'export.batchTagsNote': '덮어쓰기 방식입니다. 보낸 목록이 곧 결과이며, 빈 목록은 모든 태그를 지우는 것과 같습니다.',
  'export.batchConfirm': '{count}개 계정에 ‘{action}’을 적용할까요?',
  'export.batchConfirmTags': '{count}개 계정의 태그를 {tags}(으)로 덮어쓸까요?',
  'export.batchTagsPicker': '태그 선택',
  'export.batchTagsNone': '태그 선택 안 함(모두 지우기)',
  'export.batchRunning': '처리 중…',
  'export.batchDone': '{updated}개 계정을 갱신했습니다',
  'export.batchDoneUnchanged': '그중 {unchanged}개는 이미 그 상태여서 변경되지 않았습니다',
  'export.batchSkipped': '{count}개 건너뜀',
  'export.batchMax': '한 번에 최대 200개 계정까지 처리합니다',
  'export.gotoUsers': '사용자 관리로 이동',
  'export.noSelection': '먼저 사용자 관리 페이지에서 계정을 선택하세요.',
  'export.selected': '{count}개 계정 선택됨',
  'export.clearSelection': '선택 해제',
  'export.selectionHint': '선택은 이 페이지에 유지되며, 페이지를 닫으면 사라집니다.',

  'session.title': '로그인과 세션',
  'session.eyebrow': 'SESSIONS',
  'session.copy': '아직 유효한 로그인 상태를 보고, 필요하면 계정의 모든 기기에서 강제 로그아웃합니다.',
  'session.refresh': '새로 고침',
  'session.loadFailed': '세션 목록을 불러오지 못했습니다.',
  'session.privacyTitle': '전체 토큰을 보여주지 않는 이유',
  'session.privacyNote': '토큰 자체가 로그인 자격 증명입니다. 관리자가 같은 세션인지 구분할 수 있도록 처음 8자만 표시하는데, 그만큼으로 누구도 해당 사용자로 로그인할 수 없습니다 — 이 페이지의 스크린샷을 받은 사람도 마찬가지입니다. 이는 의도적인 제한이며 아직 완성되지 않은 기능이 아닙니다.',
  'session.expireNote': '세션은 {hours}시간 동안 활동이 없으면 만료되며, 계속 활동하는 동안에는 자동으로 연장됩니다.',
  'session.filterEmail': '계정으로 필터링',
  'session.filterPlaceholder': '전체 이메일 주소',
  'session.search': '검색',
  'session.clearFilter': '지우기',
  'session.summary': '사이트 전체 {total}개 세션, {scanned}개의 키를 확인했습니다',
  'session.truncated': '스캔이 한도인 {scanned}개 키에서 중단되었으므로 이 목록은 완전하지 않습니다.',
  'session.empty': '현재 세션이 없습니다.',
  'session.emptyBody': '로그인한 사람이 없거나 모든 세션이 만료되었습니다.',
  'session.colUser': '계정',
  'session.colToken': '세션',
  'session.colCreated': '생성 시각',
  'session.colExpires': '만료 시각',
  'session.colRemaining': '남은 시간',
  'session.colActions': '동작',
  'session.unknown': '알 수 없음',
  'session.adminBadge': '관리자',
  'session.revoke': '강제 로그아웃',
  'session.revokeTitle': '{email} 강제 로그아웃',
  'session.revokeMessage': '이 계정의 현재 {count}개 세션이 즉시 모두 무효화되어 모든 기기의 로그인 상태가 지워집니다. 사용자는 다시 로그인해야 합니다. 계속할까요?',
  'session.revokeRunning': '취소 중…',
  'session.revokeDone': '세션 {count}개를 취소했습니다',
  'session.revokeNone': '이 계정에는 활성 세션이 없습니다',
  'session.revokeFailed': '로그아웃을 확인하지 못했습니다.',
  'session.revokeUnavailable': '로그아웃이 실패했다고 단정할 수는 없습니다. 스캔이 키 수 한도에 도달해 일부 세션에 도달하지 못했을 수 있습니다. 잠시 후 다시 시도하세요.',
  'session.titleColumnNote': '식별용 접두일 뿐, 로그인에는 사용할 수 없음',

  'block.title': 'IP 차단 목록',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': '남용이 확인된 출처 주소를 차단 목록에 넣습니다. 목록은 Redis에 있으므로 재시작이나 배포로 해제되지 않습니다.',
  'block.refresh': '새로 고침',
  'block.loadFailed': '차단 목록을 불러오지 못했습니다.',
  'block.unavailable': '이 사이트는 Redis에 연결되어 있지 않아 차단 기능이 활성화되어 있지 않습니다.',
  'block.unavailableNote': '차단 목록에는 세션·미디어 토큰과 같은 Redis가 필요합니다. 연결되면 이 페이지가 작동하기 시작합니다. 그전까지의 방어는 속도 제한뿐이며(프로세스 내이고 재시작하면 사라집니다), 그 외에는 없습니다.',
  'block.add': '차단',
  'block.addTitle': 'IP 주소 차단',
  'block.addMessage': '차단된 출처 주소는 기간이 끝날 때까지 모든 쓰기 요청(글·댓글·좋아요·신고·이미지 업로드·로그인 이동)을 거부당합니다. 읽기는 영향받지 않습니다.',
  'block.ipLabel': 'IP 주소',
  'block.ipPlaceholder': '203.0.113.9 또는 2001:db8::1',
  'block.durationLabel': '차단 기간',
  'block.reasonLabel': '사유',
  'block.reasonPlaceholder': '이 주소를 차단하는 이유(감사 기록에 남습니다)',
  'block.reasonHint': '사유는 감사 기록에만 남습니다. 차단된 사람에게는 보이지 않고, 공개 오류 메시지에도 나오지 않습니다.',
  'block.blocking': '차단 중…',
  'block.done': '{ip} 차단했습니다',
  'block.removed': '{ip} 차단을 해제했습니다',
  'block.removedNone': '{ip}는 원래 차단되어 있지 않았습니다',
  'block.failed': '차단 작업에 실패했습니다.',
  'block.unavailableService': '차단 목록을 쓸 수 없습니다(Redis 없음)',
  'block.colIp': 'IP',
  'block.colExpires': '만료',
  'block.colRemaining': '남은 시간',
  'block.colActions': '동작',
  'block.unblock': '차단 해제',
  'block.unblockTitle': '{ip} 차단 해제',
  'block.unblockMessage': '이 주소는 즉시 정상적인 쓰기 권한을 되찾습니다. 계속할까요?',
  'block.empty': '차단 목록이 비어 있습니다.',
  'block.emptyBody': '차단된 출처가 없습니다. 이것이 정상 상태입니다. 차단은 언제나 관리자의 결정이며, 시스템이 자동으로 차단하는 일은 없습니다.',
  'block.notAutoNote': '이 목록은 자동으로 채워지지 않습니다. 속도 제한을 넘긴 출처는 429만 받고 여기에 자동으로 추가되지 않습니다. 같은 출구가 사무실이나 NAT 전체일 수 있고, 자동 차단은 그들에게까지 피해를 주기 때문입니다.',
  'block.scopeNoteLabel': '범위',
  'block.notAutoNoteLabel': '자동 차단 없음',
  'block.maxNoteLabel': '최대 기간',
  'block.scopeNote': '차단은 쓰기 요청만 막습니다. 글·댓글·정적 자산 읽기는 영향받지 않으며 차단된 사람도 로그인하고 내용을 볼 수 있습니다. 의도적입니다. 읽기 엔드포인트는 의도적으로 속도 제한이 없습니다(그렇지 않으면 익명 방문자가 이용할 수 없습니다), 차단은 같은 범위를 덮습니다.',
  'block.maxNote': '차단 한 번은 최대 365일까지입니다. 더 긴 기간은 1년으로 잘립니다. 만료 시각이 숫자로 저장되므로 \'영구\'는 아무도 기억하지 않고 스스로 풀리지 않는 차단이 되기 때문입니다.',
  'block.count': '차단 {count}건',
  'block.ipInvalid': 'IP 형식이 올바르지 않습니다. IPv4 또는 IPv6 주소를 입력하세요. CIDR 범위는 지원하지 않습니다.',
  'announce.label': '사이트 공지',
  'announce.publicNote': '공지',
  'announce.closeAria': '이 공지 닫기',
  'announce.publishedOn': '{date} 게시',
  'announce.expiresOn': '{date} 만료',
  'announce.neverExpires': '만료 없음',
  'announce.pinnedBadge': '고정',
  'announce.title': '공지',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': '사이트 전체 상단에 공지를 하나 표시합니다. 한 번에 하나만 적용되며 새 공지를 게시하면 이전 공지는 자동으로 비활성화됩니다.',
  'announce.refresh': '새로 고침',
  'announce.loadFailed': '공지 목록을 불러오지 못했습니다.',
  'announce.new': '새 공지 게시',
  'announce.edit': '이 공지 수정',
  'announce.deactivate': '비활성화',
  'announce.reactivate': '다시 활성화',
  'announce.deleteNote': '공지는 삭제되지 않고 비활성화되기만 합니다. 이력을 남겨야 언제 누가 공개했는지 알 수 있습니다.',
  'announce.bodyLabel': '공지 내용',
  'announce.bodyPlaceholder': '예: 시스템이 목요일 02:00–04:00에 점검합니다.',
  'announce.bodyHint': '최대 300자. 일반 텍스트이며 줄바꿈이 유지됩니다.',
  'announce.activeLabel': '바로 표시',
  'announce.expiryLabel': '유효 기간',
  'announce.expiryNever': '자동으로 만료되지 않음',
  'announce.expiryHours': '{hours}시간 후',
  'announce.expiryDays': '{days}일 후',
  'announce.saving': '저장 중…',
  'announce.published': '공지를 게시했습니다',
  'announce.updated': '공지를 수정했습니다',
  'announce.deactivated': '공지를 비활성화했습니다',
  'announce.reactivated': '공지를 다시 활성화했습니다',
  'announce.saveFailed': '공지 작업에 실패했습니다.',
  'announce.empty': '아직 공지가 없습니다.',
  'announce.emptyBody': '게시하면 모든 방문자의 페이지 최상단에 표시됩니다.',
  'announce.colBody': '내용',
  'announce.colState': '상태',
  'announce.colAuthor': '게시자',
  'announce.colCreated': '게시 시각',
  'announce.colActions': '동작',
  'announce.stateActive': '표시 중',
  'announce.stateInactive': '비활성',
  'announce.stateExpired': '만료됨',
  'announce.confirmDeactivate': '게시하면 현재 공지가 비활성화되고 모든 방문자가 새 내용을 즉시 봅니다. 계속할까요?',
  'announce.confirmEdit': '이 공지의 내용이나 유효 기간을 수정할까요?',
  'announce.count': '총 {count}건',
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
  'posts.pin': '고정',
  'posts.unpin': '고정 해제',
  'posts.pinTitle': '이 글 고정하기',
  'posts.unpinTitle': '이 글 고정 해제하기',
  'posts.pinMessage': '고정하면 모든 방문자의 동적 맨 앞에 남아 새 글로 밀려나지 않습니다.',
  'posts.unpinMessage': '고정을 해제하면 시간순 위치로 돌아갑니다.',
  'posts.pinDone': '고정했습니다',
  'posts.unpinDone': '고정을 해제했습니다',
  'posts.pinFailed': '고정 작업에 실패했습니다.',
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
  'title.adminMonitor': '시스템 모니터링｜{brand} 관리자',
  'title.adminLog': '작업 기록｜{brand} 관리자',
  'title.adminStats': '콘텐츠 추이｜{brand} 관리자',
  'title.adminExport': '내보내기와 일괄 작업｜{brand} 관리자',
  'title.adminSessions': '로그인과 세션｜{brand} 관리자',
  'title.adminBlocks': 'IP 차단 목록｜{brand} 관리자',
  'title.adminAnnouncements': '공지｜{brand} 관리자',
};
