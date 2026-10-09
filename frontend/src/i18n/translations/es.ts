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
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 hora',
  'common.oneDay': '1 día',
  'common.sevenDays': '7 días',
  'common.thirtyDays': '30 días',
  'common.oneYear': '1 año',
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
  'error.fallbackCommentEdit':
    'No se pudo guardar el comentario',
  'error.fallbackCommentDelete':
    'No se pudo eliminar el comentario',
  'error.fallbackReport':
    'Error al presentar la denuncia',
  'error.fallbackProfile':
    'Error al cargar el perfil de la persona',
  'error.fallbackProfileSave':
    'Error al guardar',
  'error.fallbackAvatarUpload':
    'Error al subir la imagen de perfil',
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
  'post.permalink':
    'Enlace permanente',
  'post.editedBadge':
    'editado',
  'post.editContentLabel':
    'Contenido de la publicación',
  'post.editMax':
    'Máximo 10000 caracteres',
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
  'comment.editedBadge':
    'editado',
  'comment.editContentLabel':
    'Contenido del comentario',
  'comment.editFailed':
    'No se pudo guardar el comentario. Intente de nuevo más tarde.',
  'comment.deleteFailed':
    'No se pudo eliminar el comentario. Intente de nuevo más tarde.',
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
  'newPost.imagePreviewAlt':
    'Vista previa de la imagen que se enviará',
  'newPost.draftNote':
    'El borrador se guarda automáticamente en este dispositivo (solo el texto; la imagen elegida no se conserva).',
  'postPage.loading':
    'Cargando la publicación...',
  'postPage.missing':
    'Puede que esta publicación se haya eliminado o que el enlace sea incorrecto.',
  'postPage.failed':
    'Error al cargar la publicación. Intente de nuevo más tarde.',
  'postPage.label':
    'Publicación',
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
  'profile.avatarLabel':
    'Imagen de perfil',
  'profile.avatarHint':
    'La imagen de perfil se mostrará en las publicaciones y comentarios que hagas. Puedes subir JPG, PNG, GIF o WebP.',
  'profile.avatarChoose':
    'Elegir imagen',
  'profile.avatarRemove':
    'Quitar imagen de perfil',
  'profile.avatarUploading':
    'Subiendo...',
  'profile.avatarPreviewAlt':
    'Vista previa de la nueva imagen de perfil',
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
  'profile.postsLabel':
    'Mis publicaciones',
  'profile.emptyPosts':
    'Todavía no has publicado nada.',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'Perfil público',
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
  'admin.navMonitor': 'Supervisión del sistema',
  'admin.navLog': 'Registro de acciones',
  'admin.navStats': 'Tendencias de contenido',
  'admin.navExport': 'Exportación y acciones masivas',
  'admin.navSessions': 'Inicios de sesión y sesiones',
  'admin.navBlocks': 'Lista de bloqueo de IP',
  'admin.navAnnouncements': 'Anuncios',
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

  'monitor.title': 'Supervisión del sistema',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'Vista en vivo del estado de los servicios, el volumen de solicitudes, la distribución de latencias y los contadores de limitadores de tasa.',
  'monitor.refresh': 'Actualizar',
  'monitor.refreshing': 'Cargando…',
  'monitor.autoRefresh': 'Actualización automática',
  'monitor.autoRefreshOn': 'Actualización automática cada {seconds} s',
  'monitor.autoRefreshOff': 'Actualización automática en pausa',
  'monitor.nextUpdate': 'Actualización en {seconds} s',
  'monitor.loadFailed': 'No se pudieron cargar los datos de supervisión.',
  'monitor.loadFailedHint': 'Comprueba que hayas iniciado sesión como administrador y que el backend siga en marcha.',
  'monitor.pausedHint': 'La actualización automática está en pausa; la vista muestra la última lectura correcta.',
  'monitor.visibilityPaused': 'La página está en segundo plano, por lo que la actualización automática está en pausa.',
  'monitor.lastUpdated': 'Actualizado a las {time}',
  'monitor.probeTook': 'Sondas de dependencias: {ms} ms',
  'monitor.unreachable': 'El backend no responde. La vista se queda en la última lectura correcta.',
  'monitor.depsTitle': 'Estado de los servicios',
  'monitor.depsNote': 'Cada lectura sondea una vez cada dependencia; el tiempo de espera por dependencia es de 2 s y las tres se ejecutan en paralelo.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'Motor de búsqueda',
  'monitor.stateOk': 'Operativo',
  'monitor.stateDown': 'Inaccesible',
  'monitor.stateDisabled': 'No activado',
  'monitor.depSearchFallback': 'ES_URL no está definido, así que la búsqueda usa la comparación por palabra clave de MySQL.',
  'monitor.depDisabled': 'No se inyectó ningún cliente de Redis, así que las funciones multimedia están desactivadas.',
  'monitor.depLatency': 'Respuesta en {ms} ms',
  'monitor.depKeys': '{count} claves',
  'monitor.depMemory': 'Memoria {size}',
  'monitor.depPoolUsage': 'Conexiones {inUse}/{open} (máximo {max})',
  'monitor.depPoolWait': '{count} esperas, {ms} ms en total',
  'monitor.depRedisPool': 'Aciertos {hits} / fallos {misses}',
  'monitor.depEngineMysql': 'Comparación por palabra clave de MySQL',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'Resumen de solicitudes',
  'monitor.statUptime': 'Tiempo en marcha',
  'monitor.statRequests': 'Solicitudes totales',
  'monitor.statErrorRate': 'Tasa de error',
  'monitor.statP95': 'Latencia P95',
  'monitor.statInFlight': 'En curso',
  'monitor.statGoroutines': 'Goroutines',
  'monitor.statHeap': 'Memoria del montón',
  'monitor.statDbPool': 'Conexiones a la base de datos',
  'monitor.statRateLimited': 'Bloqueadas por límite',
  'monitor.statCountWithPeak': 'pico {peak}',
  'monitor.statCountWithInUse': '{inUse} en uso, {idle} libres',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} núcleos lógicos · {gc} ciclos de GC',
  'monitor.noData': 'Todavía no hay solicitudes.',
  'monitor.noDataBody': 'Las solicitudes desde el arranque del servicio aparecerán aquí; este panel está vacío por ahora.',
  'monitor.timelineTitle': 'Tráfico de los últimos {minutes} minutos',
  'monitor.timelineNote': 'Los totales se acumulan desde el arranque de este proceso y se reinician al reiniciar; cada percentil de latencia es el límite superior de un cubo del histograma, por lo que solo toma valores discretos. Un minuto sin barra significa que no hubo tráfico en ese momento.',
  'monitor.timelineLive': 'Este proceso',
  'monitor.timelineHistory': 'Antes del reinicio',
  'monitor.timelineLegendVolume': 'Solicitudes',
  'monitor.timelineLegendError': 'Errores 5xx',
  'monitor.timelinePeak': 'Pico {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'No hay agregados históricos disponibles en la base de datos; si el arranque no puede leerlos (tabla inexistente o sin permisos) solo se muestra el proceso actual.',
  'monitor.routesTitle': 'Por ruta',
  'monitor.routesNote': 'Las rutas se normalizan (los identificadores numéricos y las direcciones de correo se convierten en :id), así que distintos identificadores de la misma ruta se cuentan juntos.',
  'monitor.colRoute': 'Ruta',
  'monitor.colCount': 'Solicitudes',
  'monitor.colAvg': 'Media',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'Más lenta',
  'monitor.colErrors': 'Errores',
  'monitor.routeOther': 'Otras (límite de rutas alcanzado)',
  'monitor.clientsTitle': 'Direcciones de origen',
  'monitor.clientsNote': 'Ordenadas por número de solicitudes. Una dirección tomada de X-Forwarded-For o X-Real-IP no se ha verificado con un proxy de confianza: comprueba que tu proxy sobrescriba esas cabeceras antes de actuar.',
  'monitor.noClients': 'Aún no hay orígenes vigilados.',
  'monitor.noClientsBody': 'Cada solicitud se registra con su dirección de origen. Esta tabla está vacía por ahora.',
  'monitor.clientsDropped': 'El número de orígenes alcanzó el límite de {limit}; las {count} direcciones vistas hace más tiempo quedaron fuera del seguimiento. Esta línea significa que la lista está incompleta, no que solo vinieran estas personas.',
  'monitor.colIp': 'Dirección',
  'monitor.colSource': 'Origen',
  'monitor.colRateLimited': 'Limitadas por tasa',
  'monitor.colBanned': 'Rechazadas por bloqueo',
  'monitor.colLastRoute': 'Último acceso',
  'monitor.colActions': 'Acciones',
  'monitor.colBlock': 'Bloquear',
  'monitor.blocking': 'Bloqueando…',
  'monitor.sourcePeer': 'Par',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.trustLegacy': 'TRUSTED_PROXY_CIDRS no está definido: el servidor mantiene el comportamiento antiguo y prioriza la entrada más a la izquierda de X-Forwarded-For. Hasta confirmar que hay un proxy delante que sobrescribe realmente estas cabeceras y que los usuarios no pueden esquivarlo, el límite de tasa y los bloqueos de IP se pueden esquivar con una sola cabecera falsificada, y la dirección de origen del registro de auditoría no sirve como prueba.',
  'monitor.trustConfigured': 'Hay proxies de confianza configurados: X-Forwarded-For / X-Real-IP solo se aceptan si el par de conexión está en una de estas redes; en caso contrario se usa la dirección del par. Redes en vigor: {cidrs}.',
  'monitor.trustBroken': 'Se declaró TRUSTED_PROXY_CIDRS pero ninguna entrada se puede interpretar como rango CIDR ({declared}), así que sigue aplicándose el comportamiento antiguo sin configurar.',
  'monitor.trustPartial': 'Las siguientes entradas de TRUSTED_PROXY_CIDRS no se pueden interpretar como rangos CIDR ({invalid}), así que las cabeceras reenviadas desde esos rangos nunca se aceptan. Las solicitudes que llegan por esos rangos se agrupan por la dirección del par, es decir, comparten el mismo presupuesto de límite de tasa y la misma consulta de la lista de bloqueo.',
  'monitor.blockTitle': 'Bloquear {ip}',
  'monitor.blockMessage': 'Las peticiones de escritura desde esta dirección (publicaciones, comentarios, me gusta, denuncias, subidas de imágenes y redirecciones de inicio de sesión) se rechazarán durante {duration}. La lectura no se ve afectada. ¿Bloquear?',
  'monitor.blockReason': 'Bloqueado desde la página de monitorización',
  'monitor.blocked': '{ip} bloqueada',
  'monitor.blockFailed': 'La operación de bloqueo falló.',
  'monitor.limitsTitle': 'Limitadores de tasa',
  'monitor.limitsNote': 'Cada grupo tiene su propio presupuesto según el coste de sus endpoints; los bloqueos cuentan todos los 429 desde el arranque de este proceso.',
  'monitor.colLimiter': 'Limitador',
  'monitor.colBudget': 'Presupuesto',
  'monitor.colAllowed': 'Permitidas',
  'monitor.colBlocked': 'Bloqueadas',
  'monitor.colTracked': 'Orígenes vigilados',
  'monitor.colBlockedRate': 'Tasa de bloqueo',
  'monitor.limitContent': 'Escrituras de contenido',
  'monitor.limitUpload': 'Subidas de imágenes',
  'monitor.limitAuth': 'Inicio de sesión OAuth',
  'monitor.limitBudget': '{limit} por {window} s',
  'monitor.limitUnknown': '(desconocido)',
  'monitor.noLimits': 'No hay limitadores disponibles.',
  'monitor.noLimitsBody': 'Los limitadores de tasa aún no se han creado, así que sus contadores no están disponibles.',

  'log.title': 'Registro de acciones',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'Consulta cada cambio hecho en la administración: quién, cuándo, sobre qué objetivo y qué campos pasaron de qué a qué.',
  'log.refresh': 'Actualizar',
  'log.loadFailed': 'No se pudo cargar el registro de acciones.',
  'log.empty': 'Ninguna acción coincide con los filtros.',
  'log.emptyBody': 'Amplía los filtros o confirma que realmente no hubo acciones en ese periodo.',
  'log.count': 'Entradas {from}–{to} de {total}',
  'log.retention': 'Los registros se conservan {days} días y luego los borra una tarea en segundo plano. Esta página no tiene ningún botón para borrar entradas: un registro de auditoría que puede borrar sus propios registros no es un registro de auditoría.',
  'log.filterActor': 'Autor',
  'log.filterAction': 'Acción',
  'log.filterTargetType': 'Tipo de recurso',
  'log.filterFrom': 'Desde',
  'log.filterTo': 'Hasta',
  'log.filterAll': 'Todos',
  'log.filterApply': 'Aplicar filtros',
  'log.filterReset': 'Limpiar filtros',
  'log.filterTargetHint': 'Haz clic en el objetivo de cualquier fila para ver solo las acciones que lo afectaron.',
  'log.colTime': 'Hora',
  'log.colActor': 'Autor',
  'log.colAction': 'Acción',
  'log.colTarget': 'Objetivo',
  'log.colChanges': 'Cambios',
  'log.colOrigin': 'Origen',
  'log.noChanges': '(sin cambios de campo)',
  'log.changedTo': 'cambiado a',
  'log.removed': '(eliminado)',
  'log.created': '(creado)',
  'log.requestId': 'request {id}',
  'log.page': 'Página {page}',
  'log.targetUser': 'Usuario',
  'log.targetPost': 'Publicación',
  'log.targetComment': 'Comentario',
  'log.targetReport': 'Denuncia',
  'log.targetTag': 'Etiqueta',
  'log.targetSystem': 'Sistema',
  'log.actionUserSuspend': 'Suspendido',
  'log.actionUserReinstate': 'Reactivado',
  'log.actionUserTags': 'Etiquetas modificadas',
  'log.actionUserPost': 'Publicado como usuario',
  'log.actionUserComment': 'Comentado como usuario',
  'log.actionUserContent': 'Contenido eliminado',
  'log.actionPostCreate': 'Publicación creada',
  'log.actionPostUpdate': 'Publicación editada',
  'log.actionPostDelete': 'Publicación eliminada',
  'log.actionCommentCreate': 'Comentario creado',
  'log.actionCommentUpdate': 'Comentario editado',
  'log.actionCommentDelete': 'Comentario eliminado',
  'log.actionReportCreate': 'Denuncia creada',
  'log.actionReportResolve': 'Denuncia admitida',
  'log.actionReportReject': 'Denuncia descartada',
  'log.actionReportUpdate': 'Denuncia editada',
  'log.actionReportDelete': 'Denuncia eliminada',
  'log.actionTagCreate': 'Etiqueta creada',
  'log.actionTagUpdate': 'Etiqueta renombrada',
  'log.actionTagDelete': 'Etiqueta eliminada',
  'log.fieldStatus': 'Estado de la cuenta',
  'log.fieldContent': 'Contenido',
  'log.fieldName': 'Nombre',
  'log.fieldTags': 'Etiquetas',
  'log.fieldReason': 'Motivo',
  'log.fieldAuthorEmail': 'Autor',
  'log.fieldReporterEmail': 'Denunciante',
  'log.fieldTargetType': 'Tipo de recurso',
  'log.fieldTargetId': 'Id del recurso',
  'log.fieldPostId': 'Id de publicación',
  'log.fieldCommentId': 'Id de comentario',
  'log.fieldPostIdShort': 'Publicación',
  'log.fieldCommentIdShort': 'Comentario',
  'log.fieldTarget': 'Objetivo',
  'log.fieldAssignmentsRemoved': 'Asignaciones eliminadas',
  'log.truncated': 'truncado',

  'stats.title': 'Tendencias de contenido',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'Cuántos usuarios, publicaciones y comentarios nuevos llegan cada día, y qué publicaciones, etiquetas y autores están más activos ahora.',
  'stats.refresh': 'Actualizar',
  'stats.loadFailed': 'No se pudieron cargar las estadísticas de contenido.',
  'stats.window': 'Mostrar los últimos',
  'stats.windowDays': '{days} días',
  'stats.windowClamped': '(hasta 90 días)',
  'stats.windowNote': 'Cada día se recorta en la zona horaria local de la máquina del servidor. Si el sitio corre en UTC y la administración está en otra zona, los números de hoy parecen bajos: es la zona horaria, no una caída del tráfico.',
  'stats.generatedAt': 'Estadísticas generadas el {time}',
  'stats.seriesTitle': 'Nuevos por día',
  'stats.seriesNote': 'Las tres curvas tienen escalas distintas, así que se muestran por separado en lugar de superpuestas.',
  'stats.seriesUsers': 'Usuarios nuevos',
  'stats.seriesPosts': 'Publicaciones nuevas',
  'stats.seriesComments': 'Comentarios nuevos',
  'stats.seriesEmpty': 'Sin datos en este periodo.',
  'stats.totalsTitle': 'Totales del periodo',
  'stats.totalsNote': 'Son las cantidades añadidas en este periodo, no los totales actuales del sitio.',
  'stats.totalUsers': 'Usuarios nuevos',
  'stats.totalPosts': 'Publicaciones nuevas',
  'stats.totalComments': 'Comentarios nuevos',
  'stats.totalLikes': 'Me gusta nuevos',
  'stats.topPostsTitle': 'Publicaciones populares',
  'stats.topPostsNote': 'Ordenadas por comentarios y me gusta, contando solo las publicadas en el periodo.',
  'stats.topTagsTitle': 'Etiquetas populares',
  'stats.topTagsNote': 'Ordenadas por usuarios asignados, sin límite de tiempo: una etiqueta es una cualidad, no un suceso.',
  'stats.topAuthorsTitle': 'Autores activos',
  'stats.topAuthorsNote': 'Ordenados por publicaciones en el periodo, con los comentarios aparte.',
  'stats.colExcerpt': 'Extracto',
  'stats.colEngagement': 'Interacción',
  'stats.colPosts': 'Publicaciones',
  'stats.colComments': 'Comentarios',
  'stats.colUsers': 'Usuarios',
  'stats.colAuthor': 'Autor',
  'stats.empty': 'Sin datos en este periodo.',
  'stats.emptyBody': 'Amplía el periodo o confirma que realmente no hubo contenido nuevo.',
  'stats.engagement': '{comments} comentarios・{likes} me gusta',
  'stats.rank': 'N.º {rank}',

  'export.title': 'Exportación y acciones masivas',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'Exporta los datos del sitio en CSV para cotejar, o actúa sobre varias cuentas a la vez.',
  'export.download': 'Descargar CSV',
  'export.downloading': 'Preparando…',
  'export.exportTitle': 'Exportar',
  'export.exportNote': 'Cada exportación está limitada a 50.000 filas; más allá solo se incluyen las más recientes. Las columnas de las tres exportaciones coinciden con las de las páginas de administración, así que el cotejo es directo.',
  'export.exportUsers': 'Lista de usuarios',
  'export.exportUsersNote': 'Correo, estado, fechas de creación y modificación, número de publicaciones y comentarios.',
  'export.exportPosts': 'Lista de publicaciones',
  'export.exportPostsNote': 'Id, autor, primeros 200 caracteres del contenido, fecha de creación, comentarios y me gusta.',
  'export.exportReports': 'Lista de denuncias',
  'export.exportReportsNote': 'Id, objetivo denunciado, denunciante, motivo, estado y rastro de la revisión.',
  'export.safety': 'El archivo empieza con una marca de orden de bytes UTF-8, así que Excel lo abre sin caracteres corruptos.',
  'export.safetyPrefix': 'Los valores que empiezan por = + - @ o por un espacio invisible reciben un apóstrofo inicial: es lo que hace que la hoja los trate como texto y no los ejecute como fórmulas. El prefijo es deliberado; no lo pidan eliminar.',
  'export.batchTitle': 'Acciones masivas',
  'export.batchNote': 'Los botones se activan al seleccionar cuentas en la página Usuarios. Una acción masiva se aplica entera o no se aplica: no existe un resultado parcial.',
  'export.batchSuspend': 'Suspender selección',
  'export.batchReinstate': 'Reactivar selección',
  'export.batchTags': 'Aplicar etiquetas',
  'export.batchTagsNote': 'Semántica de sobrescritura: la lista enviada pasa a ser el resultado. Enviar una lista vacía las quita todas.',
  'export.batchConfirm': '¿Aplicar «{action}» a {count} cuentas?',
  'export.batchConfirmTags': '¿Sobrescribir las etiquetas de {count} cuentas con {tags}?',
  'export.batchTagsPicker': 'Elegir etiquetas',
  'export.batchTagsNone': 'Sin etiquetas (quitarlas todas)',
  'export.batchRunning': 'Procesando…',
  'export.batchDone': '{updated} cuentas actualizadas',
  'export.batchDoneUnchanged': 'de las cuales {unchanged} ya estaban en el estado objetivo y no cambiaron',
  'export.batchSkipped': '{count} omitidas',
  'export.batchMax': 'Hasta 200 cuentas por lote',
  'export.gotoUsers': 'Ir a Usuarios',
  'export.noSelection': 'Selecciona primero cuentas en la página Usuarios.',
  'export.selected': '{count} cuentas seleccionadas',
  'export.clearSelection': 'Quitar selección',
  'export.selectionHint': 'La selección se conserva en esta página y desaparece al cerrarla.',

  'session.title': 'Inicios de sesión y sesiones',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'Mira qué inicios de sesión siguen siendo válidos y fuerza el cierre de sesión de una cuenta en todos sus dispositivos.',
  'session.refresh': 'Actualizar',
  'session.loadFailed': 'No se pudo cargar la lista de sesiones.',
  'session.privacyTitle': 'Por qué no se muestra el token completo',
  'session.privacyNote': 'El token es en sí mismo la credencial de acceso. Solo se muestran los ocho primeros caracteres, para que una administración pueda reconocer dos filas como la misma sesión; eso no basta para que nadie acceda en nombre del usuario, tampoco quien reciba una captura de esta página. Es una limitación deliberada, no una función inacabada.',
  'session.expireNote': 'Una sesión caduca tras {hours} horas sin actividad; cualquier petición la renueva.',
  'session.filterEmail': 'Filtrar por cuenta',
  'session.filterPlaceholder': 'Dirección de correo completa',
  'session.search': 'Buscar',
  'session.clearFilter': 'Limpiar',
  'session.summary': '{total} sesiones en todo el sitio, {scanned} claves revisadas',
  'session.truncated': 'El barrido alcanzó el límite de {scanned} claves y terminó antes de tiempo, así que esta lista está incompleta.',
  'session.empty': 'No hay sesiones ahora mismo.',
  'session.emptyBody': 'Nadie ha iniciado sesión o todas las sesiones han caducado.',
  'session.colUser': 'Cuenta',
  'session.colToken': 'Sesión',
  'session.colCreated': 'Creada',
  'session.colExpires': 'Caduca',
  'session.colRemaining': 'Restante',
  'session.colActions': 'Acción',
  'session.unknown': 'Desconocido',
  'session.adminBadge': 'Administración',
  'session.revoke': 'Forzar cierre de sesión',
  'session.revokeTitle': 'Forzar el cierre de sesión de {email}',
  'session.revokeMessage': 'Las {count} sesiones actuales de la cuenta se invalidarán de inmediato y se borrará el inicio de sesión en todos sus dispositivos. La persona tendrá que volver a iniciar sesión. ¿Continuar?',
  'session.revokeRunning': 'Revocando…',
  'session.revokeDone': 'Se revocaron {count} sesiones',
  'session.revokeNone': 'Esta cuenta no tiene sesiones activas',
  'session.revokeFailed': 'No se pudo confirmar el cierre de sesión.',
  'session.revokeUnavailable': 'No significa que la revocación haya fallado: el barrido alcanzó su límite de claves, así que puede que algunas sesiones no se alcanzaran. Inténtalo de nuevo en un momento.',
  'session.titleColumnNote': 'Prefijo identificativo únicamente; no sirve para iniciar sesión',

  'block.title': 'Lista de bloqueo de IP',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'Pon en la lista las direcciones de origen cuyo abuso está confirmado. La lista vive en Redis: ni un reinicio ni un despliegue la borran.',
  'block.refresh': 'Actualizar',
  'block.loadFailed': 'No se pudo cargar la lista de bloqueo.',
  'block.unavailable': 'Este sitio no tiene conexión a Redis, así que el bloqueo no está activo.',
  'block.unavailableNote': 'La lista necesita el mismo Redis que las sesiones y los tokens de medios. Una vez conectado, esta página funcionará; hasta entonces la única protección es la limitación de tasa (por proceso, se reinicia al reiniciar el servidor).',
  'block.add': 'Bloquear',
  'block.addTitle': 'Bloquear una dirección IP',
  'block.addMessage': 'Una dirección bloqueada se rechaza en toda petición de escritura (publicaciones, comentarios, me gusta, denuncias, subidas de imágenes y redirecciones de inicio de sesión) hasta que expire. La lectura no se ve afectada.',
  'block.ipLabel': 'Dirección IP',
  'block.ipPlaceholder': '203.0.113.9 o 2001:db8::1',
  'block.durationLabel': 'Duración',
  'block.reasonLabel': 'Motivo',
  'block.reasonPlaceholder': 'Por qué se bloquea esta dirección (queda en el registro de auditoría)',
  'block.reasonHint': 'El motivo solo va al registro de auditoría. Nunca se le muestra a la persona bloqueada ni aparece en un mensaje de error público.',
  'block.blocking': 'Bloqueando…',
  'block.done': '{ip} bloqueada',
  'block.removed': '{ip} desbloqueada',
  'block.removedNone': '{ip} no estaba bloqueada',
  'block.failed': 'La operación de bloqueo falló.',
  'block.unavailableService': 'Lista de bloqueo no disponible (sin Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'Caduca',
  'block.colRemaining': 'Restante',
  'block.colActions': 'Acción',
  'block.unblock': 'Desbloquear',
  'block.unblockTitle': 'Desbloquear {ip}',
  'block.unblockMessage': 'Esta dirección recupera de inmediato el acceso normal de escritura. ¿Continuar?',
  'block.empty': 'La lista de bloqueo está vacía.',
  'block.emptyBody': 'No hay ninguna dirección bloqueada. Ese es el estado normal: el bloqueo siempre es una decisión de la administración y el sistema nunca bloquea a nadie automáticamente.',
  'block.notAutoNote': 'Esta lista nunca se llena sola. Una dirección que supera su límite de tasa solo recibe un 429; no se añade aquí automáticamente, porque la misma salida puede pertenecer a toda una oficina o a todo un NAT, y el bloqueo automático también les afectaría.',
  'block.scopeNoteLabel': 'Alcance',
  'block.notAutoNoteLabel': 'Nunca automático',
  'block.maxNoteLabel': 'Duración máxima',
  'block.scopeNote': 'El bloqueo solo detiene las peticiones de escritura. Leer publicaciones, comentarios y recursos estáticos sigue siendo posible, y una persona bloqueada todavía puede iniciar sesión y ver contenido: es deliberado, porque los puntos de lectura no se limitan a propósito (si no, un visitante anónimo no podría usar el sitio) y el bloqueo cubre el mismo conjunto.',
  'block.maxNote': 'Un bloqueo dura como máximo 365 días. Un tiempo mayor se recorta a un año: la caducidad se guarda como número, y «para siempre» sería un bloqueo que nadie recuerda y que nunca se levanta solo.',
  'block.count': '{count} direcciones bloqueadas',
  'block.ipInvalid': 'Dirección IP no válida. Escribe una dirección IPv4 o IPv6; los rangos CIDR no son compatibles.',
  'announce.label': 'Anuncio del sitio',
  'announce.publicNote': 'Aviso',
  'announce.closeAria': 'Cerrar este anuncio',
  'announce.publishedOn': 'Publicado el {date}',
  'announce.expiresOn': 'Caduca el {date}',
  'announce.neverExpires': 'Sin caducidad',
  'announce.pinnedBadge': 'Fijado',
  'announce.title': 'Anuncios',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'Muestra un anuncio sobre todas las páginas del sitio. Solo uno está activo a la vez: publicar uno nuevo desactiva el anterior.',
  'announce.refresh': 'Actualizar',
  'announce.loadFailed': 'No se pudieron cargar los anuncios.',
  'announce.new': 'Publicar un anuncio nuevo',
  'announce.edit': 'Editar',
  'announce.deactivate': 'Desactivar',
  'announce.reactivate': 'Reactivar',
  'announce.deleteNote': 'Los anuncios nunca se borran, solo se desactivan: conservar el historial permite responder cuándo se publicó y por quién.',
  'announce.bodyLabel': 'Texto del anuncio',
  'announce.bodyPlaceholder': 'Por ejemplo: el sistema estará en mantenimiento el jueves de 02:00 a 04:00.',
  'announce.bodyHint': 'Hasta 300 caracteres. Texto plano; los saltos de línea se conservan.',
  'announce.activeLabel': 'Mostrar de inmediato',
  'announce.expiryLabel': 'Vigencia',
  'announce.expiryNever': 'Nunca caduca automáticamente',
  'announce.expiryHours': 'en {hours} horas',
  'announce.expiryDays': 'en {days} días',
  'announce.saving': 'Guardando…',
  'announce.published': 'Anuncio publicado',
  'announce.updated': 'Anuncio actualizado',
  'announce.deactivated': 'Anuncio desactivado',
  'announce.reactivated': 'Anuncio reactivado',
  'announce.saveFailed': 'La operación de anuncio falló.',
  'announce.empty': 'Todavía no hay anuncios.',
  'announce.emptyBody': 'En cuanto publiques uno aparecerá arriba en la página de cada visitante.',
  'announce.colBody': 'Texto',
  'announce.colState': 'Estado',
  'announce.colAuthor': 'Publicado por',
  'announce.colCreated': 'Publicado el',
  'announce.colActions': 'Acción',
  'announce.stateActive': 'Visible',
  'announce.stateInactive': 'Desactivado',
  'announce.stateExpired': 'Caducado',
  'announce.confirmDeactivate': 'Publicarlo desactivará el actual, y todos verán el texto nuevo de inmediato. ¿Continuar?',
  'announce.confirmEdit': '¿Actualizar el texto o la vigencia de este anuncio?',
  'announce.count': '{count} en total',
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
  'posts.edited':
    'Publicación actualizada.',
  'posts.editFailed':
    'No se pudo guardar la publicación. Intente de nuevo más tarde.',
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
  'posts.pin': 'Fijar',
  'posts.unpin': 'Desfijar',
  'posts.pinTitle': 'Fijar esta publicación',
  'posts.unpinTitle': 'Desfijar esta publicación',
  'posts.pinMessage': 'Una vez fijada, esta publicación se queda arriba del feed de todos y ninguna publicación nueva la desplaza.',
  'posts.unpinMessage': 'Al desfijarla vuelve a su lugar por orden de tiempo.',
  'posts.pinDone': 'Fijada',
  'posts.unpinDone': 'Desfijada',
  'posts.pinFailed': 'La operación de fijar falló.',
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
  'title.post':
    'Publicación｜{site}',
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
  'title.adminMonitor': 'Supervisión del sistema｜{brand} Administración',
  'title.adminLog': 'Registro de acciones｜{brand} Administración',
  'title.adminStats': 'Tendencias de contenido｜{brand} Administración',
  'title.adminExport': 'Exportación y acciones masivas｜{brand} Administración',
  'title.adminSessions': 'Inicios de sesión y sesiones｜{brand} Administración',
  'title.adminBlocks': 'Lista de bloqueo de IP｜{brand} Administración',
  'title.adminAnnouncements': 'Anuncios｜{brand} Administración',
};
