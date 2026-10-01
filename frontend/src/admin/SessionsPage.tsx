/*
 * 登入與 Session 管理頁（/admin/sessions）
 *
 * 職責：列出目前仍有效的登入狀態，並提供「強制登出某個帳號的所有 session」。
 *
 * 這個頁面存在的理由是「cookie 即憑證」設計（見 backend/forum/session/session.go
 * 檔頭第一點）留下的一個洞：token 不輪替、不綁 IP、不綁 UA，因此遭竊的 cookie
 * 在過期前可被完整重用，而後臺原本沒有任何手段提前止血。「立刻撤銷所有 session」
 * 是唯一能提前止血的動作。
 *
 * 三個界面上必須存在的說明（缺任何一個都會造成實際的錯誤理解）
 *
 *   1. **為什麼沒有完整 token**（session.privacyNote）
 *      token 就是憑證。把它顯示出來，會讓「截圖分享」「有人在旁邊看螢幕」
 *      變成一次完整的手法移交。少了這段說明，第一個管理員會以為是未完成的功能
 *      而要求補上。
 *
 *   2. **滑動續期**（session.expireNote）
 *      使用者看到「剩 3 小時」會以為 3 小時後被登出，而實際上只要有活動就會一直
 *      延續。不說的話，這一頁會被讀成「離開 3 小時就會被踢」。
 *
 *   3. **掃描可能不完整**（session.truncated）
 *      後端的掃描有 key 數上限，而「沒列出來」若沒有被明確標成「沒掃完」，就會
 *      被讀成「這個帳號只有這些 session」—— 而那會讓管理員漏掉真正該看的。
 *
 * 刻意不做的事
 *
 *   - 不顯示每支 session 的 IP 與 User-Agent。session hash 裡沒有存這兩項，
 *     而為了顯示它們就得新增欄位 —— 那是為了診斷而擴大憑證的儲存面。token
 *     不綁 IP/UA 這件事本身就是既有的已知限制，補上這兩項也不會讓它消失。
 *   - 不提供「登出所有 session」。那是一個非常容易誤按的按鈕，而且按下之後
 *     管理員自己也被踢出，症狀是「我按了登出全部，結果我也登出了，而且沒有
 *     辦法再進來登出所有人」。
 */

import { useCallback, useEffect, useState } from 'react';

import { ApiError, adminApi } from '../api/admin';
import { errorMessage, formatDateTime, formatNumber, text } from '../core';
import { t, usePageTitle } from '../i18n';
import type { AdminSessionRevokeResponse, AdminSessionsResponse, AdminSessionView } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import { BusyButton, EmptyState, FormStatus } from './ui';

const SESSION_COLUMNS = 6;

/**
 * 剩餘時間的八階門檻（0–1 的比例）。
 *
 * 階梯而非連續長度有兩個理由：CSP 不允許 inline style（因此不能用
 * style={{ width }}），而八階對「快要到期嗎」這個判斷已經綽綽有餘。
 * 這個表達的損失是「剩 3 小時」與「剩 3.5 小時」看起來一樣 —— 而實際數字
 * 就寫在長條旁邊，因此那個損失不影響判讀。
 */
const REMAINING_BINS = [0.05, 0.15, 0.3, 0.45, 0.6, 0.75, 0.9, 1.01];

export function SessionsPage() {
  usePageTitle('title.adminSessions');

  const { toast, dialog } = useAdmin();
  const [data, setData] = useState<AdminSessionsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [emailDraft, setEmailDraft] = useState('');
  // 套用到查詢的 email。分開是為了讓「輸入」與「已套用」是兩個明確的狀態 ——
  // 每敲一個字就打一次 API 會讓這個頁面在管理員還沒打好 email 時就送出
  // 十幾個查詢。
  const [emailFilter, setEmailFilter] = useState('');
  const [revokingEmail, setRevokingEmail] = useState<string | null>(null);

  const load = useCallback(
    async (filter: string) => {
      setLoading(true);
      const query = filter ? `?email=${encodeURIComponent(filter)}` : '';
      try {
        const response = await adminApi<AdminSessionsResponse>(`/api/admin/sessions${query}`);
        setData(response);
        setError(null);
      } catch (thrown) {
        const message = errorMessage(thrown, t('session.loadFailed'));
        setError(message);
        toast(message, 'error');
      } finally {
        setLoading(false);
      }
    },
    [toast],
  );

  useEffect(() => {
    void load(emailFilter);
  }, [emailFilter, load]);

  const applyFilter = useCallback(() => {
    setEmailFilter(emailDraft.trim());
  }, [emailDraft]);

  const clearFilter = useCallback(() => {
    setEmailDraft('');
    setEmailFilter('');
  }, []);

  const revoke = useCallback(
    async (row: AdminSessionView) => {
      // 強制登出是不可逆的（使用者必須重新走 Google 登入），因此一定先問一次，
      // 而且訊息裡要寫清楚「有幾支 session 會被清掉」—— 那個數字讓管理者能
      // 判斷這是「清掉一台手機」還是「清掉整個帳號的所有裝置」。
      const ok = await dialog.confirm({
        title: t('session.revokeTitle', { email: row.email }),
        message: t('session.revokeMessage', { count: formatNumber(countByEmail(data, row.email)) }),
        confirmLabel: t('session.revoke'),
      });
      if (!ok) return;

      setRevokingEmail(row.email);
      try {
        const result = await adminApi<AdminSessionRevokeResponse, { email: string }>('/api/admin/sessions/revoke', {
          method: 'POST',
          body: { email: row.email },
        });
        if (result.revoked > 0) {
          toast(t('session.revokeDone', { count: formatNumber(result.revoked) }), 'ok');
        } else {
          // 0 筆不是錯誤：使用者可能在兩次點擊之間自己登出了。要說成
          // 「沒有可撤銷的 session」而不是「失敗」—— 後者會讓管理員以為
          // 撤銷機制壞了。
          toast(t('session.revokeNone'), 'ok');
        }
        await load(emailFilter);
      } catch (thrown) {
        // 409（掃描被截斷）與其他錯誤的處置方式不同，因此要分流 —— 而
        // errorMessage 只回訊息字串、沒有狀態碼，所以必須從 ApiError 上拿。
        //
        // 差別在於下一步該做什麼：409 的意思是「無法確認是否全部撤銷」，
        // 管理者應該等一下再試；其他錯誤（例如網路斷了）重試多半也一樣。
        // 兩者都附上 revokeUnavailable 那句解釋，因為它在兩種情況下都成立 ——
        // 它明確寫了「這不表示撤銷失敗」，那正是管理者此刻最需要的判斷依據。
        const isTruncated = thrown instanceof ApiError && thrown.status === 409;
        const message = errorMessage(thrown, t('session.revokeFailed'));
        toast(isTruncated ? `${message}　${t('session.revokeUnavailable')}` : message, 'error');
      } finally {
        setRevokingEmail(null);
      }
    },
    [data, dialog, emailFilter, load, toast],
  );

  const items = data?.items ?? [];

  return (
    <AdminShell active="/admin/sessions" pageTitle={t('session.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('session.eyebrow')}</p>
            <h1>{t('session.title')}</h1>
            <p className="page-head__copy">{t('session.copy')}</p>
          </div>
          <div className="page-head__actions">
            <BusyButton busy={loading} className="btn btn--sm" onClick={() => void load(emailFilter)}>
              {t('session.refresh')}
            </BusyButton>
          </div>
        </header>

        {error ? <FormStatus message={error} tone="error" /> : null}

        <section className="panel" aria-labelledby="session-list-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="session-list-title">
                {t('session.title')}
              </h2>
              <p className="panel__note">
                {data
                  ? t('session.summary', {
                      total: formatNumber(data.totalActive),
                      scanned: formatNumber(data.scanned),
                    })
                  : t('common.loading')}
              </p>
            </div>
          </div>
          <div className="panel__body">
            {/* 兩段說明放在表格之前而不是之後：它們是「讀這張表之前必須先知道的
                前提」，放在下面就變成看完才補的附註，而那時已經有人照著錯誤的
                理解做了決定。 */}
            <div className="session-privacy">
              <p className="session-privacy__title">{t('session.privacyTitle')}</p>
              <p className="session-privacy__note">{t('session.privacyNote')}</p>
            </div>
            <p className="session-expire">
              {t('session.expireNote', { hours: formatNumber(data?.expireInHours ?? 0) })}
            </p>

            {data?.truncated ? (
              <p className="session-truncated">
                {t('session.truncated', { scanned: formatNumber(data.scanned) })}
              </p>
            ) : null}

            <div className="session-filter">
              <div className="field">
                <label className="field__label" htmlFor="session-email">
                  {t('session.filterEmail')}
                </label>
                <input
                  id="session-email"
                  className="input"
                  value={emailDraft}
                  placeholder={t('session.filterPlaceholder')}
                  onChange={(event) => setEmailDraft(event.target.value)}
                  // Enter 直接套用：這一頁只有一個輸入框，而「打完按 Enter」
                  // 是最自然的意圖 —— 加一個查詢按鈕並不會讓它更自然。
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') applyFilter();
                  }}
                />
              </div>
              <button className="btn btn--primary" onClick={applyFilter}>
                {t('session.search')}
              </button>
              {emailFilter ? (
                <button className="btn btn--ghost" onClick={clearFilter}>
                  {t('session.clearFilter')}
                </button>
              ) : null}
            </div>
          </div>

          <div className="table-wrap">
            <table className="table session-table">
              <thead>
                <tr>
                  <th scope="col">{t('session.colUser')}</th>
                  <th scope="col" title={t('session.titleColumnNote')}>
                    {t('session.colToken')}
                  </th>
                  <th scope="col">{t('session.colCreated')}</th>
                  <th scope="col">{t('session.colExpires')}</th>
                  <th scope="col" className="num">
                    {t('session.colRemaining')}
                  </th>
                  <th scope="col">{t('session.colActions')}</th>
                </tr>
              </thead>
              <tbody aria-busy={loading}>
                {items.length === 0 && !loading ? (
                  <tr>
                    <td colSpan={SESSION_COLUMNS}>
                      <EmptyState title={t('session.empty')}>{t('session.emptyBody')}</EmptyState>
                    </td>
                  </tr>
                ) : (
                  items.map((row) => (
                    <tr key={row.tokenPrefix + row.createdAt}>
                      <td>
                        <span className="cell-primary">{row.email}</span>
                        {row.isAdmin ? <span className="badge badge--info">{t('session.adminBadge')}</span> : null}
                      </td>
                      <td>
                        <span className="session-token">{row.tokenPrefix}</span>
                      </td>
                      <td>
                        <span className="cell-sub u-nowrap">
                          {row.createdAt ? formatDateTime(row.createdAt) : t('session.unknown')}
                        </span>
                      </td>
                      <td>
                        <span className="cell-sub u-nowrap">
                          {row.expiresAt ? formatDateTime(row.expiresAt) : t('session.unknown')}
                        </span>
                      </td>
                      <td className="num">
                        <RemainingBar session={row} expireInHours={data?.expireInHours ?? 0} />
                      </td>
                      <td>
                        <button
                          className="btn btn--sm btn--danger"
                          disabled={revokingEmail !== null}
                          onClick={() => void revoke(row)}
                        >
                          {revokingEmail === row.email ? t('session.revokeRunning') : t('session.revoke')}
                        </button>
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
 * 剩餘時間的長條 + 文字。
 *
 * 文字是必要的：長條只表達「相對多少」，而管理者要判斷的是「這個帳號還有
 * 多久會自己過期」。兩者一起給，長條負責快速掃視、文字負責精確判讀。
 */
function RemainingBar({ session, expireInHours }: { session: AdminSessionView; expireInHours: number }) {
  // TTL 未知（負值）或 expire 為 0 時無法算比例，此時不畫長條 —— 一根沒有
  // 意義的長條比沒有長條更糟。
  if (session.ttlSeconds < 0 || expireInHours <= 0) {
    return <span className="muted">{t('session.unknown')}</span>;
  }
  const total = expireInHours * 3600;
  const ratio = Math.min(1, Math.max(0, session.ttlSeconds / total));
  const bin = REMAINING_BINS.findIndex((edge) => ratio < edge);
  const level = bin < 0 ? REMAINING_BINS.length - 1 : bin;

  return (
    <span className="u-nowrap">
      <span className={`session-remaining session-remaining--${level}`} aria-hidden="true">
        <span className="session-remaining__fill" />
      </span>
      <span className="cell-sub">{text(formatRemaining(session.ttlSeconds))}</span>
    </span>
  );
}

/**
 * 把剩餘秒數寫成「2 天 3 小時」這種形式。
 *
 * 刻意不顯示成「2.9 天」：那是把兩個不同單位混在一起的一個小數，而管理者
 * 要做的判斷（「他大概多久會自己消失」）在天與小時的尺度上就夠了。
 */
function formatRemaining(seconds: number): string {
  const days = Math.floor(seconds / 86_400);
  const hours = Math.floor((seconds % 86_400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

/** countByEmail 數出某個 email 在目前這一頁有幾支 session。
 *
 *  用途是確認對話框裡的「N 筆」對得上使用者實際看到的列數。若只回 1
 *  （按鈕所在的那一列），管理者會以為「只登出這一筆」，而實際上撤銷的是
 *  **該帳號的所有** session —— 那個差別很大，因此確認訊息裡的數字必須是
 *  使用者看得到的數字。
 *
 *  局限：它只數目前這一頁（受 limit 影響）。若某個帳號有 30 支 session 而
 *  頁面只顯示 10 支，訊息會說 10 而實際是 30。這個不精確是有意接受的 ——
 *  取得精確數字需要另一支 API，而為了確認對話框的文案多打一次全站掃描
 *  不划算。訊息裡寫的是「目前的 N 筆」而不是「共 N 筆」。
 */
function countByEmail(data: AdminSessionsResponse | null, email: string): number {
  if (!data) return 0;
  return data.items.filter((item) => item.email === email).length;
}
