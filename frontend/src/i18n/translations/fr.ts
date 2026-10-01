/*
 * fr catalog (src/i18n/translations/fr.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const fr: Record<MessageKey, string> = {
  'common.cancel':
    'Annuler',
  'common.save':
    'Enregistrer',
  'common.submitting':
    'Envoi…',
  'common.delete':
    'Supprimer',
  'common.edit':
    'Modifier',
  'common.search':
    'Rechercher',
  'common.loading':
    'Chargement…',
  'common.loadFailed':
    'Échec du chargement',
  'common.refresh':
    'Actualiser',
  'common.nextStep':
    'Prochaine étape',
  'common.prevPage':
    'Page précédente',
  'common.nextPage':
    'Page suivante',
  'common.create':
    'Créer',
  'common.publish':
    'Publier',
  'common.placeholder':
    '—',
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 heure',
  'common.oneDay': '1 jour',
  'common.sevenDays': '7 jours',
  'common.thirtyDays': '30 jours',
  'common.oneYear': '1 an',
  'common.backToHome':
    'Retour à l\'accueil',
  'common.backToForumHome':
    'Retour à l\'accueil du forum',
  'common.backOnePage':
    'Retour à la page précédente',
  'error.request':
    'Veuillez réessayer plus tard.',
  'error.requestStatus':
    'La requête a échoué (HTTP {status}).',
  'error.loginRequired':
    'Veuillez vous connecter pour continuer.',
  'error.adminSessionExpired':
    'Votre session de connexion est expirée. Vous serez bientôt redirigé vers la page de connexion.',
  'error.fallbackLoad':
    'Échec du chargement',
  'error.fallbackSearch':
    'Échec de la recherche',
  'error.fallbackLike':
    'Échec du J\'aime',
  'error.fallbackComments':
    'Échec du chargement des commentaires',
  'error.fallbackCommentPost':
    'Échec de l\'ajout d\'un commentaire',
  'error.fallbackReport':
    'Échec de la soumission du signalement',
  'error.fallbackProfile':
    'Échec de la lecture du profil',
  'error.fallbackProfileSave':
    'Échec de l\'enregistrement',
  'error.fallbackPublish':
    'Format de réponse invalide lors de la publication de la réponse',
  'error.fallbackUpload':
    'Format de réponse invalide lors de l\'envoi de l\'image',
  'error.fallbackNotFound':
    'Utilisateur introuvable',
  'error.fallbackFollow':
    'Échec du suivi',
  'auth.checking':
    'Vérification de la connexion en cours…',
  'auth.statusUnknown':
    'Impossible de confirmer l\'état de connexion',
  'auth.feedLoggedIn':
    'Connexion effectuée, vous pouvez publier',
  'auth.feedLoggedOut':
    'Connectez-vous pour publier',
  'auth.profileLoggedIn':
    'Connecté',
  'auth.profileLoggedOut':
    'Connectez-vous pour configurer votre profil',
  'auth.googleLogin':
    'Connexion Google',
  'auth.loginWithGoogle':
    'Se connecter avec un compte Google',
  'auth.loginWithGoogleAdmin':
    'Se connecter avec un compte administrateur Google',
  'auth.logout':
    'Déconnexion',
  'install.button':
    'Installer l\'application',
  'install.hint':
    'Le navigateur ne propose pas actuellement l\'installation automatique. Ouvrez le menu du navigateur, puis sélectionnez « Installer l\'application » ou « Ajouter à l\'écran d\'accueil ».',
  'bottomNav.label':
    'Navigation principale',
  'bottomNav.home':
    'Accueil',
  'bottomNav.new':
    'Nouveau',
  'bottomNav.profile':
    'Profil',
  'i18n.ariaLabel':
    'Choisir la langue',
  'i18n.current':
    'Langue : {name}',
  'feed.searchPlaceholder':
    'Rechercher dans les publications',
  'feed.searchAriaLabel':
    'Rechercher des publications',
  'feed.searchResultsLabel':
    'Résultats de recherche',
  'feed.postsLabel':
    'Publications du forum',
  'feed.searchFailed':
    'La recherche a échoué. Veuillez réessayer plus tard.',
  'feed.searching':
    'Recherche en cours…',
  'feed.searchMore':
    'Chargement de plus de résultats de recherche…',
  'feed.searchMoreFailed':
    'Échec du chargement',
  'feed.searchFound':
    '{total} résultat(s) trouvé(s)',
  'feed.searchDegraded':
    '{base} (la fonction de recherche n\'est pas activée ; vous utilisez actuellement la recherche par mots-clés dans la base de données)',
  'feed.searchTotal':
    '{total} résultats au total',
  'feed.searchNoResults':
    'Aucune publication ne contient « {query} ».',
  'feed.loadingPosts':
    'Chargement des publications en cours…',
  'feed.loadMorePosts':
    'Chargement d\'autres publications…',
  'feed.postsFailed':
    'Échec du chargement des publications. Veuillez réessayer plus tard.',
  'feed.postsFailedShort':
    'Échec du chargement. Veuillez réessayer plus tard',
  'feed.scrollMore':
    'Faites défiler vers le bas pour charger plus de contenu',
  'feed.endOfFeed':
    'Vous avez atteint la fin du fil',
  'feed.noPosts':
    'Aucune publication pour le moment. Soyez le premier à partager votre avis.',
  'feed.likeFailed':
    'Échec de l\'ajout au J\'aime ou de Je n\'aime plus. Veuillez réessayer plus tard.',
  'post.authorAnonymous':
    'Anonyme',
  'post.report':
    'Signaler la publication',
  'post.imageAlt':
    'Image de la publication',
  'post.unlike':
    'Je n\'aime plus',
  'post.like':
    'J\'aime',
  'post.reply':
    'Réponse',
  'comment.loading':
    'Chargement des commentaires…',
  'comment.none':
    'Aucun commentaire',
  'comment.loadFailed':
    'Échec du chargement des commentaires. Veuillez réessayer plus tard',
  'comment.placeholder':
    'Écrire un commentaire…',
  'comment.max':
    'Maximum : 2 000 caractères',
  'comment.submit':
    'Publier',
  'comment.failed':
    'Échec de l\'ajout du commentaire. Veuillez réessayer plus tard.',
  'comment.report':
    'Signaler',
  'comment.more':
    'Chargement d\'autres commentaires…',
  'report.reasonPlaceholder':
    'Veuillez saisir le motif du signalement (maximum 500 caractères)',
  'report.note':
    'Le signalement sera transmis aux administrateurs du site',
  'report.formLabel':
    'Champ de saisie du signalement',
  'report.submit':
    'Soumettre le signalement',
  'report.failed':
    'Échec de la soumission du signalement. Veuillez réessayer plus tard.',
  'report.sent':
    'Votre signalement a bien été transmis. Merci pour votre retour.',
  'newPost.avatarYou':
    'Vous',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'Nouvelle publication',
  'newPost.loginFirst':
    'Connectez-vous d\'abord avec Google avant de publier',
  'newPost.contentPlaceholder':
    'Partagez votre avis…',
  'newPost.addImage':
    'Ajouter une image',
  'newPost.emailPrivate':
    'Votre email ne sera pas diffusé.',
  'newPost.submit':
    'Publier le post.',
  'newPost.publishing':
    'Publication en cours…',
  'newPost.uploading':
    'Téléchargement de l’image en cours…',
  'newPost.failed':
    'La publication a échoué. Veuillez réessayer plus tard.',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'Profil',
  'profile.edit':
    'Modifier',
  'profile.loginPrompt':
    'Connectez-vous pour définir le pseudo et la bio de votre forum.',
  'profile.nicknameLabel':
    'Pseudo du forum',
  'profile.notSet':
    'Non défini',
  'profile.notSetBio':
    'Non défini.',
  'profile.nicknameInput':
    'Pseudo',
  'profile.nicknamePlaceholder':
    'Saisissez votre pseudo',
  'profile.nicknameHint':
    'Le pseudo s’affichera sur les articles que vous publiez, jusqu’à 30 caractères.',
  'profile.bioLabel':
    'Bio',
  'profile.bioPlaceholder':
    'Présentez-vous (facultatif)',
  'profile.bioHint':
    'Maximum 500 caractères.',
  'profile.updated':
    'Votre profil a été mis à jour.',
  'profile.saving':
    'Enregistrement en cours…',
  'profile.saveFailed':
    'L’enregistrement a échoué. Veuillez réessayer plus tard.',
  'profile.loadFailed':
    'Échec du chargement. Veuillez réessayer plus tard.',
  'profile.followingEntry':
    'Mes abonnements',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'Profil public',
  'publicProfile.avatar':
    'Anonyme',
  'publicProfile.loading':
    'Chargement…',
  'publicProfile.invalidLinkName':
    'Lien vers le profil public invalide.',
  'publicProfile.invalidLinkBio':
    'Accédez au profil depuis le nom de l’auteur dans un article du forum.',
  'publicProfile.notFound':
    'Cet utilisateur n’a pas été trouvé.',
  'publicProfile.anonymous':
    'Utilisateur anonyme.',
  'publicProfile.noBio':
    'Cet utilisateur n’a pas encore défini de profil public.',
  'publicProfile.loadFailed':
    'Échec du chargement du profil public.',
  'publicProfile.postsLabel':
    'Publications',
  'publicProfile.emptyPosts':
    'Cette personne n’a encore rien publié.',

  'follow.label':
    'Suivre cet utilisateur',
  'follow.action':
    'Suivre',
  'follow.actionDone':
    'Abonné',
  'follow.unfollow':
    'Ne plus suivre',
  'follow.done':
    'Vous suivez désormais cet utilisateur.',
  'follow.failed':
    'Échec du suivi. Veuillez réessayer plus tard.',

  'following.peopleLabel':
    'Utilisateurs que vous suivez',
  'following.postsLabel':
    'Publications des utilisateurs suivis',
  'following.peopleLoading':
    'Chargement des abonnements...',
  'following.emptyPeople':
    'Vous ne suivez personne pour le moment. Appuyez sur « Suivre » sur une publication, ou suivez quelqu’un depuis son profil public.',
  'following.emptyPosts':
    'Les utilisateurs que vous suivez n’ont encore rien publié.',
  'following.peopleFailed':
    'Échec du chargement des abonnements.',
  'following.postsFailed':
    'Échec du chargement des publications.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'Bon retour',
  'login.body':
    '{site} est un espace pour tous ceux qui ont des idées à partager. Pas besoin de formulaire d’inscription — un compte Google suffit pour commencer à publier.',
  'login.browseFirst':
    'Regardez d’abord la page d’accueil',
  'admin.skipToMain':
    'Accéder directement au contenu principal',
  'admin.railLabel':
    'Menu d’administration',
  'admin.railBrandAria':
    'Accueil de {site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'Fonctionnalités principales',
  'admin.railGovernance':
    'Gouvernance',
  'admin.railMode':
    'Mode administrateur',
  'admin.railExit':
    'Retour au forum',
  'admin.topbarMenu':
    'Basculer le menu d’administration',
  'admin.statusOnline':
    'En ligne',
  'admin.topbarForum':
    'Forum',
  'admin.logoutFailed':
    'Échec de la déconnexion. Veuillez réessayer plus tard.',
  'admin.navUsers':
    'Gestion des utilisateurs',
  'admin.navPosts':
    'Articles du forum',
  'admin.navReports':
    'Gestion des signalements',
  'admin.navMonitor': 'Supervision du système',
  'admin.navLog': 'Journal des actions',
  'admin.navStats': 'Tendances du contenu',
  'admin.navExport': 'Export et actions groupées',
  'admin.navSessions': 'Connexions et sessions',
  'admin.navBlocks': 'Liste de blocage d\'IP',
  'admin.navAnnouncements': 'Annonces',
  'admin.listLoadFailed':
    'Échec du chargement.',
  'admin.dlgClose':
    'Fermer',
  'admin.dlgConfirm':
    'Confirmer',
  'admin.dlgSave':
    'Enregistrer',
  'admin.dlgApplyTags':
    'Appliquer les Tag',
  'admin.dlgNoTags':
    'Il n’y a actuellement aucun Tag à appliquer. Ajoutez-en d’abord un dans « Gestion des Tag » ci-dessous.',

  'monitor.title': 'Supervision du système',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'Vue en direct de l\'état des services, du volume de requêtes, de la répartition des latences et des compteurs des limiteurs de débit.',
  'monitor.refresh': 'Actualiser',
  'monitor.refreshing': 'Chargement…',
  'monitor.autoRefresh': 'Actualisation automatique',
  'monitor.autoRefreshOn': 'Actualisation automatique toutes les {seconds} s',
  'monitor.autoRefreshOff': 'Actualisation automatique en pause',
  'monitor.nextUpdate': 'Mise à jour dans {seconds} s',
  'monitor.loadFailed': 'Échec du chargement des données de supervision.',
  'monitor.loadFailedHint': 'Vérifiez que vous êtes connecté en tant qu\'administrateur et que le backend fonctionne toujours.',
  'monitor.pausedHint': 'L\'actualisation automatique est en pause ; l\'affichage montre la dernière lecture réussie.',
  'monitor.visibilityPaused': 'La page est en arrière-plan, l\'actualisation automatique est donc en pause.',
  'monitor.lastUpdated': 'Mis à jour à {time}',
  'monitor.probeTook': 'Sondes de dépendances : {ms} ms',
  'monitor.unreachable': 'Le backend ne répond pas. L\'affichage est figé sur la dernière lecture réussie.',
  'monitor.depsTitle': 'État des services',
  'monitor.depsNote': 'Chaque lecture sonde réellement chaque dépendance une fois ; le délai par dépendance est de 2 s et les trois sondes sont parallèles.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'Moteur de recherche',
  'monitor.stateOk': 'Opérationnel',
  'monitor.stateDown': 'Injoignable',
  'monitor.stateDisabled': 'Non activé',
  'monitor.depSearchFallback': 'ES_URL n\'est pas défini, la recherche utilise donc une comparaison par mot-clé MySQL.',
  'monitor.depDisabled': 'Aucun client Redis n\'a été injecté, les fonctions média sont désactivées.',
  'monitor.depLatency': 'Réponse en {ms} ms',
  'monitor.depKeys': '{count} clés',
  'monitor.depMemory': 'Mémoire {size}',
  'monitor.depPoolUsage': 'Connexions {inUse}/{open} (max {max})',
  'monitor.depPoolWait': '{count} attentes, {ms} ms au total',
  'monitor.depRedisPool': 'Touches {hits} / échecs {misses}',
  'monitor.depEngineMysql': 'Comparaison par mot-clé MySQL',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'Vue d\'ensemble des requêtes',
  'monitor.statUptime': 'Temps de fonctionnement',
  'monitor.statRequests': 'Requêtes au total',
  'monitor.statErrorRate': 'Taux d\'erreur',
  'monitor.statP95': 'Latence P95',
  'monitor.statInFlight': 'Requêtes en cours',
  'monitor.statGoroutines': 'Goroutines',
  'monitor.statHeap': 'Mémoire du tas',
  'monitor.statDbPool': 'Connexions à la base',
  'monitor.statRateLimited': 'Bloqués par limitation',
  'monitor.statCountWithPeak': 'pic {peak}',
  'monitor.statCountWithInUse': '{inUse} en cours, {idle} inactives',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} cœurs logiques · {gc} cycles GC',
  'monitor.noData': 'Aucune requête pour l\'instant.',
  'monitor.noDataBody': 'Les requêtes traitées depuis le démarrage du service apparaîtront ici ; ce panneau est actuellement vide.',
  'monitor.timelineTitle': 'Débit sur les {minutes} dernières minutes',
  'monitor.timelineNote': 'Les totaux sont cumulés depuis le démarrage de ce processus et repartent de zéro après un redémarrage ; chaque centile de latence est une borne supérieure de seau d\'histogramme, donc seules des valeurs discrètes apparaissent. Une minute sans barre signifie qu\'il n\'y avait aucun trafic à ce moment-là.',
  'monitor.timelineLive': 'Ce processus',
  'monitor.timelineHistory': 'Avant redémarrage',
  'monitor.timelineLegendVolume': 'Requêtes',
  'monitor.timelineLegendError': 'Erreurs 5xx',
  'monitor.timelinePeak': 'Pic {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'Aucun historique agrégé n\'est disponible en base ; si le démarrage ne peut pas le lire (table absente ou droits insuffisants), seul le processus courant est affiché.',
  'monitor.routesTitle': 'Par route',
  'monitor.routesNote': 'Les chemins sont normalisés (les identifiants numériques et les adresses e-mail deviennent :id), donc les différents identifiants d\'une même route sont comptés ensemble.',
  'monitor.colRoute': 'Route',
  'monitor.colCount': 'Requêtes',
  'monitor.colAvg': 'Moyenne',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'Plus lente',
  'monitor.colErrors': 'Erreurs',
  'monitor.routeOther': 'Autres (limite de routes atteinte)',
  'monitor.limitsTitle': 'Limiteurs de débit',
  'monitor.limitsNote': 'Chaque groupe a son propre budget calé sur le coût de ses endpoints ; les blocages comptabilisent toutes les réponses 429 depuis le démarrage de ce processus.',
  'monitor.colLimiter': 'Limiteur',
  'monitor.colBudget': 'Budget',
  'monitor.colAllowed': 'Autorisées',
  'monitor.colBlocked': 'Bloquées',
  'monitor.colTracked': 'Sources suivies',
  'monitor.colBlockedRate': 'Taux de blocage',
  'monitor.limitContent': 'Écritures de contenu',
  'monitor.limitUpload': 'Envoi d’images',
  'monitor.limitAuth': 'Connexion OAuth',
  'monitor.limitBudget': '{limit} par {window} s',
  'monitor.limitUnknown': '(inconnu)',
  'monitor.noLimits': 'Aucun limiteur disponible.',
  'monitor.noLimitsBody': 'Les limiteurs de débit n\'ont pas encore été créés ; leurs compteurs sont indisponibles.',

  'log.title': 'Journal des actions',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'Consultez chaque modification faite dans l\'administration : qui, quand, sur quelle cible, et quels champs sont passés de quoi à quoi.',
  'log.refresh': 'Actualiser',
  'log.loadFailed': 'Échec du chargement du journal des actions.',
  'log.empty': 'Aucune action ne correspond aux filtres.',
  'log.emptyBody': 'Élargissez les filtres, ou confirmez qu\'aucune action n\'a eu lieu sur cette période.',
  'log.count': 'Entrées {from}–{to} sur {total}',
  'log.retention': 'Les enregistrements sont conservés {days} jours puis supprimés par une tâche de fond. Cette page n\'a aucun bouton pour supprimer une entrée : un journal d\'audit capable d\'effacer ses propres traces n\'est pas un journal d\'audit.',
  'log.filterActor': 'Auteur',
  'log.filterAction': 'Action',
  'log.filterTargetType': 'Type de ressource',
  'log.filterFrom': 'Du',
  'log.filterTo': 'Au',
  'log.filterAll': 'Tous',
  'log.filterApply': 'Appliquer les filtres',
  'log.filterReset': 'Effacer les filtres',
  'log.filterTargetHint': 'Cliquez sur la cible d\'une entrée pour n\'afficher que les actions qui la concernent.',
  'log.colTime': 'Heure',
  'log.colActor': 'Auteur',
  'log.colAction': 'Action',
  'log.colTarget': 'Cible',
  'log.colChanges': 'Modifications',
  'log.colOrigin': 'Origine',
  'log.noChanges': '(aucun champ modifié)',
  'log.changedTo': 'remplace par',
  'log.removed': '(supprimé)',
  'log.created': '(créé)',
  'log.requestId': 'request {id}',
  'log.page': 'Page {page}',
  'log.targetUser': 'Utilisateur',
  'log.targetPost': 'Message',
  'log.targetComment': 'Commentaire',
  'log.targetReport': 'Signalement',
  'log.targetTag': 'Étiquette',
  'log.targetSystem': 'Système',
  'log.actionUserSuspend': 'Compte suspendu',
  'log.actionUserReinstate': 'Compte réactivé',
  'log.actionUserTags': 'Étiquettes modifiées',
  'log.actionUserPost': 'Publication pour un utilisateur',
  'log.actionUserComment': 'Commentaire pour un utilisateur',
  'log.actionUserContent': 'Suppression de son contenu',
  'log.actionPostCreate': 'Message créé',
  'log.actionPostUpdate': 'Message modifié',
  'log.actionPostDelete': 'Message supprimé',
  'log.actionCommentCreate': 'Commentaire créé',
  'log.actionCommentUpdate': 'Commentaire modifié',
  'log.actionCommentDelete': 'Commentaire supprimé',
  'log.actionReportCreate': 'Signalement créé',
  'log.actionReportResolve': 'Signalement retenu',
  'log.actionReportReject': 'Signalement rejeté',
  'log.actionReportUpdate': 'Signalement modifié',
  'log.actionReportDelete': 'Signalement supprimé',
  'log.actionTagCreate': 'Étiquette créée',
  'log.actionTagUpdate': 'Étiquette renommée',
  'log.actionTagDelete': 'Étiquette supprimée',
  'log.fieldStatus': 'Statut du compte',
  'log.fieldContent': 'Contenu',
  'log.fieldName': 'Nom',
  'log.fieldTags': 'Étiquettes',
  'log.fieldReason': 'Motif',
  'log.fieldAuthorEmail': 'Auteur du contenu',
  'log.fieldReporterEmail': 'Auteur du signalement',
  'log.fieldTargetType': 'Type de ressource',
  'log.fieldTargetId': 'Identifiant',
  'log.fieldPostId': 'Id du message',
  'log.fieldCommentId': 'Id du commentaire',
  'log.fieldPostIdShort': 'Message',
  'log.fieldCommentIdShort': 'Commentaire',
  'log.fieldTarget': 'Cible',
  'log.fieldAssignmentsRemoved': 'Affectations supprimées',
  'log.truncated': 'tronqué',

  'stats.title': 'Tendances du contenu',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'Combien de nouveaux utilisateurs, messages et commentaires arrivent chaque jour, et quels messages, étiquettes et auteurs sont les plus actifs en ce moment.',
  'stats.refresh': 'Actualiser',
  'stats.loadFailed': 'Échec du chargement des statistiques de contenu.',
  'stats.window': 'Afficher les',
  'stats.windowDays': '{days} derniers jours',
  'stats.windowClamped': '(90 jours maximum)',
  'stats.windowNote': 'Chaque jour est délimité dans le fuseau horaire local de la machine serveur. Si le site tourne en UTC et que l\'administrateur est ailleurs, les chiffres du jour paraissent justes : c\'est le fuseau, pas une baisse du trafic.',
  'stats.generatedAt': 'Statistiques générées le {time}',
  'stats.seriesTitle': 'Nouveautés par jour',
  'stats.seriesNote': 'Les trois courbes ont des échelles indépendantes : elles sont affichées séparément plutôt que superposées.',
  'stats.seriesUsers': 'Nouveaux utilisateurs',
  'stats.seriesPosts': 'Nouveaux messages',
  'stats.seriesComments': 'Nouveaux commentaires',
  'stats.seriesEmpty': 'Aucune donnée sur cette période.',
  'stats.totalsTitle': 'Totaux de la période',
  'stats.totalsNote': 'Ce sont les quantités ajoutées sur cette période, pas les totaux actuels du site.',
  'stats.totalUsers': 'Nouveaux utilisateurs',
  'stats.totalPosts': 'Nouveaux messages',
  'stats.totalComments': 'Nouveaux commentaires',
  'stats.totalLikes': 'Nouveaux favoris',
  'stats.topPostsTitle': 'Messages populaires',
  'stats.topPostsNote': 'Classés par commentaires et favoris, en ne comptant que les messages publiés sur la période.',
  'stats.topTagsTitle': 'Étiquettes populaires',
  'stats.topTagsNote': 'Classées par nombre d\'utilisateurs, sans limite de durée : une étiquette est une qualité, pas un événement.',
  'stats.topAuthorsTitle': 'Auteurs actifs',
  'stats.topAuthorsNote': 'Classés par nombre de messages sur la période, le nombre de commentaires étant indiqué séparément.',
  'stats.colExcerpt': 'Extrait',
  'stats.colEngagement': 'Engagement',
  'stats.colPosts': 'Messages',
  'stats.colComments': 'Commentaires',
  'stats.colUsers': 'Utilisateurs',
  'stats.colAuthor': 'Auteur',
  'stats.empty': 'Aucune donnée sur cette période.',
  'stats.emptyBody': 'Élargissez la période, ou confirmez que rien de nouveau ne s\'est réellement produit.',
  'stats.engagement': '{comments} commentaires・{likes} favoris',
  'stats.rank': 'N° {rank}',

  'export.title': 'Export et actions groupées',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'Exportez les données du site en CSV pour les rapprocher, ou traitez plusieurs comptes d\'un coup.',
  'export.download': 'Télécharger le CSV',
  'export.downloading': 'Préparation…',
  'export.exportTitle': 'Export',
  'export.exportNote': 'Chaque export est limité à 50 000 lignes ; au-delà, seules les plus récentes sont incluses. Les colonnes des trois exports correspondent à celles des pages d\'administration, donc la comparaison est directe.',
  'export.exportUsers': 'Liste des utilisateurs',
  'export.exportUsersNote': 'E-mail, statut, dates de création et de mise à jour, nombre de messages et de commentaires.',
  'export.exportPosts': 'Liste des messages',
  'export.exportPostsNote': 'Id, auteur, 200 premiers caractères du contenu, date de création, commentaires et favoris.',
  'export.exportReports': 'Liste des signalements',
  'export.exportReportsNote': 'Id, cible signalée, auteur du signalement, motif, statut et trace de traitement.',
  'export.safety': 'Le fichier commence par un BOM UTF-8 : Excel l\'ouvre sans caractères illisibles.',
  'export.safetyPrefix': 'Les valeurs commençant par = + - @ ou par une espace invisible reçoivent une apostrophe en tête — c\'est ce qui oblige le tableur à les traiter comme du texte et non comme des formules. Ce préfixe est délibéré, ne demandez pas à le retirer.',
  'export.batchTitle': 'Actions groupées',
  'export.batchNote': 'Les boutons s\'activent après avoir sélectionné des comptes dans la page Utilisateurs. Une action groupée s\'applique entièrement ou pas du tout ; il n\'existe pas de résultat partiel.',
  'export.batchSuspend': 'Suspendre la sélection',
  'export.batchReinstate': 'Réactiver la sélection',
  'export.batchTags': 'Appliquer des étiquettes',
  'export.batchTagsNote': 'Sémantique d\'écrasement : la liste envoyée devient le résultat. Une liste vide supprime toutes les étiquettes.',
  'export.batchConfirm': 'Appliquer « {action} » à {count} comptes ?',
  'export.batchConfirmTags': 'Écraser les étiquettes de {count} comptes par {tags} ?',
  'export.batchTagsPicker': 'Choisir des étiquettes',
  'export.batchTagsNone': 'Aucune étiquette (tout supprimer)',
  'export.batchRunning': 'Traitement…',
  'export.batchDone': '{updated} comptes mis à jour',
  'export.batchDoneUnchanged': 'dont {unchanged} étaient déjà dans l\'état cible et n\'ont pas changé',
  'export.batchSkipped': '{count} ignorés',
  'export.batchMax': '200 comptes maximum par lot',
  'export.gotoUsers': 'Aller à Utilisateurs',
  'export.noSelection': 'Sélectionnez d\'abord des comptes dans la page Utilisateurs.',
  'export.selected': '{count} comptes sélectionnés',
  'export.clearSelection': 'Effacer la sélection',
  'export.selectionHint': 'La sélection est conservée sur cette page et disparaît à sa fermeture.',

  'session.title': 'Connexions et sessions',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'Voyez quelles connexions sont encore valides et forcez la déconnexion d\'un compte sur tous ses appareils.',
  'session.refresh': 'Actualiser',
  'session.loadFailed': 'Échec du chargement de la liste des sessions.',
  'session.privacyTitle': 'Pourquoi le jeton complet n\'est pas affiché',
  'session.privacyNote': 'Le jeton est lui-même l\'identifiant de connexion. Seuls les huit premiers caractères sont affichés, pour qu\'un administrateur puisse reconnaître deux lignes comme la même session ; cela ne suffit à personne pour se connecter à la place de l\'utilisateur — pas même à celui qui reçoit une capture d\'écran de cette page. C\'est une limite délibérée, pas une fonctionnalité inachevée.',
  'session.expireNote': 'Une session expire après {hours} heures sans activité ; toute requête la prolonge.',
  'session.filterEmail': 'Filtrer par compte',
  'session.filterPlaceholder': 'Adresse e-mail complète',
  'session.search': 'Rechercher',
  'session.clearFilter': 'Effacer',
  'session.summary': '{total} sessions sur le site, {scanned} clés examinées',
  'session.truncated': 'L\'analyse a atteint sa limite de {scanned} clés et s\'est arrêtée tôt : cette liste est incomplète.',
  'session.empty': 'Aucune session pour le moment.',
  'session.emptyBody': 'Personne n\'est connecté, ou toutes les sessions ont expiré.',
  'session.colUser': 'Compte',
  'session.colToken': 'Session',
  'session.colCreated': 'Créée',
  'session.colExpires': 'Expire',
  'session.colRemaining': 'Restant',
  'session.colActions': 'Action',
  'session.unknown': 'Inconnu',
  'session.adminBadge': 'Administrateur',
  'session.revoke': 'Forcer la déconnexion',
  'session.revokeTitle': 'Forcer la déconnexion de {email}',
  'session.revokeMessage': 'Les {count} sessions actuelles du compte seront invalidées immédiatement et la connexion effacée sur tous les appareils. L\'utilisateur devra se reconnecter. Continuer ?',
  'session.revokeRunning': 'Révocation…',
  'session.revokeDone': '{count} sessions révoquées',
  'session.revokeNone': 'Ce compte n\'a aucune session active',
  'session.revokeFailed': 'Impossible de confirmer la déconnexion.',
  'session.revokeUnavailable': 'Cela ne signifie pas que la révocation a échoué : l\'analyse a atteint sa limite de clés, donc certaines sessions peuvent n\'avoir pas été atteintes. Réessayez dans un instant.',
  'session.titleColumnNote': 'Préfixe d\'identification seulement, inutilisable pour se connecter',

  'block.title': 'Liste de blocage d\'IP',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'Mettez sur la liste les adresses dont l\'abus est confirmé. La liste vit dans Redis : ni un redémarrage ni un déploiement ne l\'efface.',
  'block.refresh': 'Actualiser',
  'block.loadFailed': 'Échec du chargement de la liste de blocage.',
  'block.unavailable': 'Ce site n\'a pas de connexion Redis : le blocage n\'est pas activé.',
  'block.unavailableNote': 'La liste a besoin du même Redis que les sessions et les jetons de médias. Une fois connectée, cette page fonctionnera ; jusque-là la seule protection est la limitation de débit (par processus, réinitialisée au redémarrage).',
  'block.add': 'Bloquer',
  'block.addTitle': 'Bloquer une adresse IP',
  'block.addMessage': 'Une adresse bloquée est refusée sur toute requête d\'écriture (messages, commentaires, favoris, signalements, envois d\'images, redirections de connexion) jusqu\'à l\'expiration. La lecture n\'est pas affectée.',
  'block.ipLabel': 'Adresse IP',
  'block.ipPlaceholder': '203.0.113.9 ou 2001:db8::1',
  'block.durationLabel': 'Durée',
  'block.reasonLabel': 'Motif',
  'block.reasonPlaceholder': 'Pourquoi cette adresse est bloquée (consigné dans le journal d\'audit)',
  'block.reasonHint': 'Le motif n\'alimente que le journal d\'audit. Il n\'est jamais montré à la personne bloquée ni dans un message d\'erreur public.',
  'block.blocking': 'Blocage…',
  'block.done': '{ip} bloquée',
  'block.removed': '{ip} débloquée',
  'block.removedNone': '{ip} n\'était pas bloquée',
  'block.failed': 'L\'opération de blocage a échoué.',
  'block.unavailableService': 'Liste de blocage indisponible (pas de Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'Expire',
  'block.colRemaining': 'Restant',
  'block.colActions': 'Action',
  'block.unblock': 'Débloquer',
  'block.unblockTitle': 'Débloquer {ip}',
  'block.unblockMessage': 'Cette adresse retrouve immédiatement son accès normal en écriture. Continuer ?',
  'block.empty': 'La liste de blocage est vide.',
  'block.emptyBody': 'Aucune adresse n\'est bloquée. C\'est l\'état normal : le blocage est toujours une décision d\'un administrateur, et le système ne bloque jamais personne automatiquement.',
  'block.notAutoNote': 'Cette liste ne se remplit jamais d\'elle-même. Une adresse qui dépasse sa limite de débit ne reçoit qu\'un 429 ; elle n\'est pas ajoutée ici automatiquement, car la même sortie peut appartenir à tout un bureau ou tout un NAT et un blocage automatique les toucherait aussi.',
  'block.scopeNoteLabel': 'Portée',
  'block.notAutoNoteLabel': 'Jamais automatique',
  'block.maxNoteLabel': 'Durée maximale',
  'block.scopeNote': 'Le blocage n\'arrête que les requêtes d\'écriture. Lire les messages, les commentaires et les ressources statiques reste possible, et une personne bloquée peut encore se connecter et lire : c\'est délibéré, les points de terminaison de lecture ne sont volontairement pas limités (sinon les visiteurs anonymes ne pourraient pas utiliser le site), et le blocage couvre le même ensemble.',
  'block.maxNote': 'Un blocage dure au plus 365 jours. Une durée plus longue est ramenée à un an : l\'expiration est stockée sous forme de nombre, et « définitivement » deviendrait un blocage dont personne ne se souvient et qui ne se lève jamais tout seul.',
  'block.count': '{count} adresses bloquées',
  'block.ipInvalid': 'Adresse IP invalide. Saisissez une adresse IPv4 ou IPv6 ; les plages CIDR ne sont pas prises en charge.',
  'announce.label': 'Annonce du site',
  'announce.publicNote': 'Avis',
  'announce.closeAria': 'Fermer cette annonce',
  'announce.publishedOn': 'Publiée le {date}',
  'announce.expiresOn': 'Expire le {date}',
  'announce.neverExpires': 'Sans expiration',
  'announce.pinnedBadge': 'Épinglé',
  'announce.title': 'Annonces',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'Afficher une annonce en haut de toutes les pages du site. Une seule est active à la fois : en publier une nouvelle, l\'ancienne est désactivée.',
  'announce.refresh': 'Actualiser',
  'announce.loadFailed': 'Échec du chargement des annonces.',
  'announce.new': 'Publier une nouvelle annonce',
  'announce.edit': 'Modifier',
  'announce.deactivate': 'Désactiver',
  'announce.reactivate': 'Réactiver',
  'announce.deleteNote': 'Les annonces ne sont jamais supprimées, seulement désactivées : garder l\'historique permet de répondre quand elle a été publiée et par qui.',
  'announce.bodyLabel': 'Texte de l\'annonce',
  'announce.bodyPlaceholder': 'Par exemple : le système sera en maintenance jeudi de 02:00 à 04:00.',
  'announce.bodyHint': '300 caractères maximum. Texte brut, les retours à la ligne sont conservés.',
  'announce.activeLabel': 'Afficher immédiatement',
  'announce.expiryLabel': 'Validité',
  'announce.expiryNever': 'N\'expire jamais automatiquement',
  'announce.expiryHours': 'dans {hours} heures',
  'announce.expiryDays': 'dans {days} jours',
  'announce.saving': 'Enregistrement…',
  'announce.published': 'Annonce publiée',
  'announce.updated': 'Annonce mise à jour',
  'announce.deactivated': 'Annonce désactivée',
  'announce.reactivated': 'Annonce réactivée',
  'announce.saveFailed': 'L\'opération a échoué.',
  'announce.empty': 'Aucune annonce pour l\'instant.',
  'announce.emptyBody': 'Dès que vous en publiez une, elle apparaîtra en haut de la page de chaque visiteur.',
  'announce.colBody': 'Texte',
  'announce.colState': 'État',
  'announce.colAuthor': 'Publiée par',
  'announce.colCreated': 'Publiée le',
  'announce.colActions': 'Action',
  'announce.stateActive': 'Affichée',
  'announce.stateInactive': 'Désactivée',
  'announce.stateExpired': 'Expirée',
  'announce.confirmDeactivate': 'La publication désactivera l\'annonce actuelle, et chaque verra immédiatement le nouveau texte. Continuer ?',
  'announce.confirmEdit': 'Modifier le texte ou la validité de cette annonce ?',
  'announce.count': '{count} au total',
  'users.title':
    'Gestion des utilisateurs',
  'users.contentAction':
    'Contenu',
  'users.updateContentFailed':
    'Échec de l’action sur le contenu.',
  'users.updated':
    'Contenu mis à jour.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'Consultez les statistiques d’activité des utilisateurs du forum, les Tag et l’état des comptes, puis gérez les suspensions et la modération du contenu, article par article.',
  'users.refresh':
    'Actualiser les données',
  'users.statTotal':
    'Nombre total d’utilisateurs',
  'users.statActive':
    'Actifs',
  'users.statSuspended':
    'Suspendus',
  'users.statContent':
    'Total des articles et des commentaires',
  'users.count':
    '{count} utilisateurs',
  'users.tagsCount':
    '{count} Tag',
  'users.loadFailed':
    'Échec du chargement des données des utilisateurs.',
  'users.tagsLoadFailed':
    'Échec du chargement des données des Tag.',
  'users.panelTitle':
    'Utilisateurs du forum',
  'users.tagsPanelTitle':
    'Tag des utilisateurs',
  'users.colUser':
    'Utilisateur',
  'users.colTags':
    'Tag',
  'users.colStatus':
    'Statut',
  'users.colPosts':
    'Articles',
  'users.colComments':
    'Commentaires',
  'users.colLikes':
    'J\'aime',
  'users.colLastActivity':
    'Dernière activité',
  'users.colActions':
    'Actions',
  'users.nicknameUnset':
    'Pseudo non défini',
  'users.notSet':
    'Non défini',
  'users.statusActive':
    'Actif',
  'users.statusSuspended':
    'Suspendu',
  'users.emptyTitle':
    'Aucune donnée utilisateur n’est actuellement disponible',
  'users.emptyBody':
    'Ce panneau n’est vide que si aucun utilisateur ne s’est connecté au forum avec Google.',
  'users.tagsEmptyTitle':
    'Aucun Tag n’est actuellement disponible',
  'users.tagsEmptyBody':
    'Créez d’abord un Tag pour l’appliquer à un utilisateur dans la liste.',
  'users.addTag':
    'Ajouter un Tag',
  'users.colName':
    'Nom',
  'users.colCreated':
    'Date de création',
  'users.colUpdated':
    'Date de mise à jour',
  'users.renameTag':
    'Renommer',
  'users.suspend':
    'Suspendre',
  'users.restore':
    'Rétablir',
  'users.statusDialogTitle':
    '{action} ce utilisateur',
  'users.statusSuspendMessage':
    '{email} ne pourra plus se connecter au forum. Les articles et les commentaires existants seront conservés. Voulez-vous continuer ?',
  'users.statusRestoreMessage':
    '{email} récupérera ses permissions de connexion et de publication. Voulez-vous continuer ?',
  'users.userSuspended':
    'Utilisateur suspendu.',
  'users.userRestored':
    'Utilisateur rétabli.',
  'users.updateStatusFailed':
    'Échec de la mise à jour du statut de l’utilisateur.',
  'users.editTagsTitle':
    'Modifier les Tag · {user}',
  'users.editTagsMessage':
    'Sélectionnez les Tag à appliquer ; les désélectionner tous revient à supprimer tous les Tag de cet utilisateur.',
  'users.tagsUpdated':
    'Les Tag de l’utilisateur ont été mis à jour.',
  'users.updateTagsFailed':
    'Échec de la mise à jour des Tag de l’utilisateur.',
  'users.contentLoadFailed':
    'Échec du chargement du contenu.',
  'users.contentLoadFailedShort':
    'Échec du chargement du contenu.',
  'users.contentPanelTitle':
    'Contenu de l’utilisateur',
  'users.contentCount':
    '{posts} articles · {comments} commentaires',
  'users.contentLoading':
    'Chargement des articles et des commentaires…',
  'users.addPost':
    'Ajouter un article',
  'users.addComment':
    'Ajouter un commentaire',
  'users.postsColumn':
    'Articles',
  'users.commentsColumn':
    'Commentaires',
  'users.noPosts':
    'Aucun article',
  'users.noComments':
    'Aucun commentaire',
  'users.postRef':
    'Article #{id}',
  'users.editRecordTitle':
    'Modifier {kind} #{id}',
  'users.deleteRecordTitle':
    'Supprimer {kind} #{id}',
  'users.deleteRecordMessage':
    'La suppression est irréversible : les J’aime associés ainsi que les données associées seront également supprimés. Voulez-vous continuer ?',
  'users.contentLabel':
    'Contenu',
  'users.addPostTitle':
    'Ajouter un article',
  'users.addPostMessage':
    'Ce contenu sera publié au nom de cet utilisateur ; l’identité de l’auteur ne peut pas être falsifiée.',
  'users.postContentLabel':
    'Contenu de l’article',
  'users.postContentPlaceholder':
    'Saisir le contenu de l’article',
  'users.pickPostTitle':
    'Choisir un article',
  'users.postIdLabel':
    'ID de l’article',
  'users.postIdPlaceholder':
    'Numéro de l’article à commenter',
  'users.addCommentTitle':
    'Ajouter un commentaire',
  'users.commentContentLabel':
    'Contenu du commentaire',
  'users.commentContentPlaceholder':
    'Saisir le contenu du commentaire',
  'users.createTagTitle':
    'Ajouter un Tag',
  'users.createTagMessage':
    'Les Tag peuvent servir à classer les utilisateurs, par exemple « Modérateur », « Actif » ou « Banni ».',
  'users.tagNameLabel':
    'Nom du Tag',
  'users.tagNamePlaceholder':
    'Jusqu’à 50 caractères',
  'users.renameTagTitle':
    'Renommer un Tag',
  'users.renameTagMessage':
    'Tous les utilisateurs qui utilisent ce Tag verront le nouveau nom.',
  'users.deleteTagTitle':
    'Supprimer le Tag « {name} »',
  'users.deleteTagMessage':
    'Après sa suppression, ce Tag sera retiré de tous les utilisateurs et ne pourra pas être rétabli. Voulez-vous continuer ?',
  'users.deleteTagConfirm':
    'Supprimer un Tag',
  'users.tagCreated':
    'Tag créé.',
  'users.createTagFailed':
    'Échec de la création du Tag.',
  'users.tagUpdated':
    'Tag mis à jour.',
  'users.updateTagFailed':
    'Échec de la mise à jour du Tag.',
  'users.tagDeleted':
    'Tag supprimé.',
  'users.deleteTagFailed':
    'Échec de la suppression du Tag.',
  'users.refreshDone':
    'Données actualisées.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'Administration',
  'users.signinBody':
    'Une gouvernance centralisée du contenu pour que chaque revue soit claire, rapide et traçable.',
  'users.signinStep1':
    'Vérification de sécurité',
  'users.signinStep2':
    'Gestion des utilisateurs',
  'users.signinStep3':
    'Examen du contenu',
  'users.signinPanelTitle':
    'Se connecter à Admin Console',
  'users.signinPanelBody':
    'Admin Console est réservé aux comptes administrateurs Google autorisés. Veuillez vous connecter en tant qu’administrateur.',
  'posts.title':
    'Articles de forum',
  'posts.searching':
    'Recherche en cours…',
  'posts.searchDegraded':
    '(Le service de recherche n’est pas activé ; recherche par mots-clés dans la base de données)',
  'posts.searchSummary':
    'Recherche de « {query} » : {total} résultat(s) trouvé(s), {shown} affiché(s) sur cette page',
  'posts.pendingCount':
    '{count} à traiter',
  'posts.pageSummary':
    'Page {page} / {pages}, {count} articles sur cette page',
  'posts.listLoadFailed':
    'Échec du chargement de la liste des articles.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'Créez, modifiez et supprimez des articles de forum, et gérez le contenu au niveau des commentaires.',
  'posts.toReports':
    'Gestion des signalements',
  'posts.editorTitleNew':
    'Ajouter un article',
  'posts.editorTitleEdit':
    'Modifier l’article #{id}',
  'posts.editorNote':
    'Publié en tant qu’administrateur ; l’auteur provient de l’identité connectée, et l’identité de l’auteur ne peut pas être falsifiée dans la requête.',
  'posts.cancelEdit':
    'Annuler la modification',
  'posts.contentLabel':
    'Contenu de l’article',
  'posts.contentPlaceholder':
    'Saisir le contenu de l’article',
  'posts.saveChanges':
    'Enregistrer les modifications',
  'posts.emptyContent':
    'Le contenu de l’article ne peut pas être vide.',
  'posts.saving':
    'Enregistrement en cours…',
  'posts.saved':
    'Article mis à jour.',
  'posts.published':
    'Article publié.',
  'posts.saveFailed':
    'Échec de l’enregistrement.',
  'posts.deleteTitle':
    'Supprimer l’article #{id}',
  'posts.deleteMessage':
    'La suppression est irréversible : tous les commentaires sous cet article seront également supprimés. Voulez-vous continuer ?',
  'posts.deleteConfirm':
    'Supprimer l’article',
  'posts.deleted':
    'Article supprimé.',
  'posts.deleteFailed':
    'Échec de la suppression de l’article.',
  'posts.commentUpdated':
    'Commentaire mis à jour.',
  'posts.commentActionFailed':
    'Échec de l’action sur le commentaire.',
  'posts.pickPostTitle':
    'Choisir l’article auquel le commentaire est associé',
  'posts.pickPostMessage':
    'L’article de cette ligne est utilisé par défaut ; pour l’associer à un autre article, modifiez son numéro.',
  'posts.postIdLabel':
    'ID de l’article',
  'posts.addCommentAtTitle':
    'Ajouter un commentaire à l’article #{id}',
  'posts.addCommentMessage':
    'Ce commentaire sera publié en tant qu’administrateur.',
  'posts.commentContentLabel':
    'Contenu du commentaire',
  'posts.commentContentPlaceholder':
    'Saisir le contenu du commentaire',
  'posts.add':
    'Ajouter',
  'posts.editCommentTitle':
    'Modifier le commentaire #{id}',
  'posts.deleteCommentTitle':
    'Supprimer le commentaire #{id}',
  'posts.deleteCommentMessage':
    'La suppression est irréversible. Voulez-vous continuer ?',
  'posts.listTitle':
    'Liste des articles',
  'posts.clearSearch':
    'Effacer la recherche',
  'posts.searchLabel':
    'Recherche',
  'posts.searchPlaceholder':
    'Rechercher dans le contenu des messages ou l’adresse Email complète de l’auteur du message',
  'posts.searchHint':
    'Tries selon leur pertinence ; saisissez une adresse Email complète pour trouver tous les messages de cet utilisateur. La recherche remplace la pagination, et les résultats affichent au maximum 25 messages.',
  'posts.searchTotal':
    'Total de {total} résultats de recherche',
  'posts.searchFailed':
    'Échec de la recherche.',
  'posts.searchStatusFailed':
    'Échec de la recherche',
  'posts.colContentImage':
    'Contenu et images',
  'posts.colEngagement':
    'Interactions',
  'posts.colComments':
    'Commentaires',
  'posts.colAuthor':
    'Auteur',
  'posts.imageAlt':
    'Image de l’article',
  'posts.likes':
    '{count} J\'aime',
  'posts.author':
    'Auteur : {name}',
  'posts.emptyTitle':
    'Aucun message de forum pour le moment',
  'posts.emptyBody':
    'Vous pouvez créer votre premier message à l’aide de l’éditeur ci-dessus.',
  'posts.emptySearchTitle':
    'Aucun message correspondant',
  'posts.emptySearchBody':
    'Aucun message ne correspond à « {query} ». Essayez un autre mot-clé.',
  'posts.commentCount':
    '{count} commentaires',
  'posts.noComments':
    'Aucun commentaire',
  'posts.reportsTitle':
    'Signalements en attente',
  'posts.allReports':
    'Tous les signalements',
  'posts.colReportedContent':
    'Contenu signalé',
  'posts.colReason':
    'Motif du signalement',
  'posts.colReporter':
    'Auteur du signalement',
  'posts.colTime':
    'Date',
  'posts.colVerdict':
    'Décision',
  'posts.emptyReportsTitle':
    'Aucun signalement en attente',
  'posts.emptyReportsBody':
    'Tous les signalements ont été traités.',
  'posts.verdictResolved':
    'Traités',
  'posts.verdictRejected':
    'Non fondé',
  'posts.verdictDialogTitle':
    'Marquer le signalement #{id} comme « {label} »',
  'posts.verdictDialogMessage':
    'Après marquage, le signalement quittera la liste en attente, mais les données resteront sur la page de gestion des signalements. Continuer ?',
  'posts.verdictConfirm':
    'Marquer comme {label}',
  'posts.verdictDone':
    'Le signalement a été marqué comme {label}.',
  'posts.verdictFailed':
    'Échec de la mise à jour du statut du signalement.',
  'posts.pin': 'Épingler',
  'posts.unpin': 'Désépingler',
  'posts.pinTitle': 'Épingler ce message',
  'posts.unpinTitle': 'Désépingler ce message',
  'posts.pinMessage': 'Une fois épinglé, ce message reste tout en haut du flux de chacun, et les nouveaux messages ne peuvent pas le faire descendre.',
  'posts.unpinMessage': 'La désépinglage ramène ce message à sa place dans l’ordre chronologique.',
  'posts.pinDone': 'Épinglé',
  'posts.unpinDone': 'Désépinglé',
  'posts.pinFailed': 'L\'opération d\'épinglage a échoué.',
  'reports.title':
    'Gestion des signalements',
  'reports.listSummary':
    '{count} · {filter}',
  'reports.listLoadFailed':
    'Échec du chargement de la liste des signalements.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'Décider des signalements un par un : « Valider » supprime le contenu signalé ; « Non fondé » le conserve. Le retour à « En attente » efface la date de décision existante.',
  'reports.backToPosts':
    'Retour aux messages',
  'reports.editorTitle':
    'Modifier le signalement #{id}',
  'reports.editorNote':
    'Vous pouvez corriger le motif et le statut du signalement ; la cible et l’auteur du signalement sont des données existantes et ne sont pas modifiées ici.',
  'reports.targetTypeLabel':
    'Type de cible',
  'reports.targetIdLabel':
    'ID de cible',
  'reports.reporterEmailLabel':
    'Email de l’auteur du signalement',
  'reports.statusLabel':
    'Statut',
  'reports.reasonLabel':
    'Motif du signalement',
  'reports.reasonHint':
    'Au maximum 500 caractères ; il sera affiché directement aux autres administrateurs à titre d’élément de décision.',
  'reports.filterLabel':
    'Filtrer par statut',
  'reports.filterAll':
    'Tout',
  'reports.listTitle':
    'Liste des signalements',
  'reports.colTarget':
    'Contenu ciblé',
  'reports.colReason':
    'Motif du signalement',
  'reports.colReporter':
    'Auteur du signalement',
  'reports.colStatus':
    'Statut',
  'reports.targetGone':
    '(Le contenu a été supprimé)',
  'reports.author':
    'Auteur : {name}',
  'reports.deleteTitle':
    'Supprimer le signalement #{id}',
  'reports.deleteMessage':
    'La suppression concerne la fiche de signalement elle-même ; le contenu signalé n’est pas affecté et ne peut pas être rétabli. Continuer ?',
  'reports.deleteConfirm':
    'Supprimer le signalement',
  'reports.deleted':
    'Le signalement a été supprimé.',
  'reports.deleteFailed':
    'Échec de la suppression du signalement.',
  'reports.updated':
    'Le signalement a été mis à jour.',
  'reports.saveFailed':
    'Échec de l’enregistrement du signalement.',
  'reports.approveTitle':
    'Valider le signalement #{id}',
  'reports.approveGoneMessage':
    'Le {kind} #{id} signalé n’existe plus ; celui-ci sera uniquement marqué comme traité.',
  'reports.approveMessage':
    'La suppression définitive du {kind} #{id} signalé (et, s’il s’agit d’un message, de tous ses commentaires) sera effectuée, puis ce signalement sera marqué comme traité. Continuer ?',
  'reports.approveConfirm':
    'Valider et supprimer le message',
  'reports.approveGoneDone':
    'Le contenu n’existe plus ; le signalement a été marqué comme traité.',
  'reports.approveDone':
    'Le message a été supprimé et le signalement marqué comme traité.',
  'reports.approveFailed':
    'Échec de la validation du signalement.',
  'reports.approveTitleGone':
    'Le contenu a été supprimé ; seul le signalement sera marqué',
  'reports.approveTitleFull':
    'Supprimer le contenu signalé et le marquer comme traité',
  'reports.rejectTitle':
    'Le signalement #{id} est non fondé',
  'reports.rejectMessage':
    'Non fondé signifie que le contenu signalé n’a pas à être traité ; il sera conservé à l’identique. Continuer ?',
  'reports.rejectConfirm':
    'Marquer comme non fondé',
  'reports.rejectDone':
    'Le signalement a été marqué comme non fondé.',
  'reports.statusFailed':
    'Échec de la mise à jour du statut du signalement.',
  'reports.emptyTitle':
    'Aucun signalement pour le moment',
  'reports.emptyBody':
    'Aucune fiche ne correspond à ce filtre.',
  'reports.rejectTitleAttr':
    'Conserver le contenu et marquer uniquement le signalement comme non fondé',
  'kind.post':
    'Article',
  'kind.comment':
    'Commentaire',
  'reports.statusPending':
    'En attente',
  'reports.statusResolved':
    'Traités',
  'reports.statusRejected':
    'Non fondé',
  'title.forum':
    '{site}',
  'title.login':
    'Connexion | {site}',
  'title.newPost':
    'Nouveau message | {site}',
  'title.profile':
    'Profil personnel | {site}',
  'title.publicProfile':
    'Profil public | {site}',
  'title.following':
    'Abonnements | {site}',
  'title.adminUsers':
    'Gestion des utilisateurs | Back-office {brand}',
  'title.adminLogin':
    'Connexion | Back-office {brand}',
  'title.adminPosts':
    'Messages du forum | Back-office {brand}',
  'title.adminReports':
    'Gestion des signalements | Back-office {brand}',
  'title.adminMonitor': 'Supervision du système｜{brand} Admin',
  'title.adminLog': 'Journal des actions｜{brand} Admin',
  'title.adminStats': 'Tendances du contenu｜{brand} Admin',
  'title.adminExport': 'Export et actions groupées｜{brand} Admin',
  'title.adminSessions': 'Connexions et sessions｜{brand} Admin',
  'title.adminBlocks': 'Liste de blocage d\'IP｜{brand} Admin',
  'title.adminAnnouncements': 'Annonces｜{brand} Admin',
};
