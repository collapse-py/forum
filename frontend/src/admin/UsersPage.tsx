/*
 * 用戶管理頁（/admin）
 *
 * 職責：登入閘門、用戶清單與統計、標籤管理，以及在用戶列上展開的內容治理面板。
 * fetch、toast、對話框與導覽列全部來自 admin 的 provider 與殼層。
 *
 * 相較舊版的三處行為變更，都是可用性而非功能：
 *   1. 標籤指派從「prompt 裡塞一段 [x] 清單、要求手打逗號分隔的 ID」改成
 *      真正的核取方塊清單（provider 的 pickTags）。
 *   2. 所有確認／輸入都改用 <dialog>，不再依賴 window.confirm / window.prompt。
 *   3. 使用者內容面板改為「就地展開一列」而不是舊版的同一個按鈕反覆觸發
 *      切換；資料只載入一次，切換時不再重新抓整份內容。
 *
 * 第四處是 React 版才補上的：內容治理動作（新增／刪除文章或留言）之後會重抓
 * 用戶清單。舊版只重畫展開面板，於是「文章數」欄要等使用者手動按更新資料
 * 才會跟上 —— 那個數字是這一頁的主要資訊，顯示過期比慢 200ms 嚴重。
 *
 * provider 由入口檔（src/entries/admin.tsx）包在最外層，與殼層的關係是
 * AdminProvider > AdminShell > 本頁，因此 useAdmin() 在整棵樹都可用。
 */

import { Fragment, useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';

import { adminApi, checkAdmin } from '../api/admin';
import { errorMessage, errorText, formatDateTime, formatNumber, text } from '../core';
import { msg, t, tr, usePageTitle, type Message } from '../i18n';
import { SITE_SHORT_NAME } from '../site';
import type { AdminTag, AdminUser, AdminUserContent, AdminUserPost, BatchResult, ItemsResponse } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import {
  BusyButton,
  EmptyState,
  ListBody,
  failedList,
  isEmpty,
  loadingList,
  readyList,
  type ListState,
} from './ui';

const USER_COLUMNS = 9;
const TAG_COLUMNS = 5;

type Gate = 'checking' | 'admin' | 'denied';

/** 展開列的狀態。key 是 email，因此不同用戶的載入互不影響。 */
type DetailState =
  | { phase: 'loading' }
  | { phase: 'ready'; data: AdminUserContent }
  | { phase: 'error'; message: Message | string };

type RecordKind = 'post' | 'comment';

const RECORD_PATH: Record<RecordKind, string> = { post: 'posts', comment: 'comments' };
/*
 * 內容種類的名稱。函式而非常數：常數會在 import 時把當下的語言固定住，而這個
 * 標籤會出現在確認框標題與檢舉列表裡，兩處都必須跟著語言走。
 */
function recordLabel(kind: RecordKind): string {
  return kind === 'post' ? t('kind.post') : t('kind.comment');
}
const RECORD_MAX_LENGTH: Record<RecordKind, number> = { post: 10000, comment: 2000 };

export function UsersPage() {
  const { toast, dialog, setUserCount } = useAdmin();

  const [gate, setGate] = useState<Gate>('checking');
  const [users, setUsers] = useState<ListState<AdminUser>>(loadingList<AdminUser>());
  const [tags, setTags] = useState<ListState<AdminTag>>(loadingList<AdminTag>());
  // 摘要與表單狀態列存的是 key+參數或後端回的字串，render 時才翻譯。存字串的話，
  // 使用者切換語言後這兩行會停在舊語言 —— 而摘要列通常就是切換後第一眼會去看的
  // 地方（見 runtime.ts 的「延後翻譯的訊息」）。
  const [usersSummary, setUsersSummary] = useState<Message | string>(() => msg('common.loading'));
  const [tagsSummary, setTagsSummary] = useState<Message | string>(() => msg('common.loading'));
  const [refreshing, setRefreshing] = useState(false);
  const [expandedEmail, setExpandedEmail] = useState<string | null>(null);
  const [details, setDetails] = useState<Record<string, DetailState>>({});
  /*
   * 批次選取。
   *
   * 存成 Set 而不是陣列：勾選、切換、重載都會用到「包含測試」，而陣列
   * 每次都要 O(n) 掃描。它是 state，因此每次更新都要產生新的 Set（React
   * 靠參考相等判斷有沒有變），這在 200 個選取時是完全可以接受的。
   *
   * 這裡存的是 email 而非索引：索引會因為重新載入而失效，而 email 是這張
   * 表的天然鍵（也是後端批次端點的識別值）。
   */
  const [selection, setSelection] = useState<ReadonlySet<string>>(() => new Set());
  const [batchRunning, setBatchRunning] = useState<'suspend' | 'reinstate' | 'tags' | null>(null);

  /*
   * 標題取決於閘門結果：通過後是這個頁面的標題，沒通過就維持 admin.html 原本
   * 的「登入」標題（那一頁未登入時本來就該顯示登入畫面）。
   */
  usePageTitle(gate === 'admin' ? 'title.adminUsers' : gate === 'denied' ? 'title.adminLogin' : null);

  /* --- 資料載入 ----------------------------------------------------------- */

  const loadUsers = useCallback(async () => {
    setUsers(loadingList<AdminUser>());
    setUsersSummary(msg('common.loading'));
    try {
      const data = await adminApi<ItemsResponse<AdminUser>>('/api/admin/users');
      const items = Array.isArray(data?.items) ? data.items : [];
      setUsers(readyList(items));
      setUsersSummary(msg('users.count', { count: formatNumber(items.length) }));
      setUserCount(items.length);
    } catch (error) {
      const message = errorText(error, msg('users.loadFailed'));
      setUsers(failedList<AdminUser>(tr(message)));
      setUsersSummary(msg('common.loadFailed'));
      setUserCount(null);
      toast(tr(message), 'error');
    }
  }, [setUserCount, toast]);

  const loadTags = useCallback(async () => {
    setTags(loadingList<AdminTag>());
    setTagsSummary(msg('common.loading'));
    try {
      const data = await adminApi<ItemsResponse<AdminTag>>('/api/admin/tags');
      const items = Array.isArray(data?.items) ? data.items : [];
      setTags(readyList(items));
      setTagsSummary(msg('users.tagsCount', { count: formatNumber(items.length) }));
    } catch (error) {
      const message = errorText(error, msg('users.tagsLoadFailed'));
      setTags(failedList<AdminTag>(tr(message)));
      setTagsSummary(msg('common.loadFailed'));
      toast(tr(message), 'error');
    }
  }, [toast]);

  // 閘門：未取得管理員身分時只渲染登入畫面，儀表板不發任何請求
  // （舊版靠 loadUsers() 開頭的 dashboardView.hidden 檢查達成同一件事）。
  useEffect(() => {
    void (async () => {
      if (await checkAdmin()) {
        // admin.html 的 <title> 是登入頁的標題（那是它未登入時該顯示的東西）；
        // 通過閘門後要換回這個頁面的標題，否則分頁籤會一直顯示「登入」。
        setGate('admin');
        return;
      }
      setGate('denied');
    })();
  }, []);

  useEffect(() => {
    if (gate !== 'admin') return;
    void (async () => {
      await loadUsers();
      await loadTags();
    })();
  }, [gate, loadUsers, loadTags]);

  /* --- 統計 --------------------------------------------------------------- */

  const stats = useMemo(() => {
    if (users.phase !== 'ready') return null;
    const items = users.items;
    const suspended = items.filter((user) => user.status === 'SUSPENDED').length;
    return {
      total: items.length,
      active: items.length - suspended,
      suspended,
      posts: items.reduce((sum, user) => sum + (Number(user.postCount) || 0), 0),
      comments: items.reduce((sum, user) => sum + (Number(user.commentCount) || 0), 0),
    };
  }, [users]);

  /* --- 用戶列動作 --------------------------------------------------------- */

  const changeUserStatus = async (user: AdminUser) => {
    const suspended = user.status === 'SUSPENDED';
    const next = suspended ? 'ACTIVE' : 'SUSPENDED';
    const action = suspended ? t('users.restore') : t('users.suspend');
    const ok = await dialog.confirm({
      title: t('users.statusDialogTitle', { action }),
      message:
        next === 'SUSPENDED'
          ? t('users.statusSuspendMessage', { email: user.email })
          : t('users.statusRestoreMessage', { email: user.email }),
      confirmLabel: action,
      tone: next === 'SUSPENDED' ? 'danger' : 'primary',
    });
    if (!ok) return;
    try {
      await adminApi<null, { status: string }>(`/api/admin/users/${encodeURIComponent(user.email)}`, {
        method: 'PATCH',
        body: { status: next },
      });
      // 這裡不把 action 塞回句型：中文的「已停權」同時是動詞與分詞，英文的
      // suspend 卻只有動詞身分，分詞是 suspended。共用一個 {action} 參數會讓
      // 其中一種語言讀成「用戶已Suspend.」，所以分成兩條各自完整的句子。
      toast(suspended ? t('users.userRestored') : t('users.userSuspended'), 'ok');
      await loadUsers();
    } catch (error) {
      toast(errorMessage(error, t('users.updateStatusFailed')), 'error');
    }
  };

  const assignUserTags = async (user: AdminUser) => {
    const available = tags.phase === 'ready' ? tags.items : [];
    const current = Array.isArray(user.tags) ? user.tags : [];
    const selected = await dialog.pickTags({
      title: t('users.editTagsTitle', { user: text(user.nickname) || user.email }),
      message: t('users.editTagsMessage'),
      tags: available,
      selectedIds: current.map((tag) => tag.id),
    });
    if (selected === null) return;
    try {
      await adminApi<null, { tagIds: number[] }>(`/api/admin/users/${encodeURIComponent(user.email)}/tags`, {
        method: 'PUT',
        body: { tagIds: selected },
      });
      toast(t('users.tagsUpdated'), 'ok');
      await loadUsers();
    } catch (error) {
      toast(errorMessage(error, t('users.updateTagsFailed')), 'error');
    }
  };

  /* --- 展開的內容面板 ----------------------------------------------------- */

  const toggleDetail = useCallback(
    async (email: string) => {
      if (expandedEmail === email) {
        setExpandedEmail(null);
        return;
      }
      setExpandedEmail(email);
      // 已經載入過就不再重抓：舊版把資料留在那一列裡，切回來是零請求。
      if (details[email]) return;
      setDetails((previous) => ({ ...previous, [email]: { phase: 'loading' } }));
      try {
        const data = await adminApi<AdminUserContent>(`/api/admin/users/${encodeURIComponent(email)}/content`);
        setDetails((previous) => ({ ...previous, [email]: { phase: 'ready', data } }));
      } catch (error) {
        const message = errorText(error, msg('users.contentLoadFailed'));
        setDetails((previous) => ({ ...previous, [email]: { phase: 'error', message } }));
        setExpandedEmail(null);
        toast(tr(message), 'error');
      }
    },
    [details, expandedEmail, toast],
  );

  const mutateDetail = useCallback(
    async (email: string, path: string, method: string, body: Record<string, unknown> | null) => {
      try {
        // 沒有 body 的請求（DELETE）不送 Content-Type，也不送 "null" 這個別處不在的欄位。
        await adminApi<null, Record<string, unknown> | null>(path, body === null ? { method } : { method, body });
        toast(t('users.updated'), 'ok');
        const data = await adminApi<AdminUserContent>(`/api/admin/users/${encodeURIComponent(email)}/content`);
        setDetails((previous) => ({ ...previous, [email]: { phase: 'ready', data } }));
        await loadUsers();
      } catch (error) {
        toast(errorMessage(error, t('users.updateContentFailed')), 'error');
      }
    },
    [loadUsers, toast],
  );

  const addPost = async (email: string) => {
    const content = await dialog.promptText({
      title: t('users.addPostTitle'),
      message: t('users.addPostMessage'),
      label: t('users.postContentLabel'),
      placeholder: t('users.postContentPlaceholder'),
      maxLength: RECORD_MAX_LENGTH.post,
      rows: 7,
      confirmLabel: t('common.publish'),
    });
    if (content === null) return;
    await mutateDetail(email, `/api/admin/users/${encodeURIComponent(email)}/posts`, 'POST', { content });
  };

  const addComment = async (email: string) => {
    const postId = await dialog.promptText({
      title: t('users.pickPostTitle'),
      label: t('users.postIdLabel'),
      placeholder: t('users.postIdPlaceholder'),
      confirmLabel: t('common.nextStep'),
    });
    if (postId === null) return;
    const content = await dialog.promptText({
      title: t('users.addCommentTitle'),
      label: t('users.commentContentLabel'),
      placeholder: t('users.commentContentPlaceholder'),
      maxLength: RECORD_MAX_LENGTH.comment,
      rows: 5,
      confirmLabel: t('common.publish'),
    });
    if (content === null) return;
    await mutateDetail(email, `/api/admin/users/${encodeURIComponent(email)}/comments`, 'POST', {
      postId: Number(postId),
      content,
    });
  };

  const editRecord = async (email: string, kind: RecordKind, id: number) => {
    const current = details[email];
    if (!current || current.phase !== 'ready') return;
    const records: AdminUserPost[] = kind === 'post' ? (current.data.posts ?? []) : (current.data.comments ?? []);
    const record = records.find((item) => item.id === id);
    const content = await dialog.promptText({
      title: t('users.editRecordTitle', { kind: recordLabel(kind), id }),
      label: t('users.contentLabel'),
      value: record?.content ?? '',
      maxLength: RECORD_MAX_LENGTH[kind],
      rows: 7,
    });
    if (content === null) return;
    await mutateDetail(email, `/api/admin/forum/${RECORD_PATH[kind]}/${id}`, 'PUT', { content });
  };

  const deleteRecord = async (email: string, kind: RecordKind, id: number) => {
    const ok = await dialog.confirm({
      title: t('users.deleteRecordTitle', { kind: recordLabel(kind), id }),
      message: t('users.deleteRecordMessage'),
      confirmLabel: t('common.delete'),
    });
    if (!ok) return;
    await mutateDetail(email, `/api/admin/forum/${RECORD_PATH[kind]}/${id}`, 'DELETE', null);
  };

  /* --- 標籤 --------------------------------------------------------------- */

  const createTag = async () => {
    const name = await dialog.promptText({
      title: t('users.createTagTitle'),
      message: t('users.createTagMessage'),
      label: t('users.tagNameLabel'),
      placeholder: t('users.tagNamePlaceholder'),
      maxLength: 50,
      confirmLabel: t('common.create'),
    });
    if (name === null) return;
    try {
      await adminApi<null, { name: string }>('/api/admin/tags', { method: 'POST', body: { name } });
      toast(t('users.tagCreated'), 'ok');
      await loadTags();
      await loadUsers();
    } catch (error) {
      // 名稱重複時後端回 409，這裡直接呈現它的訊息。
      toast(errorMessage(error, t('users.createTagFailed')), 'error');
    }
  };

  const renameTag = async (tag: AdminTag) => {
    const next = await dialog.promptText({
      title: t('users.renameTagTitle'),
      message: t('users.renameTagMessage'),
      label: t('users.tagNameLabel'),
      value: tag.name,
      maxLength: 50,
    });
    if (next === null || next === tag.name) return;
    try {
      await adminApi<null, { name: string }>(`/api/admin/tags/${tag.id}`, { method: 'PATCH', body: { name: next } });
      toast(t('users.tagUpdated'), 'ok');
      await loadTags();
      await loadUsers();
    } catch (error) {
      toast(errorMessage(error, t('users.updateTagFailed')), 'error');
    }
  };

  const removeTag = async (tag: AdminTag) => {
    const ok = await dialog.confirm({
      title: t('users.deleteTagTitle', { name: tag.name }),
      message: t('users.deleteTagMessage'),
      confirmLabel: t('users.deleteTagConfirm'),
    });
    if (!ok) return;
    try {
      await adminApi(`/api/admin/tags/${tag.id}`, { method: 'DELETE' });
      toast(t('users.tagDeleted'), 'ok');
      await loadTags();
      await loadUsers();
    } catch (error) {
      toast(errorMessage(error, t('users.deleteTagFailed')), 'error');
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    await loadUsers();
    await loadTags();
    setRefreshing(false);
    toast(t('users.refreshDone'), 'ok');
  };

  /* --- 批次操作 ------------------------------------------------------------- */

  /*
   * 選取的 email 清單**在 render 時**從 users.items 推導，而不是直接用
   * selection 的內容。
   *
   * 這個設計處理兩件不會報錯、但會讓操作結果無法解釋的事：
   *   1. 使用者按「重新整理」之後，某個帳號可能已不存在（被刪掉、或資料
   *      剛好換成沒有它的匯入結果）。若直接用 selection，那個 email 還是會
   *      被送到後端，然後出現在 skipped 清單裡 —— 讓使用者以為是自己選錯了。
   *   2. 選取必須依畫面上的順序（後端會依這個順序回 skipped，因此那個順序
   *      是「與勾選順序一致」的保證）。
   * 兩個需求一起指向「從當前列表推導」，而且它讓 selection 的生命週期變成
   * 純粹的勾選意圖，不需要在重新載入時手動清理。
   */
  const selectedEmails = useMemo(() => {
    if (users.phase !== 'ready' || selection.size === 0) return [];
    return users.items.filter((user) => selection.has(user.email)).map((user) => user.email);
  }, [selection, users]);

  const toggleSelected = useCallback((email: string) => {
    setSelection((previous) => {
      const next = new Set(previous);
      if (!next.delete(email)) next.add(email);
      return next;
    });
  }, []);

  // 全選 / 取消全選的按鈕放在批次列而不是表頭：表頭的核取方塊是「全選」最
  // 直覺的位置，但那一欄還要在沒有選取時顯示「0 / 0」之類的輔助文字，而
  // 批次列已經承載了選取相關的所有說明。
  const allSelected = selectedEmails.length > 0 && selectedEmails.length === (users.phase === 'ready' ? users.items.length : 0);

  const clearSelection = useCallback(() => setSelection(new Set()), []);

  /*
   * 批次操作的共用流程：確認 → 送出 → 報告結果 → 重載。
   *
   * 結果的呈現刻意分成兩段（toast 給一句話、skipped 給清單）：批次停權 200
   * 個帳號而其中 3 個不存在時，只說「已更新 197 個」會讓使用者以為那 3 個
   * 也被處理了。因此 skipped 一定被顯示，而且一定指出 email 與原因。
   */
  const runBatch = useCallback(
    async (kind: 'suspend' | 'reinstate' | 'tags', tagIds: number[] | null, confirmMessage: string) => {
      if (selectedEmails.length === 0) return;
      const ok = await dialog.confirm({
        title: kind === 'tags' ? t('export.batchTags') : t(kind === 'suspend' ? 'export.batchSuspend' : 'export.batchReinstate'),
        message: confirmMessage,
        confirmLabel: t('common.confirm'),
      });
      if (!ok) return;

      setBatchRunning(kind);
      try {
        const path = kind === 'tags' ? '/api/admin/batch/tags' : '/api/admin/batch/status';
        const body = kind === 'tags' ? { emails: selectedEmails, tagIds: tagIds ?? [] } : { emails: selectedEmails, status: kind === 'suspend' ? 'SUSPENDED' : 'ACTIVE' };
        const result = await adminApi<BatchResult, typeof body>(path, { method: 'POST', body });
        const updated = result?.counts?.updated ?? 0;
        const unchanged = result?.counts?.unchanged ?? 0;
        const skipped = result?.skipped ?? [];

        let summary = t('export.batchDone', { updated: formatNumber(updated) });
        if (unchanged > 0) {
          summary += `　${t('export.batchDoneUnchanged', { unchanged: formatNumber(unchanged) })}`;
        }
        toast(summary, skipped.length > 0 ? 'error' : 'ok');

        // skipped 逐項顯示 email 與後端給的原因。後端的 reason 是中文說明
        // （例如「帳號不存在」），而 email 與 reason 都要原樣顯示 —— 這是
        // 唯一能讓使用者知道「我該去處理什麼」的資訊。
        for (const item of skipped) {
          toast(`${item.email} — ${item.reason}`, 'error');
        }

        // 成功之後清掉選取：那一批已經處理完了，留著選取會讓使用者以為可以
        // 再按一次（而那會得到「未變更」）。
        clearSelection();
        await loadUsers();
      } catch (error) {
        toast(errorMessage(error, t('common.updateFailed')), 'error');
      } finally {
        setBatchRunning(null);
      }
    },
    [clearSelection, dialog, loadUsers, selectedEmails, toast],
  );

  const batchSuspend = useCallback(() => {
    void runBatch('suspend', null, t('export.batchConfirm', {
      count: formatNumber(selectedEmails.length),
      action: t('export.batchSuspend'),
    }));
  }, [runBatch, selectedEmails.length]);

  const batchReinstate = useCallback(() => {
    void runBatch('reinstate', null, t('export.batchConfirm', {
      count: formatNumber(selectedEmails.length),
      action: t('export.batchReinstate'),
    }));
  }, [runBatch, selectedEmails.length]);

  const batchTags = useCallback(async () => {
    const available = tags.phase === 'ready' ? tags.items : [];
    const picked = await dialog.pickTags({
      title: t('export.batchTags'),
      message: t('export.batchTagsNote'),
      tags: available,
      selectedIds: [],
    });
    if (picked === null) return;
    const names = available.filter((tag) => picked.includes(tag.id)).map((tag) => tag.name);
    // picked 為空陣列是「清除所有標籤」，不是「取消」。dialog 對取消回傳 null
    // （見 provider 的 pickTags），因此這裡的空陣列一定是有意的選擇。
    void runBatch('tags', picked, t('export.batchConfirmTags', {
      count: formatNumber(selectedEmails.length),
      tags: names.length > 0 ? names.join('、') : t('export.batchTagsNone'),
    }));
  }, [dialog, runBatch, selectedEmails.length, tags]);

  /* --- 畫面 --------------------------------------------------------------- */

  return (
    <AdminShell active="/admin" pageTitle={t('users.title')}>
      {gate === 'denied' ? <SignInView /> : null}

      {gate === 'admin' ? (
        <div className="stack">
          <header className="page-head">
            <div className="page-head__text">
              <p className="eyebrow">{t('users.eyebrow')}</p>
              <h1>{t('users.title')}</h1>
              <p className="page-head__copy">{t('users.copy')}</p>
            </div>
            <div className="page-head__actions">
              <BusyButton busy={refreshing} className="btn" onClick={() => void handleRefresh()}>
                {t('users.refresh')}
              </BusyButton>
            </div>
          </header>

          <div className="stats">
            <div className="stat">
              <span className="stat__label">{t('users.statTotal')}</span>
              <span className="stat__value">{stats ? formatNumber(stats.total) : '—'}</span>
            </div>
            <div className="stat stat--ok">
              <span className="stat__label">{t('users.statActive')}</span>
              <span className="stat__value">{stats ? formatNumber(stats.active) : '—'}</span>
            </div>
            <div className="stat stat--warn">
              <span className="stat__label">{t('users.statSuspended')}</span>
              <span className="stat__value">{stats ? formatNumber(stats.suspended) : '—'}</span>
            </div>
            <div className="stat">
              <span className="stat__label">{t('users.statContent')}</span>
              <span className="stat__value">
                {stats ? `${formatNumber(stats.posts)} / ${formatNumber(stats.comments)}` : '—'}
              </span>
            </div>
          </div>

          <section className="panel" aria-labelledby="users-title">
            <div className="panel__head">
              <div className="panel__titles">
                <h2 className="panel__title" id="users-title">
                  {t('users.panelTitle')}
                </h2>
                <p className="panel__note">{tr(usersSummary)}</p>
              </div>
              {/* 批次列。位置在 panel__head 而不是表格上方：它是「對已選取者
                  做什麼」的控制項，而選取狀態是從表格裡的核取方塊來的 ——
                  把它放在表格正上方會讓它看起來像篩選器。 */}
              {selectedEmails.length > 0 ? (
                <div className="panel__actions batch-bar">
                  <span className="batch-bar__count">{t('export.selected', { count: formatNumber(selectedEmails.length) })}</span>
                  <BusyButton busy={batchRunning === 'suspend'} className="btn btn--sm btn--danger" onClick={batchSuspend}>
                    {t('export.batchSuspend')}
                  </BusyButton>
                  <BusyButton busy={batchRunning === 'reinstate'} className="btn btn--sm" onClick={batchReinstate}>
                    {t('export.batchReinstate')}
                  </BusyButton>
                  <BusyButton busy={batchRunning === 'tags'} className="btn btn--sm" onClick={() => void batchTags()}>
                    {t('export.batchTags')}
                  </BusyButton>
                  <button className="btn btn--sm btn--ghost" onClick={clearSelection}>
                    {t('export.clearSelection')}
                  </button>
                </div>
              ) : (
                <div className="panel__actions">
                  <p className="batch-bar__hint">{t('export.noSelection')}</p>
                </div>
              )}
            </div>
            <div className="table-wrap">
              <table className="table table--users">
                <thead>
                  <tr>
                      <th scope="col" className="users-pick">
                        <button
                          className="btn btn--sm btn--ghost"
                          onClick={() =>
                            allSelected ? clearSelection() : setSelection(new Set(users.phase === 'ready' ? users.items.map((user) => user.email) : []))
                          }
                          disabled={users.phase !== 'ready' || users.items.length === 0}
                          title={
                            allSelected
                              ? t('export.clearSelection')
                              : t('export.selected', { count: formatNumber(users.phase === 'ready' ? users.items.length : 0) })
                          }
                        >
                          {allSelected ? t('export.clearSelection') : t('common.selectAll')}
                        </button>
                      </th>
                    <th scope="col">{t('users.colUser')}</th>
                    <th scope="col">{t('users.colTags')}</th>
                    <th scope="col">{t('users.colStatus')}</th>
                    <th scope="col" className="num">
                      {t('users.colPosts')}
                    </th>
                    <th scope="col" className="num">
                      {t('users.colComments')}
                    </th>
                    <th scope="col" className="num">
                      {t('users.colLikes')}
                    </th>
                    <th scope="col">{t('users.colLastActivity')}</th>
                    <th scope="col">{t('users.colActions')}</th>
                  </tr>
                </thead>
                <tbody>
                  <ListBody state={users} columns={USER_COLUMNS} skeletonRows={6}>
                    {(items) =>
                      items.map((user) => {
                        const suspended = user.status === 'SUSPENDED';
                        const expanded = expandedEmail === user.email;
                        const userTags = Array.isArray(user.tags) ? user.tags : [];
                        return (
                          <Fragment key={user.email}>
                            <tr>
                              <td className="users-pick">
                                <input
                                  type="checkbox"
                                  checked={selection.has(user.email)}
                                  onChange={() => toggleSelected(user.email)}
                                  aria-label={user.email}
                                />
                              </td>
                              <td>
                                <span className="cell-primary">{text(user.nickname) || t('users.nicknameUnset')}</span>
                                <span className="cell-sub">{user.email}</span>
                              </td>
                              <td>
                                {userTags.length > 0 ? (
                                  userTags.map((tag) => (
                                    <span className="chip" key={tag.id}>
                                      {tag.name}
                                    </span>
                                  ))
                                ) : (
                                  <span className="muted">{t('users.notSet')}</span>
                                )}
                              </td>
                              <td>
                                <span className={`badge ${suspended ? 'badge--danger' : 'badge--ok'}`}>
                                  {suspended ? t('users.statusSuspended') : t('users.statusActive')}
                                </span>
                              </td>
                              <td className="num">{formatNumber(user.postCount)}</td>
                              <td className="num">{formatNumber(user.commentCount)}</td>
                              <td className="num">{formatNumber(user.likeCount)}</td>
                              <td>
                                <span className="cell-sub u-nowrap">{formatDateTime(user.updatedAt)}</span>
                              </td>
                              <td>
                                <div className="row-actions">
                                  <button
                                    className="btn btn--sm"
                                    type="button"
                                    onClick={() => void assignUserTags(user)}
                                  >
                                    {t('users.colTags')}
                                  </button>
                                  <button
                                    className="btn btn--sm"
                                    type="button"
                                    aria-expanded={expanded}
                                    onClick={() => void toggleDetail(user.email)}
                                  >
                                    {t('users.contentAction')}
                                  </button>
                                  <button
                                    className={`btn btn--sm${suspended ? '' : ' btn--danger'}`}
                                    type="button"
                                    onClick={() => void changeUserStatus(user)}
                                  >
                                    {suspended ? t('users.restore') : t('users.suspend')}
                                  </button>
                                </div>
                              </td>
                            </tr>
                            {expanded ? (
                              <tr className="table--sub">
                                <td colSpan={USER_COLUMNS}>
                                  <UserContentPanel
                                    state={details[user.email]}
                                    onAddPost={() => void addPost(user.email)}
                                    onAddComment={() => void addComment(user.email)}
                                    onEdit={(kind, id) => void editRecord(user.email, kind, id)}
                                    onDelete={(kind, id) => void deleteRecord(user.email, kind, id)}
                                  />
                                </td>
                              </tr>
                            ) : null}
                          </Fragment>
                        );
                      })
                    }
                  </ListBody>
                </tbody>
              </table>
            </div>
            {isEmpty(users) ? (
              <EmptyState title={t('users.emptyTitle')}>{t('users.emptyBody')}</EmptyState>
            ) : null}
          </section>

          <section className="panel" aria-labelledby="tags-title">
            <div className="panel__head">
              <div className="panel__titles">
                <h2 className="panel__title" id="tags-title">
                  {t('users.tagsPanelTitle')}
                </h2>
                <p className="panel__note">{tr(tagsSummary)}</p>
              </div>
              <div className="panel__actions">
                <button className="btn" type="button" onClick={() => void createTag()}>
                  {t('users.addTag')}
                </button>
              </div>
            </div>
            <div className="table-wrap">
              <table className="table table--tags">
                <thead>
                  <tr>
                    <th scope="col">{t('users.colName')}</th>
                    <th scope="col">ID</th>
                    <th scope="col">{t('users.colCreated')}</th>
                    <th scope="col">{t('users.colUpdated')}</th>
                    <th scope="col">{t('users.colActions')}</th>
                  </tr>
                </thead>
                <tbody>
                  <ListBody state={tags} columns={TAG_COLUMNS} skeletonRows={3}>
                    {(items) =>
                      items.map((tag) => (
                        <tr key={tag.id}>
                          <td>
                            <span className="cell-primary">{tag.name}</span>
                          </td>
                          <td className="id">#{tag.id}</td>
                          <td>
                            <span className="cell-sub u-nowrap">{formatDateTime(tag.createdAt)}</span>
                          </td>
                          <td>
                            <span className="cell-sub u-nowrap">{formatDateTime(tag.updatedAt)}</span>
                          </td>
                          <td>
                            <div className="row-actions">
                              <button className="btn btn--sm" type="button" onClick={() => void renameTag(tag)}>
                                {t('users.renameTag')}
                              </button>
                              <button
                                className="btn btn--sm btn--danger"
                                type="button"
                                onClick={() => void removeTag(tag)}
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
            {isEmpty(tags) ? (
              <EmptyState title={t('users.tagsEmptyTitle')}>{t('users.tagsEmptyBody')}</EmptyState>
            ) : null}
          </section>
        </div>
      ) : null}
    </AdminShell>
  );
}

/* ==========================================================================
   展開列的內容治理面板
   ========================================================================== */

interface UserContentPanelProps {
  state: DetailState | undefined;
  onAddPost: () => void;
  onAddComment: () => void;
  onEdit: (kind: RecordKind, id: number) => void;
  onDelete: (kind: RecordKind, id: number) => void;
}

function UserContentPanel({ state, onAddPost, onAddComment, onEdit, onDelete }: UserContentPanelProps) {
  if (!state || state.phase === 'loading') {
    return (
      <div className="loading">
        <span className="spinner" aria-hidden="true" />
        <span>{t('users.contentLoading')}</span>
      </div>
    );
  }

  if (state.phase === 'error') {
    return (
      <div className="state">
        <strong>{t('users.contentLoadFailedShort')}</strong>
        <p>{tr(state.message)}</p>
      </div>
    );
  }

  const posts = Array.isArray(state.data.posts) ? state.data.posts : [];
  const comments = Array.isArray(state.data.comments) ? state.data.comments : [];

  return (
    <div className="detail">
      <div className="detail__head">
        <strong>{t('users.contentPanelTitle')}</strong>
        <span className="detail__count">
          {t('users.contentCount', {
            posts: formatNumber(posts.length),
            comments: formatNumber(comments.length),
          })}
        </span>
        <button className="btn btn--sm" type="button" onClick={onAddPost}>
          {t('users.addPost')}
        </button>
        <button className="btn btn--sm" type="button" onClick={onAddComment}>
          {t('users.addComment')}
        </button>
      </div>
      <div className="detail__cols">
        <section className="detail__col">
          <h4>{t('users.postsColumn')}</h4>
          {posts.length > 0 ? (
            posts.map((post) => (
              <ContentRecord
                key={post.id}
                kind="post"
                id={post.id}
                content={post.content}
                createdAt={post.createdAt}
                onEdit={onEdit}
                onDelete={onDelete}
              />
            ))
          ) : (
            <p className="muted">{t('users.noPosts')}</p>
          )}
        </section>
        <section className="detail__col">
          <h4>{t('users.commentsColumn')}</h4>
          {comments.length > 0 ? (
            comments.map((comment) => (
              <ContentRecord
                key={comment.id}
                kind="comment"
                id={comment.id}
                content={comment.content}
                createdAt={comment.createdAt}
                extra={<span>{t('users.postRef', { id: comment.postId })}</span>}
                onEdit={onEdit}
                onDelete={onDelete}
              />
            ))
          ) : (
            <p className="muted">{t('users.noComments')}</p>
          )}
        </section>
      </div>
    </div>
  );
}

interface ContentRecordProps {
  kind: RecordKind;
  id: number;
  content: string;
  createdAt: string | undefined;
  /** 顯示在 #id 旁邊的額外資訊（留言所屬的文章編號）。 */
  extra?: ReactNode;
  onEdit: (kind: RecordKind, id: number) => void;
  onDelete: (kind: RecordKind, id: number) => void;
}

function ContentRecord({ kind, id, content, createdAt, extra, onEdit, onDelete }: ContentRecordProps) {
  return (
    <article className="record">
      <div>
        <div className="record__meta">
          <span className="id">#{id}</span>
          {extra}
          <span>{formatDateTime(createdAt)}</span>
        </div>
        <p className="record__text">{content}</p>
      </div>
      <div className="record__actions">
        <button className="btn btn--sm" type="button" onClick={() => onEdit(kind, id)}>
          {t('common.edit')}
        </button>
        <button className="btn btn--sm btn--danger" type="button" onClick={() => onDelete(kind, id)}>
          {t('common.delete')}
        </button>
      </div>
    </article>
  );
}

/* ==========================================================================
   登入閘門
   ========================================================================== */

/**
 * 未取得管理員身分時取代整個儀表板。
 *
 * /admin/forum 與 /admin/forum-report 不共用這個元件：它們在 checkAdmin()
 * 失敗時直接 replace 到 /admin，讓「驗證身分」只有一個入口。
 */
function SignInView() {
  return (
    <section className="signin" aria-labelledby="signin-title">
      <div className="signin__aside">
        <p className="eyebrow">{t('users.signinEyebrow', { brand: SITE_SHORT_NAME })}</p>
        <h2>{t('users.signinTitle')}</h2>
        <p>{t('users.signinBody')}</p>
        <div className="signin__steps">
          <span>
            <strong>01</strong>
            {t('users.signinStep1')}
          </span>
          <span>
            <strong>02</strong>
            {t('users.signinStep2')}
          </span>
          <span>
            <strong>03</strong>
            {t('users.signinStep3')}
          </span>
        </div>
      </div>
      <div className="signin__panel">
        <div className="signin__inner">
          <h3 id="signin-title">{t('users.signinPanelTitle')}</h3>
          <p>{t('users.signinPanelBody')}</p>
          <a className="google-btn" href="/auth/google?return=%2Fadmin">
            <span className="google-btn__mark" aria-hidden="true">
              G
            </span>
            {t('auth.loginWithGoogleAdmin')}
          </a>
        </div>
      </div>
    </section>
  );
}
