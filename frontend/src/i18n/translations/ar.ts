/*
 * ar catalog (src/i18n/translations/ar.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const ar: Record<MessageKey, string> = {
  'common.cancel':
    'إلغاء',
  'common.save':
    'حفظ',
  'common.submitting':
    'جارٍ الإرسال...',
  'common.delete':
    'حذف',
  'common.edit':
    'تحرير',
  'common.search':
    'بحث',
  'common.loading':
    'جارٍ التحميل...',
  'common.loadFailed':
    'فشل التحميل',
  'common.refresh':
    'تحديث',
  'common.nextStep':
    'الخطوة التالية',
  'common.prevPage':
    'الصفحة السابقة',
  'common.nextPage':
    'الصفحة التالية',
  'common.create':
    'إنشاء',
  'common.publish':
    'نشر',
  'common.placeholder':
    '—',
  'common.backToHome':
    'العودة إلى الصفحة الرئيسية',
  'common.backToForumHome':
    'العودة إلى الصفحة الرئيسية للمنتدى',
  'common.backOnePage':
    'العودة إلى الصفحة السابقة',
  'error.request':
    'انتظر ثم أعد المحاولة.',
  'error.requestStatus':
    'فشل الطلب (HTTP {status})',
  'error.loginRequired':
    'سجّل الدخول أولاً ثم واصل.',
  'error.adminSessionExpired':
    'انتهت جلسة تسجيل الدخول، وسيتم العودة إلى صفحة تسجيل الدخول.',
  'error.fallbackLoad':
    'فشل التحميل',
  'error.fallbackSearch':
    'فشل البحث',
  'error.fallbackLike':
    'فشل الإعجاب',
  'error.fallbackComments':
    'فشل تحميل التعليقات',
  'error.fallbackCommentPost':
    'فشل إرسال التعليق',
  'error.fallbackReport':
    'فشل البلاغ',
  'error.fallbackProfile':
    'فشل قراءة البيانات الشخصية',
  'error.fallbackProfileSave':
    'فشل الحفظ',
  'error.fallbackPublish':
    'صيغة الرد عند النشر غير صحيحة',
  'error.fallbackUpload':
    'صيغة استجابة رفع الصورة غير صحيحة',
  'error.fallbackNotFound':
    'لم يتم العثور على هذا المستخدم',
  'error.fallbackFollow':
    'تعذّرت المتابعة',
  'auth.checking':
    'جارٍ التحقق من حالة تسجيل الدخول...',
  'auth.statusUnknown':
    'تعذر التحقق من حالة تسجيل الدخول',
  'auth.feedLoggedIn':
    'سجلت الدخول، ويمكنك إنشاء منشور',
  'auth.feedLoggedOut':
    'سجّل الدخول أولاً لإنشاء منشور',
  'auth.profileLoggedIn':
    'سجلت الدخول',
  'auth.profileLoggedOut':
    'سجّل الدخول أولاً لتحديث البيانات الشخصية',
  'auth.googleLogin':
    'تسجيل الدخول باستخدام Google',
  'auth.loginWithGoogle':
    'تسجيل الدخول باستخدام حساب Google',
  'auth.loginWithGoogleAdmin':
    'تسجيل الدخول باستخدام حساب Google الإداري',
  'auth.logout':
    'تسجيل الخروج',
  'install.button':
    'تثبيت التطبيق',
  'install.hint':
    'لا يوفر المتصفح الحالي تنبيهاً للتثبيت التلقائي. افتح قائمة المتصفح، ثم اختر «تثبيت التطبيق» أو «إضافة إلى الشاشة الرئيسية».',
  'bottomNav.label':
    'التنقل الرئيسي',
  'bottomNav.home':
    'الصفحة الرئيسية',
  'bottomNav.new':
    'إنشاء',
  'bottomNav.profile':
    'الملف الشخصي',
  'i18n.ariaLabel':
    'اختيار اللغة',
  'i18n.current':
    'اللغة: {name}',
  'feed.searchPlaceholder':
    'البحث في محتوى المنشورات',
  'feed.searchAriaLabel':
    'البحث عن المنشورات',
  'feed.searchResultsLabel':
    'نتائج البحث',
  'feed.postsLabel':
    'منشورات المنتدى',
  'feed.searchFailed':
    'فشل البحث، انتظر ثم أعد المحاولة.',
  'feed.searching':
    'جارٍ البحث...',
  'feed.searchMore':
    'جارٍ تحميل نتائج بحث إضافية...',
  'feed.searchMoreFailed':
    'فشل تحميل المزيد',
  'feed.searchFound':
    'تم العثور على {total} نتيجة',
  'feed.searchDegraded':
    '{base} (لم يتم تفعيل خدمة البحث، حاليًا تتم المقارنة بالكلمات المفتاحية في قاعدة البيانات)',
  'feed.searchTotal':
    'مجموع {total} نتيجة',
  'feed.searchNoResults':
    'لم يتم العثور على منشورات تحتوي على «{query}».',
  'feed.loadingPosts':
    'جارٍ تحميل المنشورات...',
  'feed.loadMorePosts':
    'تحميل منشورات إضافية...',
  'feed.postsFailed':
    'فشل تحميل المنشورات، انتظر ثم أعد المحاولة.',
  'feed.postsFailedShort':
    'فشل التحميل، انتظر ثم أعد المحاولة',
  'feed.scrollMore':
    'مرّر إلى الأسفل لتحميل المزيد',
  'feed.endOfFeed':
    'وصلت إلى آخر المنشورات',
  'feed.noPosts':
    'لا توجد منشورات بعد. اترك فكرتك الأولى.',
  'feed.likeFailed':
    'فشل الإعجاب أو إلغاء الإعجاب، انتظر ثم أعد المحاولة.',
  'post.authorAnonymous':
    'مجهول',
  'post.report':
    'الإبلاغ عن المنشور',
  'post.imageAlt':
    'صورة المنشور',
  'post.unlike':
    'إلغاء الإعجاب',
  'post.like':
    'إعجاب',
  'post.reply':
    'الرد',
  'comment.loading':
    'جارٍ تحميل التعليقات...',
  'comment.none':
    'لا توجد تعليقات',
  'comment.loadFailed':
    'فشل تحميل التعليقات، انتظر ثم أعد المحاولة',
  'comment.placeholder':
    'اكتب تعليقك...',
  'comment.max':
    'الحد الأقصى 2000 حرف',
  'comment.submit':
    'إرسال تعليق',
  'comment.failed':
    'فشل إرسال التعليق، انتظر ثم أعد المحاولة.',
  'comment.report':
    'الإبلاغ عن التعليق',
  'comment.more':
    'تحميل تعليقات إضافية...',
  'report.reasonPlaceholder':
    'أدخل سبب البلاغ (الحد الأقصى 500 حرف)',
  'report.note':
    'سيُرسل البلاغ إلى مشرفي الموقع',
  'report.formLabel':
    'حقل إدخال البلاغ',
  'report.submit':
    'إرسال البلاغ',
  'report.failed':
    'فشل البلاغ، انتظر ثم أعد المحاولة.',
  'report.sent':
    'تم إرسال البلاغ، شكرًا على إبلاغك.',
  'newPost.avatarYou':
    'أنت',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'إنشاء منشور',
  'newPost.loginFirst':
    'سجّل الدخول باستخدام Google أولاً قبل إنشاء منشور',
  'newPost.contentPlaceholder':
    'شارك أفكارك...',
  'newPost.addImage':
    'إضافة صورة',
  'newPost.emailPrivate':
    'سيظل email سريًا',
  'newPost.submit':
    'نشر منشور',
  'newPost.publishing':
    'جارٍ النشر...',
  'newPost.uploading':
    'جارٍ تحميل الصورة...',
  'newPost.failed':
    'فشل النشر، حاول مرة أخرى لاحقًا.',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'الملف الشخصي',
  'profile.edit':
    'تحرير',
  'profile.loginPrompt':
    'بعد تسجيل الدخول، يمكنك إعداد الاسم المستعار ونبذة المنتدى.',
  'profile.nicknameLabel':
    'الاسم المستعار في المنتدى',
  'profile.notSet':
    'غير محدد',
  'profile.notSetBio':
    'لم يتم إعداد نبذة.',
  'profile.nicknameInput':
    'الاسم المستعار',
  'profile.nicknamePlaceholder':
    'أدخل الاسم المستعار',
  'profile.nicknameHint':
    'سيُعرض الاسم المستعار على المنشورات التي تنشرها، بحد أقصى 30 حرفًا.',
  'profile.bioLabel':
    'نبذة',
  'profile.bioPlaceholder':
    'تعرّف بنفسك (اختياري)',
  'profile.bioHint':
    'بحد أقصى 500 حرفًا.',
  'profile.updated':
    'تم تحديث الملف الشخصي.',
  'profile.saving':
    'جارٍ الحفظ...',
  'profile.saveFailed':
    'فشل الحفظ، حاول مرة أخرى لاحقًا.',
  'profile.loadFailed':
    'فشل التحميل، حاول مرة أخرى لاحقًا.',
  'profile.followingEntry':
    'متابَعون',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'الملف الشخصي العام',
  'publicProfile.avatar':
    'مستخدم مجهول',
  'publicProfile.loading':
    'جارٍ التحميل...',
  'publicProfile.invalidLinkName':
    'رابط الملف الشخصي العام غير صالح',
  'publicProfile.invalidLinkBio':
    'انتهِ إلى الملف الشخصي من خلال اسم مؤلف المنشور في المنتدى.',
  'publicProfile.notFound':
    'لم يتم العثور على هذا المستخدم',
  'publicProfile.anonymous':
    'مستخدم مجهول',
  'publicProfile.noBio':
    'لم يحدد هذا المستخدم معلوماته العامة بعد.',
  'publicProfile.loadFailed':
    'فشل تحميل المعلومات العامة.',
  'publicProfile.postsLabel':
    'المنشورات',
  'publicProfile.emptyPosts':
    'لم ينشر هذا المستخدم شيئًا بعد.',

  'follow.label':
    'متابعة هذا المستخدم',
  'follow.action':
    'متابعة',
  'follow.actionDone':
    'تتم المتابعة',
  'follow.unfollow':
    'إلغاء المتابعة',
  'follow.done':
    'أنت الآن تتابع هذا المستخدم.',
  'follow.failed':
    'تعذّرت المتابعة. حاول مرة أخرى لاحقًا.',

  'following.peopleLabel':
    'المستخدمون الذين تتابعهم',
  'following.postsLabel':
    'منشورات المستخدمين الذين تتابعهم',
  'following.peopleLoading':
    'جارٍ تحميل قائمة المتابعة...',
  'following.emptyPeople':
    'لا تتابع أي شخص بعد. اضغط «متابعة» على منشور، أو تابع شخصًا من ملفه العام.',
  'following.emptyPosts':
    'لم ينشر المستخدمون الذين تتابعهم شيئًا بعد.',
  'following.peopleFailed':
    'تعذّر تحميل قائمة المتابعة.',
  'following.postsFailed':
    'تعذّر تحميل منشورات المتابَعون.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'مرحبًا بعودتك',
  'login.body':
    '{site} مساحة للجميع ممن يريدون تدوين أفكارهم. لا حاجة إلى نموذج تسجيل — حساب Google واحد يكفي للبدء في نشر المنشورات.',
  'login.browseFirst':
    'اطّلع على الصفحة الرئيسية أولًا',
  'admin.skipToMain':
    'الانتقال إلى المحتوى الرئيسي',
  'admin.railLabel':
    'قائمة الإدارة',
  'admin.railBrandAria':
    'الصفحة الرئيسية لـ {site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'الوظائف الرئيسية',
  'admin.railGovernance':
    'الحوكمة',
  'admin.railMode':
    'وضع المسؤول',
  'admin.railExit':
    'العودة إلى المنتدى',
  'admin.topbarMenu':
    'تبديل قائمة الإدارة',
  'admin.statusOnline':
    'متصل',
  'admin.topbarForum':
    'المنتدى',
  'admin.logoutFailed':
    'فشل تسجيل الخروج، حاول مرة أخرى لاحقًا.',
  'admin.navUsers':
    'إدارة المستخدمين',
  'admin.navPosts':
    'منشورات المنتدى',
  'admin.navReports':
    'إدارة البلاغات',
  'admin.listLoadFailed':
    'فشل التحميل',
  'admin.dlgClose':
    'إغلاق النافذة',
  'admin.dlgConfirm':
    'تأكيد',
  'admin.dlgSave':
    'حفظ',
  'admin.dlgApplyTags':
    'تطبيق الأوسمة',
  'admin.dlgNoTags':
    'لا توجد أوسمة قابلة للتطبيق حاليًا؛ أضيف واحدة أدناه من «إدارة الأوسمة».',
  'users.title':
    'إدارة المستخدمين',
  'users.contentAction':
    'المحتوى',
  'users.updateContentFailed':
    'فشلت عملية المحتوى.',
  'users.updated':
    'تم تحديث المحتوى.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'راجع إحصاءات أنشطة المستخدمين وأوسمتهم وحالات الحسابات، وعالج الحظر وإدارة المحتوى لكل منشور.',
  'users.refresh':
    'تحديث البيانات',
  'users.statTotal':
    'إجمالي المستخدمين',
  'users.statActive':
    'مفعل',
  'users.statSuspended':
    'محظور',
  'users.statContent':
    'إجمالي المنشورات/التعليقات',
  'users.count':
    '{count} مستخدمين',
  'users.tagsCount':
    '{count} أوسمة',
  'users.loadFailed':
    'فشل تحميل بيانات المستخدمين.',
  'users.tagsLoadFailed':
    'فشل تحميل بيانات الأوسمة.',
  'users.panelTitle':
    'مستخدمو المنتدى',
  'users.tagsPanelTitle':
    'أوسمة المستخدمين',
  'users.colUser':
    'المستخدم',
  'users.colTags':
    'الأوسمة',
  'users.colStatus':
    'الحالة',
  'users.colPosts':
    'المنشورات',
  'users.colComments':
    'التعليقات',
  'users.colLikes':
    'الإعجابات',
  'users.colLastActivity':
    'آخر نشاط',
  'users.colActions':
    'الإجراءات',
  'users.nicknameUnset':
    'لم يُحدد الاسم المستعار',
  'users.notSet':
    'غير محدد',
  'users.statusActive':
    'مفعل',
  'users.statusSuspended':
    'محظور',
  'users.emptyTitle':
    'لا توجد بيانات مستخدمين حاليًا',
  'users.emptyBody':
    'لن تظهر هذه الصفحة فارغة إلا إذا لم يستخدم المستخدمون Google لتسجيل الدخول إلى المنتدى.',
  'users.tagsEmptyTitle':
    'لا توجد أوسمة حاليًا',
  'users.tagsEmptyBody':
    'أنشئ وسومًا أولًا حتى تتمكن من تطبيقها على مستخدم في قائمة المستخدمين.',
  'users.addTag':
    'إضافة وسوم',
  'users.colName':
    'الاسم',
  'users.colCreated':
    'وقت الإنشاء',
  'users.colUpdated':
    'وقت التحديث',
  'users.renameTag':
    'إعادة تسمية',
  'users.suspend':
    'حظر',
  'users.restore':
    'استعادة',
  'users.statusDialogTitle':
    'تطبيق {action} على هذا المستخدم',
  'users.statusSuspendMessage':
    'لن يتمكن {email} من تسجيل الدخول إلى المنتدى مرة أخرى، وسيُحفظ المنشورات والتعليقات الحالية. المتابعة؟',
  'users.statusRestoreMessage':
    'سيستعيد {email} صلاحيات تسجيل الدخول ونشر المنشورات. المتابعة؟',
  'users.userSuspended':
    'تم حظر المستخدم.',
  'users.userRestored':
    'تمت استعادة المستخدم.',
  'users.updateStatusFailed':
    'فشل تحديث حالة المستخدم.',
  'users.editTagsTitle':
    'تعديل الوسوم · {user}',
  'users.editTagsMessage':
    'حدد الوسوم المراد تطبيقها؛ يؤدي إلغاء تحديدها جميعًا إلى إزالة جميع وسوم هذا المستخدم.',
  'users.tagsUpdated':
    'تمت تحديث وسوم المستخدم.',
  'users.updateTagsFailed':
    'فشل تحديث وسوم المستخدم.',
  'users.contentLoadFailed':
    'فشل تحميل المحتوى.',
  'users.contentLoadFailedShort':
    'فشل تحميل المحتوى.',
  'users.contentPanelTitle':
    'محتوى المستخدم',
  'users.contentCount':
    '{posts} منشور · {comments} تعليق',
  'users.contentLoading':
    'جارٍ تحميل المنشورات والتعليقات…',
  'users.addPost':
    'إضافة منشور',
  'users.addComment':
    'إضافة تعليق',
  'users.postsColumn':
    'المنشورات',
  'users.commentsColumn':
    'التعليقات',
  'users.noPosts':
    'لا توجد منشورات بعد',
  'users.noComments':
    'لا توجد تعليقات بعد',
  'users.postRef':
    'المنشور #{id}',
  'users.editRecordTitle':
    'تعديل {kind} #{id}',
  'users.deleteRecordTitle':
    'حذف {kind} #{id}',
  'users.deleteRecordMessage':
    'بعد الحذف لن تتمكن من استردادها، وسيتم أيضًا إزالة الإعجابات المرتبطة والبيانات ذات الصلة. المتابعة؟',
  'users.contentLabel':
    'المحتوى',
  'users.addPostTitle':
    'إضافة منشور',
  'users.addPostMessage':
    'سيُنشر هذا المحتوى باسم المستخدم هذا، ولا يمكن التلاعب بحقل المؤلف.',
  'users.postContentLabel':
    'محتوى المنشور',
  'users.postContentPlaceholder':
    'أدخل محتوى المنشور',
  'users.pickPostTitle':
    'اختر المنشور',
  'users.postIdLabel':
    'معرّف المنشور ID',
  'users.postIdPlaceholder':
    'رقم المنشور الذي تريد التعليق عليه',
  'users.addCommentTitle':
    'إضافة تعليق',
  'users.commentContentLabel':
    'محتوى التعليق',
  'users.commentContentPlaceholder':
    'أدخل محتوى التعليق',
  'users.createTagTitle':
    'إضافة وسوم',
  'users.createTagMessage':
    'يمكن استخدام الوسوم لتصنيف المستخدمين، مثل «مضيف» أو «نشط» أو «محظور».',
  'users.tagNameLabel':
    'اسم الوسم',
  'users.tagNamePlaceholder':
    'الحد الأقصى 50 حرفًا',
  'users.renameTagTitle':
    'إعادة تسمية الوسم',
  'users.renameTagMessage':
    'سيشاهد جميع المستخدمين الذين يستخدمون هذا الوسم الاسم الجديد.',
  'users.deleteTagTitle':
    'حذف الوسم «{name}»',
  'users.deleteTagMessage':
    'بعد الحذف، ستُزال هذه الوسوم عن جميع المستخدمين أيضًا، ولن يمكن استردادها. المتابعة؟',
  'users.deleteTagConfirm':
    'حذف الوسم',
  'users.tagCreated':
    'تم إنشاء الوسم.',
  'users.createTagFailed':
    'فشل إنشاء الوسم.',
  'users.tagUpdated':
    'تم تحديث الوسم.',
  'users.updateTagFailed':
    'فشل تحديث الوسم.',
  'users.tagDeleted':
    'تم حذف الوسم.',
  'users.deleteTagFailed':
    'فشل حذف الوسم.',
  'users.refreshDone':
    'تم التحديث إلى أحدث البيانات.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'لوحة التحكم',
  'users.signinBody':
    'إدارة محتوى موحدة تجعل كل عملية مراجعة واضحة وسريعة وقابلة للتتبع.',
  'users.signinStep1':
    'التحقق الآمن',
  'users.signinStep2':
    'إدارة المستخدمين',
  'users.signinStep3':
    'مراجعة المحتوى',
  'users.signinPanelTitle':
    'تسجيل الدخول إلى Admin Console',
  'users.signinPanelBody':
    'يتوفر Admin Console فقط لحسابات مسؤولي Google المخولة؛ سجّل الدخول بصفتك مسؤولًا.',
  'posts.title':
    'منتدى المنشورات',
  'posts.searching':
    'جارٍ البحث…',
  'posts.searchDegraded':
    '(لم يتم تمكين خدمة البحث؛ تتم مقارنة الكلمات المفتاحية باستخدام قاعدة البيانات)',
  'posts.searchSummary':
    'عثر على {total} نتيجة لـ«{query}»، ويظهر في هذه الصفحة {shown} نتيجة',
  'posts.pendingCount':
    '{count} قيد الانتظار',
  'posts.pageSummary':
    'الصفحة {page} / {pages}، يحتوي هذا الصف على {count} منشور',
  'posts.listLoadFailed':
    'فشل تحميل قائمة المنشورات.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'إنشاء المنشورات وتعديلها وحذفها، مع إدارة المحتوى على مستوى التعليقات.',
  'posts.toReports':
    'إدارة البلاغات',
  'posts.editorTitleNew':
    'إضافة منشور',
  'posts.editorTitleEdit':
    'تعديل المنشور #{id}',
  'posts.editorNote':
    'ينشُر كإداري؛ يأتي المؤلف من هوية تسجيل الدخول، ولا يمكن التلاعب ببيانات المؤلف في الطلب.',
  'posts.cancelEdit':
    'إلغاء التحرير',
  'posts.contentLabel':
    'محتوى المنشور',
  'posts.contentPlaceholder':
    'أدخل محتوى المنشور',
  'posts.saveChanges':
    'حفظ التغييرات',
  'posts.emptyContent':
    'لا يجوز أن يكون محتوى المنشور فارغًا.',
  'posts.saving':
    'جارٍ الحفظ…',
  'posts.saved':
    'تم تحديث المنشور.',
  'posts.published':
    'تم نشر المنشور.',
  'posts.saveFailed':
    'فشل الحفظ.',
  'posts.deleteTitle':
    'حذف المنشور #{id}',
  'posts.deleteMessage':
    'بعد الحذف لن تتمكن من استردادها، وسيتم أيضًا إزالة جميع التعليقات الموجودة أسفل هذا المنشور. المتابعة؟',
  'posts.deleteConfirm':
    'حذف المنشور',
  'posts.deleted':
    'تم حذف المنشور.',
  'posts.deleteFailed':
    'فشل حذف المنشور.',
  'posts.commentUpdated':
    'تم تحديث التعليق.',
  'posts.commentActionFailed':
    'فشل تنفيذ الإجراء الخاص بالتعليق.',
  'posts.pickPostTitle':
    'اختر المنشور الذي ينتمي إليه التعليق',
  'posts.pickPostMessage':
    'الافتراضي هو المنشور في هذا الصف؛ إذا أردت ربطه بمنشور آخر، فغيّر رقم المنشور إلى الرقم الصحيح.',
  'posts.postIdLabel':
    'معرّف المنشور ID',
  'posts.addCommentAtTitle':
    'إضافة تعليق إلى المنشور #{id}',
  'posts.addCommentMessage':
    'سيُنشر هذا التعليق بصفتك مسؤولًا.',
  'posts.commentContentLabel':
    'محتوى التعليق',
  'posts.commentContentPlaceholder':
    'أدخل محتوى التعليق',
  'posts.add':
    'إضافة',
  'posts.editCommentTitle':
    'تعديل التعليق #{id}',
  'posts.deleteCommentTitle':
    'حذف التعليق #{id}',
  'posts.deleteCommentMessage':
    'بعد الحذف لن تتمكن من استردادها. المتابعة؟',
  'posts.listTitle':
    'قائمة المنشورات',
  'posts.clearSearch':
    'مسح البحث',
  'posts.searchLabel':
    'بحث بالكلمات المفتاحية',
  'posts.searchPlaceholder':
    'محتوى المنشور أو Email الكامل للمؤلف',
  'posts.searchHint':
    'مرتبطة حسب الصلة؛ أدخل Email كاملاً للبحث عن جميع المنشورات الخاصة بهذا المستخدم. يستبدل البحث التصفح، وتُظهر النتائج بحد أقصى 25 نتيجة.',
  'posts.searchTotal':
    'إجمالي نتائج البحث {total}',
  'posts.searchFailed':
    'فشل البحث.',
  'posts.searchStatusFailed':
    'فشل البحث',
  'posts.colContentImage':
    'المحتوى والصورة',
  'posts.colEngagement':
    'التفاعل',
  'posts.colComments':
    'التعليقات',
  'posts.colAuthor':
    'المؤلف',
  'posts.imageAlt':
    'الوصف البديل لصورة المنشور',
  'posts.likes':
    '{count} إعجاب',
  'posts.author':
    'المؤلف: {name}',
  'posts.emptyTitle':
    'لا توجد حالياً منشورات في المنتدى',
  'posts.emptyBody':
    'يمكنك إنشاء أول منشور باستخدام المحرر أعلاه.',
  'posts.emptySearchTitle':
    'لا توجد منشورات مطابقة',
  'posts.emptySearchBody':
    'لم يتم العثور على منشور يطابق «{query}». جرّب كلمة بحث أخرى.',
  'posts.commentCount':
    '{count} تعليقاً',
  'posts.noComments':
    'لا توجد تعليقات',
  'posts.reportsTitle':
    'البلاغات قيد المعالجة',
  'posts.allReports':
    'كل البلاغات',
  'posts.colReportedContent':
    'المحتوى المبلّغ عنه',
  'posts.colReason':
    'سبب البلاغ',
  'posts.colReporter':
    'صاحب البلاغ',
  'posts.colTime':
    'الوقت',
  'posts.colVerdict':
    'الحكم',
  'posts.emptyReportsTitle':
    'لا توجد بلاغات قيد المعالجة',
  'posts.emptyReportsBody':
    'تم الفصل في جميع البلاغات.',
  'posts.verdictResolved':
    'مُعالَج',
  'posts.verdictRejected':
    'غير مبرر',
  'posts.verdictDialogTitle':
    'تحديد البلاغ #{id} على أنه «{label}»',
  'posts.verdictDialogMessage':
    'بعد التحديد، سيغادر البلاغ قائمة البلاتيج قيد المعالجة، لكن ستبقى البيانات محفوظة في صفحة إدارة البلاغات. هل تريد الاستمرار؟',
  'posts.verdictConfirm':
    'تحديد {label}',
  'posts.verdictDone':
    'تم تحديد البلاغ على أنه {label}.',
  'posts.verdictFailed':
    'فشل تحديث حالة البلاغ.',
  'reports.title':
    'إدارة البلاغات',
  'reports.listSummary':
    '{count} بلاغ · {filter}',
  'reports.listLoadFailed':
    'فشل تحميل قائمة البلاغات.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'الفصل في البلاغات فردياً: يؤدي «قبول» إلى حذف المحتوى المبلّغ عنه، بينما يُبقي «عدم جدوى» النص كما هو. عند إعادته إلى «قيد المعالجة»، سيُحذف وقت الحكم السابق.',
  'reports.backToPosts':
    'العودة إلى المنشورات',
  'reports.editorTitle':
    'تحرير البلاغ #{id}',
  'reports.editorNote':
    'يمكنك تصحيح سبب البلاغ وحالته؛ فالهدف وصاحب البلاغ سجلات قائمة، ولن تُغيَّرا هنا.',
  'reports.targetTypeLabel':
    'نوع الهدف',
  'reports.targetIdLabel':
    'الهدف ID',
  'reports.reporterEmailLabel':
    'صاحب البلاغ Email',
  'reports.statusLabel':
    'الحالة',
  'reports.reasonLabel':
    'سبب البلاغ',
  'reports.reasonHint':
    'حد أقصى 500 حرف، وستُعرض مباشرة على المديرين الآخرين كأساس للفصل.',
  'reports.filterLabel':
    'تصفية حسب الحالة',
  'reports.filterAll':
    'الكل',
  'reports.listTitle':
    'قائمة البلاغات',
  'reports.colTarget':
    'المحتوى المستهدف',
  'reports.colReason':
    'سبب البلاغ',
  'reports.colReporter':
    'صاحب البلاغ',
  'reports.colStatus':
    'الحالة',
  'reports.targetGone':
    '（المحتوى محذوف）',
  'reports.author':
    'المؤلف: {name}',
  'reports.deleteTitle':
    'حذف البلاغ #{id}',
  'reports.deleteMessage':
    'سيُحذف سجل البلاغ نفسه، ولن يتأثر المحتوى المبلّغ عنه، ولن يمكن استرجاعه. هل تريد الاستمرار؟',
  'reports.deleteConfirm':
    'حذف البلاغ',
  'reports.deleted':
    'تم حذف البلاغ.',
  'reports.deleteFailed':
    'فشل حذف البلاغ.',
  'reports.updated':
    'تم تحديث البلاغ.',
  'reports.saveFailed':
    'فشل حفظ البلاغ.',
  'reports.approveTitle':
    'قبول البلاغ #{id}',
  'reports.approveGoneMessage':
    'المحتوى المبلّغ عنه من نوع {kind} #{id} غير موجود، وسيُحدَّد هذا البلاغ فقط على أنه مُعالَج.',
  'reports.approveMessage':
    'سيؤدي ذلك إلى حذف المحتوى المبلّغ عنه من نوع {kind} #{id} بشكل دائم（إذا كان منشوراً، فسيتم إزالة جميع التعليقات الموجودة أسفوله أيضًا）ووسم هذا البلاغ على أنه مُعالَج. هل تريد الاستمرار؟',
  'reports.approveConfirm':
    'قبول وحذف المنشور',
  'reports.approveGoneDone':
    'المحتوى غير موجود، وتم تحديد البلاغ على أنه مُعالَج.',
  'reports.approveDone':
    'تم حذف المحتوى وتحديد البلاغ على أنه مُعالَج.',
  'reports.approveFailed':
    'فشل قبول البلاغ.',
  'reports.approveTitleGone':
    'تم حذف المحتوى، وسيتم تحديد البلاغ فقط',
  'reports.approveTitleFull':
    'حذف المحتوى المبلّغ عنه وتحديده على أنه مُعالَج',
  'reports.rejectTitle':
    'البلاغ #{id} غير مبرر',
  'reports.rejectMessage':
    'عدم الجدارة يعني أن المحتوى المبلّغ عنه لا يحتاج إلى معالجة، وسيبقى كما هو. هل تريد الاستمرار؟',
  'reports.rejectConfirm':
    'تحديد على أنه غير مبرر',
  'reports.rejectDone':
    'تم تحديد البلاغ على أنه غير مبرر.',
  'reports.statusFailed':
    'فشل تحديث حالة البلاغ.',
  'reports.emptyTitle':
    'لا توجد حالياً بلاغات',
  'reports.emptyBody':
    'لا توجد سجلات تحت هذا عامل التصفية.',
  'reports.rejectTitleAttr':
    'إبقاء المحتوى وتحديد البلاغ فقط على أنه غير مبرر',
  'kind.post':
    'منشور',
  'kind.comment':
    'تعليق',
  'reports.statusPending':
    'قيد المعالجة',
  'reports.statusResolved':
    'مُعالَج',
  'reports.statusRejected':
    'غير مبرر',
  'title.forum':
    '{site}',
  'title.login':
    'تسجيل الدخول｜{site}',
  'title.newPost':
    'إضافة منشور｜{site}',
  'title.profile':
    'الملف الشخصي｜{site}',
  'title.publicProfile':
    'الملف الشخصي العام｜{site}',
  'title.following':
    'المتابَعون｜{site}',
  'title.adminUsers':
    'إدارة المستخدمين｜لوحة تحكم {brand}',
  'title.adminLogin':
    'تسجيل الدخول｜لوحة تحكم {brand}',
  'title.adminPosts':
    'منشورات المنتدى｜لوحة تحكم {brand}',
  'title.adminReports':
    'إدارة البلاغات｜لوحة تحكم {brand}',
};
