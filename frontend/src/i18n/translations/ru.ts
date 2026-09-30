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
};
