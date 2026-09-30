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
};
