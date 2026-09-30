/*
 * 檢舉管理頁（/admin/forum-report）
 *
 * 職責：裁定檢舉（通過並刪文／不成立）、更正紀錄與狀態篩選。
 *
 * 與舊版的差異：
 *   - 舊版永遠打 ?status=ALL，「待處理」要靠另一頁（/admin/forum）才看得到，
 *     這個頁面等於沒有篩選。後端本來就支援 status 參數，這裡直接接上分段控制。
 *   - 舊版有「新增檢舉」（POST /api/admin/forum/reports），讓管理員補登站外回報。
 *     這個入口已拿掉：檢舉由使用者從站內流程產生，管理員的職責是裁定而不是代客發舉。
 *     編輯器因此只在按「編輯」之後出現，預設畫面就是列表。
 *   - 舊版的裁定只改狀態（而且只在 /admin/forum 的待處理區塊），內容留在原地。
 *     這一頁的「通過（刪文）」會先刪掉被檢舉的文章／留言再標為已處理 ——
 *     「成立」實際上要做的事就是讓違規內容消失，拆成兩步做過的結果只是狀態變了。
 *   - 舊版編輯後不會把輸入框焦點或捲動做任何處理；現在會。
 *   - 舊版刪除失敗只把「刪除檢舉失敗」寫進表單狀態列（同樣在頁面上方，
 *     列表底部看不到）；改為 toast。
 */

import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';

import { ApiError, adminApi, checkAdmin } from '../api/admin';
import { msg, t, tr, usePageTitle, type Message } from '../i18n';
import { errorMessage, errorText, formatDateTime, formatNumber, text } from '../core';
import type { AdminReport, ItemsResponse } from '../types';
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

const REPORT_COLUMNS = 6;

type Filter = 'PENDING' | 'ALL' | 'RESOLVED' | 'REJECTED';
/** 可由列表直接套用的兩種裁定；PENDING 是退回待處理，不在這裡提供。 */
type Verdict = 'RESOLVED' | 'REJECTED';

const STATUS_BADGES: Record<string, string> = { PENDING: 'badge--warn', RESOLVED: 'badge--ok', REJECTED: 'badge--info' };

/**
 * 狀態與篩選器的顯示名稱。
 *
 * 都是函式而非常數陣列：常數會在 import 時把當下的語言固定住，切換語言之後
 * badge 與分段控制項就會停在第一種語言，而同一頁的其他地方卻已經換了。
 */
function statusLabel(status: string): string {
  if (status === 'PENDING') return t('reports.statusPending');
  if (status === 'RESOLVED') return t('reports.statusResolved');
  if (status === 'REJECTED') return t('reports.statusRejected');
  // 後端寫入時擋掉其他值，但字串來自資料庫；顯示原始值比顯示空白安全。
  return status;
}

function filters(): { value: Filter; label: string }[] {
  return [
    { value: 'PENDING', label: t('reports.statusPending') },
    { value: 'ALL', label: t('reports.filterAll') },
    { value: 'RESOLVED', label: t('reports.statusResolved') },
    { value: 'REJECTED', label: t('reports.statusRejected') },
  ];
}

/**
 * 被檢舉的內容已不存在時的顯示文字。
 *
 * targetContent 為空字串就代表目標已被刪除：文章與留言的內容在寫入時都經過
 * 非空白驗證（validateAdminForumPost / validateAdminForumComment），因此
 * 不可能是「原本就是空的內容」。這是「通過（刪文）」執行後的正常狀態。
 */
/*
 * 被檢舉的內容已不存在時的顯示文字。
 *
 * 原本是模組層常數，現在是函式：targetContent 為空字串就代表目標已被刪除
 * （文章與留言的內容在寫入時都經過非空白驗證，因此不可能是「原本就是空的
 * 內容」），而這個字串會出現在每一列的目標欄，必須跟著語言走。
 */
function targetGoneLabel(): string {
  return t('reports.targetGone');
}

/** targetType 只可能是 post 或 comment（後端寫入時擋掉其他值），未知的值一律當文章。 */
function targetLabel(item: AdminReport): string {
  return t(item.targetType === 'comment' ? 'kind.comment' : 'kind.post');
}

/** 目標是否已被刪除。見 TARGET_GONE 的說明。 */
function targetGone(item: AdminReport): boolean {
  return text(item.targetContent).trim() === '';
}

interface ReportForm {
  targetType: string;
  targetId: string;
  reporterEmail: string;
  status: string;
  reason: string;
}

const EMPTY_FORM: ReportForm = {
  targetType: 'post',
  targetId: '',
  reporterEmail: '',
  status: 'PENDING',
  reason: '',
};

export function ReportAdminPage() {
  usePageTitle('title.adminReports');

  const { toast, dialog } = useAdmin();

  const [authorized, setAuthorized] = useState(false);
  const [filter, setFilter] = useState<Filter>('ALL');
  const [reports, setReports] = useState<ListState<AdminReport>>(loadingList<AdminReport>());
  // 摘要與表單狀態列存 key+參數或後端回的字串，render 時才翻譯（見 runtime.ts 的
  // 「延後翻譯的訊息」）。
  const [reportSummary, setReportSummary] = useState<Message | string>(() => msg('common.loading'));
  const [refreshing, setRefreshing] = useState(false);

  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<ReportForm>(EMPTY_FORM);
  const [formStatus, setFormStatus] = useState<{ message: Message | string | null; tone: FormTone }>({
    message: null,
    tone: '',
  });
  const [saving, setSaving] = useState(false);
  const [decidingId, setDecidingId] = useState<number | null>(null);
  const editorPanelRef = useRef<HTMLElement>(null);
  const reasonRef = useRef<HTMLTextAreaElement>(null);

  /* --- 資料載入 ----------------------------------------------------------- */

  const loadReports = useCallback(
    async (status: Filter) => {
      setReports(loadingList<AdminReport>());
      setReportSummary(msg('common.loading'));
      try {
        const data = await adminApi<ItemsResponse<AdminReport>>(`/api/admin/forum/reports?status=${status}`);
        const items = Array.isArray(data?.items) ? data.items : [];
        setReports(readyList(items));
        setReportSummary(
          msg('reports.listSummary', {
            count: formatNumber(items.length),
            filter: filters().find((item) => item.value === status)?.label ?? status,
          }),
        );
      } catch (error) {
        const message = errorText(error, msg('reports.listLoadFailed'));
        setReports(failedList<AdminReport>(tr(message)));
        setReportSummary(msg('common.loadFailed'));
        toast(tr(message), 'error');
      }
    },
    [toast],
  );

  useEffect(() => {
    void (async () => {
      if (!(await checkAdmin())) {
        window.location.replace('/admin');
        return;
      }
      setAuthorized(true);
      await loadReports('ALL');
    })();
  }, [loadReports]);

  /* --- 編輯器 ------------------------------------------------------------- */

  const resetForm = useCallback(() => {
    setEditingId(null);
    setForm(EMPTY_FORM);
    setFormStatus({ message: null, tone: '' });
  }, []);

  const startEditing = (item: AdminReport) => {
    setEditingId(item.id);
    setForm({
      targetType: item.targetType,
      targetId: String(item.targetId),
      reporterEmail: item.reporterEmail,
      status: item.status,
      reason: item.reason,
    });
    setFormStatus({ message: null, tone: '' });
    requestAnimationFrame(() => {
      reasonRef.current?.focus();
      editorPanelRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
  };

  const saveReport = async () => {
    // 這一頁不再新增檢舉，PUT 的目標只能是正在編輯的那一筆。
    if (editingId === null) return;
    const payload = {
      targetType: form.targetType,
      targetId: Number(form.targetId),
      reporterEmail: form.reporterEmail.trim(),
      reason: form.reason.trim(),
      status: form.status,
    };
    setSaving(true);
    setFormStatus({ message: t('posts.saving'), tone: '' });
    try {
      await adminApi<null, typeof payload>(`/api/admin/forum/reports/${editingId}`, {
        method: 'PUT',
        body: payload,
      });
      // 面板在 editingId 歸零後就會消失，因此成功訊息只走 toast（表單狀態列留給失敗）。
      resetForm();
      toast(t('reports.updated'), 'ok');
      await loadReports(filter);
    } catch (error) {
      const message = errorMessage(error, t('reports.saveFailed'));
      setFormStatus({ message, tone: 'error' });
      toast(tr(message), 'error');
    } finally {
      setSaving(false);
    }
  };

  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void saveReport();
  };

  const deleteReport = async (item: AdminReport) => {
    const ok = await dialog.confirm({
      title: t('reports.deleteTitle', { id: item.id }),
      message: t('reports.deleteMessage'),
      confirmLabel: t('reports.deleteConfirm'),
    });
    if (!ok) return;
    try {
      await adminApi(`/api/admin/forum/reports/${item.id}`, { method: 'DELETE' });
      toast(t('reports.deleted'), 'ok');
      if (editingId === item.id) resetForm();
      await loadReports(filter);
    } catch (error) {
      toast(errorMessage(error, t('reports.deleteFailed')), 'error');
    }
  };

  const changeFilter = async (next: Filter) => {
    setFilter(next);
    // 篩選改變時收起編輯狀態：正在編輯的紀錄可能已經不在結果集裡。
    if (editingId !== null) resetForm();
    await loadReports(next);
  };

  /* --- 檢舉裁定 ----------------------------------------------------------- */

  /** 只改狀態。審核軌跡（reviewed_at / reviewed_by）由後端依 status 一併寫入。 */
  const patchStatus = async (item: AdminReport, status: Verdict) => {
    await adminApi<null, { status: Verdict }>(`/api/admin/forum/reports/${item.id}`, {
      method: 'PATCH',
      body: { status },
    });
    // 正在編輯的紀錄被裁定後，列表裡的內容已經過時，收起面板避免誤以為剛才的修改還在。
    if (editingId === item.id) resetForm();
  };

  /**
   * 通過：刪掉被檢舉的內容，再把檢舉標為已處理。
   *
   * 順序刻意是「先刪文、後改狀態」。反過來會出現最糟的組合：刪文失敗（權限、
   * 連線、對象已不存在以外的錯誤）時檢舉已經標成已處理，管理員會以為事情做完了，
   * 違規內容卻還在線上；此時按重來也只會因為狀態已是 RESOLVED 而看起來沒反應。
   * 現在的順序下，刪文成功但狀態更新失敗只會留下「待處理」的一筆，重按一次即可。
   */
  const approveReport = async (item: AdminReport) => {
    const gone = targetGone(item);
    const kind = targetLabel(item);
    const ok = await dialog.confirm({
      title: t('reports.approveTitle', { id: item.id }),
      message: gone
        ? t('reports.approveGoneMessage', { kind, id: item.targetId })
        : t('reports.approveMessage', { kind, id: item.targetId }),
      confirmLabel: t('reports.approveConfirm'),
      tone: 'danger',
    });
    if (!ok) return;
    setDecidingId(item.id);
    try {
      if (!gone) {
        try {
          await adminApi(`/api/admin/forum/${item.targetType === 'comment' ? 'comments' : 'posts'}/${item.targetId}`, {
            method: 'DELETE',
          });
        } catch (error) {
          // 內容在確認框停留期間被別人刪掉（後端回 404）不算失敗：目標確實已經
          // 不在了，繼續標為已處理即可。其他狀態碼照原樣往外丟。
          if (!(error instanceof ApiError) || error.status !== 404) throw error;
        }
      }
      await patchStatus(item, 'RESOLVED');
      toast(gone ? t('reports.approveGoneDone') : t('reports.approveDone'), 'ok');
      await loadReports(filter);
    } catch (error) {
      toast(errorMessage(error, t('reports.approveFailed')), 'error');
    } finally {
      setDecidingId(null);
    }
  };

  /** 不成立：原文保留，只把檢舉移出待處理清單。 */
  const rejectReport = async (item: AdminReport) => {
    const ok = await dialog.confirm({
      title: t('reports.rejectTitle', { id: item.id }),
      message: t('reports.rejectMessage'),
      confirmLabel: t('reports.rejectConfirm'),
      tone: 'primary',
    });
    if (!ok) return;
    setDecidingId(item.id);
    try {
      await patchStatus(item, 'REJECTED');
      toast(t('reports.rejectDone'), 'ok');
      await loadReports(filter);
    } catch (error) {
      toast(errorMessage(error, t('reports.statusFailed')), 'error');
    } finally {
      setDecidingId(null);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    await loadReports(filter);
    setRefreshing(false);
    toast(t('users.refreshDone'), 'ok');
  };

  if (!authorized) {
    return (
      <AdminShell active="/admin/forum-report" pageTitle={t('reports.title')}>
        {null}
      </AdminShell>
    );
  }

  return (
    <AdminShell active="/admin/forum-report" pageTitle={t('reports.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('reports.eyebrow')}</p>
            <h1>{t('reports.title')}</h1>
            <p className="page-head__copy">
              {t('reports.copy')}
            </p>
          </div>
          <div className="page-head__actions">
            <BusyButton busy={refreshing} className="btn" onClick={() => void handleRefresh()}>
              {t('common.refresh')}
            </BusyButton>
            <a className="btn" href="/admin/forum">
              {t('reports.backToPosts')}
            </a>
          </div>
        </header>

        {/*
          編輯器只在編輯中出現：這一頁沒有「新增檢舉」，預設畫面就是列表本身。
          目標與檢舉人是既有紀錄，以 readOnly 顯示 —— PUT 是全欄位取代，這三個值
          仍要原樣送回，但管理員在這裡改它們只會把一筆檢舉指向別的內容。
        */}
        {editingId === null ? null : (
          <section className="panel" aria-labelledby="report-editor-title" ref={editorPanelRef}>
            <div className="panel__head">
              <div className="panel__titles">
                <h2 className="panel__title" id="report-editor-title">
                  {t('reports.editorTitle', { id: editingId })}
                </h2>
                <p className="panel__note">{t('reports.editorNote')}</p>
              </div>
              <div className="panel__actions">
                <button className="btn btn--ghost" type="button" onClick={resetForm}>
                  {t('posts.cancelEdit')}
                </button>
              </div>
            </div>
            <div className="panel__body">
              <form onSubmit={onSubmit}>
                <div className="form-grid">
                  <label className="field" htmlFor="report-target-type">
                    <span className="field__label">{t('reports.targetTypeLabel')}</span>
                    <input
                      className="input"
                      id="report-target-type"
                      type="text"
                      readOnly
                      value={t(form.targetType === 'comment' ? 'kind.comment' : 'kind.post')}
                    />
                  </label>
                  <label className="field" htmlFor="report-target-id">
                    <span className="field__label">{t('reports.targetIdLabel')}</span>
                    <input className="input" id="report-target-id" type="number" readOnly value={form.targetId} />
                  </label>
                  <label className="field" htmlFor="reporter-email">
                    <span className="field__label">{t('reports.reporterEmailLabel')}</span>
                    <input className="input" id="reporter-email" type="email" readOnly value={form.reporterEmail} />
                  </label>
                  <label className="field" htmlFor="report-status">
                    <span className="field__label">{t('reports.statusLabel')}</span>
                    <select
                      className="select"
                      id="report-status"
                      value={form.status}
                      onChange={(event) => setForm({ ...form, status: event.target.value })}
                    >
                      <option value="PENDING">{t('reports.statusPending')}</option>
                      <option value="RESOLVED">{t('reports.statusResolved')}</option>
                      <option value="REJECTED">{t('reports.statusRejected')}</option>
                    </select>
                  </label>
                </div>
                <label className="field" htmlFor="report-reason">
                  <span className="field__label">{t('reports.reasonLabel')}</span>
                  <textarea
                    className="textarea"
                    id="report-reason"
                    ref={reasonRef}
                    maxLength={500}
                    required
                    value={form.reason}
                    onChange={(event) => setForm({ ...form, reason: event.target.value })}
                  />
                  <span className="field__hint">{t('reports.reasonHint')}</span>
                </label>
                <div className="form-footer">
                  <FormStatus message={formStatus.message === null ? '' : tr(formStatus.message)} tone={formStatus.tone} />
                  <button className={`btn btn--primary${saving ? ' is-busy' : ''}`} type="submit" disabled={saving}>
                    {t('posts.saveChanges')}
                  </button>
                </div>
              </form>
            </div>
          </section>
        )}

        <section className="panel" aria-labelledby="report-list-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="report-list-title">
                {t('reports.listTitle')}
              </h2>
              <p className="panel__note">{tr(reportSummary)}</p>
            </div>
            <div className="panel__actions">
              <div className="segmented" role="group" aria-label={t('reports.filterLabel')}>
                {filters().map((item) => (
                  <button
                    key={item.value}
                    type="button"
                    data-status-filter={item.value}
                    aria-pressed={filter === item.value}
                    onClick={() => void changeFilter(item.value)}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
          <div className="table-wrap">
            <table className="table table--reports">
              <thead>
                <tr>
                  <th scope="col">{t('reports.colTarget')}</th>
                  <th scope="col">{t('reports.colReason')}</th>
                  <th scope="col">{t('reports.colReporter')}</th>
                  <th scope="col">{t('reports.colStatus')}</th>
                  <th scope="col">{t('users.colCreated')}</th>
                  <th scope="col">{t('users.colActions')}</th>
                </tr>
              </thead>
              <tbody>
                <ListBody state={reports} columns={REPORT_COLUMNS} skeletonRows={5}>
                  {(items) =>
                    items.map((item) => (
                      <tr key={item.id}>
                        <td>
                          <span className="cell-primary">
                            {targetLabel(item)} #{item.targetId}
                          </span>
                          <span className="cell-clamp">{targetGone(item) ? targetGoneLabel() : item.targetContent}</span>
                          <span className="cell-sub">{t('reports.author', { name: text(item.targetAuthor) || t('common.placeholder') })}</span>
                        </td>
                        <td>
                          <p className="cell-clamp">{item.reason}</p>
                        </td>
                        <td>
                          <span className="cell-sub">{item.reporterEmail}</span>
                        </td>
                        <td>
                          <span className={`badge ${STATUS_BADGES[item.status] ?? 'badge--info'}`}>
                            {statusLabel(item.status)}
                          </span>
                        </td>
                        <td>
                          <span className="cell-sub u-nowrap">{formatDateTime(item.createdAt)}</span>
                        </td>
                        <td>
                          <div className="row-actions">
                            <BusyButton
                              busy={decidingId === item.id}
                              className="btn btn--sm btn--danger"
                              onClick={() => void approveReport(item)}
                              title={targetGone(item) ? t('reports.approveTitleGone') : t('reports.approveTitleFull')}
                            >
                              {t('reports.approveConfirm')}
                            </BusyButton>
                            <BusyButton
                              busy={decidingId === item.id}
                              className="btn btn--sm"
                              onClick={() => void rejectReport(item)}
                              title={t('reports.rejectTitleAttr')}
                            >
                              {t('reports.statusRejected')}
                            </BusyButton>
                            <button className="btn btn--sm" type="button" onClick={() => startEditing(item)}>
                              {t('common.edit')}
                            </button>
                            <button
                              className="btn btn--sm btn--danger"
                              type="button"
                              onClick={() => void deleteReport(item)}
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
          {isEmpty(reports) ? (
            <EmptyState title={t('reports.emptyTitle')}>{t('reports.emptyBody')}</EmptyState>
          ) : null}
        </section>
      </div>
    </AdminShell>
  );
}
