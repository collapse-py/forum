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
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 Stunde',
  'common.oneDay': '1 Tag',
  'common.sevenDays': '7 Tage',
  'common.thirtyDays': '30 Tage',
  'common.oneYear': '1 Jahr',
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
  'error.fallbackCommentEdit':
    'Kommentar konnte nicht gespeichert werden',
  'error.fallbackCommentDelete':
    'Kommentar konnte nicht gelöscht werden',
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
  'post.permalink':
    'Permalink',
  'post.editedBadge':
    'bearbeitet',
  'post.editContentLabel':
    'Beitragstext',
  'post.editMax':
    'Maximal 10.000 Zeichen',
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
  'comment.editedBadge':
    'bearbeitet',
  'comment.editContentLabel':
    'Kommentartext',
  'comment.editFailed':
    'Kommentar konnte nicht gespeichert werden, bitte später erneut versuchen.',
  'comment.deleteFailed':
    'Kommentar konnte nicht gelöscht werden, bitte später erneut versuchen.',
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
  'newPost.imagePreviewAlt':
    'Vorschau des hochzuladenden Bildes',
  'newPost.draftNote':
    'Der Entwurf wird auf diesem Gerät automatisch gespeichert (nur Text, ein ausgewähltes Bild bleibt nicht erhalten).',
  'postPage.loading':
    'Beitrag wird geladen...',
  'postPage.missing':
    'Dieser Beitrag wurde möglicherweise gelöscht, oder der Link ist falsch.',
  'postPage.failed':
    'Der Beitrag konnte nicht geladen werden, bitte später erneut versuchen.',
  'postPage.label':
    'Beitrag',
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
  'profile.postsLabel':
    'Meine Beiträge',
  'profile.emptyPosts':
    'Du hast noch nichts geschrieben.',
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
  'admin.navMonitor': 'Systemüberwachung',
  'admin.navLog': 'Aktionsprotokoll',
  'admin.navStats': 'Inhaltstrends',
  'admin.navExport': 'Export und Sammelaktionen',
  'admin.navSessions': 'Anmeldungen & Sessions',
  'admin.navBlocks': 'IP-Sperrliste',
  'admin.navAnnouncements': 'Ankündigungen',
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

  'monitor.title': 'Systemüberwachung',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'Live-Ansicht von Dienstzustand, Anfragevolumen, Latenzverteilung und Zählern der Ratenbegrenzer.',
  'monitor.refresh': 'Aktualisieren',
  'monitor.refreshing': 'Wird geladen…',
  'monitor.autoRefresh': 'Automatische Aktualisierung',
  'monitor.autoRefreshOn': 'Automatische Aktualisierung alle {seconds} s',
  'monitor.autoRefreshOff': 'Automatische Aktualisierung pausiert',
  'monitor.nextUpdate': 'Aktualisierung in {seconds} s',
  'monitor.loadFailed': 'Überwachungsdaten konnten nicht geladen werden.',
  'monitor.loadFailedHint': 'Prüfen Sie, ob Sie als Administrator angemeldet sind und das Backend noch läuft.',
  'monitor.pausedHint': 'Die automatische Aktualisierung ist pausiert; die Ansicht zeigt den letzten erfolgreichen Abruf.',
  'monitor.visibilityPaused': 'Die Seite ist im Hintergrund, daher ist die automatische Aktualisierung pausiert.',
  'monitor.lastUpdated': 'Aktualisiert um {time}',
  'monitor.probeTook': 'Abhängigkeitsprüfungen: {ms} ms',
  'monitor.unreachable': 'Das Backend antwortet nicht. Die Ansicht steht beim letzten erfolgreichen Abruf.',
  'monitor.depsTitle': 'Dienstzustand',
  'monitor.depsNote': 'Jeder Abruf prüft jede Abhängigkeit einmal; das Zeitlimit je Abhängigkeit beträgt 2 s, alle drei laufen parallel.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'Suchmaschine',
  'monitor.stateOk': 'Erreichbar',
  'monitor.stateDown': 'Nicht erreichbar',
  'monitor.stateDisabled': 'Nicht aktiviert',
  'monitor.depSearchFallback': 'ES_URL ist nicht gesetzt, die Suche nutzt daher einen MySQL-Schlüsselwortabgleich.',
  'monitor.depDisabled': 'Es wurde kein Redis-Client übergeben, Medienfunktionen sind deaktiviert.',
  'monitor.depLatency': 'Antwort in {ms} ms',
  'monitor.depKeys': '{count} Schlüssel',
  'monitor.depMemory': 'Speicher {size}',
  'monitor.depPoolUsage': 'Verbindungen {inUse}/{open} (max. {max})',
  'monitor.depPoolWait': '{count} Warteschlangen, insgesamt {ms} ms',
  'monitor.depRedisPool': 'Treffer {hits} / Fehlversuche {misses}',
  'monitor.depEngineMysql': 'MySQL-Schlüsselwortabgleich',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'Anfrageübersicht',
  'monitor.statUptime': 'Laufzeit',
  'monitor.statRequests': 'Anfragen gesamt',
  'monitor.statErrorRate': 'Fehlerrate',
  'monitor.statP95': 'P95-Latenz',
  'monitor.statInFlight': 'Anfragen in Bearbeitung',
  'monitor.statGoroutines': 'Goroutines',
  'monitor.statHeap': 'Heap-Speicher',
  'monitor.statDbPool': 'Datenbankverbindungen',
  'monitor.statRateLimited': 'Ratenbegrenzt',
  'monitor.statCountWithPeak': 'Spitze {peak}',
  'monitor.statCountWithInUse': '{inUse} belegt, {idle} frei',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} logische Kerne · {gc} GC-Durchläufe',
  'monitor.noData': 'Noch keine Anfragen.',
  'monitor.noDataBody': 'Anfragen seit dem Start des Dienstes erscheinen hier; dieser Bereich ist derzeit leer.',
  'monitor.timelineTitle': 'Verkehr der letzten {minutes} Minuten',
  'monitor.timelineNote': 'Die Summen werden seit dem Start dieses Prozesses aufsummiert und nach einem Neustart zurückgesetzt; jeder Latenz-Perzentilwert ist eine Obergrenze eines Histogramm-Faches und landet daher nur auf diskreten Stufen. Eine Minute ohne Balken bedeutet, dass es zu diesem Zeitpunkt keinen Verkehr gab.',
  'monitor.timelineLive': 'Dieser Prozess',
  'monitor.timelineHistory': 'Vor dem Neustart',
  'monitor.timelineLegendVolume': 'Anfragen',
  'monitor.timelineLegendError': '5xx-Fehler',
  'monitor.timelinePeak': 'Spitze {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'In der Datenbank sind keine historischen Aggregate verfügbar; wenn der Start sie nicht lesen kann (Tabelle fehlt oder keine Berechtigung), wird nur der aktuelle Prozess angezeigt.',
  'monitor.routesTitle': 'Nach Route',
  'monitor.routesNote': 'Pfade werden normalisiert (numerische IDs und E-Mail-Adressen werden zu :id), daher werden verschiedene IDs derselben Route zusammengezählt.',
  'monitor.colRoute': 'Route',
  'monitor.colCount': 'Anfragen',
  'monitor.colAvg': 'Mittelwert',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'Langsamste',
  'monitor.colErrors': 'Fehler',
  'monitor.routeOther': 'Sonstige (Routenlimit erreicht)',
  'monitor.clientsTitle': 'Absenderadressen',
  'monitor.clientsNote': 'Nach Anzahl der Anfragen sortiert. Eine aus X-Forwarded-For oder X-Real-IP übernommene Adresse wurde nicht gegen einen vertrauenswürdigen Proxy geprüft — vergewissern Sie sich vorher, dass Ihr Proxy diese Header überschreibt.',
  'monitor.noClients': 'Noch keine Quellen erfasst.',
  'monitor.noClientsBody': 'Jede Anfrage wird ihrer Absenderadresse zugeordnet. Diese Tabelle ist derzeit leer.',
  'monitor.clientsDropped': 'Die Anzahl der Quellen hat das Limit {limit} erreicht; die {count} am längsten nicht gesehenen Adressen wurden aus der Erfassung entfernt. Diese Zeile bedeutet, dass die Liste unvollständig ist, nicht dass nur so viele Personen hier waren.',
  'monitor.colIp': 'Adresse',
  'monitor.colSource': 'Herkunft',
  'monitor.colRateLimited': 'Ratenbegrenzt',
  'monitor.colBanned': 'Durch Sperre abgelehnt',
  'monitor.colLastRoute': 'Zuletzt aufgerufen',
  'monitor.colActions': 'Aktionen',
  'monitor.colBlock': 'Sperren',
  'monitor.blocking': 'Wird gesperrt…',
  'monitor.sourcePeer': 'Gegenstelle',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.trustLegacy': 'TRUSTED_PROXY_CIDRS ist nicht gesetzt: Der Server behält das alte Verhalten und bevorzugt den linkesten Eintrag aus X-Forwarded-For. Bis bestätigt ist, dass davor ein Proxy steht, der diese Header tatsächlich überschreibt und dass Benutzer ihn nicht umgehen können, lassen sich Rate-Limit und IP-Sperren mit einem einzigen gefälschten Header umgehen; die Quelladresse im Audit-Log ist kein Beleg.',
  'monitor.trustConfigured': 'Vertrauenswürdige Proxies sind konfiguriert: X-Forwarded-For / X-Real-IP werden nur akzeptiert, wenn die Gegenstelle in einem dieser Netze liegt; sonst zählt die Adresse der Gegenstelle. Derzeit wirksam: {cidrs}.',
  'monitor.trustBroken': 'TRUSTED_PROXY_CIDRS ist angegeben, aber kein Eintrag lässt sich als CIDR-Bereich parsen ({declared}) – es gilt also weiterhin das alte Verhalten ohne Einstellung.',
  'monitor.trustPartial': 'Die folgenden Einträge von TRUSTED_PROXY_CIDRS lassen sich nicht als CIDR-Bereiche parsen ({invalid}); weitergeleitete Header aus diesen Bereichen werden daher nie akzeptiert. Anfragen aus diesen Bereichen werden nach der Gegenstellenadresse gruppiert und teilen sich damit ein Rate-Limit-Budget und eine Blocklist-Abfrage.',
  'monitor.blockTitle': '{ip} sperren',
  'monitor.blockMessage': 'Schreibanfragen von dieser Adresse (Beiträge, Kommentare, Likes, Meldungen, Bild-Uploads, Anmelde-Weiterleitungen) werden {duration} abgelehnt. Lesen ist nicht betroffen. Sperren?',
  'monitor.blockReason': 'Von der Monitorseite gesperrt',
  'monitor.blocked': '{ip} gesperrt',
  'monitor.blockFailed': 'Der Sperrvorgang ist fehlgeschlagen.',
  'monitor.limitsTitle': 'Ratenbegrenzer',
  'monitor.limitsNote': 'Jede Gruppe hat ein eigenes Budget, passend zu den Kosten ihrer Endpunkte; die Blockierungen zählen alle 429-Antworten seit Prozessstart.',
  'monitor.colLimiter': 'Begrenzer',
  'monitor.colBudget': 'Budget',
  'monitor.colAllowed': 'Erlaubt',
  'monitor.colBlocked': 'Blockiert',
  'monitor.colTracked': 'Erfasste Quellen',
  'monitor.colBlockedRate': 'Sperrquote',
  'monitor.limitContent': 'Inhalts-Schreibzugriffe',
  'monitor.limitUpload': 'Bild-Uploads',
  'monitor.limitAuth': 'OAuth-Anmeldung',
  'monitor.limitBudget': '{limit} pro {window} s',
  'monitor.limitUnknown': '(unbekannt)',
  'monitor.noLimits': 'Keine Ratenbegrenzer verfügbar.',
  'monitor.noLimitsBody': 'Die Ratenbegrenzer wurden noch nicht erstellt, ihre Zähler sind nicht verfügbar.',

  'log.title': 'Aktionsprotokoll',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'Jede Änderung im Adminbereich nachschlagen: wer, wann, auf welchem Ziel und welche Felder sich von was zu was geändert haben.',
  'log.refresh': 'Aktualisieren',
  'log.loadFailed': 'Das Aktionsprotokoll konnte nicht geladen werden.',
  'log.empty': 'Keine Aktion entspricht den Filtern.',
  'log.emptyBody': 'Filter lockern oder bestätigen, dass in diesem Zeitraum tatsächlich nichts passiert ist.',
  'log.count': 'Einträge {from}–{to} von {total}',
  'log.retention': 'Einträge werden {days} Tage aufbewahrt und danach von einem Hintergrundprozess entfernt. Diese Seite hat keine Schaltfläche zum Löschen — ein Audit-Protokoll, das seine eigenen Einträge löschen kann, ist kein Audit-Protokoll.',
  'log.filterActor': 'Ausführende',
  'log.filterAction': 'Aktion',
  'log.filterTargetType': 'Ressourcentyp',
  'log.filterFrom': 'Von',
  'log.filterTo': 'Bis',
  'log.filterAll': 'Alle',
  'log.filterApply': 'Filter anwenden',
  'log.filterReset': 'Filter zurücksetzen',
  'log.filterTargetHint': 'Auf ein Ziel in einer Zeile klicken, um nur die Aktionen darauf zu sehen.',
  'log.colTime': 'Zeit',
  'log.colActor': 'Ausführende',
  'log.colAction': 'Aktion',
  'log.colTarget': 'Ziel',
  'log.colChanges': 'Änderungen',
  'log.colOrigin': 'Herkunft',
  'log.noChanges': '(keine Feldänderung)',
  'log.changedTo': 'geändert zu',
  'log.removed': '(gelöscht)',
  'log.created': '(erstellt)',
  'log.requestId': 'request {id}',
  'log.page': 'Seite {page}',
  'log.targetUser': 'Benutzer',
  'log.targetPost': 'Beitrag',
  'log.targetComment': 'Kommentar',
  'log.targetReport': 'Meldung',
  'log.targetTag': 'Schlagwort',
  'log.targetSystem': 'System',
  'log.actionUserSuspend': 'gesperrt',
  'log.actionUserReinstate': 'entsperrt',
  'log.actionUserTags': 'Schlagwörter geändert',
  'log.actionUserPost': 'Als Benutzer gepostet',
  'log.actionUserComment': 'Als Benutzer kommentiert',
  'log.actionUserContent': 'Inhalte gelöscht',
  'log.actionPostCreate': 'Beitrag erstellt',
  'log.actionPostUpdate': 'Beitrag bearbeitet',
  'log.actionPostDelete': 'Beitrag gelöscht',
  'log.actionCommentCreate': 'Kommentar erstellt',
  'log.actionCommentUpdate': 'Kommentar bearbeitet',
  'log.actionCommentDelete': 'Kommentar gelöscht',
  'log.actionReportCreate': 'Meldung erstellt',
  'log.actionReportResolve': 'Meldung bestätigt',
  'log.actionReportReject': 'Meldung abgelehnt',
  'log.actionReportUpdate': 'Meldung bearbeitet',
  'log.actionReportDelete': 'Meldung gelöscht',
  'log.actionTagCreate': 'Schlagwort erstellt',
  'log.actionTagUpdate': 'Schlagwort umbenannt',
  'log.actionTagDelete': 'Schlagwort gelöscht',
  'log.fieldStatus': 'Kontostatus',
  'log.fieldContent': 'Inhalt',
  'log.fieldName': 'Name',
  'log.fieldTags': 'Schlagwörter',
  'log.fieldReason': 'Grund',
  'log.fieldAuthorEmail': 'Autor',
  'log.fieldReporterEmail': 'Meldende Person',
  'log.fieldTargetType': 'Ressourcentyp',
  'log.fieldTargetId': 'Ressourcen-ID',
  'log.fieldPostId': 'Beitrags-ID',
  'log.fieldCommentId': 'Kommentar-ID',
  'log.fieldPostIdShort': 'Beitrag',
  'log.fieldCommentIdShort': 'Kommentar',
  'log.fieldTarget': 'Ziel',
  'log.fieldAssignmentsRemoved': 'Zuordnungen entfernt',
  'log.truncated': 'gekürzt',

  'stats.title': 'Inhaltstrends',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'Wie viele neue Nutzer, Beiträge und Kommentare pro Tag eintreichen und welche Beiträge, Schlagwörter und Autoren gerade am aktivsten sind.',
  'stats.refresh': 'Aktualisieren',
  'stats.loadFailed': 'Die Inhaltsstatistik konnte nicht geladen werden.',
  'stats.window': 'Zeitraum',
  'stats.windowDays': 'Letzte {days} Tage',
  'stats.windowClamped': '(höchstens 90 Tage)',
  'stats.windowNote': 'Jeder Tag wird in der lokalen Zeitzone des Servers abgeschnitten. Läuft die Seite in UTC und sitzt die Administration anderswo, wirkt der heutige Wert zu niedrig — das ist die Zeitzone, kein Traffic-Rückgang.',
  'stats.generatedAt': 'Statistik erzeugt um {time}',
  'stats.seriesTitle': 'Neu pro Tag',
  'stats.seriesNote': 'Die drei Kurven haben unterschiedliche Skalen und werden deshalb getrennt statt übereinander gezeigt.',
  'stats.seriesUsers': 'Neue Nutzer',
  'stats.seriesPosts': 'Neue Beiträge',
  'stats.seriesComments': 'Neue Kommentare',
  'stats.seriesEmpty': 'In diesem Zeitraum keine Daten.',
  'stats.totalsTitle': 'Summen für den Zeitraum',
  'stats.totalsNote': 'Das ist die im Zeitraum hinzugefügte Menge, nicht der aktuelle Gesamtbestand.',
  'stats.totalUsers': 'Neue Nutzer',
  'stats.totalPosts': 'Neue Beiträge',
  'stats.totalComments': 'Neue Kommentare',
  'stats.totalLikes': 'Neue Likes',
  'stats.topPostsTitle': 'Beliebte Beiträge',
  'stats.topPostsNote': 'Nach Kommentaren und Likes sortiert, gezählt werden nur Beiträge aus dem Zeitraum.',
  'stats.topTagsTitle': 'Beliebte Schlagwörter',
  'stats.topTagsNote': 'Nach Anzahl der zugewiesenen Nutzer sortiert, ohne Zeitgrenze — ein Schlagwort ist eine Eigenschaft, kein Ereignis.',
  'stats.topAuthorsTitle': 'Aktive Autoren',
  'stats.topAuthorsNote': 'Nach Beiträgen im Zeitraum sortiert, Kommentarzahlen separat.',
  'stats.colExcerpt': 'Auszug',
  'stats.colEngagement': 'Resonanz',
  'stats.colPosts': 'Beiträge',
  'stats.colComments': 'Kommentare',
  'stats.colUsers': 'Nutzer',
  'stats.colAuthor': 'Autor',
  'stats.empty': 'In diesem Zeitraum keine Daten.',
  'stats.emptyBody': 'Zeitraum vergrößern oder bestätigen, dass wirklich nichts Neues passiert ist.',
  'stats.engagement': '{comments} Kommentare・{likes} Likes',
  'stats.rank': 'Platz {rank}',

  'export.title': 'Export und Sammelaktionen',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'Stationsdaten als CSV exportieren und abgleichen oder viele Konten auf einmal bearbeiten.',
  'export.download': 'CSV herunterladen',
  'export.downloading': 'Wird vorbereitet…',
  'export.exportTitle': 'Export',
  'export.exportNote': 'Jeder Export ist auf 50.000 Zeilen begrenzt; darüber hinaus werden nur die neuesten aufgenommen. Die Spalten aller drei Exporte entsprechen denen der Verwaltungsseiten und lassen sich daher direkt abgleichen.',
  'export.exportUsers': 'Nutzerliste',
  'export.exportUsersNote': 'E-Mail, Status, Erstellungs- und Änderungszeit, Beiträge und Kommentare.',
  'export.exportPosts': 'Beitragsliste',
  'export.exportPostsNote': 'Id, Autor, die ersten 200 Zeichen des Inhalts, Erstellungszeit, Kommentare und Likes.',
  'export.exportReports': 'Meldungen',
  'export.exportReportsNote': 'Id, gemeldetes Ziel, Meldende Person, Grund, Status und Bearbeitungsverlauf.',
  'export.safety': 'Die Datei beginnt mit einem UTF-8-BOM, Excel öffnet sie also ohne Ersatzzeichen.',
  'export.safetyPrefix': 'Werte, die mit = + - @ oder einem unsichtbaren Leerzeichen beginnen, erhalten ein führendes Apostroph — so behandelt die Tabelle sie als Text und führt sie nicht als Formel aus. Das Präfix ist Absicht; bitte nicht um seine Entfernung bitten.',
  'export.batchTitle': 'Sammelaktionen',
  'export.batchNote': 'Die Sammelknöpfe werden aktiv, sobald in der Nutzerverwaltung Konten ausgewählt sind. Eine Sammelaktion wird entweder ganz oder gar nicht angewendet; ein teilweises Ergebnis gibt es nicht.',
  'export.batchSuspend': 'Auswahl sperren',
  'export.batchReinstate': 'Auswahl entsperren',
  'export.batchTags': 'Schlagwörter anwenden',
  'export.batchTagsNote': 'Überschreibend: Die gesendete Liste wird das Ergebnis. Eine leere Liste entfernt alle Schlagwörter.',
  'export.batchConfirm': '{action} auf {count} Konten anwenden?',
  'export.batchConfirmTags': 'Schlagwörter von {count} Konten mit {tags} überschreiben?',
  'export.batchTagsPicker': 'Schlagwörter wählen',
  'export.batchTagsNone': 'Keine Schlagwörter (alle entfernen)',
  'export.batchRunning': 'Wird ausgeführt…',
  'export.batchDone': '{updated} Konten aktualisiert',
  'export.batchDoneUnchanged': 'davon waren {unchanged} bereits im Zielzustand und blieben unverändert',
  'export.batchSkipped': '{count} übersprungen',
  'export.batchMax': 'Höchstens 200 Konten pro Durchgang',
  'export.gotoUsers': 'Zur Nutzerverwaltung',
  'export.noSelection': 'Wähle zuerst Konten in der Nutzerverwaltung aus.',
  'export.selected': '{count} Konten ausgewählt',
  'export.clearSelection': 'Auswahl aufheben',
  'export.selectionHint': 'Die Auswahl gilt nur auf dieser Seite und verschwindet beim Schließen.',

  'session.title': 'Anmeldungen & Sessions',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'Sieh, welche Anmeldungen noch gültig sind, und melde ein Konto auf allen Geräten ab.',
  'session.refresh': 'Aktualisieren',
  'session.loadFailed': 'Die Session-Liste konnte nicht geladen werden.',
  'session.privacyTitle': 'Warum das vollständige Token nicht angezeigt wird',
  'session.privacyNote': 'Das Token ist selbst das Anmeldeberechtnis. Angezeigt werden nur die ersten acht Zeichen, damit eine Administration erkennen kann, ob zwei Zeilen dieselbe Session sind — das reicht nicht aus, um sich als jemand anderes einzuloggen, auch nicht für denjenigen, der einen Screenshot dieser Seite bekommt. Das ist eine bewusste Grenze, keine unfertige Funktion.',
  'session.expireNote': 'Eine Session läuft nach {hours} Stunden ohne Aktivität ab; jede Anfrage verlängert sie.',
  'session.filterEmail': 'Nach Konto filtern',
  'session.filterPlaceholder': 'Vollständige E-Mail-Adresse',
  'session.search': 'Suchen',
  'session.clearFilter': 'Zurücksetzen',
  'session.summary': '{total} Sessions auf der gesamten Seite, {scanned} Schlüssel geprüft',
  'session.truncated': 'Die Suche erreichte ihr Limit von {scanned} Schlüsseln und endete vorzeitig; diese Liste ist unvollständig.',
  'session.empty': 'Derzeit keine Sessions.',
  'session.emptyBody': 'Niemand ist angemeldet, oder alle Sessions sind abgelaufen.',
  'session.colUser': 'Konto',
  'session.colToken': 'Session',
  'session.colCreated': 'Erstellt',
  'session.colExpires': 'Läuft ab',
  'session.colRemaining': 'Rest',
  'session.colActions': 'Aktion',
  'session.unknown': 'Unbekannt',
  'session.adminBadge': 'Administration',
  'session.revoke': 'Abmelden erzwingen',
  'session.revokeTitle': '{email} zwangsweise abmelden',
  'session.revokeMessage': 'Die {count} aktuellen Sessions des Kontos werden sofort ungültig und die Anmeldung auf allen Geräten gelöscht. Die Person muss sich neu anmelden. Fortfahren?',
  'session.revokeRunning': 'Wird widerrufen…',
  'session.revokeDone': '{count} Sessions widerrufen',
  'session.revokeNone': 'Dieses Konto hat keine aktiven Sessions',
  'session.revokeFailed': 'Die Abmeldung konnte nicht bestätigt werden.',
  'session.revokeUnavailable': 'Das bedeutet nicht, dass der Widerruf fehlgeschlagen ist: Die Suche erreichte ihr Schlüssellimit, sodass manche Sessions nicht erreicht wurden. Versuche es gleich noch einmal.',
  'session.titleColumnNote': 'Nur ein Erkennungspräfix, nicht zur Anmeldung verwendbar',

  'block.title': 'IP-Sperrliste',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'Nehmen Sie Absender-Adressen, deren Missbrauch bestätigt ist, auf die Sperrliste. Die Liste liegt in Redis – weder Neustart noch Deployment heben sie auf.',
  'block.refresh': 'Aktualisieren',
  'block.loadFailed': 'Die Sperrliste konnte nicht geladen werden.',
  'block.unavailable': 'Diese Website hat keine Redis-Verbindung, daher ist die Sperrfunktion nicht aktiv.',
  'block.unavailableNote': 'Die Sperrliste braucht dasselbe Redis wie Sessions und Medien-Token. Sobald es verbunden ist, funktioniert diese Seite; bis dahin gibt es nur die Ratenbegrenzung (prozesslokal, bei Neustart verloren).',
  'block.add': 'Sperren',
  'block.addTitle': 'Eine IP-Adresse sperren',
  'block.addMessage': 'Eine gesperrte Absenderadresse wird bei allen Schreibanfragen abgelehnt (Beiträge, Kommentare, Likes, Meldungen, Bild-Uploads, Anmelde-Weiterleitungen), bis die Sperre abläuft. Lesen ist nicht betroffen.',
  'block.ipLabel': 'IP-Adresse',
  'block.ipPlaceholder': '203.0.113.9 oder 2001:db8::1',
  'block.durationLabel': 'Dauer',
  'block.reasonLabel': 'Grund',
  'block.reasonPlaceholder': 'Warum diese Adresse gesperrt wird (im Audit-Protokoll vermerkt)',
  'block.reasonHint': 'Der Grund fließt nur ins Audit-Protokoll. Er wird der gesperrten Person nie gezeigt und erscheint nie in einer öffentlichen Fehlermeldung.',
  'block.blocking': 'Wird gesperrt…',
  'block.done': '{ip} gesperrt',
  'block.removed': '{ip} entsperrt',
  'block.removedNone': '{ip} war ohnehin nicht gesperrt',
  'block.failed': 'Der Sperrvorgang ist fehlgeschlagen.',
  'block.unavailableService': 'Sperrliste nicht verfügbar (kein Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'Läuft ab',
  'block.colRemaining': 'Rest',
  'block.colActions': 'Aktion',
  'block.unblock': 'Entsperren',
  'block.unblockTitle': '{ip} entsperren',
  'block.unblockMessage': 'Diese Adresse erhält sofort wieder normalen Schreibzugriff. Fortfahren?',
  'block.empty': 'Die Sperrliste ist leer.',
  'block.emptyBody': 'Es ist keine Adresse gesperrt. Das ist der Normalzustand: Sperren ist immer eine Entscheidung einer Administration, und das System sperrt nie jemanden automatisch.',
  'block.notAutoNote': 'Diese Liste füllt sich nie von selbst. Eine Adresse, die ihr Ratenlimit überschreitet, erhält nur eine 429; sie wird hier nicht automatisch eingetragen – dieselbe Leitung kann zu einem ganzen Büro oder einem ganzen NAT gehören, und automatisches Sperren würde auch sie treffen.',
  'block.scopeNoteLabel': 'Umfang',
  'block.notAutoNoteLabel': 'Nie automatisch',
  'block.maxNoteLabel': 'Höchstdauer',
  'block.scopeNote': 'Sperren hält nur Schreibanfragen auf. Beiträge, Kommentare und statische Dateien lassen sich weiter lesen, und eine gesperrte Person kann sich weiterhin anmelden und Inhalte sehen: das ist beabsichtigt, weil Leser-Endpunkte bewusst nicht begrenzt werden (sonst könnten anonyme Besucher die Seite nicht nutzen), und Sperren deckt dieselbe Menge ab.',
  'block.maxNote': 'Eine Sperre dauert höchstens 365 Tage. Längere Zeiten werden auf ein Jahr gekürzt, denn der Ablauf wird als Zahl gespeichert – „für immer" würde zu einer Sperre, an die sich niemand erinnert und die sich nie von selbst hebt.',
  'block.count': '{count} gesperrte Adressen',
  'block.ipInvalid': 'Ungültige IP-Adresse. Bitte eine IPv4- oder IPv6-Adresse eingeben; CIDR-Bereiche werden nicht unterstützt.',
  'announce.label': 'Website-Ankündigung',
  'announce.publicNote': 'Hinweis',
  'announce.closeAria': 'Diese Ankündigung schließen',
  'announce.publishedOn': 'Veröffentlicht am {date}',
  'announce.expiresOn': 'Läuft am {date} ab',
  'announce.neverExpires': 'Ohne Ablauf',
  'announce.pinnedBadge': 'Angepinnt',
  'announce.title': 'Ankündigungen',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'Oben auf jeder Seite eine Ankündigung anzeigen. Es ist immer nur eine aktiv; eine neue zu veröffentlichen deaktiviert die vorherige.',
  'announce.refresh': 'Aktualisieren',
  'announce.loadFailed': 'Die Ankündigungen konnten nicht geladen werden.',
  'announce.new': 'Neue Ankündigung veröffentlichen',
  'announce.edit': 'Bearbeiten',
  'announce.deactivate': 'Deaktivieren',
  'announce.reactivate': 'Reaktivieren',
  'announce.deleteNote': 'Ankündigungen werden nie gelöscht, nur deaktiviert — die Historie beantwortet die Frage, wann sie von wem veröffentlicht wurde.',
  'announce.bodyLabel': 'Text der Ankündigung',
  'announce.bodyPlaceholder': 'Zum Beispiel: Das System wird am Donnerstag von 02:00 bis 04:00 gewartet.',
  'announce.bodyHint': 'Höchstens 300 Zeichen. Klartext, Zeilenumbrüche bleiben erhalten.',
  'announce.activeLabel': 'Sofort anzeigen',
  'announce.expiryLabel': 'Gültigkeit',
  'announce.expiryNever': 'Läuft nie automatisch ab',
  'announce.expiryHours': 'in {hours} Stunden',
  'announce.expiryDays': 'in {days} Tagen',
  'announce.saving': 'Wird gespeichert…',
  'announce.published': 'Ankündigung veröffentlicht',
  'announce.updated': 'Ankündigung aktualisiert',
  'announce.deactivated': 'Ankündigung deaktiviert',
  'announce.reactivated': 'Ankündigung reaktiviert',
  'announce.saveFailed': 'Der Vorgang ist fehlgeschlagen.',
  'announce.empty': 'Noch keine Ankündigungen.',
  'announce.emptyBody': 'Sobald Sie eine veröffentlichen, erscheint sie oben auf jeder Besucherseite.',
  'announce.colBody': 'Text',
  'announce.colState': 'Status',
  'announce.colAuthor': 'Veröffentlicht von',
  'announce.colCreated': 'Veröffentlicht am',
  'announce.colActions': 'Aktion',
  'announce.stateActive': 'Angezeigt',
  'announce.stateInactive': 'Deaktiviert',
  'announce.stateExpired': 'Abgelaufen',
  'announce.confirmDeactivate': 'Das Veröffentlichen deaktiviert die aktuelle Ankündigung, und jeder sieht sofort den neuen Text. Fortfahren?',
  'announce.confirmEdit': 'Text oder Gültigkeit dieser Ankündigung ändern?',
  'announce.count': '{count} insgesamt',
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
  'posts.edited':
    'Beitrag aktualisiert.',
  'posts.editFailed':
    'Der Beitrag konnte nicht gespeichert werden, bitte später erneut versuchen.',
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
  'posts.pin': 'Anheften',
  'posts.unpin': 'Lösen',
  'posts.pinTitle': 'Diesen Beitrag anheften',
  'posts.unpinTitle': 'Anheftung aufheben',
  'posts.pinMessage': 'Ein angehefteter Beitrag bleibt ganz oben in allen Feeds und wird von neuen Beiträgen nicht verdrängt.',
  'posts.unpinMessage': 'Nach dem Lösen kehrt der Beitrag an seine Stelle in der Zeitreihenfolge zurück.',
  'posts.pinDone': 'Angeheftet',
  'posts.unpinDone': 'Anheftung aufgehoben',
  'posts.pinFailed': 'Das Anheften ist fehlgeschlagen.',
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
  'title.post':
    'Beitrag｜{site}',
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
  'title.adminMonitor': 'Systemüberwachung｜{brand} Verwaltung',
  'title.adminLog': 'Aktionsprotokoll｜{brand} Verwaltung',
  'title.adminStats': 'Inhaltstrends｜{brand} Verwaltung',
  'title.adminExport': 'Export und Sammelaktionen｜{brand} Verwaltung',
  'title.adminSessions': 'Anmeldungen & Sessions｜{brand} Verwaltung',
  'title.adminBlocks': 'IP-Sperrliste｜{brand} Verwaltung',
  'title.adminAnnouncements': 'Ankündigungen｜{brand} Verwaltung',
};
