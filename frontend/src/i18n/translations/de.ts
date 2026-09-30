/*
 * de catalog (src/i18n/translations/de.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const de: Record<MessageKey, string> = {
  'common.cancel':
    'Abbrechen',
  'common.save':
    'Speichern',
  'common.submitting':
    'Wird gesendet...',
  'common.delete':
    'Löschen',
  'common.edit':
    'Bearbeiten',
  'common.search':
    'Suchen',
  'common.loading':
    'Laden...',
  'common.loadFailed':
    'Laden fehlgeschlagen.',
  'common.refresh':
    'Aktualisieren',
  'common.nextStep':
    'Nächster Schritt',
  'common.prevPage':
    'Vorherige Seite',
  'common.nextPage':
    'Nächste Seite',
  'common.create':
    'Erstellen',
  'common.publish':
    'Veröffentlichen',
  'common.placeholder':
    '—',
  'common.backToHome':
    'Zur Hauptseite',
  'common.backToForumHome':
    'Zurück zur Forumstartseite',
  'common.backOnePage':
    'Zurück zur vorherigen Seite',
  'error.request':
    'Bitte später erneut versuchen.',
  'error.requestStatus':
    'Die Anfrage ist fehlgeschlagen (HTTP {status}).',
  'error.loginRequired':
    'Bitte melden Sie sich zuerst an, um fortzufahren.',
  'error.adminSessionExpired':
    'Die Anmeldesitzung ist abgelaufen; Sie werden zur Anmeldeseite zurückgeleitet.',
  'error.fallbackLoad':
    'Laden fehlgeschlagen.',
  'error.fallbackSearch':
    'Suche fehlgeschlagen.',
  'error.fallbackLike':
    'Gefällt-mir-Vorgang fehlgeschlagen.',
  'error.fallbackComments':
    'Kommentare konnten nicht geladen werden.',
  'error.fallbackCommentPost':
    'Kommentar konnte nicht gesendet werden.',
  'error.fallbackReport':
    'Meldung konnte nicht gesendet werden.',
  'error.fallbackProfile':
    'Profil konnte nicht geladen werden.',
  'error.fallbackProfileSave':
    'Speichern fehlgeschlagen.',
  'error.fallbackPublish':
    'Antwortformat ist ungültig.',
  'error.fallbackUpload':
    'Die Antwort des Bild-Uploads ist ungültig.',
  'error.fallbackNotFound':
    'Dieser Benutzer wurde nicht gefunden.',
  'error.fallbackFollow':
    'Folgen fehlgeschlagen.',
  'auth.checking':
    'Anmeldestatus wird geprüft...',
  'auth.statusUnknown':
    'Anmeldestatus ist unbekannt',
  'auth.feedLoggedIn':
    'Angemeldet; Beiträge möglich.',
  'auth.feedLoggedOut':
    'Nach der Anmeldung können Sie Beiträge veröffentlichen.',
  'auth.profileLoggedIn':
    'Angemeldet.',
  'auth.profileLoggedOut':
    'Nach der Anmeldung können Sie Ihr Profil festlegen.',
  'auth.googleLogin':
    'Google-Anmeldung',
  'auth.loginWithGoogle':
    'Mit einem Google-Konto anmelden',
  'auth.loginWithGoogleAdmin':
    'Mit einem Google-Administrator-Konto anmelden',
  'auth.logout':
    'Abmelden',
  'install.button':
    'App installieren',
  'install.hint':
    'Der aktuelle Browser bietet keine automatische Installation an. Öffnen Sie das Browsermenü und wählen Sie „App installieren“ oder „Zu Startbildschirm hinzufügen“.',
  'bottomNav.label':
    'Hauptnavigation',
  'bottomNav.home':
    'Startseite',
  'bottomNav.new':
    'Neu',
  'bottomNav.profile':
    'Profil',
  'i18n.ariaLabel':
    'Sprache auswählen',
  'i18n.current':
    'Sprache: {name}',
  'feed.searchPlaceholder':
    'Beiträge durchsuchen',
  'feed.searchAriaLabel':
    'Beiträge suchen',
  'feed.searchResultsLabel':
    'Suchergebnisse',
  'feed.postsLabel':
    'Forenbeiträge',
  'feed.searchFailed':
    'Suche fehlgeschlagen, bitte später erneut versuchen.',
  'feed.searching':
    'Suche läuft...',
  'feed.searchMore':
    'Weitere Suchergebnisse laden...',
  'feed.searchMoreFailed':
    'Weitere Ergebnisse konnten nicht geladen werden.',
  'feed.searchFound':
    '{total} Beiträge gefunden',
  'feed.searchDegraded':
    '{base} (Der Suchdienst ist nicht aktiviert; derzeit werden Datenbankbegriffe abgeglichen)',
  'feed.searchTotal':
    'Insgesamt {total} Ergebnisse',
  'feed.searchNoResults':
    'Keine Beiträge mit „{query}“ gefunden.',
  'feed.loadingPosts':
    'Beiträge werden geladen...',
  'feed.loadMorePosts':
    'Weitere Beiträge laden...',
  'feed.postsFailed':
    'Beiträge konnten nicht geladen werden, bitte später erneut versuchen.',
  'feed.postsFailedShort':
    'Laden fehlgeschlagen, bitte später erneut versuchen.',
  'feed.scrollMore':
    'Nach unten scrollen, um weitere Beiträge zu laden',
  'feed.endOfFeed':
    'Am Ende der Beiträge',
  'feed.noPosts':
    'Es gibt noch keine Beiträge. Hinterlasse als Erster deine Gedanken.',
  'feed.likeFailed':
    'Gefällt-mir- oder Gefällt-mir-nicht-mehr-Vorgang fehlgeschlagen, bitte später erneut versuchen.',
  'post.authorAnonymous':
    'Anonym',
  'post.report':
    'Beitrag melden',
  'post.imageAlt':
    'Beitragsbild',
  'post.unlike':
    'Gefällt mir nicht mehr',
  'post.like':
    'Gefällt mir',
  'post.reply':
    'Antwort',
  'comment.loading':
    'Kommentare werden geladen...',
  'comment.none':
    'Noch keine Kommentare',
  'comment.loadFailed':
    'Kommentare konnten nicht geladen werden, bitte später erneut versuchen.',
  'comment.placeholder':
    'Kommentar verfassen...',
  'comment.max':
    'Maximal 2000 Zeichen',
  'comment.submit':
    'Kommentar',
  'comment.failed':
    'Kommentar konnte nicht gesendet werden, bitte später erneut versuchen.',
  'comment.report':
    'Meldung',
  'comment.more':
    'Weitere Kommentare laden...',
  'report.reasonPlaceholder':
    'Grund für die Meldung eingeben (maximal 500 Zeichen)',
  'report.note':
    'Die Meldung wird an die Administratoren der Plattform gesendet.',
  'report.formLabel':
    'Eingabefeld für die Meldung',
  'report.submit':
    'Meldung senden',
  'report.failed':
    'Meldung konnte nicht gesendet werden, bitte später erneut versuchen.',
  'report.sent':
    'Die Meldung wurde gesendet. Danke für Ihren Hinweis.',
  'newPost.avatarYou':
    'Du',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'Beitrag erstellen',
  'newPost.loginFirst':
    'Bitte melden Sie sich zuerst mit Google an, bevor Sie einen Beitrag veröffentlichen',
  'newPost.contentPlaceholder':
    'Teilen Sie Ihre Gedanken...',
  'newPost.addImage':
    'Bild hinzufügen',
  'newPost.emailPrivate':
    'Deine Email wird nicht veröffentlicht',
  'newPost.submit':
    'Beitrag veröffentlichen',
  'newPost.publishing':
    'Veröffentlichen...',
  'newPost.uploading':
    'Bild wird hochgeladen...',
  'newPost.failed':
    'Veröffentlichung fehlgeschlagen. Bitte versuche es später erneut.',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'Profil',
  'profile.edit':
    'Bearbeiten',
  'profile.loginPrompt':
    'Nach dem Anmelden kannst du deinen Anzeigename und dein Profilbio festlegen.',
  'profile.nicknameLabel':
    'Forum-Anzeigename',
  'profile.notSet':
    'Noch nicht festgelegt',
  'profile.notSetBio':
    'Noch nicht für dein Profil festgelegt.',
  'profile.nicknameInput':
    'Anzeigename',
  'profile.nicknamePlaceholder':
    'Anzeigename eingeben',
  'profile.nicknameHint':
    'Dein Anzeigename erscheint auf den von dir veröffentlichten Beiträgen. Maximal 30 Zeichen.',
  'profile.bioLabel':
    'Über mich',
  'profile.bioPlaceholder':
    'Stell dich kurz vor (optional)',
  'profile.bioHint':
    'Maximal 500 Zeichen.',
  'profile.updated':
    'Dein Profil wurde aktualisiert.',
  'profile.saving':
    'Wird gespeichert...',
  'profile.saveFailed':
    'Speichern fehlgeschlagen. Bitte versuche es später erneut.',
  'profile.loadFailed':
    'Profil konnte nicht geladen werden. Bitte versuche es später erneut.',
  'profile.followingEntry':
    'Meine Follows',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'Öffentliches Profil',
  'publicProfile.avatar':
    'Anonym',
  'publicProfile.loading':
    'Wird geladen...',
  'publicProfile.invalidLinkName':
    'Ungleicher öffentlicher Profil-Link',
  'publicProfile.invalidLinkBio':
    'Öffne dein Profil über den Autorennamen in einem Forum-Beitrag.',
  'publicProfile.notFound':
    'Dieser Benutzer wurde nicht gefunden',
  'publicProfile.anonymous':
    'Anonymer Benutzer',
  'publicProfile.noBio':
    'Dieser Benutzer hat noch keine öffentlichen Profilinformationen festgelegt.',
  'publicProfile.loadFailed':
    'Öffentliche Profilinformationen konnten nicht geladen werden.',
  'publicProfile.postsLabel':
    'Beiträge',
  'publicProfile.emptyPosts':
    'Diese Person hat noch nichts geschrieben.',

  'follow.label':
    'Diesem Benutzer folgen',
  'follow.action':
    'Folgen',
  'follow.actionDone':
    'Gefolgt',
  'follow.unfollow':
    'Entfolgen',
  'follow.done':
    'Du folgst diesem Benutzer jetzt.',
  'follow.failed':
    'Folgen fehlgeschlagen. Bitte versuche es später erneut.',

  'following.peopleLabel':
    'Benutzer, denen du folgst',
  'following.postsLabel':
    'Beiträge der Benutzer, denen du folgst',
  'following.peopleLoading':
    'Folge-Liste wird geladen...',
  'following.emptyPeople':
    'Du folgst noch niemandem. Tippe bei einem Beitrag auf „Folgen“ oder folge jemandem über dessen öffentliches Profil.',
  'following.emptyPosts':
    'Die Benutzer, denen du folgst, haben noch nichts geschrieben.',
  'following.peopleFailed':
    'Folge-Liste konnte nicht geladen werden.',
  'following.postsFailed':
    'Beiträge der gefolgten Benutzer konnten nicht geladen werden.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'Willkommen zurück',
  'login.body':
    '{site} ist ein Ort für alle, die ihre Gedanken festhalten wollen. Kein Anmeldeformular nötig – mit einem Google-Konto kannst du sofort Beiträge verfassen.',
  'login.browseFirst':
    'Schau dir zuerst die Startseite an',
  'admin.skipToMain':
    'Direkt zu Hauptinhalt springen',
  'admin.railLabel':
    'Menü für Administration',
  'admin.railBrandAria':
    'Startseite von {site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'Hauptfunktionen',
  'admin.railGovernance':
    'Verwaltung',
  'admin.railMode':
    'Admin-Modus',
  'admin.railExit':
    'Zum Forum zurück',
  'admin.topbarMenu':
    'Verwaltungsmenü umschalten',
  'admin.statusOnline':
    'Verbindung normal',
  'admin.topbarForum':
    'Forum',
  'admin.logoutFailed':
    'Abmeldung fehlgeschlagen. Bitte versuche es später erneut.',
  'admin.navUsers':
    'Benutzerverwaltung',
  'admin.navPosts':
    'Forum-Beiträge',
  'admin.navReports':
    'Meldungsverwaltung',
  'admin.listLoadFailed':
    'Laden fehlgeschlagen.',
  'admin.dlgClose':
    'Fenster schließen',
  'admin.dlgConfirm':
    'Bestätigen',
  'admin.dlgSave':
    'Speichern',
  'admin.dlgApplyTags':
    'Tags anwenden',
  'admin.dlgNoTags':
    'Derzeit können keine Tags angewendet werden. Bitte füge unten zuerst einen „Tag“ hinzu.',
  'users.title':
    'Benutzerverwaltung',
  'users.contentAction':
    'Inhalte',
  'users.updateContentFailed':
    'Inhaltsaktion fehlgeschlagen.',
  'users.updated':
    'Inhalte wurden aktualisiert.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'Sieh dir die Aktivitätsstatistiken, Tags und den Kontostatus der Forum-Benutzer an und verwalte Sperren sowie die Einzelinhaltsgovernance.',
  'users.refresh':
    'Daten aktualisieren',
  'users.statTotal':
    'Gesamtzahl der Benutzer',
  'users.statActive':
    'Aktiv',
  'users.statSuspended':
    'Gesperrt',
  'users.statContent':
    'Gesamtzahl der Beiträge und Kommentare',
  'users.count':
    '{count} Benutzer',
  'users.tagsCount':
    '{count} Tags',
  'users.loadFailed':
    'Benutzerinformationen konnten nicht geladen werden.',
  'users.tagsLoadFailed':
    'Tag-Informationen konnten nicht geladen werden.',
  'users.panelTitle':
    'Forum-Benutzer',
  'users.tagsPanelTitle':
    'Benutzer-Tags',
  'users.colUser':
    'Benutzer',
  'users.colTags':
    'Tags',
  'users.colStatus':
    'Status',
  'users.colPosts':
    'Beiträge',
  'users.colComments':
    'Kommentare',
  'users.colLikes':
    'Gefällt mir',
  'users.colLastActivity':
    'Letzte Aktivität',
  'users.colActions':
    'Aktionen',
  'users.nicknameUnset':
    'Anzeigename nicht festgelegt',
  'users.notSet':
    'Nicht festgelegt',
  'users.statusActive':
    'Aktiv',
  'users.statusSuspended':
    'Gesperrt',
  'users.emptyTitle':
    'Derzeit gibt es keine Benutzerinformationen',
  'users.emptyBody':
    'Dieser Bereich ist nur leer, wenn sich kein Benutzer über Google beim Forum angemeldet hat.',
  'users.tagsEmptyTitle':
    'Derzeit gibt es keine Tags',
  'users.tagsEmptyBody':
    'Erstelle zuerst Tags, damit du sie in der Benutzerliste zuordnen kannst.',
  'users.addTag':
    'Tag hinzufügen',
  'users.colName':
    'Name',
  'users.colCreated':
    'Erstellt am',
  'users.colUpdated':
    'Aktualisiert am',
  'users.renameTag':
    'Tag umbenennen',
  'users.suspend':
    'Sperren',
  'users.restore':
    'Wiederherstellen',
  'users.statusDialogTitle':
    '{action} diesen Benutzer',
  'users.statusSuspendMessage':
    '{email} kann sich nicht mehr beim Forum anmelden. Bestehende Beiträge und Kommentare bleiben erhalten. Möchtest du fortfahren?',
  'users.statusRestoreMessage':
    '{email} erhält seine Anmelde- und Beitragsberechtigungen zurück. Möchtest du fortfahren?',
  'users.userSuspended':
    'Benutzer wurde gesperrt.',
  'users.userRestored':
    'Benutzer wurde wiederhergestellt.',
  'users.updateStatusFailed':
    'Benutzerstatus konnte nicht aktualisiert werden.',
  'users.editTagsTitle':
    '{user} bearbeiten',
  'users.editTagsMessage':
    'Wählen Sie die Tags aus, die Sie anwenden möchten; wenn Sie alle abwählen, werden alle Tags dieses Benutzers entfernt.',
  'users.tagsUpdated':
    'Die Benutzer-Tags wurden aktualisiert.',
  'users.updateTagsFailed':
    'Benutzer-Tags konnten nicht aktualisiert werden.',
  'users.contentLoadFailed':
    'Der Inhalt konnte nicht geladen werden.',
  'users.contentLoadFailedShort':
    'Der Inhalt konnte nicht geladen werden.',
  'users.contentPanelTitle':
    'Benutzerinhalte',
  'users.contentCount':
    '{posts} Beiträge · {comments} Kommentare',
  'users.contentLoading':
    'Beiträge und Kommentare werden geladen…',
  'users.addPost':
    'Beitrag hinzufügen',
  'users.addComment':
    'Kommentar hinzufügen',
  'users.postsColumn':
    'Beiträge',
  'users.commentsColumn':
    'Kommentare',
  'users.noPosts':
    'Noch keine Beiträge',
  'users.noComments':
    'Noch keine Kommentare',
  'users.postRef':
    'Beitrag #{id}',
  'users.editRecordTitle':
    'Bearbeite {kind} #{id}',
  'users.deleteRecordTitle':
    'Lösche {kind} #{id}',
  'users.deleteRecordMessage':
    'Nach dem Löschen ist eine Wiederherstellung nicht möglich; verwandte „Gefällt mir“-Angaben und verknüpfte Daten werden ebenfalls entfernt. Möchten Sie fortfahren?',
  'users.contentLabel':
    'Inhalt',
  'users.addPostTitle':
    'Beitrag hinzufügen',
  'users.addPostMessage':
    'Dieser Inhalt wird in der Rolle dieses Benutzers veröffentlicht; der Autorenname kann nicht gefälscht werden.',
  'users.postContentLabel':
    'Beitragsinhalt',
  'users.postContentPlaceholder':
    'Gib den Beitragsinhalt ein',
  'users.pickPostTitle':
    'Beitrag auswählen',
  'users.postIdLabel':
    'Beitrag-ID',
  'users.postIdPlaceholder':
    'Beitrag-ID, auf die ein Kommentar hinterlassen werden soll',
  'users.addCommentTitle':
    'Kommentar hinzufügen',
  'users.commentContentLabel':
    'Kommentarinhalt',
  'users.commentContentPlaceholder':
    'Gib den Kommentarinhalt ein',
  'users.createTagTitle':
    'Tag hinzufügen',
  'users.createTagMessage':
    'Tags können verwendet werden, um Benutzer zu klassifizieren, z. B. „Moderator“, „aktiv“ oder „blockiert“.',
  'users.tagNameLabel':
    'Tag-Bezeichnung',
  'users.tagNamePlaceholder':
    'Maximal 50 Zeichen',
  'users.renameTagTitle':
    'Tag umbenennen',
  'users.renameTagMessage':
    'Alle Benutzer, die diesen Tag verwenden, sehen den neuen Namen.',
  'users.deleteTagTitle':
    'Tag „{name}“ löschen',
  'users.deleteTagMessage':
    'Nach dem Löschen wird dieser Tag bei allen Benutzern entfernt und kann nicht wiederhergestellt werden. Möchten Sie fortfahren?',
  'users.deleteTagConfirm':
    'Tag löschen',
  'users.tagCreated':
    'Tag wurde erstellt.',
  'users.createTagFailed':
    'Tag konnte nicht erstellt werden.',
  'users.tagUpdated':
    'Tag wurde aktualisiert.',
  'users.updateTagFailed':
    'Tag konnte nicht aktualisiert werden.',
  'users.tagDeleted':
    'Tag wurde gelöscht.',
  'users.deleteTagFailed':
    'Tag konnte nicht gelöscht werden.',
  'users.refreshDone':
    'Auf den neuesten Stand aktualisiert.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'Administration',
  'users.signinBody':
    'Zentrale Inhaltsmoderation ermöglicht eine klare, schnelle und nachvollziehbare Prüfung.',
  'users.signinStep1':
    'Sichere Verifizierung',
  'users.signinStep2':
    'Benutzerverwaltung',
  'users.signinStep3':
    'Inhaltsprüfung',
  'users.signinPanelTitle':
    'Admin Console anmelden',
  'users.signinPanelBody':
    'Die Admin Console ist nur für autorisierte Google-Administrator-Konten geöffnet. Melden Sie sich bitte als Administrator an.',
  'posts.title':
    'Forum-Beiträge',
  'posts.searching':
    'Suche läuft…',
  'posts.searchDegraded':
    '(Suchdienst ist nicht aktiviert; Abgleich mit Datenbank-Schlüsselwörtern)',
  'posts.searchSummary':
    'Für „{query}“ wurden {total} Treffer gefunden; auf dieser Seite werden {shown} angezeigt',
  'posts.pendingCount':
    '{count} offene Einträge',
  'posts.pageSummary':
    'Seite {page} von {pages}, auf dieser Seite {count} Beiträge',
  'posts.listLoadFailed':
    'Beitragsliste konnte nicht geladen werden.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'Erstelle, bearbeite und lösche Forumbeiträge und moderiere Inhalte auf der Ebene der Kommentare.',
  'posts.toReports':
    'Meldungsverwaltung',
  'posts.editorTitleNew':
    'Neuen Beitrag hinzufügen',
  'posts.editorTitleEdit':
    'Beitrag #{id} bearbeiten',
  'posts.editorNote':
    'Als Administrator veröffentlichen; der Autor wird aus dem angemeldeten Benutzer übernommen, und das Autorenfeld kann nicht gefälscht werden.',
  'posts.cancelEdit':
    'Bearbeitung abbrechen',
  'posts.contentLabel':
    'Beitragsinhalt',
  'posts.contentPlaceholder':
    'Gib den Beitragsinhalt ein',
  'posts.saveChanges':
    'Änderungen speichern',
  'posts.emptyContent':
    'Der Beitragsinhalt darf nicht leer sein.',
  'posts.saving':
    'Wird gespeichert…',
  'posts.saved':
    'Beitrag wurde aktualisiert.',
  'posts.published':
    'Beitrag wurde veröffentlicht.',
  'posts.saveFailed':
    'Speichern fehlgeschlagen.',
  'posts.deleteTitle':
    'Beitrag #{id} löschen',
  'posts.deleteMessage':
    'Nach dem Löschen ist eine Wiederherstellung nicht möglich; auch alle Kommentare unter diesem Beitrag werden entfernt. Möchten Sie fortfahren?',
  'posts.deleteConfirm':
    'Beitrag löschen',
  'posts.deleted':
    'Beitrag wurde gelöscht.',
  'posts.deleteFailed':
    'Beitrag konnte nicht gelöscht werden.',
  'posts.commentUpdated':
    'Kommentar wurde aktualisiert.',
  'posts.commentActionFailed':
    'Die Kommentaraktion ist fehlgeschlagen.',
  'posts.pickPostTitle':
    'Beitrag auswählen, auf den der Kommentar verweist',
  'posts.pickPostMessage':
    'Standardmäßig wird der Beitrag in dieser Zeile verwendet; wenn der Kommentar an einem anderen Beitrag angehängt werden soll, ändere die passende Beitrag-ID.',
  'posts.postIdLabel':
    'Beitrag-ID',
  'posts.addCommentAtTitle':
    'Kommentar zu Beitrag #{id} hinzufügen',
  'posts.addCommentMessage':
    'Dieser Kommentar wird als Administrator veröffentlicht.',
  'posts.commentContentLabel':
    'Kommentarinhalt',
  'posts.commentContentPlaceholder':
    'Gib den Kommentarinhalt ein',
  'posts.add':
    'Hinzufügen',
  'posts.editCommentTitle':
    'Kommentar #{id} bearbeiten',
  'posts.deleteCommentTitle':
    'Kommentar #{id} löschen',
  'posts.deleteCommentMessage':
    'Nach dem Löschen ist eine Wiederherstellung nicht möglich. Möchten Sie fortfahren?',
  'posts.listTitle':
    'Beitragsliste',
  'posts.clearSearch':
    'Suche löschen',
  'posts.searchLabel':
    'Keyword-Suche',
  'posts.searchPlaceholder':
    'Postinhalt oder vollständige Email des Postautors',
  'posts.searchHint':
    'Nach Relevanz sortiert; Geben Sie eine vollständige Email ein, um alle Beiträge dieses Benutzers zu finden. Die Suche ersetzt die Seitenanzeige; es werden maximal 25 Ergebnisse angezeigt.',
  'posts.searchTotal':
    'Suchergebnisse insgesamt {total} Einträge',
  'posts.searchFailed':
    'Suche fehlgeschlagen.',
  'posts.searchStatusFailed':
    'Suche fehlgeschlagen',
  'posts.colContentImage':
    'Inhalt und Bild',
  'posts.colEngagement':
    'Interaktion',
  'posts.colComments':
    'Kommentare',
  'posts.colAuthor':
    'Autor',
  'posts.imageAlt':
    'Bild zum Beitrag',
  'posts.likes':
    '{count} Gefällt mir',
  'posts.author':
    'Autor: {name}',
  'posts.emptyTitle':
    'Derzeit keine Forenbeiträge',
  'posts.emptyBody':
    'Sie können den ersten Beitrag mit dem Editor oben erstellen.',
  'posts.emptySearchTitle':
    'Keine passenden Beiträge',
  'posts.emptySearchBody':
    'Für „{query}“ wurde kein Beitrag gefunden. Versuchen Sie es mit einem anderen Suchbegriff.',
  'posts.commentCount':
    '{count} Kommentare',
  'posts.noComments':
    'Noch keine Kommentare',
  'posts.reportsTitle':
    'Ausstehende Meldungen',
  'posts.allReports':
    'Alle Meldungen',
  'posts.colReportedContent':
    'Gemeldeter Inhalt',
  'posts.colReason':
    'Grund der Meldung',
  'posts.colReporter':
    'Melder',
  'posts.colTime':
    'Zeitpunkt',
  'posts.colVerdict':
    'Entscheidung',
  'posts.emptyReportsTitle':
    'Keine ausstehenden Meldungen',
  'posts.emptyReportsBody':
    'Alle Meldungen wurden entschieden.',
  'posts.verdictResolved':
    'Erledigt',
  'posts.verdictRejected':
    'Unbegründet',
  'posts.verdictDialogTitle':
    'Meldung #{id} als „{label}“ markieren',
  'posts.verdictDialogMessage':
    'Nach der Markierung verlässt die Meldung die Liste der ausstehenden Meldungen, die Daten bleiben jedoch auf der Meldungsverwaltung. Möchten Sie fortfahren?',
  'posts.verdictConfirm':
    'Als {label} markieren',
  'posts.verdictDone':
    'Meldung als {label} markiert.',
  'posts.verdictFailed':
    'Fehler beim Aktualisieren des Meldungstatus.',
  'reports.title':
    'Meldungsverwaltung',
  'reports.listSummary':
    '{count} Meldungen · {filter}',
  'reports.listLoadFailed':
    'Fehler beim Laden der Meldungsliste.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'Jede Meldung einzeln bewerten: Durch „Bestätigen“ wird der gemeldete Inhalt gelöscht, durch „Unbegründet“ bleibt der ursprüngliche Text erhalten. Durch die Rückkehr zu „Ausstehend“ werden bestehende Bewertungszeiten entfernt.',
  'reports.backToPosts':
    'Zurück zu den Beiträgen',
  'reports.editorTitle':
    'Meldung #{id} bearbeiten',
  'reports.editorNote':
    'Meldungsgrund und Status können korrigiert werden; Ziel und Melder sind vorhandene Datensätze und werden hier nicht geändert.',
  'reports.targetTypeLabel':
    'Zieltyp',
  'reports.targetIdLabel':
    'Ziel-ID',
  'reports.reporterEmailLabel':
    'Email des Melders',
  'reports.statusLabel':
    'Status',
  'reports.reasonLabel':
    'Grund der Meldung',
  'reports.reasonHint':
    'Bis zu 500 Zeichen; er wird anderen Administratorinnen und Administratoren direkt als Grundlage für die Bewertung angezeigt.',
  'reports.filterLabel':
    'Nach Status filtern',
  'reports.filterAll':
    'Alle',
  'reports.listTitle':
    'Meldungsliste',
  'reports.colTarget':
    'Zielinhalt',
  'reports.colReason':
    'Grund der Meldung',
  'reports.colReporter':
    'Melder',
  'reports.colStatus':
    'Status',
  'reports.targetGone':
    '(Inhalt wurde gelöscht)',
  'reports.author':
    'Autor: {name}',
  'reports.deleteTitle':
    'Meldung #{id} löschen',
  'reports.deleteMessage':
    'Dieser Meldungsdatensatz selbst wird gelöscht; der gemeldete Inhalt bleibt unverändert und kann nicht wiederhergestellt werden. Möchten Sie fortfahren?',
  'reports.deleteConfirm':
    'Meldung löschen',
  'reports.deleted':
    'Meldung wurde gelöscht.',
  'reports.deleteFailed':
    'Fehler beim Löschen der Meldung.',
  'reports.updated':
    'Meldung wurde aktualisiert.',
  'reports.saveFailed':
    'Fehler beim Speichern der Meldung.',
  'reports.approveTitle':
    'Meldung #{id} bestätigen',
  'reports.approveGoneMessage':
    'Der gemeldete {kind} #{id} existiert nicht mehr; dieser Datensatz wird nur als erledigt markiert.',
  'reports.approveMessage':
    'Der gemeldete {kind} #{id} wird dauerhaft gelöscht (falls es sich um einen Beitrag handelt, werden auch alle Kommentare darunter entfernt), und diese Meldung wird als erledigt markiert. Möchten Sie fortfahren?',
  'reports.approveConfirm':
    'Bestätigen und Beitrag löschen',
  'reports.approveGoneDone':
    'Der Inhalt existiert nicht mehr; die Meldung wurde als erledigt markiert.',
  'reports.approveDone':
    'Beitrag wurde gelöscht und die Meldung als erledigt markiert.',
  'reports.approveFailed':
    'Fehler beim Bestätigen der Meldung.',
  'reports.approveTitleGone':
    'Inhalt wurde gelöscht; nur die Meldung wird markiert',
  'reports.approveTitleFull':
    'Gemeldeten Inhalt löschen und als erledigt markieren',
  'reports.rejectTitle':
    'Meldung #{id} als unbegründet markieren',
  'reports.rejectMessage':
    'Unbegründet bedeutet, dass für den gemeldeten Inhalt keine Bearbeitung erforderlich ist; der Inhalt wird unverändert beibehalten. Möchten Sie fortfahren?',
  'reports.rejectConfirm':
    'Als unbegründet markieren',
  'reports.rejectDone':
    'Meldung wurde als unbegründet markiert.',
  'reports.statusFailed':
    'Fehler beim Aktualisieren des Meldungsstatus.',
  'reports.emptyTitle':
    'Derzeit keine Meldungen',
  'reports.emptyBody':
    'Unter diesem Filter befinden sich keine Datensätze.',
  'reports.rejectTitleAttr':
    'Inhalt behalten und nur die Meldung als unbegründet markieren',
  'kind.post':
    'Beitrag',
  'kind.comment':
    'Kommentar',
  'reports.statusPending':
    'Ausstehend',
  'reports.statusResolved':
    'Erledigt',
  'reports.statusRejected':
    'Unbegründet',
  'title.forum':
    '{site}',
  'title.login':
    'Anmeldung | {site}',
  'title.newPost':
    'Neuen Beitrag erstellen | {site}',
  'title.profile':
    'Profil | {site}',
  'title.publicProfile':
    'Öffentliches Profil | {site}',
  'title.following':
    'Folge | {site}',
  'title.adminUsers':
    'Benutzerverwaltung | {brand}-Backend',
  'title.adminLogin':
    'Anmeldung | {brand}-Backend',
  'title.adminPosts':
    'Forumbeiträge | {brand}-Backend',
  'title.adminReports':
    'Meldungsverwaltung | {brand}-Backend',
};
