/*
 * pt-BR catalog (src/i18n/translations/pt-BR.ts)
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const ptBR: Record<MessageKey, string> = {
  'common.cancel':
    'Cancar',
  'common.save':
    'Salvar',
  'common.submitting':
    'Enviando...',
  'common.delete':
    'Excluir',
  'common.edit':
    'Editar',
  'common.search':
    'Buscar',
  'common.loading':
    'Carregando...',
  'common.loadFailed':
    'Falha ao carregar',
  'common.refresh':
    'Atualizar',
  'common.nextStep':
    'Próximo passo',
  'common.prevPage':
    'Página anterior',
  'common.nextPage':
    'Próxima página',
  'common.create':
    'Criar',
  'common.publish':
    'Publicar',
  'common.placeholder':
    '—',
  'common.backToHome':
    'Voltar para a página inicial',
  'common.backToForumHome':
    'Voltar à página inicial do fórum',
  'common.backOnePage':
    'Voltar para a página anterior',
  'error.request':
    'Tente novamente mais tarde.',
  'error.requestStatus':
    'A requisição falhou (HTTP {status})',
  'error.loginRequired':
    'Faça login antes de continuar.',
  'error.adminSessionExpired':
    'Sua sessão expirou e a tela de login será exibida.',
  'error.fallbackLoad':
    'Falha ao carregar',
  'error.fallbackSearch':
    'Falha na pesquisa',
  'error.fallbackLike':
    'Falha ao Curtir ou Descurtir',
  'error.fallbackComments':
    'Falha ao carregar comentários',
  'error.fallbackCommentPost':
    'Falha ao enviar comentário',
  'error.fallbackReport':
    'Falha ao denunciar',
  'error.fallbackProfile':
    'Falha ao carregar o perfil',
  'error.fallbackProfileSave':
    'Falha ao salvar',
  'error.fallbackPublish':
    'Formato de resposta de publicação inválido',
  'error.fallbackUpload':
    'Formato de resposta de envio de imagem inválido',
  'error.fallbackNotFound':
    'Usuário não encontrado.',
  'error.fallbackFollow':
    'Falha ao seguir.',
  'auth.checking':
    'Verificando login...',
  'auth.statusUnknown':
    'Status do login desconhecido.',
  'auth.feedLoggedIn':
    'Conectado. É possível publicar.',
  'auth.feedLoggedOut':
    'Faça login para publicar.',
  'auth.profileLoggedIn':
    'Conectado.',
  'auth.profileLoggedOut':
    'Faça login para configurar o perfil.',
  'auth.googleLogin':
    'Entrar com Google',
  'auth.loginWithGoogle':
    'Entrar com uma conta do Google',
  'auth.loginWithGoogleAdmin':
    'Entrar com uma conta de administrador do Google',
  'auth.logout':
    'Sair',
  'install.button':
    'Instalar aplicativo',
  'install.hint':
    'O navegador atual não oferece instalação automática. Abra o menu do navegador e selecione «Instalar aplicativo» ou «Adicionar à tela inicial».',
  'bottomNav.label':
    'Navegação principal',
  'bottomNav.home':
    'Início',
  'bottomNav.new':
    'Novo',
  'bottomNav.profile':
    'Perfil',
  'i18n.ariaLabel':
    'Selecionar idioma',
  'i18n.current':
    'Idioma: {name}',
  'feed.searchPlaceholder':
    'Pesquisar conteúdo das publicações',
  'feed.searchAriaLabel':
    'Pesquisar publicações',
  'feed.searchResultsLabel':
    'Resultados da pesquisa',
  'feed.postsLabel':
    'Publicações do fórum',
  'feed.searchFailed':
    'Falha na pesquisa. Tente novamente.',
  'feed.searching':
    'Pesquisando...',
  'feed.searchMore':
    'Carregar mais resultados de pesquisa...',
  'feed.searchMoreFailed':
    'Falha ao carregar mais resultados',
  'feed.searchFound':
    'Encontrados {total}',
  'feed.searchDegraded':
    '{base} (O serviço de pesquisa não está habilitado; a pesquisa atual usa comparação de palavras-chave no banco de dados).',
  'feed.searchTotal':
    'Há {total} resultados no total.',
  'feed.searchNoResults':
    'Não foram encontradas publicações que contenham «{query}».',
  'feed.loadingPosts':
    'Carregando publicações...',
  'feed.loadMorePosts':
    'Carregar mais publicações...',
  'feed.postsFailed':
    'Falha ao carregar publicações. Tente novamente.',
  'feed.postsFailedShort':
    'Falha ao carregar. Tente novamente.',
  'feed.scrollMore':
    'Deslize para baixo para carregar mais',
  'feed.endOfFeed':
    'Fim da lista.',
  'feed.noPosts':
    'Ainda não há publicações. Seja o primeiro a deixar sua opinião.',
  'feed.likeFailed':
    'Falha ao Curtir ou Descurtir. Tente novamente.',
  'post.authorAnonymous':
    'Anônimo',
  'post.report':
    'Denunciar publicação',
  'post.imageAlt':
    'Imagem da publicação',
  'post.unlike':
    'Descurtir',
  'post.like':
    'Curtir',
  'post.reply':
    'Responder',
  'comment.loading':
    'Carregando comentários...',
  'comment.none':
    'Ainda não há comentários',
  'comment.loadFailed':
    'Falha ao carregar comentários. Tente novamente.',
  'comment.placeholder':
    'Escreva um comentário...',
  'comment.max':
    'No máximo 2000 caracteres',
  'comment.submit':
    'Enviar comentário',
  'comment.failed':
    'Falha ao enviar comentário. Tente novamente.',
  'comment.report':
    'Denunciar',
  'comment.more':
    'Carregar mais comentários...',
  'report.reasonPlaceholder':
    'Insira o motivo da denúncia (no máximo 500 caracteres)',
  'report.note':
    'A denúncia será enviada aos administradores do site',
  'report.formLabel':
    'Campo de denúncia',
  'report.submit':
    'Enviar denúncia',
  'report.failed':
    'Falha ao enviar denúncia. Tente novamente.',
  'report.sent':
    'Denúncia enviada. Obrigado pelo retorno.',
  'newPost.avatarYou':
    'Você',
  'newPost.eyebrow':
    'NEW POST',
  'newPost.title':
    'Nova publicação',
  'newPost.loginFirst':
    'Faça login com o Google antes de publicar',
  'newPost.contentPlaceholder':
    'Compartilhe sua opinião...',
  'newPost.addImage':
    'Adicionar imagem',
  'newPost.emailPrivate':
    'Seu Email não será publicado',
  'newPost.submit':
    'Publicar',
  'newPost.publishing':
    'Publicação...',
  'newPost.uploading':
    'Enviando imagem...',
  'newPost.failed':
    'Falha ao publicar. Tente novamente em instantes.',
  'profile.eyebrow':
    'YOUR PROFILE',
  'profile.title':
    'Dados pessoais',
  'profile.edit':
    'Editar',
  'profile.loginPrompt':
    'Faça login para definir o apelido e a biografia do seu fórum.',
  'profile.nicknameLabel':
    'Apelido do fórum',
  'profile.notSet':
    'Não configurado',
  'profile.notSetBio':
    'Biografia ainda não definida.',
  'profile.nicknameInput':
    'Apelido',
  'profile.nicknamePlaceholder':
    'Digite um apelido',
  'profile.nicknameHint':
    'O apelido será exibido nas suas publicações, com até 30 caracteres.',
  'profile.bioLabel':
    'Biografia',
  'profile.bioPlaceholder':
    'Conte um pouco sobre você (opcional)',
  'profile.bioHint':
    'No máximo 500 caracteres.',
  'profile.updated':
    'Seus dados pessoais foram atualizados.',
  'profile.saving':
    'Salvando...',
  'profile.saveFailed':
    'Falha ao salvar. Tente novamente em instantes.',
  'profile.loadFailed':
    'Falha ao carregar. Tente novamente em instantes.',
  'profile.followingEntry':
    'Quem eu sigo',
  'publicProfile.eyebrow':
    'PUBLIC PROFILE',
  'publicProfile.title':
    'Dados pessoais públicos',
  'publicProfile.avatar':
    'Anônimo',
  'publicProfile.loading':
    'Carregando...',
  'publicProfile.invalidLinkName':
    'Link para a página pública inválido',
  'publicProfile.invalidLinkBio':
    'Acesse a página da pessoa que publicou a publicação no fórum.',
  'publicProfile.notFound':
    'Usuário não encontrado',
  'publicProfile.anonymous':
    'Usuário anônimo',
  'publicProfile.noBio':
    'Esta pessoa ainda não definiu seus dados públicos.',
  'publicProfile.loadFailed':
    'Falha ao carregar os dados públicos.',
  'publicProfile.postsLabel':
    'Publicações',
  'publicProfile.emptyPosts':
    'Esta pessoa ainda não publicou nada.',

  'follow.label':
    'Seguir esta pessoa',
  'follow.action':
    'Seguir',
  'follow.actionDone':
    'Seguindo',
  'follow.unfollow':
    'Deixar de seguir',
  'follow.done':
    'Agora você segue esta pessoa.',
  'follow.failed':
    'Falha ao seguir. Tente novamente em instantes.',

  'following.peopleLabel':
    'Pessoas que você segue',
  'following.postsLabel':
    'Publicações de quem você segue',
  'following.peopleLoading':
    'Carregando quem você segue...',
  'following.emptyPeople':
    'Você ainda não segue ninguém. Toque em «Seguir» em uma publicação ou siga alguém pelo perfil público.',
  'following.emptyPosts':
    'Ninguém que você segue publicou ainda.',
  'following.peopleFailed':
    'Falha ao carregar quem você segue.',
  'following.postsFailed':
    'Falha ao carregar as publicações.',
  'login.eyebrow':
    'MEMBER ACCESS',
  'login.title':
    'Bem-vindo de volta',
  'login.body':
    'O {site} é um espaço para todos que gostam de escrever suas ideias. Não há formulário de cadastro — com uma conta do Google, você já pode começar a publicar.',
  'login.browseFirst':
    'Veja a página inicial primeiro',
  'admin.skipToMain':
    'Ir para o conteúdo principal',
  'admin.railLabel':
    'Menu do painel administrativo',
  'admin.railBrandAria':
    'Início do {site}',
  'admin.consoleName':
    'Admin Console',
  'admin.railNavLabel':
    'Principais recursos',
  'admin.railGovernance':
    'Governança',
  'admin.railMode':
    'Modo de administrador',
  'admin.railExit':
    'Voltar ao fórum',
  'admin.topbarMenu':
    'Alternar o menu do painel administrativo',
  'admin.statusOnline':
    'Conexão normal',
  'admin.topbarForum':
    'Fórum',
  'admin.logoutFailed':
    'Falha ao sair. Tente novamente em instantes.',
  'admin.navUsers':
    'Gerenciamento de usuários',
  'admin.navPosts':
    'Publicações do fórum',
  'admin.navReports':
    'Gerenciamento de denúncias',
  'admin.listLoadFailed':
    'Falha ao carregar',
  'admin.dlgClose':
    'Fech. janela',
  'admin.dlgConfirm':
    'Confirmar',
  'admin.dlgSave':
    'Salvar',
  'admin.dlgApplyTags':
    'Aplicar etiquetas',
  'admin.dlgNoTags':
    'Neste momento, não há etiquetas a aplicar. Primeiro, crie uma em «Gerenciamento de etiquetas» abaixo.',
  'users.title':
    'Gerenciamento de usuários',
  'users.contentAction':
    'Conteúdo',
  'users.updateContentFailed':
    'Falha ao atualizar o conteúdo.',
  'users.updated':
    'O conteúdo foi atualizado.',
  'users.eyebrow':
    'USER MANAGEMENT',
  'users.copy':
    'Veja as estatísticas de atividade dos usuários do fórum, as etiquetas e o status das contas, e gerencie suspensões e a governança de conteúdo item por item.',
  'users.refresh':
    'Atualizar dados',
  'users.statTotal':
    'Total de usuários',
  'users.statActive':
    'Ativos',
  'users.statSuspended':
    'Suspensas',
  'users.statContent':
    'Total de publicações e comentários',
  'users.count':
    '{count} usuários',
  'users.tagsCount':
    '{count} etiquetas',
  'users.loadFailed':
    'Falha ao carregar os dados dos usuários.',
  'users.tagsLoadFailed':
    'Falha ao carregar as etiquetas.',
  'users.panelTitle':
    'Usuários do fórum',
  'users.tagsPanelTitle':
    'Etiquetas dos usuários',
  'users.colUser':
    'Usuário',
  'users.colTags':
    'Etiquetas',
  'users.colStatus':
    'Status',
  'users.colPosts':
    'Publicações',
  'users.colComments':
    'Comentários',
  'users.colLikes':
    'Curtir',
  'users.colLastActivity':
    'Última atividade',
  'users.colActions':
    'Ações',
  'users.nicknameUnset':
    'Apelido não definido',
  'users.notSet':
    'Não definido',
  'users.statusActive':
    'Ativo',
  'users.statusSuspended':
    'Suspensa',
  'users.emptyTitle':
    'Nenhum dado de usuário no momento',
  'users.emptyBody':
    'Este local só ficará vazio caso nenhum usuário tenha feito login no fórum com o Google.',
  'users.tagsEmptyTitle':
    'Nenhuma etiqueta no momento',
  'users.tagsEmptyBody':
    'Crie etiquetas primeiro para poder aplicá-las aos usuários na lista.',
  'users.addTag':
    'Adicionar etiqueta',
  'users.colName':
    'Nome',
  'users.colCreated':
    'Data de criação',
  'users.colUpdated':
    'Data de atualização',
  'users.renameTag':
    'Renomear',
  'users.suspend':
    'Suspender',
  'users.restore':
    'Restaurar',
  'users.statusDialogTitle':
    '{action} este usuário',
  'users.statusSuspendMessage':
    '{email} não poderá fazer login novamente no fórum. As publicações e comentários existentes serão preservados. Deseja continuar?',
  'users.statusRestoreMessage':
    '{email} recuperará o acesso ao fórum e a permissão para publicar. Deseja continuar?',
  'users.userSuspended':
    'Usuário suspenso.',
  'users.userRestored':
    'Usuário restaurado.',
  'users.updateStatusFailed':
    'Falha ao atualizar o status do usuário.',
  'users.editTagsTitle':
    'Editar etiqueta · {user}',
  'users.editTagsMessage':
    'Selecione as etiquetas a aplicar; desmarcar tudo equivale a remover todas as etiquetas deste usuário.',
  'users.tagsUpdated':
    'As etiquetas do usuário foram atualizadas.',
  'users.updateTagsFailed':
    'Falha ao atualizar as etiquetas do usuário.',
  'users.contentLoadFailed':
    'Falha ao carregar o conteúdo.',
  'users.contentLoadFailedShort':
    'Falha ao carregar o conteúdo.',
  'users.contentPanelTitle':
    'Conteúdo do usuário',
  'users.contentCount':
    '{posts} publicações · {comments} comentários',
  'users.contentLoading':
    'Carregando publicações e comentários…',
  'users.addPost':
    'Nova publicação',
  'users.addComment':
    'Novo comentário',
  'users.postsColumn':
    'Publicações',
  'users.commentsColumn':
    'Comentários',
  'users.noPosts':
    'Nenhuma publicação',
  'users.noComments':
    'Nenhum comentário',
  'users.postRef':
    'Publicação #{id}',
  'users.editRecordTitle':
    'Editar {kind} #{id}',
  'users.deleteRecordTitle':
    'Excluir {kind} #{id}',
  'users.deleteRecordMessage':
    'A exclusão é irreversível e também removerá curtidas e dados relacionados. Deseja continuar?',
  'users.contentLabel':
    'Conteúdo',
  'users.addPostTitle':
    'Nova publicação',
  'users.addPostMessage':
    'Esta publicação será publicada em nome do usuário; o campo do autor não pode ser falsificado.',
  'users.postContentLabel':
    'Conteúdo da publicação',
  'users.postContentPlaceholder':
    'Digite o conteúdo da publicação',
  'users.pickPostTitle':
    'Selecionar publicação',
  'users.postIdLabel':
    'ID da publicação',
  'users.postIdPlaceholder':
    'Código da publicação para comentar',
  'users.addCommentTitle':
    'Novo comentário',
  'users.commentContentLabel':
    'Conteúdo do comentário',
  'users.commentContentPlaceholder':
    'Digite o conteúdo do comentário',
  'users.createTagTitle':
    'Nova etiqueta',
  'users.createTagMessage':
    'As etiquetas podem ser usadas para classificar usuários, como "moderador", "ativo" ou "bloqueado".',
  'users.tagNameLabel':
    'Nome da etiqueta',
  'users.tagNamePlaceholder':
    'No máximo 50 caracteres',
  'users.renameTagTitle':
    'Renomear etiqueta',
  'users.renameTagMessage':
    'Todos os usuários que usam esta etiqueta verão o novo nome.',
  'users.deleteTagTitle':
    'Excluir a etiqueta "{name}"',
  'users.deleteTagMessage':
    'Após a exclusão, a etiqueta será removida de todos os usuários e não poderá ser restaurada. Deseja continuar?',
  'users.deleteTagConfirm':
    'Excluir etiqueta',
  'users.tagCreated':
    'Etiqueta criada.',
  'users.createTagFailed':
    'Falha ao criar a etiqueta.',
  'users.tagUpdated':
    'Etiqueta atualizada.',
  'users.updateTagFailed':
    'Falha ao atualizar a etiqueta.',
  'users.tagDeleted':
    'Etiqueta excluída.',
  'users.deleteTagFailed':
    'Falha ao excluir a etiqueta.',
  'users.refreshDone':
    'Atualização concluída com os dados mais recentes.',
  'users.signinEyebrow':
    '{brand} ADMIN',
  'users.signinTitle':
    'Painel administrativo',
  'users.signinBody':
    'Governança centralizada de conteúdo, tornando cada moderação clara, rápida e rastreável.',
  'users.signinStep1':
    'Verificação de segurança',
  'users.signinStep2':
    'Governança de usuários',
  'users.signinStep3':
    'Moderação de conteúdo',
  'users.signinPanelTitle':
    'Painel de gerenciamento',
  'users.signinPanelBody':
    'O Admin Console é destinado apenas a contas de administradores Google autorizadas. Entre com a identidade de administrador.',
  'posts.title':
    'Publicações do fórum',
  'posts.searching':
    'Pesquisando…',
  'posts.searchDegraded':
    '(O serviço de pesquisa não está habilitado; usa comparação por palavras-chave no banco de dados)',
  'posts.searchSummary':
    '{total} correspondências encontradas para "{query}"; esta página exibe {shown}',
  'posts.pendingCount':
    '{count} pendentes',
  'posts.pageSummary':
    'Página {page} de {pages}; {count} nesta página',
  'posts.listLoadFailed':
    'Falha ao carregar a lista de publicações.',
  'posts.eyebrow':
    'POST MODERATION',
  'posts.copy':
    'Crie, edite e exclua publicações do fórum e trate a governança de conteúdo no nível dos comentários.',
  'posts.toReports':
    'Gerenciamento de denúncias',
  'posts.editorTitleNew':
    'Nova publicação',
  'posts.editorTitleEdit':
    'Editar publicação #{id}',
  'posts.editorNote':
    'Publicado como administrador; o autor é obtido da sessão, e o autor da solicitação não pode ser falsificado.',
  'posts.cancelEdit':
    'Cancelar edição',
  'posts.contentLabel':
    'Conteúdo da publicação',
  'posts.contentPlaceholder':
    'Digite o conteúdo da publicação',
  'posts.saveChanges':
    'Salvar alterações',
  'posts.emptyContent':
    'O conteúdo da publicação não pode ficar vazio.',
  'posts.saving':
    'Salvando…',
  'posts.saved':
    'Publicação atualizada.',
  'posts.published':
    'Publicação publicada.',
  'posts.saveFailed':
    'Falha ao salvar.',
  'posts.deleteTitle':
    'Excluir publicação #{id}',
  'posts.deleteMessage':
    'A exclusão é irreversível e todos os comentários abaixo desta publicação também serão removidos. Deseja continuar?',
  'posts.deleteConfirm':
    'Excluir publicação',
  'posts.deleted':
    'Publicação excluída.',
  'posts.deleteFailed':
    'Falha ao excluir a publicação.',
  'posts.commentUpdated':
    'Comentário atualizado.',
  'posts.commentActionFailed':
    'Falha ao executar a ação do comentário.',
  'posts.pickPostTitle':
    'Selecionar a publicação a que o comentário pertence',
  'posts.pickPostMessage':
    'O padrão é a publicação desta linha; se quiser vinculá-la a outra, altere o ID da publicação correto.',
  'posts.postIdLabel':
    'ID da publicação',
  'posts.addCommentAtTitle':
    'Adicionar comentário à publicação #{id}',
  'posts.addCommentMessage':
    'Este comentário será publicado como administrador.',
  'posts.commentContentLabel':
    'Conteúdo do comentário',
  'posts.commentContentPlaceholder':
    'Digite o conteúdo do comentário',
  'posts.add':
    'Adicionar',
  'posts.editCommentTitle':
    'Editar comentário #{id}',
  'posts.deleteCommentTitle':
    'Excluir comentário #{id}',
  'posts.deleteCommentMessage':
    'A exclusão é irreversível. Deseja continuar?',
  'posts.listTitle':
    'Lista de publicações',
  'posts.clearSearch':
    'Limpar pesquisa',
  'posts.searchLabel':
    'Busca por palavra-chave',
  'posts.searchPlaceholder':
    'Conteúdo da publicação ou o Email completo do autor da publicação',
  'posts.searchHint':
    'Ordenadas por relevância; insira o Email completo para encontrar todas as publicações desse usuário. A busca substitui a paginação, exibindo no máximo 25 resultados.',
  'posts.searchTotal':
    'Resultados da busca: {total}',
  'posts.searchFailed':
    'A busca falhou.',
  'posts.searchStatusFailed':
    'Falha na busca',
  'posts.colContentImage':
    'Conteúdo e imagem',
  'posts.colEngagement':
    'Interações',
  'posts.colComments':
    'Comentários',
  'posts.colAuthor':
    'Autor',
  'posts.imageAlt':
    'Imagem da publicação',
  'posts.likes':
    '{count} Curtir',
  'posts.author':
    'Autor: {name}',
  'posts.emptyTitle':
    'Nenhum tópico de fórum no momento',
  'posts.emptyBody':
    'Use o editor acima para criar a primeira publicação.',
  'posts.emptySearchTitle':
    'Nenhuma publicação encontrada',
  'posts.emptySearchBody':
    'Nenhuma publicação correspondente a “{query}”. Tente outro termo de pesquisa.',
  'posts.commentCount':
    '{count} comentários',
  'posts.noComments':
    'Ainda não há comentários',
  'posts.reportsTitle':
    'Denúncias pendentes',
  'posts.allReports':
    'Todas as denúncias',
  'posts.colReportedContent':
    'Conteúdo denunciado',
  'posts.colReason':
    'Motivo da denúncia',
  'posts.colReporter':
    'Quem denunciou',
  'posts.colTime':
    'Horário',
  'posts.colVerdict':
    'Resultado',
  'posts.emptyReportsTitle':
    'Nenhuma denúncia pendente',
  'posts.emptyReportsBody':
    'Todas as denúncias foram analisadas.',
  'posts.verdictResolved':
    'Analisada',
  'posts.verdictRejected':
    'Sem fundamento',
  'posts.verdictDialogTitle':
    'Marcar a denúncia #{id} como “{label}”',
  'posts.verdictDialogMessage':
    'Após marcar, a denúncia sairá da lista de pendentes, mas os dados continuarão no painel de gerenciamento de denúncias. Continuar?',
  'posts.verdictConfirm':
    'Marcar como{label}',
  'posts.verdictDone':
    'A denúncia foi marcada como{label}.',
  'posts.verdictFailed':
    'Falha ao atualizar o status da denúncia.',
  'reports.title':
    'Gerenciamento de denúncias',
  'reports.listSummary':
    '{count} registros · {filter}',
  'reports.listLoadFailed':
    'Falha ao carregar a lista de denúncias.',
  'reports.eyebrow':
    'REPORT MODERATION',
  'reports.copy':
    'Analise cada denúncia: “Aprovada” remove o conteúdo denunciado; “Sem fundamento” mantém o conteúdo original. Voltar a “Pendente” limpa a data da análise.',
  'reports.backToPosts':
    'Voltar às publicações',
  'reports.editorTitle':
    'Editar a denúncia #{id}',
  'reports.editorNote':
    'É possível corrigir o motivo e o status da denúncia; o alvo e quem denunciou vêm dos registros existentes e não são alterados aqui.',
  'reports.targetTypeLabel':
    'Tipo de alvo',
  'reports.targetIdLabel':
    'ID do alvo',
  'reports.reporterEmailLabel':
    'Email de quem denunciou',
  'reports.statusLabel':
    'Status',
  'reports.reasonLabel':
    'Motivo da denúncia',
  'reports.reasonHint':
    'No máximo 500 caracteres; será exibido diretamente aos outros administradores como base para a análise.',
  'reports.filterLabel':
    'Filtrar por status',
  'reports.filterAll':
    'Todas',
  'reports.listTitle':
    'Lista de denúncias',
  'reports.colTarget':
    'Conteúdo do alvo',
  'reports.colReason':
    'Motivo da denúncia',
  'reports.colReporter':
    'Quem denunciou',
  'reports.colStatus':
    'Status',
  'reports.targetGone':
    '(conteúdo excluído)',
  'reports.author':
    'Autor: {name}',
  'reports.deleteTitle':
    'Excluir a denúncia #{id}',
  'reports.deleteMessage':
    'A exclusão afeta apenas este registro de denúncia; o conteúdo denunciado permanecerá inalterado e não poderá ser restaurado. Continuar?',
  'reports.deleteConfirm':
    'Excluir denúncia',
  'reports.deleted':
    'A denúncia foi excluída.',
  'reports.deleteFailed':
    'Falha ao excluir a denúncia.',
  'reports.updated':
    'A denúncia foi atualizada.',
  'reports.saveFailed':
    'Falha ao salvar a denúncia.',
  'reports.approveTitle':
    'Aprovar a denúncia #{id}',
  'reports.approveGoneMessage':
    'O conteúdo do {kind} #{id} denunciado já não existe; este registro será marcado como analisado.',
  'reports.approveMessage':
    'Será excluído permanentemente o {kind} #{id} denunciado (se for uma publicação, todos os comentários abaixo dessa publicação também serão removidos), e esta denúncia será marcada como analisada. Continuar?',
  'reports.approveConfirm':
    'Aprovar e excluir publicação',
  'reports.approveGoneDone':
    'O conteúdo já não existe; a denúncia foi marcada como analisada.',
  'reports.approveDone':
    'Publicação excluída e denúncia marcada como analisada.',
  'reports.approveFailed':
    'Falha ao aprovar a denúncia.',
  'reports.approveTitleGone':
    'O conteúdo foi excluído; apenas a denúncia será marcada.',
  'reports.approveTitleFull':
    'Excluir o conteúdo denunciado e marcar como analisada',
  'reports.rejectTitle':
    'Denúncia #{id} sem fundamento',
  'reports.rejectMessage':
    'Sem fundamento significa que o conteúdo denunciado não requer nenhuma ação; ele será mantido inalterado. Continuar?',
  'reports.rejectConfirm':
    'Marcar como sem fundamento',
  'reports.rejectDone':
    'A denúncia foi marcada como sem fundamento.',
  'reports.statusFailed':
    'Falha ao atualizar o status da denúncia.',
  'reports.emptyTitle':
    'Nenhuma denúncia no mo/* mento',
  'reports.emptyBody':
    'Não há registros para */ este filtro.',
  'reports.rejectTitleAttr':
    'Manter o conteúdo e marcar apenas a denúncia como sem fundamento',
  'kind.post':
    'Publicação',
  'kind.comment':
    'Comentário',
  'reports.statusPending':
    'Pendente',
  'reports.statusResolved':
    'Analisada',
  'reports.statusRejected':
    'Sem fundamento',
  'title.forum':
    '{site}',
  'title.login':
    'Entrar｜{site}',
  'title.newPost':
    'Nova publicação｜{site}',
  'title.profile':
    'Perfil｜{site}',
  'title.publicProfile':
    'Perfil público｜{site}',
  'title.following':
    'Seguindo｜{site}',
  'title.adminUsers':
    'Gerenciamento de usuários｜Painel administrativo {brand}',
  'title.adminLogin':
    'Entrar｜Painel administrativo {brand}',
  'title.adminPosts':
    'Publicações do fórum｜Painel administrativo {brand}',
  'title.adminReports':
    'Gerenciamento de denúncias｜Painel administrativo {brand}',
};
