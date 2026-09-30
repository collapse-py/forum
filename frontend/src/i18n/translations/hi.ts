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
  'error.fallbackReport':
    'रिपोर्ट करने में विफलता',
  'error.fallbackProfile':
    'व्यक्तिगत जानकारी पढ़ने में विफलता',
  'error.fallbackProfileSave':
    'संभालने में विफलता',
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
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'सार्वजनिक प्रोफ़ाइल',
  'publicProfile.avatar':
    'अनोनिमस',
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
};
