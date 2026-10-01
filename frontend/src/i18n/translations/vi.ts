/*
 * vi catalog (src/i18n/translations/vi.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const vi: Record<MessageKey, string> = {
  'common.cancel':
    'Hủy',
  'common.save':
    'Lưu',
  'common.submitting':
    'Đang gửi...',
  'common.delete':
    'Xóa',
  'common.edit':
    'Sửa',
  'common.search':
    'Tìm kiếm',
  'common.loading':
    'Đang tải…',
  'common.loadFailed':
    'Tải không thành công',
  'common.refresh':
    'Làm mới',
  'common.nextStep':
    'Bước tiếp theo',
  'common.prevPage':
    'Trang trước',
  'common.nextPage':
    'Trang tiếp theo',
  'common.create':
    'Tạo',
  'common.publish':
    'Đăng',
  'common.placeholder':
    '—',
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 giờ',
  'common.oneDay': '1 ngày',
  'common.sevenDays': '7 ngày',
  'common.thirtyDays': '30 ngày',
  'common.oneYear': '1 năm',
  'common.backToHome':
    'Về trang chủ',
  'common.backToForumHome':
    'Về trang chủ diễn đàn',
  'common.backOnePage':
    'Quay lại trang trước',
  'error.request':
    'Xin vui lòng thử lại sau.',
  'error.requestStatus':
    'Yêu cầu không thành công (HTTP {status})',
  'error.loginRequired':
    'Xin hãy đăng nhập trước rồi tiếp tục.',
  'error.adminSessionExpired':
    'Trạng thái đăng nhập đã hết hạn, bạn sẽ sớm được đưa về trang đăng nhập.',
  'error.fallbackLoad':
    'Tải không thành công',
  'error.fallbackSearch':
    'Tìm kiếm không thành công',
  'error.fallbackLike':
    'Thích hoặc bỏ thích không thành công',
  'error.fallbackComments':
    'Tải bình luận không thành công',
  'error.fallbackCommentPost':
    'Gửi bình luận không thành công',
  'error.fallbackReport':
    'Báo cáo không thành công',
  'error.fallbackProfile':
    'Đã đọc hồ sơ cá nhân không thành công',
  'error.fallbackProfileSave':
    'Lưu không thành công',
  'error.fallbackPublish':
    'Định dạng phản hồi khi đăng không hợp lệ',
  'error.fallbackUpload':
    'Định dạng phản hồi khi tải lên ảnh không hợp lệ',
  'error.fallbackNotFound':
    'Không tìm thấy người dùng này',
  'error.fallbackFollow':
    'Không thể theo dõi',
  'auth.checking':
    'Đang kiểm tra trạng thái đăng nhập...',
  'auth.statusUnknown':
    'Không thể xác nhận trạng thái đăng nhập',
  'auth.feedLoggedIn':
    'Đã đăng nhập, bạn có thể đăng bài',
  'auth.feedLoggedOut':
    'Hãy đăng nhập trước rồi đăng bài',
  'auth.profileLoggedIn':
    'Đã đăng nhập',
  'auth.profileLoggedOut':
    'Hãy đăng nhập trước rồi cài đặt hồ sơ cá nhân',
  'auth.googleLogin':
    'Đăng nhập bằng Google',
  'auth.loginWithGoogle':
    'Đăng nhập bằng tài khoản Google',
  'auth.loginWithGoogleAdmin':
    'Đăng nhập bằng tài khoản quản trị viên Google',
  'auth.logout':
    'Đăng xuất',
  'install.button':
    'Cài đặt ứng dụng',
  'install.hint':
    'Trình duyệt hiện tại không có tùy chọn cài đặt tự động. Hãy mở trình đơn trình duyệt, chọn «Cài đặt ứng dụng» hoặc «Thêm vào màn hình chính».',
  'bottomNav.label':
    'Điều hướng chính',
  'bottomNav.home':
    'Trang chủ',
  'bottomNav.new':
    'Thêm',
  'bottomNav.profile':
    'Cá nhân',
  'i18n.ariaLabel':
    'Chọn ngôn ngữ',
  'i18n.current':
    'Ngôn ngữ: {name}',
  'feed.searchPlaceholder':
    'Tìm kiếm nội dung bài đăng',
  'feed.searchAriaLabel':
    'Tìm kiếm bài đăng',
  'feed.searchResultsLabel':
    'Kết quả tìm kiếm',
  'feed.postsLabel':
    'Bài viết diễn đàn',
  'feed.searchFailed':
    'Tìm kiếm không thành công, xin vui lòng thử lại sau.',
  'feed.searching':
    'Đang tìm kiếm...',
  'feed.searchMore':
    'Tải thêm kết quả tìm kiếm...',
  'feed.searchMoreFailed':
    'Tải thêm không thành công',
  'feed.searchFound':
    'Đã tìm thấy {total} bài',
  'feed.searchDegraded':
    '{base} (Dịch vụ tìm kiếm chưa được bật, hiện đang so khớp từ khóa trong cơ sở dữ liệu)',
  'feed.searchTotal':
    'Tổng cộng {total} kết quả',
  'feed.searchNoResults':
    'Không tìm thấy bài đăng chứa «{query}».',
  'feed.loadingPosts':
    'Đang tải bài viết...',
  'feed.loadMorePosts':
    'Tải thêm bài viết...',
  'feed.postsFailed':
    'Tải bài viết không thành công, xin vui lòng thử lại sau.',
  'feed.postsFailedShort':
    'Tải không thành công, xin vui lòng thử lại',
  'feed.scrollMore':
    'Vuốt xuống để tải thêm',
  'feed.endOfFeed':
    'Đã đến cuối danh sách',
  'feed.noPosts':
    'Chưa có bài viết nào, hãy để lại suy nghĩ đầu tiên.',
  'feed.likeFailed':
    'Thích hoặc bỏ thích không thành công, xin vui lòng thử lại sau.',
  'post.authorAnonymous':
    'Ẩn danh',
  'post.report':
    'Báo cáo bài viết',
  'post.imageAlt':
    'Ảnh bài đăng',
  'post.unlike':
    'Bỏ thích',
  'post.like':
    'Thích',
  'post.reply':
    'Trả lời',
  'comment.loading':
    'Đang tải bình luận...',
  'comment.none':
    'Chưa có bình luận',
  'comment.loadFailed':
    'Tải bình luận không thành công, xin vui lòng thử lại',
  'comment.placeholder':
    'Viết bình luận...',
  'comment.max':
    'Đa 2000 ký tự',
  'comment.submit':
    'Bình luận',
  'comment.failed':
    'Gửi bình luận không thành công, xin vui lòng thử lại.',
  'comment.report':
    'Báo cáo',
  'comment.more':
    'Tải thêm bình luận...',
  'report.reasonPlaceholder':
    'Nhập lý do báo cáo (đa 500 ký tự)',
  'report.note':
    'Báo cáo của bạn sẽ được gửi đến quản trị viên trang web',
  'report.formLabel':
    'Ô nhập báo cáo',
  'report.submit':
    'Gửi báo cáo',
  'report.failed':
    'Báo cáo không thành công, xin vui lòng thử lại.',
  'report.sent':
    'Báo cáo đã được gửi, cảm ơn bạn đã phản hồi.',
  'newPost.avatarYou':
    'Bạn',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'Thêm bài đăng',
  'newPost.loginFirst':
    'Hãy đăng nhập bằng Google trước rồi đăng bài',
  'newPost.contentPlaceholder':
    'Chia sẻ suy nghĩ của bạn...',
  'newPost.addImage':
    'Thêm ảnh',
  'newPost.emailPrivate':
    'Email của bạn sẽ không được công khai',
  'newPost.submit':
    'Đăng bài',
  'newPost.publishing':
    'Đang đăng bài...',
  'newPost.uploading':
    'Đang tải ảnh lên...',
  'newPost.failed':
    'Đăng bài không thành công, vui lòng thử lại sau.',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'Hồ sơ cá nhân',
  'profile.edit':
    'Sửa',
  'profile.loginPrompt':
    'Đăng nhập để thiết lập biệt danh và tiểu sử diễn đàn của bạn.',
  'profile.nicknameLabel':
    'Biệt danh diễn đàn',
  'profile.notSet':
    'Chưa thiết lập',
  'profile.notSetBio':
    'Chưa thiết lập tiểu sử.',
  'profile.nicknameInput':
    'Biệt danh',
  'profile.nicknamePlaceholder':
    'Nhập biệt danh',
  'profile.nicknameHint':
    'Biệt danh sẽ hiển thị trên bài viết bạn đã đăng, tối đa 30 ký tự.',
  'profile.bioLabel':
    'Tiểu sử',
  'profile.bioPlaceholder':
    'Giới thiệu bản thân (không bắt buộc)',
  'profile.bioHint':
    'Tối đa 500 ký tự.',
  'profile.updated':
    'Hồ sơ cá nhân đã được cập nhật.',
  'profile.saving':
    'Đang lưu...',
  'profile.saveFailed':
    'Lưu không thành công, vui lòng thử lại sau.',
  'profile.loadFailed':
    'Không thể tải dữ liệu, vui lòng thử lại sau.',
  'profile.followingEntry':
    'Đang theo dõi',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'Hồ sơ cá nhân công khai',
  'publicProfile.avatar':
    'Ảnh đại diện',
  'publicProfile.loading':
    'Đang tải...',
  'publicProfile.invalidLinkName':
    'Liên kết hồ sơ công khai không hợp lệ',
  'publicProfile.invalidLinkBio':
    'Vui lòng truy cập hồ sơ cá nhân từ tên tác giả trong bài viết diễn đàn.',
  'publicProfile.notFound':
    'Không tìm thấy người dùng này',
  'publicProfile.anonymous':
    'Người dùng ẩn danh',
  'publicProfile.noBio':
    'Người dùng này chưa thiết lập hồ sơ công khai.',
  'publicProfile.loadFailed':
    'Không thể tải hồ sơ công khai.',
  'publicProfile.postsLabel':
    'Bài viết',
  'publicProfile.emptyPosts':
    'Người này chưa đăng bài nào.',

  'follow.label':
    'Theo dõi người dùng này',
  'follow.action':
    'Theo dõi',
  'follow.actionDone':
    'Đang theo dõi',
  'follow.unfollow':
    'Bỏ theo dõi',
  'follow.done':
    'Bạn đã theo dõi người dùng này.',
  'follow.failed':
    'Không thể theo dõi, vui lòng thử lại sau.',

  'following.peopleLabel':
    'Người bạn theo dõi',
  'following.postsLabel':
    'Bài viết của người bạn theo dõi',
  'following.peopleLoading':
    'Đang tải danh sách theo dõi...',
  'following.emptyPeople':
    'Bạn chưa theo dõi ai. Nhấn «Theo dõi» trên một bài viết, hoặc theo dõi ai đó từ hồ sơ công khai của họ.',
  'following.emptyPosts':
    'Những người bạn theo dõi chưa đăng bài nào.',
  'following.peopleFailed':
    'Không thể tải danh sách theo dõi.',
  'following.postsFailed':
    'Không thể tải bài viết của người bạn theo dõi.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'Chào mừng trở lại',
  'login.body':
    '{site} là không gian dành cho tất cả những người muốn viết ra suy nghĩ. Không cần biểu mẫu đăng ký — chỉ cần một tài khoản Google là có thể bắt đầu đăng bài.',
  'login.browseFirst':
    'Xem trang chủ trước',
  'admin.skipToMain':
    'Nhảy đến nội dung chính',
  'admin.railLabel':
    'Menü quản trị',
  'admin.railBrandAria':
    'Trang chủ {site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'Tính năng chính',
  'admin.railGovernance':
    'Quản trị',
  'admin.railMode':
    'Chế độ quản trị viên',
  'admin.railExit':
    'Về diễn đàn',
  'admin.topbarMenu':
    'Chuyển menu quản trị',
  'admin.statusOnline':
    'Kết nối bình thường',
  'admin.topbarForum':
    'Diễn đàn',
  'admin.logoutFailed':
    'Đăng xuất không thành công, vui lòng thử lại sau.',
  'admin.navUsers':
    'Quản lý người dùng',
  'admin.navPosts':
    'Bài viết diễn đàn',
  'admin.navReports':
    'Quản lý báo cáo',
  'admin.navMonitor': 'Giám sát hệ thống',
  'admin.navLog': 'Nhật ký thao tác',
  'admin.navStats': 'Xu hướng nội dung',
  'admin.navExport': 'Xuất và thao tác hàng loạt',
  'admin.navSessions': 'Đăng nhập & phiên làm việc',
  'admin.navBlocks': 'Danh sách chặn IP',
  'admin.navAnnouncements': 'Thông báo',
  'admin.listLoadFailed':
    'Tải không thành công',
  'admin.dlgClose':
    'Đóng cửa sổ',
  'admin.dlgConfirm':
    'Xác nhận',
  'admin.dlgSave':
    'Lưu',
  'admin.dlgApplyTags':
    'Áp dụng nhãn',
  'admin.dlgNoTags':
    'Hiện tại không có nhãn nào có thể áp dụng; vui lòng thêm nhãn trong “Quản lý nhãn” bên dưới.',

  'monitor.title': 'Giám sát hệ thống',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'Xem trực tiếp tình trạng dịch vụ, lưu lượng yêu cầu, phân bố độ trễ và bộ đếm của bộ giới hạn tốc độ.',
  'monitor.refresh': 'Làm mới',
  'monitor.refreshing': 'Đang tải…',
  'monitor.autoRefresh': 'Tự động làm mới',
  'monitor.autoRefreshOn': 'Tự động làm mới mỗi {seconds} giây',
  'monitor.autoRefreshOff': 'Đã tạm dừng tự động làm mới',
  'monitor.nextUpdate': 'Cập nhật sau {seconds} giây',
  'monitor.loadFailed': 'Không tải được dữ liệu giám sát.',
  'monitor.loadFailedHint': 'Hãy kiểm tra rằng bạn đã đăng nhập với tư cách quản trị viên và phía máy chủ vẫn đang chạy.',
  'monitor.pausedHint': 'Tự động làm mới đang tạm dừng; màn hình hiển thị kết quả đọc thành công gần nhất.',
  'monitor.visibilityPaused': 'Trang đang ở chế độ nền nên tự động làm mới tạm dừng.',
  'monitor.lastUpdated': 'Cập nhật lúc {time}',
  'monitor.probeTook': 'Thăm dò phụ thuộc: {ms} ms',
  'monitor.unreachable': 'Phía máy chủ không phản hồi. Màn hình đứng ở lần đọc thành công gần nhất.',
  'monitor.depsTitle': 'Tình trạng dịch vụ',
  'monitor.depsNote': 'Mỗi lần đọc sẽ thăm dò thực sự từng phụ thuộc một lần; thời gian chờ mỗi phụ thuộc là 2 giây và cả ba chạy song song.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'Công cụ tìm kiếm',
  'monitor.stateOk': 'Bình thường',
  'monitor.stateDown': 'Không kết nối được',
  'monitor.stateDisabled': 'Chưa bật',
  'monitor.depSearchFallback': 'Chưa đặt ES_URL nên tìm kiếm dùng so khớp từ khóa của MySQL.',
  'monitor.depDisabled': 'Không có client Redis nào được tiêm vào, nên tính năng media đang tắt.',
  'monitor.depLatency': 'Phản hồi trong {ms} ms',
  'monitor.depKeys': '{count} khóa',
  'monitor.depMemory': 'Bộ nhớ {size}',
  'monitor.depPoolUsage': 'Kết nối {inUse}/{open} (tối đa {max})',
  'monitor.depPoolWait': 'Chờ {count} lần, tổng {ms} ms',
  'monitor.depRedisPool': 'Trúng {hits} / trượt {misses}',
  'monitor.depEngineMysql': 'So khớp từ khóa MySQL',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'Tổng quan yêu cầu',
  'monitor.statUptime': 'Thời gian chạy',
  'monitor.statRequests': 'Tổng số yêu cầu',
  'monitor.statErrorRate': 'Tỉ lệ lỗi',
  'monitor.statP95': 'Độ trễ P95',
  'monitor.statInFlight': 'Đang xử lý',
  'monitor.statGoroutines': 'Goroutine',
  'monitor.statHeap': 'Bộ nhớ heap',
  'monitor.statDbPool': 'Kết nối cơ sở dữ liệu',
  'monitor.statRateLimited': 'Bị giới hạn',
  'monitor.statCountWithPeak': 'đỉnh {peak}',
  'monitor.statCountWithInUse': '{inUse} đang dùng, {idle} rảnh',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} lõi logic · {gc} chu kỳ GC',
  'monitor.noData': 'Chưa có yêu cầu nào.',
  'monitor.noDataBody': 'Các yêu cầu tính từ lúc dịch vụ khởi động sẽ xuất hiện ở đây; hiện khung này còn trống.',
  'monitor.timelineTitle': 'Lưu lượng {minutes} phút gần nhất',
  'monitor.timelineNote': 'Tổng số được cộng dồn từ khi tiến trình này bắt đầu và đặt lại khi khởi động lại; mỗi phân vị độ trễ là cận trên của một nhóm trong biểu đồ, nên chỉ nhận các giá trị rời rạc. Một phút không có cột nghĩa là thời điểm đó không có lưu lượng.',
  'monitor.timelineLive': 'Tiến trình này',
  'monitor.timelineHistory': 'Trước khi khởi động lại',
  'monitor.timelineLegendVolume': 'Yêu cầu',
  'monitor.timelineLegendError': 'Lỗi 5xx',
  'monitor.timelinePeak': 'Đỉnh {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'Cơ sở dữ liệu không có số liệu tổng hợp lịch sử; nếu lúc khởi động không đọc được (thiếu bảng hoặc thiếu quyền) thì chỉ hiển thị tiến trình hiện tại.',
  'monitor.routesTitle': 'Theo tuyến',
  'monitor.routesNote': 'Đường dẫn được chuẩn hóa (số và địa chỉ email thay bằng :id), nên các id khác nhau của cùng một tuyến được tính chung.',
  'monitor.colRoute': 'Tuyến',
  'monitor.colCount': 'Yêu cầu',
  'monitor.colAvg': 'Trung bình',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'Chậm nhất',
  'monitor.colErrors': 'Lỗi',
  'monitor.routeOther': 'Khác (đã chạm giới hạn số tuyến)',
  'monitor.limitsTitle': 'Bộ giới hạn tốc độ',
  'monitor.limitsNote': 'Mỗi nhóm có hạn mức riêng, tương ứng với chi phí của các endpoint; số lần chặn là tổng số phản hồi 429 kể từ khi tiến trình này bắt đầu.',
  'monitor.colLimiter': 'Bộ giới hạn',
  'monitor.colBudget': 'Hạn mức',
  'monitor.colAllowed': 'Cho qua',
  'monitor.colBlocked': 'Chặn',
  'monitor.colTracked': 'Nguồn đang theo dõi',
  'monitor.colBlockedRate': 'Tỉ lệ chặn',
  'monitor.limitContent': 'Ghi nội dung',
  'monitor.limitUpload': 'Tải ảnh lên',
  'monitor.limitAuth': 'Đăng nhập OAuth',
  'monitor.limitBudget': '{limit} lần mỗi {window} giây',
  'monitor.limitUnknown': '(không rõ)',
  'monitor.noLimits': 'Không có bộ giới hạn nào.',
  'monitor.noLimitsBody': 'Các bộ giới hạn tốc độ chưa được tạo nên không có bộ đếm.',

  'log.title': 'Nhật ký thao tác',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'Tra cứu mọi thay đổi trong trang quản trị: ai, lúc nào, với đối tượng nào, và trường nào đổi từ gì sang gì.',
  'log.refresh': 'Làm mới',
  'log.loadFailed': 'Không tải được nhật ký thao tác.',
  'log.empty': 'Không có thao tác nào khớp bộ lọc.',
  'log.emptyBody': 'Hãy nới lỏng bộ lọc, hoặc xác nhận rằng quả thực không có thao tác nào trong khoảng thời gian này.',
  'log.count': 'Bản ghi {from}–{to} trên {total}',
  'log.retention': 'Bản ghi được giữ {days} ngày rồi xóa bởi tác vụ nền. Trang này không có nút xóa bản ghi — nhật ký kiểm toán có thể tự xóa dấu vết của chính nó thì không phải nhật ký kiểm toán.',
  'log.filterActor': 'Người thực hiện',
  'log.filterAction': 'Thao tác',
  'log.filterTargetType': 'Loại tài nguyên',
  'log.filterFrom': 'Từ',
  'log.filterTo': 'Đến',
  'log.filterAll': 'Tất cả',
  'log.filterApply': 'Áp dụng bộ lọc',
  'log.filterReset': 'Xóa bộ lọc',
  'log.filterTargetHint': 'Nhấn vào đối tượng ở bất kỳ dòng nào để chỉ xem các thao tác liên quan.',
  'log.colTime': 'Thời điểm',
  'log.colActor': 'Người thực hiện',
  'log.colAction': 'Thao tác',
  'log.colTarget': 'Đối tượng',
  'log.colChanges': 'Thay đổi',
  'log.colOrigin': 'Nguồn',
  'log.noChanges': '(không có thay đổi trường)',
  'log.changedTo': 'đổi thành',
  'log.removed': '(đã xóa)',
  'log.created': '(đã tạo)',
  'log.requestId': 'request {id}',
  'log.page': 'Trang {page}',
  'log.targetUser': 'Người dùng',
  'log.targetPost': 'Bài viết',
  'log.targetComment': 'Bình luận',
  'log.targetReport': 'Báo cáo',
  'log.targetTag': 'Thẻ',
  'log.targetSystem': 'Hệ thống',
  'log.actionUserSuspend': 'Đã khoá',
  'log.actionUserReinstate': 'Đã khoá lại',
  'log.actionUserTags': 'Đã đổi thẻ',
  'log.actionUserPost': 'Đăng bài thay người dùng',
  'log.actionUserComment': 'Bình luận thay người dùng',
  'log.actionUserContent': 'Đã xóa nội dung của họ',
  'log.actionPostCreate': 'Tạo bài viết',
  'log.actionPostUpdate': 'Sửa bài viết',
  'log.actionPostDelete': 'Xóa bài viết',
  'log.actionCommentCreate': 'Tạo bình luận',
  'log.actionCommentUpdate': 'Sửa bình luận',
  'log.actionCommentDelete': 'Xóa bình luận',
  'log.actionReportCreate': 'Tạo báo cáo',
  'log.actionReportResolve': 'Báo cáo được chấp nhận',
  'log.actionReportReject': 'Báo cáo bị bác',
  'log.actionReportUpdate': 'Sửa báo cáo',
  'log.actionReportDelete': 'Xóa báo cáo',
  'log.actionTagCreate': 'Tạo thẻ',
  'log.actionTagUpdate': 'Đổi tên thẻ',
  'log.actionTagDelete': 'Xóa thẻ',
  'log.fieldStatus': 'Trạng thái tài khoản',
  'log.fieldContent': 'Nội dung',
  'log.fieldName': 'Tên',
  'log.fieldTags': 'Thẻ',
  'log.fieldReason': 'Lý do',
  'log.fieldAuthorEmail': 'Tác giả',
  'log.fieldReporterEmail': 'Người báo cáo',
  'log.fieldTargetType': 'Loại tài nguyên',
  'log.fieldTargetId': 'Mã tài nguyên',
  'log.fieldPostId': 'Mã bài viết',
  'log.fieldCommentId': 'Mã bình luận',
  'log.fieldPostIdShort': 'Bài viết',
  'log.fieldCommentIdShort': 'Bình luận',
  'log.fieldTarget': 'Đối tượng',
  'log.fieldAssignmentsRemoved': 'Số liên kết đã gỡ',
  'log.truncated': 'đã cắt bớt',

  'stats.title': 'Xu hướng nội dung',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'Mỗi ngày có bao nhiêu người dùng, bài viết và bình luận mới, cùng những bài viết, thẻ và tác giả nào đang hoạt động nhất.',
  'stats.refresh': 'Làm mới',
  'stats.loadFailed': 'Không tải được thống kê nội dung.',
  'stats.window': 'Hiển thị',
  'stats.windowDays': '{days} ngày gần nhất',
  'stats.windowClamped': '(tối đa 90 ngày)',
  'stats.windowNote': 'Mỗi ngày được tính theo múi giờ địa phương của máy chủ. Nếu trang chạy ở UTC mà quản trị viên ở múi giờ khác, số liệu hôm nay sẽ thấp hơn — đó là do múi giờ, không phải lưu lượng giảm.',
  'stats.generatedAt': 'Thống kê tạo lúc {time}',
  'stats.seriesTitle': 'Mới mỗi ngày',
  'stats.seriesNote': 'Ba đường có thang đo khác nhau nên hiển thị riêng thay vì chồng lên nhau.',
  'stats.seriesUsers': 'Người dùng mới',
  'stats.seriesPosts': 'Bài viết mới',
  'stats.seriesComments': 'Bình luận mới',
  'stats.seriesEmpty': 'Không có dữ liệu trong khoảng này.',
  'stats.totalsTitle': 'Tổng trong khoảng',
  'stats.totalsNote': 'Đây là số lượng được thêm trong khoảng thời gian này, không phải tổng hiện tại của trang.',
  'stats.totalUsers': 'Người dùng mới',
  'stats.totalPosts': 'Bài viết mới',
  'stats.totalComments': 'Bình luận mới',
  'stats.totalLikes': 'Lượt thích mới',
  'stats.topPostsTitle': 'Bài viết được yêu thích',
  'stats.topPostsNote': 'Xếp theo bình luận và lượt thích, chỉ tính các bài viết đăng trong khoảng này.',
  'stats.topTagsTitle': 'Thẻ phổ biến',
  'stats.topTagsNote': 'Xếp theo số người được gắn thẻ, không giới hạn thời gian — thẻ là thuộc tính chứ không phải sự kiện.',
  'stats.topAuthorsTitle': 'Tác giả hoạt động',
  'stats.topAuthorsNote': 'Xếp theo số bài trong khoảng, số bình luận hiển thị riêng.',
  'stats.colExcerpt': 'Trích đoạn',
  'stats.colEngagement': 'Tương tác',
  'stats.colPosts': 'Bài viết',
  'stats.colComments': 'Bình luận',
  'stats.colUsers': 'Người dùng',
  'stats.colAuthor': 'Tác giả',
  'stats.empty': 'Không có dữ liệu trong khoảng này.',
  'stats.emptyBody': 'Hãy nới khoảng thời gian, hoặc xác nhận rằng quả thực không có nội dung mới.',
  'stats.engagement': '{comments} bình luận・{likes} lượt thích',
  'stats.rank': 'Hạng {rank}',

  'export.title': 'Xuất và thao tác hàng loạt',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'Xuất dữ liệu trang ra CSV để đối chiếu, hoặc xử lý nhiều tài khoản cùng lúc.',
  'export.download': 'Tải CSV',
  'export.downloading': 'Đang chuẩn bị…',
  'export.exportTitle': 'Xuất',
  'export.exportNote': 'Mỗi bản xuất tối đa 5 vạn dòng; vượt quá đó chỉ lấy các dòng mới nhất. Cột của cả ba bản xuất trùng với các cột ở trang quản trị nên có thể đối chiếu trực tiếp.',
  'export.exportUsers': 'Danh sách người dùng',
  'export.exportUsersNote': 'Email, trạng thái, thời điểm tạo và cập nhật, số bài viết và bình luận.',
  'export.exportPosts': 'Danh sách bài viết',
  'export.exportPostsNote': 'Mã, tác giả, 200 ký tự đầu nội dung, thời điểm tạo, số bình luận và lượt thích.',
  'export.exportReports': 'Danh sách báo cáo',
  'export.exportReportsNote': 'Mã, đối tượng bị báo, người báo, lý do, trạng thái và lịch sử xử lý.',
  'export.safety': 'Tệp bắt đầu bằng dấu thứ tự byte UTF-8 nên Excel mở không bị lỗi chữ.',
  'export.safetyPrefix': 'Giá trị bắt đầu bằng = + - @ hoặc khoảng trắng vô hình sẽ được thêm dấu nháy đơn ở đầu — nhờ đó bảng tính coi chúng là văn bản thay vì chạy như công thức. Tiền tố này là cố ý, đừng yêu cầu gỡ bỏ.',
  'export.batchTitle': 'Thao tác hàng loạt',
  'export.batchNote': 'Các nút bật lên sau khi bạn chọn tài khoản ở trang Người dùng. Một thao tác hàng loạt áp dụng toàn bộ hoặc không áp dụng gì cả; không có kết quả áp dụng một phần.',
  'export.batchSuspend': 'Khóa tài khoản đã chọn',
  'export.batchReinstate': 'Mở khóa tài khoản đã chọn',
  'export.batchTags': 'Gắn thẻ hàng loạt',
  'export.batchTagsNote': 'Ngữ nghĩa ghi đè: danh sách gửi đi chính là kết quả. Gửi danh sách r��ng nghĩa là xóa hết thẻ.',
  'export.batchConfirm': 'Áp dụng “{action}” cho {count} tài khoản?',
  'export.batchConfirmTags': 'Ghi đè thẻ của {count} tài khoản bằng {tags}?',
  'export.batchTagsPicker': 'Chọn thẻ',
  'export.batchTagsNone': 'Không chọn thẻ nào (xóa hết)',
  'export.batchRunning': 'Đang xử lý…',
  'export.batchDone': 'Đã cập nhật {updated} tài khoản',
  'export.batchDoneUnchanged': 'trong đó {unchanged} vốn đã ở trạng thái đích nên không đổi',
  'export.batchSkipped': 'Bỏ qua {count}',
  'export.batchMax': 'Tối đa 200 tài khoản mỗi lần',
  'export.gotoUsers': 'Đến trang Người dùng',
  'export.noSelection': 'Hãy chọn tài khoản ở trang Người dùng trước.',
  'export.selected': 'Đã chọn {count} tài khoản',
  'export.clearSelection': 'Bỏ chọn',
  'export.selectionHint': 'Lựa chọn chỉ giữ ở trang này và biến mất khi đóng trang.',

  'session.title': 'Đăng nhập & phiên làm việc',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'Xem những đăng nhập nào còn hiệu lực, và buộc đăng xuất một tài khoản khỏi mọi thiết bị.',
  'session.refresh': 'Làm mới',
  'session.loadFailed': 'Không tải được danh sách phiên làm việc.',
  'session.privacyTitle': 'Vì sao không hiện toàn bộ token',
  'session.privacyNote': 'Bản thân token chính là thông tin đăng nhập. Chỉ hiện tám ký tự đầu để quản trị viên phân biệt được hai dòng là cùng một phiên — và như vậy không đủ để bất kỳ ai đăng nhập thay người dùng, kể cả người nhận được ảnh chụp màn hình trang này. Đây là giới hạn có chủ đích, không phải tính năng dở dang.',
  'session.expireNote': 'Một phiên hết hạn sau {hours} giờ không hoạt động; mọi yêu cầu đều gia hạn lại.',
  'session.filterEmail': 'Lọc theo tài khoản',
  'session.filterPlaceholder': 'Địa chỉ email đầy đủ',
  'session.search': 'Tìm',
  'session.clearFilter': 'Xóa',
  'session.summary': '{total} phiên trên toàn trang, đã kiểm tra {scanned} khoá',
  'session.truncated': 'Lượt quét chạm giới hạn {scanned} khoá nên dừng sớm, do đó danh sách này không đầy đủ.',
  'session.empty': 'Hiện không có phiên nào.',
  'session.emptyBody': 'Không ai đang đăng nhập, hoặc mọi phiên đã hết hạn.',
  'session.colUser': 'Tài khoản',
  'session.colToken': 'Phiên',
  'session.colCreated': 'Thời điểm tạo',
  'session.colExpires': 'Hết hạn',
  'session.colRemaining': 'Còn',
  'session.colActions': 'Thao tác',
  'session.unknown': 'Không rõ',
  'session.adminBadge': 'Quản trị',
  'session.revoke': 'Buộc đăng xuất',
  'session.revokeTitle': 'Buộc đăng xuất {email}',
  'session.revokeMessage': '{count} phiên hiện tại của tài khoản sẽ mất hiệu lực ngay và trạng thái đăng nhập trên mọi thiết bị sẽ bị xóa. Người dùng phải đăng nhập lại. Tiếp tục?',
  'session.revokeRunning': 'Đang thu hồi…',
  'session.revokeDone': 'Đã thu hồi {count} phiên',
  'session.revokeNone': 'Tài khoản này không có phiên nào đang hoạt động',
  'session.revokeFailed': 'Không xác nhận được việc đăng xuất.',
  'session.revokeUnavailable': 'Điều đó không có nghĩa thu hồi thất bại: lượt quét chạm giới hạn khoá, nên có thể có phiên chưa được quét tới. Hãy thử lại sau giây lát.',
  'session.titleColumnNote': 'Chỉ là tiền tố để nhận diện, không dùng để đăng nhập',

  'block.title': 'Danh sách chặn IP',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'Đưa các địa chỉ nguồn đã xác nhận lạm dụng vào danh sách. Danh sách nằm trong Redis nên không bị xóa khi khởi động lại hay triển khai.',
  'block.refresh': 'Làm mới',
  'block.loadFailed': 'Không tải được danh sách chặn.',
  'block.unavailable': 'Trang này không có kết nối Redis nên tính năng chặn chưa bật.',
  'block.unavailableNote': 'Danh sách cần đúng Redis dùng cho phiên đăng nhập và token media. Khi kết nối xong, trang này sẽ hoạt động; cho tới đó biện pháp duy nhất là giới hạn tần suất (trong tiến trình, mất khi khởi động lại).',
  'block.add': 'Chặn',
  'block.addTitle': 'Chặn một địa chỉ IP',
  'block.addMessage': 'Địa chỉ bị chặn sẽ bị từ chối mọi yêu cầu ghi (bài viết, bình luận, thích, báo cáo, tải ảnh lên, chuyển hướng đăng nhập) cho tới khi hết hạn. Đọc không bị ảnh hưởng.',
  'block.ipLabel': 'Địa chỉ IP',
  'block.ipPlaceholder': '203.0.113.9 hoặc 2001:db8::1',
  'block.durationLabel': 'Thời hạn',
  'block.reasonLabel': 'Lý do',
  'block.reasonPlaceholder': 'Vì sao chặn địa chỉ này (được ghi vào nhật ký kiểm toán)',
  'block.reasonHint': 'Lý do chỉ vào nhật ký kiểm toán. Người bị chặn không nhìn thấy và nó cũng không xuất hiện trong thông báo lỗi công khai.',
  'block.blocking': 'Đang chặn…',
  'block.done': 'Đã chặn {ip}',
  'block.removed': 'Đã bỏ chặn {ip}',
  'block.removedNone': '{ip} vốn không bị chặn',
  'block.failed': 'Thao tác chặn thất bại.',
  'block.unavailableService': 'Danh sách chặn không dùng được (thiếu Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'Hết hạn',
  'block.colRemaining': 'Còn lại',
  'block.colActions': 'Thao tác',
  'block.unblock': 'Bỏ chặn',
  'block.unblockTitle': 'Bỏ chặn {ip}',
  'block.unblockMessage': 'Địa chỉ này sẽ lấy lại quyền ghi bình thường ngay. Tiếp tục?',
  'block.empty': 'Danh sách chặn trống.',
  'block.emptyBody': 'Không có địa chỉ nào bị chặn. Đây là trạng thái bình thường: chặn luôn là quyết định của quản trị, và hệ thống không bao giờ tự chặn ai.',
  'block.notAutoNote': 'Danh sách này không bao giờ tự đầy. Một địa chỉ vượt hạn mức chỉ nhận 429; nó không được tự thêm vào đây, vì cùng một cổng ra có thể thuộc cả một văn phòng hay cả một NAT, và chặn tự động sẽ trúng cả họ.',
  'block.scopeNoteLabel': 'Phạm vi',
  'block.notAutoNoteLabel': 'Không tự động',
  'block.maxNoteLabel': 'Thời hạn tối đa',
  'block.scopeNote': 'Chặn chỉ chặn yêu cầu ghi. Đọc bài viết, bình luận và tài nguyên tĩnh vẫn được, và người bị chặn vẫn đăng nhập và xem nội dung: điều đó là cố ý, vì các endpoint đọc cố tình không giới hạn (nếu không thì khách không đăng nhập sẽ không dùng được), và chặn bao phủ đúng tập đó.',
  'block.maxNote': 'Một lần chặn tối đa 365 ngày. Thời gian dài hơn bị rút về một năm: thời điểm hết hạn được lưu dạng số, nên “vĩnh viễn” sẽ thành một lệnh chặn không ai nhớ và không tự được gỡ.',
  'block.count': '{count} địa chỉ bị chặn',
  'block.ipInvalid': 'Địa chỉ IP không hợp lệ. Hãy nhập địa chỉ IPv4 hoặc IPv6; không hỗ trợ dải CIDR.',
  'announce.label': 'Thông báo trên trang',
  'announce.publicNote': 'Thông báo',
  'announce.closeAria': 'Đóng thông báo này',
  'announce.publishedOn': 'Đăng lúc {date}',
  'announce.expiresOn': 'Hết hạn lúc {date}',
  'announce.neverExpires': 'Không hết hạn',
  'announce.pinnedBadge': 'Ghim',
  'announce.title': 'Thông báo',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'Hiển thị một thông báo ở trên cùng mọi trang của trang web. Chỉ một thông báo có hiệu lực tại một thời điểm — đăng thông báo mới sẽ vô hiệu hóa thông báo cũ.',
  'announce.refresh': 'Làm mới',
  'announce.loadFailed': 'Không tải được danh sách thông báo.',
  'announce.new': 'Đăng thông báo mới',
  'announce.edit': 'Sửa',
  'announce.deactivate': 'Vô hiệu hóa',
  'announce.reactivate': 'Kích hoạt lại',
  'announce.deleteNote': 'Thông báo không bao giờ bị xóa, chỉ bị vô hiệu hóa — giữ lại lịch sử là để trả lời được nó được đăng khi nào và bởi ai.',
  'announce.bodyLabel': 'Nội dung thông báo',
  'announce.bodyPlaceholder': 'Ví dụ: hệ thống sẽ bảo trì vào thứ Năm từ 02:00 đến 04:00.',
  'announce.bodyHint': 'Tối đa 300 ký tự. Văn bản thuần, giữ ngắt dòng.',
  'announce.activeLabel': 'Hiển thị ngay',
  'announce.expiryLabel': 'Thời hạn',
  'announce.expiryNever': 'Không tự hết hạn',
  'announce.expiryHours': 'sau {hours} giờ',
  'announce.expiryDays': 'sau {days} ngày',
  'announce.saving': 'Đang lưu…',
  'announce.published': 'Đã đăng thông báo',
  'announce.updated': 'Đã cập nhật thông báo',
  'announce.deactivated': 'Đã vô hiệu hóa thông báo',
  'announce.reactivated': 'Đã kích hoạt lại thông báo',
  'announce.saveFailed': 'Thao tác thông báo thất bại.',
  'announce.empty': 'Chưa có thông báo nào.',
  'announce.emptyBody': 'Ngay khi bạn đăng một thông báo, nó sẽ xuất hiện ở đầu trang của mọi khách truy cập.',
  'announce.colBody': 'Nội dung',
  'announce.colState': 'Trạng thái',
  'announce.colAuthor': 'Người đăng',
  'announce.colCreated': 'Thời điểm đăng',
  'announce.colActions': 'Thao tác',
  'announce.stateActive': 'Đang hiển thị',
  'announce.stateInactive': 'Đã vô hiệu hóa',
  'announce.stateExpired': 'Đã hết hạn',
  'announce.confirmDeactivate': 'Đăng thông báo này sẽ vô hiệu hóa thông báo hiện tại, và mọi người sẽ thấy nội dung mới ngay lập tức. Tiếp tục?',
  'announce.confirmEdit': 'Sửa nội dung hoặc thời hạn của thông báo này?',
  'announce.count': 'Tổng {count}',
  'users.title':
    'Quản lý người dùng',
  'users.contentAction':
    'Nội dung',
  'users.updateContentFailed':
    'Không thể thao tác nội dung.',
  'users.updated':
    'Nội dung đã được cập nhật.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'Xem thống kê hoạt động của người dùng diễn đàn, nhãn và trạng thái tài khoản, đồng thời xử lý việc khóa người dùng và quản lý từng nội dung.',
  'users.refresh':
    'Cập nhật dữ liệu',
  'users.statTotal':
    'Tổng số người dùng',
  'users.statActive':
    'Đang hoạt động',
  'users.statSuspended':
    'Đã khóa',
  'users.statContent':
    'Bài viết／bình luận',
  'users.count':
    '{count} người dùng',
  'users.tagsCount':
    '{count} nhãn',
  'users.loadFailed':
    'Không thể tải dữ liệu người dùng.',
  'users.tagsLoadFailed':
    'Không thể tải dữ liệu nhãn.',
  'users.panelTitle':
    'Người dùng diễn đàn',
  'users.tagsPanelTitle':
    'Nhãn của người dùng',
  'users.colUser':
    'Người dùng',
  'users.colTags':
    'Nhãn',
  'users.colStatus':
    'Trạng thái',
  'users.colPosts':
    'Bài viết',
  'users.colComments':
    'Bình luận',
  'users.colLikes':
    'Thích',
  'users.colLastActivity':
    'Hoạt động gần nhất',
  'users.colActions':
    'Thao tác',
  'users.nicknameUnset':
    'Chưa thiết lập biệt danh',
  'users.notSet':
    'Chưa thiết lập',
  'users.statusActive':
    'Đang hoạt động',
  'users.statusSuspended':
    'Đã khóa',
  'users.emptyTitle':
    'Hiện tại không có dữ liệu người dùng',
  'users.emptyBody':
    'Nơi này sẽ trống chỉ khi không có người dùng nào đăng nhập diễn đàn bằng Google.',
  'users.tagsEmptyTitle':
    'Hiện tại không có nhãn',
  'users.tagsEmptyBody':
    'Trước tiên hãy tạo nhãn để có thể áp dụng cho đối tượng trong danh sách người dùng.',
  'users.addTag':
    'Thêm nhãn',
  'users.colName':
    'Tên',
  'users.colCreated':
    'Thời gian tạo',
  'users.colUpdated':
    'Thời gian cập nhật',
  'users.renameTag':
    'Đổi tên',
  'users.suspend':
    'Khóa',
  'users.restore':
    'Khôi phục',
  'users.statusDialogTitle':
    '{action} người dùng này',
  'users.statusSuspendMessage':
    '{email} sẽ không thể đăng nhập diễn đàn nữa, các bài viết và bình luận hiện có sẽ được giữ lại. Có tiếp tục không?',
  'users.statusRestoreMessage':
    '{email} sẽ khôi phục quyền đăng nhập và đăng bài. Có tiếp tục không?',
  'users.userSuspended':
    'Người dùng đã được khóa.',
  'users.userRestored':
    'Người dùng đã được khôi phục.',
  'users.updateStatusFailed':
    'Cập nhật trạng thái người dùng thất bại.',
  'users.editTagsTitle':
    'Sửa nhãn · {user}',
  'users.editTagsMessage':
    'Chọn nhãn cần áp dụng; bỏ chọn tất cả đồng nghĩa với việc loại bỏ tất cả nhãn của người dùng này.',
  'users.tagsUpdated':
    'Nhãn của người dùng đã được cập nhật.',
  'users.updateTagsFailed':
    'Cập nhật nhãn người dùng thất bại.',
  'users.contentLoadFailed':
    'Nội dung tải không thành công.',
  'users.contentLoadFailedShort':
    'Nội dung tải không thành công.',
  'users.contentPanelTitle':
    'Nội dung người dùng',
  'users.contentCount':
    '{posts} bài viết · {comments} bình luận',
  'users.contentLoading':
    'Đang tải bài viết và bình luận…',
  'users.addPost':
    'Thêm bài viết',
  'users.addComment':
    'Thêm bình luận',
  'users.postsColumn':
    'bài viết',
  'users.commentsColumn':
    'bình luận',
  'users.noPosts':
    'Chưa có bài viết',
  'users.noComments':
    'Chưa có bình luận',
  'users.postRef':
    'bài viết #{id}',
  'users.editRecordTitle':
    'Sửa {kind} #{id}',
  'users.deleteRecordTitle':
    'Xóa {kind} #{id}',
  'users.deleteRecordMessage':
    'Sau khi xóa, không thể khôi phục; các lượt thích và dữ liệu liên quan cũng sẽ bị loại bỏ. Bạn có chắc muốn tiếp tục không?',
  'users.contentLabel':
    'Nội dung',
  'users.addPostTitle':
    'Thêm bài viết',
  'users.addPostMessage':
    'Nội dung này sẽ được đăng với tư cách người dùng đó; trường tác giả không thể bị giả mạo.',
  'users.postContentLabel':
    'Nội dung bài viết',
  'users.postContentPlaceholder':
    'Nhập nội dung bài viết',
  'users.pickPostTitle':
    'Chọn bài viết',
  'users.postIdLabel':
    'id bài viết',
  'users.postIdPlaceholder':
    'ID của bài viết cần bình luận',
  'users.addCommentTitle':
    'Thêm bình luận',
  'users.commentContentLabel':
    'Nội dung bình luận',
  'users.commentContentPlaceholder':
    'Nhập nội dung bình luận',
  'users.createTagTitle':
    'Thêm nhãn',
  'users.createTagMessage':
    'Nhãn có thể dùng để phân loại người dùng, chẳng hạn «quản trị viên», «hoạt động» hoặc «đã bị khóa».',
  'users.tagNameLabel':
    'Tên nhãn',
  'users.tagNamePlaceholder':
    'Tối đa 50 ký tự',
  'users.renameTagTitle':
    'Đổi tên nhãn',
  'users.renameTagMessage':
    'Tất cả người dùng sử dụng nhãn này sẽ thấy tên mới.',
  'users.deleteTagTitle':
    'Xóa nhãn «{name}»',
  'users.deleteTagMessage':
    'Sau khi xóa, nhãn này trên tất cả người dùng sẽ bị loại bỏ và không thể khôi phục. Bạn có chắc muốn tiếp tục không?',
  'users.deleteTagConfirm':
    'Xóa nhãn',
  'users.tagCreated':
    'Nhãn đã được tạo.',
  'users.createTagFailed':
    'Tạo nhãn thất bại.',
  'users.tagUpdated':
    'Nhãn đã được cập nhật.',
  'users.updateTagFailed':
    'Cập nhật nhãn thất bại.',
  'users.tagDeleted':
    'Nhãn đã bị xóa.',
  'users.deleteTagFailed':
    'Xóa nhãn thất bại.',
  'users.refreshDone':
    'Đã cập nhật đến dữ liệu mới nhất.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'Trang quản trị',
  'users.signinBody':
    'Quản trị nội dung tập trung, giúp mọi lượt kiểm duyệt đều rõ ràng, nhanh chóng và có thể theo dõi được.',
  'users.signinStep1':
    'Xác minh bảo mật',
  'users.signinStep2':
    'Quản trị người dùng',
  'users.signinStep3':
    'Kiểm duyệt nội dung',
  'users.signinPanelTitle':
    'Đăng nhập Admin Console',
  'users.signinPanelBody':
    'Admin Console chỉ dành cho tài khoản quản trị viên Google đã được ủy quyền; vui lòng đăng nhập với tư cách quản trị viên.',
  'posts.title':
    'bài viết diễn đàn',
  'posts.searching':
    'Đang tìm kiếm…',
  'posts.searchDegraded':
    '(Dịch vụ tìm kiếm chưa được bật, so sánh từ khóa trong cơ sở dữ liệu)',
  'posts.searchSummary':
    '«{query}» tìm thấy {total} mục, trang này hiển thị {shown} mục',
  'posts.pendingCount':
    '{count} mục đang chờ xử lý',
  'posts.pageSummary':
    'Trang {page} / {pages}, trang này có {count} bài',
  'posts.listLoadFailed':
    'Danh sách bài viết tải không thành công.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'Tạo, sửa và xóa bài viết diễn đàn, đồng thời quản trị nội dung ở cấp bình luận.',
  'posts.toReports':
    'Quản trị báo cáo',
  'posts.editorTitleNew':
    'Thêm bài viết',
  'posts.editorTitleEdit':
    'Sửa bài viết #{id}',
  'posts.editorNote':
    'Đăng với tư cách quản trị viên; tác giả được lấy từ định danh đã đăng nhập, nội dung yêu cầu không thể giả mạo tác giả.',
  'posts.cancelEdit':
    'Hủy sửa',
  'posts.contentLabel':
    'Nội dung bài viết',
  'posts.contentPlaceholder':
    'Nhập nội dung bài viết',
  'posts.saveChanges':
    'Lưu thay đổi',
  'posts.emptyContent':
    'Nội dung bài viết không được để trống.',
  'posts.saving':
    'Đang lưu…',
  'posts.saved':
    'Bài viết đã được cập nhật.',
  'posts.published':
    'Bài viết đã được đăng.',
  'posts.saveFailed':
    'Lưu thất bại.',
  'posts.deleteTitle':
    'Xóa bài viết #{id}',
  'posts.deleteMessage':
    'Sau khi xóa, không thể khôi phục; tất cả bình luận dưới bài viết này cũng sẽ bị loại bỏ. Bạn có chắc muốn tiếp tục không?',
  'posts.deleteConfirm':
    'Xóa bài viết',
  'posts.deleted':
    'Bài viết đã bị xóa.',
  'posts.deleteFailed':
    'Xóa bài viết thất bại.',
  'posts.commentUpdated':
    'Bình luận đã được cập nhật.',
  'posts.commentActionFailed':
    'Thao tác bình luận thất bại.',
  'posts.pickPostTitle':
    'Chọn bài viết mà bình luận thuộc về',
  'posts.pickPostMessage':
    'Mặc định là bài viết trong hàng này; nếu muốn gắn vào bài khác, hãy thay bằng đúng số bài viết.',
  'posts.postIdLabel':
    'id bài viết',
  'posts.addCommentAtTitle':
    'Thêm bình luận vào bài viết #{id}',
  'posts.addCommentMessage':
    'Bình luận này sẽ được đăng với tư cách quản trị viên.',
  'posts.commentContentLabel':
    'Nội dung bình luận',
  'posts.commentContentPlaceholder':
    'Nhập nội dung bình luận',
  'posts.add':
    'Thêm',
  'posts.editCommentTitle':
    'Sửa bình luận #{id}',
  'posts.deleteCommentTitle':
    'Xóa bình luận #{id}',
  'posts.deleteCommentMessage':
    'Sau khi xóa, không thể khôi phục. Bạn có chắc muốn tiếp tục không?',
  'posts.listTitle':
    'Danh sách bài viết',
  'posts.clearSearch':
    'Xóa bộ lọc tìm kiếm',
  'posts.searchLabel':
    'Tìm kiếm từ khóa',
  'posts.searchPlaceholder':
    'Nội dung bài viết, hoặc Email đầy đủ của tác giả bài viết',
  'posts.searchHint':
    'Được sắp xếp theo mức độ liên quan; nhập Email đầy đủ để tìm tất cả bài viết của người dùng này. Tìm kiếm thay thế phân trang, tối đa hiển thị 25 kết quả.',
  'posts.searchTotal':
    'Tổng cộng {total} kết quả tìm kiếm',
  'posts.searchFailed':
    'Tìm kiếm thất bại.',
  'posts.searchStatusFailed':
    'Tìm kiếm thất bại',
  'posts.colContentImage':
    'Nội dung và hình ảnh',
  'posts.colEngagement':
    'Tương tác',
  'posts.colComments':
    'Bình luận',
  'posts.colAuthor':
    'Tác giả',
  'posts.imageAlt':
    'Hình ảnh đính kèm bài viết',
  'posts.likes':
    '{count} lượt thích',
  'posts.author':
    'Tác giả: {name}',
  'posts.emptyTitle':
    'Hiện không có bài viết diễn đàn',
  'posts.emptyBody':
    'Bạn có thể dùng trình soạn thảo ở phía trên để tạo bài viết đầu tiên.',
  'posts.emptySearchTitle':
    'Không có bài viết phù hợp',
  'posts.emptySearchBody':
    'Không có bài viết nào khớp với \\"{query}\\". Thử dùng từ khóa khác.',
  'posts.commentCount':
    '{count} bình luận',
  'posts.noComments':
    'Chưa có bình luận',
  'posts.reportsTitle':
    'Báo cáo cần xử lý',
  'posts.allReports':
    'Tất cả báo cáo',
  'posts.colReportedContent':
    'Nội dung bị báo cáo',
  'posts.colReason':
    'Lý do báo cáo',
  'posts.colReporter':
    'Người báo cáo',
  'posts.colTime':
    'Thời gian',
  'posts.colVerdict':
    'Kết luận',
  'posts.emptyReportsTitle':
    'Không có báo cáo cần xử lý',
  'posts.emptyReportsBody':
    'Tất cả báo cáo đã được kết luận.',
  'posts.verdictResolved':
    'Đã xử lý',
  'posts.verdictRejected':
    'Không hợp lệ',
  'posts.verdictDialogTitle':
    'Đánh dấu báo cáo #{id} là \\"{label}\\"',
  'posts.verdictDialogMessage':
    'Sau khi đánh dấu, báo cáo sẽ rời danh sách cần xử lý, nhưng dữ liệu vẫn được giữ trên trang quản lý báo cáo. Tiếp tục không?',
  'posts.verdictConfirm':
    'Đánh dấu {label}',
  'posts.verdictDone':
    'Báo cáo đã được đánh dấu {label}.',
  'posts.verdictFailed':
    'Không thể cập nhật trạng thái báo cáo.',
  'posts.pin': 'Ghim',
  'posts.unpin': 'Bỏ ghim',
  'posts.pinTitle': 'Ghim bài viết này',
  'posts.unpinTitle': 'Bỏ ghim bài viết này',
  'posts.pinMessage': 'Sau khi ghim, bài viết này luôn ở đầu bảng tin của mọi người và bài mới không thể đẩy nó xuống.',
  'posts.unpinMessage': 'Bỏ ghim sau bài viết sẽ trở về vị trí theo thời gian.',
  'posts.pinDone': 'Đã ghim',
  'posts.unpinDone': 'Đã bỏ ghim',
  'posts.pinFailed': 'Thao tác ghim thất bại.',
  'reports.title':
    'Quản lý báo cáo',
  'reports.listSummary':
    '{count} báo cáo · {filter}',
  'reports.listLoadFailed':
    'Không thể tải danh sách báo cáo.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'Quyết định từng báo cáo: «Thông qua» sẽ xóa nội dung bị báo cáo, «Không hợp lệ» sẽ giữ nguyên văn bản. Đổi lại «Chờ xử lý» sẽ xóa thời gian kết luận hiện có.',
  'reports.backToPosts':
    'Về bài viết',
  'reports.editorTitle':
    'Sửa báo cáo #{id}',
  'reports.editorNote':
    'Có thể bổ sung lý do và trạng thái báo cáo; mục tiêu và người báo cáo là bản ghi hiện có, không thay đổi tại đây.',
  'reports.targetTypeLabel':
    'Loại mục tiêu',
  'reports.targetIdLabel':
    'ID mục tiêu',
  'reports.reporterEmailLabel':
    'Email người báo cáo',
  'reports.statusLabel':
    'Trạng thái',
  'reports.reasonLabel':
    'Lý do báo cáo',
  'reports.reasonHint':
    'Tối đa 500 ký tự, sẽ được hiển thị trực tiếp cho các quản trị viên khác làm căn cứ để quyết định.',
  'reports.filterLabel':
    'Lọc theo trạng thái',
  'reports.filterAll':
    'Tất cả',
  'reports.listTitle':
    'Danh sách báo cáo',
  'reports.colTarget':
    'Nội dung mục tiêu',
  'reports.colReason':
    'Lý do báo cáo',
  'reports.colReporter':
    'Người báo cáo',
  'reports.colStatus':
    'Trạng thái',
  'reports.targetGone':
    '（Nội dung đã bị xóa）',
  'reports.author':
    'Tác giả: {name}',
  'reports.deleteTitle':
    'Xóa báo cáo #{id}',
  'reports.deleteMessage':
    'Báo cáo bị xóa là bản ghi báo cáo này; nội dung bị báo cáo không bị ảnh hưởng và không thể phục hồi. Tiếp tục không?',
  'reports.deleteConfirm':
    'Xóa báo cáo',
  'reports.deleted':
    'Báo cáo đã bị xóa.',
  'reports.deleteFailed':
    'Không thể xóa báo cáo.',
  'reports.updated':
    'Báo cáo đã được cập nhật.',
  'reports.saveFailed':
    'Không thể lưu báo cáo.',
  'reports.approveTitle':
    'Thông qua báo cáo #{id}',
  'reports.approveGoneMessage':
    'Nội dung {kind} #{id} bị báo cáo đã không còn tồn tại; báo cáo này chỉ sẽ được đánh dấu là đã xử lý.',
  'reports.approveMessage':
    'Sẽ xóa vĩnh viễn nội dung {kind} #{id} bị báo cáo (nếu là bài viết, tất cả bình luận dưới bài viết đó cũng sẽ được loại bỏ), đồng thời đánh dấu báo cáo này là đã xử lý. Tiếp tục không?',
  'reports.approveConfirm':
    'Thông qua và xóa bài viết',
  'reports.approveGoneDone':
    'Nội dung đã không còn tồn tại, báo cáo đã được đánh dấu là đã xử lý.',
  'reports.approveDone':
    'Đã xóa bài viết và đánh dấu báo cáo là đã xử lý.',
  'reports.approveFailed':
    'Không thể thông qua báo cáo.',
  'reports.approveTitleGone':
    'Nội dung đã bị xóa, chỉ đánh dấu báo cáo',
  'reports.approveTitleFull':
    'Xóa nội dung bị báo cáo và đánh dấu là đã xử lý',
  'reports.rejectTitle':
    'Báo cáo #{id} không hợp lệ',
  'reports.rejectMessage':
    'Không hợp lệ có nghĩa là nội dung bị báo cáo không cần xử lý; nội dung sẽ được giữ nguyên. Tiếp tục không?',
  'reports.rejectConfirm':
    'Đánh dấu là không hợp lệ',
  'reports.rejectDone':
    'Báo cáo đã được đánh dấu là không hợp lệ.',
  'reports.statusFailed':
    'Không thể cập nhật trạng thái báo cáo.',
  'reports.emptyTitle':
    'Hiện không có báo cáo',
  'reports.emptyBody':
    'Không có bản ghi nào với bộ lọc này.',
  'reports.rejectTitleAttr':
    'Giữ nội dung, chỉ đánh dấu báo cáo là không hợp lệ',
  'kind.post':
    'Bài viết',
  'kind.comment':
    'Bình luận',
  'reports.statusPending':
    'Chờ xử lý',
  'reports.statusResolved':
    'Đã xử lý',
  'reports.statusRejected':
    'Không hợp lệ',
  'title.forum':
    '{site}',
  'title.login':
    'Đăng nhập｜{site}',
  'title.newPost':
    'Tạo bài viết｜{site}',
  'title.profile':
    'Hồ sơ cá nhân｜{site}',
  'title.publicProfile':
    'Hồ sơ cá nhân công khai｜{site}',
  'title.following':
    'Đang theo dõi｜{site}',
  'title.adminUsers':
    'Quản lý người dùng｜{brand} Admin Console',
  'title.adminLogin':
    'Đăng nhập｜{brand} Admin Console',
  'title.adminPosts':
    'Bài viết diễn đàn｜{brand} Admin Console',
  'title.adminReports':
    'Quản lý báo cáo｜{brand} Admin Console',
  'title.adminMonitor': 'Giám sát hệ thống｜{brand} Quản trị',
  'title.adminLog': 'Nhật ký thao tác｜{brand} Quản trị',
  'title.adminStats': 'Xu hướng nội dung｜{brand} Quản trị',
  'title.adminExport': 'Xuất và thao tác hàng loạt｜{brand} Quản trị',
  'title.adminSessions': 'Đăng nhập & phiên làm việc｜{brand} Quản trị',
  'title.adminBlocks': 'Danh sách chặn IP｜{brand} Quản trị',
  'title.adminAnnouncements': 'Thông báo｜{brand} Quản trị',
};
