/*
 * 管理員操作稽核紀錄頁（/admin/log）
 *
 * 職責：查詢並呈現「誰在什麼時候對哪個對象做了什麼、哪些欄位從什麼變成什麼」。
 *
 * 這個頁面的設計目標與其他四個後臺頁不同。用戶管理與檢舉管理是「做事」的頁面
 * —— 使用者來這裡是為了改變狀態，操作結束就離開。稽核紀錄頁是「查帳」的頁面
 * —— 使用者來這裡通常是因為**出了事**：某篇文不見了、某個帳號被停權了、
 * 有人說不是我弄的。因此三個設計決定都圍繞「出事了要能快速找到原因」：
 *
 *   1. 篩選器優先於表格。預設顯示最近 50 筆（依時間倒序），而「最近發生了
 *      什麼」是查帳時最常見的第一個問題。表格第二欄就是操作者與動作，
 *      那是「嚴重程度」的第一個判斷依據。
 *   2. 欄位級 diff 直接顯示，不放進「展開細節」裡。稽核的用途是舉證，而
 *      需要點兩下才看到的資訊在截圖轉傳時會消失。
 *   3. 每個對象都可點擊成為篩選條件。「這篇文被誰動過」是查帳時第二常見的
 *      問題（第一個是「誰動的」），把它做成一次點擊而不是要求使用者手動
 *      複製 targetId 到篩選器。
 *
 * 刻意不做的事：
 *
 *   - 不提供刪除或修改紀錄的按鈕。一個能改自己紀錄的稽核頁沒有稽核價值；
 *     唯一的清理途徑是後端的時間式保留期（AUDIT_RETENTION_DAYS），而
 *     它的存在會寫在頁面上（見 retentionHint），因為「查得到多久」是
 *     判斷紀錄能不能當證據的前提。
 *   - 不顯示 requestId 以外的內部識別碼。requestId 保留是為了和存取日誌
 *     對照（「那筆操作對應到 access log 的哪一行」），而它在正常情況下
 *     不需要被看見。
 */

import { useCallback, useEffect, useMemo, useState } from 'react';

import { adminApi } from '../api/admin';
import { errorMessage, formatDateTime, formatNumber, text } from '../core';
import { t, usePageTitle } from '../i18n';
import type { AdminActionChange, AdminActionEntry, AdminActionLogResponse, AdminActionTarget } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import { BusyButton, EmptyState, FormStatus } from './ui';

/** 每頁筆數。必須與後端 audit.DefaultLimit 一致 —— 見該常數的說明。 */
const PAGE_SIZE = 50;

const LOG_COLUMNS = 6;

interface Filters {
  actor: string;
  action: string;
  targetType: string;
  targetId: string;
  from: string;
  to: string;
}

const EMPTY_FILTERS: Filters = { actor: '', action: '', targetType: '', targetId: '', from: '', to: '' };

/**
 * 目標類別的顯示順序。
 *
 * 依「後臺的操作頻率」排列而不是字母序：使用者最常關心的是使用者與文章，
 * 把它們排在前面可以讓篩選器的前兩格就是最常用的選項。
 */
const TARGET_ORDER: AdminActionTarget[] = ['user', 'post', 'comment', 'report', 'tag', 'system'];

/* ==========================================================================
   名稱對照
   ========================================================================== */

/**
 * 動作名稱 → 顯示字串。
 *
 * 刻意不從名稱本身推導（把 "post.delete" 變成「刪除／文章」）：那會讓中文
 * 讀起來像機器輸出，而動作名稱是**這個專案**的詞彙，需要和後臺其他頁面
 * 的按鈕用語一致（例如「刪文」而不是「刪除文章」）。
 *
 * 未知名稱原樣輸出：後端新增一種動作而前端還沒跟上時，那一筆紀錄必須仍然
 * 看得出是什麼，而不是變成空白。
 */
const ACTION_LABELS: Record<string, () => string> = {
  'user.suspend': () => t('log.actionUserSuspend'),
  'user.reinstate': () => t('log.actionUserReinstate'),
  'user.tags.update': () => t('log.actionUserTags'),
  'user.posts.create': () => t('log.actionUserPost'),
  'user.comments.create': () => t('log.actionUserComment'),
  'user.content.delete': () => t('log.actionUserContent'),
  'post.create': () => t('log.actionPostCreate'),
  'post.update': () => t('log.actionPostUpdate'),
  'post.delete': () => t('log.actionPostDelete'),
  'comment.create': () => t('log.actionCommentCreate'),
  'comment.update': () => t('log.actionCommentUpdate'),
  'comment.delete': () => t('log.actionCommentDelete'),
  'report.create': () => t('log.actionReportCreate'),
  'report.resolve': () => t('log.actionReportResolve'),
  'report.reject': () => t('log.actionReportReject'),
  'report.update': () => t('log.actionReportUpdate'),
  'report.delete': () => t('log.actionReportDelete'),
  'tag.create': () => t('log.actionTagCreate'),
  'tag.update': () => t('log.actionTagUpdate'),
  'tag.delete': () => t('log.actionTagDelete'),
};

function actionLabel(action: string): string {
  const label = ACTION_LABELS[action];
  return label ? label() : action;
}

/**
 * 動作名稱 → 嚴重度。
 *
 * 決定的是徽章的顏色，不是「這件事對不對」。分三級的理由是查帳時的第一個
 * 判斷是「這筆要不要立刻處理」：
 *   - danger 不可逆且直接影響使用者：刪文、刪留言、停權、刪標籤
 *   - warn   可逆但改變了使用者看到的內容：改文、改留言、裁定檢舉
 *   - ok     純新增或純字典操作
 *
 * 刻意不用「嚴重度」這個詞：它暗示一個客觀排名，而這裡其實是「我的站
 * 對這件事的在意程度」，那屬於站台政策。
 */
const ACTION_TONES: Record<string, 'danger' | 'warn' | 'ok'> = {
  'user.suspend': 'danger',
  'user.content.delete': 'danger',
  'post.delete': 'danger',
  'comment.delete': 'danger',
  'tag.delete': 'danger',
  'report.delete': 'danger',
  'post.update': 'warn',
  'comment.update': 'warn',
  'user.reinstate': 'ok',
  'user.tags.update': 'ok',
  'user.posts.create': 'warn',
  'user.comments.create': 'warn',
  'post.create': 'ok',
  'comment.create': 'ok',
  'report.create': 'ok',
  'report.resolve': 'warn',
  'report.reject': 'ok',
  'report.update': 'warn',
  'tag.create': 'ok',
  'tag.update': 'ok',
};

function actionTone(action: string): 'danger' | 'warn' | 'ok' {
  return ACTION_TONES[action] ?? 'ok';
}

const TARGET_LABELS: Record<string, () => string> = {
  user: () => t('log.targetUser'),
  post: () => t('log.targetPost'),
  comment: () => t('log.targetComment'),
  report: () => t('log.targetReport'),
  tag: () => t('log.targetTag'),
  system: () => t('log.targetSystem'),
};

function targetLabel(type: string): string {
  const label = TARGET_LABELS[type];
  return label ? label() : type;
}

/** 欄位名 → 顯示字串。未知欄位原樣輸出（理由同 actionLabel）。 */
const FIELD_LABELS: Record<string, () => string> = {
  status: () => t('log.fieldStatus'),
  content: () => t('log.fieldContent'),
  name: () => t('log.fieldName'),
  tags: () => t('log.fieldTags'),
  reason: () => t('log.fieldReason'),
  authorEmail: () => t('log.fieldAuthorEmail'),
  reporterEmail: () => t('log.fieldReporterEmail'),
  targetType: () => t('log.fieldTargetType'),
  targetId: () => t('log.fieldTargetId'),
  postId: () => t('log.fieldPostId'),
  commentId: () => t('log.fieldCommentId'),
  target: () => t('log.fieldTarget'),
  assignmentsRemoved: () => t('log.fieldAssignmentsRemoved'),
};

function fieldLabel(field: string): string {
  if (field === 'postId') return t('log.fieldPostIdShort');
  if (field === 'commentId') return t('log.fieldCommentIdShort');
  const label = FIELD_LABELS[field];
  return label ? label() : field;
}

/* ==========================================================================
   頁面
   ========================================================================== */

export function AdminLogPage() {
  usePageTitle('title.adminLog');

  const { toast } = useAdmin();
  const [items, setItems] = useState<AdminActionEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [actions, setActions] = useState<string[]>([]);
  const [actors, setActors] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // filters 是「已套用」的條件，draft 是「表單上正在編輯」的條件。
  //
  // 分成兩份是為了讓「改條件 → 按套用」變成一次明確的動作。若只有一份，
  // 每敲一個字就會打一次 API，而稽核表在「某個管理員反覆查帳」時正是
  // 查詢量最大的時候。
  const [draft, setDraft] = useState<Filters>(EMPTY_FILTERS);
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS);
  const [page, setPage] = useState(0);

  const load = useCallback(
    async (next: Filters, offset: number) => {
      setLoading(true);
      // offset 而不是頁碼：後端回應裡沒有總頁數（那是刻意不給的 ——
      // 稽核紀錄的總數會隨篩選改變，用「第 N 頁」做 URL 參數在分頁中途
      // 篩選條件改變時會指向不存在的頁）。改成自己算頁數並在請求裡傳 offset。
      const params = new URLSearchParams({ offset: String(offset), limit: String(PAGE_SIZE) });
      for (const [key, value] of Object.entries(next)) {
        if (value) params.set(key, value);
      }
      try {
        const response = await adminApi<AdminActionLogResponse>(`/api/admin/log?${params.toString()}`);
        setItems(response.items ?? []);
        setTotal(response.total ?? 0);
        setActions(response.actions ?? []);
        setActors(response.actors ?? []);
        setError(null);
      } catch (thrown) {
        const message = errorMessage(thrown, t('log.loadFailed'));
        setError(message);
        toast(message, 'error');
      } finally {
        setLoading(false);
      }
    },
    [toast],
  );

  // 初次載入，以及篩選／頁碼改變時重新查詢。
  useEffect(() => {
    void load(filters, page * PAGE_SIZE);
  }, [filters, page, load]);

  const applyFilters = useCallback(() => {
    setFilters(draft);
    setPage(0);
  }, [draft]);

  const resetFilters = useCallback(() => {
    setDraft(EMPTY_FILTERS);
    setFilters(EMPTY_FILTERS);
    setPage(0);
  }, []);

  // 點某一列的對象 → 只看與它有關的操作。這是「這篇文被誰動過」最快的路徑。
  const filterByTarget = useCallback((entry: AdminActionEntry) => {
    const next: Filters = { ...EMPTY_FILTERS, targetType: entry.targetType, targetId: entry.targetId };
    setDraft(next);
    setFilters(next);
    setPage(0);
  }, []);

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const from = total === 0 ? 0 : page * PAGE_SIZE + 1;
  const to = Math.min(total, (page + 1) * PAGE_SIZE);
  const targetTypes = useMemo(() => {
    // 只列出資料庫裡真的出現過的類別，但保持 TARGET_ORDER 的順序（未知類別排最後）。
    const seen = new Set(items.map((entry) => entry.targetType));
    return TARGET_ORDER.filter((type) => seen.has(type));
  }, [items]);

  return (
    <AdminShell active="/admin/log" pageTitle={t('log.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('log.eyebrow')}</p>
            <h1>{t('log.title')}</h1>
            <p className="page-head__copy">{t('log.copy')}</p>
          </div>
          <div className="page-head__actions">
            <BusyButton busy={loading} className="btn btn--sm" onClick={() => void load(filters, page * PAGE_SIZE)}>
              {t('log.refresh')}
            </BusyButton>
          </div>
        </header>

        {/* --- 篩選列 ---------------------------------------------------------
         * 用 field 群組而不是一個 .filters 容器：這五個條件的寬度差異很大
         * （操作者 email 比日期欄寬得多），用同一個 flex 格會讓 email 欄被
         * 壓到看不全。各自的 field 讓它們依內容決定寬度。
         * ------------------------------------------------------------------ */}
        <section className="panel" aria-labelledby="log-filter-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="log-filter-title">
                {t('log.filterApply')}
              </h2>
            </div>
          </div>
          <div className="panel__body">
            <div className="log-filters">
              <div className="field">
                <label className="field__label" htmlFor="log-actor">
                  {t('log.filterActor')}
                </label>
                <select
                  id="log-actor"
                  className="input"
                  value={draft.actor}
                  onChange={(event) => setDraft({ ...draft, actor: event.target.value })}
                >
                  <option value="">{t('log.filterAll')}</option>
                  {actors.map((actor) => (
                    <option key={actor} value={actor}>
                      {actor}
                    </option>
                  ))}
                </select>
              </div>

              <div className="field">
                <label className="field__label" htmlFor="log-action">
                  {t('log.filterAction')}
                </label>
                <select
                  id="log-action"
                  className="input"
                  value={draft.action}
                  onChange={(event) => setDraft({ ...draft, action: event.target.value })}
                >
                  <option value="">{t('log.filterAll')}</option>
                  {actions.map((action) => (
                    <option key={action} value={action}>
                      {actionLabel(action)}
                    </option>
                  ))}
                </select>
              </div>

              <div className="field">
                <label className="field__label" htmlFor="log-target-type">
                  {t('log.filterTargetType')}
                </label>
                <select
                  id="log-target-type"
                  className="input"
                  value={draft.targetType}
                  onChange={(event) => setDraft({ ...draft, targetType: event.target.value })}
                >
                  <option value="">{t('log.filterAll')}</option>
                  {targetTypes.map((type) => (
                    <option key={type} value={type}>
                      {targetLabel(type)}
                    </option>
                  ))}
                </select>
              </div>

              <div className="field">
                <label className="field__label" htmlFor="log-target-id">
                  {t('log.colTarget')}
                </label>
                <input
                  id="log-target-id"
                  className="input"
                  value={draft.targetId}
                  onChange={(event) => setDraft({ ...draft, targetId: event.target.value })}
                />
              </div>

              <div className="field">
                <label className="field__label" htmlFor="log-from">
                  {t('log.filterFrom')}
                </label>
                {/* type="date" 是原生的日期選擇器，值格式為 YYYY-MM-DD；
                    後端把它解讀成「當天 00:00:00」到「當天 23:59:59」，
                    因此選同一天作為起訖會得到完整一天而不是零筆。 */}
                <input
                  id="log-from"
                  className="input"
                  type="date"
                  value={draft.from}
                  onChange={(event) => setDraft({ ...draft, from: event.target.value })}
                />
              </div>

              <div className="field">
                <label className="field__label" htmlFor="log-to">
                  {t('log.filterTo')}
                </label>
                <input
                  id="log-to"
                  className="input"
                  type="date"
                  value={draft.to}
                  onChange={(event) => setDraft({ ...draft, to: event.target.value })}
                />
              </div>
            </div>

            <div className="log-filters__actions">
              <button className="btn btn--primary" onClick={applyFilters}>
                {t('log.filterApply')}
              </button>
              <button className="btn btn--ghost" onClick={resetFilters}>
                {t('log.filterReset')}
              </button>
              {/* 已套用的條件以文字列出：篩選列本身是「草稿」，若不另外顯示
                  「現在實際套用的是什麼」，使用者改了欄位但還沒按套用時會
                  不知道畫面上的資料對應到哪一組條件。 */}
              {hasActiveFilter(filters) ? (
                <span className="log-filters__active">{describeFilters(filters, actions)}</span>
              ) : null}
            </div>
          </div>
        </section>

        {error ? <FormStatus message={error} tone="error" /> : null}

        {/* --- 紀錄表 ------------------------------------------------------- */}
        <section className="panel" aria-labelledby="log-list-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="log-list-title">
                {t('log.count', { from, to, total: formatNumber(total) })}
              </h2>
            </div>
          </div>
          <div className="table-wrap">
            <table className="table log-table">
              <thead>
                <tr>
                  <th scope="col">{t('log.colTime')}</th>
                  <th scope="col">{t('log.colActor')}</th>
                  <th scope="col">{t('log.colAction')}</th>
                  <th scope="col">{t('log.colTarget')}</th>
                  <th scope="col">{t('log.colChanges')}</th>
                  <th scope="col">{t('log.colOrigin')}</th>
                </tr>
              </thead>
              <tbody aria-busy={loading}>
                {items.length === 0 && !loading ? (
                  <tr>
                    <td colSpan={LOG_COLUMNS}>
                      <EmptyState title={t('log.empty')}>{t('log.emptyBody')}</EmptyState>
                    </td>
                  </tr>
                ) : (
                  items.map((entry) => (
                    <tr key={entry.id}>
                      <td className="u-nowrap log-time">{formatActionTime(entry.createdAt)}</td>
                      <td>
                        <span className="cell-primary">{entry.actorEmail}</span>
                      </td>
                      <td>
                        <span className={`badge badge--${actionTone(entry.action)}`}>
                          {targetLabel(entry.targetType)}・{actionLabel(entry.action)}
                        </span>
                      </td>
                      <td>
                        {/* 整個目標欄位都是按鈕：它的文字長度差異很大（email 可能
                            佔滿一列，數字編號只有三個字元），把按鈕範圍綁在文字
                            本身才能讓「很短的那些」也點得到。 */}
                        <button
                          className="link log-target"
                          onClick={() => filterByTarget(entry)}
                          title={t('log.filterTargetHint')}
                        >
                          <span className="log-target__id">{entry.targetId}</span>
                          {/* targetLabel 與 targetId 相同時不重複顯示。
                              後端對 user 類的目標會把 email 同時放進兩個欄位
                              （id 供查詢、label 供辨識），而兩者對 user 來說
                              就是同一個字串 —— 重複一次等於把 email 顯示兩遍。 */}
                          {entry.targetLabel && entry.targetLabel !== entry.targetId ? (
                            <span className="cell-sub log-target__label">{entry.targetLabel}</span>
                          ) : null}
                        </button>
                      </td>
                      <td>
                        <ChangeList changes={entry.changes ?? []} />
                      </td>
                      <td>
                        <span className="cell-sub u-nowrap">{entry.clientIp || '—'}</span>
                        {entry.requestId ? (
                          <span className="cell-sub u-nowrap">{t('log.requestId', { id: entry.requestId.slice(0, 8) })}</span>
                        ) : null}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* --- 分頁 ---------------------------------------------------------
           * 只有「上一頁／下一頁」而沒有頁碼列表。理由是稽核紀錄的查詢結果
           * 通常是「往回翻找某個時間點」，跳頁反而會跳過中間的紀錄。
           * ------------------------------------------------------------------ */}
          {pageCount > 1 ? (
            <div className="panel__foot pagination">
              <button
                className="btn btn--ghost"
                disabled={page === 0 || loading}
                onClick={() => setPage((value) => Math.max(0, value - 1))}
              >
                ‹
              </button>
              <span className="pagination__info">
                {t('log.page', { page: page + 1 })} / {formatNumber(pageCount)}
              </span>
              <button
                className="btn btn--ghost"
                disabled={page + 1 >= pageCount || loading}
                onClick={() => setPage((value) => value + 1)}
              >
                ›
              </button>
            </div>
          ) : null}
        </section>

        <p className="log-retention">{t('log.retention', { days: 90 })}</p>
      </div>
    </AdminShell>
  );
}

/* ==========================================================================
   子元件
   ========================================================================== */

/**
 * 欄位級 diff 的呈現。
 *
 * 每個變更一行，「舊值 → 新值」。刻意不用顏色（綠/紅）表達增刪：值的內容是
 * 使用者的文字而不是數字，用顏色會讓「把 A 改成 B」看起來像「刪了 A」——
 * 而那正是這個頁面要避免的誤讀。
 */
function ChangeList({ changes }: { changes: AdminActionChange[] }) {
  if (changes.length === 0) {
    return <span className="muted">{t('log.noChanges')}</span>;
  }
  return (
    <ul className="log-changes">
      {changes.map((change, index) => (
        <li className="log-change" key={`${change.field}-${index}`}>
          <span className="log-change__field">{fieldLabel(change.field)}</span>
          <span className="log-change__values">
            <ChangeValue value={change.before} kind="before" />
            <span className="log-change__arrow" aria-label={t('log.changedTo')}>
              →
            </span>
            <ChangeValue value={change.after} kind="after" />
            {/* 截斷標記必須緊貼在值旁邊：它是「這不是全文」的唯一提示，
                而這一頁的用途是舉證 —— 讀者若以為看到的是全文，就會拿它
                當作「管理員改了整段話」的證明，而事實上後端只保留了前 200
                個位元組。把它放在欄位名旁邊會讓它看起來像在描述整個欄位。 */}
            {change.truncated ? <span className="log-change__cut">{t('log.truncated')}</span> : null}
          </span>
        </li>
      ))}
    </ul>
  );
}

/**
 * 單一變更值。
 *
 * 空字串在這裡有兩種可能的意義，兩者都被呈現成「（新增）」或「（已刪除）」：
 *   - 新增類操作（tag.create）沒有 before，傳的是空字串
 *   - 刪除類操作（post.delete）after 是「（文章已刪除）」這種後端寫入的字面
 * 後者的顯示完全依賴那個字面值，這是刻意的：後端選擇在 audit 裡寫下
 * 「（文章已刪除）」而不是留空，是為了讓稽核紀錄單獨存在（被匯出時）
 * 仍然讀得懂。
 */
function ChangeValue({ value, kind }: { value: string; kind: 'before' | 'after' }) {
  if (value === '') {
    return <span className="log-change__empty">{kind === 'before' ? t('log.created') : t('log.removed')}</span>;
  }
  return <span className={`log-change__value log-change__value--${kind}`}>{value}</span>;
}

/* ==========================================================================
   小工具
   ========================================================================== */

/**
 * 把後端的 "2026-09-30 15:04:05"（UTC，無時區標記）轉成本地時間顯示。
 *
 * 刻意自己解析而不用 new Date(string)：後端送的是「空格分隔、無 T、無時區」
 * 的格式，new Date 在部分引擎會把它當成當地時間而算錯（小時數最多差一個
 * 時區）。把 T 與 Z 補上才是明確的 UTC。
 */
function formatActionTime(raw: string): string {
  const normalized = raw.includes('T') ? raw : `${raw.replace(' ', 'T')}${raw.endsWith('Z') ? '' : 'Z'}`;
  const date = new Date(normalized);
  if (Number.isNaN(date.getTime())) return text(raw);
  return formatDateTime(date);
}

function hasActiveFilter(filters: Filters): boolean {
  return Object.values(filters).some((value) => value !== '');
}

/** 把已套用的條件轉成一句話，讓「畫面上的資料對應哪組條件」隨時可見。 */
function describeFilters(filters: Filters, actions: string[]): string {
  const parts: string[] = [];
  if (filters.actor) parts.push(filters.actor);
  if (filters.action) {
    // 只在名單裡找得到時才用譯名：使用者可以手動輸入一個不存在的動作名
    // （那是「查不到」的合理請求，例如想確認某個舊版本的名稱），那種情況
    // 應該原樣顯示他輸入的字串，而不是顯示一個對不上的譯名。
    parts.push(actions.includes(filters.action) ? actionLabel(filters.action) : filters.action);
  }
  if (filters.targetType) parts.push(targetLabel(filters.targetType));
  if (filters.targetId) parts.push(filters.targetId);
  if (filters.from) parts.push(`≥ ${filters.from}`);
  if (filters.to) parts.push(`≤ ${filters.to}`);
  return parts.join(' · ');
}
