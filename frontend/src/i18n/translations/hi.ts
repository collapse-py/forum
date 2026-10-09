/*
 * hi catalog (src/i18n/translations/hi.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const hi: Record<MessageKey, string> = {
  'common.cancel':
    'रद्द करें',
  'common.save':
    'संभालें',
  'common.submitting':
    'भेजा जा रहा है...',
  'common.delete':
    'हटाएँ',
  'common.edit':
    'संपादित करें',
  'common.search':
    'खोजें',
  'common.loading':
    'लोड हो रहा है…',
  'common.loadFailed':
    'लोड नहीं हो पाया',
  'common.refresh':
    'ताज़ा करें',
  'common.nextStep':
    'अगला चरण',
  'common.prevPage':
    'पिछला पृष्ठ',
  'common.nextPage':
    'अगला पृष्ठ',
  'common.create':
    'बनाएँ',
  'common.publish':
    'प्रकाशित करें',
  'common.placeholder':
    '—',
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 घंटा',
  'common.oneDay': '1 दिन',
  'common.sevenDays': '7 दिन',
  'common.thirtyDays': '30 दिन',
  'common.oneYear': '1 वर्ष',
  'common.backToHome':
    'मुख्य पृष्ठ पर जाएँ',
  'common.backToForumHome':
    'फोरम मुख्य पृष्ठ पर जाएँ',
  'common.backOnePage':
    'पिछले पृष्ठ पर जाएँ',
  'error.request':
    'कृपया बाद में फिर प्रयास करें।',
  'error.requestStatus':
    'अनुरोध विफल (HTTP {status})',
  'error.loginRequired':
    'कृपया पहले प्रवेश करें, फिर जारी रखें।',
  'error.adminSessionExpired':
    'प्रवेश अवस्था समाप्त हो चुकी है और अब प्रवेश पृष्ठ पर लौटाया जाएगा।',
  'error.fallbackLoad':
    'लोड नहीं हो पाया',
  'error.fallbackSearch':
    'खोज नहीं हो पाई',
  'error.fallbackLike':
    'पसंद करने में विफलता',
  'error.fallbackComments':
    'टिप्पणियाँ लोड नहीं हो पाईं',
  'error.fallbackCommentPost':
    'टिप्पणी दर्ज करने में विफलता',
  'error.fallbackCommentEdit':
    'टिप्पणी सहेजी नहीं जा सकी',
  'error.fallbackCommentDelete':
    'टिप्पणी हटाई नहीं जा सकी',
  'error.fallbackReport':
    'रिपोर्ट करने में विफलता',
  'error.fallbackProfile':
    'व्यक्तिगत जानकारी पढ़ने में विफलता',
  'error.fallbackProfileSave':
    'संभालने में विफलता',
  'error.fallbackAvatarUpload':
    'अवतार अपलोड करने में विफलता',
  'error.fallbackPublish':
    'प्रकाशित करने के लिए जवाब का स्वरूप गलत है',
  'error.fallbackUpload':
    'चित्र अपलोड का जवाब गलत स्वरूप का है',
  'error.fallbackNotFound':
    'यह उपयोगकर्ता नहीं मिला',
  'error.fallbackFollow':
    'अनफ़ॉलो करना विफल',
  'auth.checking':
    'प्रवेश अवस्था की जांच चल रही है...',
  'auth.statusUnknown':
    'प्रवेश अवस्था की पुष्टि नहीं हो सकी',
  'auth.feedLoggedIn':
    'प्रवेश हुआ है, पोस्ट कर सकते हैं',
  'auth.feedLoggedOut':
    'पोस्ट करने के लिए प्रवेश करें',
  'auth.profileLoggedIn':
    'प्रवेश हुआ है',
  'auth.profileLoggedOut':
    'प्रोफ़ाइल सेट करने के लिए प्रवेश करें',
  'auth.googleLogin':
    'Google से प्रवेश करें',
  'auth.loginWithGoogle':
    'Google अकाउंट से प्रवेश करें',
  'auth.loginWithGoogleAdmin':
    'Google प्रबंधक अकाउंट से प्रवेश करें',
  'auth.logout':
    'प्रवेश छोड़ें',
  'install.button':
    'App स्थापित करें',
  'install.hint':
    'वर्तमान ब्राउज़र में अपने आप स्थापित करने का संकेत नहीं मिलता; कृपया ब्राउज़र मेनू खोलें और “एप्लिकेशन स्थापित करें” या “मुख्य पृष्ठ पर जोड़ें” चुनें।',
  'bottomNav.label':
    'मुख्य नाविगेशन',
  'bottomNav.home':
    'मुख्य पृष्ठ',
  'bottomNav.new':
    'नया',
  'bottomNav.profile':
    'व्यक्तिगत',
  'i18n.ariaLabel':
    'भाषा चुनें',
  'i18n.current':
    'भाषा: {name}',
  'feed.searchPlaceholder':
    'पोस्ट की सामग्री खोजें',
  'feed.searchAriaLabel':
    'पोस्ट खोजें',
  'feed.searchResultsLabel':
    'खोज परिणाम',
  'feed.postsLabel':
    'फोरम पोस्ट',
  'feed.searchFailed':
    'खोज विफल हो गई, कृपया बाद में फिर प्रयास करें।',
  'feed.searching':
    'खोज चल रही है...',
  'feed.searchMore':
    'अधिक खोज परिणाम लोड करें...',
  'feed.searchMoreFailed':
    'अधिक लोड नहीं हो पाया',
  'feed.searchFound':
    '{total} मिले',
  'feed.searchDegraded':
    '{base} (खोज सेवा सक्रिय नहीं है, इस समय डेटाबेस की कीवर्ड से मिलान किया जा रहा है)',
  'feed.searchTotal':
    'कुल {total} परिणाम',
  'feed.searchNoResults':
    '“{query}” शब्द वाला कोई पोस्ट नहीं मिला।',
  'feed.loadingPosts':
    'पोस्ट लोड हो रहे हैं...',
  'feed.loadMorePosts':
    'अधिक पोस्ट लोड करें...',
  'feed.postsFailed':
    'पोस्ट लोड नहीं हो पाये, कृपया बाद में फिर प्रयास करें।',
  'feed.postsFailedShort':
    'लोड नहीं हो पाया, कृपया बाद में फिर प्रयास करें',
  'feed.scrollMore':
    'नीचे स्वाइप करें ताकि अधिक लोड हो सके',
  'feed.endOfFeed':
    'अब खाते का अंत हो गया',
  'feed.noPosts':
    'अभी कोई पोस्ट नहीं है; पहला विचार दर्ज करें।',
  'feed.likeFailed':
    'पसंद या पसंद नहीं करने में विफलता हुई, कृपया बाद में फिर प्रयास करें।',
  'post.authorAnonymous':
    'अनाम',
  'post.report':
    'पोस्ट रिपोर्ट करें',
  'post.imageAlt':
    'पोस्ट की छवि',
  'post.unlike':
    'पसंद नहीं',
  'post.like':
    'पसंद',
  'post.reply':
    'प्रतिक्रिया',
  'post.permalink':
    'स्थायी लिंक',
  'post.editedBadge':
    'संपादित',
  'post.editContentLabel':
    'पोस्ट सामग्री',
  'post.editMax':
    'अधिकतम 10000 अक्षर',
  'comment.loading':
    'टिप्पणी लोड हो रही है...',
  'comment.none':
    'अभी कोई टिप्पणी नहीं',
  'comment.loadFailed':
    'टिप्पणी लोड नहीं हो पाई, कृपया बाद में फिर प्रयास करें',
  'comment.placeholder':
    'टिप्पणी लिखें...',
  'comment.max':
    'अधिकतम 2000 अक्षर',
  'comment.submit':
    'टिप्पणी दर्ज करें',
  'comment.failed':
    'टिप्पणी दर्ज करने में विफलता हुई, कृपया बाद में फिर प्रयास करें।',
  'comment.report':
    'रिपोर्ट',
  'comment.more':
    'अधिक टिप्पणी लोड करें...',
  'comment.editedBadge':
    'संपादित',
  'comment.editContentLabel':
    'टिप्पणी सामग्री',
  'comment.editFailed':
    'टिप्पणी सहेजी नहीं जा सकी, कृपया बाद में फिर प्रयास करें।',
  'comment.deleteFailed':
    'टिप्पणी हटाई नहीं जा सकी, कृपया बाद में फिर प्रयास करें।',
  'report.reasonPlaceholder':
    'रिपोर्ट का कारण लिखें (अधिकतम 500 अक्षर)',
  'report.note':
    'रिपोर्ट स्टेशन प्रबंधकों को भेजी जाएगी',
  'report.formLabel':
    'रिपोर्ट इनपुट फ़ील्ड',
  'report.submit':
    'रिपोर्ट भेजें',
  'report.failed':
    'रिपोर्ट भेजने में विफलता हुई, कृपया बाद में फिर प्रयास करें।',
  'report.sent':
    'रिपोर्ट भेज दी गई, आपकी जानकारी के लिए धन्यवाद।',
  'newPost.avatarYou':
    'आप',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'नया पोस्ट',
  'newPost.loginFirst':
    'कृपया पहले Google से प्रवेश करें, फिर पोस्ट करें',
  'newPost.contentPlaceholder':
    'अपना विचार साझा करें...',
  'newPost.addImage':
    'नया चित्र जोड़ें',
  'newPost.emailPrivate':
    'आपका email प्रकाशित नहीं होगा',
  'newPost.submit':
    'पोस्ट प्रकाशित करें',
  'newPost.publishing':
    'पोस्ट प्रकाशित हो रहा है...',
  'newPost.uploading':
    'चित्र अपलोड हो रहा है...',
  'newPost.failed':
    'पोस्ट करने में सफलता मिली नहीं। कृपया बाद में फिर प्रयास करें।',
  'newPost.imagePreviewAlt':
    'अपलोड की जाने वाली छवि का पूर्वावलोकन',
  'newPost.draftNote':
    'आपका मसौदा इस डिवाइस पर अपने आप सहेजा जाता है (केवल पाठ; चुनी गई छवि नहीं रहती)।',
  'postPage.loading':
    'पोस्ट लोड हो रही है...',
  'postPage.missing':
    'यह पोस्ट शायद हटा दी गई है, या लिंक गलत है।',
  'postPage.failed':
    'पोस्ट लोड नहीं हो पाई, कृपया बाद में फिर प्रयास करें।',
  'postPage.label':
    'पोस्ट',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'व्यक्तिगत प्रोफ़ाइल',
  'profile.edit':
    'संशोधित करें',
  'profile.loginPrompt':
    'लॉग इन करने के बाद, आप अपना फोरम उपनाम और परिचय सेट कर सकते हैं।',
  'profile.nicknameLabel':
    'फोरम उपनाम',
  'profile.notSet':
    'अभी सेट नहीं किया गया है',
  'profile.notSetBio':
    'अभी कोई परिचय सेट नहीं किया गया है।',
  'profile.nicknameInput':
    'उपनाम',
  'profile.nicknamePlaceholder':
    'उपनाम दर्ज करें',
  'profile.nicknameHint':
    'उपनाम आपकी पोस्ट पर प्रदरशित होगा, अधिकतम 30 अक्षर।',
  'profile.avatarLabel':
    'अवतार',
  'profile.avatarHint':
    'अवतार आपकी पोस्ट और टिप्पणियों पर प्रदर्शित होगा, JPG, PNG, GIF या WebP अपलोड किया जा सकता है।',
  'profile.avatarChoose':
    'चित्र चुनें',
  'profile.avatarRemove':
    'अवतार हटाएँ',
  'profile.avatarUploading':
    'अपलोड हो रहा है...',
  'profile.avatarPreviewAlt':
    'नए अवतार का पूर्वावलोकन',
  'profile.bioLabel':
    'परिचय',
  'profile.bioPlaceholder':
    'थोड़ा अपने बारे में बताएँ (वैकल्पिक)',
  'profile.bioHint':
    'अधिकतम 500 अक्षर।',
  'profile.updated':
    'प्रोफ़ाइल अपडेट हो गई है।',
  'profile.saving':
    'संशोधन संरक्षित हो रहा है...',
  'profile.saveFailed':
    'संशोधन संरक्षित नहीं हो सका। कृपया बाद में फिर प्रयास करें।',
  'profile.loadFailed':
    'डेटा पढ़ने में सफलता मिली नहीं। कृपया बाद में फिर प्रयास करें।',
  'profile.followingEntry':
    'मेरी फ़ॉलोइंग',
  'profile.postsLabel':
    'मेरी पोस्ट',
  'profile.emptyPosts':
    'आपने अभी तक कोई पोस्ट नहीं की है।',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'सार्वजनिक प्रोफ़ाइल',
  'publicProfile.loading':
    'लोड हो रहा है...',
  'publicProfile.invalidLinkName':
    'प्रोफ़ाइल का अमान्य सार्वजनिक पृष्ठ संलग्नक',
  'publicProfile.invalidLinkBio':
    'कृपया फोरम पोस्ट के लेखक के उपनाम से प्रोफ़ाइल खोलें।',
  'publicProfile.notFound':
    'इस उपयोगकर्ता को नहीं मिला',
  'publicProfile.anonymous':
    'अनोनिमस उपयोगकर्ता',
  'publicProfile.noBio':
    'इस उपयोगकर्ता ने सार्वजनिक प्रोफ़ाइल सेट नहीं की है।',
  'publicProfile.loadFailed':
    'सार्वजनिक प्रोफ़ाइल लोड नहीं हो सकी।',
  'publicProfile.postsLabel':
    'पोस्ट',
  'publicProfile.emptyPosts':
    'इस उपयोगकर्ता ने अभी तक कोई पोस्ट नहीं की है।',

  'follow.label':
    'इस उपयोगकर्ता को फ़ॉलो करें',
  'follow.action':
    'फ़ॉलो करें',
  'follow.actionDone':
    'फ़ॉलो कर रहे हैं',
  'follow.unfollow':
    'अनफ़ॉलो करें',
  'follow.done':
    'अब आप इस उपयोगकर्ता को फ़ॉलो कर रहे हैं।',
  'follow.failed':
    'फ़ॉलो नहीं हो सका। कृपया बाद में फिर प्रयास करें।',

  'following.peopleLabel':
    'आप जिनका फ़ॉलो करते हैं',
  'following.postsLabel':
    'फ़ॉलो किए गए उपयोगकर्ताओं की पोस्ट',
  'following.peopleLoading':
    'फ़ॉलोइंग सूची लोड हो रही है...',
  'following.emptyPeople':
    'आप अभी तक किसी को फ़ॉलो नहीं करते। किसी पोस्ट पर «फ़ॉलो करें» दबाएँ, या किसी उपयोगकर्ता की सार्वजनिक प्रोफ़ाइल से उन्हें फ़ॉलो करें।',
  'following.emptyPosts':
    'जिन्हें आप फ़ॉलो करते हैं, उन्होंने अभी तक कोई पोस्ट नहीं की है।',
  'following.peopleFailed':
    'फ़ॉलोइंग सूची लोड नहीं हो सकी।',
  'following.postsFailed':
    'फ़ॉलो किए गए उपयोगकर्ताओं की पोस्ट लोड नहीं हो सकीं।',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'स्वागत है, फिर मिला',
  'login.body':
    '{site} विचार लिखने वाले सभी के लिए एक जगह है। रजिस्ट्रेशन फॉर्म की आवश्यकता नहीं है — Google अकाउंट से ही पोस्ट करना शुरू करें।',
  'login.browseFirst':
    'पहले होमपेज देखें',
  'admin.skipToMain':
    'मुख्य सामग्री पर जाएँ',
  'admin.railLabel':
    'प्रबंधन मेनू',
  'admin.railBrandAria':
    '{site} होमपेज',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'प्रमुख कार्य',
  'admin.railGovernance':
    'शासन',
  'admin.railMode':
    'प्रबंधक मोड',
  'admin.railExit':
    'फोरम पर लौटें',
  'admin.topbarMenu':
    'प्रबंधन मेनू खोलें/बंद करें',
  'admin.statusOnline':
    'ऑनलाइन है',
  'admin.topbarForum':
    'फोरम',
  'admin.logoutFailed':
    'लॉग आउट नहीं हो सका। कृपया बाद में फिर प्रयास करें।',
  'admin.navUsers':
    'उपयोगकर्ता प्रबंधन',
  'admin.navPosts':
    'फोरम पोस्ट',
  'admin.navReports':
    'रिपोर्ट प्रबंधन',
  'admin.navMonitor': 'सिस्टम मॉनिटरिंग',
  'admin.navLog': 'कार्य लॉग',
  'admin.navStats': 'सामग्री की प्रवृत्ति',
  'admin.navExport': 'निर्यात और बड़े पैमाने की कार्रवाई',
  'admin.navSessions': 'साइन-इन और सत्र',
  'admin.navBlocks': 'IP अवरुद्ध सूची',
  'admin.navAnnouncements': 'घोषणाएँ',
  'admin.listLoadFailed':
    'लोड नहीं हो सका',
  'admin.dlgClose':
    'खिड़की बंद करें',
  'admin.dlgConfirm':
    'पुष्टि करें',
  'admin.dlgSave':
    'संरक्षित करें',
  'admin.dlgApplyTags':
    'टैग लागू करें',
  'admin.dlgNoTags':
    'अभी कोई टैग लागू करने के लिए उपलब्ध नहीं है; कृपया नीचे “टैग प्रबंधन” में नया बनाएँ।',

  'monitor.title': 'सिस्टम मॉनिटरिंग',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'सेवा की सेहत, अनुरोधों की मात्रा, विलंबता वितरण और रेट लिमिटर की गिनती लाइव देखें।',
  'monitor.refresh': 'रिफ्रेश करें',
  'monitor.refreshing': 'लोड हो रहा है…',
  'monitor.autoRefresh': 'स्वतः रिफ्रेश',
  'monitor.autoRefreshOn': 'हर {seconds} सेकंड में स्वतः रिफ्रेश',
  'monitor.autoRefreshOff': 'स्वतः रिफ्रेश रुका हुआ है',
  'monitor.nextUpdate': '{seconds} सेकंड में अपडेट',
  'monitor.loadFailed': 'मॉनिटरिंग डेटा लोड नहीं हो सका।',
  'monitor.loadFailedHint': 'जाँचें कि आप व्यवस्थापक के रूप में साइन इन हैं और सर्वर अभी भी चल रहा है।',
  'monitor.pausedHint': 'स्वतः रिफ्रेश रुका हुआ है; स्क्रीन पर पिछली सफल रीड का परिणाम है।',
  'monitor.visibilityPaused': 'टैब पृष्ठभूमि में है, इसलिए स्वतः रिफ्रेश रुका हुआ है।',
  'monitor.lastUpdated': '{time} पर अपडेट किया गया',
  'monitor.probeTook': 'निर्भरता जाँच: {ms} मि.से.',
  'monitor.unreachable': 'सर्वर प्रतिक्रिया नहीं दे रहा है। स्क्रीन पिछली सफल रीड पर रुकी हुई है।',
  'monitor.depsTitle': 'सेवाओं की सेहत',
  'monitor.depsNote': 'हर रीड हर निर्भरता की एक बार वास्तविक जाँच करता है; प्रति निर्भरता समय-सीमा 2 सेकंड है और तीनों एक साथ चलती हैं।',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'खोज इंजन',
  'monitor.stateOk': 'स्वस्थ',
  'monitor.stateDown': 'पहुँच योग्य नहीं',
  'monitor.stateDisabled': 'सक्षम नहीं',
  'monitor.depSearchFallback': 'ES_URL सेट नहीं है, इसलिए खोज MySQL की कीवर्ड मिलान पर लौटती है।',
  'monitor.depDisabled': 'कोई Redis क्लाइंट नहीं दिया गया, इसलिए मीडिया सुविधाएँ बंद हैं।',
  'monitor.depLatency': '{ms} मि.से. में प्रतिक्रिया',
  'monitor.depKeys': '{count} कुंजियाँ',
  'monitor.depMemory': 'मेमोरी {size}',
  'monitor.depPoolUsage': 'कनेक्शन {inUse}/{open} (अधिकतम {max})',
  'monitor.depPoolWait': '{count} बार प्रतीक्षा, कुल {ms} मि.से.',
  'monitor.depRedisPool': 'हिट {hits} / मिस {misses}',
  'monitor.depEngineMysql': 'MySQL कीवर्ड मिलान',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'अनुरोध अवलोकन',
  'monitor.statUptime': 'चलने का समय',
  'monitor.statRequests': 'कुल अनुरोध',
  'monitor.statErrorRate': 'त्रुटि दर',
  'monitor.statP95': 'P95 विलंबता',
  'monitor.statInFlight': 'प्रगति में',
  'monitor.statGoroutines': 'Goroutines',
  'monitor.statHeap': 'हीप मेमोरी',
  'monitor.statDbPool': 'डेटाबेस कनेक्शन',
  'monitor.statRateLimited': 'रेट लिमिट से रोके गए',
  'monitor.statCountWithPeak': 'शिखर {peak}',
  'monitor.statCountWithInUse': '{inUse} उपयोग में, {idle} खाली',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} लॉजिकल कोर · {gc} GC चक्र',
  'monitor.noData': 'अभी तक कोई अनुरोध नहीं।',
  'monitor.noDataBody': 'सेवा शुरू होने के बाद के अनुरोध यहाँ दिखेंगे; यह खंड अभी खाली है।',
  'monitor.timelineTitle': 'पिछले {minutes} मिनट का ट्रैफ़िक',
  'monitor.timelineNote': 'कुल आँकड़े इस प्रक्रिया के शुरू होने से जमा होते हैं और पुनःआरंभ पर शून्य हो जाते हैं; विलंबता का हर प्रतिशतताइल हिस्टोग्राम की एक बाल्टी की ऊपरी सीमा है, इसलिए वह केवल असंतत मान लेता है। जिस मिनट में कोई पट्टी नहीं है, उसका अर्थ है कि तब ट्रैफ़िक नहीं था।',
  'monitor.timelineLive': 'यह प्रक्रिया',
  'monitor.timelineHistory': 'पुनःआरंभ से पहले',
  'monitor.timelineLegendVolume': 'अनुरोध',
  'monitor.timelineLegendError': '5xx त्रुटियाँ',
  'monitor.timelinePeak': 'शिखर {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'डेटाबेस में कोई ऐतिहासिक सारांश उपलब्ध नहीं है; यदि आरंभ पर उन्हें पढ़ा नहीं जा सका (तालिका अनुपस्थित या अनुमति नहीं), तो केवल वर्तमान प्रक्रिया दिखाई जाती है।',
  'monitor.routesTitle': 'रूट के अनुसार',
  'monitor.routesNote': 'पथ सामान्यीकृत होते हैं (संख्यात्मक पहचान और ईमेल पते :id बन जाते हैं), इसलिए एक ही रूट के अलग-अलग पहचान एक साथ गिने जाते हैं।',
  'monitor.colRoute': 'रूट',
  'monitor.colCount': 'अनुरोध',
  'monitor.colAvg': 'औसत',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'सबसे धीमा',
  'monitor.colErrors': 'त्रुटियाँ',
  'monitor.routeOther': 'अन्य (रूट सीमा पूरी)',
  'monitor.clientsTitle': 'स्रोत पते',
  'monitor.clientsNote': 'अनुरोधों की संख्या के अनुसार क्रम में। X-Forwarded-For या X-Real-IP से लिया गया पता किसी भरोसेमंद प्रॉक्सी से सत्यापित नहीं हुआ है — कार्रवाई से पहले जाँच लें कि आपका प्रॉक्सी इन हेडरों को ओवरराइट करता है या नहीं।',
  'monitor.noClients': 'अभी तक कोई स्रोत ट्रैक नहीं हुआ।',
  'monitor.noClientsBody': 'हर अनुरोध उसके स्रोत पते के साथ दर्ज होता है। यह तालिका अभी खाली है।',
  'monitor.clientsDropped': 'स्रोतों की संख्या सीमा {limit} तक पहुँच गई; सबसे पहले देखे गए {count} पते ट्रैकिंग से हटा दिए गए। यह पंक्ति यह मतलब नहीं कि केवल इतने ही लोग आए; इसका मतलब है कि यह सूची अधूरी है।',
  'monitor.colIp': 'पता',
  'monitor.colSource': 'स्रोत',
  'monitor.colRateLimited': 'रेट लिमिट से रोके गए',
  'monitor.colBanned': 'अवरुद्ध सूची से रोके गए',
  'monitor.colLastRoute': 'अंतिम पथ',
  'monitor.colActions': 'क्रियाएँ',
  'monitor.colBlock': 'अवरुद्ध करें',
  'monitor.blocking': 'अवरुद्ध हो रहा है…',
  'monitor.sourcePeer': 'कनेक्शन स्रोत',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.trustLegacy': 'TRUSTED_PROXY_CIDRS सेट नहीं है: सर्वर पुराना व्यवहार बनाए रखता है और X-Forwarded-For के सबसे बाएँ प्रविष्टि को प्राथमिकता देता है। जब तक यह पुष्ट न हो कि आगे कोई प्रॉक्सी है जो वास्तव में ये हेडर अधिलेखित करता है और उपयोक्ता उसे बायपाकर सीधे कनेक्ट नहीं हो सकते, तब तक रेट लिमिट और IP ब्लॉक एक ही नकली हेडर से बायपास हो सकते हैं, और ऑडिट लॉग का स्रोत पता सबूत नहीं है।',
  'monitor.trustConfigured': 'विश्वसनीय प्रॉक्सी सेट हैं: X-Forwarded-For / X-Real-IP तब ही स्वीकार किए जाते हैं जब कनेक्शन का साथी इनमें से किसी नेटवर्क में हो; अन्यथा साथी का पता उपयोग होता है। अभी लागू: {cidrs}।',
  'monitor.trustBroken': 'TRUSTED_PROXY_CIDRS घोषित है पर कोई भी प्रविष्टि CIDR श्रेणी के रूप में पढ़ी नहीं जा सकी ({declared}), इसलिए सेट न होने वाला पुराना व्यवहार अब भी लागू है।',
  'monitor.trustPartial': 'TRUSTED_PROXY_CIDRS के निम्नलिखित प्रविष्टियाँ CIDR श्रेणियों के रूप में पढ़ी नहीं जा सकतीं ({invalid}), इसलिए उन श्रेणियों से आने वाले फ़ॉरवर्ड किए गए हेडर कभी स्वीकार नहीं होते। उन श्रेणियों से आने वाले अनुरोध कनेक्शन साथी के पते के अनुसार बँटते हैं, यानी वे एक ही रेट-लिमिट बजट और एक ही ब्लॉकलिस्ट क्वेरी साझा करते हैं।',
  'monitor.blockTitle': '{ip} अवरुद्ध करें',
  'monitor.blockMessage': 'इस पते से लेखन अनुरोध (पोस्ट, टिप्पणी, लाइक, रिपोर्ट, छवि अपलोड, लॉगिन रीडायरेक्ट) {duration} तक अस्वीकार किए जाएँगे। पढ़ना प्रभावित नहीं होता। क्या अवरुद्ध करें?',
  'monitor.blockReason': 'मॉनिटर पेज से अवरुद्ध',
  'monitor.blocked': '{ip} अवरुद्ध कर दिया गया',
  'monitor.blockFailed': 'अवरुद्ध करने का कार्य विफल रहा।',
  'monitor.limitsTitle': 'रेट लिमिटर',
  'monitor.limitsNote': 'हर समूह के पास अपने एंडपॉइंट की लागत के अनुसार अपना बजट है; रोकी गई संख्या इस प्रक्रिया के शुरू होने से अब तक की सभी 429 प्रतिक्रियाँ हैं।',
  'monitor.colLimiter': 'लिमिटर',
  'monitor.colBudget': 'बजट',
  'monitor.colAllowed': 'अनुमत',
  'monitor.colBlocked': 'रोकी गईं',
  'monitor.colTracked': 'ट्रैक किए जा रहे स्रोत',
  'monitor.colBlockedRate': 'अवरुद्ध दर',
  'monitor.limitContent': 'कॉन्टेंट लेखन',
  'monitor.limitUpload': 'छवि अपलोड',
  'monitor.limitAuth': 'OAuth लॉगिन',
  'monitor.limitBudget': '{window} सेकंड में {limit}',
  'monitor.limitUnknown': '(अज्ञात)',
  'monitor.noLimits': 'कोई रेट लिमिटर उपलब्ध नहीं।',
  'monitor.noLimitsBody': 'रेट लिमिटर अभी बनाए नहीं गए हैं, इसलिए उनकी गिनती उपलब्ध नहीं है।',

  'log.title': 'कार्य लॉग',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'एडमिन कंसोल में हर बदलाव देखें: किसने, कब, किस लक्ष्य पर, और कौन से fields किसमें बदले।',
  'log.refresh': 'रिफ्रेश करें',
  'log.loadFailed': 'कार्य लॉग लोड नहीं हो सका।',
  'log.empty': 'फ़िल्टर से मेल खाता कोई कार्य नहीं।',
  'log.emptyBody': 'फ़िल्टर ढीले करें, या पुष्टि करें कि इस अवधि में सचमुच कोई कार्य नहीं हुआ।',
  'log.count': 'प्रविष्टियाँ {from}–{to} / {total} में से',
  'log.retention': 'रिकॉर्ड {days} दिन तक रखे जाते हैं और फिर पृष्ठभूमि कार्य हटा देता है। इस पेज पर रिकॉर्ड मिटाने का कोई बटन नहीं है — जो ऑडिट लॉग अपने निशान मिटा सके, वह ऑडिट लॉग नहीं है।',
  'log.filterActor': 'कर्ता',
  'log.filterAction': 'कार्य',
  'log.filterTargetType': 'संसाधन प्रकार',
  'log.filterFrom': 'से',
  'log.filterTo': 'तक',
  'log.filterAll': 'सभी',
  'log.filterApply': 'फ़िल्टर लगाएँ',
  'log.filterReset': 'फ़िल्टर हटाएँ',
  'log.filterTargetHint': 'किसी भी पंक्ति के लक्ष्य पर क्लिक करके केवल उसी से जुड़े कार्य देखें।',
  'log.colTime': 'समय',
  'log.colActor': 'कर्ता',
  'log.colAction': 'कार्य',
  'log.colTarget': 'लक्ष्य',
  'log.colChanges': 'बदलाव',
  'log.colOrigin': 'स्रोत',
  'log.noChanges': '(कोई field बदलाव नहीं)',
  'log.changedTo': 'बदलकर',
  'log.removed': '(हटा दिया गया)',
  'log.created': '(नया बनाया गया)',
  'log.requestId': 'request {id}',
  'log.page': 'पृष्ठ {page}',
  'log.targetUser': 'उपयोगकर्ता',
  'log.targetPost': 'पोस्ट',
  'log.targetComment': 'टिप्पणी',
  'log.targetReport': 'रिपोर्ट',
  'log.targetTag': 'टैग',
  'log.targetSystem': 'सिस्टम',
  'log.actionUserSuspend': 'निलंबित',
  'log.actionUserReinstate': 'बहाल किया',
  'log.actionUserTags': 'टैग बदले',
  'log.actionUserPost': 'उपयोगकर्ता की ओर से पोस्ट',
  'log.actionUserComment': 'उपयोगकर्ता की ओर से टिप्पणी',
  'log.actionUserContent': 'उनका सामग्री हटाई',
  'log.actionPostCreate': 'पोस्ट बनी',
  'log.actionPostUpdate': 'पोस्ट संपादित',
  'log.actionPostDelete': 'पोस्ट हटाई',
  'log.actionCommentCreate': 'टिप्पणी बनी',
  'log.actionCommentUpdate': 'टिप्पणी संपादित',
  'log.actionCommentDelete': 'टिप्पणी हटाई',
  'log.actionReportCreate': 'रिपोर्ट बनी',
  'log.actionReportResolve': 'रिपोर्ट स्वीकृत',
  'log.actionReportReject': 'रिपोर्ट अस्वीकृत',
  'log.actionReportUpdate': 'रिपोर्ट संपादित',
  'log.actionReportDelete': 'रिपोर्ट हटाई',
  'log.actionTagCreate': 'टैग बनाया',
  'log.actionTagUpdate': 'टैग का नाम बदला',
  'log.actionTagDelete': 'टैग हटाया',
  'log.fieldStatus': 'खाता स्थिति',
  'log.fieldContent': 'सामग्री',
  'log.fieldName': 'नाम',
  'log.fieldTags': 'टैग',
  'log.fieldReason': 'कारण',
  'log.fieldAuthorEmail': 'लेखक',
  'log.fieldReporterEmail': 'रिपोर्टकर्ता',
  'log.fieldTargetType': 'संसाधन प्रकार',
  'log.fieldTargetId': 'संसाधन आईडी',
  'log.fieldPostId': 'पोस्ट आईडी',
  'log.fieldCommentId': 'टिप्पणी आईडी',
  'log.fieldPostIdShort': 'पोस्ट',
  'log.fieldCommentIdShort': 'टिप्पणी',
  'log.fieldTarget': 'लक्ष्य',
  'log.fieldAssignmentsRemoved': 'हटाई गई बाइंडिंग',
  'log.truncated': 'छोटा किया गया',

  'stats.title': 'सामग्री की प्रवृत्ति',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'हर दिन कितने नए उपयोगकर्ता, पोस्ट और टिप्पणियाँ आती हैं, और अभी कौन से पोस्ट, टैग और लेखक सबसे सक्रिय हैं।',
  'stats.refresh': 'रिफ्रेश करें',
  'stats.loadFailed': 'सामग्री आँकड़े लोड नहीं हो सके।',
  'stats.window': 'दिखाएँ',
  'stats.windowDays': 'पिछले {days} दिन',
  'stats.windowClamped': '(अधिकतम 90 दिन)',
  'stats.windowNote': 'हर दिन सर्वर मशीन के स्थानीय समय क्षेत्र के अनुसार कटा जाता है। यदि साइट UTC पर चलता है और प्रशासक किसी और क्षेत्र में है, तो आज के आँकड़े कम दिखेंगे — यह समय क्षेत्र का अंतर है, ट्रैफ़िक की गिरावट नहीं।',
  'stats.generatedAt': 'आँकड़े {time} पर बनाए गए',
  'stats.seriesTitle': 'प्रतिदिन नया',
  'stats.seriesNote': 'तीनों रेखाओं का पैमाना अलग-अलग है, इसलिए उन्हें एक के ऊपर एक नहीं बल्कि अलग-अलग दिखाया जाता है।',
  'stats.seriesUsers': 'नए उपयोगकर्ता',
  'stats.seriesPosts': 'नई पोस्ट',
  'stats.seriesComments': 'नई टिप्पणियाँ',
  'stats.seriesEmpty': 'इस अवधि में कोई डेटा नहीं।',
  'stats.totalsTitle': 'अवधि का योग',
  'stats.totalsNote': 'यह इस अवधि में जोड़ा गया मात्रा है, साइट का वर्तमान योग नहीं।',
  'stats.totalUsers': 'नए उपयोगकर्ता',
  'stats.totalPosts': 'नई पोस्ट',
  'stats.totalComments': 'नई टिप्पणियाँ',
  'stats.totalLikes': 'नए लाइक',
  'stats.topPostsTitle': 'लोकप्रिय पोस्ट',
  'stats.topPostsNote': 'टिप्पणियाँ और लाइक के अनुसार क्रमबद्ध, केवल इस अवधि में प्रकाशित पोस्ट गिने गए।',
  'stats.topTagsTitle': 'लोकप्रिय टैग',
  'stats.topTagsNote': 'कितने उपयोगकर्ताओं के पास हैं, उसके अनुसार क्रमबद्ध, समय-सीमा के बिना — टैग एक गुण है, घटना नहीं।',
  'stats.topAuthorsTitle': 'सक्रिय लेखक',
  'stats.topAuthorsNote': 'इस अवधि की पोस्ट के अनुसार क्रमबद्ध, टिप्पणियों की संख्या अलग दिखाई जाती है।',
  'stats.colExcerpt': 'अंश',
  'stats.colEngagement': 'सहभागिता',
  'stats.colPosts': 'पोस्ट',
  'stats.colComments': 'टिप्पणियाँ',
  'stats.colUsers': 'उपयोगकर्ता',
  'stats.colAuthor': 'लेखक',
  'stats.empty': 'इस अवधि में कोई डेटा नहीं।',
  'stats.emptyBody': 'अवधि बढ़ाएँ, या पुष्टि करें कि सचमुच कुछ नया नहीं हुआ।',
  'stats.engagement': '{comments} टिप्पणियाँ・{likes} लाइक',
  'stats.rank': 'स्थान {rank}',

  'export.title': 'निर्यात और बड़े पैमाने की कार्रवाई',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'मिलान के लिए साइट का डेटा CSV में निर्यात करें, या एक साथ कई खातों पर कार्रवाई करें।',
  'export.download': 'CSV डाउनलोड करें',
  'export.downloading': 'तैयार हो रहा है…',
  'export.exportTitle': 'निर्यात',
  'export.exportNote': 'हर निर्यात अधिकतम 50 हज़ार पंक्तियों तक सीमित है; उसके बाद केवल नवीनतम पंक्तियाँ शामिल होती हैं। तीनों निर्यात के कॉलम प्रशासन पृष्ठों से मेल खाते हैं, इसलिए मिलान सीधा होता है।',
  'export.exportUsers': 'उपयोगकर्ता सूची',
  'export.exportUsersNote': 'ईमेल, स्थिति, बनाने और बदलने का समय, पोस्ट और टिप्पणी की संख्या।',
  'export.exportPosts': 'पोस्ट सूची',
  'export.exportPostsNote': 'आईडी, लेखक, सामग्री के पहले 200 अक्षर, बनाने का समय, टिप्पणियाँ और लाइक।',
  'export.exportReports': 'रिपोर्ट सूची',
  'export.exportReportsNote': 'आईडी, रिपोर्ट की गई वस्तु, रिपोर्टकर्ता, कारण, स्थिति और समीक्षा का रिकॉर्ड।',
  'export.safety': 'फ़ाइल UTF-8 बाइट ऑर्डर मार्क से शुरू होती है, इसलिए Excel उसे खोलकर विकृत अक्षर नहीं दिखाता।',
  'export.safetyPrefix': 'जो मान = + - @ या अदृश्य रिक्तस्थान से शुरू होते हैं, उनके आगे एकल उद्धरण चिह्न लगता है — इसी से स्प्रेडशीट उन्हें पाठ मानकर सूत्र के रूप में चलाने से रोकती है। यह उपसर्ग जानबूझकर रखा गया है, इसे हटाने को न कहें।',
  'export.batchTitle': 'बड़े पैमाने की कार्रवाई',
  'export.batchNote': 'उपयोगकर्ता पृष्ठ पर खाते चुनने के बाद ये बटन सक्रिय होते हैं। बड़े पैमाने की कार्रवाई पूरी तरह लागू होती है या बिल्कुल नहीं; ऐसा कोई आंशिक परिणाम नहीं होता।',
  'export.batchSuspend': 'चयनित निलंबित करें',
  'export.batchReinstate': 'चयनित पुनः सक्रिय करें',
  'export.batchTags': 'टैग लगाएँ',
  'export.batchTagsNote': 'ओवरराइट शब्दावली: भेजी गई सूची ही परिणाम बन जाती है। खाली सूची भेजने का अर्थ सभी टैग हटाना है।',
  'export.batchConfirm': '{count} खातों पर “{action}” लागू करें?',
  'export.batchConfirmTags': '{count} खातों के टैग को {tags} से ओवरराइट करें?',
  'export.batchTagsPicker': 'टैग चुनें',
  'export.batchTagsNone': 'कोई टैग नहीं (सभी हटाएँ)',
  'export.batchRunning': 'चल रहा है…',
  'export.batchDone': '{updated} खाते अपडेट हुए',
  'export.batchDoneUnchanged': 'उनमें से {unchanged} पहले से लक्षित स्थिति में थे और बदले नहीं',
  'export.batchSkipped': '{count} छोड़ दिए गए',
  'export.batchMax': 'एक बार में अधिकतम 200 खाते',
  'export.gotoUsers': 'उपयोगकर्ताओं पर जाएँ',
  'export.noSelection': 'पहले उपयोगकर्ता पृष्ठ पर खाते चुनें।',
  'export.selected': '{count} खाते चुने गए',
  'export.clearSelection': 'चयन हटाएँ',
  'export.selectionHint': 'चयन केवल इस पृष्ठ पर रहता है और पृष्ठ बंद करने पर मिट जाता है।',

  'session.title': 'साइन-इन और सत्र',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'देखें कि कौन-से साइन-इन अब भी मान्य हैं, और किसी खाते को हर डिवाइस से बाहर करें।',
  'session.refresh': 'रिफ्रेश करें',
  'session.loadFailed': 'सत्र सूची लोड नहीं हो सकी।',
  'session.privacyTitle': 'पूरा टोकन क्यों नहीं दिखाया जाता',
  'session.privacyNote': 'टोकन ही लॉग-इन का प्रमाण है। केवल पहले आठ अक्षर दिखाए जाते हैं ताकि प्रशासक बता सके कि दो पंक्तियाँ एक ही सत्र हैं — और इतना किसी को भी उस उपयोगकर्ता के रूप में साइन-इन नहीं करने देता, यहाँ तक कि जिसे इस पेज का स्क्रीनशॉट मिल जाए। यह एक जानबूझकर सीमा है, अधूरी सुविधा नहीं।',
  'session.expireNote': 'सत्र {hours} घंटे की निष्क्रियता के बाद समाप्त हो जाता है; कोई भी अनुरोध इसे आगे बढ़ा देता है।',
  'session.filterEmail': 'खाते से छाँटें',
  'session.filterPlaceholder': 'पूरा ईमेल पता',
  'session.search': 'खोजें',
  'session.clearFilter': 'साफ़ करें',
  'session.summary': 'पूरी साइट पर {total} सत्र, {scanned} कुंजियाँ जाँची गईं',
  'session.truncated': 'स्कैन {scanned} कुंजियों की सीमा तक पहुँचकर पहले ही रुक गया, इसलिए यह सूची अधूरी है।',
  'session.empty': 'अभी कोई सत्र नहीं।',
  'session.emptyBody': 'कोई साइन-इन नहीं है, या सभी सत्र समाप्त हो चुके हैं।',
  'session.colUser': 'खाता',
  'session.colToken': 'सत्र',
  'session.colCreated': 'बनाया गया',
  'session.colExpires': 'समाप्त',
  'session.colRemaining': 'शेष',
  'session.colActions': 'कार्रवाई',
  'session.unknown': 'अज्ञात',
  'session.adminBadge': 'प्रशासन',
  'session.revoke': 'बाहर करने के लिए बाध्य करें',
  'session.revokeTitle': '{email} को बाहर करने के लिए बाध्य करें',
  'session.revokeMessage': 'इस खाते के {count} मौजूदा सत्र तुरंत अमान्य हो जाएँगे और हर डिवाइस से साइन-इन मिट जाएगा। उपयोगकर्ता को दोबारा साइन-इन करना होगा। जारी रखें?',
  'session.revokeRunning': 'रद्द किया जा रहा है…',
  'session.revokeDone': '{count} सत्र रद्द किए गए',
  'session.revokeNone': 'इस खाते का कोई सक्रिय सत्र नहीं',
  'session.revokeFailed': 'साइन-आउट की पुष्टि नहीं हो सकी।',
  'session.revokeUnavailable': 'इसका यह अर्थ नहीं कि रद्दीकरण विफल रहा: स्कैन अपनी कुंजी सीमा तक पहुँच गया, इसलिए कुछ सत्र तक पहुँचा ही नहीं होगा। थोड़ी देर बाद फिर कोशिश करें।',
  'session.titleColumnNote': 'केवल पहचान हेतु उपसर्ग, साइन-इन के लिए अनुपयोगी',

  'block.title': 'IP अवरुद्ध सूची',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'जिन पतों के दुरुपयोग की पुष्टि हो चुकी हो, उन्हें इस सूची में डालें। सूची Redis में है, इसलिए न पुनःआरंभ इसे मिटाता है और न कोई परिनियोजन।',
  'block.refresh': 'रिफ्रेश करें',
  'block.loadFailed': 'अवरुद्ध सूची लोड नहीं हो सकी।',
  'block.unavailable': 'इस साइट का Redis से संपर्क नहीं है, इसलिए अवरुद्धी सक्षम नहीं।',
  'block.unavailableNote': 'इस सूची को वही Redis चाहिए जो सत्रों और मीडिया टोकन के लिए इस्तेमाल होता है। जुड़ते ही यह पेज काम करने लगेगा; उससे पहले एकमात्र सुरक्षा दर दर (प्रति-प्रक्रिया) है, जो पुनःआरंभ पर मिट जाती है।',
  'block.add': 'अवरुद्ध करें',
  'block.addTitle': 'एक IP अवरुद्ध करें',
  'block.addMessage': 'अवरुद्ध पता हर लेखन अनुरोध (पोस्ट, टिप्पणी, लाइक, रिपोर्ट, छवि अपलोड, लॉगिन रीडायरेक्ट) पर अस्वीकार किया जाता है, जब तक समय समाप्त न हो। पढ़ना प्रभावित नहीं होता।',
  'block.ipLabel': 'IP पता',
  'block.ipPlaceholder': '203.0.113.9 या 2001:db8::1',
  'block.durationLabel': 'अवधि',
  'block.reasonLabel': 'कारण',
  'block.reasonPlaceholder': 'यह पता क्यों अवरुद्ध किया जा रहा है (ऑडिट लॉग में दर्ज होगा)',
  'block.reasonHint': 'कारण केवल ऑडिट लॉग में जाता है। यह अवरुद्ध व्यक्ति को कभी नहीं दिखाया जाता और सार्वजनिक त्रुटि संदेश में भी नहीं आता।',
  'block.blocking': 'अवरुद्ध हो रहा है…',
  'block.done': '{ip} अवरुद्ध कर दिया गया',
  'block.removed': '{ip} का अवरुद्ध हटा दिया गया',
  'block.removedNone': '{ip} पहले से अवरुद्ध नहीं था',
  'block.failed': 'अवरुद्ध करने का कार्य विफल रहा।',
  'block.unavailableService': 'अवरुद्ध सूची अनुपलब्ध (Redis नहीं)',
  'block.colIp': 'IP',
  'block.colExpires': 'समाप्ति',
  'block.colRemaining': 'शेष',
  'block.colActions': 'कार्रवाई',
  'block.unblock': 'अवरुद्ध हटाएँ',
  'block.unblockTitle': '{ip} का अवरुद्ध हटाएँ',
  'block.unblockMessage': 'यह पता तुरंत सामान्य लेखन पहुँच वापस पा लेगा। जारी रखें?',
  'block.empty': 'अवरुद्ध सूची खाली है।',
  'block.emptyBody': 'कोई पता अवरुद्ध नहीं है। यही सामान्य अवस्था है: अवरुद्धी हमेशा व्यवस्थापक का निर्णय होती है, और प्रणाली स्वयं किसी को अवरुद्ध नहीं करती।',
  'block.notAutoNote': 'यह सूची स्वयं भरती नहीं। दर से अधिक अनुरोध करने वाले पते को केवल 429 मिलता है; वह यहाँ स्वतः नहीं जोड़ा जाता, क्योंकि वही निकास किसी पूरे कार्यालय या पूरे NAT का हो सकता है, और स्वचालित अवरुद्धी से वे भी प्रभावित होंगे।',
  'block.scopeNoteLabel': 'दायरा',
  'block.notAutoNoteLabel': 'कभी स्वचालित नहीं',
  'block.maxNoteLabel': 'अधिकतम अवधि',
  'block.scopeNote': 'अवरुद्धी केवल लेखन अनुरोध रोकती है। पोस्ट, टिप्पणी और स्थिर संपत्तियाँ पढ़ी जा सकती हैं, और अवरुद्ध व्यक्ति फिर भी साइन इन करके सामग्री देख सकता है: यह जानबूझकर है, क्योंकि पढ़ने वाले एंडपॉइंट जानबूझकर दर-सीमित नहीं हैं (वरना अनाम आगंतुक साइट का उपयोग ही नहीं कर पाते), और अवरुद्धी उसी समूह को कवर करती है।',
  'block.maxNote': 'एक अवरुद्धी अधिकतम 365 दिन रहती है। इससे अधिक अवधि को एक साल तक घटा दिया जाता है: समाप्ति एक संख्या के रूप में संग्रहीत होती है, और “हमेशा के लिए” ऐसी अवरुद्धी बन जाएगी जिसे कोई याद न रखे और जो कभी स्वतः हटे नहीं।',
  'block.count': '{count} पते अवरुद्ध',
  'block.ipInvalid': 'IP पता मान्य नहीं है। IPv4 या IPv6 पता दर्ज करें; CIDR श्रृंखला समर्थित नहीं है।',
  'announce.label': 'साइट घोषणा',
  'announce.publicNote': 'सूचना',
  'announce.closeAria': 'यह घोषणा बंद करें',
  'announce.publishedOn': '{date} को प्रकाशित',
  'announce.expiresOn': '{date} को समाप्त',
  'announce.neverExpires': 'कोई समाप्ति तिथि नहीं',
  'announce.pinnedBadge': 'पिन किया गया',
  'announce.title': 'घोषणाएँ',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'साइट के हर पृष्ठ पर एक घोषणा दिखाएँ। एक समय में केवल एक ही प्रभावी होती है — नई घोषणा प्रकाशित करने पर पिछली अप्रभावी हो जाती है।',
  'announce.refresh': 'रिफ्रेश करें',
  'announce.loadFailed': 'घोषणाएँ लोड नहीं हो सकीं।',
  'announce.new': 'नई घोषणा प्रकाशित करें',
  'announce.edit': 'संपादित करें',
  'announce.deactivate': 'निष्क्रिय करें',
  'announce.reactivate': 'पुनः सक्रिय करें',
  'announce.deleteNote': 'घोषणाएँ कभी नहीं हटाई जातीं, केवल निष्क्रिय की जाती हैं — इतिहास रखना ही यह बताता है कि यह कब और किसने प्रकाशित की।',
  'announce.bodyLabel': 'घोषणा का पाठ',
  'announce.bodyPlaceholder': 'उदाहरण: गुरुवार को 02:00–04:00 बजे तक सिस्टम में रखरखाव होगा।',
  'announce.bodyHint': 'अधिकतम 300 अक्षर। सादा पाठ; पंक्तियाँ बनी रहती हैं।',
  'announce.activeLabel': 'तुरंत दिखाएँ',
  'announce.expiryLabel': 'वैधता',
  'announce.expiryNever': 'स्वतः कभी समाप्त नहीं होती',
  'announce.expiryHours': '{hours} घंटे बाद',
  'announce.expiryDays': '{days} दिन बाद',
  'announce.saving': 'सहेजा जा रहा है…',
  'announce.published': 'घोषणा प्रकाशित',
  'announce.updated': 'घोषणा अपडेट हुई',
  'announce.deactivated': 'घोषणा निष्क्रिय की गई',
  'announce.reactivated': 'घोषणा पुनः सक्रिय की गई',
  'announce.saveFailed': 'घोषणा का कार्य विफल रहा।',
  'announce.empty': 'अभी तक कोई घोषणा नहीं।',
  'announce.emptyBody': 'जैसे ही आप एक प्रकाशित करेंगे, वह हर आगंतुक के पृष्ठ पर सबसे ऊपर दिखेगी।',
  'announce.colBody': 'पाठ',
  'announce.colState': 'स्थिति',
  'announce.colAuthor': 'प्रकाशक',
  'announce.colCreated': 'प्रकाशन समय',
  'announce.colActions': 'कार्रवाई',
  'announce.stateActive': 'प्रदर्शित',
  'announce.stateInactive': 'निष्क्रिय',
  'announce.stateExpired': 'समाप्त',
  'announce.confirmDeactivate': 'इसे प्रकाशित करने पर मौजूदा घोषणा निष्क्रिय हो जाएगी, और सभी को तुरंत नया पाठ दिखेगा। जारी रखें?',
  'announce.confirmEdit': 'इस घोषणा का पाठ या वैधता बदलें?',
  'announce.count': 'कुल {count}',
  'users.title':
    'उपयोगकर्ता प्रबंधन',
  'users.contentAction':
    'सामग्री',
  'users.updateContentFailed':
    'सामग्री की क्रिया असफल।',
  'users.updated':
    'सामग्री अपडेट हो गई है।',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'फोरम उपयोगकर्ताओं की गतिविधि संख्या, टैग और अकाउंट स्थिति देखें, तथा निलंबन और पोस्ट-दर-पोस्ट सामग्री प्रबंधन करें।',
  'users.refresh':
    'डेटा अद्यतन करें',
  'users.statTotal':
    'कुल उपयोगकर्ता',
  'users.statActive':
    'सक्रिय',
  'users.statSuspended':
    'निलंबित',
  'users.statContent':
    'पोस्ट / टिप्पणियों की कुल संख्या',
  'users.count':
    '{count} उपयोगकर्ता',
  'users.tagsCount':
    '{count} टैग',
  'users.loadFailed':
    'उपयोगकर्ता डेटा लोड नहीं हो सका।',
  'users.tagsLoadFailed':
    'टैग डेटा लोड नहीं हो सका।',
  'users.panelTitle':
    'फोरम उपयोगकर्ता',
  'users.tagsPanelTitle':
    'उपयोगकर्ता टैग',
  'users.colUser':
    'उपयोगकर्ता',
  'users.colTags':
    'टैग',
  'users.colStatus':
    'स्थिति',
  'users.colPosts':
    'पोस्ट',
  'users.colComments':
    'टिप्पणियाँ',
  'users.colLikes':
    'पसंद',
  'users.colLastActivity':
    'अंतिम गतिविधि',
  'users.colActions':
    'प्रक्रिया',
  'users.nicknameUnset':
    'उपनाम सेट नहीं किया गया',
  'users.notSet':
    'सेट नहीं किया गया',
  'users.statusActive':
    'सक्रिय',
  'users.statusSuspended':
    'निलंबित',
  'users.emptyTitle':
    'अभी कोई उपयोगकर्ता डेटा नहीं है',
  'users.emptyBody':
    'यदि Google से लॉग इन करने वाले कोई उपयोगकर्ता नहीं हैं, तो यह खाली होगा।',
  'users.tagsEmptyTitle':
    'अभी कोई टैग नहीं है',
  'users.tagsEmptyBody':
    'पहले टैग बनाएँ, फिर उपयोगकर्ता सूची में व्यक्ति को टैग कर सकें।',
  'users.addTag':
    'टैग जोड़ें',
  'users.colName':
    'नाम',
  'users.colCreated':
    'निर्मित समय',
  'users.colUpdated':
    'अद्यतन समय',
  'users.renameTag':
    'टैग का नाम बदलें',
  'users.suspend':
    'निलंबित',
  'users.restore':
    'पुनर्स्थापित करें',
  'users.statusDialogTitle':
    '{action} इस उपयोगकर्ता को',
  'users.statusSuspendMessage':
    '{email} को अब फोरम में लॉग इन नहीं करने मिलेगा; उनकी प्रचलित पोस्ट और टिप्पणियाँ बनी रहेंगी। जारी रखें?',
  'users.statusRestoreMessage':
    '{email} का लॉग इन और पोस्ट करने का अधिकार पुनर्प्राप्त हो जाएगा। जारी रखें?',
  'users.userSuspended':
    'उपयोगकर्ता निलंबित हो चुका है।',
  'users.userRestored':
    'पुनर्प्राप्त कर दिया गया है।',
  'users.updateStatusFailed':
    'प्रयोगकर्ता की स्थिति अपडेट करने में असफलता हुई।',
  'users.editTagsTitle':
    'टैग संपादित करें · {user}',
  'users.editTagsMessage':
    'लागू करने वाले टैग चुनें; सभी चुनाव हटाने से इस प्रयोगकर्ता के सभी टैग हट जाएंगे।',
  'users.tagsUpdated':
    'प्रयोगकर्ता के टैग अपडेट कर दिए गए हैं।',
  'users.updateTagsFailed':
    'प्रयोगकर्ता के टैग अपडेट करने में असफलता हुई।',
  'users.contentLoadFailed':
    'सामग्री लोड करने में असफलता हुई।',
  'users.contentLoadFailedShort':
    'सामग्री लोड नहीं हुई।',
  'users.contentPanelTitle':
    'प्रयोगकर्ता की सामग्री',
  'users.contentCount':
    '{posts} पोस्ट · {comments} टिप्पणियाँ',
  'users.contentLoading':
    'पोस्ट और टिप्पणियाँ लोड हो रही हैं...',
  'users.addPost':
    'नया पोस्ट जोड़ें',
  'users.addComment':
    'नई टिप्पणी जोड़ें',
  'users.postsColumn':
    'पोस्ट',
  'users.commentsColumn':
    'टिप्पणी',
  'users.noPosts':
    'कोई पोस्ट नहीं है',
  'users.noComments':
    'कोई टिप्पणी नहीं है',
  'users.postRef':
    'पोस्ट #{id}',
  'users.editRecordTitle':
    '{kind} संपादित करें #{id}',
  'users.deleteRecordTitle':
    '{kind} हटाएँ #{id}',
  'users.deleteRecordMessage':
    'हटाने के बाद इसे पुनर्प्राप्त नहीं किया जा सकता; संबंधित पसंद और संबंधित डेटा भी हटा दिए जाएंगे। क्या जारी रखें?',
  'users.contentLabel':
    'सामग्री',
  'users.addPostTitle':
    'नया पोस्ट जोड़ें',
  'users.addPostMessage':
    'यह सामग्री उस प्रयोगकर्ता के नाम से प्रकाशित होगी; लेखक का क्षेत्र मनमाने से नहीं बदला जा सकता।',
  'users.postContentLabel':
    'पोस्ट सामग्री',
  'users.postContentPlaceholder':
    'पोस्ट की सामग्री दर्ज करें',
  'users.pickPostTitle':
    'पोस्ट चुनें',
  'users.postIdLabel':
    'पोस्ट ID',
  'users.postIdPlaceholder':
    'टिप्पणी करने वाले पोस्ट का नंबर',
  'users.addCommentTitle':
    'नई टिप्पणी जोड़ें',
  'users.commentContentLabel':
    'टिप्पणी सामग्री',
  'users.commentContentPlaceholder':
    'टिप्पणी की सामग्री दर्ज करें',
  'users.createTagTitle':
    'नया टैग जोड़ें',
  'users.createTagMessage':
    'टैग का उपयोग प्रयोगकर्ताओं को श्रेणीबद्ध करने के लिए किया जा सकता है, जैसे “प्रबंधक”, “सक्रिय” या “निषिद्ध”।',
  'users.tagNameLabel':
    'टैग का नाम',
  'users.tagNamePlaceholder':
    'अधिकतम 50 अक्षर',
  'users.renameTagTitle':
    'टैग का नाम बदलें',
  'users.renameTagMessage':
    'इस टैग से जुड़े सभी प्रयोगकर्ता नया नाम देखेंगे।',
  'users.deleteTagTitle':
    'टैग “{name}” हटाएँ',
  'users.deleteTagMessage':
    'हटाने के बाद, इस टैग से जुड़े सभी प्रयोगकर्ताओं से यह टैग भी हट जाएगा और इसे पुनर्प्राप्त नहीं किया जा सकेगा। क्या जारी रखें?',
  'users.deleteTagConfirm':
    'टैग हटाएँ',
  'users.tagCreated':
    'टैग बना दिया गया है।',
  'users.createTagFailed':
    'टैग बनाने में असफलता हुई।',
  'users.tagUpdated':
    'टैग अपडेट कर दिया गया है।',
  'users.updateTagFailed':
    'टैग अपडेट करने में असफलता हुई।',
  'users.tagDeleted':
    'टैग हटा दिया गया है।',
  'users.deleteTagFailed':
    'टैग हटाने में असफलता हुई।',
  'users.refreshDone':
    'ताज़ा डेटा पर अपडेट हो चुका है।',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'बैकएंड प्रबंधन',
  'users.signinBody':
    'केंद्रित सामग्री प्रबंधन, जिससे प्रत्येक समीक्षा स्पष्ट, त्वरित और ट्रेस करने योग्य बनती है।',
  'users.signinStep1':
    'सुरक्षित सत्यापन',
  'users.signinStep2':
    'प्रयोगकर्ता प्रबंधन',
  'users.signinStep3':
    'सामग्री समीक्षा',
  'users.signinPanelTitle':
    'प्रबंधन कंसोल में प्रवेश करें',
  'users.signinPanelBody':
    'प्रबंधन कंसोल केवल अनुमति प्राप्त Google प्रबंधक खातों को खोला जाता है; कृपया प्रबंधक के रूप में लॉगिन करें।',
  'posts.title':
    'फोरम पोस्ट',
  'posts.searching':
    'खोज रहे हैं...',
  'posts.searchDegraded':
    '(खोज सेवा सक्रिय नहीं है, डेटाबेस की कुंजी शब्दों से मिलान किया जा रहा है)',
  'posts.searchSummary':
    '“{query}” के लिए {total} परिणाम मिले; इस पृष्ठ पर {shown} दिखाए गए',
  'posts.pendingCount':
    '{count} प्रतीक्षा में',
  'posts.pageSummary':
    'पृष्ठ {page} / {pages}, इस पृष्ठ पर {count} पोस्ट',
  'posts.listLoadFailed':
    'पोस्ट सूची लोड करने में असफलता हुई।',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'फोरम पोस्ट बनाएँ, संपादित करें और हटाएँ, और टिप्पणी स्तर पर सामग्री प्रबंधन करें।',
  'posts.toReports':
    'रिपोर्ट प्रबंधन',
  'posts.editorTitleNew':
    'नया पोस्ट जोड़ें',
  'posts.editorTitleEdit':
    'पोस्ट #{id} संपादित करें',
  'posts.editorNote':
    'प्रबंधक के रूप में पोस्ट करें; लेखक लॉगिन किए गए प्रयोगकर्ता से लिया जाता है, अनुरोध की सामग्री से लेखक को झूठा नहीं किया जा सकता।',
  'posts.cancelEdit':
    'संपादन रद्द करें',
  'posts.contentLabel':
    'पोस्ट सामग्री',
  'posts.contentPlaceholder':
    'पोस्ट की सामग्री दर्ज करें',
  'posts.saveChanges':
    'परिवर्तन सुरक्षित करें',
  'posts.emptyContent':
    'पोस्ट सामग्री खाली नहीं हो सकती।',
  'posts.saving':
    'सुरक्षित किया जा रहा है...',
  'posts.saved':
    'पोस्ट अपडेट हो गया है।',
  'posts.published':
    'पोस्ट प्रकाशित हो गया है।',
  'posts.saveFailed':
    'सुरक्षित करने में असफलता हुई।',
  'posts.deleteTitle':
    'पोस्ट #{id} हटाएँ',
  'posts.deleteMessage':
    'हटाने के बाद इसे पुनर्प्राप्त नहीं किया जा सकता; उस पोस्ट के नीचे सभी टिप्पणियाँ भी हटा दी जाएँगी। क्या जारी रखें?',
  'posts.deleteConfirm':
    'पोस्ट हटाएँ',
  'posts.deleted':
    'पोस्ट हटा दिया गया है।',
  'posts.deleteFailed':
    'पोस्ट हटाने में असफलता हुई।',
  'posts.edited':
    'पोस्ट अपडेट हो गई।',
  'posts.editFailed':
    'पोस्ट सहेजी नहीं जा सकी, कृपया बाद में फिर प्रयास करें।',
  'posts.commentUpdated':
    'टिप्पणी अपडेट हो गई है।',
  'posts.commentActionFailed':
    'टिप्पणी क्रिया असफल हुई।',
  'posts.pickPostTitle':
    'टिप्पणी से संबंधित पोस्ट चुनें',
  'posts.pickPostMessage':
    'डिफ़ॉल्ट रूप से इसी पंक्ति का पोस्ट है; किसी अन्य पोस्ट पर लगाने के लिए सही पोस्ट नंबर बदलें।',
  'posts.postIdLabel':
    'पोस्ट ID',
  'posts.addCommentAtTitle':
    'पोस्ट #{id} में नई टिप्पणी जोड़ें',
  'posts.addCommentMessage':
    'यह टिप्पणी प्रबंधक के रूप में पोस्ट होगी।',
  'posts.commentContentLabel':
    'टिप्पणी सामग्री',
  'posts.commentContentPlaceholder':
    'टिप्पणी की सामग्री दर्ज करें',
  'posts.add':
    'जोड़ें',
  'posts.editCommentTitle':
    'टिप्पणी #{id} संपादित करें',
  'posts.deleteCommentTitle':
    'टिप्पणी #{id} हटाएँ',
  'posts.deleteCommentMessage':
    'हटाने के बाद इसे पुनर्प्राप्त नहीं किया जा सकता। क्या जारी रखें?',
  'posts.listTitle':
    'पोस्ट सूची',
  'posts.clearSearch':
    'खोज हटाएँ',
  'posts.searchLabel':
    'कीवार्ड खोज',
  'posts.searchPlaceholder':
    'पोस्ट का सामग्री या पोस्ट लेखक का पूर्ण Email खोजें',
  'posts.searchHint':
    'तुल्यता के अनुसार क्रमबद्ध; पूर्ण Email दर्ज करें ताकि उस उपयोगकर्ता की सभी पोस्ट मिल सकें। खोज पेजिंग की जगह ले लेती है, और परिणाम अधिकतम 25 दिखाए जाएंगे।',
  'posts.searchTotal':
    'खोज परिणाम कुल {total}',
  'posts.searchFailed':
    'खोज असफल हो गई।',
  'posts.searchStatusFailed':
    'खोज असफल',
  'posts.colContentImage':
    'सामग्री और चित्र',
  'posts.colEngagement':
    'प्रतिक्रिया',
  'posts.colComments':
    'टिप्पणियाँ',
  'posts.colAuthor':
    'लेखक',
  'posts.imageAlt':
    'पोस्ट का चित्र',
  'posts.likes':
    '{count} पसंद',
  'posts.author':
    'लेखक: {name}',
  'posts.emptyTitle':
    'अभी कोई फोरम पोस्ट नहीं है',
  'posts.emptyBody':
    'ऊपर दिए गए एडिटर से प्रथम पोस्ट बनाई जा सकती है।',
  'posts.emptySearchTitle':
    'कोई मिलती पोस्ट नहीं है',
  'posts.emptySearchBody':
    '«{query}» के लिए कोई पोस्ट मिली ही नहीं। कृपया कोई अन्य कीवार्ड आज़माएँ।',
  'posts.commentCount':
    '{count} टिप्पणियाँ',
  'posts.noComments':
    'अभी कोई टिप्पणी नहीं है',
  'posts.reportsTitle':
    'लंबित रिपोर्ट',
  'posts.allReports':
    'सभी रिपोर्ट',
  'posts.colReportedContent':
    'रिपोर्ट किया गया सामग्री',
  'posts.colReason':
    'कारण',
  'posts.colReporter':
    'रिपोर्ट करने वाला',
  'posts.colTime':
    'समय',
  'posts.colVerdict':
    'निर्णय',
  'posts.emptyReportsTitle':
    'लंबित कोई रिपोर्ट नहीं है',
  'posts.emptyReportsBody':
    'सभी रिपोर्ट पर निर्णय ले लिया गया है।',
  'posts.verdictResolved':
    'हल किया गया',
  'posts.verdictRejected':
    'अमान्य',
  'posts.verdictDialogTitle':
    'रिपोर्ट #{id} को «{label}» के रूप में चिह्नित करें',
  'posts.verdictDialogMessage':
    'चिह्नित करने के बाद यह रिपोर्ट लंबित सूची से हट जाएगी, लेकिन डेटा रिपोर्ट प्रबंधन पृष्ठ पर बना रहेगा। जारी रखें?',
  'posts.verdictConfirm':
    '{label} के रूप में चिह्नित करें',
  'posts.verdictDone':
    'रिपोर्ट को {label} के रूप में चिह्नित किया गया।',
  'posts.verdictFailed':
    'रिपोर्ट स्थिति अपडेट करने में असफलता।',
  'posts.pin': 'पिन करें',
  'posts.unpin': 'अनपिन करें',
  'posts.pinTitle': 'इस पोस्ट को पिन करें',
  'posts.unpinTitle': 'इस पोस्ट को अनपिन करें',
  'posts.pinMessage': 'पिन किए जाने पर यह पोस्ट सबकी फ़ीड में सबसे ऊपर रहता है और नई पोस्ट इसे नीचे नहीं धकेल सकतीं।',
  'posts.unpinMessage': 'अनपिन करने पर यह पोस्ट समय के क्रम में अपनी जगह पर लौट जाती है।',
  'posts.pinDone': 'पिन किया गया',
  'posts.unpinDone': 'अनपिन किया गया',
  'posts.pinFailed': 'पिन करने का कार्य विफल रहा।',
  'reports.title':
    'रिपोर्ट प्रबंधन',
  'reports.listSummary':
    '{count} रिकॉर्ड · {filter}',
  'reports.listLoadFailed':
    'रिपोर्ट सूची लोड करने में असफलता।',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'एक-एक करके रिपोर्ट पर निर्णय लें: «स्वीकृत» चिह्नित करने से रिपोर्ट किया गया सामग्री हट जाएगी, जबकि «अमान्य» चिह्नित करने से मूल सामग्री बना रहेगी। «लंबित» पर वापस बदलने से पूर्व निर्णय समय हट जाएगा।',
  'reports.backToPosts':
    'पोस्ट पर लौटें',
  'reports.editorTitle':
    'रिपोर्ट #{id} संपादित करें',
  'reports.editorNote':
    'रिपोर्ट के कारण और स्थिति बदली जा सकती हैं; लक्ष्य और रिपोर्ट करने वाला पूर्व-रजिस्टर्ड रिकॉर्ड हैं, इसलिए इन्हें यहाँ नहीं बदला जाता।',
  'reports.targetTypeLabel':
    'लक्ष्य प्रकार',
  'reports.targetIdLabel':
    'लक्ष्य ID',
  'reports.reporterEmailLabel':
    'रिपोर्ट करने वाले का Email',
  'reports.statusLabel':
    'स्थिति',
  'reports.reasonLabel':
    'कारण',
  'reports.reasonHint':
    'अधिकतम 500 अक्षर; इसे अन्य प्रशासकों को निर्णय के आधार के रूप में सीधे दिखाया जाएगा।',
  'reports.filterLabel':
    'स्थिति के अनुसार छानें',
  'reports.filterAll':
    'सभी',
  'reports.listTitle':
    'रिपोर्ट सूची',
  'reports.colTarget':
    'लक्ष्य सामग्री',
  'reports.colReason':
    'कारण',
  'reports.colReporter':
    'रिपोर्ट करने वाला',
  'reports.colStatus':
    'स्थिति',
  'reports.targetGone':
    '(सामग्री हट चुकी है)',
  'reports.author':
    'लेखक: {name}',
  'reports.deleteTitle':
    'रिपोर्ट #{id} हटाएँ',
  'reports.deleteMessage':
    'यह रिपोर्ट रिकॉर्ड ही हटाई जाएगी; रिपोर्ट किया गया सामग्री प्रभावित नहीं होगी और उसे फिर से पुनर्स्थापित नहीं किया जा सकेगा। जारी रखें?',
  'reports.deleteConfirm':
    'रिपोर्ट हटाएँ',
  'reports.deleted':
    'रिपोर्ट हटा दी गई।',
  'reports.deleteFailed':
    'रिपोर्ट हटाने में असफलता।',
  'reports.updated':
    'रिपोर्ट अपडेट कर दी गई।',
  'reports.saveFailed':
    'रिपोर्ट संरक्षित करने में असफलता।',
  'reports.approveTitle':
    'रिपोर्ट #{id} स्वीकार करें',
  'reports.approveGoneMessage':
    'रिपोर्ट किया गया {kind} #{id} अब मौजूद नहीं है; यह केवल हल किया गया के रूप में चिह्नित किया जाएगा।',
  'reports.approveMessage':
    'रिपोर्ट किया गया {kind} #{id} हमेशा के लिए हटाए जाएगा (यदि यह पोस्ट है, तो उसकी सभी टिप्पणियाँ भी हटाई जाएंगी), और यह रिपोर्ट हल किया गया के रूप में चिह्नित किया जाएगा। जारी रखें?',
  'reports.approveConfirm':
    'रिपोर्ट स्वीकार करें और पोस्ट हटाएँ',
  'reports.approveGoneDone':
    'सामग्री अब मौजूद नहीं है, रिपोर्ट हल किया गया के रूप में चिह्नित की गई।',
  'reports.approveDone':
    'पोस्ट हटा दी गई और रिपोर्ट हल किया गया के रूप में चिह्नित की गई।',
  'reports.approveFailed':
    'रिपोर्ट स्वीकार करने में असफलता।',
  'reports.approveTitleGone':
    'सामग्री हट चुकी है, केवल रिपोर्ट चिह्नित की जाएगी',
  'reports.approveTitleFull':
    'रिपोर्ट किया गया सामग्री हटाएँ और रिपोर्ट को हल किया गया के रूप में चिह्नित करें',
  'reports.rejectTitle':
    'रिपोर्ट #{id} अमान्य',
  'reports.rejectMessage':
    'अमान्य का अर्थ है कि रिपोर्ट किया गया सामग्री पर कोई कार्रवाई आवश्यक नहीं है, और उसे जैसा है तैसा ही रखा जाएगा। जारी रखें?',
  'reports.rejectConfirm':
    'अमान्य के रूप में चिह्नित करें',
  'reports.rejectDone':
    'रिपोर्ट को अमान्य के रूप में चिह्नित किया गया।',
  'reports.statusFailed':
    'रिपोर्ट स्थिति अपडेट करने में असफलता।',
  'reports.emptyTitle':
    'अभी कोई रिपोर्ट नहीं है',
  'reports.emptyBody':
    'इस फिल्टर के अंतर्गत कोई रिकॉर्ड नहीं है।',
  'reports.rejectTitleAttr':
    'सामग्री बनाए रखें और केवल रिपोर्ट को अमान्य के रूप में चिह्नित करें',
  'kind.post':
    'पोस्ट',
  'kind.comment':
    'टिप्पणी',
  'reports.statusPending':
    'लंबित',
  'reports.statusResolved':
    'हल किया गया',
  'reports.statusRejected':
    'अमान्य',
  'title.forum':
    '{site}',
  'title.login':
    'लॉग इन | {site}',
  'title.newPost':
    'नई पोस्ट | {site}',
  'title.profile':
    'प्रोफ़ाइल | {site}',
  'title.publicProfile':
    'सार्वजनिक प्रोफ़ाइल | {site}',
  'title.post':
    'पोस्ट｜{site}',
  'title.following':
    'फ़ॉलोइंग | {site}',
  'title.adminUsers':
    'उपयोगकर्ता प्रबंधन | {brand} Admin Console',
  'title.adminLogin':
    'लॉग इन | {brand} Admin Console',
  'title.adminPosts':
    'फोरम पोस्ट | {brand} Admin Console',
  'title.adminReports':
    'रिपोर्ट प्रबंधन | {brand} Admin Console',
  'title.adminMonitor': 'सिस्टम मॉनिटरिंग｜{brand} व्यवस्थापक',
  'title.adminLog': 'कार्य लॉग｜{brand} व्यवस्थापक',
  'title.adminStats': 'सामग्री की प्रवृत्ति｜{brand} व्यवस्थापक',
  'title.adminExport': 'निर्यात और बड़े पैमाने की कार्रवाई｜{brand} व्यवस्थापक',
  'title.adminSessions': 'साइन-इन और सत्र｜{brand} व्यवस्थापक',
  'title.adminBlocks': 'IP अवरुद्ध सूची｜{brand} व्यवस्थापक',
  'title.adminAnnouncements': 'घोषणाएँ｜{brand} व्यवस्थापक',
};
