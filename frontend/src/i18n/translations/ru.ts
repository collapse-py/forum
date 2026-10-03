/*
 * ru catalog (src/i18n/translations/ru.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const ru: Record<MessageKey, string> = {
  'common.cancel':
    'Отмена',
  'common.save':
    'Сохранить',
  'common.submitting':
    'Отправка...',
  'common.delete':
    'Удалить',
  'common.edit':
    'Изменить',
  'common.search':
    'Поиск',
  'common.loading':
    'Загрузка…',
  'common.loadFailed':
    'Не удалось загрузить',
  'common.refresh':
    'Обновить',
  'common.nextStep':
    'Следующий шаг',
  'common.prevPage':
    'Назад',
  'common.nextPage':
    'Далее',
  'common.create':
    'Создать',
  'common.publish':
    'Опубликовать',
  'common.placeholder':
    '—',
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 час',
  'common.oneDay': '1 день',
  'common.sevenDays': '7 дней',
  'common.thirtyDays': '30 дней',
  'common.oneYear': '1 год',
  'common.backToHome':
    'Вернуться на главную',
  'common.backToForumHome':
    'Вернуться на главную форума',
  'common.backOnePage':
    'Вернуться на предыдущую страницу',
  'error.request':
    'Пожалуйста, повторите попытку позже.',
  'error.requestStatus':
    'Не удалось выполнить запрос (HTTP {status}).',
  'error.loginRequired':
    'Сначала войдите в систему, а затем продолжите.',
  'error.adminSessionExpired':
    'Сессия входа истекла. Вас перенаправят на страницу входа.',
  'error.fallbackLoad':
    'Не удалось загрузить',
  'error.fallbackSearch':
    'Не удалось выполнить поиск',
  'error.fallbackLike':
    'Не удалось поставить отметку «Нравится»',
  'error.fallbackComments':
    'Не удалось загрузить комментарии',
  'error.fallbackCommentPost':
    'Не удалось отправить комментарий',
  'error.fallbackCommentEdit':
    'Не удалось сохранить комментарий',
  'error.fallbackCommentDelete':
    'Не удалось удалить комментарий',
  'error.fallbackReport':
    'Не удалось отправить жалобу',
  'error.fallbackProfile':
    'Не удалось прочитать профиль',
  'error.fallbackProfileSave':
    'Не удалось сохранить',
  'error.fallbackPublish':
    'Неверный формат публикации',
  'error.fallbackUpload':
    'Неверный формат ответа при загрузке изображения',
  'error.fallbackNotFound':
    'Пользователь не найден',
  'error.fallbackFollow':
    'Не удалось подписаться',
  'auth.checking':
    'Проверка входа...',
  'auth.statusUnknown':
    'Не удалось определить состояние входа',
  'auth.feedLoggedIn':
    'Вы вошли в систему, можно публиковать публикации.',
  'auth.feedLoggedOut':
    'Войдите в систему, чтобы публиковать публикации.',
  'auth.profileLoggedIn':
    'Вы вошли в систему',
  'auth.profileLoggedOut':
    'Войдите в систему, чтобы настроить профиль.',
  'auth.googleLogin':
    'Войти с Google',
  'auth.loginWithGoogle':
    'Войти с помощью Google',
  'auth.loginWithGoogleAdmin':
    'Войти с помощью административного аккаунта Google',
  'auth.logout':
    'Выйти',
  'install.button':
    'Установить приложение',
  'install.hint':
    'В данный момент браузер не предлагает автоматическую установку. Откройте меню браузера и выберите «Установить приложение» или «Добавить на главный экран».',
  'bottomNav.label':
    'Основная навигация',
  'bottomNav.home':
    'Главная',
  'bottomNav.new':
    'Новое',
  'bottomNav.profile':
    'Профиль',
  'i18n.ariaLabel':
    'Выбрать язык',
  'i18n.current':
    'Язык: {name}',
  'feed.searchPlaceholder':
    'Поиск по содержимому публикаций',
  'feed.searchAriaLabel':
    'Поиск публикаций',
  'feed.searchResultsLabel':
    'Результаты поиска',
  'feed.postsLabel':
    'Публикации форума',
  'feed.searchFailed':
    'Не удалось выполнить поиск. Повторите попытку позже.',
  'feed.searching':
    'Поиск...',
  'feed.searchMore':
    'Загрузить больше результатов поиска...',
  'feed.searchMoreFailed':
    'Не удалось загрузить результаты поиска',
  'feed.searchFound':
    'Найдено: {total}',
  'feed.searchDegraded':
    '{base} (служба поиска не включена, сейчас выполняется сопоставление по ключевым словам в базе данных)',
  'feed.searchTotal':
    'Всего результатов: {total}',
  'feed.searchNoResults':
    'Не найдено публикаций, содержащих «{query}».',
  'feed.loadingPosts':
    'Загрузка публикаций...',
  'feed.loadMorePosts':
    'Загрузить больше публикаций...',
  'feed.postsFailed':
    'Не удалось загрузить публикации. Повторите попытку позже.',
  'feed.postsFailedShort':
    'Не удалось загрузить. Повторите попытку позже',
  'feed.scrollMore':
    'Проведите вниз, чтобы загрузить больше',
  'feed.endOfFeed':
    'Конец ленты',
  'feed.noPosts':
    'Пока нет публикаций. Оставьте первым свое мнение.',
  'feed.likeFailed':
    'Не удалось поставить или отменить отметку «Нравится». Повторите попытку позже.',
  'post.authorAnonymous':
    'Анонимно',
  'post.report':
    'Жалоба на публикацию',
  'post.imageAlt':
    'Изображение публикации',
  'post.unlike':
    'Больше не нравится',
  'post.like':
    'Нравится',
  'post.reply':
    'Ответить',
  'post.permalink':
    'Постоянная ссылка',
  'post.editedBadge':
    'изменено',
  'post.editContentLabel':
    'Текст публикации',
  'post.editMax':
    'Максимум 10000 символов',
  'comment.loading':
    'Загрузка комментариев...',
  'comment.none':
    'Комментариев пока нет',
  'comment.loadFailed':
    'Не удалось загрузить комментарии. Повторите попытку позже',
  'comment.placeholder':
    'Напишите комментарий...',
  'comment.max':
    'Максимум 2000 символов',
  'comment.submit':
    'Отправить комментарий',
  'comment.failed':
    'Не удалось отправить комментарий. Повторите попытку позже.',
  'comment.report':
    'Жалоба',
  'comment.more':
    'Загрузить больше комментариев...',
  'comment.editedBadge':
    'изменено',
  'comment.editContentLabel':
    'Текст комментария',
  'comment.editFailed':
    'Не удалось сохранить комментарий. Повторите попытку позже.',
  'comment.deleteFailed':
    'Не удалось удалить комментарий. Повторите попытку позже.',
  'report.reasonPlaceholder':
    'Введите причину жалобы (максимум 500 символов)',
  'report.note':
    'Жалобы будут переданы администраторам сайта',
  'report.formLabel':
    'Поле ввода жалобы',
  'report.submit':
    'Отправить жалобу',
  'report.failed':
    'Не удалось отправить жалобу. Повторите попытку позже.',
  'report.sent':
    'Жалоба отправлена. Спасибо за сообщение.',
  'newPost.avatarYou':
    'Вы',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'Создать публикацию',
  'newPost.loginFirst':
    'Сначала войдите в систему с помощью Google, затем создайте публикацию.',
  'newPost.contentPlaceholder':
    'Поделитесь своими мыслями...',
  'newPost.addImage':
    'Добавить изображение',
  'newPost.emailPrivate':
    'Ваш email не будет опубликован',
  'newPost.submit':
    'Опубликовать пост',
  'newPost.publishing':
    'Публикация...',
  'newPost.uploading':
    'Загрузка изображения...',
  'newPost.failed':
    'Не удалось выполнить публикацию. Повторите попытку позже.',
  'newPost.imagePreviewAlt':
    'Предпросмотр загружаемого изображения',
  'newPost.draftNote':
    'Черновик автоматически сохраняется на этом устройстве (только текст; выбранное изображение не сохраняется).',
  'postPage.loading':
    'Загрузка публикации...',
  'postPage.missing':
    'Эта публикация могла быть удалена, или ссылка неверна.',
  'postPage.failed':
    'Не удалось загрузить публикацию. Повторите попытку позже.',
  'postPage.label':
    'Публикация',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'Личный профиль',
  'profile.edit':
    'Изменить',
  'profile.loginPrompt':
    'Войдите в систему, чтобы задать псевдоним и краткое описание своего профиля на форуме.',
  'profile.nicknameLabel':
    'Псевдоним на форуме',
  'profile.notSet':
    'Не задан',
  'profile.notSetBio':
    'Описание профиля не задано.',
  'profile.nicknameInput':
    'Псевдоним',
  'profile.nicknamePlaceholder':
    'Введите псевдоним',
  'profile.nicknameHint':
    'Псевдоним будет отображаться в ваших публикациях на форуме. Максимум 30 символов.',
  'profile.bioLabel':
    'Описание профиля',
  'profile.bioPlaceholder':
    'Расскажите о себе (необязательно)',
  'profile.bioHint':
    'Максимум 500 символов.',
  'profile.updated':
    'Профиль обновлен.',
  'profile.saving':
    'Сохранение...',
  'profile.saveFailed':
    'Не удалось сохранить. Повторите попытку позже.',
  'profile.loadFailed':
    'Не удалось загрузить. Повторите попытку позже.',
  'profile.followingEntry':
    'Мои подписки',
  'profile.postsLabel':
    'Мои публикации',
  'profile.emptyPosts':
    'Вы ещё ничего не опубликовали.',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'Открытый профиль',
  'publicProfile.avatar':
    'Аватар',
  'publicProfile.loading':
    'Загрузка...',
  'publicProfile.invalidLinkName':
    'Недействительная ссылка на открытый профиль',
  'publicProfile.invalidLinkBio':
    'Откройте профиль автора публикации на форуме.',
  'publicProfile.notFound':
    'Пользователь не найден',
  'publicProfile.anonymous':
    'Анонимный пользователь',
  'publicProfile.noBio':
    'У этого пользователя пока не задан открытый профиль.',
  'publicProfile.loadFailed':
    'Не удалось загрузить открытый профиль.',
  'publicProfile.postsLabel':
    'Публикации',
  'publicProfile.emptyPosts':
    'Этот пользователь ещё ничего не опубликовал.',

  'follow.label':
    'Подписаться на этого пользователя',
  'follow.action':
    'Подписаться',
  'follow.actionDone':
    'Вы подписаны',
  'follow.unfollow':
    'Отписаться',
  'follow.done':
    'Вы подписались на этого пользователя.',
  'follow.failed':
    'Не удалось подписаться. Повторите попытку позже.',

  'following.peopleLabel':
    'Пользователи, на которых вы подписаны',
  'following.postsLabel':
    'Публикации пользователей, на которых вы подписаны',
  'following.peopleLoading':
    'Загрузка списка подписок...',
  'following.emptyPeople':
    'Вы пока ни на кого не подписаны. Нажмите «Подписаться» под публикацией или подпишитесь через открытый профиль пользователя.',
  'following.emptyPosts':
    'Пользователи, на которых вы подписаны, ещё ничего не опубликовали.',
  'following.peopleFailed':
    'Не удалось загрузить список подписок.',
  'following.postsFailed':
    'Не удалось загрузить публикации.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'С возвращением',
  'login.body':
    '{site} — пространство для всех, кто делится мыслями. Для начала публикации достаточно Google-аккаунта — регистрироваться не нужно.',
  'login.browseFirst':
    'Сначала посмотрите главную страницу',
  'admin.skipToMain':
    'Перейти к основному содержанию',
  'admin.railLabel':
    'Меню администратора',
  'admin.railBrandAria':
    'Главная страница {site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'Основные функции',
  'admin.railGovernance':
    'Управление',
  'admin.railMode':
    'Режим администратора',
  'admin.railExit':
    'Вернуться на форум',
  'admin.topbarMenu':
    'Переключить меню администратора',
  'admin.statusOnline':
    'Подключение в порядке',
  'admin.topbarForum':
    'Форум',
  'admin.logoutFailed':
    'Не удалось выйти из системы. Повторите попытку позже.',
  'admin.navUsers':
    'Управление пользователями',
  'admin.navPosts':
    'Публикации на форуме',
  'admin.navReports':
    'Управление жалобами',
  'admin.navMonitor': 'Мониторинг системы',
  'admin.navLog': 'Журнал действий',
  'admin.navStats': 'Динамика содержимого',
  'admin.navExport': 'Экспорт и массовые действия',
  'admin.navSessions': 'Вход и сессии',
  'admin.navBlocks': 'Список блокировок IP',
  'admin.navAnnouncements': 'Объявления',
  'admin.listLoadFailed':
    'Не удалось загрузить список.',
  'admin.dlgClose':
    'Закрыть окно',
  'admin.dlgConfirm':
    'Подтвердить',
  'admin.dlgSave':
    'Сохранить',
  'admin.dlgApplyTags':
    'Применить метки',
  'admin.dlgNoTags':
    'Сейчас нет меток, которые можно применить. Сначала создайте их в разделе «Управление метками» ниже.',

  'monitor.title': 'Мониторинг системы',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'Живой просмотр состояния сервисов, объёма запросов, распределения задержек и счётчиков ограничителей частоты.',
  'monitor.refresh': 'Обновить',
  'monitor.refreshing': 'Загрузка…',
  'monitor.autoRefresh': 'Автообновление',
  'monitor.autoRefreshOn': 'Автообновление каждые {seconds} с',
  'monitor.autoRefreshOff': 'Автообновление приостановлено',
  'monitor.nextUpdate': 'Обновление через {seconds} с',
  'monitor.loadFailed': 'Не удалось загрузить данные мониторинга.',
  'monitor.loadFailedHint': 'Проверьте, что вы вошли как администратор, а сервер ещё работает.',
  'monitor.pausedHint': 'Автообновление приостановлено: на экране последний успешный результат.',
  'monitor.visibilityPaused': 'Страница в фоне, поэтому автообновление приостановлено.',
  'monitor.lastUpdated': 'Обновлено в {time}',
  'monitor.probeTook': 'Проверки зависимостей: {ms} мс',
  'monitor.unreachable': 'Сервер не отвечает. Экран застыл на последнем успешном результате.',
  'monitor.depsTitle': 'Состояние сервисов',
  'monitor.depsNote': 'При каждом чтении каждая зависимость проверяется один раз; таймаут на зависимость — 2 с, все три идут параллельно.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'Поисковый движок',
  'monitor.stateOk': 'Работает',
  'monitor.stateDown': 'Недоступен',
  'monitor.stateDisabled': 'Не включён',
  'monitor.depSearchFallback': 'ES_URL не задан, поэтому поиск использует сравнение по ключевым словам в MySQL.',
  'monitor.depDisabled': 'Клиент Redis не передан, поэтому медиафункции отключены.',
  'monitor.depLatency': 'Ответ за {ms} мс',
  'monitor.depKeys': 'Ключей: {count}',
  'monitor.depMemory': 'Память {size}',
  'monitor.depPoolUsage': 'Соединения {inUse}/{open} (максимум {max})',
  'monitor.depPoolWait': 'Ожиданий: {count}, всего {ms} мс',
  'monitor.depRedisPool': 'Попадания {hits} / промахи {misses}',
  'monitor.depEngineMysql': 'Сравнение по ключевым словам MySQL',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'Обзор запросов',
  'monitor.statUptime': 'Время работы',
  'monitor.statRequests': 'Всего запросов',
  'monitor.statErrorRate': 'Доля ошибок',
  'monitor.statP95': 'Задержка P95',
  'monitor.statInFlight': 'В обработке',
  'monitor.statGoroutines': 'Горутины',
  'monitor.statHeap': 'Память кучи',
  'monitor.statDbPool': 'Соединения с БД',
  'monitor.statRateLimited': 'Отклонено ограничителем',
  'monitor.statCountWithPeak': 'пик {peak}',
  'monitor.statCountWithInUse': 'занято {inUse}, свободно {idle}',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} логических ядер · циклов GC {gc}',
  'monitor.noData': 'Запросов пока нет.',
  'monitor.noDataBody': 'Запросы после запуска сервиса появятся здесь; сейчас эта область пуста.',
  'monitor.timelineTitle': 'Трафик за последние {minutes} минут',
  'monitor.timelineNote': 'Итоги накапливаются с момента запуска этого процесса и обнуляются при перезапуске; каждый процентиль задержки — верхняя граница сегмента гистограммы, поэтому принимает только дискретные значения. Минута без столбца означает, что в это время трафика не было.',
  'monitor.timelineLive': 'Текущий процесс',
  'monitor.timelineHistory': 'До перезапуска',
  'monitor.timelineLegendVolume': 'Запросы',
  'monitor.timelineLegendError': 'Ошибки 5xx',
  'monitor.timelinePeak': 'Пик {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'В базе нет доступных исторических агрегатов; если при запуске их не удалось прочитать (нет таблицы или нет прав), показывается только текущий процесс.',
  'monitor.routesTitle': 'По маршрутам',
  'monitor.routesNote': 'Пути нормализуются (числовые идентификаторы и адреса почты заменяются на :id), поэтому разные id одной конечной точки считаются вместе.',
  'monitor.colRoute': 'Маршрут',
  'monitor.colCount': 'Запросы',
  'monitor.colAvg': 'Среднее',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'Самый долгий',
  'monitor.colErrors': 'Ошибки',
  'monitor.routeOther': 'Прочие (достигнут лимит маршрутов)',
  'monitor.clientsTitle': 'Адреса источников',
  'monitor.clientsNote': 'Отсортированы по числу запросов. Адрес из X-Forwarded-For или X-Real-IP не проверен доверенным прокси — прежде чем действовать, убедитесь, что прокси перезаписывает эти заголовки.',
  'monitor.noClients': 'Источники ещё не отслеживаются.',
  'monitor.noClientsBody': 'Каждый запрос учитывается по адресу источника. Сейчас эта таблица пуста.',
  'monitor.clientsDropped': 'Число источников достигло предела {limit}; адреса, которые дольше всего не встречались ({count}), выведены из отслеживания. Эта строка означает, что список неполон, а не что приходили только эти посетители.',
  'monitor.colIp': 'Адрес',
  'monitor.colSource': 'Источник',
  'monitor.colRateLimited': 'Ограничены по частоте',
  'monitor.colBanned': 'Отклонены блокировкой',
  'monitor.colLastRoute': 'Последний запрос',
  'monitor.colActions': 'Действия',
  'monitor.colBlock': 'Заблокировать',
  'monitor.blocking': 'Блокировка…',
  'monitor.sourcePeer': 'Соединение',
  'monitor.sourceXff': 'X-Forwarded-For',
  'monitor.sourceRealIp': 'X-Real-IP',
  'monitor.trustLegacy': 'TRUSTED_PROXY_CIDRS не задан: сервер сохраняет прежнее поведение и берёт самый левый элемент X-Forwarded-For. Пока не подтверждено, что перед ним стоит прокси, который действительно перезаписывает эти заголовки, и что пользователи не могут его обойти, ограничение частоты и блокировки по IP обходятся одним подделанным заголовком, а адрес источника в журнале аудита не является доказательством.',
  'monitor.trustConfigured': 'Доверенные прокси настроены: X-Forwarded-For / X-Real-IP принимаются, только если соединение пришло из одной из этих сетей, иначе используется адрес соединения. Действующие сети: {cidrs}.',
  'monitor.trustBroken': 'TRUSTED_PROXY_CIDRS задан, но ни одну запись не удалось разобрать как диапазон CIDR ({declared}), поэтому по-прежнему действует прежнее поведение без настройки.',
  'monitor.trustPartial': 'Следующие записи TRUSTED_PROXY_CIDRS не удаётся разобрать как диапазоны CIDR ({invalid}), поэтому пересылаемые ими заголовки никогда не принимаются. Запросы, приходящие через эти диапазоны, группируются по адресу соединения, то есть делят один бюджет ограничения частоты и один запрос к списку блокировок.',
  'monitor.blockTitle': 'Заблокировать {ip}',
  'monitor.blockMessage': 'Запросы на запись с этого адреса (записи, комментарии, лайки, жалобы, загрузка изображений, переходы входа) будут отклоняться {duration}. Чтение не затрагивается. Заблокировать?',
  'monitor.blockReason': 'Заблокировано со страницы мониторинга',
  'monitor.blocked': '{ip} заблокирован',
  'monitor.blockFailed': 'Не удалось выполнить блокировку.',
  'monitor.limitsTitle': 'Ограничители частоты',
  'monitor.limitsNote': 'У каждой группы свой бюджет, соответствующий стоимости её конечных точек; блокировки — это все ответы 429 с момента запуска процесса.',
  'monitor.colLimiter': 'Ограничитель',
  'monitor.colBudget': 'Бюджет',
  'monitor.colAllowed': 'Пропущено',
  'monitor.colBlocked': 'Заблокировано',
  'monitor.colTracked': 'Отслеживаемых источников',
  'monitor.colBlockedRate': 'Доля заблокированных',
  'monitor.limitContent': 'Записи контента',
  'monitor.limitUpload': 'Загрузка изображений',
  'monitor.limitAuth': 'Вход через OAuth',
  'monitor.limitBudget': '{limit} за {window} с',
  'monitor.limitUnknown': '(неизвестно)',
  'monitor.noLimits': 'Нет доступных ограничителей.',
  'monitor.noLimitsBody': 'Ограничители частоты ещё не созданы, их счётчики недоступны.',

  'log.title': 'Журнал действий',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'Посмотрите каждое изменение в админке: кто, когда, над каким объектом и какие поля из чего в что превратились.',
  'log.refresh': 'Обновить',
  'log.loadFailed': 'Не удалось загрузить журнал действий.',
  'log.empty': 'Нет действий, подходящих под фильтры.',
  'log.emptyBody': 'Расширьте фильтры или убедитесь, что за этот период действий действительно не было.',
  'log.count': 'Записи {from}–{to} из {total}',
  'log.retention': 'Записи хранятся {days} дней, затем удаляются фоновой задачей. На этой странице нет кнопки удаления: журнал аудита, который может стирать собственные записи, аудитом не является.',
  'log.filterActor': 'Кто',
  'log.filterAction': 'Действие',
  'log.filterTargetType': 'Тип ресурса',
  'log.filterFrom': 'С',
  'log.filterTo': 'По',
  'log.filterAll': 'Все',
  'log.filterApply': 'Применить фильтры',
  'log.filterReset': 'Сбросить фильтры',
  'log.filterTargetHint': 'Нажмите на объект в любой строке, чтобы увидеть только относящиеся к нему действия.',
  'log.colTime': 'Время',
  'log.colActor': 'Кто',
  'log.colAction': 'Действие',
  'log.colTarget': 'Объект',
  'log.colChanges': 'Изменения',
  'log.colOrigin': 'Источник',
  'log.noChanges': '(без изменений полей)',
  'log.changedTo': 'изменено на',
  'log.removed': '(удалено)',
  'log.created': '(создано)',
  'log.requestId': 'request {id}',
  'log.page': 'Страница {page}',
  'log.targetUser': 'Пользователь',
  'log.targetPost': 'Запись',
  'log.targetComment': 'Комментарий',
  'log.targetReport': 'Жалоба',
  'log.targetTag': 'Метка',
  'log.targetSystem': 'Система',
  'log.actionUserSuspend': 'Заблокирован',
  'log.actionUserReinstate': 'Восстановлен',
  'log.actionUserTags': 'Метки изменены',
  'log.actionUserPost': 'Опубликовано от имени пользователя',
  'log.actionUserComment': 'Комментарий от имени пользователя',
  'log.actionUserContent': 'Его содержимое удалено',
  'log.actionPostCreate': 'Запись создана',
  'log.actionPostUpdate': 'Запись изменена',
  'log.actionPostDelete': 'Запись удалена',
  'log.actionCommentCreate': 'Комментарий создан',
  'log.actionCommentUpdate': 'Комментарий изменён',
  'log.actionCommentDelete': 'Комментарий удалён',
  'log.actionReportCreate': 'Жалоба создана',
  'log.actionReportResolve': 'Жалоба удовлетворена',
  'log.actionReportReject': 'Жалоба отклонена',
  'log.actionReportUpdate': 'Жалоба изменена',
  'log.actionReportDelete': 'Жалоба удалена',
  'log.actionTagCreate': 'Метка создана',
  'log.actionTagUpdate': 'Метка переименована',
  'log.actionTagDelete': 'Метка удалена',
  'log.fieldStatus': 'Статус аккаунта',
  'log.fieldContent': 'Содержимое',
  'log.fieldName': 'Название',
  'log.fieldTags': 'Метки',
  'log.fieldReason': 'Причина',
  'log.fieldAuthorEmail': 'Автор',
  'log.fieldReporterEmail': 'Заявитель',
  'log.fieldTargetType': 'Тип ресурса',
  'log.fieldTargetId': 'ID ресурса',
  'log.fieldPostId': 'ID записи',
  'log.fieldCommentId': 'ID комментария',
  'log.fieldPostIdShort': 'Запись',
  'log.fieldCommentIdShort': 'Комментарий',
  'log.fieldTarget': 'Объект',
  'log.fieldAssignmentsRemoved': 'Удалено привязок',
  'log.truncated': 'усечено',

  'stats.title': 'Динамика содержимого',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'Сколько новых пользователей, записей и комментариев появляется каждый день, и что активнее всего прямо сейчас.',
  'stats.refresh': 'Обновить',
  'stats.loadFailed': 'Не удалось загрузить статистику содержимого.',
  'stats.window': 'Показать последние',
  'stats.windowDays': '{days} дн.',
  'stats.windowClamped': '(не более 90 дней)',
  'stats.windowNote': 'Сутки отсчитываются в локальном часовом поясе машины сервера. Если сайт работает в UTC, а администратор в другом поясе, сегодняшние числа покажутся меньше — это часовой пояс, а не спад трафика.',
  'stats.generatedAt': 'Статистика собрана {time}',
  'stats.seriesTitle': 'Новое за день',
  'stats.seriesNote': 'У трёх графиков разные масштабы, поэтому они показаны отдельно, а не с наложением.',
  'stats.seriesUsers': 'Новые пользователи',
  'stats.seriesPosts': 'Новые записи',
  'stats.seriesComments': 'Новые комментарии',
  'stats.seriesEmpty': 'Нет данных за этот период.',
  'stats.totalsTitle': 'Итоги за период',
  'stats.totalsNote': 'Это количество, добавленное за период, а не текущие итоги сайта.',
  'stats.totalUsers': 'Новые пользователи',
  'stats.totalPosts': 'Новые записи',
  'stats.totalComments': 'Новые комментарии',
  'stats.totalLikes': 'Новые лайки',
  'stats.topPostsTitle': 'Популярные записи',
  'stats.topPostsNote': 'По числу комментариев и лайков, учитываются только записи за период.',
  'stats.topTagsTitle': 'Популярные метки',
  'stats.topTagsNote': 'По числу пользователей, без ограничения по времени: метка — это свойство, а не событие.',
  'stats.topAuthorsTitle': 'Активные авторы',
  'stats.topAuthorsNote': 'По числу записей за период, комментарии показаны отдельно.',
  'stats.colExcerpt': 'Фрагмент',
  'stats.colEngagement': 'Активность',
  'stats.colPosts': 'Записи',
  'stats.colComments': 'Комментарии',
  'stats.colUsers': 'Пользователи',
  'stats.colAuthor': 'Автор',
  'stats.empty': 'Нет данных за этот период.',
  'stats.emptyBody': 'Увеличьте период или убедитесь, что нового содержимого действительно не было.',
  'stats.engagement': '{comments} комм.・{likes} лайков',
  'stats.rank': '№ {rank}',

  'export.title': 'Экспорт и массовые действия',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'Выгружайте данные сайта в CSV для сверки или обрабатывайте сразу несколько аккаунтов.',
  'export.download': 'Скачать CSV',
  'export.downloading': 'Подготовка…',
  'export.exportTitle': 'Экспорт',
  'export.exportNote': 'Каждый экспорт ограничен 50 000 строк, дальше попадают только самые новые. Колонки всех трёх выгрузок совпадают с колонками страниц админки, поэтому сверка идёт напрямую.',
  'export.exportUsers': 'Список пользователей',
  'export.exportUsersNote': 'Email, статус, время создания и изменения, число записей и комментариев.',
  'export.exportPosts': 'Список записей',
  'export.exportPostsNote': 'Id, автор, первые 200 символов текста, время создания, комментарии и лайки.',
  'export.exportReports': 'Список жалоб',
  'export.exportReportsNote': 'Id, объект жалобы, жалобщик, причина, статус и история рассмотрения.',
  'export.safety': 'Файл начинается с BOM в UTF-8, поэтому Excel открывает его без кракозябров.',
  'export.safetyPrefix': 'Значения, начинающиеся с = + - @ или невидимого пробела, получают ведущий апостроф — именно это заставляет таблицу считать их текстом, а не выполнять как формулы. Префикс добавлен намеренно, не просите его убрать.',
  'export.batchTitle': 'Массовые действия',
  'export.batchNote': 'Кнопки активируются после выбора аккаунтов на странице Пользователи. Массовое действие применяется целиком или не применяется совсем — частичного результата не бывает.',
  'export.batchSuspend': 'Приостановить выбранных',
  'export.batchReinstate': 'Восстановить выбранных',
  'export.batchTags': 'Задать метки',
  'export.batchTagsNote': 'Семантика перезаписи: отправленный список становится результатом. Пустой список снимает все метки.',
  'export.batchConfirm': 'Применить «{action}» к {count} аккаунтам?',
  'export.batchConfirmTags': 'Перезаписать метки у {count} аккаунтов на {tags}?',
  'export.batchTagsPicker': 'Выбрать метки',
  'export.batchTagsNone': 'Без меток (снять все)',
  'export.batchRunning': 'Выполняется…',
  'export.batchDone': 'Обновлено аккаунтов: {updated}',
  'export.batchDoneUnchanged': 'из них {unchanged} уже были в нужном состоянии и не изменились',
  'export.batchSkipped': 'Пропущено: {count}',
  'export.batchMax': 'Не более 200 аккаунтов за раз',
  'export.gotoUsers': 'Перейти к пользователям',
  'export.noSelection': 'Сначала выберите аккаунты на странице Пользователи.',
  'export.selected': 'Выбрано аккаунтов: {count}',
  'export.clearSelection': 'Снять выделение',
  'export.selectionHint': 'Выделение сохраняется только на этой странице и пропадает при её закрытии.',

  'session.title': 'Вход и сессии',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'Посмотрите, какие входы ещё действительны, и принудительно завершите сессии аккаунта на всех устройствах.',
  'session.refresh': 'Обновить',
  'session.loadFailed': 'Не удалось загрузить список сессий.',
  'session.privacyTitle': 'Почему полный токен не показан',
  'session.privacyNote': 'Токен сам по себе является учётными данными для входа. Показаны только первые восемь символов — чтобы администратор мог отличить одну и ту же сессию в двух строках; этого недостаточно, чтобы войти от имени пользователя, в том числе тому, кто получит скриншот этой страницы. Это осознанное ограничение, а не незавершённая функция.',
  'session.expireNote': 'Сессия истекает после {hours} часов без активности; любой запрос продлевает её.',
  'session.filterEmail': 'Фильтр по аккаунту',
  'session.filterPlaceholder': 'Полный адрес электронной почты',
  'session.search': 'Найти',
  'session.clearFilter': 'Сбросить',
  'session.summary': '{total} сессий на сайте, проверено {scanned} ключей',
  'session.truncated': 'Просмотр достиг предела в {scanned} ключей и завершился раньше времени, поэтому список неполон.',
  'session.empty': 'Сейчас нет ни одной сессии.',
  'session.emptyBody': 'Никто не вошёл в систему, либо все сессии истекли.',
  'session.colUser': 'Аккаунт',
  'session.colToken': 'Сессия',
  'session.colCreated': 'Создана',
  'session.colExpires': 'Истекает',
  'session.colRemaining': 'Осталось',
  'session.colActions': 'Действие',
  'session.unknown': 'Неизвестно',
  'session.adminBadge': 'Администратор',
  'session.revoke': 'Принудительный выход',
  'session.revokeTitle': 'Принудительный выход из {email}',
  'session.revokeMessage': 'Текущие {count} сессий аккаунта будут немедленно аннулированы, а вход на всех устройствах сброшен. Пользователю придётся войти заново. Продолжить?',
  'session.revokeRunning': 'Отзыв…',
  'session.revokeDone': 'Отозвано сессий: {count}',
  'session.revokeNone': 'У этого аккаунта нет активных сессий',
  'session.revokeFailed': 'Не удалось подтвердить выход.',
  'session.revokeUnavailable': 'Это не значит, что отзыв не удался: просмотр достиг предела ключей, поэтому некоторые сессии могли не попасть в результат. Попробуйте ещё раз через мгновение.',
  'session.titleColumnNote': 'Только префикс для опознания, для входа не годится',

  'block.title': 'Список блокировок IP',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'Добавляйте в список адреса, злоупотребление которыми подтверждено. Список хранится в Redis — ни перезапуск, ни развёртывание его не снимают.',
  'block.refresh': 'Обновить',
  'block.loadFailed': 'Не удалось загрузить список блокировок.',
  'block.unavailable': 'Сайт не подключён к Redis, поэтому блокировка не включена.',
  'block.unavailableNote': 'Списку нужен тот же Redis, что и сессиям и медиа-токенам. Как только он подключён, страница заработает; до этого единственная защита — ограничение частоты (в пределах процесса, сбрасывается при перезапуске).',
  'block.add': 'Заблокировать',
  'block.addTitle': 'Заблокировать адрес IP',
  'block.addMessage': 'Заблокированный адрес отклоняется на любом запросе на запись (записи, комментарии, лайки, жалобы, загрузка изображений, переходы входа) до истечения срока. Чтение не затрагивается.',
  'block.ipLabel': 'Адрес IP',
  'block.ipPlaceholder': '203.0.113.9 или 2001:db8::1',
  'block.durationLabel': 'Срок',
  'block.reasonLabel': 'Причина',
  'block.reasonPlaceholder': 'Почему блокируется этот адрес (попадёт в журнал аудита)',
  'block.reasonHint': 'Причина идёт только в журнал аудита. Заблокированному её не показывают, и в публичном сообщении об ошибке она не появляется.',
  'block.blocking': 'Блокировка…',
  'block.done': '{ip} заблокирован',
  'block.removed': '{ip} разблокирован',
  'block.removedNone': '{ip} и не был заблокирован',
  'block.failed': 'Не удалось выполнить блокировку.',
  'block.unavailableService': 'Список блокировок недоступен (нет Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'Истекает',
  'block.colRemaining': 'Осталось',
  'block.colActions': 'Действие',
  'block.unblock': 'Разблокировать',
  'block.unblockTitle': 'Разблокировать {ip}',
  'block.unblockMessage': 'Этот адрес сразу получит обычный доступ на запись. Продолжить?',
  'block.empty': 'Список блокировок пуст.',
  'block.emptyBody': 'Ни один адрес не заблокирован. Это обычное состояние: блокировка — всегда решение администратора, и система никогда никого не блокирует автоматически.',
  'block.notAutoNote': 'Этот список не наполняется сам. Адрес, превысивший лимит частоты, получает только 429; сюда он автоматически не попадает, потому что тот же выход может принадлежать целому офису или целому NAT, и автоматическая блокировка задела бы их тоже.',
  'block.scopeNoteLabel': 'Охват',
  'block.notAutoNoteLabel': 'Никогда автоматически',
  'block.maxNoteLabel': 'Максимальная длительность',
  'block.scopeNote': 'Блокировка останавливает только запросы на запись. Читать записи, комментарии и статические файлы по-прежнему можно, а заблокированный человек может войти и видеть содержимое: это сделано намеренно, потому что точки чтения сознательно не ограничивают (иначе анонимные посетители не смогли бы пользоваться сайтом), и блокировка покрывает тот же набор.',
  'block.maxNote': 'Одна блокировка длится не больше 365 дней. Больший срок урезается до года: срок хранится числом, и «навсегда» превратилось бы в блокировку, о которой никто не помнит и которая сама не снимется.',
  'block.count': '{count} заблокированных адресов',
  'block.ipInvalid': 'Неверный адрес IP. Укажите адрес IPv4 или IPv6; диапазоны CIDR не поддерживаются.',
  'announce.label': 'Объявление сайта',
  'announce.publicNote': 'Объявление',
  'announce.closeAria': 'Закрыть это объявление',
  'announce.publishedOn': 'Опубликовано {date}',
  'announce.expiresOn': 'Истекает {date}',
  'announce.neverExpires': 'Без срока',
  'announce.pinnedBadge': 'Закреплено',
  'announce.title': 'Объявления',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'Показывать одно объявление над всеми страницами сайта. Действует только одно одновременно: публикация нового отключает предыдущее.',
  'announce.refresh': 'Обновить',
  'announce.loadFailed': 'Не удалось загрузить объявления.',
  'announce.new': 'Опубликовать новое объявление',
  'announce.edit': 'Изменить',
  'announce.deactivate': 'Отключить',
  'announce.reactivate': 'Включить снова',
  'announce.deleteNote': 'Объявления никогда не удаляются, только отключаются: история отвечает на вопрос, когда и кем оно было опубликовано.',
  'announce.bodyLabel': 'Текст объявления',
  'announce.bodyPlaceholder': 'Например: в четверг с 02:00 до 04:00 будет техническое обслуживание.',
  'announce.bodyHint': 'Не более 300 символов. Обычный текст, переносы строк сохраняются.',
  'announce.activeLabel': 'Показать сразу',
  'announce.expiryLabel': 'Срок действия',
  'announce.expiryNever': 'Никогда не истекает автоматически',
  'announce.expiryHours': 'через {hours} часов',
  'announce.expiryDays': 'через {days} дней',
  'announce.saving': 'Сохранение…',
  'announce.published': 'Объявление опубликовано',
  'announce.updated': 'Объявление обновлено',
  'announce.deactivated': 'Объявление отключено',
  'announce.reactivated': 'Объявление включено снова',
  'announce.saveFailed': 'Не удалось выполнить операцию с объявлением.',
  'announce.empty': 'Объявлений пока нет.',
  'announce.emptyBody': 'Как только вы опубликуете одно, оно появится вверху каждой страницы посетителя.',
  'announce.colBody': 'Текст',
  'announce.colState': 'Состояние',
  'announce.colAuthor': 'Опубликовал',
  'announce.colCreated': 'Опубликовано',
  'announce.colActions': 'Действие',
  'announce.stateActive': 'Показывается',
  'announce.stateInactive': 'Отключено',
  'announce.stateExpired': 'Истекло',
  'announce.confirmDeactivate': 'Публикация отключит текущее, и все сразу увидят новый текст. Продолжить?',
  'announce.confirmEdit': 'Изменить текст или срок действия этого объявления?',
  'announce.count': 'Всего {count}',
  'users.title':
    'Управление пользователями',
  'users.contentAction':
    'Действие с содержанием',
  'users.updateContentFailed':
    'Не удалось выполнить действие с содержанием.',
  'users.updated':
    'Содержимое обновлено.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'Просматривайте статистику активности пользователей форума, метки и состояние учетных записей, а также управляйте блокировками и модерацией контента по каждой публикации.',
  'users.refresh':
    'Обновить данные',
  'users.statTotal':
    'Всего пользователей',
  'users.statActive':
    'Активны',
  'users.statSuspended':
    'Заблокированы',
  'users.statContent':
    'Всего публикаций и комментариев',
  'users.count':
    '{count} пользователей',
  'users.tagsCount':
    '{count} меток',
  'users.loadFailed':
    'Не удалось загрузить данные пользователей.',
  'users.tagsLoadFailed':
    'Не удалось загрузить данные меток.',
  'users.panelTitle':
    'Пользователи форума',
  'users.tagsPanelTitle':
    'Метки пользователей',
  'users.colUser':
    'Пользователь',
  'users.colTags':
    'Метки',
  'users.colStatus':
    'Статус',
  'users.colPosts':
    'Публикации',
  'users.colComments':
    'Комментарии',
  'users.colLikes':
    'Нравится',
  'users.colLastActivity':
    'Последняя активность',
  'users.colActions':
    'Действия',
  'users.nicknameUnset':
    'Псевдоним не задан',
  'users.notSet':
    'Не задано',
  'users.statusActive':
    'Активен',
  'users.statusSuspended':
    'Заблокирован',
  'users.emptyTitle':
    'Сейчас нет данных о пользователях',
  'users.emptyBody':
    'Здесь может быть пусто, пока никто не войдёт на форум через Google.',
  'users.tagsEmptyTitle':
    'Сейчас нет меток',
  'users.tagsEmptyBody':
    'Сначала создайте метки, чтобы затем назначать их пользователям в списке.',
  'users.addTag':
    'Добавить метку',
  'users.colName':
    'Название',
  'users.colCreated':
    'Дата создания',
  'users.colUpdated':
    'Дата обновления',
  'users.renameTag':
    'Переименовать',
  'users.suspend':
    'Заблокировать',
  'users.restore':
    'Восстановить',
  'users.statusDialogTitle':
    '{action} этого пользователя',
  'users.statusSuspendMessage':
    'После этого {email} больше не сможет входить на форум. Существующие публикации и комментарии будут сохранены. Продолжить?',
  'users.statusRestoreMessage':
    'После этого {email} снова сможет входить на форум и публиковать сообщения. Продолжить?',
  'users.userSuspended':
    'Пользователь заблокирован.',
  'users.userRestored':
    'Пользователь восстановлен.',
  'users.updateStatusFailed':
    'Не удалось обновить статус пользователя.',
  'users.editTagsTitle':
    'Изменить метки · {user}',
  'users.editTagsMessage':
    'Выберите метки, которые нужно применить; отмена выбора всех меток эквивалентна удалению всех меток у этого пользователя.',
  'users.tagsUpdated':
    'Метки пользователя обновлены.',
  'users.updateTagsFailed':
    'Не удалось обновить метки пользователя.',
  'users.contentLoadFailed':
    'Не удалось загрузить содержимое.',
  'users.contentLoadFailedShort':
    'Не удалось загрузить содержимое.',
  'users.contentPanelTitle':
    'Контент пользователя',
  'users.contentCount':
    '{posts} публикаций · {comments} комментариев',
  'users.contentLoading':
    'Загрузка публикаций и комментариев…',
  'users.addPost':
    'Добавить публикацию',
  'users.addComment':
    'Добавить комментарий',
  'users.postsColumn':
    'Публикации',
  'users.commentsColumn':
    'Комментарии',
  'users.noPosts':
    'Нет публикаций',
  'users.noComments':
    'Нет комментариев',
  'users.postRef':
    'Публикация #{id}',
  'users.editRecordTitle':
    'Изменить {kind} #{id}',
  'users.deleteRecordTitle':
    'Удалить {kind} #{id}',
  'users.deleteRecordMessage':
    'После удаления восстановить данные невозможно; связанные отметки «Нравится» и связанные данные также будут удалены. Продолжить?',
  'users.contentLabel':
    'Контент',
  'users.addPostTitle':
    'Добавить публикацию',
  'users.addPostMessage':
    'Это содержание будет опубликовано от имени этого пользователя; поле автора нельзя подделать.',
  'users.postContentLabel':
    'Содержимое публикации',
  'users.postContentPlaceholder':
    'Введите содержимое публикации',
  'users.pickPostTitle':
    'Выбрать публикацию',
  'users.postIdLabel':
    'Публикация ID',
  'users.postIdPlaceholder':
    'ID публикации для комментария',
  'users.addCommentTitle':
    'Добавить комментарий',
  'users.commentContentLabel':
    'Содержимое комментария',
  'users.commentContentPlaceholder':
    'Введите содержимое комментария',
  'users.createTagTitle':
    'Добавить метку',
  'users.createTagMessage':
    'Метки можно использовать для классификации пользователей, например «модератор», «активный» или «заблокированный».',
  'users.tagNameLabel':
    'Название метки',
  'users.tagNamePlaceholder':
    'Не более 50 символов',
  'users.renameTagTitle':
    'Переименовать метку',
  'users.renameTagMessage':
    'Все пользователи, использующие эту метку, увидят новое название.',
  'users.deleteTagTitle':
    'Удалить метку «{name}»',
  'users.deleteTagMessage':
    'После удаления эта метка будет удалена у всех пользователей, и восстановить её будет невозможно. Продолжить?',
  'users.deleteTagConfirm':
    'Удалить метку',
  'users.tagCreated':
    'Метка создана.',
  'users.createTagFailed':
    'Не удалось создать метку.',
  'users.tagUpdated':
    'Метка обновлена.',
  'users.updateTagFailed':
    'Не удалось обновить метку.',
  'users.tagDeleted':
    'Метка удалена.',
  'users.deleteTagFailed':
    'Не удалось удалить метку.',
  'users.refreshDone':
    'Данные обновлены до актуального состояния.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'Admin Console',
  'users.signinBody':
    'Централизованное управление контентом делает каждый процесс модерации ясным, быстрым и отслеживаемым.',
  'users.signinStep1':
    'Безопасная проверка',
  'users.signinStep2':
    'Управление пользователями',
  'users.signinStep3':
    'Модерация контента',
  'users.signinPanelTitle':
    'Войти в Admin Console',
  'users.signinPanelBody':
    'Admin Console доступна только авторизованным администраторам Google. Войдите в систему от имени администратора.',
  'posts.title':
    'Форумные публикации',
  'posts.searching':
    'Поиск…',
  'posts.searchDegraded':
    '(Сервис поиска не включен; выполняется сопоставление по ключевым словам в базе данных)',
  'posts.searchSummary':
    'В запросе «{query}» найдено {total} записей; на этой странице показано {shown} записей',
  'posts.pendingCount':
    '{count} записей в очереди на обработку',
  'posts.pageSummary':
    'Страница {page} / {pages}, на этой странице {count} публикаций',
  'posts.listLoadFailed':
    'Не удалось загрузить список публикаций.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'Создавайте, редактируйте и удаляйте форумные публикации, а также управляйте модерацией на уровне комментариев.',
  'posts.toReports':
    'Управление жалобами',
  'posts.editorTitleNew':
    'Добавить публикацию',
  'posts.editorTitleEdit':
    'Изменить публикацию #{id}',
  'posts.editorNote':
    'Публикуйте от имени администратора; автор определяется по учётной записи, с которой выполнен вход, а запрос не позволяет подделать автора.',
  'posts.cancelEdit':
    'Отменить редактирование',
  'posts.contentLabel':
    'Содержимое публикации',
  'posts.contentPlaceholder':
    'Введите содержимое публикации',
  'posts.saveChanges':
    'Сохранить изменения',
  'posts.emptyContent':
    'Содержимое публикации не может быть пустым.',
  'posts.saving':
    'Сохранение…',
  'posts.saved':
    'Публикация обновлена.',
  'posts.published':
    'Публикация опубликована.',
  'posts.saveFailed':
    'Не удалось сохранить изменения.',
  'posts.deleteTitle':
    'Удалить публикацию #{id}',
  'posts.deleteMessage':
    'После удаления восстановить эту публикацию будет невозможно; все комментарии под ней также будут удалены. Продолжить?',
  'posts.deleteConfirm':
    'Удалить публикацию',
  'posts.deleted':
    'Публикация удалена.',
  'posts.deleteFailed':
    'Не удалось удалить публикацию.',
  'posts.edited':
    'Публикация обновлена.',
  'posts.editFailed':
    'Не удалось сохранить публикацию. Повторите попытку позже.',
  'posts.commentUpdated':
    'Комментарий обновлён.',
  'posts.commentActionFailed':
    'Не удалось выполнить действие с комментарием.',
  'posts.pickPostTitle':
    'Выбрать публикацию, к которой относится комментарий',
  'posts.pickPostMessage':
    'По умолчанию выбрана публикация в этой строке; чтобы привязать комментарий к другой публикации, измените её ID.',
  'posts.postIdLabel':
    'ID публикации',
  'posts.addCommentAtTitle':
    'Добавить комментарий в публикацию #{id}',
  'posts.addCommentMessage':
    'Этот комментарий будет опубликован от имени администратора.',
  'posts.commentContentLabel':
    'Содержимое комментария',
  'posts.commentContentPlaceholder':
    'Введите содержимое комментария',
  'posts.add':
    'Добавить',
  'posts.editCommentTitle':
    'Изменить комментарий #{id}',
  'posts.deleteCommentTitle':
    'Удалить комментарий #{id}',
  'posts.deleteCommentMessage':
    'После удаления восстановить комментарий будет невозможно. Продолжить?',
  'posts.listTitle':
    'Список публикаций',
  'posts.clearSearch':
    'Очистить поиск',
  'posts.searchLabel':
    'Ключевой поиск',
  'posts.searchPlaceholder':
    'Содержимое публикации или полный Email автора публикации',
  'posts.searchHint':
    'Отсортировано по релевантности; введите полный Email, чтобы найти все публикации этого пользователя. Поиск заменяет разделение на страницы; отображается не более 25 результатов.',
  'posts.searchTotal':
    'Всего результатов поиска: {total}',
  'posts.searchFailed':
    'Поиск не выполнен.',
  'posts.searchStatusFailed':
    'Поиск не выполнен',
  'posts.colContentImage':
    'Содержимое и изображение',
  'posts.colEngagement':
    'Вовлечённость',
  'posts.colComments':
    'Комментарии',
  'posts.colAuthor':
    'Автор',
  'posts.imageAlt':
    'Изображение публикации',
  'posts.likes':
    '{count} «Нравится»',
  'posts.author':
    'Автор: {name}',
  'posts.emptyTitle':
    'В форуме пока нет публикаций',
  'posts.emptyBody':
    'Можно создать первую публикацию, используя редактор выше.',
  'posts.emptySearchTitle':
    'Нет подходящих публикаций',
  'posts.emptySearchBody':
    'Для запроса «{query}» не найдено ни одной публикации. Попробуйте другой ключевой запрос.',
  'posts.commentCount':
    'Комментариев: {count}',
  'posts.noComments':
    'Комментариев пока нет',
  'posts.reportsTitle':
    'Жалобы на обработку',
  'posts.allReports':
    'Все жалобы',
  'posts.colReportedContent':
    'Содержимое жалобы',
  'posts.colReason':
    'Причина жалобы',
  'posts.colReporter':
    'Податель жалобы',
  'posts.colTime':
    'Время',
  'posts.colVerdict':
    'Решение',
  'posts.emptyReportsTitle':
    'Нет жалоб на обработку',
  'posts.emptyReportsBody':
    'Все жалобы рассмотрены.',
  'posts.verdictResolved':
    'Рассмотрено',
  'posts.verdictRejected':
    'Отклонено',
  'posts.verdictDialogTitle':
    'Установить для жалобы #{id} статус «{label}»',
  'posts.verdictDialogMessage':
    'После установки жалоба покинет список на обработке, но данные останутся в журнале управления жалобами. Продолжить?',
  'posts.verdictConfirm':
    'Установить{label}',
  'posts.verdictDone':
    'Жалоба отмечена как {label}.',
  'posts.verdictFailed':
    'Не удалось обновить статус жалобы.',
  'posts.pin': 'Закрепить',
  'posts.unpin': 'Открепить',
  'posts.pinTitle': 'Закрепить эту запись',
  'posts.unpinTitle': 'Открепить эту запись',
  'posts.pinMessage': 'Закреплённая запись остаётся в самом верху ленты у всех, и новые записи её не вытеснят.',
  'posts.unpinMessage': 'После открепления запись вернётся на своё место по времени.',
  'posts.pinDone': 'Закреплено',
  'posts.unpinDone': 'Откреплено',
  'posts.pinFailed': 'Не удалось закрепить запись.',
  'reports.title':
    'Управление жалобами',
  'reports.listSummary':
    'Всего: {count} · {filter}',
  'reports.listLoadFailed':
    'Не удалось загрузить список жалоб.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'Разберите каждую жалобу по отдельности: «Принято» удалит жалобу, а «Отклонено» сохранит исходный текст. Возврат к «На обработке» очистит время рассмотрения.',
  'reports.backToPosts':
    'Вернуться к публикациям',
  'reports.editorTitle':
    'Изменить жалобу #{id}',
  'reports.editorNote':
    'Можно исправить причину и статус жалобы; цель и податель уже указаны в существующей записи и здесь не изменяются.',
  'reports.targetTypeLabel':
    'Тип цели',
  'reports.targetIdLabel':
    'ID цели',
  'reports.reporterEmailLabel':
    'Email подателя жалобы',
  'reports.statusLabel':
    'Статус',
  'reports.reasonLabel':
    'Причина жалобы',
  'reports.reasonHint':
    'Не более 500 символов; другим администраторам текст будет показан как основание для решения.',
  'reports.filterLabel':
    'Фильтр по статусу',
  'reports.filterAll':
    'Все',
  'reports.listTitle':
    'Список жалоб',
  'reports.colTarget':
    'Содержимое цели',
  'reports.colReason':
    'Причина жалобы',
  'reports.colReporter':
    'Податель жалобы',
  'reports.colStatus':
    'Статус',
  'reports.targetGone':
    '(Содержимое удалено)',
  'reports.author':
    'Автор: {name}',
  'reports.deleteTitle':
    'Удалить жалобу #{id}',
  'reports.deleteMessage':
    'Будет удалена сама эта запись; содержимое, по которому подана жалоба, не пострадает, и восстановить её будет невозможно. Продолжить?',
  'reports.deleteConfirm':
    'Удалить жалобу',
  'reports.deleted':
    'Жалоба удалена.',
  'reports.deleteFailed':
    'Не удалось удалить жалобу.',
  'reports.updated':
    'Жалоба обновлена.',
  'reports.saveFailed':
    'Не удалось сохранить жалобу.',
  'reports.approveTitle':
    'Принять жалобу #{id}',
  'reports.approveGoneMessage':
    'Жалоба на {kind} #{id} больше не актуальна; она будет отмечена как рассмотренная.',
  'reports.approveMessage':
    'Вы навсегда удалите {kind} #{id}. Если это публикация, вместе с ней будут удалены все комментарии под ней. Эта жалоба будет отмечена как рассмотренная. Продолжить?',
  'reports.approveConfirm':
    'Принять и удалить',
  'reports.approveGoneDone':
    'Содержимое отсутствует; жалоба отмечена как рассмотренная.',
  'reports.approveDone':
    'Публикация удалена; жалоба отмечена как рассмотренная.',
  'reports.approveFailed':
    'Не удалось принять жалобу.',
  'reports.approveTitleGone':
    'Содержимое уже удалено; будет отмечена только жалоба',
  'reports.approveTitleFull':
    'Удалить жалобное содержимое и отметить жалобу как рассмотренную',
  'reports.rejectTitle':
    'Жалоба #{id} отклонена',
  'reports.rejectMessage':
    'Отклонение означает, что содержимое не требует обработки; оно будет сохранено без изменений. Продолжить?',
  'reports.rejectConfirm':
    'Установить статус «Отклонено»',
  'reports.rejectDone':
    'Жалоба отмечена как отклонённая.',
  'reports.statusFailed':
    'Не удалось обновить статус жалобы.',
  'reports.emptyTitle':
    'Сейчас нет жалоб',
  'reports.emptyBody':
    'По выбранным критериям записей нет.',
  'reports.rejectTitleAttr':
    'Сохранить содержимое и отметить жалобу как отклонённую',
  'kind.post':
    'публикация',
  'kind.comment':
    'комментарий',
  'reports.statusPending':
    'На обработке',
  'reports.statusResolved':
    'Рассмотрена',
  'reports.statusRejected':
    'Отклонена',
  'title.forum':
    '{site}',
  'title.login':
    'Вход | {site}',
  'title.newPost':
    'Новая публикация | {site}',
  'title.profile':
    'Профиль пользователя | {site}',
  'title.publicProfile':
    'Открытый профиль пользователя | {site}',
  'title.post':
    'Публикация｜{site}',
  'title.following':
    'Подписки | {site}',
  'title.adminUsers':
    'Управление пользователями | Панель администратора {brand}',
  'title.adminLogin':
    'Вход | Панель администратора {brand}',
  'title.adminPosts':
    'Публикации форума | Панель администратора {brand}',
  'title.adminReports':
    'Управление жалобами | Панель администратора {brand}',
  'title.adminMonitor': 'Мониторинг системы｜{brand} админка',
  'title.adminLog': 'Журнал действий｜{brand} админка',
  'title.adminStats': 'Динамика содержимого｜{brand} админка',
  'title.adminExport': 'Экспорт и массовые действия｜{brand} админка',
  'title.adminSessions': 'Вход и сессии｜{brand} админка',
  'title.adminBlocks': 'Список блокировок IP｜{brand} админка',
  'title.adminAnnouncements': 'Объявления｜{brand} админка',
};
