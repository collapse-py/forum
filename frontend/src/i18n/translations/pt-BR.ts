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
  'common.confirm': 'Confirm',
  'common.updateFailed': 'Update failed',
  'common.selectAll': 'Select all',
  'common.oneHour': '1 hora',
  'common.oneDay': '1 dia',
  'common.sevenDays': '7 dias',
  'common.thirtyDays': '30 dias',
  'common.oneYear': '1 ano',
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
  'admin.navMonitor': 'Monitoramento do sistema',
  'admin.navLog': 'Registro de ações',
  'admin.navStats': 'Tendências de conteúdo',
  'admin.navExport': 'Exportação e ações em massa',
  'admin.navSessions': 'Logins e sessões',
  'admin.navBlocks': 'Lista de bloqueio de IP',
  'admin.navAnnouncements': 'Avisos',
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

  'monitor.title': 'Monitoramento do sistema',
  'monitor.eyebrow': 'SYSTEM MONITORING',
  'monitor.copy': 'Visão ao vivo da saúde dos serviços, volume de requisições, distribuição de latência e contadores dos limitadores de taxa.',
  'monitor.refresh': 'Atualizar',
  'monitor.refreshing': 'Carregando…',
  'monitor.autoRefresh': 'Atualização automática',
  'monitor.autoRefreshOn': 'Atualização automática a cada {seconds} s',
  'monitor.autoRefreshOff': 'Atualização automática pausada',
  'monitor.nextUpdate': 'Atualização em {seconds} s',
  'monitor.loadFailed': 'Falha ao carregar os dados de monitoramento.',
  'monitor.loadFailedHint': 'Confirme que você está autenticado como administrador e que o backend continua no ar.',
  'monitor.pausedHint': 'A atualização automática está pausada; a tela mostra a última leitura bem-sucedida.',
  'monitor.visibilityPaused': 'A página está em segundo plano, então a atualização automática está pausada.',
  'monitor.lastUpdated': 'Atualizado às {time}',
  'monitor.probeTook': 'Sondas de dependências: {ms} ms',
  'monitor.unreachable': 'O backend não está respondendo. A tela ficou na última leitura bem-sucedida.',
  'monitor.depsTitle': 'Saúde dos serviços',
  'monitor.depsNote': 'Cada leitura sonda cada dependência uma vez; o tempo limite por dependência é de 2 s e as três rodam em paralelo.',
  'monitor.depMysql': 'MySQL',
  'monitor.depRedis': 'Redis',
  'monitor.depSearch': 'Mecanismo de busca',
  'monitor.stateOk': 'Operacional',
  'monitor.stateDown': 'Inacessível',
  'monitor.stateDisabled': 'Não ativado',
  'monitor.depSearchFallback': 'ES_URL não está definido, então a busca usa a comparação por palavra-chave do MySQL.',
  'monitor.depDisabled': 'Nenhum cliente Redis foi injetado, então os recursos de mídia estão desativados.',
  'monitor.depLatency': 'Respondeu em {ms} ms',
  'monitor.depKeys': '{count} chaves',
  'monitor.depMemory': 'Memória {size}',
  'monitor.depPoolUsage': 'Conexões {inUse}/{open} (máximo {max})',
  'monitor.depPoolWait': '{count} esperas, {ms} ms no total',
  'monitor.depRedisPool': 'Acertos {hits} / falhas {misses}',
  'monitor.depEngineMysql': 'Comparação por palavra-chave do MySQL',
  'monitor.depEngineEs': 'Elasticsearch',
  'monitor.statsTitle': 'Resumo de requisições',
  'monitor.statUptime': 'Tempo no ar',
  'monitor.statRequests': 'Requisições totais',
  'monitor.statErrorRate': 'Taxa de erro',
  'monitor.statP95': 'Latência P95',
  'monitor.statInFlight': 'Em andamento',
  'monitor.statGoroutines': 'Goroutines',
  'monitor.statHeap': 'Memória de heap',
  'monitor.statDbPool': 'Conexões com o banco',
  'monitor.statRateLimited': 'Bloqueadas por limite',
  'monitor.statCountWithPeak': 'pico {peak}',
  'monitor.statCountWithInUse': '{inUse} em uso, {idle} ociosas',
  'monitor.statBlockedSplit': '4xx {client} / 5xx {server}',
  'monitor.goVersion': 'Go {version} · {cpu} núcleos lógicos · {gc} ciclos de GC',
  'monitor.noData': 'Ainda não há requisições.',
  'monitor.noDataBody': 'As requisições desde a inicialização do serviço aparecerão aqui; este painel está vazio por enquanto.',
  'monitor.timelineTitle': 'Tráfico dos últimos {minutes} minutos',
  'monitor.timelineNote': 'Os totais são acumulados desde o início deste processo e zeram no reinício; cada percentil de latência é o limite superior de uma faixa do histograma, então só assume valores discretos. Um minuto sem barra significa que não houve tráfego naquele momento.',
  'monitor.timelineLive': 'Este processo',
  'monitor.timelineHistory': 'Antes do reinício',
  'monitor.timelineLegendVolume': 'Requisições',
  'monitor.timelineLegendError': 'Erros 5xx',
  'monitor.timelinePeak': 'Pico {count}',
  'monitor.timelineHour': '{time}',
  'monitor.timelineNoHistory': 'Não há agregados históricos disponíveis no banco; se a inicialização não conseguir lê-los (tabela ausente ou sem permissão), só o processo atual é mostrado.',
  'monitor.routesTitle': 'Por rota',
  'monitor.routesNote': 'Os caminhos são normalizados (identificadores numéricos e e-mails viram :id), então ids diferentes da mesma rota são contados juntos.',
  'monitor.colRoute': 'Rota',
  'monitor.colCount': 'Requisições',
  'monitor.colAvg': 'Média',
  'monitor.colP50': 'P50',
  'monitor.colP95': 'P95',
  'monitor.colMax': 'Mais lenta',
  'monitor.colErrors': 'Erros',
  'monitor.routeOther': 'Outras (limite de rotas atingido)',
  'monitor.limitsTitle': 'Limitadores de taxa',
  'monitor.limitsNote': 'Cada grupo tem um orçamento próprio, dimensionado pelo custo de seus endpoints; os bloqueios contam todas as respostas 429 desde o início deste processo.',
  'monitor.colLimiter': 'Limitador',
  'monitor.colBudget': 'Orçamento',
  'monitor.colAllowed': 'Permitidas',
  'monitor.colBlocked': 'Bloqueadas',
  'monitor.colTracked': 'Origens rastreadas',
  'monitor.colBlockedRate': 'Taxa de bloqueio',
  'monitor.limitContent': 'Escritas de conteúdo',
  'monitor.limitUpload': 'Envio de imagens',
  'monitor.limitAuth': 'Login via OAuth',
  'monitor.limitBudget': '{limit} a cada {window} s',
  'monitor.limitUnknown': '(desconhecido)',
  'monitor.noLimits': 'Nenhum limitador disponível.',
  'monitor.noLimitsBody': 'Os limitadores de taxa ainda não foram criados, então seus contadores não estão disponíveis.',

  'log.title': 'Registro de ações',
  'log.eyebrow': 'ADMIN ACTION LOG',
  'log.copy': 'Consulte cada alteração feita no admin: quem, quando, em qual alvo e quais campos passaram de quê para quê.',
  'log.refresh': 'Atualizar',
  'log.loadFailed': 'Falha ao carregar o registro de ações.',
  'log.empty': 'Nenhuma ação corresponde aos filtros.',
  'log.emptyBody': 'Amplie os filtros ou confirme que realmente não houve ação nesse período.',
  'log.count': 'Registros {from}–{to} de {total}',
  'log.retention': 'Os registros são mantidos por {days} dias e depois removidos por uma tarefa em segundo plano. Esta página não tem botão para apagar registros — um log de auditoria que pode apagar as próprias entradas não é um log de auditoria.',
  'log.filterActor': 'Responsável',
  'log.filterAction': 'Ação',
  'log.filterTargetType': 'Tipo de recurso',
  'log.filterFrom': 'De',
  'log.filterTo': 'Até',
  'log.filterAll': 'Todos',
  'log.filterApply': 'Aplicar filtros',
  'log.filterReset': 'Limpar filtros',
  'log.filterTargetHint': 'Clique no alvo de qualquer linha para ver só as ações que o afetaram.',
  'log.colTime': 'Hora',
  'log.colActor': 'Responsável',
  'log.colAction': 'Ação',
  'log.colTarget': 'Alvo',
  'log.colChanges': 'Alterações',
  'log.colOrigin': 'Origem',
  'log.noChanges': '(sem alteração de campo)',
  'log.changedTo': 'alterado para',
  'log.removed': '(removido)',
  'log.created': '(criado)',
  'log.requestId': 'request {id}',
  'log.page': 'Página {page}',
  'log.targetUser': 'Usuário',
  'log.targetPost': 'Postagem',
  'log.targetComment': 'Comentário',
  'log.targetReport': 'Denúncia',
  'log.targetTag': 'Tag',
  'log.targetSystem': 'Sistema',
  'log.actionUserSuspend': 'Suspenso',
  'log.actionUserReinstate': 'Reativado',
  'log.actionUserTags': 'Tags alteradas',
  'log.actionUserPost': 'Publicado como usuário',
  'log.actionUserComment': 'Comentado como usuário',
  'log.actionUserContent': 'Conteúdo dele removido',
  'log.actionPostCreate': 'Postagem criada',
  'log.actionPostUpdate': 'Postagem editada',
  'log.actionPostDelete': 'Postagem removida',
  'log.actionCommentCreate': 'Comentário criado',
  'log.actionCommentUpdate': 'Comentário editado',
  'log.actionCommentDelete': 'Comentário removido',
  'log.actionReportCreate': 'Denúncia criada',
  'log.actionReportResolve': 'Denúncia procedente',
  'log.actionReportReject': 'Denúncia improcedente',
  'log.actionReportUpdate': 'Denúncia editada',
  'log.actionReportDelete': 'Denúncia removida',
  'log.actionTagCreate': 'Tag criada',
  'log.actionTagUpdate': 'Tag renomeada',
  'log.actionTagDelete': 'Tag removida',
  'log.fieldStatus': 'Status da conta',
  'log.fieldContent': 'Conteúdo',
  'log.fieldName': 'Nome',
  'log.fieldTags': 'Tags',
  'log.fieldReason': 'Motivo',
  'log.fieldAuthorEmail': 'Autor',
  'log.fieldReporterEmail': 'Denunciante',
  'log.fieldTargetType': 'Tipo de recurso',
  'log.fieldTargetId': 'Id do recurso',
  'log.fieldPostId': 'Id da postagem',
  'log.fieldCommentId': 'Id do comentário',
  'log.fieldPostIdShort': 'Postagem',
  'log.fieldCommentIdShort': 'Comentário',
  'log.fieldTarget': 'Alvo',
  'log.fieldAssignmentsRemoved': 'Vínculos removidos',
  'log.truncated': 'truncado',

  'stats.title': 'Tendências de conteúdo',
  'stats.eyebrow': 'CONTENT TRENDS',
  'stats.copy': 'Quantos usuários, postagens e comentários novos chegam por dia, e quais postagens, tags e autores estão mais ativos agora.',
  'stats.refresh': 'Atualizar',
  'stats.loadFailed': 'Falha ao carregar as estatísticas de conteúdo.',
  'stats.window': 'Mostrar os últimos',
  'stats.windowDays': '{days} dias',
  'stats.windowClamped': '(até 90 dias)',
  'stats.windowNote': 'Cada dia é cortado no fuso horário local da máquina do servidor. Se o site roda em UTC e a administração está em outro fuso, os números de hoje parecem baixos: é o fuso, não queda de tráfego.',
  'stats.generatedAt': 'Estatísticas geradas em {time}',
  'stats.seriesTitle': 'Novos por dia',
  'stats.seriesNote': 'As três curvas têm escalas diferentes, então são mostradas separadamente em vez de sobrepostas.',
  'stats.seriesUsers': 'Novos usuários',
  'stats.seriesPosts': 'Novas postagens',
  'stats.seriesComments': 'Novos comentários',
  'stats.seriesEmpty': 'Sem dados neste período.',
  'stats.totalsTitle': 'Totais do período',
  'stats.totalsNote': 'São as quantidades adicionadas neste período, não os totais atuais do site.',
  'stats.totalUsers': 'Novos usuários',
  'stats.totalPosts': 'Novas postagens',
  'stats.totalComments': 'Novos comentários',
  'stats.totalLikes': 'Novos curtidas',
  'stats.topPostsTitle': 'Postagens populares',
  'stats.topPostsNote': 'Ordenadas por comentários e curtidas, contando só as publicadas no período.',
  'stats.topTagsTitle': 'Tags populares',
  'stats.topTagsNote': 'Ordenadas por quantos usuários as têm, sem limite de tempo: uma tag é um atributo, não um evento.',
  'stats.topAuthorsTitle': 'Autores ativos',
  'stats.topAuthorsNote': 'Ordenados por postagens no período, com os comentários à parte.',
  'stats.colExcerpt': 'Trecho',
  'stats.colEngagement': 'Interação',
  'stats.colPosts': 'Postagens',
  'stats.colComments': 'Comentários',
  'stats.colUsers': 'Usuários',
  'stats.colAuthor': 'Autor',
  'stats.empty': 'Sem dados neste período.',
  'stats.emptyBody': 'Amplie o período ou confirme que realmente não houve conteúdo novo.',
  'stats.engagement': '{comments} comentários・{likes} curtidas',
  'stats.rank': 'Nº {rank}',

  'export.title': 'Exportação e ações em massa',
  'export.eyebrow': 'EXPORT & BULK ACTIONS',
  'export.copy': 'Exporte os dados do site em CSV para conferência ou atue em várias contas de uma vez.',
  'export.download': 'Baixar CSV',
  'export.downloading': 'Preparando…',
  'export.exportTitle': 'Exportar',
  'export.exportNote': 'Cada exportação é limitada a 50 mil linhas; além disso, só entram as mais recentes. As colunas das três exportações são iguais às das páginas de administração, então a conferência é direta.',
  'export.exportUsers': 'Lista de usuários',
  'export.exportUsersNote': 'E-mail, status, datas de criação e atualização, contagem de postagens e comentários.',
  'export.exportPosts': 'Lista de postagens',
  'export.exportPostsNote': 'Id, autor, primeiros 200 caracteres do conteúdo, data de criação, comentários e curtidas.',
  'export.exportReports': 'Lista de denúncias',
  'export.exportReportsNote': 'Id, alvo denunciado, denunciante, motivo, status e registro da análise.',
  'export.safety': 'O arquivo começa com uma marca de ordem de bytes em UTF-8, então o Excel abre sem caracteres quebrados.',
  'export.safetyPrefix': 'Valores que começam com = + - @ ou com um espaço invisível recebem um apóstrofo na frente — é isso que faz a planilha tratá-los como texto em vez de executá-los como fórmula. O prefixo é proposital; não peça para removê-lo.',
  'export.batchTitle': 'Ações em massa',
  'export.batchNote': 'Os botões ficam ativos depois que você seleciona contas na página Usuários. Uma ação em massa é aplicada inteira ou não é aplicada; não existe resultado parcial.',
  'export.batchSuspend': 'Suspender seleção',
  'export.batchReinstate': 'Reativar seleção',
  'export.batchTags': 'Aplicar tags',
  'export.batchTagsNote': 'Semântica de sobrescrita: a lista enviada passa a ser o resultado. Enviar lista vazia remove todas as tags.',
  'export.batchConfirm': 'Aplicar “{action}” a {count} contas?',
  'export.batchConfirmTags': 'Sobrescrever as tags de {count} contas com {tags}?',
  'export.batchTagsPicker': 'Escolher tags',
  'export.batchTagsNone': 'Nenhuma tag (remover todas)',
  'export.batchRunning': 'Processando…',
  'export.batchDone': '{updated} contas atualizadas',
  'export.batchDoneUnchanged': 'destas, {unchanged} já estavam no estado alvo e não mudaram',
  'export.batchSkipped': '{count} ignoradas',
  'export.batchMax': 'No máximo 200 contas por lote',
  'export.gotoUsers': 'Ir para Usuários',
  'export.noSelection': 'Selecione primeiro as contas na página Usuários.',
  'export.selected': '{count} contas selecionadas',
  'export.clearSelection': 'Limpar seleção',
  'export.selectionHint': 'A seleção fica só nesta página e some ao fechá-la.',

  'session.title': 'Logins e sessões',
  'session.eyebrow': 'SESSIONS',
  'session.copy': 'Veja quais logins ainda valem e force o encerramento de sessão de uma conta em todos os dispositivos.',
  'session.refresh': 'Atualizar',
  'session.loadFailed': 'Falha ao carregar a lista de sessões.',
  'session.privacyTitle': 'Por que o token completo não aparece',
  'session.privacyNote': 'O token é ele próprio a credencial de login. Só os oito primeiros caracteres são exibidos, para que a administração reconheça duas linhas como a mesma sessão — e isso não basta para que alguém entre como aquele usuário, nem mesmo quem receber um print desta página. É um limite proposital, não um recurso inacabado.',
  'session.expireNote': 'Uma sessão expira após {hours} horas sem atividade; qualquer requisição a renova.',
  'session.filterEmail': 'Filtrar por conta',
  'session.filterPlaceholder': 'Endereço de e-mail completo',
  'session.search': 'Buscar',
  'session.clearFilter': 'Limpar',
  'session.summary': '{total} sessões no site, {scanned} chaves verificadas',
  'session.truncated': 'A varredura atingiu o limite de {scanned} chaves e parou antes, então esta lista está incompleta.',
  'session.empty': 'Nenhuma sessão no momento.',
  'session.emptyBody': 'Ninguém está conectado, ou todas as sessões expiraram.',
  'session.colUser': 'Conta',
  'session.colToken': 'Sessão',
  'session.colCreated': 'Criada',
  'session.colExpires': 'Expira',
  'session.colRemaining': 'Restante',
  'session.colActions': 'Ação',
  'session.unknown': 'Desconhecido',
  'session.adminBadge': 'Administração',
  'session.revoke': 'Forçar encerramento',
  'session.revokeTitle': 'Forçar encerramento de {email}',
  'session.revokeMessage': 'As {count} sessões atuais da conta serão invalidadas na hora e o login será apagado em todos os dispositivos. A pessoa terá que entrar de novo. Continuar?',
  'session.revokeRunning': 'Revogando…',
  'session.revokeDone': '{count} sessões revogadas',
  'session.revokeNone': 'Esta conta não tem sessões ativas',
  'session.revokeFailed': 'Não foi possível confirmar o encerramento.',
  'session.revokeUnavailable': 'Isso não significa que a revogação falhou: a varredura atingiu o limite de chaves, então algumas sessões podem não ter sido alcançadas. Tente de novo em instantes.',
  'session.titleColumnNote': 'Apenas prefixo identificador, inutilizável para entrar',

  'block.title': 'Lista de bloqueio de IP',
  'block.eyebrow': 'IP BLOCKLIST',
  'block.copy': 'Coloque na lista os endereços de origem cujo abuso está confirmado. A lista fica no Redis: nem reiniciar nem implantar a limpa.',
  'block.refresh': 'Atualizar',
  'block.loadFailed': 'Falha ao carregar a lista de bloqueio.',
  'block.unavailable': 'Este site não tem conexão com o Redis, então o bloqueio não está ativo.',
  'block.unavailableNote': 'A lista precisa do mesmo Redis que as sessões e os tokens de mídia. Depois de conectá-lo, esta página funciona; até lá a única proteção é a limitação de taxa (por processo, zerada no reinício).',
  'block.add': 'Bloquear',
  'block.addTitle': 'Bloquear um endereço IP',
  'block.addMessage': 'Um endereço bloqueado é recusado em toda requisição de escrita (postagens, comentários, curtidas, denúncias, uploads de imagem e redirecionamentos de login) até o prazo terminar. A leitura não é afetada.',
  'block.ipLabel': 'Endereço IP',
  'block.ipPlaceholder': '203.0.113.9 ou 2001:db8::1',
  'block.durationLabel': 'Duração',
  'block.reasonLabel': 'Motivo',
  'block.reasonPlaceholder': 'Por que este endereço está sendo bloqueado (fica no registro de auditoria)',
  'block.reasonHint': 'O motivo vai apenas para o registro de auditoria. Nunca é mostrado à pessoa bloqueada nem aparece em mensagem de erro pública.',
  'block.blocking': 'Bloqueando…',
  'block.done': '{ip} bloqueado',
  'block.removed': '{ip} desbloqueado',
  'block.removedNone': '{ip} nem estava bloqueado',
  'block.failed': 'A operação de bloqueio falhou.',
  'block.unavailableService': 'Lista de bloqueio indisponível (sem Redis)',
  'block.colIp': 'IP',
  'block.colExpires': 'Expira',
  'block.colRemaining': 'Restante',
  'block.colActions': 'Ação',
  'block.unblock': 'Desbloquear',
  'block.unblockTitle': 'Desbloquear {ip}',
  'block.unblockMessage': 'Este endereço recupera imediatamente o acesso normal de escrita. Continuar?',
  'block.empty': 'A lista de bloqueio está vazia.',
  'block.emptyBody': 'Nenhum endereço está bloqueado. Esse é o estado normal: o bloqueio é sempre uma decisão da administração, e o sistema nunca bloqueia ninguém automaticamente.',
  'block.notAutoNote': 'Esta lista nunca se preenche sozinha. Um endereço que estoura o limite de taxa só recebe um 429; ele não é adicionado aqui automaticamente, porque a mesma saída pode pertencer a um escritório inteiro ou a uma NAT inteira, e o bloqueio automático pegaria eles também.',
  'block.scopeNoteLabel': 'Escopo',
  'block.notAutoNoteLabel': 'Nunca automático',
  'block.maxNoteLabel': 'Duração máxima',
  'block.scopeNote': 'O bloqueio só interrompe requisições de escrita. Ler postagens, comentários e arquivos estáticos continua possível, e uma pessoa bloqueada ainda consegue entrar e ver o conteúdo: isso é proposital, porque os endpoints de leitura não são limitados de propósito (senão um visitante anônimo não conseguiria usar o site), e o bloqueio cobre o mesmo conjunto.',
  'block.maxNote': 'Um bloqueio dura no máximo 365 dias. Um prazo maior é cortado para um ano: a expiração é guardada como número, e "para sempre" viraria um bloqueio que ninguém lembra e que nunca se levanta sozinho.',
  'block.count': '{count} endereços bloqueados',
  'block.ipInvalid': 'Endereço IP inválido. Informe um endereço IPv4 ou IPv6; faixas CIDR não são suportadas.',
  'announce.label': 'Aviso do site',
  'announce.publicNote': 'Aviso',
  'announce.closeAria': 'Fechar este aviso',
  'announce.publishedOn': 'Publicado em {date}',
  'announce.expiresOn': 'Expira em {date}',
  'announce.neverExpires': 'Sem expiração',
  'announce.pinnedBadge': 'Fixado',
  'announce.title': 'Avisos',
  'announce.eyebrow': 'ANNOUNCEMENTS',
  'announce.copy': 'Exibir um aviso acima de todas as páginas do site. Só um fica ativo por vez: publicar um novo desativa o anterior.',
  'announce.refresh': 'Atualizar',
  'announce.loadFailed': 'Falha ao carregar os avisos.',
  'announce.new': 'Publicar um novo aviso',
  'announce.edit': 'Editar',
  'announce.deactivate': 'Desativar',
  'announce.reactivate': 'Reativar',
  'announce.deleteNote': 'Avisos nunca são apagados, só desativados: manter o histórico responde quando foi publicado e por quem.',
  'announce.bodyLabel': 'Texto do aviso',
  'announce.bodyPlaceholder': 'Por exemplo: o sistema estará em manutenção na quinta, das 02:00 às 04:00.',
  'announce.bodyHint': 'Até 300 caracteres. Texto simples; as quebras de linha são mantidas.',
  'announce.activeLabel': 'Mostrar imediatamente',
  'announce.expiryLabel': 'Validade',
  'announce.expiryNever': 'Nunca expira automaticamente',
  'announce.expiryHours': 'em {hours} horas',
  'announce.expiryDays': 'em {days} dias',
  'announce.saving': 'Salvando…',
  'announce.published': 'Aviso publicado',
  'announce.updated': 'Aviso atualizado',
  'announce.deactivated': 'Aviso desativado',
  'announce.reactivated': 'Aviso reativado',
  'announce.saveFailed': 'A operação de aviso falhou.',
  'announce.empty': 'Ainda não há avisos.',
  'announce.emptyBody': 'Assim que você publicar um, ele aparece no topo da página de cada visitante.',
  'announce.colBody': 'Texto',
  'announce.colState': 'Estado',
  'announce.colAuthor': 'Publicado por',
  'announce.colCreated': 'Publicado em',
  'announce.colActions': 'Ação',
  'announce.stateActive': 'Exibido',
  'announce.stateInactive': 'Desativado',
  'announce.stateExpired': 'Expirado',
  'announce.confirmDeactivate': 'Publicar vai desativar o atual, e todos verão o texto novo imediatamente. Continuar?',
  'announce.confirmEdit': 'Alterar o texto ou a validade deste aviso?',
  'announce.count': '{count} no total',
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
  'posts.pin': 'Fixar',
  'posts.unpin': 'Desafixar',
  'posts.pinTitle': 'Fixar esta postagem',
  'posts.unpinTitle': 'Desafixar esta postagem',
  'posts.pinMessage': 'Depois de fixada, esta postagem fica no topo do feed de todos e nenhuma postagem nova a desloca.',
  'posts.unpinMessage': 'Ao Desafixar, a postagem volta ao lugar por ordem de tempo.',
  'posts.pinDone': 'Fixada',
  'posts.unpinDone': 'Desafixada',
  'posts.pinFailed': 'A operação de fixar falhou.',
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
  'title.adminMonitor': 'Monitoramento do sistema｜{brand} Administração',
  'title.adminLog': 'Registro de ações｜{brand} Administração',
  'title.adminStats': 'Tendências de conteúdo｜{brand} Administração',
  'title.adminExport': 'Exportação e ações em massa｜{brand} Administração',
  'title.adminSessions': 'Logins e sessões｜{brand} Administração',
  'title.adminBlocks': 'Lista de bloqueio de IP｜{brand} Administração',
  'title.adminAnnouncements': 'Avisos｜{brand} Administração',
};
