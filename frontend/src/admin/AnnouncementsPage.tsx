/*
 * 站內公告管理頁（/admin/announcements）
 *
 * 職責：發佈、修改、停用站內公告。
 *
 * 這兩個操作（公告與文章置頂）刻意分成兩頁而不是同一頁：它們的「單元」不同 ——
 * 公告是全站唯一一個資源（一則橫幅），置頂是每一篇文章上的一個旗標。把它們
 * 放在一起會讓「這頁在管理什麼」變成一個需要解釋的問題。置頂按鈕因此在
 * 「論壇文章」管理頁的每一列上（見 PostsAdminPage）。
 *
 * 「同時只有一則生效」是這個頁面最重要的規則，因此它出現在三個地方：
 * 頁面說明、發佈後的 toast、以及列表的狀態欄 —— 三者都說同一件事，因為它
 * 不是一個使用者能自己發現的細節（他們只會發現「我的舊公告不見了」）。
 */

import { useCallback, useEffect, useState } from 'react';

import { adminApi } from '../api/admin';
import { errorMessage, formatDateTime, formatNumber } from '../core';
import { t, usePageTitle } from '../i18n';
import type { AdminAnnouncementView, AdminAnnouncementsResponse } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import { BusyButton, EmptyState, FormStatus } from './ui';

const ANNOUNCE_COLUMNS = 5;

/** 內容上限。與後端的 maxAnnouncementLength 一致 —— 兩處的數字要一樣。 */
const MAX_BODY = 300;

/**
 * 有效時間的可選值（小時）。刻意是有限清單而不是「輸入小時數」——
 * 理由與封鎖時長相同：數字欄位最常見的填法是把「7」當成「7 分鐘」，
 * 而那會讓一則該留一週的公告在一小時後消失。
 *
 * 0 代表永不自動過期，放在最後：它是較少見的選項，而預設值刻意不是它
 * （預設 7 天，因為「一則沒有結束時間的公告」是該被想過才做的決定）。
 */
const EXPIRY_OPTIONS = [
  { hours: 24, key: 'announce.expiryHours', params: { hours: 24 } },
  { hours: 24 * 3, key: 'announce.expiryHours', params: { hours: 72 } },
  { hours: 24 * 7, key: 'announce.expiryDays', params: { days: 7 } },
  { hours: 24 * 30, key: 'announce.expiryDays', params: { days: 30 } },
  { hours: 0, key: 'announce.expiryNever', params: {} },
] as const;

/**
 * 預設有效時間。抽成常數是因為它出現在三個彼此無關的地方（新表單預設、
 * 編輯時取不到剩餘時間的退場值、從列表重新啟用已過期公告），三處必須一致，
 * 否則「按重新啟用」會給出一個和管理員在表單裡看到的預設不同的時長。
 */
const DEFAULT_EXPIRY_HOURS = 24 * 7;

export function AnnouncementsPage() {
  usePageTitle('title.adminAnnouncements');

  const { toast, dialog } = useAdmin();
  const [items, setItems] = useState<AdminAnnouncementView[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // 編輯中的那一則。null 表示「正在建立新的」—— 這與「正在看列表」是同一個
  // 畫面狀態，因此刻意不另外開一個「有沒有正在編輯」的布林：那個布林會
  // 與 items 的載入狀態糾纏在一起，而用 id 與否兩種值就足以表達。
  const [editing, setEditing] = useState<AdminAnnouncementView | null>(null);
  const [body, setBody] = useState('');
  const [active, setActive] = useState(true);
  const [expiryHours, setExpiryHours] = useState<number>(24 * 7);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await adminApi<AdminAnnouncementsResponse>('/api/admin/announcements');
      setItems(response.items ?? []);
      setError(null);
    } catch (thrown) {
      const message = errorMessage(thrown, t('announce.loadFailed'));
      setError(message);
      toast(message, 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const resetForm = useCallback(() => {
    setEditing(null);
    setBody('');
    setActive(true);
    setExpiryHours(DEFAULT_EXPIRY_HOURS);
    setFormError(null);
  }, []);

  const startEdit = useCallback((item: AdminAnnouncementView) => {
    setEditing(item);
    setBody(item.body);
    setActive(item.active);
    // 既有公告的到期時間是絕對時間戳，而表單要的是「幾小時後」。
    // 換算是用剩餘時間而不是原始小時數：一則三年前設定的 7 天期公告，
    // 原始值會是 168（小時），而那在編輯時已經毫無意義。取不到就回到預設的
    // 7 天，讓管理員重新設定一個合理的長度。
    setExpiryHours(hoursUntil(item.expiresAt) ?? DEFAULT_EXPIRY_HOURS);
    setFormError(null);
  }, []);

  const save = useCallback(async () => {
    setBusy(true);
    setFormError(null);
    try {
      if (editing === null) {
        const result = await adminApi<{ id: number }, { body: string; active: boolean; hoursUntilExpiry: number }>(
          '/api/admin/announcements',
          { method: 'POST', body: { body, active, hoursUntilExpiry: expiryHours } },
        );
        toast(t('announce.published'), 'ok');
        void result;
      } else {
        await adminApi<{ ok: boolean }, { body: string; active: boolean; hoursUntilExpiry: number }>(
          `/api/admin/announcements/${editing.id}`,
          { method: 'PATCH', body: { body, active, hoursUntilExpiry: expiryHours } },
        );
        toast(t('announce.updated'), 'ok');
      }
      resetForm();
      await load();
    } catch (thrown) {
      const message = errorMessage(thrown, t('announce.saveFailed'));
      setFormError(message);
      toast(message, 'error');
    } finally {
      setBusy(false);
    }
  }, [active, body, editing, expiryHours, load, resetForm, toast]);

  const setAnnouncementActive = useCallback(
    async (item: AdminAnnouncementView, next: boolean) => {
      if (next) {
        const ok = await dialog.confirm({
          title: t('announce.reactivate'),
          message: t('announce.confirmEdit'),
        });
        if (!ok) return;
      } else {
        const ok = await dialog.confirm({
          title: t('announce.deactivate'),
          // 停用有立即可見的效果（所有訪客），所以確認訊息必須說出後果，
          // 而不只是問「確定嗎」。
          message: t('announce.confirmDeactivate'),
        });
        if (!ok) return;
      }
      try {
        // hoursUntilExpiry **必須**帶原本的到期語意，不能固定送 0。
        //
        // 後端把 hoursUntilExpiry=0 解讀成「expires_at 設為 NULL（永不過期）」，
        // 因此固定送 0 會讓「重新啟用一則設定了 7 天有效期的公告」把它變成
        // 永久顯示 —— 而那個語意在畫面上完全看不到，也沒有任何錯誤訊息。
        //
        // 三種情況因此要分開處理：
        //   - 還沒到期 → 帶**剩餘**小時數，讓它維持原本被設定的那個時點
        //     （後端是相對時間，每差一小時送出都會讓到期點位移一小時，
        //     因此停用時送出、日後啟用時不送出，位移就只累積一次）
        //   - 已經過期 → 重新啟用等於「重新給它一個時長」，用表單預設值；
        //     這也正是管理員在表單裡自己按「重新啟用」時會得到的結果
        //   - 本來就沒有到期時間 → 0 正是原本的語意
        const hours = item.expiresAt ? hoursUntil(item.expiresAt) ?? DEFAULT_EXPIRY_HOURS : 0;
        await adminApi<{ ok: boolean }, { body: string; active: boolean; hoursUntilExpiry: number }>(
          `/api/admin/announcements/${item.id}`,
          { method: 'PATCH', body: { body: item.body, active: next, hoursUntilExpiry: hours } },
        );
        toast(next ? t('announce.reactivated') : t('announce.deactivated'), 'ok');
        await load();
      } catch (thrown) {
        toast(errorMessage(thrown, t('announce.saveFailed')), 'error');
      }
    },
    [dialog, load, toast],
  );

  const bodyLength = [...body].length;
  const overLimit = bodyLength > MAX_BODY;

  return (
    <AdminShell active="/admin/announcements" pageTitle={t('announce.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('announce.eyebrow')}</p>
            <h1>{t('announce.title')}</h1>
            <p className="page-head__copy">{t('announce.copy')}</p>
          </div>
          <div className="page-head__actions">
            <BusyButton busy={loading} className="btn btn--sm" onClick={() => void load()}>
              {t('announce.refresh')}
            </BusyButton>
          </div>
        </header>

        {error ? <FormStatus message={error} tone="error" /> : null}

        {/* --- 發佈／修改表單 ------------------------------------------------ */}
        <section className="panel" aria-labelledby="announce-form-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="announce-form-title">
                {editing === null ? t('announce.new') : t('announce.edit')}
              </h2>
            </div>
          </div>
          <div className="panel__body">
            <div className="announce-form">
              <div className="field">
                <label className="field__label" htmlFor="announce-body">
                  {t('announce.bodyLabel')}
                </label>
                <textarea
                  id="announce-body"
                  className="input"
                  rows={4}
                  maxLength={MAX_BODY + 1}
                  value={body}
                  placeholder={t('announce.bodyPlaceholder')}
                  onChange={(event) => setBody(event.target.value)}
                />
                <p className="field__hint">
                  {t('announce.bodyHint')}
                  {/* 計數器用 aria-live：它是唯一會在使用者打字時改變的東西，
                      讀螢幕軟體的使用者需要知道接近上限了。 */}
                  <span className={`announce-counter${overLimit ? ' announce-counter--over' : ''}`} aria-live="polite">
                    {formatNumber(bodyLength)} / {formatNumber(MAX_BODY)}
                  </span>
                </p>
              </div>

              <div className="announce-form__row">
                <div className="field">
                  <label className="field__check" htmlFor="announce-active">
                    <input
                      id="announce-active"
                      type="checkbox"
                      checked={active}
                      onChange={(event) => setActive(event.target.checked)}
                    />
                    <span>{t('announce.activeLabel')}</span>
                  </label>
                </div>
                <div className="field">
                  <label className="field__label" htmlFor="announce-expiry">
                    {t('announce.expiryLabel')}
                  </label>
                  <select
                    id="announce-expiry"
                    className="input"
                    value={expiryHours}
                    onChange={(event) => setExpiryHours(Number(event.target.value))}
                  >
                    {EXPIRY_OPTIONS.map((option) => (
                      <option key={option.hours} value={option.hours}>
                        {t(option.key, option.params)}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="announce-form__row">
                <BusyButton busy={busy} className="btn btn--primary" onClick={() => void save()}>
                  {busy ? t('announce.saving') : editing === null ? t('announce.new') : t('announce.edit')}
                </BusyButton>
                {editing === null ? null : (
                  <button className="btn btn--ghost" onClick={resetForm}>
                    {t('common.cancel')}
                  </button>
                )}
              </div>

              {formError ? <FormStatus message={formError} tone="error" /> : null}
            </div>
          </div>
        </section>

        {/* --- 公告清單 ---------------------------------------------------- */}
        <section className="panel" aria-labelledby="announce-list-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="announce-list-title">
                {t('announce.count', { count: formatNumber(items.length) })}
              </h2>
              <p className="panel__note">{t('announce.deleteNote')}</p>
            </div>
          </div>
          <div className="table-wrap">
            <table className="table announce-table">
              <thead>
                <tr>
                  <th scope="col">{t('announce.colBody')}</th>
                  <th scope="col">{t('announce.colState')}</th>
                  <th scope="col">{t('announce.colAuthor')}</th>
                  <th scope="col">{t('announce.colCreated')}</th>
                  <th scope="col">{t('announce.colActions')}</th>
                </tr>
              </thead>
              <tbody aria-busy={loading}>
                {items.length === 0 && !loading ? (
                  <tr>
                    <td colSpan={ANNOUNCE_COLUMNS}>
                      <EmptyState title={t('announce.empty')}>{t('announce.emptyBody')}</EmptyState>
                    </td>
                  </tr>
                ) : (
                  items.map((item) => (
                    <tr key={item.id}>
                      <td>
                        <span className="announce-body">{item.body}</span>
                      </td>
                      <td>
                        <StateBadge item={item} />
                      </td>
                      <td>
                        <span className="announce-table__author">{item.createdBy}</span>
                        {item.updatedBy && item.updatedBy !== item.createdBy ? (
                          <span className="announce-table__author">{item.updatedBy}</span>
                        ) : null}
                      </td>
                      <td>
                        <span className="cell-sub u-nowrap">{formatDateTime(item.createdAt)}</span>
                      </td>
                      <td>
                        <div className="row-actions">
                          <button className="btn btn--sm" onClick={() => startEdit(item)}>
                            {t('announce.edit')}
                          </button>
                          <button
                            className={`btn btn--sm${item.active ? '' : ' btn--primary'}`}
                            onClick={() => void setAnnouncementActive(item, !item.active)}
                          >
                            {item.active ? t('announce.deactivate') : t('announce.reactivate')}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </AdminShell>
  );
}

/**
 * 公告狀態的徽章。
 *
 * 三個狀態分成兩組顏色，理由是它們對管理員的意義不同：
 *   - 「顯示中」與「已過期」都是**沒有開關可按**的狀態（已過期的要重新設定
 *     時間），因此用中性色。
 *   - 「已停用」是可以隨時打開的（按重新啟用），因此用醒目色。
 * 若把三者都給不同顏色，顏色就不再傳達「這個能不能按」。
 */
function StateBadge({ item }: { item: AdminAnnouncementView }) {
  if (item.effective) {
    return <span className="badge badge--ok">{t('announce.stateActive')}</span>;
  }
  if (item.active) {
    // active 但沒有生效 → 一定是過期了（後端查詢時會排除過期的）。
    return <span className="badge badge--info">{t('announce.stateExpired')}</span>;
  }
  return <span className="badge badge--warn">{t('announce.stateInactive')}</span>;
}

/**
 * 把絕對到期時間換算成「還剩幾小時」，回傳 null 代表「永不過期」或「已過期」。
 *
 * 取**剩餘**而不是原始小時數的理由見呼叫端：一則很久以前設定的 7 天期公告，
 * 原始值 168 在編輯時已經毫無意義。
 */
function hoursUntil(expiresAt: string | undefined): number | null {
  if (!expiresAt) return null;
  const parsed = Date.parse(expiresAt);
  if (Number.isNaN(parsed)) return null;
  const hours = Math.round((parsed - Date.now()) / 3_600_000);
  return hours > 0 ? hours : null;
}
