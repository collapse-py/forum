/*
 * 論壇文章頁（/admin/forum）
 *
 * 職責：文章建立／編輯／刪除、留言層級治理、分頁、關鍵字搜尋，
 * 以及待處理檢舉的快速裁定。
 *
 * 與舊版的差異：
 *   - 舊版把「新增留言」「編輯留言」塞在 window.prompt 裡，一次連續兩個 prompt
 *     要人用「取消」跳出；現在各自是獨立、有標題的對話框。
 *   - 舊版 loadPosts / loadReports 各自用裸 fetch，錯誤只寫進表單狀態列，
 *     而該狀態列位在頁面上方 —— 捲到列表底部時完全看不到錯誤。改為 toast。
 *   - 舊版分頁是「1..N 全部列出」，文章一多就是數百顆按鈕；改成視窗式分頁。
 *   - 留言列加上 CSS 裡本來就存在、卻從未被掛上去的 .thread__item，
 *     讓 style.css 已有的卡片版面（框線、底色、間距）真正生效。
 *   - 搜尋取代分頁（不是並存）：結果是「依相關性」而不是「依時間」，
 *     兩種排序的頁碼放在一起會讓人搞不清自己在看第幾頁。搜尋結果沿用同一個
 *     ListState，因此表格、編輯與刪除都不必為搜尋再寫一份。
 */

import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';

import { adminApi, checkAdmin } from '../api/admin';
import { msg, t, tr, usePageTitle, type Message } from '../i18n';
import { errorMessage, errorText, formatDateTime, formatNumber, text } from '../core';
import { SearchGlyph } from '../icons';
import type { AdminPost, AdminReport, ItemsResponse, PagedResponse, SearchResponse } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import {
  BusyButton,
  EmptyState,
  FormStatus,
  ListBody,
  failedList,
  isEmpty,
  loadingList,
  readyList,
  type FormTone,
  type ListState,
} from './ui';

const POST_COLUMNS = 6;
const REPORT_COLUMNS = 5;
const COMMENT_MAX_LENGTH = 2000;
/** 搜尋時單頁筆數。與後端 pageSizeMax（25）一致，超過會被夾回。 */
const SEARCH_PAGE_SIZE = 25;
/** 後端 engine 值：'mysql' 代表 ES 未啟用或故障，結果是逐字比對。 */
const MYSQL_ENGINE = 'mysql';

export function ForumAdminPage() {
  usePageTitle('title.adminPosts');

  const { toast, dialog } = useAdmin();

  const [authorized, setAuthorized] = useState(false);

  /* --- 編輯器 ------------------------------------------------------------- */

  const [editingId, setEditingId] = useState<number | null>(null);
  const [content, setContent] = useState('');
  // 表單狀態列存的是 key+參數或後端回的字串，render 時才翻譯；存字串的話切換
  // 語言後這一行不會跟著換（見 runtime.ts 的「延後翻譯的訊息」）。
  const [formStatus, setFormStatus] = useState<{ message: Message | string | null; tone: FormTone }>({
    message: null,
    tone: '',
  });
  const [saving, setSaving] = useState(false);
  const editorRef = useRef<HTMLTextAreaElement>(null);
  const editorPanelRef = useRef<HTMLElement>(null);

  /* --- 文章列表 ----------------------------------------------------------- */

  const [posts, setPosts] = useState<ListState<AdminPost>>(loadingList<AdminPost>());
  const [postsSummary, setPostsSummary] = useState<Message | string>(() => msg('common.loading'));
  const [page, setPage] = useState(1);
  const [pages, setPages] = useState(1);
  const [refreshing, setRefreshing] = useState(false);

  /*
   * 搜尋。keyword 是輸入框（隨打隨變），activeQuery 是已送出的關鍵字
   * （決定畫面顯示搜尋結果還是分頁列表）。兩者分開是為了「每敲一個字就搜一次」
   * 這種昂貴行為不會發生 —— 送出只發生在按 Enter 或按搜尋鈕。
   */
  const [keyword, setKeyword] = useState('');
  const [activeQuery, setActiveQuery] = useState('');
  const [searchMeta, setSearchMeta] = useState({ total: 0, engine: '' });

  /* --- 待處理檢舉 --------------------------------------------------------- */

  const [reports, setReports] = useState<ListState<AdminReport>>(loadingList<AdminReport>());
  const [reportSummary, setReportSummary] = useState<Message | string>(() => msg('common.loading'));
  const [reportRefreshing, setReportRefreshing] = useState(false);
  const [decidingReportId, setDecidingReportId] = useState<number | null>(null);

  /* --- 資料載入 ----------------------------------------------------------- */

  const loadPosts = useCallback(async (target: number) => {
    setPosts(loadingList<AdminPost>());
    setPostsSummary(msg('common.loading'));
    try {
      const data = await adminApi<PagedResponse<AdminPost>>(`/api/admin/forum/posts?page=${target}`);
      const total = Math.max(1, Number(data?.pages) || 1);
      // 刪到最後一頁的最後一筆時，請求的頁碼可能已經不存在；退回最後一頁
      // 而不是顯示一個空列表（與舊版的 if (page > pages) page = pages 相同）。
      const effective = target > total ? total : target;
      setPages(total);
      if (effective !== target) setPage(effective);
      const items = Array.isArray(data?.items) ? data.items : [];
      setPosts(readyList(items));
      setPostsSummary(
        msg('posts.pageSummary', {
          page: formatNumber(effective),
          pages: formatNumber(total),
          count: formatNumber(items.length),
        }),
      );
    } catch (error) {
      const message = errorText(error, msg('posts.listLoadFailed'));
      setPosts(failedList<AdminPost>(tr(message)));
      setPostsSummary(msg('common.loadFailed'));
      toast(tr(message), 'error');
    }
  }, [toast]);

  const loadReports = useCallback(async () => {
    setReports(loadingList<AdminReport>());
    setReportSummary(msg('common.loading'));
    try {
      const data = await adminApi<ItemsResponse<AdminReport>>('/api/admin/forum/reports?status=PENDING');
      const items = Array.isArray(data?.items) ? data.items : [];
      setReports(readyList(items));
      setReportSummary(msg('posts.pendingCount', { count: formatNumber(items.length) }));
    } catch (error) {
      const message = errorText(error, msg('reports.listLoadFailed'));
      setReports(failedList<AdminReport>(tr(message)));
      setReportSummary(msg('common.loadFailed'));
      toast(tr(message), 'error');
    }
  }, [toast]);

  /*
   * 搜尋文章。查詢字串為空時不動作 —— 呼叫端（送出表單、清除鈕）各自決定
   * 要回到分頁列表還是維持原狀。
   *
   * 結果寫進與分頁列表相同的 posts 狀態，因此表格、編輯、刪除、留言治理
   * 全都不需要為搜尋另寫一份；「目前顯示的是哪一種」由 activeQuery 判斷，
   * 它的唯一用途是決定要不要畫分頁列。
   */
  const searchPosts = useCallback(
    async (query: string) => {
      const trimmed = query.trim();
      if (trimmed === '') return;
      setActiveQuery(trimmed);
      setPosts(loadingList<AdminPost>());
      setPostsSummary(msg('posts.searching'));
      try {
        const data = await adminApi<SearchResponse<AdminPost>>(
          `/api/admin/forum/search?q=${encodeURIComponent(trimmed)}&limit=${SEARCH_PAGE_SIZE}`,
        );
        const items = Array.isArray(data?.items) ? data.items : [];
        const total = Number(data?.total) || 0;
        const engine = String(data?.engine ?? '');
        setPosts(readyList(items));
        setSearchMeta({ total, engine });
        // 結果只顯示前 25 筆，總數可能更多：把「還有多少沒顯示」講清楚，
        // 否則管理員會以為搜尋漏掉了結果。
        setPostsSummary(
          msg('posts.searchSummary', { query: trimmed, total: formatNumber(total), shown: formatNumber(items.length) }) +
            (engine === MYSQL_ENGINE ? t('posts.searchDegraded') : ''),
        );
      } catch (error) {
        const message = errorMessage(error, t('posts.searchFailed'));
        setPosts(failedList<AdminPost>(tr(message)));
        setPostsSummary(msg('posts.searchStatusFailed'));
        toast(tr(message), 'error');
      }
    },
    [toast],
  );

  /** 回到分頁列表：清掉搜尋狀態並重抓目前這一頁。 */
  const clearPostSearch = useCallback(async () => {
    setActiveQuery('');
    setSearchMeta({ total: 0, engine: '' });
    await loadPosts(page);
  }, [loadPosts, page]);

  /*
   * 動作完成後的重新載入：搜尋中就重跑搜尋，否則重抓分頁。
   * 少了這個判斷，刪掉搜尋結果裡的一篇文章之後畫面會回到與查詢無關的分頁頁面。
   */
  const reloadPosts = useCallback(async () => {
    if (activeQuery !== '') await searchPosts(activeQuery);
    else await loadPosts(page);
  }, [activeQuery, loadPosts, page, searchPosts]);

  useEffect(() => {
    void (async () => {
      if (!(await checkAdmin())) {
        // adminApi() 內的 401 也會導回 /admin；這裡補的是「從未登入」的情況。
        window.location.replace('/admin');
        return;
      }
      setAuthorized(true);
      await Promise.all([loadPosts(1), loadReports()]);
    })();
  }, [loadPosts, loadReports]);

  /* --- 文章動作 ----------------------------------------------------------- */

  const resetEditor = useCallback(() => {
    setEditingId(null);
    setContent('');
    setFormStatus({ message: null, tone: '' });
  }, []);

  const startEditing = (item: AdminPost) => {
    setEditingId(item.id);
    setContent(item.content);
    setFormStatus({ message: null, tone: '' });
    // 等 React 提交完這次 render 再對焦與捲動，否則焦點會落在還沒換掉
    // 內容的輸入框上、捲動也會對著舊高度。
    requestAnimationFrame(() => {
      editorRef.current?.focus();
      editorPanelRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
  };

  const savePost = async () => {
    const trimmed = content.trim();
    if (!trimmed) {
      setFormStatus({ message: t('posts.emptyContent'), tone: 'error' });
      return;
    }
    // resetEditor() 會把 editingId 清空，所以要先記下這次是新增還是修改，
    // 否則後面的成功訊息永遠只會說「已發佈」。
    const wasEditing = editingId !== null;
    setSaving(true);
    setFormStatus({ message: t('posts.saving'), tone: '' });
    try {
      await adminApi<null, { content: string }>(
        wasEditing ? `/api/admin/forum/posts/${editingId}` : '/api/admin/forum/posts',
        { method: wasEditing ? 'PUT' : 'POST', body: { content: trimmed } },
      );
      resetEditor();
      await reloadPosts();
      toast(wasEditing ? t('posts.saved') : t('posts.published'), 'ok');
    } catch (error) {
      const message = errorMessage(error, t('posts.saveFailed'));
      setFormStatus({ message, tone: 'error' });
      toast(tr(message), 'error');
    } finally {
      setSaving(false);
    }
  };

  /*
   * 置頂／取消置頂。
   *
   * 兩件事刻意做成同一支函式：它們是同一個資源的同一個屬性的兩種值，而
   * 分成兩個函式會讓「按了按鈕之後要重載哪一份資料」有兩份可能 —— 而那
   * 正是會出現「列表顯示已置頂、但公開頁沒有」的地方。
   *
   * 送出的是「目標狀態」而不是「翻轉」：翻轉在連點兩下時會得到相反的結果
   * （使用者按一下想置頂、再按一下想取消，剛好是對的；但網路重送或使用者
   * 失去耐心再按一次就變成反的）。目標狀態在重送時冪等。
   */
  const togglePin = async (item: AdminPost) => {
    const next = item.pinned !== true;
    const ok = await dialog.confirm({
      title: next ? t('posts.pinTitle') : t('posts.unpinTitle'),
      message: next ? t('posts.pinMessage') : t('posts.unpinMessage'),
      confirmLabel: next ? t('posts.pin') : t('posts.unpin'),
    });
    if (!ok) return;
    try {
      await adminApi(`/api/admin/forum/posts/${item.id}/pin`, { method: 'POST', body: { pinned: next } });
      toast(next ? t('posts.pinDone') : t('posts.unpinDone'), 'ok');
      // 重載而不是就地改那一列：置頂會改變**排序**（公開動態的第一頁），
      // 而後臺的列表是依時間排序的 —— 因此就地改會讓列表的順序與公開頁
      // 不一致，而那正是使用者接下來要去比對的東西。
      await reloadPosts();
    } catch (error) {
      toast(errorMessage(error, t('posts.pinFailed')), 'error');
    }
  };

  const deletePost = async (item: AdminPost) => {
    const ok = await dialog.confirm({
      title: t('posts.deleteTitle', { id: item.id }),
      message: t('posts.deleteMessage'),
      confirmLabel: t('posts.deleteConfirm'),
    });
    if (!ok) return;
    try {
      await adminApi(`/api/admin/forum/posts/${item.id}`, { method: 'DELETE' });
      toast(t('posts.deleted'), 'ok');
      if (editingId === item.id) resetEditor();
      await reloadPosts();
    } catch (error) {
      toast(errorMessage(error, t('posts.deleteFailed')), 'error');
    }
  };

  const changeComment = async (method: string, path: string, body?: Record<string, unknown>) => {
    try {
      await adminApi<null, Record<string, unknown> | undefined>(path, body === undefined ? { method } : { method, body });
      toast(t('posts.commentUpdated'), 'ok');
      await reloadPosts();
    } catch (error) {
      toast(errorMessage(error, t('posts.commentActionFailed')), 'error');
    }
  };

  /*
   * 新增留言是兩步：先確認留言要掛在哪一篇文章（預填目前這篇，但允許改
   * 指到別篇 —— 這是後臺治理的必要彈性），再輸入內容。舊版是連續兩個
   * window.prompt，取消第一個要按一次「取消」再按「確定」；改成兩個獨立
   * 對話框後，每一步各自可取消。
   */
  const addComment = async (postId: number) => {
    const targetPostId = await dialog.promptText({
      title: t('posts.pickPostTitle'),
      message: t('posts.pickPostMessage'),
      label: t('posts.postIdLabel'),
      value: String(postId),
      confirmLabel: t('common.nextStep'),
    });
    if (targetPostId === null || !targetPostId.trim()) return;
    const value = await dialog.promptText({
      title: t('posts.addCommentAtTitle', { id: targetPostId.trim() }),
      message: t('posts.addCommentMessage'),
      label: t('posts.commentContentLabel'),
      placeholder: t('posts.commentContentPlaceholder'),
      maxLength: COMMENT_MAX_LENGTH,
      rows: 5,
      confirmLabel: t('posts.add'),
    });
    if (value === null) return;
    await changeComment('POST', '/api/admin/forum/comments', {
      postId: Number(targetPostId.trim()),
      content: value,
    });
  };

  // 留言的後臺端點是 /api/admin/forum/comments/{id}，本身不帶文章編號，
  // 因此這兩個動作不需要 postId（舊版之所以要帶，是因為它從 DOM 的
  // closest('.post-comments') 反查，作者欄位只是純展示）。
  const editComment = async (commentId: number, current: string) => {
    const value = await dialog.promptText({
      title: t('posts.editCommentTitle', { id: commentId }),
      label: t('posts.commentContentLabel'),
      value: current,
      maxLength: COMMENT_MAX_LENGTH,
      rows: 5,
    });
    if (value === null) return;
    await changeComment('PUT', `/api/admin/forum/comments/${commentId}`, { content: value });
  };

  const deleteComment = async (commentId: number) => {
    const ok = await dialog.confirm({
      title: t('posts.deleteCommentTitle', { id: commentId }),
      message: t('posts.deleteCommentMessage'),
      confirmLabel: t('common.delete'),
    });
    if (!ok) return;
    await changeComment('DELETE', `/api/admin/forum/comments/${commentId}`);
  };

  /* --- 檢舉裁定 ----------------------------------------------------------- */

  const decideReport = async (item: AdminReport, status: 'RESOLVED' | 'REJECTED') => {
    const label = status === 'RESOLVED' ? t('posts.verdictResolved') : t('posts.verdictRejected');
    const ok = await dialog.confirm({
      title: t('posts.verdictDialogTitle', { id: item.id, label }),
      message: t('posts.verdictDialogMessage'),
      confirmLabel: t('posts.verdictConfirm', { label }),
      tone: status === 'RESOLVED' ? 'primary' : 'danger',
    });
    if (!ok) return;
    setDecidingReportId(item.id);
    try {
      await adminApi<null, { status: string }>(`/api/admin/forum/reports/${item.id}`, {
        method: 'PATCH',
        body: { status },
      });
      toast(t('posts.verdictDone', { label }), 'ok');
      await loadReports();
    } catch (error) {
      toast(errorMessage(error, t('posts.verdictFailed')), 'error');
    } finally {
      setDecidingReportId(null);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    await reloadPosts();
    setRefreshing(false);
    toast(t('users.refreshDone'), 'ok');
  };

  const handleReportRefresh = async () => {
    setReportRefreshing(true);
    await loadReports();
    setReportRefreshing(false);
  };

  if (!authorized) {
    // 身分確認完成前不渲染任何介面：這段時間內的按鈕若可點，會打出一整批
    // 必然 401 的請求。
    return (
      <AdminShell active="/admin/forum" pageTitle={t('posts.title')}>
        {null}
      </AdminShell>
    );
  }

  return (
    <AdminShell active="/admin/forum" pageTitle={t('posts.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('posts.eyebrow')}</p>
            <h1>{t('posts.title')}</h1>
            <p className="page-head__copy">{t('posts.copy')}</p>
          </div>
          <div className="page-head__actions">
            <BusyButton busy={refreshing} className="btn" onClick={() => void handleRefresh()}>
              {t('common.refresh')}
            </BusyButton>
            <a className="btn" href="/admin/forum-report">
              {t('posts.toReports')}
            </a>
          </div>
        </header>

        <section className="panel" aria-labelledby="editor-title" ref={editorPanelRef}>
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="editor-title">
                {editingId === null ? t('posts.editorTitleNew') : t('posts.editorTitleEdit', { id: editingId })}
              </h2>
              <p className="panel__note">{t('posts.editorNote')}</p>
            </div>
            <div className="panel__actions">
              {editingId === null ? null : (
                <button className="btn btn--ghost" type="button" onClick={resetEditor}>
                  {t('posts.cancelEdit')}
                </button>
              )}
            </div>
          </div>
          <div className="panel__body">
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void savePost();
              }}
            >
              <label className="field" htmlFor="post-content">
                <span className="field__label">{t('posts.contentLabel')}</span>
                <textarea
                  className="textarea"
                  id="post-content"
                  ref={editorRef}
                  maxLength={10000}
                  placeholder={t('posts.contentPlaceholder')}
                  required
                  value={content}
                  onChange={(event) => setContent(event.target.value)}
                />
              </label>
              <div className="form-footer">
                <FormStatus message={formStatus.message === null ? '' : tr(formStatus.message)} tone={formStatus.tone} />
                {/*
                  這一顆是 type="submit"（不是 BusyButton 的 type="button"）：
                  它就在 <form> 裡，Enter 與螢幕閱讀器的表單送出都該走同一條路徑。
                */}
                <button
                  className={`btn btn--primary${saving ? ' is-busy' : ''}`}
                  type="submit"
                  disabled={saving}
                >
                  {editingId === null ? t('posts.editorTitleNew') : t('posts.saveChanges')}
                </button>
              </div>
            </form>
          </div>
        </section>

        <section className="panel" aria-labelledby="posts-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="posts-title">
                {t('posts.listTitle')}
              </h2>
              <p className="panel__note">{tr(postsSummary)}</p>
            </div>
            {/*
              搜尋取代分頁，因此這顆清除鈕與分頁列互斥出現。放在 panel__actions
              是為了與列表統計同一列，管理員不必捲動去找「怎麼回到全部」。
            */}
            <div className="panel__actions">
              {activeQuery === '' ? null : (
                <button className="btn btn--ghost" type="button" onClick={() => void clearPostSearch()}>
                  {t('posts.clearSearch')}
                </button>
              )}
            </div>
          </div>
          <div className="panel__body">
            {/*
              搜尋表單獨立於表格之外，而不是塞進 panel__head：搜尋框有自己的
              label 與輸入框（可及性需要），塞進只有標題與按鈕的 head 會讓版面
              在窄視窗下擠成一行。form-grid 只放這一欄，讓輸入框有合理的長度。
            */}
            <form
              className="search-form"
              role="search"
              onSubmit={(event: FormEvent<HTMLFormElement>) => {
                event.preventDefault();
                const trimmed = keyword.trim();
                if (trimmed === '') void clearPostSearch();
                else void searchPosts(trimmed);
              }}
            >
              <label className="field" htmlFor="post-search">
                <span className="field__label">{t('posts.searchLabel')}</span>
                {/*
                  放大鏡疊在輸入框左側，與公開頁 /forum 的搜尋列是同一個 SearchGlyph，
                  因此兩處的形狀與粗細不會各畫各的。位置由 .search-field 的 CSS 處理，
                  這裡不寫 inline style（CSP 的 style-src 'self' 不允許）。
                */}
                <span className="search-field">
                  <SearchGlyph className="search-field__icon" />
                  <input
                    className="input"
                    id="post-search"
                    type="search"
                    value={keyword}
                    maxLength={100}
                    placeholder={t('posts.searchPlaceholder')}
                    onChange={(event) => setKeyword(event.target.value)}
                  />
                </span>
                <span className="field__hint">
                  {t('posts.searchHint')}
                </span>
              </label>
              <div className="form-footer">
                {activeQuery === '' ? null : (
                  <FormStatus message={t('posts.searchTotal', { total: formatNumber(searchMeta.total) })} tone="" />
                )}
                <button className="btn btn--primary" type="submit">
                  {t('common.search')}
                </button>
              </div>
            </form>
          </div>
          <div className="table-wrap">
            <table className="table table--posts">
              <thead>
                <tr>
                  <th scope="col">{t('posts.colContentImage')}</th>
                  <th scope="col">{t('posts.colEngagement')}</th>
                  <th scope="col">{t('posts.colComments')}</th>
                  <th scope="col">{t('posts.colAuthor')}</th>
                  <th scope="col">{t('users.colCreated')}</th>
                  <th scope="col">{t('users.colActions')}</th>
                </tr>
              </thead>
              <tbody>
                <ListBody state={posts} columns={POST_COLUMNS} skeletonRows={6}>
                  {(items) =>
                    items.map((item) => (
                      <tr key={item.id}>
                        <td>
                          <p className="cell-clamp">{item.content}</p>
                          {item.imageUrl ? (
                            <a className="post-figure" href={item.imageUrl} target="_blank" rel="noreferrer noopener">
                              <img src={item.imageUrl} alt={t('posts.imageAlt')} loading="lazy" />
                            </a>
                          ) : null}
                        </td>
                        <td>
                          <span className="metric">{t('posts.likes', { count: formatNumber(item.likeCount) })}</span>
                        </td>
                        <td>
                          <CommentThread
                            post={item}
                            onAdd={() => void addComment(item.id)}
                            onEdit={(commentId, current) => void editComment(commentId, current)}
                            onDelete={(commentId) => void deleteComment(commentId)}
                          />
                        </td>
                        <td>
                          <span className="cell-sub">{text(item.authorEmail)}</span>
                        </td>
                        <td>
                          <span className="cell-sub u-nowrap">{formatDateTime(item.createdAt)}</span>
                        </td>
                        <td>
                          <div className="row-actions">
                            {/*
                              置頂鈕的位置在「修改」之前：它改變的是這篇文章
                              在公開頁的**位置**，而「修改」改變的是內容。從左
                              到右是「它會出現在哪裡 → 它說什麼 → 要不要刪掉」，
                              與管理者思考一件事的順序一致。
                            */}
                            <button
                              className={`btn btn--sm${item.pinned === true ? ' btn--primary' : ''}`}
                              type="button"
                              onClick={() => void togglePin(item)}
                            >
                              {item.pinned === true ? t('posts.unpin') : t('posts.pin')}
                            </button>
                            <button className="btn btn--sm" type="button" onClick={() => startEditing(item)}>
                              {t('common.edit')}
                            </button>
                            <button
                              className="btn btn--sm btn--danger"
                              type="button"
                              onClick={() => void deletePost(item)}
                            >
                              {t('common.delete')}
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))
                  }
                </ListBody>
              </tbody>
            </table>
          </div>
          {isEmpty(posts) ? (
            <EmptyState title={activeQuery === '' ? t('posts.emptyTitle') : t('posts.emptySearchTitle')}>
              {activeQuery === ''
                ? t('posts.emptyBody')
                : t('posts.emptySearchBody', { query: activeQuery })}
            </EmptyState>
          ) : null}
          {activeQuery === '' ? <Pagination page={page} pages={pages} onSelect={(next) => void loadPosts(next)} /> : null}
        </section>

        <section className="panel" aria-labelledby="reports-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="reports-title">
                {t('posts.reportsTitle')}
              </h2>
              <p className="panel__note">{tr(reportSummary)}</p>
            </div>
            <div className="panel__actions">
              <BusyButton busy={reportRefreshing} className="btn" onClick={() => void handleReportRefresh()}>
                {t('common.refresh')}
              </BusyButton>
              <a className="btn" href="/admin/forum-report">
                {t('posts.allReports')}
              </a>
            </div>
          </div>
          <div className="table-wrap">
            <table className="table table--reports">
              <thead>
                <tr>
                  <th scope="col">{t('posts.colReportedContent')}</th>
                  <th scope="col">{t('posts.colReason')}</th>
                  <th scope="col">{t('posts.colReporter')}</th>
                  <th scope="col">{t('posts.colTime')}</th>
                  <th scope="col">{t('posts.colVerdict')}</th>
                </tr>
              </thead>
              <tbody>
                <ListBody state={reports} columns={REPORT_COLUMNS} skeletonRows={3}>
                  {(items) =>
                    items.map((item) => (
                      <tr key={item.id}>
                        <td>
                          <span className="cell-primary">
                            {t(item.targetType === 'comment' ? 'kind.comment' : 'kind.post')} #{item.targetId}
                          </span>
                          <span className="cell-clamp">{text(item.targetContent)}</span>
                          <span className="cell-sub">{t('posts.author', { name: text(item.targetAuthor) })}</span>
                        </td>
                        <td>
                          <p className="cell-clamp">{item.reason}</p>
                        </td>
                        <td>
                          <span className="cell-sub">{item.reporterEmail}</span>
                        </td>
                        <td>
                          <span className="cell-sub u-nowrap">{formatDateTime(item.createdAt)}</span>
                        </td>
                        <td>
                          <div className="row-actions">
                            <BusyButton
                              busy={decidingReportId === item.id}
                              className="btn btn--sm"
                              onClick={() => void decideReport(item, 'RESOLVED')}
                            >
                              {t('posts.verdictResolved')}
                            </BusyButton>
                            <BusyButton
                              busy={decidingReportId === item.id}
                              className="btn btn--sm btn--ghost"
                              onClick={() => void decideReport(item, 'REJECTED')}
                            >
                              {t('posts.verdictRejected')}
                            </BusyButton>
                          </div>
                        </td>
                      </tr>
                    ))
                  }
                </ListBody>
              </tbody>
            </table>
          </div>
          {isEmpty(reports) ? (
            <EmptyState title={t('posts.emptyReportsTitle')}>{t('posts.emptyReportsBody')}</EmptyState>
          ) : null}
        </section>
      </div>
    </AdminShell>
  );
}

/* ==========================================================================
   留言摺疊
   ========================================================================== */

interface CommentThreadProps {
  post: AdminPost;
  onAdd: () => void;
  onEdit: (commentId: number, current: string) => void;
  onDelete: (commentId: number) => void;
}

function CommentThread({ post, onAdd, onEdit, onDelete }: CommentThreadProps) {
  const comments = Array.isArray(post.comments) ? post.comments : [];
  const addButton = (
    <button className="btn btn--sm" type="button" onClick={onAdd}>
      {t('users.addComment')}
    </button>
  );

  if (comments.length === 0) {
    return (
      <div className="thread__empty">
        <span>{t('posts.noComments')}</span>
        {addButton}
      </div>
    );
  }

  return (
    <details className="thread">
      <summary className="thread__toggle">
          {t('posts.commentCount', { count: formatNumber(post.commentCount || 0) })}
        </summary>
      <ul className="thread__list">
        {comments.map((comment) => (
          <li className="thread__item" key={comment.id}>
            <div>
              <strong>{text(comment.authorEmail)}</strong>
              <span>{comment.content}</span>
            </div>
            <div className="row-actions">
              <button
                className="btn btn--sm"
                type="button"
                onClick={() => onEdit(comment.id, comment.content)}
              >
                {t('common.edit')}
              </button>
              <button
                className="btn btn--sm btn--danger"
                type="button"
                onClick={() => onDelete(comment.id)}
              >
                {t('common.delete')}
              </button>
            </div>
          </li>
        ))}
      </ul>
      {addButton}
    </details>
  );
}

/* ==========================================================================
   分頁
   ========================================================================== */

interface PaginationProps {
  page: number;
  pages: number;
  onSelect: (page: number) => void;
}

function Pagination({ page, pages, onSelect }: PaginationProps) {
  if (pages <= 1) return null;
  return (
    <div className="pagination">
      {buildPageButtons(page, pages).map((button, index) => (
        <button
          // 分頁鈕沒有穩定識別（同一頁可能有多顆「1」與「…」），用位置當 key
          // 是唯一正確的選擇 —— 內容變動時重繪本來就是預期行為。
          key={index}
          type="button"
          disabled={button.disabled}
          aria-current={button.current ? 'page' : undefined}
          onClick={() => button.target > 0 && onSelect(button.target)}
        >
          {button.label}
        </button>
      ))}
    </div>
  );
}

interface PageButton {
  label: string;
  /** 0 代表省略號：只是視覺提示，不可點。 */
  target: number;
  current: boolean;
  disabled: boolean;
}

/**
 * 視窗式分頁：只在當前頁前後各留 3 頁。
 *
 * 舊版是 1..N 完整列舉，文章一多就是數百顆按鈕把整個面板撐爆。
 */
function buildPageButtons(page: number, pages: number): PageButton[] {
  const buttons: PageButton[] = [];
  const add = (label: string, target: number, current: boolean) => {
    buttons.push({ label, target, current, disabled: current || target === 0 });
  };

  const windowStart = Math.max(1, Math.min(page - 3, pages - 6));
  const windowEnd = Math.min(pages, windowStart + 6);

  if (page > 1) add(t('common.prevPage'), page - 1, false);
  if (windowStart > 1) add('1', 1, false);
  if (windowStart > 2) add('…', 0, false);
  for (let index = windowStart; index <= windowEnd; index += 1) {
    add(String(index), index, index === page);
  }
  if (windowEnd < pages - 1) add('…', 0, false);
  if (windowEnd < pages) add(String(pages), pages, false);
  if (page < pages) add(t('common.nextPage'), page + 1, false);

  return buttons;
}
