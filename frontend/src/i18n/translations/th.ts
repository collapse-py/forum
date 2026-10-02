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
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 ชั่วโมง',
  'common.oneDay': '1 วัน',
  'common.sevenDays': '7 วัน',
  'common.thirtyDays': '30 วัน',
  'common.oneYear': '1 ปี',
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
  'admin.navMonitor': 'การตรวจสอบระบบ',
  'admin.navLog': 'บันทึกการดำเนินการ',
  'admin.navStats': 'แนวโน้มเนื้อหา',
  'admin.navExport': 'ส่งออกและทำหลายรายการพร้อมกัน',
  'admin.navSessions': 'การเข้าสู่ระบบและเซสชัน',
  'admin.navBlocks': 'รายชื่อ IP ที่ถูกบล็อก',
  'admin.navAnnouncements': 'ประกาศ',
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

  'monitor.title': 'การตรวจสอบระบบ',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'ดูสถานะบริการ ปริมาณคำขอ การกระจายตัวของความหน่วง และตัวนับของตัวจำกัดอัตราแบบเรียลไทม์',
  'monitor.refresh': 'รีเฟรช',
  'monitor.refreshing': 'กำลังโหลด…',
  'monitor.autoRefresh': 'รีเฟรชอัตโนมัติ',
  'monitor.autoRefreshOn': 'รีเฟรชอัตโนมัติทุก {seconds} วินาที',
  'monitor.autoRefreshOff': 'หยุดรีเฟรชอัตโนมัติไว้',
  'monitor.nextUpdate': 'อัปเดตในอีก {seconds} วินาที',
  'monitor.loadFailed': 'โหลดข้อมูลการตรวจสอบไม่สำเร็จ',
  'monitor.loadFailedHint': 'ตรวจสอบว่าคุณเข้าสู่ระบบในชื่อผู้ดูแลระบบแล้ว และเซิร์ฟเวอร์ยังทำงานอยู่',
  'monitor.pausedHint': 'รีเฟรชอัตโนมัติถูกหยุดไว้ หน้าจอจึงแสดงผลการอ่านล่าสุดที่สำเร็จ',
  'monitor.visibilityPaused': 'แท็บอยู่เบื้องหลัง จึงหยุดรีเฟรชอัตโนมัติไว้',
  'monitor.lastUpdated': 'อัปเดตเมื่อ {time}',
  'monitor.probeTook': 'การตรวจสอบการพึ่งพาใช้เวลา {ms} มิลลิวินาที',
  'monitor.unreachable': 'เซิร์ฟเวอร์ไม่ตอบสนอง หน้าจอจึงค้างอยู่ที่ผลการอ่านล่าสุดที่สำเร็จ',
  'monitor.depsTitle': 'สถานะบริการ',
  'monitor.depsNote': 'การอ่านแต่ละครั้งจะตรวจสอบการพึ่งพาแต่ละตัวจริงหนึ่งครั้ง เวลาจำกัดต่อตัวคือ 2 วินาที และทั้งสามตัวทำงานพร้อมกัน',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'เครื่องมือค้นหา',
  'monitor.stateOk': 'ปกติ',
  'monitor.stateDown': 'เชื่อมต่อไม่ได้',
  'monitor.stateDisabled': 'ยังไม่ได้เปิดใช้',
  'monitor.depSearchFallback': 'ยังไม่ได้ตั้งค่า ES_URL การค้นหาจึงใช้การเทียบคำสำคัญของ MySQL',
  'monitor.depDisabled': 'ไม่ได้ส่งคลายเน็ต Redis เข้ามา ฟีเจอร์สื่อจึงปิดอยู่',
  'monitor.depLatency': 'ตอบกลับใน {ms} มิลลิวินาที',
  'monitor.depKeys': '{count} คีย์',
  'monitor.depMemory': 'หน่วยความจำ {size}',
  'monitor.depPoolUsage': 'การเชื่อมต่อ {inUse}/{open} (สูงสุด {max})',
  'monitor.depPoolWait': 'รอ {count} ครั้ง รวม {ms} มิลลิวินาที',
  'monitor.depRedisPool': 'พบ {hits} / ไม่พบ {misses}',
  'monitor.depEngineMysql': 'เทียบคำสำคัญของ MySQL',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'ภาพรวมคำขอ',
  'monitor.statUptime': 'เวลาทำงาน',
  'monitor.statRequests': 'คำขอทั้งหมด',
  'monitor.statErrorRate': 'อัตราข้อผิดพลาด',
  'monitor.statP95': 'ความหน่วง P95',
  'monitor.statInFlight': 'กำลังประมวลผล',
  'monitor.statGoroutines': 'Goroutine',
  'monitor.statHeap': 'หน่วยความจำฮีป',
  'monitor.statDbPool': 'การเชื่อมต่อฐานข้อมูล',
  'monitor.statRateLimited': 'ถูกจำกัดอัตรา',
  'monitor.statCountWithPeak': 'สูงสุด {peak}',
  'monitor.statCountWithInUse': 'ใช้งาน {inUse} ว่าง {idle}',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} คอร์ตเชิงตรรกะ · GC {gc} ครั้ง',
  'monitor.noData': 'ยังไม่มีคำขอ',
  'monitor.noDataBody': 'คำขอที่เกิดขึ้นตั้งแต่บริการเริ่มทำงานจะปรากฏที่นี่ ตอนนี้ส่วนนี้ยังว่าง',
  'monitor.timelineTitle': 'ปริมาณการใช้งาน {minutes} นาทีล่าสุด',
  'monitor.timelineNote': 'ยอดรวมนับสะสมตั้งแต่โปรเซสนี้เริ่มทำงานและจะรีเซ็ตเมื่อรีสตาร์ต แต่ละเปอร์เซ็นไทล์คือขอบบนของถังในฮิสโตแกรม จึงได้เฉพาะค่าที่กระโดดไม่ต่อเนื่อง นาทีที่ไม่มีแท่งหมายถึงช่วงนั้นไม่มีการใช้งาน',
  'monitor.timelineLive': 'โปรเซสนี้',
  'monitor.timelineHistory': 'ก่อนรีสตาร์ต',
  'monitor.timelineLegendVolume': 'คำขอ',
  'monitor.timelineLegendError': 'ข้อผิดพลาด 5xx',
  'monitor.timelinePeak': 'สูงสุด {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'ไม่มีข้อมูลสรุปย้อนหลังในฐานข้อมูล หากตอนเริ่มทำงานอ่านไม่ได้ (ไม่มีตารางหรือไม่มีสิทธิ์) จะแสดงเฉพาะโปรเซสปัจจุบัน',
  'monitor.routesTitle': 'ตามเส้นทาง',
  'monitor.routesNote': 'เส้นทางถูกปรับให้เป็นมาตรฐาน (ตัวเลขและอีเมลกลายเป็น :id) ตัว id ต่างกันของเส้นทางเดียวกันจึงถูกนับรวมกัน',
  'monitor.colRoute': 'เส้นทาง',
  'monitor.colCount': 'คำขอ',
  'monitor.colAvg': 'เฉลี่ย',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'ช้าที่สุด',
  'monitor.colErrors': 'ข้อผิดพลาด',
  'monitor.routeOther': 'อื่น ๆ (ถึงจำนวนเส้นทางสูงสุดแล้ว)',
  'monitor.clientsTitle': 'ที่อยู่ต้นทาง',
  'monitor.clientsNote': 'เรียงตามจำนวนคำขอ ที่อยู่ที่ดึงมาจาก X-Forwarded-For หรือ X-Real-IP ไม่ได้ผ่านการตรวจสอบจากพร็อกซีที่เชื่อถือได้ — ตรวจสอบก่อนว่าพร็อกซีของคุณเขียนทับหัวข้อเหล่านี้หรือไม่',
  'monitor.noClients': 'ยังไม่มีแหล่งที่ติดตาม',
  'monitor.noClientsBody': 'ทุกคำขอจะถูกบันทึกตามที่อยู่ต้นทาง ตอนนี้ตารางนี้ยังว่างอยู่',
  'monitor.clientsDropped': 'จำนวนแหล่งทางถึงขีดจำกัด {limit} แล้ว ที่อยู่ {count} รายที่ไม่ถูกพบมานานที่สุดถูกนำออกจากการติดตาม บรรทัดนี้หมายความว่ารายการไม่ครบถ้วน ไม่ใช่ว่ามีคนมาแค่จำนวนนี้',
  'monitor.colIp': 'ที่อยู่',
  'monitor.colSource': 'ที่มา',
  'monitor.colRateLimited': 'ถูกจำกัดอัตรา',
  'monitor.colBanned': 'ถูกบล็อกจนปฏิเสธ',
  'monitor.colLastRoute': 'เข้าถึงล่าสุด',
  'monitor.colActions': 'การดำเนินการ',
  'monitor.colBlock': 'บล็อก',
  'monitor.blocking': 'กำลังบล็อก…',
  'monitor.sourcePeer': 'ปลายทางการเชื่อมต่อ',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.trustLegacy': 'ยังไม่ได้ตั้ง TRUSTED_PROXY_CIDRS: เซิร์ฟเวอร์ยังใช้พฤติกรรมเดิมโดยให้ความสำคัญกับค่าซ้ายสุดของ X-Forwarded-For เมื่อกว่าจะยืนยันว่ามีพร็อกซีอยู่ข้างหน้าที่เขียนทับหัวข้อเหล่านี้จริง และผู้ใช้ข้ามมันไม่ได้ ข้อจำกัดอัตราและการบล็อก IP จะถูกข้ามได้ด้วยหัวข้อปลอมเพียงหนึ่งรายการ และที่อยู่ต้นทางในบันทึกการตรวจสอบก็ไม่ใช่หลักฐาน',
  'monitor.trustConfigured': 'ตั้งค่าพร็อกซีที่เชื่อถือแล้ว: จะรับ X-Forwarded-For / X-Real-IP เฉพาะเมื่อปลายทางการเชื่อมต่ออยู่ในช่วงเครือข่ายเหล่านี้เท่านั้น มิฉะนั้นจะใช้ที่อยู่ปลายทาง ช่วงที่มีผล: {cidrs}',
  'monitor.trustBroken': 'ประกาศ TRUSTED_PROXY_CIDRS แล้ว แต่ไม่มีรายการใดอ่านเป็นช่วง CIDR ได้ ({declared}) จึงยังคงใช้พฤติกรรมเดิมตอนไม่ได้ตั้งค่า',
  'monitor.trustPartial': 'รายการต่อไปนี้ของ TRUSTED_PROXY_CIDRS แปลงเป็นช่วง CIDR ไม่ได้ ({invalid}) จึงไม่เคยเชื่อถือส่วนหัวที่ถูกส่งต่อมาจากช่วงเหล่านี้ คำขอที่มาจากช่วงเหล่านี้จะถูกจัดกลุ่มตามที่อยู่ปลายทางการเชื่อมต่อ กล่าวคือใช้โควตาจำกัดอัตราและการค้นหารายชื่อที่ถูกบล็อกชุดเดียวกัน',
  'monitor.blockTitle': 'บล็อก {ip}',
  'monitor.blockMessage': 'คำขอที่เขียนข้อมูลจากที่อยู่นี้ (โพสต์ ความคิดเห็น ถูกใจ รายงาน อัปโหลดรูป และการเปลี่ยนเส้นทางเข้าสู่ระบบ) จะถูกปฏิเสธเป็นเวลา {duration} การอ่านไม่ได้รับผลกระทบ ต้องการบล็อกหรือไม่',
  'monitor.blockReason': 'บล็อกจากหน้าติดตาม',
  'monitor.blocked': 'บล็อก {ip} แล้ว',
  'monitor.blockFailed': 'การบล็อกล้มเหลว',
  'monitor.limitsTitle': 'ตัวจำกัดอัตรา',
  'monitor.limitsNote': 'แต่ละกลุ่มมีโควตาของตัวเองตามต้นทุนของ endpoint ที่มันดูแล จำนวนที่ถูกบล็อกคือจำนวนการตอบ 429 ทั้งหมดตั้งแต่โปรเซสนี้เริ่มทำงาน',
  'monitor.colLimiter': 'ตัวจำกัด',
  'monitor.colBudget': 'โควตา',
  'monitor.colAllowed': 'อนุญาต',
  'monitor.colBlocked': 'บล็อก',
  'monitor.colTracked': 'แหล่งที่กำลังติดตาม',
  'monitor.colBlockedRate': 'อัตราการบล็อก',
  'monitor.limitContent': 'การเขียนเนื้อหา',
  'monitor.limitUpload': 'การอัปโหลดรูปภาพ',
  'monitor.limitAuth': 'เข้าสู่ระบบ OAuth',
  'monitor.limitBudget': '{limit} ครั้งต่อ {window} วินาที',
  'monitor.limitUnknown': '(ไม่ทราบ)',
  'monitor.noLimits': 'ไม่มีตัวจำกัดอัตรา',
  'monitor.noLimitsBody': 'ตัวจำกัดอัตรายังไม่ถูกสร้าง จึงไม่มีตัวนับให้ใช้',

  'log.title': 'บันทึกการดำเนินการ',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'ตรวจสอบการเปลี่ยนแปลงทุกรายการในหน้าผู้ดูแล: ใคร เมื่อไร กับวัตถุใด และฟิลด์ใดเปลี่ยนจากอะไรเป็นอะไร',
  'log.refresh': 'รีเฟรช',
  'log.loadFailed': 'โหลดบันทึกการดำเนินการไม่สำเร็จ',
  'log.empty': 'ไม่มีรายการที่ตรงกับตัวกรอง',
  'log.emptyBody': 'ลองขยายตัวกรอง หรือยืนยันว่าช่วงเวลานี้ไม่มีการดำเนินการจริง',
  'log.count': 'รายการที่ {from}–{to} จาก {total}',
  'log.retention': 'เก็บบันทึกไว้ {days} วัน แล้วลบด้วยงานเบื้องหลัง หน้านี้ไม่มีปุ่มลบบันทึก — บันทึกการตรวจสอบที่ลบร่องรอยตัวเองได้ ไม่ถือเป็นบันทึกการตรวจสอบ',
  'log.filterActor': 'ผู้ดำเนินการ',
  'log.filterAction': 'การกระทำ',
  'log.filterTargetType': 'ชนิดทรัพยากร',
  'log.filterFrom': 'ตั้งแต่',
  'log.filterTo': 'ถึง',
  'log.filterAll': 'ทั้งหมด',
  'log.filterApply': 'ใช้ตัวกรอง',
  'log.filterReset': 'ล้างตัวกรอง',
  'log.filterTargetHint': 'คลิกที่วัตถุในแถวใดก็ได้เพื่อดูเฉพาะการกระทำที่เกี่ยวข้อง',
  'log.colTime': 'เวลา',
  'log.colActor': 'ผู้ดำเนินการ',
  'log.colAction': 'การกระทำ',
  'log.colTarget': 'วัตถุ',
  'log.colChanges': 'การเปลี่ยนแปลง',
  'log.colOrigin': 'ที่มา',
  'log.noChanges': '(ไม่มีการเปลี่ยนฟิลด์)',
  'log.changedTo': 'เปลี่ยนเป็น',
  'log.removed': '(ลบแล้ว)',
  'log.created': '(สร้างใหม่)',
  'log.requestId': 'request {id}',
  'log.page': 'หน้า {page}',
  'log.targetUser': 'ผู้ใช้',
  'log.targetPost': 'กระทู้',
  'log.targetComment': 'ความคิดเห็น',
  'log.targetReport': 'การรายงาน',
  'log.targetTag': 'แท็ก',
  'log.targetSystem': 'ระบบ',
  'log.actionUserSuspend': 'ระงับสิทธิ์',
  'log.actionUserReinstate': 'คืนสิทธิ์',
  'log.actionUserTags': 'เปลี่ยนแท็ก',
  'log.actionUserPost': 'ลงโพสต์แทนผู้ใช้',
  'log.actionUserComment': 'แสดงความคิดเห็นแทนผู้ใช้',
  'log.actionUserContent': 'ลบเนื้อหาของเขา',
  'log.actionPostCreate': 'สร้างกระทู้',
  'log.actionPostUpdate': 'แก้ไขกระทู้',
  'log.actionPostDelete': 'ลบกระทู้',
  'log.actionCommentCreate': 'สร้างความคิดเห็น',
  'log.actionCommentUpdate': 'แก้ไขความคิดเห็น',
  'log.actionCommentDelete': 'ลบความคิดเห็น',
  'log.actionReportCreate': 'สร้างการรายงาน',
  'log.actionReportResolve': 'รับการรายงาน',
  'log.actionReportReject': 'ไม่รับการรายงาน',
  'log.actionReportUpdate': 'แก้ไขการรายงาน',
  'log.actionReportDelete': 'ลบการรายงาน',
  'log.actionTagCreate': 'เพิ่มแท็ก',
  'log.actionTagUpdate': 'เปลี่ยนชื่อแท็ก',
  'log.actionTagDelete': 'ลบแท็ก',
  'log.fieldStatus': 'สถานะบัญชี',
  'log.fieldContent': 'เนื้อหา',
  'log.fieldName': 'ชื่อ',
  'log.fieldTags': 'แท็ก',
  'log.fieldReason': 'เหตุผล',
  'log.fieldAuthorEmail': 'ผู้เขียน',
  'log.fieldReporterEmail': 'ผู้รายงาน',
  'log.fieldTargetType': 'ชนิดทรัพยากร',
  'log.fieldTargetId': 'รหัสทรัพยากร',
  'log.fieldPostId': 'รหัสกระทู้',
  'log.fieldCommentId': 'รหัสความคิดเห็น',
  'log.fieldPostIdShort': 'กระทู้',
  'log.fieldCommentIdShort': 'ความคิดเห็น',
  'log.fieldTarget': 'วัตถุ',
  'log.fieldAssignmentsRemoved': 'จำนวนที่เลิกผูก',
  'log.truncated': 'ถูกตัด',

  'stats.title': 'แนวโน้มเนื้อหา',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'แต่ละวันมีผู้ใช้ กระทู้ และความคิดเห็นใหม่กี่รายการ และตอนนี้กระทู้ แท็ก และผู้เขียนรายใดคึกคักที่สุด',
  'stats.refresh': 'รีเฟรช',
  'stats.loadFailed': 'โหลดสถิติเนื้อหาไม่สำเร็จ',
  'stats.window': 'แสดงช่วง',
  'stats.windowDays': '{days} วันล่าสุด',
  'stats.windowClamped': '(ไม่เกิน 90 วัน)',
  'stats.windowNote': 'แต่ละวันถูกแบ่งตามเขตเวลาของเครื่องเซิร์ฟเวอร์ หากเว็บรันบน UTC แต่ผู้ดูแลอยู่คนละเขตเวลา ตัวเลขของวันนี้จะดูน้อยกว่าจริง นั่นคือผลของเขตเวลา ไม่ใช่ทราฟฟิกที่ลดลง',
  'stats.generatedAt': 'สร้างสถิติเมื่อ {time}',
  'stats.seriesTitle': 'รายการใหม่ต่อวัน',
  'stats.seriesNote': 'สามเส้นนี้มีสเกลคนละเรื่องกัน จึงแสดงแยกกันแทนที่จะซ้อนทับ',
  'stats.seriesUsers': 'ผู้ใช้ใหม่',
  'stats.seriesPosts': 'กระทู้ใหม่',
  'stats.seriesComments': 'ความคิดเห็นใหม่',
  'stats.seriesEmpty': 'ไม่มีข้อมูลในช่วงนี้',
  'stats.totalsTitle': 'ยอดรวมในช่วงนี้',
  'stats.totalsNote': 'นี่คือจำนวนที่เพิ่มเข้ามาในช่วงนี้ ไม่ใช่ยอดรวมปัจจุบันของเว็บ',
  'stats.totalUsers': 'ผู้ใช้ใหม่',
  'stats.totalPosts': 'กระทู้ใหม่',
  'stats.totalComments': 'ความคิดเห็นใหม่',
  'stats.totalLikes': 'กดถูกใจใหม่',
  'stats.topPostsTitle': 'กระทู้ยอดนิยม',
  'stats.topPostsNote': 'เรียงตามจำนวนความคิดเห็นและกดถูกใจ นับเฉพาะกระทู้ที่เผยแพร่ในช่วงนี้',
  'stats.topTagsTitle': 'แท็กยอดนิยม',
  'stats.topTagsNote': 'เรียงตามจำนวนผู้ที่ได้รับแท็ก ไม่จำกัดช่วงเวลา แท็กคือคุณสมบัติ ไม่ใช่เหตุการณ์',
  'stats.topAuthorsTitle': 'ผู้เขียนที่เคลื่อนไหว',
  'stats.topAuthorsNote': 'เรียงตามจำนวนกระทู้ในช่วงนี้ แสดงจำนวนความคิดเห็นแยกต่างหาก',
  'stats.colExcerpt': 'เนื้อหาย่อ',
  'stats.colEngagement': 'การโต้ตอบ',
  'stats.colPosts': 'กระทู้',
  'stats.colComments': 'ความคิดเห็น',
  'stats.colUsers': 'จำนวนคน',
  'stats.colAuthor': 'ผู้เขียน',
  'stats.empty': 'ไม่มีข้อมูลในช่วงนี้',
  'stats.emptyBody': 'ลองขยายช่วงเวลา หรือยืนยันว่าช่วงนี้ไม่มีเนื้อหาใหม่จริง',
  'stats.engagement': 'ความคิดเห็น {comments}・ถูกใจ {likes}',
  'stats.rank': 'อันดับที่ {rank}',

  'export.title': 'ส่งออกและทำหลายรายการพร้อมกัน',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'ส่งออกข้อมูลในเว็บเป็น CSV เพื่อกระทบยอด หรือทำกับหลายบัญชีพร้อมกัน',
  'export.download': 'ดาวน์โหลด CSV',
  'export.downloading': 'กำลังเตรียม…',
  'export.exportTitle': 'ส่งออก',
  'export.exportNote': 'การส่งออกแต่ละรายการมีไม่เกิน 5 หมื่นแถว เกินจากนั้นจะมีเฉพาะแถวใหม่ที่สุด คอลัมน์ของทั้งสามรายการตรงกับคอลัมน์บนหน้าผู้ดูแล จึงกระทบยอดได้โดยตรง',
  'export.exportUsers': 'รายชื่อผู้ใช้',
  'export.exportUsersNote': 'อีเมล สถานะ เวลาที่สร้างและแก้ไข จำนวนกระทู้และความคิดเห็น',
  'export.exportPosts': 'รายการกระทู้',
  'export.exportPostsNote': 'หมายเลข ผู้เขียน เนื้อหา 200 ตัวแรก เวลาที่สร้าง ความคิดเห็นและกดถูกใจ',
  'export.exportReports': 'รายการรายงาน',
  'export.exportReportsNote': 'หมายเลข เป้าหมายที่ถูกรายงาน ผู้รายงาน เหตุผล สถานะ และประวัติการตรวจสอบ',
  'export.safety': 'ไฟล์ขึ้นต้นด้วย BOM ของ UTF-8 จึงเปิดใน Excel แล้วไม่เพี้ยน',
  'export.safetyPrefix': 'ค่าที่ขึ้นต้นด้วย = + - @ หรือช่องว่างที่มองไม่เห็น จะถูกเติมเครื่องหมายคำพูดนำหน้า นั่นคือสิ่งที่ทำให้โปรแกรมตารางถือเป็นข้อความแทนที่จะรันเป็นสูตร การเติมนี้ตั้งใจไว้ ไม่ควรขอให้เอาออก',
  'export.batchTitle': 'ทำหลายรายการพร้อมกัน',
  'export.batchNote': 'ปุ่มจะทำงานหลังคุณเลือกบัญชีในหน้าผู้ใช้ การทำหลายรายการจะมีผลครบทั้งชุดหรือไม่มีผลเลย ไม่มีผลล��พอ',
  'export.batchSuspend': 'ระงับที่เลือก',
  'export.batchReinstate': 'คืนสถานะที่เลือก',
  'export.batchTags': 'ตั้งแท็กเป็นชุด',
  'export.batchTagsNote': 'เป็นการเขียนทับ รายการที่ส่งมาคือผลลัพธ์ การส่งรายการว่างเท่ากับลบแท็กทั้งหมด',
  'export.batchConfirm': 'จะใช้ {action} กับ {count} บัญชีหรือไม่',
  'export.batchConfirmTags': 'จะเขียนแท็กของ {count} บัญชีทับด้วย {tags} หรือไม่',
  'export.batchTagsPicker': 'เลือกแท็ก',
  'export.batchTagsNone': 'ไม่เลือกแท็ก (ลบทั้งหมด)',
  'export.batchRunning': 'กำลังทำงาน…',
  'export.batchDone': 'อัปเดตแล้ว {updated} บัญชี',
  'export.batchDoneUnchanged': 'ในจำนวนนั้น {unchanged} บัญชีอยู่ในสถานะเป้าหมายอยู่แล้วจึงไม่เปลี่ยน',
  'export.batchSkipped': 'ข้าม {count}',
  'export.batchMax': 'ครั้งละไม่เกิน 200 บัญชี',
  'export.gotoUsers': 'ไปที่ผู้ใช้',
  'export.noSelection': 'เลือกบัญชีในหน้าผู้ใช้ก่อน',
  'export.selected': 'เลือกแล้ว {count} บัญชี',
  'export.clearSelection': 'ล้างการเลือก',
  'export.selectionHint': 'การเลือกอยู่เฉพาะหน้านี้และจะหายไปเมื่อปิดหน้า',

  'session.title': 'การเข้าสู่ระบบและเซสชัน',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'ดูว่าการเข้าสู่ระบบใดยังใช้ได้อยู่ และบังคับให้บัญชีออกจากทุกอุปกรณ์',
  'session.refresh': 'รีเฟรช',
  'session.loadFailed': 'โหลดรายการเซสชันไม่สำเร็จ',
  'session.privacyTitle': 'ทำไมไม่แสดงโทเคนทั้งหมด',
  'session.privacyNote': 'โทเคนคือข้อมูลเข้าสู่ระบบโดยตรง เราแสดงเพียงแปดตัวอักษรแรก เพื่อให้ผู้ดูแลแยกได้ว่าสองแถวคือเซสชันเดียวกัน และเท่านั้นไม่พอให้ใครเข้าสู่ระบบแทนผู้ใช้ได้ แม้แต่คนที่ได้ภาพหน้าจอนี้ไป นี่เป็นข้อจำกัดโดยเจตนา ไม่ใช่ฟีเจอร์ที่ยังทำไม่เสร็จ',
  'session.expireNote': 'เซสชันจะหมดอายุหลังจากไม่มีกิจกรรมต่อเนื่อง {hours} ชั่วโมง การใช้งานใด ๆ จะต่ออายุให้อัตโนมัติ',
  'session.filterEmail': 'กรองตามบัญชี',
  'session.filterPlaceholder': 'ที่อยู่อีเมลแบบเต็ม',
  'session.search': 'ค้นหา',
  'session.clearFilter': 'ล้าง',
  'session.summary': 'ทั้งเว็บมี {total} เซสชัน ตรวจสอบ {scanned} คีย์',
  'session.truncated': 'การสแกนชนเพดานที่ {scanned} คีย์แล้วจบก่อนเวลา รายการนี้จึงไม่ครบถ้วน',
  'session.empty': 'ขณะนี้ไม่มีเซสชัน',
  'session.emptyBody': 'ไม่มีใครเข้าสู่ระบบ หรือทุกเซสชันหมดอายุแล้ว',
  'session.colUser': 'บัญชี',
  'session.colToken': 'เซสชัน',
  'session.colCreated': 'สร้างเมื่อ',
  'session.colExpires': 'หมดอายุ',
  'session.colRemaining': 'เหลือ',
  'session.colActions': 'การกระทำ',
  'session.unknown': 'ไม่ทราบ',
  'session.adminBadge': 'ผู้ดูแล',
  'session.revoke': 'บังคับออกจากระบบ',
  'session.revokeTitle': 'บังคับออกจากระบบ {email}',
  'session.revokeMessage': 'เซสชันปัจจุบันของบัญชีนี้จำนวน {count} รายการจะหมดอายุทันที และสถานะการเข้าสู่ระบบจะถูกล้างบนทุกอุปกรณ์ ผู้ใช้ต้องเข้าสู่ระบบใหม่ ต้องการดำเนินการต่อหรือไม่',
  'session.revokeRunning': 'กำลังเพิกถอน…',
  'session.revokeDone': 'เพิกถอนแล้ว {count} เซสชัน',
  'session.revokeNone': 'บัญชีนี้ไม่มีเซสชันที่ใช้งานอยู่',
  'session.revokeFailed': 'ยืนยันการออกจากระบบไม่ได้',
  'session.revokeUnavailable': 'ไม่ได้แปลว่าการเพิกถอนล้มเหลว แต่หมายความว่าการสแกนชนเพดานจำนวนคีย์ บางเซสชันจึงอาจไม่ได้ถูกสแกน ลองใหม่อีกครั้งในอีกสักครู่',
  'session.titleColumnNote': 'เป็นเพียงคำนำหน้าเพื่อระบุ ใช้เข้าสู่ระบบไม่ได้',

  'block.title': 'รายชื่อ IP ที่ถูกบล็อก',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'ใส่ที่อยู่ต้นทางที่ยืนยันแล้วว่ามีการละเมิดลงในรายชื่อ รายการนี้อยู่ใน Redis จึงไม่ถูกล้างทั้งการรีสตาร์ตหรือการ deploy',
  'block.refresh': 'รีเฟรช',
  'block.loadFailed': 'โหลดรายชื่อที่ถูกบล็อกไม่สำเร็จ',
  'block.unavailable': 'เว็บนี้ไม่ได้เชื่อมต่อ Redis ฟีเจอร์การบล็อกจึงยังไม่ทำงาน',
  'block.unavailableNote': 'รายชื่อนี้ต้องใช้ Redis ตัวเดียวกับ session และโทเคนสื่อ เมื่อเชื่อมต่อแล้วหน้านี้จะเริ่มทำงาน; ก่อนหน้านั้นการป้องกันมีเพียงการจำกัดอัตราคำขอ (อยู่ในโปรเซส หายเมื่อรีสตาร์ต)',
  'block.add': 'บล็อก',
  'block.addTitle': 'บล็อกที่อยู่ IP',
  'block.addMessage': 'ที่อยู่ที่ถูกบล็อกจะถูกปฏิเสธทุกคำขอที่เขียนข้อมูล (โพสต์ ความคิดเห็น ถูกใจ รายงาน อัปโหลดรูป และการเปลี่ยนเส้นทางเข้าสู่ระบบ) จนกว่าจะหมดเวลา การอ่านไม่ได้รับผลกระทบ',
  'block.ipLabel': 'ที่อยู่ IP',
  'block.ipPlaceholder': '203.0.113.9 หรือ 2001:db8::1',
  'block.durationLabel': 'ระยะเวลา',
  'block.reasonLabel': 'เหตุผล',
  'block.reasonPlaceholder': 'เหตุใดจึงบล็อกที่อยู่นี้ (จะบันทึกไว้ในบันทึกการตรวจสอบ)',
  'block.reasonHint': 'เหตุผลจะบันทึกลงบันทึกการตรวจสอบเท่านั้น ไม่แสดงแก่ผู้ถูกบล็อก และไม่ปรากฏในข้อความแ��้วของสาธารณะ',
  'block.blocking': 'กำลังบล็อก…',
  'block.done': 'บล็อก {ip} แล้ว',
  'block.removed': 'ปลดบล็อก {ip} แล้ว',
  'block.removedNone': '{ip} ไม่ได้ถูกบล็อกอยู่แล้ว',
  'block.failed': 'การบล็อกล้มเหลว',
  'block.unavailableService': 'ใช้รายชื่อที่ถูกบล็อกไม่ได้ (ไม่มี Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'หมดอายุ',
  'block.colRemaining': 'เหลือ',
  'block.colActions': 'การกระทำ',
  'block.unblock': 'ปลดบล็อก',
  'block.unblockTitle': 'ปลดบล็อก {ip}',
  'block.unblockMessage': 'ที่อยู่นี้จะได้สิทธิ์เขียนข้อมูลตามปกติทันที ต้องการดำเนินการต่อหรือไม่',
  'block.empty': 'รายชื่อที่ถูกบล็อกยังว่างอยู่',
  'block.emptyBody': 'ไม่มีที่อยู่ใดถูกบล็อก นี่คือสถานะปกติ: การบล็อกเป็นการตัดสินใจของผู้ดูแลเสมอ และระบบจะไม่บล็อกใครโดยอัตโนมัติ',
  'block.notAutoNote': 'รายชื่อนี้จะไม่เต็มเอง ที่อยู่ที่เกินเพดานจะได้รับเพียง 429 และจะไม่ถูกเพิ่มเข้ามาโดยอัตโนมัติ เพราะทางออกเดียวกันอาจเป็นทั้งสำนักงานหรือทั้ง NAT และการบล็อกอัตโนมัติจะไปโดนพวกเขาด้วย',
  'block.scopeNoteLabel': 'ขอบเขต',
  'block.notAutoNoteLabel': 'ไม่บล็อกอัตโนมัติ',
  'block.maxNoteLabel': 'ความยาวสูงสุด',
  'block.scopeNote': 'การบล็อกหยุดเฉพาะคำขอที่เขียนข้อมูล การอ่านกระทู้ ความคิดเห็น และไฟล์คงที่ยังทำได้ และผู้ถูกบล็อกยังเข้าสู่ระบบและดูเนื้อหาได้: เป็นการตั้งใจ เพราะจุดปลายทางแบบอ่านถูกตั้งใจไม่จำกัดอัตรา (ไม่เช่นนั้นผู้มาเยือนที่ไม่ล็อกอินจะใช้งานไม่ได้) และการบล็อกครอบคลุมชุดเดียวกัน',
  'block.maxNote': 'การบล็อกหนึ่งครั้งอยู่ได้ไม่เกิน 365 วัน เวลาที่นานกว่านี้จะถูกตัดให้เหลือหนึ่งปี เพราะเวลาหมดอายุถูกเก็บเป็นตัวเลข คำว่า “ถาวร” จะกลายเป็นการบล็อกที่ไม่มีใครจำได้และไม่มีวันปลดเอง',
  'block.count': 'บล็อกอยู่ {count} รายการ',
  'block.ipInvalid': 'รูปแบบ IP ไม่ถูกต้อง กรุณาใส่ที่อยู่ IPv4 หรือ IPv6 ไม่รองรับช่วง CIDR',
  'announce.label': 'ประกาศของเว็บไซต์',
  'announce.publicNote': 'ประกาศ',
  'announce.closeAria': 'ปิดประกาศนี้',
  'announce.publishedOn': 'ประกาศเมื่อ {date}',
  'announce.expiresOn': 'หมดอายุเมื่อ {date}',
  'announce.neverExpires': 'ไม่มีวันหมดอายุ',
  'announce.pinnedBadge': 'ปักหมุด',
  'announce.title': 'ประกาศ',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'แสดงประกาศหนึ่งรายการไว้ด้านบนของทุกหน้าในเว็บไซต์ มีผลใช้งานเพียงรายการเดียวในเวลาเดียวกัน การประกาศรายการใหม่จะปิดรายการเดิมโดยอัตโนมัติ',
  'announce.refresh': 'รีเฟรช',
  'announce.loadFailed': 'โหลดรายการประกาศไม่สำเร็จ',
  'announce.new': 'ประกาศรายการใหม่',
  'announce.edit': 'แก้ไขประกาศนี้',
  'announce.deactivate': 'ปิดใช้งาน',
  'announce.reactivate': 'เปิดใช้งานอีกครั้ง',
  'announce.deleteNote': 'ประกาศจะไม่ถูกลบ มีแต่การปิดใช้งาน — การเก็บประวัติไว้คือคำตอบว่าประกาศนี้ถูกเผยแพร่เมื่อไรและโดยใคร',
  'announce.bodyLabel': 'ข้อความประกาศ',
  'announce.bodyPlaceholder': 'เช่น ระบบจะปิดปรับปรุงวันพฤหัสบดี เวลา 02:00–04:00 น.',
  'announce.bodyHint': 'ไม่เกิน 300 ตัวอักษร ข้อความธรรมดาและคงการขึ้นบรรทัดไว้',
  'announce.activeLabel': 'แสดงทันที',
  'announce.expiryLabel': 'ระยะเวลา',
  'announce.expiryNever': 'ไม่มีวันหมดอายุอัตโนมัติ',
  'announce.expiryHours': 'อีก {hours} ชั่วโมง',
  'announce.expiryDays': 'อีก {days} วัน',
  'announce.saving': 'กำลังบันทึก…',
  'announce.published': 'ประกาศแล้ว',
  'announce.updated': 'อัปเดตประกาศแล้ว',
  'announce.deactivated': 'ปิดใช้งานประกาศแล้ว',
  'announce.reactivated': 'เปิดใช้งานประกาศแล้ว',
  'announce.saveFailed': 'การดำเนินการประกาศล้มเหลว',
  'announce.empty': 'ยังไม่มีประกาศ',
  'announce.emptyBody': 'ทันทีที่คุณประกาศ มันจะปรากฏที่ด้านบนของหน้าของผู้เยี่ยมชมทุกคน',
  'announce.colBody': 'ข้อความ',
  'announce.colState': 'สถานะ',
  'announce.colAuthor': 'ผู้ประกาศ',
  'announce.colCreated': 'เวลาประกาศ',
  'announce.colActions': 'การกระทำ',
  'announce.stateActive': 'กำลังแสดง',
  'announce.stateInactive': 'ปิดใช้งานแล้ว',
  'announce.stateExpired': 'หมดอายุแล้ว',
  'announce.confirmDeactivate': 'การประกาศจะปิดใช้งานรายการปัจจุบัน และผู้เยี่ยมชมทุกคนจะเห็นข้อความใหม่ทันที ต้องการดำเนินการต่อหรือไม่',
  'announce.confirmEdit': 'แก้ไขข้อความหรือระยะเวลาของประกาศนี้หรือไม่',
  'announce.count': 'ทั้งหมด {count}',
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
  'posts.pin': 'ปักหมุด',
  'posts.unpin': 'เอาการปักหมุดออก',
  'posts.pinTitle': 'ปักหมุดกระทู้นี้',
  'posts.unpinTitle': 'เอาการปักหมุดกระทู้นี้ออก',
  'posts.pinMessage': 'เมื่อปักหมุดแล้ว กระทู้นี้จะอยู่บนสุดของฟีดของทุกคน และกระทู้ใหม่จะดันมันลงไปไม่ได้',
  'posts.unpinMessage': 'เมื่อเอาการปักหมุดออก กระทู้นี้จะกลับไปอยู่ตามลำดับเวลา',
  'posts.pinDone': 'ปักหมุดแล้ว',
  'posts.unpinDone': 'เอาการปักหมุดออกแล้ว',
  'posts.pinFailed': 'การปักหมุดล้มเหลว',
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
  'title.adminMonitor': 'การตรวจสอบระบบ｜{brand} ผู้ดูแล',
  'title.adminLog': 'บันทึกการดำเนินการ｜{brand} ผู้ดูแล',
  'title.adminStats': 'แนวโน้มเนื้อหา｜{brand} ผู้ดูแล',
  'title.adminExport': 'ส่งออกและทำหลายรายการพร้อมกัน｜{brand} ผู้ดูแล',
  'title.adminSessions': 'การเข้าสู่ระบบและเซสชัน｜{brand} ผู้ดูแล',
  'title.adminBlocks': 'รายชื่อ IP ที่ถูกบล็อก｜{brand} ผู้ดูแล',
  'title.adminAnnouncements': 'ประกาศ｜{brand} ผู้ดูแล',
};
