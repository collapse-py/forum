/*
 * es catalog (src/i18n/translations/es.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const es: Record<MessageKey, string> = {
  'common.cancel':
    'Cancelar',
  'common.save':
    'Guardar',
  'common.submitting':
    'Enviando…',
  'common.delete':
    'Eliminar',
  'common.edit':
    'Editar',
  'common.search':
    'Buscar',
  'common.loading':
    'Cargando…',
  'common.loadFailed':
    'Error al cargar',
  'common.refresh':
    'Actualizar',
  'common.nextStep':
    'Siguiente paso',
  'common.prevPage':
    'Página anterior',
  'common.nextPage':
    'Página siguiente',
  'common.create':
    'Crear',
  'common.publish':
    'Publicar',
  'common.placeholder':
    '—',
  'common.backToHome':
    'Volver a la página principal',
  'common.backToForumHome':
    'Volver a la página principal del foro',
  'common.backOnePage':
    'Volver a la página anterior',
  'error.request':
    'Intente de nuevo más tarde.',
  'error.requestStatus':
    'Error de la solicitud (HTTP {status})',
  'error.loginRequired':
    'Inicie sesión antes de continuar.',
  'error.adminSessionExpired':
    'La sesión ha caducado; se redirigirá a la página de inicio de sesión.',
  'error.fallbackLoad':
    'Error al cargar',
  'error.fallbackSearch':
    'Error al buscar',
  'error.fallbackLike':
    'Error al dar «Me gusta»',
  'error.fallbackComments':
    'Error al cargar los comentarios',
  'error.fallbackCommentPost':
    'Error al publicar el comentario',
  'error.fallbackReport':
    'Error al presentar la denuncia',
  'error.fallbackProfile':
    'Error al cargar el perfil de la persona',
  'error.fallbackProfileSave':
    'Error al guardar',
  'error.fallbackPublish':
    'El formato de la respuesta al publicar la publicación es incorrecto.',
  'error.fallbackUpload':
    'El formato de la respuesta al subir la imagen es incorrecto.',
  'error.fallbackNotFound':
    'No se encontró esta persona.',
  'error.fallbackFollow':
    'No se pudo seguir.',
  'auth.checking':
    'Comprobando el inicio de sesión…',
  'auth.statusUnknown':
    'No se pudo confirmar el estado de inicio de sesión.',
  'auth.feedLoggedIn':
    'Sesión iniciada; se puede publicar.',
  'auth.feedLoggedOut':
    'Inicie sesión para publicar.',
  'auth.profileLoggedIn':
    'Sesión iniciada.',
  'auth.profileLoggedOut':
    'Inicie sesión para configurar su biografía.',
  'auth.googleLogin':
    'Inicio de sesión con Google',
  'auth.loginWithGoogle':
    'Iniciar sesión con una cuenta de Google',
  'auth.loginWithGoogleAdmin':
    'Iniciar sesión con una cuenta de administrador de Google',
  'auth.logout':
    'Cerrar sesión',
  'install.button':
    'Instalar la aplicación',
  'install.hint':
    'El navegador actual no ofrece una sugerencia de instalación automática. Abra el menú del navegador y seleccione «Instalar aplicación» o «Agregar a la pantalla principal».',
  'bottomNav.label':
    'Navegación principal',
  'bottomNav.home':
    'Inicio',
  'bottomNav.new':
    'Nuevo',
  'bottomNav.profile':
    'Perfil',
  'i18n.ariaLabel':
    'Seleccionar idioma',
  'i18n.current':
    'Idioma: {name}',
  'feed.searchPlaceholder':
    'Buscar en el contenido de las publicaciones',
  'feed.searchAriaLabel':
    'Buscar publicaciones',
  'feed.searchResultsLabel':
    'Resultados de búsqueda',
  'feed.postsLabel':
    'Publicaciones del foro',
  'feed.searchFailed':
    'Error al buscar. Intente de nuevo más tarde.',
  'feed.searching':
    'Buscando…',
  'feed.searchMore':
    'Cargando más resultados de búsqueda…',
  'feed.searchMoreFailed':
    'Error al cargar más resultados',
  'feed.searchFound':
    'Se encontraron {total}',
  'feed.searchDegraded':
    '{base} (el servicio de búsqueda no está habilitado; actualmente se comparan palabras clave de la base de datos).',
  'feed.searchTotal':
    'Se encontraron {total} resultados en total',
  'feed.searchNoResults':
    'No se encontraron publicaciones que contengan «{query}».',
  'feed.loadingPosts':
    'Cargando publicaciones…',
  'feed.loadMorePosts':
    'Cargando más publicaciones…',
  'feed.postsFailed':
    'Error al cargar publicaciones. Intente de nuevo más tarde.',
  'feed.postsFailedShort':
    'Error al cargar. Intente de nuevo.',
  'feed.scrollMore':
    'Deslice hacia abajo para cargar más',
  'feed.endOfFeed':
    'Ya llegó al final',
  'feed.noPosts':
    'Todavía no hay publicaciones. Deja tu primera idea.',
  'feed.likeFailed':
    'Error al dar «Me gusta» o a «Ya no me gusta». Intente de nuevo más tarde.',
  'post.authorAnonymous':
    'Anónimo',
  'post.report':
    'Denunciar publicación',
  'post.imageAlt':
    'Imagen de la publicación',
  'post.unlike':
    'Ya no me gusta',
  'post.like':
    'Me gusta',
  'post.reply':
    'Responder',
  'comment.loading':
    'Cargando comentarios…',
  'comment.none':
    'Aún no hay comentarios',
  'comment.loadFailed':
    'Error al cargar comentarios. Intente de nuevo más tarde',
  'comment.placeholder':
    'Escribe un comentario…',
  'comment.max':
    'Máximo 2000 caracteres',
  'comment.submit':
    'Comentario',
  'comment.failed':
    'Error al publicar el comentario. Intente de nuevo más tarde.',
  'comment.report':
    'Denunciar',
  'comment.more':
    'Cargando más comentarios…',
  'report.reasonPlaceholder':
    'Introduzca el motivo de la denuncia (máximo 500 caracteres)',
  'report.note':
    'La denuncia se enviará a los administradores del sitio.',
  'report.formLabel':
    'Campo de denuncia',
  'report.submit':
    'Enviar denuncia',
  'report.failed':
    'Error al presentar la denuncia. Intente de nuevo más tarde.',
  'report.sent':
    'La denuncia se ha enviado. Gracias por su informe.',
  'newPost.avatarYou':
    'Tú',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'Nueva publicación',
  'newPost.loginFirst':
    'Inicie sesión con Google antes de publicar',
  'newPost.contentPlaceholder':
    'Comparte tu idea…',
  'newPost.addImage':
    'Agregar imagen',
  'newPost.emailPrivate':
    'Tu email no se hará público',
  'newPost.submit':
    'Publicar',
  'newPost.publishing':
    'Publicando…',
  'newPost.uploading':
    'Subiendo imagen…',
  'newPost.failed':
    'No se pudo publicar. Inténtalo de nuevo más tarde.',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'Datos personales',
  'profile.edit':
    'Editar',
  'profile.loginPrompt':
    'Inicia sesión para configurar el alias y la biografía del foro.',
  'profile.nicknameLabel':
    'Alias del foro',
  'profile.notSet':
    'No configurado',
  'profile.notSetBio':
    'No se ha configurado la biografía.',
  'profile.nicknameInput':
    'Alias',
  'profile.nicknamePlaceholder':
    'Introducir alias',
  'profile.nicknameHint':
    'El alias se mostrará en las publicaciones que hagas, con un máximo de 30 caracteres.',
  'profile.bioLabel':
    'Biografía',
  'profile.bioPlaceholder':
    'Preséntate (opcional)',
  'profile.bioHint':
    'Máximo 500 caracteres.',
  'profile.updated':
    'Datos personales actualizados.',
  'profile.saving':
    'Guardando…',
  'profile.saveFailed':
    'No se pudo guardar. Inténtalo de nuevo más tarde.',
  'profile.loadFailed':
    'No se pudo cargar. Inténtalo de nuevo más tarde.',
  'profile.followingEntry':
    'A quién sigo',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'Perfil público',
  'publicProfile.avatar':
    'Anónimo',
  'publicProfile.loading':
    'Cargando…',
  'publicProfile.invalidLinkName':
    'Enlace al perfil público no válido',
  'publicProfile.invalidLinkBio':
    'Accede al perfil desde el nombre del autor de una publicación del foro.',
  'publicProfile.notFound':
    'No se encontró esta persona',
  'publicProfile.anonymous':
    'Persona anónima',
  'publicProfile.noBio':
    'Esta persona aún no ha configurado su perfil público.',
  'publicProfile.loadFailed':
    'No se pudo cargar el perfil público.',
  'publicProfile.postsLabel':
    'Publicaciones',
  'publicProfile.emptyPosts':
    'Esta persona aún no ha publicado nada.',

  'follow.label':
    'Seguir a esta persona',
  'follow.action':
    'Seguir',
  'follow.actionDone':
    'Siguiendo',
  'follow.unfollow':
    'Dejar de seguir',
  'follow.done':
    'Ahora sigues a esta persona.',
  'follow.failed':
    'No se pudo seguir. Inténtalo de nuevo más tarde.',

  'following.peopleLabel':
    'Personas que sigues',
  'following.postsLabel':
    'Publicaciones de las personas que sigues',
  'following.peopleLoading':
    'Cargando a quién sigues...',
  'following.emptyPeople':
    'Todavía no sigues a nadie. Pulsa «Seguir» en una publicación o sigue a alguien desde su perfil público.',
  'following.emptyPosts':
    'Las personas que sigues aún no han publicado nada.',
  'following.peopleFailed':
    'No se pudo cargar a quién sigues.',
  'following.postsFailed':
    'No se pudieron cargar las publicaciones.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'Te damos la bienvenida de nuevo',
  'login.body':
    '{site} es un espacio para cualquier persona que quiera dejar sus ideas. No necesitas un formulario de registro: basta con una cuenta de Google para empezar a publicar.',
  'login.browseFirst':
    'Empieza explorando la página de inicio',
  'admin.skipToMain':
    'Saltar al contenido principal',
  'admin.railLabel':
    'Menú de administración',
  'admin.railBrandAria':
    'Inicio de {site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'Funciones principales',
  'admin.railGovernance':
    'Gobernanza',
  'admin.railMode':
    'Modo de administración',
  'admin.railExit':
    'Volver al foro',
  'admin.topbarMenu':
    'Alternar menú de administración',
  'admin.statusOnline':
    'Conexión correcta',
  'admin.topbarForum':
    'Foro',
  'admin.logoutFailed':
    'No se pudo cerrar la sesión. Inténtalo de nuevo más tarde.',
  'admin.navUsers':
    'Gestión de personas',
  'admin.navPosts':
    'Publicaciones del foro',
  'admin.navReports':
    'Gestión de denuncias',
  'admin.listLoadFailed':
    'Error al cargar',
  'admin.dlgClose':
    'Cerrar ventana',
  'admin.dlgConfirm':
    'Confirmar',
  'admin.dlgSave':
    'Guardar',
  'admin.dlgApplyTags':
    'Aplicar etiquetas',
  'admin.dlgNoTags':
    'Actualmente no hay etiquetas a las que aplicar. Añádelas primero en «Gestión de etiquetas» más abajo.',
  'users.title':
    'Gestión de personas',
  'users.contentAction':
    'Contenido',
  'users.updateContentFailed':
    'No se pudo modificar el contenido.',
  'users.updated':
    'Contenido actualizado.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'Consulta las estadísticas de actividad, las etiquetas y el estado de las cuentas de las personas del foro, y gestiona suspensiones y la gestión del contenido por publicación.',
  'users.refresh':
    'Actualizar datos',
  'users.statTotal':
    'Total de personas',
  'users.statActive':
    'Activas',
  'users.statSuspended':
    'Suspendidas',
  'users.statContent':
    'Total de publicaciones/comentarios',
  'users.count':
    '{count} personas',
  'users.tagsCount':
    '{count} etiquetas',
  'users.loadFailed':
    'No se pudo cargar la información de las personas.',
  'users.tagsLoadFailed':
    'No se pudo cargar la información de las etiquetas.',
  'users.panelTitle':
    'Personas del foro',
  'users.tagsPanelTitle':
    'Etiquetas de personas',
  'users.colUser':
    'Persona',
  'users.colTags':
    'Etiquetas',
  'users.colStatus':
    'Estado',
  'users.colPosts':
    'Publicaciones',
  'users.colComments':
    'Comentarios',
  'users.colLikes':
    'Me gusta',
  'users.colLastActivity':
    'Última actividad',
  'users.colActions':
    'Acciones',
  'users.nicknameUnset':
    'Alias no configurado',
  'users.notSet':
    'No configurado',
  'users.statusActive':
    'Activas',
  'users.statusSuspended':
    'Suspendidas',
  'users.emptyTitle':
    'No hay datos de personas por el momento',
  'users.emptyBody':
    'Esto solo estará vacío cuando ninguna persona haya iniciado sesión en el foro con Google.',
  'users.tagsEmptyTitle':
    'No hay etiquetas por el momento',
  'users.tagsEmptyBody':
    'Crea primero una etiqueta para poder aplicarla a una persona en la lista de personas.',
  'users.addTag':
    'Añadir etiqueta',
  'users.colName':
    'Nombre',
  'users.colCreated':
    'Fecha de creación',
  'users.colUpdated':
    'Fecha de actualización',
  'users.renameTag':
    'Cambiar nombre',
  'users.suspend':
    'Suspender',
  'users.restore':
    'Reactivar',
  'users.statusDialogTitle':
    '{action} esta persona',
  'users.statusSuspendMessage':
    '{email} ya no podrá iniciar sesión en el foro. Se conservarán sus publicaciones y comentarios existentes. ¿Continuar?',
  'users.statusRestoreMessage':
    '{email} recuperará los permisos de inicio de sesión y publicación. ¿Continuar?',
  'users.userSuspended':
    'La persona fue suspendida.',
  'users.userRestored':
    'La persona se restauró.',
  'users.updateStatusFailed':
    'No se pudo actualizar el estado de la persona.',
  'users.editTagsTitle':
    'Editar etiqueta · {user}',
  'users.editTagsMessage':
    'Selecciona las etiquetas que deseas aplicar; cancelar todas equivale a quitar todas las etiquetas de esta persona.',
  'users.tagsUpdated':
    'Las etiquetas de la persona se actualizaron.',
  'users.updateTagsFailed':
    'No se pudieron actualizar las etiquetas de la persona.',
  'users.contentLoadFailed':
    'No se pudo cargar el contenido.',
  'users.contentLoadFailedShort':
    'No se pudo cargar el contenido.',
  'users.contentPanelTitle':
    'Contenido de la persona',
  'users.contentCount':
    '{posts} publicaciones · {comments} comentarios',
  'users.contentLoading':
    'Cargando publicaciones y comentarios…',
  'users.addPost':
    'Crear publicación',
  'users.addComment':
    'Añadir comentario',
  'users.postsColumn':
    'Publicaciones',
  'users.commentsColumn':
    'Comentarios',
  'users.noPosts':
    'No hay publicaciones',
  'users.noComments':
    'No hay comentarios',
  'users.postRef':
    'Publicación #{id}',
  'users.editRecordTitle':
    'Editar {kind} #{id}',
  'users.deleteRecordTitle':
    'Eliminar {kind} #{id}',
  'users.deleteRecordMessage':
    'La eliminación no se podrá revertir; también se eliminarán los Me gusta y los datos relacionados. ¿Continuar?',
  'users.contentLabel':
    'Contenido',
  'users.addPostTitle':
    'Crear publicación',
  'users.addPostMessage':
    'Esta publicación se publicará en nombre de esta persona; el campo de la persona autora no se puede falsificar.',
  'users.postContentLabel':
    'Contenido de la publicación',
  'users.postContentPlaceholder':
    'Introduce el contenido de la publicación',
  'users.pickPostTitle':
    'Seleccionar publicación',
  'users.postIdLabel':
    'ID de publicación',
  'users.postIdPlaceholder':
    'Número de la publicación en la que deseas comentar',
  'users.addCommentTitle':
    'Añadir comentario',
  'users.commentContentLabel':
    'Contenido del comentario',
  'users.commentContentPlaceholder':
    'Introduce el contenido del comentario',
  'users.createTagTitle':
    'Crear etiqueta',
  'users.createTagMessage':
    'Las etiquetas pueden utilizarse para clasificar a las personas, por ejemplo «moderadora», «activa» o «bloqueada».',
  'users.tagNameLabel':
    'Nombre de la etiqueta',
  'users.tagNamePlaceholder':
    'Como máximo 50 caracteres',
  'users.renameTagTitle':
    'Renombrar etiqueta',
  'users.renameTagMessage':
    'Todas las personas que usan esta etiqueta verán el nuevo nombre.',
  'users.deleteTagTitle':
    'Eliminar etiqueta «{name}»',
  'users.deleteTagMessage':
    'Al eliminarla, se quitará también esta etiqueta de todas las personas y no se podrá revertir la acción. ¿Continuar?',
  'users.deleteTagConfirm':
    'Eliminar etiqueta',
  'users.tagCreated':
    'Etiqueta creada.',
  'users.createTagFailed':
    'No se pudo crear la etiqueta.',
  'users.tagUpdated':
    'Etiqueta actualizada.',
  'users.updateTagFailed':
    'No se pudo actualizar la etiqueta.',
  'users.tagDeleted':
    'Etiqueta eliminada.',
  'users.deleteTagFailed':
    'No se pudo eliminar la etiqueta.',
  'users.refreshDone':
    'Ya se actualizó con la información más reciente.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'Panel de administración',
  'users.signinBody':
    'La gobernanza centralizada del contenido permite que cada revisión sea clara, rápida y trazable.',
  'users.signinStep1':
    'Verificación de seguridad',
  'users.signinStep2':
    'Gobernanza de la persona',
  'users.signinStep3':
    'Revisión del contenido',
  'users.signinPanelTitle':
    'Acceder al panel de administración',
  'users.signinPanelBody':
    'El panel de administración solo está disponible para cuentas de Google de personas administradoras; inicia sesión como persona administradora.',
  'posts.title':
    'Publicaciones del foro',
  'posts.searching':
    'Buscando…',
  'posts.searchDegraded':
    '(Sin servicio de búsqueda, se comparan palabras clave de la base de datos)',
  'posts.searchSummary':
    'Se encontraron {total} resultados para «{query}»; esta página muestra {shown}',
  'posts.pendingCount':
    '{count} elementos pendientes',
  'posts.pageSummary':
    'Página {page} / {pages}, {count} elementos en esta página',
  'posts.listLoadFailed':
    'No se pudo cargar la lista de publicaciones.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'Administra, edita y elimina publicaciones del foro, y gestiona el contenido a nivel de comentario.',
  'posts.toReports':
    'Gestión de denuncias',
  'posts.editorTitleNew':
    'Crear publicación',
  'posts.editorTitleEdit':
    'Editar publicación #{id}',
  'posts.editorNote':
    'Se publicará como persona administradora; la persona autora se obtiene de la sesión iniciada y no se puede falsificar la identidad de la persona autora de la solicitud.',
  'posts.cancelEdit':
    'Cancelar edición',
  'posts.contentLabel':
    'Contenido de la publicación',
  'posts.contentPlaceholder':
    'Introduce el contenido de la publicación',
  'posts.saveChanges':
    'Guardar cambios',
  'posts.emptyContent':
    'El contenido de la publicación no puede estar en blanco.',
  'posts.saving':
    'Guardando…',
  'posts.saved':
    'Publicación actualizada.',
  'posts.published':
    'Publicación publicada.',
  'posts.saveFailed':
    'No se pudo guardar.',
  'posts.deleteTitle':
    'Eliminar publicación #{id}',
  'posts.deleteMessage':
    'Al eliminarla no se podrá revertir la acción; también se eliminarán todos los comentarios debajo de esta publicación. ¿Continuar?',
  'posts.deleteConfirm':
    'Eliminar publicación',
  'posts.deleted':
    'Publicación eliminada.',
  'posts.deleteFailed':
    'No se pudo eliminar la publicación.',
  'posts.commentUpdated':
    'Comentario actualizado.',
  'posts.commentActionFailed':
    'No se pudo realizar la acción del comentario.',
  'posts.pickPostTitle':
    'Seleccionar la publicación a la que pertenece el comentario',
  'posts.pickPostMessage':
    'La publicación de esta fila es la predeterminada; si deseas adjuntarlo a otra, cambia el número de la publicación correspondiente.',
  'posts.postIdLabel':
    'ID de publicación',
  'posts.addCommentAtTitle':
    'Añadir comentario en la publicación #{id}',
  'posts.addCommentMessage':
    'Este comentario se publicará como persona administradora.',
  'posts.commentContentLabel':
    'Contenido del comentario',
  'posts.commentContentPlaceholder':
    'Introduce el contenido del comentario',
  'posts.add':
    'Añadir',
  'posts.editCommentTitle':
    'Editar comentario #{id}',
  'posts.deleteCommentTitle':
    'Eliminar comentario #{id}',
  'posts.deleteCommentMessage':
    'Al eliminarlo no se podrá revertir la acción. ¿Continuar?',
  'posts.listTitle':
    'Lista de publicaciones',
  'posts.clearSearch':
    'Eliminar búsqueda',
  'posts.searchLabel':
    'Búsqueda por palabras clave',
  'posts.searchPlaceholder':
    'Contenido de la publicación o Email completo de la persona autora de la publicación',
  'posts.searchHint':
    'Ordenados por relevancia; introduce un Email completo para encontrar todas las publicaciones de esa persona. La búsqueda sustituye la paginación; se muestran como máximo 25 resultados.',
  'posts.searchTotal':
    'Se encontraron {total} resultados de búsqueda',
  'posts.searchFailed':
    'Búsqueda fallida.',
  'posts.searchStatusFailed':
    'Búsqueda fallida',
  'posts.colContentImage':
    'Contenido e imágenes',
  'posts.colEngagement':
    'Interacción',
  'posts.colComments':
    'Comentarios',
  'posts.colAuthor':
    'Persona autora',
  'posts.imageAlt':
    'Imagen adjunta de la publicación',
  'posts.likes':
    '{count} Me gusta',
  'posts.author':
    'Persona autora：{name}',
  'posts.emptyTitle':
    'Actualmente no hay publicaciones del foro',
  'posts.emptyBody':
    'Puedes crear la primera publicación con el editor de arriba.',
  'posts.emptySearchTitle':
    'No hay publicaciones que coincidan',
  'posts.emptySearchBody':
    'No se encontraron publicaciones que coincidan con «{query}». Prueba con otra palabra clave.',
  'posts.commentCount':
    '{count} comentarios',
  'posts.noComments':
    'Todavía no hay comentarios',
  'posts.reportsTitle':
    'Denuncias pendientes',
  'posts.allReports':
    'Todas las denuncias',
  'posts.colReportedContent':
    'Contenido denunciado',
  'posts.colReason':
    'Motivo de la denuncia',
  'posts.colReporter':
    'Denunciante',
  'posts.colTime':
    'Hora',
  'posts.colVerdict':
    'Decisión',
  'posts.emptyReportsTitle':
    'No hay denuncias pendientes',
  'posts.emptyReportsBody':
    'Todas las denuncias han sido resueltas.',
  'posts.verdictResolved':
    'Resuelta',
  'posts.verdictRejected':
    'No válida',
  'posts.verdictDialogTitle':
    'Marcar la denuncia #{id} como «{label}»',
  'posts.verdictDialogMessage':
    'Una vez marcada, la denuncia saldrá de la lista de pendientes, pero los datos seguirán disponibles en la página de gestión de denuncias. ¿Quieres continuar?',
  'posts.verdictConfirm':
    'Marcar como {label}',
  'posts.verdictDone':
    'La denuncia se ha marcado como {label}.',
  'posts.verdictFailed':
    'Error al actualizar el estado de la denuncia.',
  'reports.title':
    'Gestión de denuncias',
  'reports.listSummary':
    '{count} · {filter}',
  'reports.listLoadFailed':
    'Error al cargar la lista de denuncias.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'Resuelve las denuncias una por una: «Aprobada» elimina el contenido denunciado, «No válida» lo conserva. Volver a «Pendiente» borra la fecha de la decisión anterior.',
  'reports.backToPosts':
    'Volver a las publicaciones',
  'reports.editorTitle':
    'Editar denuncia #{id}',
  'reports.editorNote':
    'Puedes corregir el motivo y el estado de la denuncia; el objetivo y la persona denunciante son registros existentes y no se cambian aquí.',
  'reports.targetTypeLabel':
    'Tipo de objetivo',
  'reports.targetIdLabel':
    'ID del objetivo',
  'reports.reporterEmailLabel':
    'Email de la persona denunciante',
  'reports.statusLabel':
    'Estado',
  'reports.reasonLabel':
    'Motivo de la denuncia',
  'reports.reasonHint':
    'Como máximo 500 caracteres; se mostrarán directamente a otros administradores como criterio para tomar una decisión.',
  'reports.filterLabel':
    'Filtrar por estado',
  'reports.filterAll':
    'Todo',
  'reports.listTitle':
    'Lista de denuncias',
  'reports.colTarget':
    'Contenido del objetivo',
  'reports.colReason':
    'Motivo de la denuncia',
  'reports.colReporter':
    'Denunciante',
  'reports.colStatus':
    'Estado',
  'reports.targetGone':
    '(El contenido se ha eliminado)',
  'reports.author':
    'Persona autora：{name}',
  'reports.deleteTitle':
    'Eliminar denuncia #{id}',
  'reports.deleteMessage':
    'Lo que se elimina es esta propia ficha de denuncia; el contenido denunciado no se ve afectado y no se puede restaurar. ¿Quieres continuar?',
  'reports.deleteConfirm':
    'Eliminar denuncia',
  'reports.deleted':
    'Denuncia eliminada.',
  'reports.deleteFailed':
    'Error al eliminar la denuncia.',
  'reports.updated':
    'Denuncia actualizada.',
  'reports.saveFailed':
    'Error al guardar la denuncia.',
  'reports.approveTitle':
    'Aprobar denuncia #{id}',
  'reports.approveGoneMessage':
    'El contenido {kind} #{id} denunciado ya no existe; esta denuncia solo se marcará como resuelta.',
  'reports.approveMessage':
    'Se eliminará de forma permanente el contenido {kind} #{id} denunciado (si es una publicación, también se eliminarán todos los comentarios de esa publicación), y esta denuncia se marcará como resuelta. ¿Quieres continuar?',
  'reports.approveConfirm':
    'Aprobar y eliminar la publicación',
  'reports.approveGoneDone':
    'El contenido ya no existe y la denuncia se ha marcado como resuelta.',
  'reports.approveDone':
    'Se ha eliminado la publicación y la denuncia se ha marcado como resuelta.',
  'reports.approveFailed':
    'Error al aprobar la denuncia.',
  'reports.approveTitleGone':
    'El contenido se ha eliminado; solo se marcará la denuncia',
  'reports.approveTitleFull':
    'Eliminar el contenido denunciado y marcar la denuncia como resuelta',
  'reports.rejectTitle':
    'La denuncia #{id} no es válida',
  'reports.rejectMessage':
    'No ser válida significa que el contenido denunciado no requiere tratamiento; se conservará tal como está. ¿Quieres continuar?',
  'reports.rejectConfirm':
    'Marcar como no válida',
  'reports.rejectDone':
    'La denuncia se ha marcado como no válida.',
  'reports.statusFailed':
    'Error al actualizar el estado de la denuncia.',
  'reports.emptyTitle':
    'Actualmente no hay denuncias',
  'reports.emptyBody':
    'No hay registros con este filtro.',
  'reports.rejectTitleAttr':
    'Conservar el contenido y marcar solo la denuncia como no válida',
  'kind.post':
    'Publicación',
  'kind.comment':
    'Comentario',
  'reports.statusPending':
    'Pendiente',
  'reports.statusResolved':
    'Resuelta',
  'reports.statusRejected':
    'No válida',
  'title.forum':
    '{site}',
  'title.login':
    'Iniciar sesión | {site}',
  'title.newPost':
    'Nueva publicación | {site}',
  'title.profile':
    'Perfil｜{site}',
  'title.publicProfile':
    'Perfil público | {site}',
  'title.following':
    'Siguiendo | {site}',
  'title.adminUsers':
    'Gestión de usuarios | {brand} Panel de administración',
  'title.adminLogin':
    'Iniciar sesión | {brand} Panel de administración',
  'title.adminPosts':
    'Publicaciones del foro | {brand} Panel de administración',
  'title.adminReports':
    'Gestión de denuncias | {brand} Panel de administración',
};
