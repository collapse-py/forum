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
};
