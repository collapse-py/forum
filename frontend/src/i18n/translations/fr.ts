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
};
