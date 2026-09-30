/*
 * th catalog (src/i18n/translations/th.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const th: Record<MessageKey, string> = {
  'common.cancel':
    'yกเลิก',
  'common.save':
    'บันทึกลง',
  'common.submitting':
    'กำลังส่ง...',
  'common.delete':
    'ลบ',
  'common.edit':
    'แก้ไข',
  'common.search':
    'ค้นหา',
  'common.loading':
    'กำลังโหลด...',
  'common.loadFailed':
    'โหลดไม่สำเร็จ',
  'common.refresh':
    'รีเฟรช',
  'common.nextStep':
    'ขั้นต่อไป',
  'common.prevPage':
    'หน้าก่อนหน้า',
  'common.nextPage':
    'หน้าถัดไป',
  'common.create':
    'สร้าง',
  'common.publish':
    'เผยแพร่',
  'common.placeholder':
    '—',
  'common.backToHome':
    'คืนหน้าแรก',
  'common.backToForumHome':
    'คืนสู่หน้าฟอรั่ม',
  'common.backOnePage':
    'ย้อนกลับหน้าเดียว',
  'error.request':
    'โปรดลองใหม่อีกครั้ง',
  'error.requestStatus':
    'คำขอไม่สำเร็จ (HTTP {status})',
  'error.loginRequired':
    'ต้องเข้าสู่ระบบก่อนจึงสามารถต่อได้',
  'error.adminSessionExpired':
    'สถานะการเข้าสู่ระบบหมดอายุ แล้วจะกลับไปหน้าเข้าสู่ระบบ',
  'error.fallbackLoad':
    'โหลดไม่สำเร็จ',
  'error.fallbackSearch':
    'ค้นหาไม่สำเร็จ',
  'error.fallbackLike':
    'กดถูกใจไม่สำเร็จ',
  'error.fallbackComments':
    'โหลดความคิดเห็นไม่สำเร็จ',
  'error.fallbackCommentPost':
    'ส่งความคิดเห็นไม่สำเร็จ',
  'error.fallbackReport':
    'รายงานไม่สำเร็จ',
  'error.fallbackProfile':
    'อ่านข้อมูลโปรไฟล์ไม่สำเร็จ',
  'error.fallbackProfileSave':
    'บันทึกลงไม่สำเร็จ',
  'error.fallbackPublish':
    'รูปแบบการเผยแพร่ตอบกลับไม่ถูกต้อง',
  'error.fallbackUpload':
    'รูปแบบคำตอบของการอัปโหลดรูปภาพไม่ถูกต้อง',
  'error.fallbackNotFound':
    'ไม่พบผู้ใช้คนนี้',
  'error.fallbackFollow':
    'ติดตามไม่สำเร็จ',
  'auth.checking':
    'กำลังตรวจสอบสถานะการเข้าสู่ระบบ...',
  'auth.statusUnknown':
    'ไม่สามารถยืนยันสถานะการเข้าสู่ระบบ',
  'auth.feedLoggedIn':
    'เข้าสู่ระบบแล้ว สามารถโพสต์ได้',
  'auth.feedLoggedOut':
    'เข้าสู่ระบบแล้วจึงสามารถโพสต์ได้',
  'auth.profileLoggedIn':
    'เข้าสู่ระบบแล้ว',
  'auth.profileLoggedOut':
    'เข้าสู่ระบบแล้วจึงสามารถตั้งค่าโปรไฟล์ได้',
  'auth.googleLogin':
    'Google เข้าสู่ระบบ',
  'auth.loginWithGoogle':
    'เข้าสู่ระบบด้วยบัญชี Google',
  'auth.loginWithGoogleAdmin':
    'เข้าสู่ระบบด้วยบัญชีผู้จัดการ Google',
  'auth.logout':
    'ออกจากระบบ',
  'install.button':
    'ติดตั้งแอป',
  'install.hint':
    'เบราว์เซอร์ยังไม่มีคำสั่งติดตั้งอัตโนมัติ โปรดเปิดเมนูเบราว์เลือกติดตั้งแอป หรือเพิ่มลงหน้าแรก',
  'bottomNav.label':
    'การนำทางหลัก',
  'bottomNav.home':
    'หน้าแรก',
  'bottomNav.new':
    'เพิ่ม',
  'bottomNav.profile':
    'โปรไฟล์',
  'i18n.ariaLabel':
    'เลือกภาษา',
  'i18n.current':
    'ภาษา: {name}',
  'feed.searchPlaceholder':
    'ค้นหาในโพสต์',
  'feed.searchAriaLabel':
    'ค้นหาโพสต์',
  'feed.searchResultsLabel':
    'ผลการค้นหา',
  'feed.postsLabel':
    'บทความฟอรั่ม',
  'feed.searchFailed':
    'ค้นหาไม่สำเร็จ โปรดลองใหม่อีกครั้ง',
  'feed.searching':
    'กำลังค้นหา...',
  'feed.searchMore':
    'โหลดผลการค้นหาเพิ่มเติม...',
  'feed.searchMoreFailed':
    'โหลดไม่สำเร็จ',
  'feed.searchFound':
    'พบ {total} รายการ',
  'feed.searchDegraded':
    '{base} (ไม่ได้เปิดใช้งานบริการค้นหา ขณะนี้ใช้การเปรียบเทียบคีย์คำจากฐานข้อมูล)',
  'feed.searchTotal':
    'มีผลลัพธ์ {total} รายการ',
  'feed.searchNoResults':
    'ไม่พบโพสต์ที่มี " {query} "',
  'feed.loadingPosts':
    'กำลังโหลดโพสต์...',
  'feed.loadMorePosts':
    'โหลดโพสต์เพิ่มเติม...',
  'feed.postsFailed':
    'โหลดโพสต์ไม่สำเร็จ โปรดลองใหม่อีกครั้ง',
  'feed.postsFailedShort':
    'โหลดไม่สำเร็จ โปรดลองใหม่อีกครั้ง',
  'feed.scrollMore':
    'ปัดลงเพื่อโหลดเพิ่มเติม',
  'feed.endOfFeed':
    'ถึงสุดท้ายแล้ว',
  'feed.noPosts':
    'ยังไม่มีโพสต์ โปรดค้างความคิดแรกก่อน',
  'feed.likeFailed':
    'กดถูกใจหรือยกเลิกถูกใจไม่สำเร็จ โปรดลองใหม่อีกครั้ง',
  'post.authorAnonymous':
    'ผู้เขียนลับชื่อ',
  'post.report':
    'รายงานโพสต์',
  'post.imageAlt':
    'ภาพประกอบโพสต์',
  'post.unlike':
    'ยกเลิกถูกใจ',
  'post.like':
    'ถูกใจ',
  'post.reply':
    'ตอบกลับ',
  'comment.loading':
    'กำลังโหลดความคิดเห็น...',
  'comment.none':
    'ยังไม่มีความคิดเห็น',
  'comment.loadFailed':
    'โหลดความคิดเห็นไม่สำเร็จ โปรดลองใหม่อีกครั้ง',
  'comment.placeholder':
    'เขียนความคิดเห็น...',
  'comment.max':
    'สูงสุด 2000 ตัวอักษร',
  'comment.submit':
    'ความคิดเห็น',
  'comment.failed':
    'ส่งความคิดเห็นไม่สำเร็จ โปรดลองใหม่อีกครั้ง',
  'comment.report':
    'รายงาน',
  'comment.more':
    'โหลดความคิดเห็นเพิ่มเติม...',
  'report.reasonPlaceholder':
    'กรุณากรอกเหตุผลรายงาน (สูงสุด 500 ตัวอักษร)',
  'report.note':
    'การรายงานจะถูกส่งให้ผู้จัดการเว็บไซต์',
  'report.formLabel':
    'ช่องกรอกรายงาน',
  'report.submit':
    'ส่งรายงาน',
  'report.failed':
    'รายงานไม่สำเร็จ โปรดลองใหม่อีกครั้ง',
  'report.sent':
    'ส่งรายงานแล้ว ขอบคุณสำหรับการแจ้งเตือน',
  'newPost.avatarYou':
    'ผู้ใช้',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'เพิ่มโพสต์',
  'newPost.loginFirst':
    'ต้องเข้าสู่ระบบด้วย Google ก่อนจึงสามารถโพสต์ได้',
  'newPost.contentPlaceholder':
    'แบ่งปันความคิดของคุณ...',
  'newPost.addImage':
    'เพิ่มภาพ',
  'newPost.emailPrivate':
    'Email ไม่ถูกเผยแพร่',
  'newPost.submit':
    'เผยแพร่โพสต์',
  'newPost.publishing':
    'กำลังเผยแพร่...',
  'newPost.uploading':
    'กำลังอัปโหลดภาพ...',
  'newPost.failed':
    'เผยแพร่ไม่สำเร็จ ให้ลองใหม่ภายหลัง',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'ข้อมูลบุคคล',
  'profile.edit':
    'แก้ไข',
  'profile.loginPrompt':
    'หลังเข้าสู่ระบบ จึงตั้งชื่อเล่นและแนะนำตัวของฟอรัมได้',
  'profile.nicknameLabel':
    'ชื่อเล่นฟอรัม',
  'profile.notSet':
    'ยังไม่ได้ตั้งค่า',
  'profile.notSetBio':
    'ยังไม่ได้ตั้งแนะนำตัว',
  'profile.nicknameInput':
    'ชื่อเล่น',
  'profile.nicknamePlaceholder':
    'กรอกชื่อเล่น',
  'profile.nicknameHint':
    'ชื่อเล่นจะแสดงกับโพสต์ที่เผยแพร่ ขีดจำกัด 30 ตัวอักษร',
  'profile.bioLabel':
    'แนะนำตัว',
  'profile.bioPlaceholder':
    'แนะนำตัว (เลือกใส่)',
  'profile.bioHint':
    'ข้อความสูงสุด 500 ตัวอักษร',
  'profile.updated':
    'อัปเดตข้อมูลบุคคลแล้ว',
  'profile.saving':
    'กำลังบันทึก...',
  'profile.saveFailed':
    'ลำลองค่าไม่สำเร็จ ให้ลองใหม่ภายหลัง',
  'profile.loadFailed':
    'โหลดข้อมูลไม่สำเร็จ ให้ลองใหม่ภายหลัง',
  'profile.followingEntry':
    'รายชื่อที่ฉันติดตาม',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'ข้อมูลบุคคลสาธารณะ',
  'publicProfile.avatar':
    'ภาพโปรไฟล์',
  'publicProfile.loading':
    'กำลังโหลด...',
  'publicProfile.invalidLinkName':
    'ลิงก์ข้อมูลบุคคลสาธารณะไม่ถูกต้อง',
  'publicProfile.invalidLinkBio':
    'เข้าสู่ข้อมูลบุคคลผ่านชื่อผู้เขียนในโพสต์ฟอรัม',
  'publicProfile.notFound':
    'ไม่พบผู้ใช้รายนี้',
  'publicProfile.anonymous':
    'ผู้ใช้อนันมัย',
  'publicProfile.noBio':
    'ผู้ใช้รายนี้ยังไม่ได้ตั้งข้อมูลสาธารณะ',
  'publicProfile.loadFailed':
    'โหลดข้อมูลสาธารณะไม่สำเร็จ',
  'publicProfile.postsLabel':
    'โพสต์',
  'publicProfile.emptyPosts':
    'ผู้ใช้รายนี้ยังไม่ได้โพสต์อะไรเลย',

  'follow.label':
    'ติดตามผู้ใช้นี้',
  'follow.action':
    'ติดตาม',
  'follow.actionDone':
    'กำลังติดตาม',
  'follow.unfollow':
    'เลิกติดตาม',
  'follow.done':
    'คุณกำลังติดตามผู้ใช้นี้แล้ว',
  'follow.failed':
    'ติดตามไม่สำเร็จ ให้ลองใหม่ภายหลัง',

  'following.peopleLabel':
    'ผู้ใช้ที่คุณติดตาม',
  'following.postsLabel':
    'โพสต์ของผู้ใช้ที่คุณติดตาม',
  'following.peopleLoading':
    'กำลังโหลดรายชื่อที่ติดตาม...',
  'following.emptyPeople':
    'คุณยังไม่ได้ติดตามใคร กด «ติดตาม» ที่โพสต์ หรือติดตามจากโปรไฟล์สาธารณะของผู้ใช้',
  'following.emptyPosts':
    'ยังไม่มีผู้ใช้ที่คุณติดตามโพสต์เข้ามา',
  'following.peopleFailed':
    'โหลดรายชื่อที่ติดตามไม่สำเร็จ',
  'following.postsFailed':
    'โหลดโพสต์ของผู้ใช้ที่คุณติดตามไม่สำเร็จ',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'ยินดีกลับมา',
  'login.body':
    '{site} เป็นพื้นที่สำหรับผู้ที่อยากขึ้นลงความคิดเห็น ไม่ต้องทำเอกสารการจดทะเบียน —— ใช้บัญชี Google หนึ่งบัญชีก็เริ่มลงโพสต์ได้',
  'login.browseFirst':
    'เข้าชมหน้าแรกก่อน',
  'admin.skipToMain':
    'ข้ามไปยังเนื้อหาหลัก',
  'admin.railLabel':
    'เมนูดูแลรับผิดชอบ',
  'admin.railBrandAria':
    'หน้าแรก{site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'ฟังก์ชันหลัก',
  'admin.railGovernance':
    'ธรรมาภิบาล',
  'admin.railMode':
    'โหมดผู้ดูแลรับผิดชอบ',
  'admin.railExit':
    'กลับสู่ฟอรัม',
  'admin.topbarMenu':
    'สับเปลี่ยนเมนูดูแลรับผิดชอบ',
  'admin.statusOnline':
    'เชื่อมต่อปกติ',
  'admin.topbarForum':
    'ฟอรัม',
  'admin.logoutFailed':
    'ออกจากระบบไม่สำเร็จ ให้ลองใหม่ภายหลัง',
  'admin.navUsers':
    'จัดการผู้ใช้',
  'admin.navPosts':
    'โพสต์ฟอรัม',
  'admin.navReports':
    'จัดการรายงาน',
  'admin.listLoadFailed':
    'โหลดไม่สำเร็จ',
  'admin.dlgClose':
    'ปิดหน้าต่าง',
  'admin.dlgConfirm':
    'ยืนยัน',
  'admin.dlgSave':
    'ลำลอง',
  'admin.dlgApplyTags':
    'ใช้แท็ก',
  'admin.dlgNoTags':
    'ปัจจุบันไม่มีแท็กให้ใช้ ให้เพิ่มใน「จัดการแท็ก」ด้านล่างก่อน',
  'users.title':
    'จัดการผู้ใช้',
  'users.contentAction':
    'เนื้อหา',
  'users.updateContentFailed':
    'ดำเนินการกับเนื้อหาไม่สำเร็จ',
  'users.updated':
    'อัปเดตเนื้อหาแล้ว',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'ดูสถิติกิจกรรมของผู้ใช้ฟอรัม แท็ก และสถานะบัญชี ขั้นตอนระงับและควบคุมดูแลรายละเอียดแต่ละเนื้อหา',
  'users.refresh':
    'อัปเดตข้อมูล',
  'users.statTotal':
    'จำนวนผู้ใช้ทั้งหมด',
  'users.statActive':
    'เปิดใช้งาน',
  'users.statSuspended':
    'ระงับแล้ว',
  'users.statContent':
    'จำนวนโพสต์รวมกับความคิดเห็น',
  'users.count':
    '{count} บัญชีผู้ใช้',
  'users.tagsCount':
    'แท็ก {count} รายการ',
  'users.loadFailed':
    'โหลดข้อมูลผู้ใช้ไม่สำเร็จ',
  'users.tagsLoadFailed':
    'โหลดข้อมูลแท็กไม่สำเร็จ',
  'users.panelTitle':
    'ผู้ใช้ฟอรัม',
  'users.tagsPanelTitle':
    'แท็กผู้ใช้',
  'users.colUser':
    'ผู้ใช้',
  'users.colTags':
    'แท็ก',
  'users.colStatus':
    'สถานะ',
  'users.colPosts':
    'โพสต์',
  'users.colComments':
    'ความคิดเห็น',
  'users.colLikes':
    'ถูกใจ',
  'users.colLastActivity':
    'กิจกรรมล่าสุด',
  'users.colActions':
    'การดำเนินการ',
  'users.nicknameUnset':
    'ไม่ได้ตั้งชื่อเล่น',
  'users.notSet':
    'ไม่ได้ตั้งค่า',
  'users.statusActive':
    'เปิดใช้งาน',
  'users.statusSuspended':
    'ระงับแล้ว',
  'users.emptyTitle':
    'ปัจจุบันไม่มีข้อมูลผู้ใช้',
  'users.emptyBody':
    'จะไม่มีข้อมูลเมื่อไม่มีผู้ใช้เข้าสู่ฟอรัมผ่าน Google',
  'users.tagsEmptyTitle':
    'ปัจจุบันไม่มีแท็ก',
  'users.tagsEmptyBody':
    'ให้สร้างแท็กก่อน จึงจะใช้กับลิสต์ผู้ใช้ได้',
  'users.addTag':
    'เพิ่มแท็ก',
  'users.colName':
    'ชื่อ',
  'users.colCreated':
    'เวลาสร้าง',
  'users.colUpdated':
    'เวลาอัปเดต',
  'users.renameTag':
    'เปลี่ยนชื่อ',
  'users.suspend':
    'ระงับ',
  'users.restore':
    'กู้คืน',
  'users.statusDialogTitle':
    '{action}ผู้ใช้รายนี้',
  'users.statusSuspendMessage':
    '{email} จะเข้าสู่ฟอรัมไม่ได้อีก ต่อไปนี้โพสต์และความคิดเห็นที่มีอยู่จะถูกเก็บรักษาไว้ ต่อเนื่องหรือไม่',
  'users.statusRestoreMessage':
    '{email} จะได้เข้าสู่ฟอรัมและลงโพสต์ได้อีก ต่อเนื่องหรือไม่',
  'users.userSuspended':
    'ระงับผู้ใช้แล้ว',
  'users.userRestored':
    'ผู้ใช้ถูกกู้คืนแล้ว',
  'users.updateStatusFailed':
    'เปลี่ยนสถานะผู้ใช้ไม่สำเร็จ',
  'users.editTagsTitle':
    'แก้ไขแท็ก · {user}',
  'users.editTagsMessage':
    'เลือกแท็กที่ต้องการใช้งาน; การยกเลือกทั้งหมดเท่ากับลบแท็กทั้งหมดของผู้ใช้คนนี้',
  'users.tagsUpdated':
    'แท็กของผู้ใช้ถูกอัปเดตแล้ว',
  'users.updateTagsFailed':
    'อัปเดตแท็กของผู้ใช้ไม่สำเร็จ',
  'users.contentLoadFailed':
    'โหลดเนื้อหาไม่สำเร็จ',
  'users.contentLoadFailedShort':
    'โหลดเนื้อหาไม่สำเร็จ',
  'users.contentPanelTitle':
    'เนื้อหาผู้ใช้',
  'users.contentCount':
    '{posts} โพสต์ · {comments} ความเห็น',
  'users.contentLoading':
    'กำลังโหลดโพสต์และความเห็น…',
  'users.addPost':
    'เพิ่มโพสต์',
  'users.addComment':
    'เพิ่มความคิดเห็น',
  'users.postsColumn':
    'โพสต์',
  'users.commentsColumn':
    'ความเห็น',
  'users.noPosts':
    'ยังไม่มีโพสต์',
  'users.noComments':
    'ยังไม่มีความเห็น',
  'users.postRef':
    'โพสต์ #{id}',
  'users.editRecordTitle':
    'แก้ไข{kind} #{id}',
  'users.deleteRecordTitle':
    'ลบ{kind} #{id}',
  'users.deleteRecordMessage':
    'การลบแล้วย้อนกลับไม่ได้ ความชอบและข้อมูลที่เชื่อมโยงกันจะถูกลบพร้อมกัน ต้องการดำเนินการต่อไหม',
  'users.contentLabel':
    'เนื้อหา',
  'users.addPostTitle':
    'เพิ่มโพสต์',
  'users.addPostMessage':
    'เนื้อหานี้จะเผยแพร่อัตโนมัติในฐานะผู้ใช้คนดังกล่าว โดยช่องผู้เขียนจะมาจากบัญชีที่ลงชื่เข้าระบบและสามารถปลอมแปลงไม่ได้',
  'users.postContentLabel':
    'เนื้อหาโพสต์',
  'users.postContentPlaceholder':
    'ป้อนเนื้อหาโพสต์',
  'users.pickPostTitle':
    'เลือกโพสต์',
  'users.postIdLabel':
    'ID โพสต์',
  'users.postIdPlaceholder':
    'หมายเลขบทความที่ต้องการแสดงความเห็น',
  'users.addCommentTitle':
    'เพิ่มความเห็น',
  'users.commentContentLabel':
    'เนื้อหาความเห็น',
  'users.commentContentPlaceholder':
    'ป้อนเนื้อหาความเห็น',
  'users.createTagTitle':
    'เพิ่มแท็ก',
  'users.createTagMessage':
    'แท็กสามารถใช้จัดประเภทผู้ใช้ เช่น "อธิการ" "กระตือรือร้น" หรือ "ถูกบล็อก"',
  'users.tagNameLabel':
    'ชื่าทแท็ก',
  'users.tagNamePlaceholder':
    'สูงสุด 50 ตัวอักษร',
  'users.renameTagTitle':
    'เปลี่ยนชื่าทแท็ก',
  'users.renameTagMessage':
    'ผู้ใช้ทั้งหมดที่ใช้แท็กนี้จะเห็นชื่อใหม่',
  'users.deleteTagTitle':
    'ลบแท็ก "{name}"',
  'users.deleteTagMessage':
    'หลังลบแล้ว แท็กนี้ของผู้ใช้ทั้งหมดจะถูกลบพร้อมกันและย้อนกลับไม่ได้ ต้องการดำเนินการต่อไหม',
  'users.deleteTagConfirm':
    'ลบแท็ก',
  'users.tagCreated':
    'แท็กถูกสร้างแล้ว',
  'users.createTagFailed':
    'สร้างแท็กไม่สำเร็จ',
  'users.tagUpdated':
    'แท็กถูกอัปเดตแล้ว',
  'users.updateTagFailed':
    'อัปเดตแท็กไม่สำเร็จ',
  'users.tagDeleted':
    'แท็กถูกลบแล้ว',
  'users.deleteTagFailed':
    'ลบแท็กไม่สำเร็จ',
  'users.refreshDone':
    'อัปเดตเป็นข้อมูลล่าสุดแล้ว',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'หน้าผู้ดูแลระบบ',
  'users.signinBody':
    'การควบคุมเนื้อหาแบบรวมศูนย์ ช่วยให้การตรวจสอบทุกครั้งชัดเจน รวดเร็ว และติดตามผลได้',
  'users.signinStep1':
    'การยืนยันความปลอดภัย',
  'users.signinStep2':
    'การควบคุมผู้ใช้',
  'users.signinStep3':
    'การตรวจสอบเนื้อหา',
  'users.signinPanelTitle':
    'เข้าสู่ Admin Console',
  'users.signinPanelBody':
    'ระบบ Admin Console มีเฉพาะบัญชีผู้ดูแล Google ที่อนุญาตให้เข้าใช้งาน โปรดเข้าสู่ระบบด้วยบัญชีผู้ดูแล',
  'posts.title':
    'ฟอเรียมโพสต์',
  'posts.searching':
    'กำลังค้นหา…',
  'posts.searchDegraded':
    '(ไม่ได้เปิดใช้บริการค้นหา ใช้การเปรียบเทียบคีย์วอร์ดในฐานข้อมูล)',
  'posts.searchSummary':
    'พบ {total} รายการสำหรับ "{query}" หน้านี้แสดง {shown} รายการ',
  'posts.pendingCount':
    '{count} รายการรอการดำเนินการ',
  'posts.pageSummary':
    'หน้า {page} / {pages} หน้า หน้านี้มี {count} รายการ',
  'posts.listLoadFailed':
    'โหลดรายชื่อโพสต์ไม่สำเร็จ',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'สร้าง แก้ไข และลบบริษทฟอเรียม พร้อมควบคุมเนื้อหาที่ระดับความเห็น',
  'posts.toReports':
    'การจัดการรายงาน',
  'posts.editorTitleNew':
    'เพิ่มโพสต์',
  'posts.editorTitleEdit':
    'แก้ไขโพสต์ #{id}',
  'posts.editorNote':
    'เผยแพร่ในฐานะผู้ดูแลระบบ; ผู้เขียนมาจากบัญชีที่เข้าสู่ระบบ ข้อความจากคำขอไม่สามารถปลอมแปลงผู้เขียนได้',
  'posts.cancelEdit':
    'ยกเลิกแก้ไข',
  'posts.contentLabel':
    'เนื้อหาโพสต์',
  'posts.contentPlaceholder':
    'ป้อนเนื้อหาโพสต์',
  'posts.saveChanges':
    'บันทึกการเปลี่ยนแปลง',
  'posts.emptyContent':
    'เนื้อหาโพสต์ไม่สามารถว่างได้',
  'posts.saving':
    'กำลังบันทึก…',
  'posts.saved':
    'อัปเดตโพสต์แล้ว',
  'posts.published':
    'เผยแพร่โพสต์แล้ว',
  'posts.saveFailed':
    'บันทึกไม่สำเร็จ',
  'posts.deleteTitle':
    'ลบโพสต์ #{id}',
  'posts.deleteMessage':
    'การลบแล้วย้อนกลับไม่ได้ ความเห็นทั้งหมดภายใต้โพสต์นี้จะถูกลบพร้อมกัน ต้องการดำเนินการต่อไหม',
  'posts.deleteConfirm':
    'ลบโพสต์',
  'posts.deleted':
    'ลบโพสต์แล้ว',
  'posts.deleteFailed':
    'ลบโพสต์ไม่สำเร็จ',
  'posts.commentUpdated':
    'อัปเดตความเห็นแล้ว',
  'posts.commentActionFailed':
    'ดำเนินการความเห็นไม่สำเร็จ',
  'posts.pickPostTitle':
    'เลือกโพสต์ที่ความเห็นอยู่',
  'posts.pickPostMessage':
    'โดยค่าเริ่มต้นคือโพสต์ในแถวเดียวกัน หากต้องการแขวนกับบทความอื่น ให้เปลี่ยนเป็นหมายเลขบทความที่ถูกต้อง',
  'posts.postIdLabel':
    'ID โพสต์',
  'posts.addCommentAtTitle':
    'เพิ่มความเห็นในโพสต์ #{id}',
  'posts.addCommentMessage':
    'ความเห็นนี้จะถูกเผยแพร่ในฐานะผู้ดูแลระบบ',
  'posts.commentContentLabel':
    'เนื้อหาความเห็น',
  'posts.commentContentPlaceholder':
    'ป้อนเนื้อหาความเห็น',
  'posts.add':
    'เพิ่ม',
  'posts.editCommentTitle':
    'แก้ไขความเห็น #{id}',
  'posts.deleteCommentTitle':
    'ลบความเห็น #{id}',
  'posts.deleteCommentMessage':
    'การลบแล้วย้อนกลับไม่ได้ ต้องการดำเนินการต่อไหม',
  'posts.listTitle':
    'รายชื่อโพสต์',
  'posts.clearSearch':
    'ล้างการค้นหา',
  'posts.searchLabel':
    'ค้นหาแบบคีย์',
  'posts.searchPlaceholder':
    'เนื้อหา POST หรือ Email เต็มรูปของผู้เขียน POST',
  'posts.searchHint':
    'จัดอันดับตามความเกี่ยวข้อง; กรอกรายการ Email เต็มรูปเพื่อหา POST ทั้งหมดที่ผู้ใช้คนนี้สร้าง ค้นหาแทนการสลับหน้า ผลลัพธ์แสดงได้สูงสุด 25 รายการ',
  'posts.searchTotal':
    'ผลการค้นหา {total} รายการ',
  'posts.searchFailed':
    'การค้นหาไม่สำเร็จ',
  'posts.searchStatusFailed':
    'การค้นหาไม่สำเร็จ',
  'posts.colContentImage':
    'เนื้อหาและภาพ',
  'posts.colEngagement':
    'การมีส่วนร่วม',
  'posts.colComments':
    'ความคิดเห็น',
  'posts.colAuthor':
    'ผู้เขียน',
  'posts.imageAlt':
    'ภาพประกอบใน POST',
  'posts.likes':
    '{count} ถูกใจ',
  'posts.author':
    'ผู้เขียน: {name}',
  'posts.emptyTitle':
    'ปัจจุบันไม่มีฟอรัม POST',
  'posts.emptyBody':
    'สามารถใช้อีเว็นด์ด้านบนเพื่อสร้าง POST แรกได้',
  'posts.emptySearchTitle':
    'ไม่พบ POST ที่ตรงกับเงื่อนไข',
  'posts.emptySearchBody':
    'ไม่พบ POST ที่ตรงกับ "{query}" ลองใช้คำค้นหาอื่นดู',
  'posts.commentCount':
    '{count} ความคิดเห็น',
  'posts.noComments':
    'ยังไม่มีความคิดเห็น',
  'posts.reportsTitle':
    'การรายงานที่รอการดำเนินการ',
  'posts.allReports':
    'การรายงานทั้งหมด',
  'posts.colReportedContent':
    'เนื้อหาที่ถูกรายงาน',
  'posts.colReason':
    'เหตุผลรายงาน',
  'posts.colReporter':
    'ผู้รายงาน',
  'posts.colTime':
    'เวลา',
  'posts.colVerdict':
    'คำตัดสิน',
  'posts.emptyReportsTitle':
    'ไม่มีการรายงานที่รอการดำเนินการ',
  'posts.emptyReportsBody':
    'การรายงานทั้งหมดผ่านการดำเนินการแล้ว',
  'posts.verdictResolved':
    'ดำเนินการแล้ว',
  'posts.verdictRejected':
    'ไม่สมบูรณ์',
  'posts.verdictDialogTitle':
    'ทำให้การรายงาน #{id} เป็น “{label}”',
  'posts.verdictDialogMessage':
    'หลังทำเครื่องหมาย การรายงานจะออกจากรายการที่รอการดำเนินการ แต่ข้อมูลยังถูกเก็บไว้ในหน้าจัดการรายงาน ต้องการดำเนินการต่อไหม',
  'posts.verdictConfirm':
    'ตั้งเป็น{label}',
  'posts.verdictDone':
    'การรายงานถูกตั้งสถานะเป็น{label}แล้ว',
  'posts.verdictFailed':
    'การอัปเดตสถานะการรายงานไม่สำเร็จ',
  'reports.title':
    'จัดการรายงาน',
  'reports.listSummary':
    '{count} รายการ · {filter}',
  'reports.listLoadFailed':
    'โหลดรายการรายงานไม่สำเร็จ',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'ตัดสินการรายงานแต่ละรายการ: “ผ่าน” จะลบเนื้อหาที่ถูกรายงาน “ไม่สมบูรณ์” จะเก็บข้อความเดิมไว้ การกลับไปเป็น “รอการดำเนินการ” จะล้างเวลาตัดสินที่มีอยู่',
  'reports.backToPosts':
    'กลับไปยังฟอรัม',
  'reports.editorTitle':
    'แก้ไขการรายงาน #{id}',
  'reports.editorNote':
    'สามารถแก้ไขเหตุผลและสถานะของการรายงาน; เป้าหมายและผู้รายงานเป็นข้อมูลเดิม ไม่ได้เปลี่ยนแปลงในนี่',
  'reports.targetTypeLabel':
    'ประเภทเป้าหมาย',
  'reports.targetIdLabel':
    'ID เป้าหมาย',
  'reports.reporterEmailLabel':
    'Email ของผู้รายงาน',
  'reports.statusLabel':
    'สถานะ',
  'reports.reasonLabel':
    'เหตุผลรายงาน',
  'reports.reasonHint':
    'สูงสุด 500 ตัวอักษร จะแสดงโดยตรงให้ผู้ดูแลอื่นใช้เป็นหลักฐานในการตัดสิน',
  'reports.filterLabel':
    'กรองตามสถานะ',
  'reports.filterAll':
    'ทั้งหมด',
  'reports.listTitle':
    'รายการรายงาน',
  'reports.colTarget':
    'เนื้อหาเป้าหมาย',
  'reports.colReason':
    'เหตุผลรายงาน',
  'reports.colReporter':
    'ผู้รายงาน',
  'reports.colStatus':
    'สถานะ',
  'reports.targetGone':
    '(ลบเนื้อหาแล้ว)',
  'reports.author':
    'ผู้เขียน: {name}',
  'reports.deleteTitle':
    'ลบการรายงาน #{id}',
  'reports.deleteMessage':
    'สิ่งที่ลบคือบันทึกรายงานนี้โดยตรง เนื้อหาที่ถูกรายงานไม่ได้รับผลกระทบและไม่สามารถฟื้นคืนได้ ต้องการดำเนินการต่อไหม',
  'reports.deleteConfirm':
    'ลบการรายงาน',
  'reports.deleted':
    'ลบการรายงานแล้ว',
  'reports.deleteFailed':
    'ลบการรายงานไม่สำเร็จ',
  'reports.updated':
    'อัปเดตการรายงานแล้ว',
  'reports.saveFailed':
    'บันทึกการรายงานไม่สำเร็จ',
  'reports.approveTitle':
    'ผ่านการรายงาน #{id}',
  'reports.approveGoneMessage':
    '{kind} #{id} ที่ถูกรายงานไม่มีอยู่แล้ว การรายงานนี้จะถูกตั้งสถานะว่าดำเนินการแล้วเท่านั้น',
  'reports.approveMessage':
    'จะลบ{kind} #{id} ที่ถูกรายงานอย่างถาวร (หากเป็นบทความ ให้ลบความคิดเห็นทั้งหมดภายใต้บทความนี้พร้อมกัน) และตั้งการรายงานนี้เป็น ดำเนินการแล้ว ต้องการดำเนินการต่อไหม',
  'reports.approveConfirm':
    'ผ่านและลบบทความ',
  'reports.approveGoneDone':
    'เนื้อหาไม่มีอยู่แล้ว การรายงานถูกตั้งสถานะว่าดำเนินการแล้ว',
  'reports.approveDone':
    'ลบบทความและตั้งการรายงานเป็น ดำเนินการแล้ว',
  'reports.approveFailed':
    'ผ่านการรายงานไม่สำเร็จ',
  'reports.approveTitleGone':
    'เนื้อหาถูกลบ แล้วจะตั้งเครื่องหมายการรายงานเท่านั้น',
  'reports.approveTitleFull':
    'ลบเนื้อหาที่ถูกรายงานและตั้งสถานะเป็น ดำเนินการแล้ว',
  'reports.rejectTitle':
    'การรายงาน #{id} ไม่สมบูรณ์',
  'reports.rejectMessage':
    'การไม่สมบูรณ์หมายถึงไม่จำเป็นต้องดำเนินการกับเนื้อหาที่ถูกรายงาน จะเก็บเนื้อหาไว้แบบเดิม ต้องการดำเนินการต่อไหม',
  'reports.rejectConfirm':
    'ตั้งเป็นไม่สมบูรณ์',
  'reports.rejectDone':
    'การรายงานถูกตั้งสถานะเป็นไม่สมบูรณ์แล้ว',
  'reports.statusFailed':
    'การอัปเดตสถานะการรายงานไม่สำเร็จ',
  'reports.emptyTitle':
    'ปัจจุบันไม่มีการรายงาน',
  'reports.emptyBody':
    'ไม่มีบันทึกใด ๆ ภายใต้ตัวกรองนี้',
  'reports.rejectTitleAttr':
    'เก็บเนื้อหาไว้ โดยตั้งการรายงานเป็นไม่สมบูรณ์เท่านั้น',
  'kind.post':
    'บทความ',
  'kind.comment':
    'ความคิดเห็น',
  'reports.statusPending':
    'รอการดำเนินการ',
  'reports.statusResolved':
    'ดำเนินการแล้ว',
  'reports.statusRejected':
    'ไม่สมบูรณ์',
  'title.forum':
    '{site}',
  'title.login':
    'เข้าสู่ระบบ｜{site}',
  'title.newPost':
    'เพิ่ม POST｜{site}',
  'title.profile':
    'โปรไฟล์บุคคล｜{site}',
  'title.publicProfile':
    'โปรไฟล์สาธารณะ｜{site}',
  'title.following':
    'กำลังติดตาม｜{site}',
  'title.adminUsers':
    'จัดการผู้ใช้｜แดชบอร์ด {brand}',
  'title.adminLogin':
    'เข้าสู่ระบบ｜แดชบอร์ด {brand}',
  'title.adminPosts':
    'POST ฟอรัม｜แดชบอร์ด {brand}',
  'title.adminReports':
    'จัดการรายงาน｜แดชบอร์ด {brand}',
};
