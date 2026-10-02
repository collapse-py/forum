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
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': 'ساعة',
  'common.oneDay': 'يوم',
  'common.sevenDays': '7 أيام',
  'common.thirtyDays': '30 يومًا',
  'common.oneYear': 'سنة',
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
  'admin.navMonitor': 'مراقبة النظام',
  'admin.navLog': 'سجل الإجراءات',
  'admin.navStats': 'اتجاهات المحتوى',
  'admin.navExport': 'تصدير وإجراءات جماعية',
  'admin.navSessions': 'الدخول والجلسات',
  'admin.navBlocks': 'قائمة حظر العناوين',
  'admin.navAnnouncements': 'الإعلانات',
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

  'monitor.title': 'مراقبة النظام',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'عرض مباشر لحالة الخدمات وحجم الطلبات وتوزيع زمن الاستجابة وعدادات محددات المعدل.',
  'monitor.refresh': 'تحديث',
  'monitor.refreshing': 'جارٍ التحميل…',
  'monitor.autoRefresh': 'تحديث تلقائي',
  'monitor.autoRefreshOn': 'تحديث تلقائي كل {seconds} ثانية',
  'monitor.autoRefreshOff': 'التحديث التلقائي متوقف مؤقتًا',
  'monitor.nextUpdate': 'التحديث خلال {seconds} ثانية',
  'monitor.loadFailed': 'تعذّر تحميل بيانات المراقبة.',
  'monitor.loadFailedHint': 'تأكد من تسجيل الدخول بحساب مدير وأن الخادم ما زال يعمل.',
  'monitor.pausedHint': 'التحديث التلقائي متوقف مؤقتًا، وتعرض الشاشة نتيجة آخر قراءة ناجحة.',
  'monitor.visibilityPaused': 'علامة التبويب في الخلفية، لذلك تم إيقاف التحديث التلقائي مؤقتًا.',
  'monitor.lastUpdated': 'حُدِّث في {time}',
  'monitor.probeTook': 'فحص الاعتماديات: {ms} مللي ثانية',
  'monitor.unreachable': 'الخادم لا يستجيب. الشاشة متوقفة عند آخر قراءة ناجحة.',
  'monitor.depsTitle': 'حالة الخدمات',
  'monitor.depsNote': 'كل قراءة تفحص كل اعتمادية مرة واحدة فعليًا؛ مهلة كل اعتمادية ثانيتان، والثلاثة تعمل بالتوازي.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'محرك البحث',
  'monitor.stateOk': 'سليم',
  'monitor.stateDown': 'غير متاح',
  'monitor.stateDisabled': 'غير مفعّل',
  'monitor.depSearchFallback': 'الإعداد ES_URL غير موجود، لذا يستخدم البحث مطابقة كلمات مفتاحية في MySQL.',
  'monitor.depDisabled': 'لم يتم تمرير عميل Redis، لذا ميزات الوسائط معطّلة.',
  'monitor.depLatency': 'استجاب خلال {ms} مللي ثانية',
  'monitor.depKeys': '{count} مفتاحًا',
  'monitor.depMemory': 'الذاكرة {size}',
  'monitor.depPoolUsage': 'الاتصالات {inUse}/{open} (الحد {max})',
  'monitor.depPoolWait': '{count} مرة انتظار، بإجمالي {ms} مللي ثانية',
  'monitor.depRedisPool': 'إصابات {hits} / إخفاقات {misses}',
  'monitor.depEngineMysql': 'مطابقة كلمات مفتاحية في MySQL',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'ملخص الطلبات',
  'monitor.statUptime': 'مدة التشغيل',
  'monitor.statRequests': 'إجمالي الطلبات',
  'monitor.statErrorRate': 'معدل الأخطاء',
  'monitor.statP95': 'زمن الاستجابة P95',
  'monitor.statInFlight': 'قيد المعالجة',
  'monitor.statGoroutines': 'Goroutines',
  'monitor.statHeap': 'ذاكرة الكومة',
  'monitor.statDbPool': 'اتصالات قاعدة البيانات',
  'monitor.statRateLimited': 'محجوبة بالمعدل',
  'monitor.statCountWithPeak': 'القمة {peak}',
  'monitor.statCountWithInUse': '{inUse} مستخدمة، {idle} خاملة',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} نواة منطقية · {gc} دورة جمع قمامة',
  'monitor.noData': 'لا توجد طلبات بعد.',
  'monitor.noDataBody': 'ستظهر هنا الطلبات التي وردت منذ بدء الخدمة، وهذه اللوحة فارغة حاليًا.',
  'monitor.timelineTitle': 'الحركة خلال آخر {minutes} دقيقة',
  'monitor.timelineNote': 'الإجماليات تراكمية منذ بدء هذه العملية وتُصفَّر عند إعادة التشغيل؛ كل مئين من زمن الاستجابة هو الحد الأعلى لدلو في المخطط التكراري، لذا يأخذ قيمًا متقطعة فقط. الدقيقة بلا عمود تعني عدم وجود حركة فيها.',
  'monitor.timelineLive': 'هذه العملية',
  'monitor.timelineHistory': 'قبل إعادة التشغيل',
  'monitor.timelineLegendVolume': 'الطلبات',
  'monitor.timelineLegendError': 'أخطاء 5xx',
  'monitor.timelinePeak': 'القمة {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'لا توجد مجاميع تاريخية في قاعدة البيانات؛ وإذا تعذّر على بدء التشغيل قراءتها (الجدول مفقود أو الصلاحية ناقصة) فلن يظهر سوى العملية الحالية.',
  'monitor.routesTitle': 'حسب المسار',
  'monitor.routesNote': 'تُوحَّد المسارات (المعرّفات الرقمية وعناوين البريد تصبح :id)، لذا تُحسب المعرّفات المختلفة للمسار نفسه معًا.',
  'monitor.colRoute': 'المسار',
  'monitor.colCount': 'الطلبات',
  'monitor.colAvg': 'المتوسط',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'الأبطأ',
  'monitor.colErrors': 'الأخطاء',
  'monitor.routeOther': 'أخرى (بلغ الحد الأقصى للمسارات)',
  'monitor.clientsTitle': 'عناوين المصادر',
  'monitor.clientsNote': 'مرتّبة تنازليًا حسب عدد الطلبات. العنوان المأخوذ من X-Forwarded-For أو X-Real-IP لم يُتحقَّق منه عبر وكيل موثوق — تأكّد من أن الوكيل يكتب فوق هذه الترويسات قبل التصرّف بناءً عليها.',
  'monitor.noClients': 'لا توجد مصادر متابَعة بعد.',
  'monitor.noClientsBody': 'يُسجَّل كل طلب على عنوان مصدره. هذا الجدول فارغ حاليًا.',
  'monitor.clientsDropped': 'بلغ عدد المصادر الحد {limit}، قد تم نقل {count} من العناوين التي مضى وقت طويل منذ آخر ظهور لها خارج المتابعة. وجود هذا السطر يعني أن القائمة ناقصة، لا أن هؤلاء وحدهم زاروا الموقع.',
  'monitor.colIp': 'العنوان',
  'monitor.colSource': 'المصدر',
  'monitor.colRateLimited': 'محجوبة بالمعدل',
  'monitor.colBanned': 'مرفوضة بالحظر',
  'monitor.colLastRoute': 'آخر مسار',
  'monitor.colActions': 'الإجراءات',
  'monitor.colBlock': 'حظر',
  'monitor.blocking': 'جارٍ الحظر…',
  'monitor.sourcePeer': 'طرف الاتصال',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.trustLegacy': 'TRUSTED_PROXY_CIDRS غير مُعرَّف: يحتفظ الخادم بالسلوك القديم ويُعطي الأولوية لعنصر X-Forwarded-For الأيسر. حتى تتأكد من وجود وكيل أمام الخادم يعيد كتابة هذه الترويسات فعليًا ومن أن المستخدمين لا يستطيعون تجاوزه، يمكن تجاوز حد المعدل وحظر عناوين IP بترويسة واحدة مزوّرة، ولا يصلح عنوان المصدر في سجل التدقيق دليلًا.',
  'monitor.trustConfigured': 'الوكلاء الموثوقون مُعرَّفون: لا يُقبَل X-Forwarded-For / X-Real-IP إلا إذا كان النظير على الاتصال ضمن إحدى هذه الشبكات، وإلا فيُستخدم عنوان النظير. الشبكات السارية: {cidrs}.',
  'monitor.trustBroken': 'تم تعريف TRUSTED_PROXY_CIDRS لكن لا يمكن تحليل أي عنصر كمدى CIDR ({declared})، لذا لا يزال السلوك القديم بلا تعريف هو الساري.',
  'monitor.trustPartial': 'لا يمكن تحليل العناصر التالية من TRUSTED_PROXY_CIDRS كنطاقات CIDR ({invalid})، لذا لا يُقبَل أي رأس مُعاد توجيهه من هذه النطاقات أبدًا. الطلبات التي تصل عبر هذه النطاقات تُجمَّع حسب عنوان النظير على الاتصال، أي أنها تتشارك حصة واحدة لحدّ المعدل واستعلامًا واحدًا لقائمة الحظر.',
  'monitor.blockTitle': 'حظر {ip}',
  'monitor.blockMessage': 'ستُرفض طلبات الكتابة من هذا العنوان (منشورات، تعليقات، إعجابات، بلاغات، رفع صور، وتحويلات تسجيل الدخول) لمدة {duration}. أما القراءة فلا تتأثر. هل تريد الحظر؟',
  'monitor.blockReason': 'محظور من صفحة المراقبة',
  'monitor.blocked': 'حُظر {ip}',
  'monitor.blockFailed': 'فشلت عملية الحظر.',
  'monitor.limitsTitle': 'محددات المعدل',
  'monitor.limitsNote': 'لكل مجموعة حصة خاصة تناسب تكلفة نقاط النهاية الخاصة بها؛ وعدد المحجوبات هو إجمالي ردود 429 منذ بدء هذه العملية.',
  'monitor.colLimiter': 'المحدد',
  'monitor.colBudget': 'الحصة',
  'monitor.colAllowed': 'المسموح',
  'monitor.colBlocked': 'المحجوب',
  'monitor.colTracked': 'مصادر متابَعة',
  'monitor.colBlockedRate': 'نسبة الحجب',
  'monitor.limitContent': 'كتابة المحتوى',
  'monitor.limitUpload': 'رفع الصور',
  'monitor.limitAuth': 'تسجيل الدخول عبر OAuth',
  'monitor.limitBudget': '{limit} لكل {window} ثانية',
  'monitor.limitUnknown': '(غير معروف)',
  'monitor.noLimits': 'لا تتوفر محددات معدل.',
  'monitor.noLimitsBody': 'لم يتم إنشاء محددات المعدل بعد، لذا عداداتها غير متاحة.',

  'log.title': 'سجل الإجراءات',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'راجع كل تغيير أُجري في لوحة الإدارة: من ومتى وعلى أي هدف، وأي الحقول تغيّرت من ماذا إلى ماذا.',
  'log.refresh': 'تحديث',
  'log.loadFailed': 'تعذّر تحميل سجل الإجراءات.',
  'log.empty': 'لا توجد إجراءات مطابقة.',
  'log.emptyBody': 'وسّع عوامل التصفية، أو تأكّد من أنه لم يحدث شيء فعلاً في هذه الفترة.',
  'log.count': 'السجلات {from}–{to} من {total}',
  'log.retention': 'تُحفظ السجلات {days} يوماً ثم يحذفها عمل في الخلفية. لا يوجد في هذه الصفحة أي زر لحذف السجلات — سجل تدقيق يمكنه محو آثاره ليس سجل تدقيق.',
  'log.filterActor': 'المنفِّذ',
  'log.filterAction': 'الإجراء',
  'log.filterTargetType': 'نوع المورد',
  'log.filterFrom': 'من',
  'log.filterTo': 'إلى',
  'log.filterAll': 'الكل',
  'log.filterApply': 'تطبيق التصفية',
  'log.filterReset': 'مسح التصفية',
  'log.filterTargetHint': 'اضغط على الهدف في أي صف لعرض الإجراءات التي تخصّه فقط.',
  'log.colTime': 'الوقت',
  'log.colActor': 'المنفِّذ',
  'log.colAction': 'الإجراء',
  'log.colTarget': 'الهدف',
  'log.colChanges': 'التغييرات',
  'log.colOrigin': 'المصدر',
  'log.noChanges': '(لا تغيير في الحقول)',
  'log.changedTo': 'تغيّر إلى',
  'log.removed': '(محذوف)',
  'log.created': '(جديد)',
  'log.requestId': 'request {id}',
  'log.page': 'صفحة {page}',
  'log.targetUser': 'مستخدم',
  'log.targetPost': 'منشور',
  'log.targetComment': 'تعليق',
  'log.targetReport': 'بلاغ',
  'log.targetTag': 'وسم',
  'log.targetSystem': 'النظام',
  'log.actionUserSuspend': 'إيقاف',
  'log.actionUserReinstate': 'استعادة',
  'log.actionUserTags': 'تغيير الوسوم',
  'log.actionUserPost': 'نشر نيابة عن مستخدم',
  'log.actionUserComment': 'تعليق نيابة عن مستخدم',
  'log.actionUserContent': 'حذف محتواه',
  'log.actionPostCreate': 'إنشاء منشور',
  'log.actionPostUpdate': 'تعديل منشور',
  'log.actionPostDelete': 'حذف منشور',
  'log.actionCommentCreate': 'إنشاء تعليق',
  'log.actionCommentUpdate': 'تعديل تعليق',
  'log.actionCommentDelete': 'حذف تعليق',
  'log.actionReportCreate': 'إنشاء بلاغ',
  'log.actionReportResolve': 'قبول البلاغ',
  'log.actionReportReject': 'رفض البلاغ',
  'log.actionReportUpdate': 'تعديل البلاغ',
  'log.actionReportDelete': 'حذف البلاغ',
  'log.actionTagCreate': 'إنشاء وسم',
  'log.actionTagUpdate': 'إعادة تسمية وسم',
  'log.actionTagDelete': 'حذف وسم',
  'log.fieldStatus': 'حالة الحساب',
  'log.fieldContent': 'المحتوى',
  'log.fieldName': 'الاسم',
  'log.fieldTags': 'الوسوم',
  'log.fieldReason': 'السبب',
  'log.fieldAuthorEmail': 'الكاتب',
  'log.fieldReporterEmail': 'البلاغ عنه',
  'log.fieldTargetType': 'نوع المورد',
  'log.fieldTargetId': 'معرّف المورد',
  'log.fieldPostId': 'معرّف المنشور',
  'log.fieldCommentId': 'معرّف التعليق',
  'log.fieldPostIdShort': 'منشور',
  'log.fieldCommentIdShort': 'تعليق',
  'log.fieldTarget': 'الهدف',
  'log.fieldAssignmentsRemoved': 'عدد الروابط المحذوفة',
  'log.truncated': 'مقتطع',

  'stats.title': 'اتجاهات المحتوى',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'كم مستخدمًا ومنشورًا وتعليقًا جديدًا يصل كل يوم، وما المنشورات والوسوم والكتّاب الأكثر نشاطًا الآن.',
  'stats.refresh': 'تحديث',
  'stats.loadFailed': 'تعذّر تحميل إحصاءات المحتوى.',
  'stats.window': 'اعرض آخر',
  'stats.windowDays': '{days} يومًا',
  'stats.windowClamped': '(حتى 90 يومًا)',
  'stats.windowNote': 'يُقسَّم كل يوم حسب المنطقة الزمنية المحلية لجهاز الخادم. إن كان الموقع يعمل بتوقيت UTC والإدارة في منطقة أخرى فستبدو أرقام اليوم أقل — это المنطقة الزمنية لا انخفاض في الحركة.',
  'stats.generatedAt': 'أُنتجت الإحصاءات في {time}',
  'stats.seriesTitle': 'الجديد يوميًا',
  'stats.seriesNote': 'الخطوط الثلاثة بمقاييس مختلفة، لذلك تُعرض منفصلة لا متراكبة.',
  'stats.seriesUsers': 'مستخدمون جدد',
  'stats.seriesPosts': 'منشورات جديدة',
  'stats.seriesComments': 'تعليقات جديدة',
  'stats.seriesEmpty': 'لا بيانات في هذه الفترة.',
  'stats.totalsTitle': 'مجموع الفترة',
  'stats.totalsNote': 'هذه الكميات المضافة خلال هذه الفترة، وليست الإجماليات الحالية للموقع.',
  'stats.totalUsers': 'مستخدمون جدد',
  'stats.totalPosts': 'منشورات جديدة',
  'stats.totalComments': 'تعليقات جديدة',
  'stats.totalLikes': 'إعجابات جديدة',
  'stats.topPostsTitle': 'المنشورات الشائعة',
  'stats.topPostsNote': 'مرتبة بعدد التعليقات والإعجابات، وتشمل المنشورات المنشورة في هذه الفترة فقط.',
  'stats.topTagsTitle': 'الوسوم الشائعة',
  'stats.topTagsNote': 'مرتبة بعدد المستخدمين الذين يحملونها، بلا حد زمني — الوسم صفة لا حدث.',
  'stats.topAuthorsTitle': 'الكتّاب النشطون',
  'stats.topAuthorsNote': 'مرتبون بعدد المنشورات في هذه الفترة، مع عدد التعليقات منفصلًا.',
  'stats.colExcerpt': 'مقتطف',
  'stats.colEngagement': 'تفاعل',
  'stats.colPosts': 'منشورات',
  'stats.colComments': 'تعليقات',
  'stats.colUsers': 'مستخدمون',
  'stats.colAuthor': 'الكاتب',
  'stats.empty': 'لا بيانات في هذه الفترة.',
  'stats.emptyBody': 'وسّع الفترة، أو تأكد أن شيئًا جديدًا لم يحدث فعلًا.',
  'stats.engagement': '{comments} تعليقًا・{likes} إعجابًا',
  'stats.rank': 'المرتبة {rank}',

  'export.title': 'تصدير وإجراءات جماعية',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'صدّر بيانات الموقع بصيغة CSV للمطابقة، أو نفّذ الإجراء على عدة حسابات دفعة واحدة.',
  'export.download': 'تنزيل CSV',
  'export.downloading': 'جارٍ التحضير…',
  'export.exportTitle': 'تصدير',
  'export.exportNote': 'يقتصر كل تصدير على 50 ألف صف؛ وما بعد ذلك يُضمَّم الأحدث فقط. أعمدة عمليات التصدير الثلاث تطابق أعمدة صفحات الإدارة، لذا المطابقة مباشرة.',
  'export.exportUsers': 'قائمة المستخدمين',
  'export.exportUsersNote': 'البريد الإلكتروني والحالة ووقت الإنشاء والتعديل وعدد المنشورات والتعليقات.',
  'export.exportPosts': 'قائمة المنشورات',
  'export.exportPostsNote': 'المعرّف والمؤلف وأول 200 حرف من المحتوى ووقت الإنشاء وعدد التعليقات والإعجابات.',
  'export.exportReports': 'قائمة البلاغات',
  'export.exportReportsNote': 'المعرّف والهدف المبلَّغ عنه والبلِّغ والسبب والحالة ومسار المراجعة.',
  'export.safety': 'يبدأ الملف بعلامة ترتيب بايت لـ UTF-8، لذلك يفتحه Excel دون تشوّه.',
  'export.safetyPrefix': 'القيم التي تبدأ بـ = + - @ أو بمسافة غير مرئية تسبقها علامة اقتباس مفردة — وهذا ما يجعل الجدول يعاملها كنص لا ينفّذها كصيغة. هذه البادئة مقصودة، فلا تطلبوا إزالتها.',
  'export.batchTitle': 'إجراءات جماعية',
  'export.batchNote': 'تتفعّل الأزرار بعد اختيار حسابات في صفحة المستخدمين. الإجراء الجماعي يُطبَّق بالكامل أو لا يُطبَّق إطلاقًا؛ لا توجد نتيجة جزئية.',
  'export.batchSuspend': 'تعطيل المحدد',
  'export.batchReinstate': 'إعادة تفعيل المحدد',
  'export.batchTags': 'تطبيق وسوم',
  'export.batchTagsNote': 'دلالة الاستبدال: القائمة المُرسلة تصبح النتيجة. إرسال قائمة فارغة يعني إزالة كل الوسوم.',
  'export.batchConfirm': 'تطبيق «{action}» على {count} حسابًا؟',
  'export.batchConfirmTags': 'استبدال وسوم {count} حسابًا بـ {tags}؟',
  'export.batchTagsPicker': 'اختيار الوسوم',
  'export.batchTagsNone': 'بلا وسوم (إزالة الكل)',
  'export.batchRunning': 'جارٍ التنفيذ…',
  'export.batchDone': 'تم تحديث {updated} حسابًا',
  'export.batchDoneUnchanged': 'منها {unchanged} كان في الحالة المطلوبة ولم يتغير',
  'export.batchSkipped': 'تم تخطي {count}',
  'export.batchMax': '200 حساب كحد أقصى في الدفعة',
  'export.gotoUsers': 'الذهاب إلى المستخدمين',
  'export.noSelection': 'اختر حسابات في صفحة المستخدمين أولًا.',
  'export.selected': 'تم تحديد {count} حسابًا',
  'export.clearSelection': 'مسح التحديد',
  'export.selectionHint': 'يبقى التحديد في هذه الصفحة فقط ويزول عند إغلاقها.',

  'session.title': 'الدخول والجلسات',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'شاهد أي عمليات الدخول ما زالت صالحة، وأجبر حسابًا على الخروج من كل الأجهزة.',
  'session.refresh': 'تحديث',
  'session.loadFailed': 'تعذّر تحميل قائمة الجلسات.',
  'session.privacyTitle': 'لماذا لا تظهر الرمز كاملًا',
  'session.privacyNote': 'الرمز نفسه هو بيانات الدخول. تُعرض أول ثمانية أحرف فقط ليتمكن المسؤول من تمييز أن صفّين هما الجلسة نفسها — وهي لا تكفي لأحد أن يدخل نيابة عن المستخدم، ولا حتى لمن يحصل على لقطة شاشة لهذه الصفحة. هذا حدّ مقصود، لا ميزة غير مكتملة.',
  'session.expireNote': 'تنتهي الجلسة بعد {hours} ساعة من عدم النشاط؛ وأي طلب يمدّدها.',
  'session.filterEmail': 'تصفية حسب الحساب',
  'session.filterPlaceholder': 'عنوان البريد الإلكتروني كاملًا',
  'session.search': 'بحث',
  'session.clearFilter': 'مسح',
  'session.summary': '{total} جلسة في الموقع كله، فُحص {scanned} مفتاحًا',
  'session.truncated': 'بلغ الفحص حدّ {scanned} مفتاحًا وتوقف مبكرًا، لذا هذه القائمة غير كاملة.',
  'session.empty': 'لا توجد جلسات حاليًا.',
  'session.emptyBody': 'لا أحد مسجّل الدخول، أو انتهت صلاحية كل الجلسات.',
  'session.colUser': 'الحساب',
  'session.colToken': 'الجلسة',
  'session.colCreated': 'أُنشئت',
  'session.colExpires': 'تنتهي',
  'session.colRemaining': 'المتبقي',
  'session.colActions': 'الإجراء',
  'session.unknown': 'غير معروف',
  'session.adminBadge': 'الإدارة',
  'session.revoke': 'إجبار الخروج',
  'session.revokeTitle': 'إجبار خروج {email}',
  'session.revokeMessage': 'ستُبطَل جلسات الحساب البالغة {count} فورًا ويُمسح تسجيل الدخول من كل الأجهزة. على المستخدم الدخول من جديد. هل تريد المتابعة؟',
  'session.revokeRunning': 'جارٍ الإبطال…',
  'session.revokeDone': 'أُبطلت {count} جلسة',
  'session.revokeNone': 'لا توجد جلسات نشطة لهذا الحساب',
  'session.revokeFailed': 'تعذّر تأكيد تسجيل الخروج.',
  'session.revokeUnavailable': 'هذا لا يعني أن الإبطال فشل: بلغ الفحص حدّ المفاتيح، لذا قد تكون بعض الجلسات لم يتم الوصول إليها. أعد المحاولة بعد قليل.',
  'session.titleColumnNote': 'بادئة تعريفية فقط، لا تصلح لتسجيل الدخول',

  'block.title': 'قائمة حظر العناوين',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'أضِف إلى القائمة العناوين التي تم التأكد من إساءة استخدامها. القائمة مخزّنة في Redis، فلا يمسحها neither إعادة تشغيل nor نشر.',
  'block.refresh': 'تحديث',
  'block.loadFailed': 'تعذّر تحميل قائمة الحظر.',
  'block.unavailable': 'هذا الموقع غير متصل بـ Redis، لذا الحظر غير مفعّل.',
  'block.unavailableNote': 'تحتاج القائمة إلى Redis نفسه المستخدَم للجلسات ووسائط الرموز. بمجرد الاتصال ستعمل هذه الصفحة، وقبل ذلك الحماية الوحيدة هي تحديد المعدل (داخل العملية، يضيع عند إعادة التشغيل).',
  'block.add': 'حظر',
  'block.addTitle': 'حظر عنوان IP',
  'block.addMessage': 'يُرفض العنوان المحظور على كل طلب كتابة (منشورات، تعليقات، إعجابات، بلاغات، رفع صور، وتحويلات تسجيل الدخول) حتى انتهاء مدته. أما القراءة فلا تتأثر.',
  'block.ipLabel': 'عنوان IP',
  'block.ipPlaceholder': '203.0.113.9 أو 2001:db8::1',
  'block.durationLabel': 'المدة',
  'block.reasonLabel': 'السبب',
  'block.reasonPlaceholder': 'لماذا يُحظر هذا العنوان (يُسجَّل في سجل التدقيق)',
  'block.reasonHint': 'يذهب السبب إلى سجل التدقيق فقط. لا يُعرض أبدًا للمperson المحظور ولا يظهر في رسالة خطأ عامة.',
  'block.blocking': 'جارٍ الحظر…',
  'block.done': 'حُظر {ip}',
  'block.removed': 'أُلغي حظر {ip}',
  'block.removedNone': '{ip} لم يكن محظورًا أصلًا',
  'block.failed': 'فشلت عملية الحظر.',
  'block.unavailableService': 'قائمة الحظر غير متاحة (لا Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'ينتهي',
  'block.colRemaining': 'المتبقي',
  'block.colActions': 'الإجراء',
  'block.unblock': 'إلغاء الحظر',
  'block.unblockTitle': 'إلغاء حظر {ip}',
  'block.unblockMessage': 'يستعيد هذا العنوان وصوله للكتابة فورًا. هل تريد المتابعة؟',
  'block.empty': 'قائمة الحظر فارغة.',
  'block.emptyBody': 'لا يوجد أي عنوان محظور. هذه هي الحالة الطبيعية: الحظر قرار من الإدارة دائمًا، والنظام لا يحظر أحدًا تلقائيًا أبدًا.',
  'block.notAutoNote': 'هذه القائمة لا تمتلئ من تلقاء نفسها. العنوان الذي يتجاوز حد المعدل يتلقى 429 فقط، ولا يُضاف هنا تلقائيًا، لأن المخرج نفسه قد يكون مكتبًا كاملًا أو شبكة NAT كاملة، والحظر التلقائي سيلحق بهم أيضًا.',
  'block.scopeNoteLabel': 'النطاق',
  'block.notAutoNoteLabel': 'ليس تلقائيًا أبدًا',
  'block.maxNoteLabel': 'أقصى مدة',
  'block.scopeNote': 'الحظر يوقف طلبات الكتابة فقط. قراءة المنشورات والتعليقات والأصول الثابتة تبقى ممكنة، ولا يزال المحظور قادرًا على تسجيل الدخول ورؤية المحتوى: ذلك مقصود، لأن نقاط النهاية للقراءة غير مقيّدة عن قصد (وإلا لعجز الزائر المجهول عن استعمال الموقع)، والحظر يغطي المجموعات نفسها.',
  'block.maxNote': 'مدّة الحظر الواحد 365 يومًا كحد أقصى. ما يزيد على ذلك يُقلَّص إلى سنة: تاريخ الانتهاء مخزّن رقمًا، و«للأبد» سيصبح حظرًا لا أحد يذكره ولا يرتفع تلقائيًا.',
  'block.count': '{count} عنوان محظور',
  'block.ipInvalid': 'عنوان IP غير صالح. أدخل عنوان IPv4 أو IPv6؛ نطاقات CIDR غير مدعومة.',
  'announce.label': 'إعلان الموقع',
  'announce.publicNote': 'إعلان',
  'announce.closeAria': 'إغلاق هذا الإعلان',
  'announce.publishedOn': 'نُشر في {date}',
  'announce.expiresOn': 'ينتهي في {date}',
  'announce.neverExpires': 'بلا انتهاء',
  'announce.pinnedBadge': 'مثبَّت',
  'announce.title': 'الإعلانات',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'عرض إعلان واحد أعلى كل صفحة في الموقع. واحد فقط يكون ساريًا في الوقت نفسه: نشر إعلان جديد يعطّل السابق.',
  'announce.refresh': 'تحديث',
  'announce.loadFailed': 'تعذّر تحميل قائمة الإعلانات.',
  'announce.new': 'نشر إعلان جديد',
  'announce.edit': 'تعديل',
  'announce.deactivate': 'تعطيل',
  'announce.reactivate': 'إعادة التفعيل',
  'announce.deleteNote': 'لا تُحذف الإعلانات أبدًا، بل تُعطَّل فقط: الاحتفاظ بالتاريخ هو ما يجيب عن سؤال متى نُشر وبمن.',
  'announce.bodyLabel': 'نص الإعلان',
  'announce.bodyPlaceholder': 'مثلًا: ستكون هناك صيانة للنظام يوم الخميس من 02:00 إلى 04:00.',
  'announce.bodyHint': '300 حرف كحد أقصى. نص عادي، وتبقى الأسطر الجديدة.',
  'announce.activeLabel': 'إظهار فورًا',
  'announce.expiryLabel': 'مدة الصلاحية',
  'announce.expiryNever': 'لا ينتهي تلقائيًا أبدًا',
  'announce.expiryHours': 'بعد {hours} ساعات',
  'announce.expiryDays': 'بعد {days} أيام',
  'announce.saving': 'جارٍ الحفظ…',
  'announce.published': 'نُشر الإعلان',
  'announce.updated': 'حُدِّث الإعلان',
  'announce.deactivated': 'عُطِّل الإعلان',
  'announce.reactivated': 'أُعيد تفعيل الإعلان',
  'announce.saveFailed': 'فشلت عملية الإعلان.',
  'announce.empty': 'لا توجد إعلانات بعد.',
  'announce.emptyBody': 'فور نشر إعلان سيظهر أعلى صفحة كل زائر.',
  'announce.colBody': 'النص',
  'announce.colState': 'الحالة',
  'announce.colAuthor': 'ناشره',
  'announce.colCreated': 'وقت النشر',
  'announce.colActions': 'الإجراء',
  'announce.stateActive': 'معروض',
  'announce.stateInactive': 'معطَّل',
  'announce.stateExpired': 'منتهي',
  'announce.confirmDeactivate': 'سيؤدي النشر إلى تعطيل الحالي، وسيرى الجميع النص الجديد فورًا. هل تريد المتابعة؟',
  'announce.confirmEdit': 'تعديل نص هذا الإعلان أو مدة صلاحيته؟',
  'announce.count': '{count} إجمالًا',
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
  'posts.pin': 'تثبيت',
  'posts.unpin': 'إلغاء التثبيت',
  'posts.pinTitle': 'تثبيت هذه المنشورة',
  'posts.unpinTitle': 'إلغاء تثبيت هذه المنشورة',
  'posts.pinMessage': 'بعد التثبيت تبقى هذه المنشورة في أعلى صفّ الجميع، ولا يمكن لأي منشورة جديدة أن تدفعها إلى الأسفل.',
  'posts.unpinMessage': 'إلغاء التثبيت يعيد المنشورة إلى موضعها بترتيب الوقت.',
  'posts.pinDone': 'تم التثبيت',
  'posts.unpinDone': 'أُلغي التثبيت',
  'posts.pinFailed': 'فشلت عملية التثبيت.',
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
  'title.adminMonitor': 'مراقبة النظام｜{brand} الإدارة',
  'title.adminLog': 'سجل الإجراءات｜{brand} الإدارة',
  'title.adminStats': 'اتجاهات المحتوى｜{brand} الإدارة',
  'title.adminExport': 'تصدير وإجراءات جماعية｜{brand} الإدارة',
  'title.adminSessions': 'الدخول والجلسات｜{brand} الإدارة',
  'title.adminBlocks': 'قائمة حظر العناوين｜{brand} الإدارة',
  'title.adminAnnouncements': 'الإعلانات｜{brand} الإدارة',
};
