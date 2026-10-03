/*
 * id catalog (src/i18n/translations/id.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const id: Record<MessageKey, string> = {
  'common.cancel':
    'Batal',
  'common.save':
    'Simpan',
  'common.submitting':
    'Sedang mengirim...',
  'common.delete':
    'Hapus',
  'common.edit':
    'Ubah',
  'common.search':
    'Cari',
  'common.loading':
    'Sedang memuat...',
  'common.loadFailed':
    'Gagal memuat',
  'common.refresh':
    'Muat ulang',
  'common.nextStep':
    'Langkah berikutnya',
  'common.prevPage':
    'Halaman sebelumnya',
  'common.nextPage':
    'Halaman berikutnya',
  'common.create':
    'Buat',
  'common.publish':
    'Terbitkan',
  'common.placeholder':
    '—',
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 jam',
  'common.oneDay': '1 hari',
  'common.sevenDays': '7 hari',
  'common.thirtyDays': '30 hari',
  'common.oneYear': '1 tahun',
  'common.backToHome':
    'Kembali ke beranda',
  'common.backToForumHome':
    'Kembali ke beranda forum',
  'common.backOnePage':
    'Kembali ke halaman sebelumnya',
  'error.request':
    'Silakan coba lagi nanti.',
  'error.requestStatus':
    'Permintaan gagal (HTTP {status})',
  'error.loginRequired':
    'Silakan masuk dahulu sebelum melanjutkan.',
  'error.adminSessionExpired':
    'Status masuk telah kedaluwarsa, Anda akan segera diarahkan ke halaman masuk.',
  'error.fallbackLoad':
    'Gagal memuat',
  'error.fallbackSearch':
    'Gagal mencari',
  'error.fallbackLike':
    'Gagal melakukan suka',
  'error.fallbackComments':
    'Gagal memuat komentar',
  'error.fallbackCommentPost':
    'Gagal mengirim komentar',
  'error.fallbackCommentEdit':
    'Gagal menyimpan komentar',
  'error.fallbackCommentDelete':
    'Gagal menghapus komentar',
  'error.fallbackReport':
    'Gagal mengirimkan laporan',
  'error.fallbackProfile':
    'Gagal membaca profil pribadi',
  'error.fallbackProfileSave':
    'Gagal menyimpan',
  'error.fallbackPublish':
    'Format respons publikasi tidak valid',
  'error.fallbackUpload':
    'Format respons unggah gambar tidak valid',
  'error.fallbackNotFound':
    'Pengguna ini tidak ditemukan',
  'error.fallbackFollow':
    'Gagal mengikuti',
  'auth.checking':
    'Memeriksa status masuk...',
  'auth.statusUnknown':
    'Status masuk tidak dapat dikonfirmasi',
  'auth.feedLoggedIn':
    'Anda sudah masuk; Anda dapat mempublikasikan artikel',
  'auth.feedLoggedOut':
    'Masuk dahulu untuk dapat mempublikasikan artikel',
  'auth.profileLoggedIn':
    'Anda sudah masuk',
  'auth.profileLoggedOut':
    'Masuk dahulu untuk dapat mengatur profil pribadi',
  'auth.googleLogin':
    'Google masuk',
  'auth.loginWithGoogle':
    'Masuk menggunakan akun Google',
  'auth.loginWithGoogleAdmin':
    'Masuk menggunakan akun Admin Google',
  'auth.logout':
    'Keluar',
  'install.button':
    'Pasang Aplikasi',
  'install.hint':
    'Pemutakhiran browser saat ini tidak menyediakan penginstalan otomatis. Buka menu browser, lalu pilih "Pasang aplikasi" atau "Tambahkan ke layar beranda".',
  'bottomNav.label':
    'Navigasi utama',
  'bottomNav.home':
    'Beranda',
  'bottomNav.new':
    'Buat baru',
  'bottomNav.profile':
    'Profil',
  'i18n.ariaLabel':
    'Pilih bahasa',
  'i18n.current':
    'Bahasa: {name}',
  'feed.searchPlaceholder':
    'Cari konten artikel',
  'feed.searchAriaLabel':
    'Cari artikel',
  'feed.searchResultsLabel':
    'Hasil pencarian',
  'feed.postsLabel':
    'Artikel forum',
  'feed.searchFailed':
    'Gagal mencari, silakan coba lagi nanti.',
  'feed.searching':
    'Sedang mencari...',
  'feed.searchMore':
    'Muat lebih banyak hasil pencarian...',
  'feed.searchMoreFailed':
    'Gagal memuat lebih banyak',
  'feed.searchFound':
    'Ditemukan {total} entri',
  'feed.searchDegraded':
    '{base} (layanan pencarian tidak aktif; saat ini menggunakan pencocokan kata kunci basis data)',
  'feed.searchTotal':
    'Pada total {total} hasil',
  'feed.searchNoResults':
    'Artikel dengan "{query}" tidak ditemukan.',
  'feed.loadingPosts':
    'Sedang memuat artikel...',
  'feed.loadMorePosts':
    'Muat lebih banyak artikel...',
  'feed.postsFailed':
    'Gagal memuat artikel, silakan coba lagi nanti.',
  'feed.postsFailedShort':
    'Gagal memuat, silakan coba lagi',
  'feed.scrollMore':
    'Geser ke bawah untuk memuat lebih banyak',
  'feed.endOfFeed':
    'Anda sudah mencapai akhir',
  'feed.noPosts':
    'Belum ada artikel; bagikan pikiran pertama Anda.',
  'feed.likeFailed':
    'Gagal melakukan suka atau membatalkan suka, silakan coba lagi nanti.',
  'post.authorAnonymous':
    'Anonim',
  'post.report':
    'Laporkan artikel',
  'post.imageAlt':
    'Gambar artikel',
  'post.unlike':
    'Batalkan suka',
  'post.like':
    'Suka',
  'post.reply':
    'Balas',
  'post.permalink':
    'Tautan permanen',
  'post.editedBadge':
    'diedit',
  'post.editContentLabel':
    'Isi kiriman',
  'post.editMax':
    'Maksimal 10.000 karakter',
  'comment.loading':
    'Sedang memuat komentar...',
  'comment.none':
    'Belum ada komentar',
  'comment.loadFailed':
    'Gagal memuat komentar, silakan coba lagi',
  'comment.placeholder':
    'Tulis komentar...',
  'comment.max':
    'Maksimal 2.000 karakter',
  'comment.submit':
    'Kirim komentar',
  'comment.failed':
    'Gagal mengirim komentar, silakan coba lagi.',
  'comment.report':
    'Laporkan',
  'comment.more':
    'Muat lebih banyak komentar...',
  'comment.editedBadge':
    'diedit',
  'comment.editContentLabel':
    'Isi komentar',
  'comment.editFailed':
    'Gagal menyimpan komentar, silakan coba lagi nanti.',
  'comment.deleteFailed':
    'Gagal menghapus komentar, silakan coba lagi nanti.',
  'report.reasonPlaceholder':
    'Masukkan alasan laporan (maksimal 500 karakter)',
  'report.note':
    'Laporan akan dikirim kepada administrator situs',
  'report.formLabel':
    'Kolom masukan laporan',
  'report.submit':
    'Kirim laporan',
  'report.failed':
    'Gagal mengirimkan laporan, silakan coba lagi.',
  'report.sent':
    'Laporan Anda telah dikirim, terima kasih atas laporannya.',
  'newPost.avatarYou':
    'Anda',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'Buat artikel',
  'newPost.loginFirst':
    'Masuk menggunakan Google dahulu sebelum memublikasikan artikel',
  'newPost.contentPlaceholder':
    'Bagikan pikiran Anda...',
  'newPost.addImage':
    'Tambahkan gambar',
  'newPost.emailPrivate':
    'Email Anda tidak akan dipublikasikan',
  'newPost.submit':
    'Terbitkan postingan',
  'newPost.publishing':
    'Sedang mempublikasikan...',
  'newPost.uploading':
    'Sedang mengunggah gambar...',
  'newPost.failed':
    'Gagal mempublikasikan. Silakan coba lagi nanti.',
  'newPost.imagePreviewAlt':
    'Pratinjau gambar yang akan diunggah',
  'newPost.draftNote':
    'Draf disimpan otomatis di perangkat ini (hanya teks; gambar yang dipilih tidak ikut tersimpan).',
  'postPage.loading':
    'Memuat kiriman...',
  'postPage.missing':
    'Kiriman ini mungkin sudah dihapus, atau tautannya salah.',
  'postPage.failed':
    'Gagal memuat kiriman, silakan coba lagi nanti.',
  'postPage.label':
    'Kiriman',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'Profil',
  'profile.edit':
    'Ubah',
  'profile.loginPrompt':
    'Sesudah login, atur nama tampilan dan bagian “Tentang Saya” untuk forum.',
  'profile.nicknameLabel':
    'Nama tampilan',
  'profile.notSet':
    'Belum diatur',
  'profile.notSetBio':
    'Belum mengatur tentang saya.',
  'profile.nicknameInput':
    'Nama tampilan',
  'profile.nicknamePlaceholder':
    'Masukkan nama tampilan',
  'profile.nicknameHint':
    'Nama tampilan akan ditampilkan pada artikel yang Anda terbitkan, maksimal 30 karakter.',
  'profile.bioLabel':
    'Tentang saya',
  'profile.bioPlaceholder':
    'Perkenalkan diri Anda (opsional)',
  'profile.bioHint':
    'Maksimal 500 karakter.',
  'profile.updated':
    'Profil diperbarui.',
  'profile.saving':
    'Menyimpan...',
  'profile.saveFailed':
    'Gagal menyimpan. Silakan coba lagi nanti.',
  'profile.loadFailed':
    'Gagal memuat. Silakan coba lagi nanti.',
  'profile.followingEntry':
    'Yang Saya Ikuti',
  'profile.postsLabel':
    'Postingan saya',
  'profile.emptyPosts':
    'Anda belum membuat postingan.',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'Profil publik',
  'publicProfile.avatar':
    'Avatar',
  'publicProfile.loading':
    'Sedang memuat...',
  'publicProfile.invalidLinkName':
    'Tautan profil publik tidak valid',
  'publicProfile.invalidLinkBio':
    'Silakan buka profil melalui nama penulis pada artikel forum.',
  'publicProfile.notFound':
    'Pengguna ini tidak ditemukan',
  'publicProfile.anonymous':
    'Pengguna anonim',
  'publicProfile.noBio':
    'Pengguna ini belum mengatur profil publik.',
  'publicProfile.loadFailed':
    'Gagal memuat profil publik.',
  'publicProfile.postsLabel':
    'Postingan',
  'publicProfile.emptyPosts':
    'Orang ini belum membuat postingan.',

  'follow.label':
    'Ikuti pengguna ini',
  'follow.action':
    'Ikuti',
  'follow.actionDone':
    'Diikuti',
  'follow.unfollow':
    'Berhenti mengikuti',
  'follow.done':
    'Sekarang kamu mengikuti pengguna ini.',
  'follow.failed':
    'Gagal mengikuti. Silakan coba lagi nanti.',

  'following.peopleLabel':
    'Yang kamu ikuti',
  'following.postsLabel':
    'Postingan dari yang kamu ikuti',
  'following.peopleLoading':
    'Memuat daftar mengikuti...',
  'following.emptyPeople':
    'Kamu belum mengikuti siapa pun. Tekan «Ikuti» pada sebuah postingan, atau ikuti seseorang dari profil publiknya.',
  'following.emptyPosts':
    'Belum ada yang kamu ikuti yang membuat postingan.',
  'following.peopleFailed':
    'Gagal memuat daftar mengikuti.',
  'following.postsFailed':
    'Gagal memuat postingan dari yang kamu ikuti.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'Selamat datang kembali',
  'login.body':
    '{site} adalah ruang untuk semua orang yang gemar menuliskan gagasan. Tidak perlu formulir pendaftaran — satu akun Google cukup untuk mulai mempublikasikan artikel.',
  'login.browseFirst':
    'Lihat beranda terlebih dahulu',
  'admin.skipToMain':
    'Lewati ke konten utama',
  'admin.railLabel':
    'Label menu admin',
  'admin.railBrandAria':
    'Beranda {site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'Fitur utama',
  'admin.railGovernance':
    'Tata kelola',
  'admin.railMode':
    'Mode admin',
  'admin.railExit':
    'Kembali ke forum',
  'admin.topbarMenu':
    'Alihkan menu admin',
  'admin.statusOnline':
    'Terhubung dengan normal',
  'admin.topbarForum':
    'Forum',
  'admin.logoutFailed':
    'Gagal keluar. Silakan coba lagi nanti.',
  'admin.navUsers':
    'Manajemen pengguna',
  'admin.navPosts':
    'Artikel forum',
  'admin.navReports':
    'Manajemen laporan',
  'admin.navMonitor': 'Pemantauan sistem',
  'admin.navLog': 'Log tindakan',
  'admin.navStats': 'Tren konten',
  'admin.navExport': 'Ekspor dan aksi massal',
  'admin.navSessions': 'Masuk & sesi',
  'admin.navBlocks': 'Daftar blokir IP',
  'admin.navAnnouncements': 'Pengumuman',
  'admin.listLoadFailed':
    'Gagal memuat',
  'admin.dlgClose':
    'Tutup jendela',
  'admin.dlgConfirm':
    'Konfirmasi',
  'admin.dlgSave':
    'Simpan',
  'admin.dlgApplyTags':
    'Terapkan tag',
  'admin.dlgNoTags':
    'Saat ini tidak ada tag yang dapat diterapkan. Silakan buat tag baru melalui “Manajemen Tag” di bawah ini.',

  'monitor.title': 'Pemantauan sistem',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'Tampilan langsung kesehatan layanan, volume permintaan, sebaran latensi, dan penghitung pembatas laju.',
  'monitor.refresh': 'Muat ulang',
  'monitor.refreshing': 'Memuat…',
  'monitor.autoRefresh': 'Muat ulang otomatis',
  'monitor.autoRefreshOn': 'Muat ulang otomatis tiap {seconds} detik',
  'monitor.autoRefreshOff': 'Muat ulang otomatis dijeda',
  'monitor.nextUpdate': 'Diperbarui dalam {seconds} detik',
  'monitor.loadFailed': 'Gagal memuat data pemantauan.',
  'monitor.loadFailedHint': 'Pastikan Anda masuk sebagai admin dan server masih berjalan.',
  'monitor.pausedHint': 'Muat ulang otomatis dijeda; layar menampilkan hasil pembacaan terakhir yang berhasil.',
  'monitor.visibilityPaused': 'Tab sedang di latar belakang, jadi muat ulang otomatis dijeda.',
  'monitor.lastUpdated': 'Diperbarui pada {time}',
  'monitor.probeTook': 'Pemeriksaan dependensi: {ms} ms',
  'monitor.unreachable': 'Server tidak merespons. Layar berhenti pada pembacaan terakhir yang berhasil.',
  'monitor.depsTitle': 'Kesehatan layanan',
  'monitor.depsNote': 'Setiap pembacaan benar-benar memeriksa setiap dependensi sekali; batas waktu per dependensi adalah 2 detik dan ketiganya berjalan paralel.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'Mesin pencari',
  'monitor.stateOk': 'Normal',
  'monitor.stateDown': 'Tidak dapat dijangkau',
  'monitor.stateDisabled': 'Tidak diaktifkan',
  'monitor.depSearchFallback': 'ES_URL belum diatur, sehingga pencarian memakai pencocokan kata kunci MySQL.',
  'monitor.depDisabled': 'Tidak ada klien Redis yang disuntikkan, jadi fitur media dimatikan.',
  'monitor.depLatency': 'Merespons dalam {ms} ms',
  'monitor.depKeys': '{count} kunci',
  'monitor.depMemory': 'Memori {size}',
  'monitor.depPoolUsage': 'Koneksi {inUse}/{open} (maks {max})',
  'monitor.depPoolWait': '{count} kali menunggu, total {ms} ms',
  'monitor.depRedisPool': 'Hit {hits} / miss {misses}',
  'monitor.depEngineMysql': 'Pencocokan kata kunci MySQL',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'Ringkasan permintaan',
  'monitor.statUptime': 'Waktu aktif',
  'monitor.statRequests': 'Total permintaan',
  'monitor.statErrorRate': 'Tingkat kesalahan',
  'monitor.statP95': 'Latensi P95',
  'monitor.statInFlight': 'Sedang berjalan',
  'monitor.statGoroutines': 'Goroutine',
  'monitor.statHeap': 'Memori heap',
  'monitor.statDbPool': 'Koneksi basis data',
  'monitor.statRateLimited': 'Dibatasi laju',
  'monitor.statCountWithPeak': 'puncak {peak}',
  'monitor.statCountWithInUse': '{inUse} terpakai, {idle} menganggur',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} inti logis · {gc} siklus GC',
  'monitor.noData': 'Belum ada permintaan.',
  'monitor.noDataBody': 'Permintaan sejak layanan mulai akan muncul di sini; panel ini masih kosong.',
  'monitor.timelineTitle': 'Traffik {minutes} menit terakhir',
  'monitor.timelineNote': 'Total bersifat kumulatif sejak proses ini mulai dan direset saat restart; setiap persentil latensi adalah batas atas satu bucket histogram, jadi hanya mengambil nilai diskret. Menit tanpa batang berarti tidak ada trafik pada saat itu.',
  'monitor.timelineLive': 'Proses ini',
  'monitor.timelineHistory': 'Sebelum restart',
  'monitor.timelineLegendVolume': 'Permintaan',
  'monitor.timelineLegendError': 'Galat 5xx',
  'monitor.timelinePeak': 'Puncak {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'Tidak ada agregat historis di basis data; bila saat start tidak bisa membacanya (tabel hilang atau tidak punya izin), hanya proses saat ini yang ditampilkan.',
  'monitor.routesTitle': 'Per rute',
  'monitor.routesNote': 'Jalur dinormalkan (id angka dan alamat sure menjadi :id), sehingga id berbeda pada rute yang sama dihitung bersama.',
  'monitor.colRoute': 'Rute',
  'monitor.colCount': 'Permintaan',
  'monitor.colAvg': 'Rata-rata',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'Terlambat',
  'monitor.colErrors': 'Kesalahan',
  'monitor.routeOther': 'Lainnya (batas jumlah rute tercapai)',
  'monitor.clientsTitle': 'Alamat sumber',
  'monitor.clientsNote': 'Diurutkan dari jumlah permintaan. Alamat yang diambil dari X-Forwarded-For atau X-Real-IP belum diverifikasi oleh proksi tepercaya — pastikan proksi Anda menimpa header tersebut sebelum bertindak.',
  'monitor.noClients': 'Belum ada sumber yang dilacak.',
  'monitor.noClientsBody': 'Setiap permintaan dicatat pada alamat sumbernya. Tabel ini masih kosong.',
  'monitor.clientsDropped': 'Jumlah sumber mencapai batas {limit}; {count} alamat yang paling lama tidak terlihat dikeluarkan dari pelacakan. Baris ini berarti daftar tidak lengkap, bukan hanya sebanyak itu orang yang datang.',
  'monitor.colIp': 'Alamat',
  'monitor.colSource': 'Asal',
  'monitor.colRateLimited': 'Dibatasi laju',
  'monitor.colBanned': 'Ditolak blokir',
  'monitor.colLastRoute': 'Terakhir diakses',
  'monitor.colActions': 'Tindakan',
  'monitor.colBlock': 'Blokir',
  'monitor.blocking': 'Memblokir…',
  'monitor.sourcePeer': 'Lawan koneksi',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.trustLegacy': 'TRUSTED_PROXY_CIDRS belum diatur: server mempertahankan perilaku lama dan memprioritaskan entri paling kiri X-Forwarded-For. Until Anda mengonfirmasi ada proxy di depan yang benar-benar menimpa header ini dan pengguna tidak dapat melewatinya, pembatasan laju dan pemblokiran IP dapat dilewati dengan satu header palsu, dan alamat sumber pada log audit bukan bukti yang sah.',
  'monitor.trustConfigured': 'Proxy tepercaya sudah dikonfigurasi: X-Forwarded-For / X-Real-IP hanya dipercaya bila pasangan koneksi berada di salah satu jaringan ini; selain itu yang dipakai adalah alamat pasangan. Jaringan yang berlaku: {cidrs}.',
  'monitor.trustBroken': 'TRUSTED_PROXY_CIDRS dideklarasikan tetapi tidak ada entri yang terbaca sebagai rentang CIDR ({declared}), sehingga perilaku lama tanpa pengaturan masih berlaku.',
  'monitor.trustPartial': 'Entri TRUSTED_PROXY_CIDRS berikut tidak dapat dibaca sebagai rentang CIDR ({invalid}), sehingga header yang diteruskan dari rentang tersebut tidak pernah dipercaya. Permintaan yang datang melalui rentang itu dikelompokkan menurut alamat pasangan koneksi, sehingga semuanya berbagi satu kuota pembatasan laju dan satu pencarian daftar blokir.',
  'monitor.blockTitle': 'Blokir {ip}',
  'monitor.blockMessage': 'Permintaan tulis dari alamat ini (unggahan, komentar, suka, laporan, unggah gambar, dan pengalihan masuk) akan ditolak selama {duration}. Membaca tidak terpengaruh. Blokir?',
  'monitor.blockReason': 'Diblokir dari halaman pemantauan',
  'monitor.blocked': '{ip} diblokir',
  'monitor.blockFailed': 'Operasi pemblokiran gagal.',
  'monitor.limitsTitle': 'Pembatas laju',
  'monitor.limitsNote': 'Setiap grup punya jatah sendiri sesuai biaya endpoint-nya; jumlah diblokir adalah seluruh respons 429 sejak proses ini mulai.',
  'monitor.colLimiter': 'Pembatas',
  'monitor.colBudget': 'Jatah',
  'monitor.colAllowed': 'Diizinkan',
  'monitor.colBlocked': 'Diblokir',
  'monitor.colTracked': 'Sumber dilacak',
  'monitor.colBlockedRate': 'Tingkat pemblokiran',
  'monitor.limitContent': 'Penulisan konten',
  'monitor.limitUpload': 'Unggah gambar',
  'monitor.limitAuth': 'Masuk OAuth',
  'monitor.limitBudget': '{limit} per {window} detik',
  'monitor.limitUnknown': '(tidak diketahui)',
  'monitor.noLimits': 'Tidak ada pembatas laju.',
  'monitor.noLimitsBody': 'Pembatas laju belum dibuat, jadi penghitungnya tidak tersedia.',

  'log.title': 'Log tindakan',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'Telusuri setiap perubahan di panel admin: siapa, kapan, pada target apa, dan field mana berubah dari apa ke apa.',
  'log.refresh': 'Muat ulang',
  'log.loadFailed': 'Gagal memuat log tindakan.',
  'log.empty': 'Tidak ada tindakan yang cocok.',
  'log.emptyBody': 'Longgarkan filter, atau pastikan memang tidak ada tindakan pada rentang waktu ini.',
  'log.count': 'Entri {from}–{to} dari {total}',
  'log.retention': 'Catatan disimpan {days} hari lalu dihapus oleh tugas latar. Halaman ini tidak punya tombol untuk menghapus catatan — log audit yang bisa menghapus jejaknya sendiri bukan log audit.',
  'log.filterActor': 'Pelaku',
  'log.filterAction': 'Tindakan',
  'log.filterTargetType': 'Jenis sumber daya',
  'log.filterFrom': 'Dari',
  'log.filterTo': 'Sampai',
  'log.filterAll': 'Semua',
  'log.filterApply': 'Terapkan filter',
  'log.filterReset': 'Bersihkan filter',
  'log.filterTargetHint': 'Klik target di baris mana pun untuk hanya melihat tindakan yang menyentuhnya.',
  'log.colTime': 'Waktu',
  'log.colActor': 'Pelaku',
  'log.colAction': 'Tindakan',
  'log.colTarget': 'Target',
  'log.colChanges': 'Perubahan',
  'log.colOrigin': 'Asal',
  'log.noChanges': '(tidak ada perubahan field)',
  'log.changedTo': 'diubah menjadi',
  'log.removed': '(dihapus)',
  'log.created': '(dibuat)',
  'log.requestId': 'request {id}',
  'log.page': 'Halaman {page}',
  'log.targetUser': 'Pengguna',
  'log.targetPost': 'Postingan',
  'log.targetComment': 'Komentar',
  'log.targetReport': 'Laporan',
  'log.targetTag': 'Tag',
  'log.targetSystem': 'Sistem',
  'log.actionUserSuspend': 'Dibekukan',
  'log.actionUserReinstate': 'Dipulihkan',
  'log.actionUserTags': 'Tag diubah',
  'log.actionUserPost': 'Memposting atas nama pengguna',
  'log.actionUserComment': 'Berkomentar atas nama pengguna',
  'log.actionUserContent': 'Kontennya dihapus',
  'log.actionPostCreate': 'Postingan dibuat',
  'log.actionPostUpdate': 'Postingan diedit',
  'log.actionPostDelete': 'Postingan dihapus',
  'log.actionCommentCreate': 'Komentar dibuat',
  'log.actionCommentUpdate': 'Komentar diedit',
  'log.actionCommentDelete': 'Komentar dihapus',
  'log.actionReportCreate': 'Laporan dibuat',
  'log.actionReportResolve': 'Laporan diterima',
  'log.actionReportReject': 'Laporan ditolak',
  'log.actionReportUpdate': 'Laporan diedit',
  'log.actionReportDelete': 'Laporan dihapus',
  'log.actionTagCreate': 'Tag dibuat',
  'log.actionTagUpdate': 'Tag diganti namanya',
  'log.actionTagDelete': 'Tag dihapus',
  'log.fieldStatus': 'Status akun',
  'log.fieldContent': 'Konten',
  'log.fieldName': 'Nama',
  'log.fieldTags': 'Tag',
  'log.fieldReason': 'Alasan',
  'log.fieldAuthorEmail': 'Penulis',
  'log.fieldReporterEmail': 'Pelapor',
  'log.fieldTargetType': 'Jenis sumber daya',
  'log.fieldTargetId': 'Id sumber daya',
  'log.fieldPostId': 'Id postingan',
  'log.fieldCommentId': 'Id komentar',
  'log.fieldPostIdShort': 'Postingan',
  'log.fieldCommentIdShort': 'Komentar',
  'log.fieldTarget': 'Target',
  'log.fieldAssignmentsRemoved': 'Asosiasi dihapus',
  'log.truncated': 'dipotong',

  'stats.title': 'Tren konten',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'Berapa banyak pengguna, postingan, dan komentar baru yang masuk setiap hari, serta postingan, tag, dan penulis yang paling aktif sekarang.',
  'stats.refresh': 'Muat ulang',
  'stats.loadFailed': 'Gagal memuat statistik konten.',
  'stats.window': 'Tampilkan',
  'stats.windowDays': '{days} hari terakhir',
  'stats.windowClamped': '(maksimal 90 hari)',
  'stats.windowNote': 'Setiap hari dipotong menurut zona waktu lokal mesin server. Jika situs berjalan di UTC sementara admin berada di zona lain, angka hari ini terlihat rendah — itu zona waktu, bukan penurunan lalu lintas.',
  'stats.generatedAt': 'Statistik dibuat pada {time}',
  'stats.seriesTitle': 'Baru per hari',
  'stats.seriesNote': 'Ketiga kurva punya skala berbeda, jadi ditampilkan terpisah bukan ditumpuk.',
  'stats.seriesUsers': 'Pengguna baru',
  'stats.seriesPosts': 'Postingan baru',
  'stats.seriesComments': 'Komentar baru',
  'stats.seriesEmpty': 'Tidak ada data pada rentang ini.',
  'stats.totalsTitle': 'Total rentang ini',
  'stats.totalsNote': 'Ini adalah jumlah yang ditambahkan selama rentang ini, bukan total saat ini.',
  'stats.totalUsers': 'Pengguna baru',
  'stats.totalPosts': 'Postingan baru',
  'stats.totalComments': 'Komentar baru',
  'stats.totalLikes': 'Suka baru',
  'stats.topPostsTitle': 'Postingan populer',
  'stats.topPostsNote': 'Diurutkan berdasarkan komentar dan suka, hanya menghitung postingan yang diterbitkan dalam rentang ini.',
  'stats.topTagsTitle': 'Tag populer',
  'stats.topTagsNote': 'Diurutkan berdasarkan jumlah pengguna yang memakai, tanpa batas waktu — tag adalah atribut, bukan peristiwa.',
  'stats.topAuthorsTitle': 'Penulis aktif',
  'stats.topAuthorsNote': 'Diurutkan berdasarkan postingan dalam rentang ini, jumlah komentar ditampilkan terpisah.',
  'stats.colExcerpt': 'Cuplikan',
  'stats.colEngagement': 'Interaksi',
  'stats.colPosts': 'Postingan',
  'stats.colComments': 'Komentar',
  'stats.colUsers': 'Pengguna',
  'stats.colAuthor': 'Penulis',
  'stats.empty': 'Tidak ada data pada rentang ini.',
  'stats.emptyBody': 'Perpanjang rentang, atau pastikan memang tidak ada konten baru.',
  'stats.engagement': '{comments} komentar・{likes} suka',
  'stats.rank': 'Peringkat {rank}',

  'export.title': 'Ekspor dan aksi massal',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'Ekspor data situs sebagai CSV untuk mencocokkan, atau tangani banyak akun sekaligus.',
  'export.download': 'Unduh CSV',
  'export.downloading': 'Menyiapkan…',
  'export.exportTitle': 'Ekspor',
  'export.exportNote': 'Setiap ekspor dibatasi 50.000 baris; selebihnya hanya baris terbaru yang disertakan. Kolom ketiga ekspor sama dengan kolom di halaman admin sehingga pencocokan bisa langsung.',
  'export.exportUsers': 'Daftar pengguna',
  'export.exportUsersNote': 'Email, status, waktu dibuat dan diperbarui, jumlah unggahan dan komentar.',
  'export.exportPosts': 'Daftar unggahan',
  'export.exportPostsNote': 'Id, penulis, 200 karakter pertama isi, waktu dibuat, komentar dan suka.',
  'export.exportReports': 'Daftar laporan',
  'export.exportReportsNote': 'Id, target yang dilaporkan, pelapor, alasan, status, dan riwayat peninjauan.',
  'export.safety': 'Berkas diawali penanda urutan byte UTF-8 sehingga Excel membukanya tanpa karakter rusak.',
  'export.safetyPrefix': 'Nilai yang diawali = + - @ atau spasi tak terlihat mendapat tanda kutip tunggal di depan — itulah yang membuat spreadsheet memperlakukannya sebagai teks, bukan menjalankan formula. Awalan ini disengaja; jangan minta menghapusnya.',
  'export.batchTitle': 'Aksi massal',
  'export.batchNote': 'Tombol aktif setelah Anda memilih akun di halaman Pengguna. Aksi massal diterapkan seluruhnya atau tidak sama sekali; tidak ada hasil sebagian.',
  'export.batchSuspend': 'Tangguhkan terpilih',
  'export.batchReinstate': 'Pulihkan terpilih',
  'export.batchTags': 'Terapkan tag',
  'export.batchTagsNote': 'Semantik timpa: daftar yang dikirim menjadi hasilnya. Mengirim daftar kosong menghapus semua tag.',
  'export.batchConfirm': 'Terapkan “{action}” ke {count} akun?',
  'export.batchConfirmTags': 'Timpa tag {count} akun dengan {tags}?',
  'export.batchTagsPicker': 'Pilih tag',
  'export.batchTagsNone': 'Tanpa tag (hapus semua)',
  'export.batchRunning': 'Sedang memproses…',
  'export.batchDone': '{updated} akun diperbarui',
  'export.batchDoneUnchanged': 'di antaranya {unchanged} sudah dalam status target sehingga tidak berubah',
  'export.batchSkipped': '{count} dilewati',
  'export.batchMax': 'Maksimal 200 akun per batch',
  'export.gotoUsers': 'Ke Pengguna',
  'export.noSelection': 'Pilih dulu akun di halaman Pengguna.',
  'export.selected': '{count} akun dipilih',
  'export.clearSelection': 'Hapus pilihan',
  'export.selectionHint': 'Pilihan hanya berlaku di halaman ini dan hilang saat ditutup.',

  'session.title': 'Masuk & sesi',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'Lihat login mana yang masih berlaku, dan paksa keluar dari sebuah akun di semua perangkat.',
  'session.refresh': 'Muat ulang',
  'session.loadFailed': 'Gagal memuat daftar sesi.',
  'session.privacyTitle': 'Mengapa token lengkap tidak ditampilkan',
  'session.privacyNote': 'Token itu sendiri adalah kredensial masuk. Hanya delapan karakter pertama yang ditampilkan, agar admin bisa membedakan bahwa dua baris adalah sesi yang sama — dan itu tidak cukup bagi siapa pun untuk masuk sebagai pengguna tersebut, termasuk orang yang menerima tangkapan layar halaman ini. Ini batas yang disengaja, bukan fitur yang belum selesai.',
  'session.expireNote': 'Sesi kedaluwarsa setelah {hours} jam tanpa aktivitas; permintaan apa pun memperpanjangnya.',
  'session.filterEmail': 'Saring berdasarkan akun',
  'session.filterPlaceholder': 'Alamat email lengkap',
  'session.search': 'Cari',
  'session.clearFilter': 'Bersihkan',
  'session.summary': '{total} sesi di seluruh situs, {scanned} kunci diperiksa',
  'session.truncated': 'Pemindaian mencapai batas {scanned} kunci dan berhenti lebih awal, jadi daftar ini tidak lengkap.',
  'session.empty': 'Tidak ada sesi saat ini.',
  'session.emptyBody': 'Tidak ada yang masuk, atau semua sesi sudah kedaluwarsa.',
  'session.colUser': 'Akun',
  'session.colToken': 'Sesi',
  'session.colCreated': 'Dibuat',
  'session.colExpires': 'Kedaluwarsa',
  'session.colRemaining': 'Tersisa',
  'session.colActions': 'Tindakan',
  'session.unknown': 'Tidak diketahui',
  'session.adminBadge': 'Admin',
  'session.revoke': 'Paksa keluar',
  'session.revokeTitle': 'Paksa keluar dari {email}',
  'session.revokeMessage': '{count} sesi aktif pada akun itu akan langsung tidak berlaku dan status masuk dihapus di semua perangkat. Pengguna harus masuk lagi. Lanjutkan?',
  'session.revokeRunning': 'Mencabut…',
  'session.revokeDone': '{count} sesi dicabut',
  'session.revokeNone': 'Akun ini tidak punya sesi aktif',
  'session.revokeFailed': 'Tidak dapat memastikan keluar.',
  'session.revokeUnavailable': 'Itu tidak berarti pencabutan gagal: pemindaian mencapai batas kunci, sehingga sebagian sesi mungkin tidak terjangkau. Coba lagi sebentar lagi.',
  'session.titleColumnNote': 'Hanya awalan untuk pengenalan, tidak bisa dipakai untuk masuk',

  'block.title': 'Daftar blokir IP',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'Masukkan alamat asal yang penyalahgunaan sudah terkonfirmasi ke daftar ini. Daftarnya ada di Redis, jadi tidak terhapus oleh restart maupun deploy.',
  'block.refresh': 'Muat ulang',
  'block.loadFailed': 'Gagal memuat daftar blokir.',
  'block.unavailable': 'Situs ini tidak punya koneksi Redis, jadi pemblokiran belum aktif.',
  'block.unavailableNote': 'Daftar ini butuh Redis yang sama dengan sesi dan token media. Setelah terhubung, halaman ini akan berfungsi; sampai itu tiba satu-satunya perlindungan adalah pembatasan laju (per proses, hilang saat restart).',
  'block.add': 'Blokir',
  'block.addTitle': 'Blokir alamat IP',
  'block.addMessage': 'Alamat yang diblokir akan ditolak pada setiap permintaan tulis (unggahan, komentar, suka, laporan, unggah gambar, dan pengalihan masuk) sampai kedaluwarsanya lewat. Membaca tidak terpengaruh.',
  'block.ipLabel': 'Alamat IP',
  'block.ipPlaceholder': '203.0.113.9 atau 2001:db8::1',
  'block.durationLabel': 'Durasi',
  'block.reasonLabel': 'Alasan',
  'block.reasonPlaceholder': 'Mengapa alamat ini diblokir (tercatat di log audit)',
  'block.reasonHint': 'Alasan hanya masuk ke log audit. Tidak pernah ditampilkan kepada orang yang diblokir dan tidak muncul di pesan galat publik.',
  'block.blocking': 'Memblokir…',
  'block.done': '{ip} diblokir',
  'block.removed': '{ip} dibuka blokirnya',
  'block.removedNone': '{ip} memang tidak sedang diblokir',
  'block.failed': 'Operasi pemblokiran gagal.',
  'block.unavailableService': 'Daftar blokir tidak tersedia (tanpa Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'Kedaluwarsa',
  'block.colRemaining': 'Tersisa',
  'block.colActions': 'Tindakan',
  'block.unblock': 'Buka blokir',
  'block.unblockTitle': 'Buka blokir {ip}',
  'block.unblockMessage': 'Alamat ini langsung mendapat akses tulis seperti biasa. Lanjutkan?',
  'block.empty': 'Daftar blokir kosong.',
  'block.emptyBody': 'Tidak ada alamat yang diblokir. Itu keadaan normal: pemblokiran selalu keputusan seorang administrator, dan sistem tidak pernah memblokir siapa pun secara otomatis.',
  'block.notAutoNote': 'Daftar ini tidak pernah terisi sendiri. Alamat yang melampaui batas lajunya hanya menerima 429; tidak ditambahkan ke sini secara otomatis, karena egress yang sama bisa milik satu kantor atau satu NAT, dan pemblokiran otomatis akan mengenai mereka juga.',
  'block.scopeNoteLabel': 'Cakupan',
  'block.notAutoNoteLabel': 'Tidak pernah otomatis',
  'block.maxNoteLabel': 'Durasi maksimum',
  'block.scopeNote': 'Pemblokiran hanya menghentikan permintaan tulis. Membaca unggahan, komentar, dan aset statis tetap bisa, dan orang yang diblokir masih bisa masuk serta melihat konten: itu disengaja, karena titik akhir baca memang tidak dibatasi (kalau tidak, pengunjung anonim tidak bisa memakai situs), dan pemblokiran mencakup kumpulan yang sama.',
  'block.maxNote': 'Satu pemblokiran berlaku paling lama 365 hari. Durasi yang lebih lama dipotong jadi setahun: masa berakhirnya disimpan sebagai angka, dan “selamanya” akan menjadi blokir yang tidak ada yang ingat dan tidak pernah lift sendiri.',
  'block.count': '{count} alamat diblokir',
  'block.ipInvalid': 'Alamat IP tidak valid. Masukkan alamat IPv4 atau IPv6; rentang CIDR tidak didukung.',
  'announce.label': 'Pengumuman situs',
  'announce.publicNote': 'Pemberitahuan',
  'announce.closeAria': 'Tutup pengumuman ini',
  'announce.publishedOn': 'Diterbitkan {date}',
  'announce.expiresOn': 'Berakhir {date}',
  'announce.neverExpires': 'Tanpa kedaluwarsa',
  'announce.pinnedBadge': 'Disematkan',
  'announce.title': 'Pengumuman',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'Tampilkan satu pengumuman di atas setiap halaman situs. Hanya satu yang berlaku pada satu waktu — menerbitkan yang baru akan menonaktifkan yang sebelumnya.',
  'announce.refresh': 'Muat ulang',
  'announce.loadFailed': 'Gagal memuat daftar pengumuman.',
  'announce.new': 'Terbitkan pengumuman baru',
  'announce.edit': 'Ubah',
  'announce.deactivate': 'Nonaktifkan',
  'announce.reactivate': 'Aktifkan kembali',
  'announce.deleteNote': 'Pengumuman tidak pernah dihapus, hanya dinonaktifkan — menyimpan riwayatnya adalah cara menjawab kapan dan siapa menerbitkannya.',
  'announce.bodyLabel': 'Teks pengumuman',
  'announce.bodyPlaceholder': 'Misalnya: sistem akan menjalankan pemeliharaan pada Kamis pukul 02:00–04:00.',
  'announce.bodyHint': 'Maksimal 300 karakter. Teks biasa; baris baru dipertahankan.',
  'announce.activeLabel': 'Tampilkan sekarang',
  'announce.expiryLabel': 'Masa berlaku',
  'announce.expiryNever': 'Tidak pernah kedaluwarsa otomatis',
  'announce.expiryHours': 'dalam {hours} jam',
  'announce.expiryDays': 'dalam {days} hari',
  'announce.saving': 'Menyimpan…',
  'announce.published': 'Pengumuman diterbitkan',
  'announce.updated': 'Pengumuman diperbarui',
  'announce.deactivated': 'Pengumuman dinonaktifkan',
  'announce.reactivated': 'Pengumuman diaktifkan kembali',
  'announce.saveFailed': 'Operasi pengumuman gagal.',
  'announce.empty': 'Belum ada pengumuman.',
  'announce.emptyBody': 'Begitu Anda menerbitkannya, ia akan muncul di atas halaman setiap pengunjung.',
  'announce.colBody': 'Teks',
  'announce.colState': 'Status',
  'announce.colAuthor': 'Diterbitkan oleh',
  'announce.colCreated': 'Waktu terbit',
  'announce.colActions': 'Tindakan',
  'announce.stateActive': 'Ditampilkan',
  'announce.stateInactive': 'Dinonaktifkan',
  'announce.stateExpired': 'Kedaluwarsa',
  'announce.confirmDeactivate': 'Menerbitkannya akan menonaktifkan yang sekarang, dan semua orang langsung melihat teks baru. Lanjutkan?',
  'announce.confirmEdit': 'Ubah teks atau masa berlaku pengumuman ini?',
  'announce.count': 'Total {count}',
  'users.title':
    'Manajemen pengguna',
  'users.contentAction':
    'Konten',
  'users.updateContentFailed':
    'Gagal mengelola konten.',
  'users.updated':
    'Konten diperbarui.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'Tinjau statistik aktivitas pengguna forum, tag, dan status akun, lalu tangani status nonaktif serta tata kelola konten pengguna satu per satu.',
  'users.refresh':
    'Perbarui data',
  'users.statTotal':
    'Total pengguna',
  'users.statActive':
    'Aktif',
  'users.statSuspended':
    'Dinonaktifkan',
  'users.statContent':
    'Total artikel／komentar',
  'users.count':
    '{count} pengguna',
  'users.tagsCount':
    '{count} tag',
  'users.loadFailed':
    'Gagal memuat data pengguna.',
  'users.tagsLoadFailed':
    'Gagal memuat data tag.',
  'users.panelTitle':
    'Pengguna forum',
  'users.tagsPanelTitle':
    'Tag pengguna',
  'users.colUser':
    'Pengguna',
  'users.colTags':
    'Tag',
  'users.colStatus':
    'Status',
  'users.colPosts':
    'Artikel',
  'users.colComments':
    'Komentar',
  'users.colLikes':
    'Suka',
  'users.colLastActivity':
    'Aktivitas terakhir',
  'users.colActions':
    'Tindakan',
  'users.nicknameUnset':
    'Nama tampilan belum diatur',
  'users.notSet':
    'Belum diatur',
  'users.statusActive':
    'Aktif',
  'users.statusSuspended':
    'Dinonaktifkan',
  'users.emptyTitle':
    'Saat ini tidak ada data pengguna',
  'users.emptyBody':
    'Pengguna ini akan kosong jika tidak ada pengguna yang pernah login ke forum melalui Google.',
  'users.tagsEmptyTitle':
    'Saat ini tidak ada tag',
  'users.tagsEmptyBody':
    'Buat tag terlebih dahulu agar dapat diterapkan pada pengguna dalam daftar pengguna.',
  'users.addTag':
    'Tambah tag',
  'users.colName':
    'Nama',
  'users.colCreated':
    'Dibuat',
  'users.colUpdated':
    'Diperbarui',
  'users.renameTag':
    'Ubah nama',
  'users.suspend':
    'Nonaktifkan',
  'users.restore':
    'Pulihkan',
  'users.statusDialogTitle':
    '{action} pengguna ini',
  'users.statusSuspendMessage':
    '{email} tidak dapat masuk ke forum lagi, sedangkan artikel dan komentar yang ada akan tetap dipertahankan. Lanjutkan?',
  'users.statusRestoreMessage':
    '{email} akan memulihkan hak masuk dan hak memublikasikan artikel. Lanjutkan?',
  'users.userSuspended':
    'Pengguna dinonaktifkan.',
  'users.userRestored':
    'Pengguna dipulihkan.',
  'users.updateStatusFailed':
    'Gagal memperbarui status pengguna.',
  'users.editTagsTitle':
    'Ubah tag · {user}',
  'users.editTagsMessage':
    'Pilih tag yang akan diterapkan; mencentang semua sama dengan menghapus seluruh tag pengguna tersebut.',
  'users.tagsUpdated':
    'Tag pengguna telah diperbarui.',
  'users.updateTagsFailed':
    'Gagal memperbarui tag pengguna.',
  'users.contentLoadFailed':
    'Gagal memuat konten.',
  'users.contentLoadFailedShort':
    'Gagal memuat konten.',
  'users.contentPanelTitle':
    'Konten pengguna',
  'users.contentCount':
    '{posts} artikel · {comments} komentar',
  'users.contentLoading':
    'Memuat artikel dan komentar…',
  'users.addPost':
    'Tambah artikel',
  'users.addComment':
    'Tambah komentar',
  'users.postsColumn':
    'Artikel',
  'users.commentsColumn':
    'Komentar',
  'users.noPosts':
    'Belum ada artikel',
  'users.noComments':
    'Belum ada komentar',
  'users.postRef':
    'Artikel #{id}',
  'users.editRecordTitle':
    'Ubah {kind} #{id}',
  'users.deleteRecordTitle':
    'Hapus {kind} #{id}',
  'users.deleteRecordMessage':
    'Penghapusan bersifat permanen; suka terkait dan data tautan juga akan dihapus. Lanjutkan?',
  'users.contentLabel':
    'Konten',
  'users.addPostTitle':
    'Tambah artikel',
  'users.addPostMessage':
    'Konten ini akan dipublikasikan atas nama pengguna tersebut; bidang penulis tidak dapat dipalsukan.',
  'users.postContentLabel':
    'Konten artikel',
  'users.postContentPlaceholder':
    'Masukkan konten artikel',
  'users.pickPostTitle':
    'Pilih artikel',
  'users.postIdLabel':
    'Artikel ID',
  'users.postIdPlaceholder':
    'Nomor artikel untuk dikomentari',
  'users.addCommentTitle':
    'Tambah komentar',
  'users.commentContentLabel':
    'Konten komentar',
  'users.commentContentPlaceholder':
    'Masukkan konten komentar',
  'users.createTagTitle':
    'Tambah tag',
  'users.createTagMessage':
    'Tag dapat digunakan untuk memberi kategori pengguna, misalnya "Moderator", "Aktif", atau "Diblokir".',
  'users.tagNameLabel':
    'Nama tag',
  'users.tagNamePlaceholder':
    'Maksimal 50 karakter',
  'users.renameTagTitle':
    'Ubah nama tag',
  'users.renameTagMessage':
    'Seluruh pengguna yang menggunakan tag ini akan melihat nama baru.',
  'users.deleteTagTitle':
    'Hapus tag "{name}"',
  'users.deleteTagMessage':
    'Setelah dihapus, seluruh tag ini pada semua pengguna akan ikut dihapus dan tidak dapat dipulihkan. Lanjutkan?',
  'users.deleteTagConfirm':
    'Hapus tag',
  'users.tagCreated':
    'Tag telah dibuat.',
  'users.createTagFailed':
    'Gagal membuat tag.',
  'users.tagUpdated':
    'Tag telah diperbarui.',
  'users.updateTagFailed':
    'Gagal memperbarui tag.',
  'users.tagDeleted':
    'Tag telah dihapus.',
  'users.deleteTagFailed':
    'Gagal menghapus tag.',
  'users.refreshDone':
    'Data telah diperbarui ke versi terbaru.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'Halaman admin',
  'users.signinBody':
    'Tata kelola konten terpusat, sehingga setiap peninjauan jelas, cepat, dan dapat ditelusuri.',
  'users.signinStep1':
    'Pemeriksaan keamanan',
  'users.signinStep2':
    'Tata kelola pengguna',
  'users.signinStep3':
    'Peninjauan konten',
  'users.signinPanelTitle':
    'Masuk ke Admin Console',
  'users.signinPanelBody':
    'Admin Console hanya tersedia bagi akun administrator Google yang telah diotorisasi. Silakan masuk sebagai administrator.',
  'posts.title':
    'Artikel forum',
  'posts.searching':
    'Mencari…',
  'posts.searchDegraded':
    '（Layanan pencarian dinonaktifkan, menggunakan pencocokan kata kunci basis data）',
  'posts.searchSummary':
    '"{query}" ditemukan {total} hasil; halaman ini menampilkan {shown} hasil',
  'posts.pendingCount':
    '{count} belum diproses',
  'posts.pageSummary':
    'Halaman {page} / {pages}, halaman ini berisi {count} artikel',
  'posts.listLoadFailed':
    'Gagal memuat daftar artikel.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'Buat, ubah, dan hapus artikel forum, serta kelola konten pada tingkat komentar.',
  'posts.toReports':
    'Pengelolaan laporan',
  'posts.editorTitleNew':
    'Tambah artikel',
  'posts.editorTitleEdit':
    'Ubah artikel #{id}',
  'posts.editorNote':
    'Dipublikasikan sebagai administrator; penulis diambil dari identitas yang masuk, sehingga identitas penulis dari konten yang diminta tidak dapat dipalsukan.',
  'posts.cancelEdit':
    'Batal mengubah',
  'posts.contentLabel':
    'Konten artikel',
  'posts.contentPlaceholder':
    'Masukkan konten artikel',
  'posts.saveChanges':
    'Simpan perubahan',
  'posts.emptyContent':
    'Konten artikel tidak boleh kosong.',
  'posts.saving':
    'Sedang menyimpan…',
  'posts.saved':
    'Artikel telah diperbarui.',
  'posts.published':
    'Artikel telah dipublikasikan.',
  'posts.saveFailed':
    'Gagal menyimpan.',
  'posts.deleteTitle':
    'Hapus artikel #{id}',
  'posts.deleteMessage':
    'Penghapusan bersifat permanen; seluruh komentar di bawah artikel juga akan dihapus. Lanjutkan?',
  'posts.deleteConfirm':
    'Hapus artikel',
  'posts.deleted':
    'Artikel telah dihapus.',
  'posts.deleteFailed':
    'Gagal menghapus artikel.',
  'posts.edited':
    'Kiriman diperbarui.',
  'posts.editFailed':
    'Gagal menyimpan kiriman, silakan coba lagi nanti.',
  'posts.commentUpdated':
    'Komentar telah diperbarui.',
  'posts.commentActionFailed':
    'Gagal melakukan tindakan pada komentar.',
  'posts.pickPostTitle':
    'Pilih artikel tempat komentar berada',
  'posts.pickPostMessage':
    'Artikel pada baris ini adalah pilihan awal; untuk menggantungkannya pada artikel lain, ubah ke nomor artikel yang benar.',
  'posts.postIdLabel':
    'Artikel ID',
  'posts.addCommentAtTitle':
    'Tambah komentar pada artikel #{id}',
  'posts.addCommentMessage':
    'Komentar ini akan dipublikasikan sebagai administrator.',
  'posts.commentContentLabel':
    'Konten komentar',
  'posts.commentContentPlaceholder':
    'Masukkan konten komentar',
  'posts.add':
    'Tambah',
  'posts.editCommentTitle':
    'Ubah komentar #{id}',
  'posts.deleteCommentTitle':
    'Hapus komentar #{id}',
  'posts.deleteCommentMessage':
    'Penghapusan bersifat permanen. Lanjutkan?',
  'posts.listTitle':
    'Daftar artikel',
  'posts.clearSearch':
    'Hapus pencarian',
  'posts.searchLabel':
    'Pencarian kata kunci',
  'posts.searchPlaceholder':
    'Isi postingan atau Email lengkap penulis postingan',
  'posts.searchHint':
    'Diurutkan berdasarkan relevansi; masukkan Email lengkap untuk menemukan semua postingan milik pengguna ini. Pencarian menggantikan paginasi, dan hasil paling banyak menampilkan 25 entri.',
  'posts.searchTotal':
    'Jumlah hasil pencarian {total}',
  'posts.searchFailed':
    'Pencarian gagal.',
  'posts.searchStatusFailed':
    'Pencarian gagal',
  'posts.colContentImage':
    'Konten dan gambar',
  'posts.colEngagement':
    'Interaksi',
  'posts.colComments':
    'Komentar',
  'posts.colAuthor':
    'Penulis',
  'posts.imageAlt':
    'Gambar artikel',
  'posts.likes':
    '{count} suka',
  'posts.author':
    'Penulis: {name}',
  'posts.emptyTitle':
    'Saat ini tidak ada artikel forum',
  'posts.emptyBody':
    'Anda dapat membuat artikel pertama menggunakan editor di atas.',
  'posts.emptySearchTitle':
    'Tidak ada artikel yang sesuai',
  'posts.emptySearchBody':
    'Tidak ada artikel yang cocok dengan "{query}". Coba gunakan kata kunci lain.',
  'posts.commentCount':
    '{count} komentar',
  'posts.noComments':
    'Belum ada komentar',
  'posts.reportsTitle':
    'Laporan yang menunggu diproses',
  'posts.allReports':
    'Semua laporan',
  'posts.colReportedContent':
    'Konten yang dilaporkan',
  'posts.colReason':
    'Alasan laporan',
  'posts.colReporter':
    'Pelapor',
  'posts.colTime':
    'Waktu',
  'posts.colVerdict':
    'Keputusan',
  'posts.emptyReportsTitle':
    'Tidak ada laporan yang menunggu diproses',
  'posts.emptyReportsBody':
    'Semua laporan sudah diputuskan.',
  'posts.verdictResolved':
    'Sudah ditangani',
  'posts.verdictRejected':
    'Tidak berdasar',
  'posts.verdictDialogTitle':
    'Tandai laporan #{id} sebagai "{label}"',
  'posts.verdictDialogMessage':
    'Setelah ditandai, laporan ini akan keluar dari daftar menunggu diproses, tetapi datanya tetap tersimpan di halaman manajemen laporan. Lanjutkan?',
  'posts.verdictConfirm':
    'Tandai sebagai {label}',
  'posts.verdictDone':
    'Laporan telah ditandai sebagai {label}.',
  'posts.verdictFailed':
    'Gagal memperbarui status laporan.',
  'posts.pin': 'Sematkan',
  'posts.unpin': 'Lepas sematan',
  'posts.pinTitle': 'Sematkan unggahan ini',
  'posts.unpinTitle': 'Lepas sematan unggahan ini',
  'posts.pinMessage': 'Setelah disematkan, unggahan ini tetap di bagian atas feed semua orang dan tidak dapat tersusun oleh unggahan baru.',
  'posts.unpinMessage': 'Setelah dilepas, unggahan ini kembali ke tempatnya menurut waktu.',
  'posts.pinDone': 'Disematkan',
  'posts.unpinDone': 'Sematan dilepas',
  'posts.pinFailed': 'Operasi menyematkan gagal.',
  'reports.title':
    'Manajemen laporan',
  'reports.listSummary':
    '{count} laporan · {filter}',
  'reports.listLoadFailed':
    'Gagal memuat daftar laporan.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'Putusan laporan dilakukan satu per satu: "Diterima" menghapus konten yang dilaporkan, sedangkan "Tidak berdasar" mempertahankan teks asli. Mengubah status kembali menjadi "Menunggu diproses" akan menghapus waktu putusan yang ada.',
  'reports.backToPosts':
    'Kembali ke artikel',
  'reports.editorTitle':
    'Edit laporan #{id}',
  'reports.editorNote':
    'Anda dapat melengkapi alasan dan status laporan; target dan pelapor adalah catatan yang sudah ada dan tidak diubah di sini.',
  'reports.targetTypeLabel':
    'Jenis target',
  'reports.targetIdLabel':
    'ID target',
  'reports.reporterEmailLabel':
    'Email pelapor',
  'reports.statusLabel':
    'Status',
  'reports.reasonLabel':
    'Alasan laporan',
  'reports.reasonHint':
    'Maksimal 500 karakter, akan langsung ditampilkan kepada administrator lain sebagai dasar penilaian.',
  'reports.filterLabel':
    'Saring berdasarkan status',
  'reports.filterAll':
    'Semua',
  'reports.listTitle':
    'Daftar laporan',
  'reports.colTarget':
    'Konten target',
  'reports.colReason':
    'Alasan laporan',
  'reports.colReporter':
    'Pelapor',
  'reports.colStatus':
    'Status',
  'reports.targetGone':
    '(Konten telah dihapus)',
  'reports.author':
    'Penulis: {name}',
  'reports.deleteTitle':
    'Hapus laporan #{id}',
  'reports.deleteMessage':
    'Yang dihapus adalah catatan laporan ini sendiri; konten yang dilaporkan tidak terpengaruh dan tidak dapat dipulihkan. Lanjutkan?',
  'reports.deleteConfirm':
    'Hapus laporan',
  'reports.deleted':
    'Laporan telah dihapus.',
  'reports.deleteFailed':
    'Gagal menghapus laporan.',
  'reports.updated':
    'Laporan telah diperbarui.',
  'reports.saveFailed':
    'Gagal menyimpan laporan.',
  'reports.approveTitle':
    'Terima laporan #{id}',
  'reports.approveGoneMessage':
    '{kind} #{id} yang dilaporkan sudah tidak ada, jadi laporan ini hanya akan ditandai sebagai sudah ditangani.',
  'reports.approveMessage':
    '{kind} #{id} yang dilaporkan akan dihapus secara permanen (jika artikel, semua komentar di bawah artikel tersebut juga akan dihapus), dan laporan ini akan ditandai sebagai sudah ditangani. Lanjutkan?',
  'reports.approveConfirm':
    'Terima dan hapus artikel',
  'reports.approveGoneDone':
    'Konten sudah tidak ada, laporan telah ditandai sebagai sudah ditangani.',
  'reports.approveDone':
    'Artikel telah dihapus dan laporan ditandai sebagai sudah ditangani.',
  'reports.approveFailed':
    'Gagal menerima laporan.',
  'reports.approveTitleGone':
    'Konten sudah dihapus; laporan hanya akan ditandai',
  'reports.approveTitleFull':
    'Hapus konten yang dilaporkan dan tandai sebagai sudah ditangani',
  'reports.rejectTitle':
    'Laporan #{id} tidak berdasar',
  'reports.rejectMessage':
    'Tidak berdasar berarti konten yang dilaporkan tidak perlu ditindaklanjuti dan akan dipertahankan sebagaimana adanya. Lanjutkan?',
  'reports.rejectConfirm':
    'Tandai sebagai tidak berdasar',
  'reports.rejectDone':
    'Laporan telah ditandai sebagai tidak berdasar.',
  'reports.statusFailed':
    'Gagal memperbarui status laporan.',
  'reports.emptyTitle':
    'Saat ini tidak ada laporan',
  'reports.emptyBody':
    'Tidak ada catatan untuk filter ini.',
  'reports.rejectTitleAttr':
    'Pertahankan konten dan hanya tandai laporan sebagai tidak berdasar',
  'kind.post':
    'Artikel',
  'kind.comment':
    'Komentar',
  'reports.statusPending':
    'Menunggu diproses',
  'reports.statusResolved':
    'Sudah ditangani',
  'reports.statusRejected':
    'Tidak berdasar',
  'title.forum':
    '{site}',
  'title.login':
    'Masuk｜{site}',
  'title.newPost':
    'Tambah Postingan｜{site}',
  'title.profile':
    'Profil Pribadi｜{site}',
  'title.publicProfile':
    'Profil Pribadi Publik｜{site}',
  'title.post':
    'Kiriman｜{site}',
  'title.following':
    'Mengikuti｜{site}',
  'title.adminUsers':
    'Manajemen Pengguna｜{brand} Halaman Admin',
  'title.adminLogin':
    'Masuk｜{brand} Halaman Admin',
  'title.adminPosts':
    'Artikel Forum｜{brand} Halaman Admin',
  'title.adminReports':
    'Manajemen Laporan｜{brand} Halaman Admin',
  'title.adminMonitor': 'Pemantauan sistem｜{brand} Admin',
  'title.adminLog': 'Log tindakan｜{brand} Admin',
  'title.adminStats': 'Tren konten｜{brand} Admin',
  'title.adminExport': 'Ekspor dan aksi massal｜{brand} Admin',
  'title.adminSessions': 'Masuk & sesi｜{brand} Admin',
  'title.adminBlocks': 'Daftar blokir IP｜{brand} Admin',
  'title.adminAnnouncements': 'Pengumuman｜{brand} Admin',
};
