/*
 * IP 封鎖名單頁（/admin/blocks）
 *
 * 職責：把已確認濫用的來源位址加進封鎖名單，或從名單移除。
 *
 * 這個頁面存在的理由是「限流不夠」：限流是行程內的滑動視窗，重啟即失效、
 * 而且不跨行程（見 backend/forum/httpapi/ratelimit.go 檔頭）。對付持續性的
 * 自動化攻擊，那意味著攻擊者只要等一次部署就重新拿到滿額度，而且沒有任何人
 * 收到通知。封鎖名單存在 Redis，因此跨行程、跨重啟都存活。
 *
 * 這��頁面刻意**不做**自動封鎖
 *
 * 超過限流額度的位址只會拿到 429，不會被自動加進名單。理由寫在
 * backend/forum/ipban 的檔頭：同一個出口位址可能是一整間辦公室或一整個
 * NAT，而自動封鎖會誤傷他們 —— 誤封正常使用者的後果比多讓一個腳本多打幾次
 * 嚴重得多。封鎖必須是管理員的決定。
 *
 * 界面上必須說清楚的三件事
 *
 *   1. **只擋寫入**（block.scopeNote）。被封鎖的人仍然可以讀取、可以登入。
 *      不說的話，被封鎖的使用者會以為「整個網站對我關掉了」而回報一個不存在的
 *      問題；而管理員會以為需要另外去動帳號權限。
 *   2. **不會自動填滿**（block.notAutoNote）。見上。
 *   3. **最長一年**（block.maxNote）。輸入更長的時間會被收斂到一年，理由見
 *      ipban.maxBanDuration。
 */

import { useCallback, useEffect, useMemo, useState } from 'react';

import { adminApi } from '../api/admin';
import { errorMessage, formatDateTime, formatNumber, text } from '../core';
import { t, usePageTitle } from '../i18n';
import type { AdminBlockRemoveResponse, AdminBlocksResponse, AdminBlockView } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import { BusyButton, EmptyState, FormStatus } from './ui';

const BLOCK_COLUMNS = 4;

/**
 * 可選的封鎖時長。
 *
 * 用明確的選項而不是「輸入分鐘數」：那個數字欄位最常見的填法是「1」（以為
 * 是 1 天），而後端只能把它當成 1 分鐘 —— 一分鐘後攻擊者就回來了，而管理員
 * 以為自己處理完了。後端仍然有一分鐘的下限作為第二道防線，但介面上不給那個
 * 機會。
 *
 * 一年對應後端的 ipban.MaxBanDuration；再往上沒有意義（後端會收斂），
 * 而給一個「永久」選項等於給管理員一個他沒打算承擔的承諾。
 */
const DURATIONS = [
  { minutes: 60, key: 'common.oneHour' },
  { minutes: 60 * 24, key: 'common.oneDay' },
  { minutes: 60 * 24 * 7, key: 'common.sevenDays' },
  { minutes: 60 * 24 * 30, key: 'common.thirtyDays' },
  { minutes: 60 * 24 * 365, key: 'common.oneYear' },
] as const;

/** 剩餘時間長條的八階門檻（0–1 的比例）。理由同 sessions.html 的同款長條。 */
const REMAINING_BINS = [0.05, 0.15, 0.3, 0.45, 0.6, 0.75, 0.9, 1.01];

/** 最長可選時長，用來把剩餘時間長條正規化。 */
const MAX_MINUTES = 60 * 24 * 365;

export function BlocksPage() {
  usePageTitle('title.adminBlocks');

  const { toast, dialog } = useAdmin();
  const [data, setData] = useState<AdminBlocksResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const [ip, setIp] = useState('');
  const [minutes, setMinutes] = useState<number>(60 * 24);
  const [reason, setReason] = useState('');
  const [formError, setFormError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await adminApi<AdminBlocksResponse>('/api/admin/blocks');
      setData(response);
      setError(null);
    } catch (thrown) {
      const message = errorMessage(thrown, t('block.loadFailed'));
      setError(message);
      toast(message, 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const available = data?.available !== false;
  const items = data?.items ?? [];

  const block = useCallback(async () => {
    const address = ip.trim();
    if (address === '') {
      setFormError(t('block.ipInvalid'));
      return;
    }
    setBusy(true);
    setFormError(null);
    try {
      const result = await adminApi<{ ip: string; expiresAt: string }, { ip: string; minutes: number; reason: string }>(
        '/api/admin/blocks',
        { method: 'POST', body: { ip: address, minutes, reason: reason.trim() } },
      );
      toast(t('block.done', { ip: result.ip }), 'ok');
      // 清空表單。刻意不做「保留 IP 只留原因」：那會讓第二次封鎖看起來像是
      // 忘了改 IP —— 而把一個人打錯的位址又封一次是這一頁最容易犯的錯。
      setIp('');
      setReason('');
      await load();
    } catch (thrown) {
      const message = errorMessage(thrown, t('block.failed'));
      setFormError(message);
      toast(message, 'error');
    } finally {
      setBusy(false);
    }
  }, [ip, load, minutes, reason, toast]);

  const unblock = useCallback(
    async (entry: AdminBlockView) => {
      const ok = await dialog.confirm({
        title: t('block.unblockTitle', { ip: entry.ip }),
        message: t('block.unblockMessage'),
        confirmLabel: t('block.unblock'),
      });
      if (!ok) return;
      try {
        const result = await adminApi<AdminBlockRemoveResponse, { ip: string; minutes: number }>(
          '/api/admin/blocks',
          { method: 'POST', body: { ip: entry.ip, minutes: 0 } },
        );
        // existed=false 代表那個位址本來就沒被封 —— 那不是錯誤，說成
        // 「它本來就沒被封鎖」比說成「解封失敗」誠實。
        toast(result.existed ? t('block.removed', { ip: result.ip }) : t('block.removedNone', { ip: result.ip }), 'ok');
        await load();
      } catch (thrown) {
        toast(errorMessage(thrown, t('block.failed')), 'error');
      }
    },
    [dialog, load, toast],
  );

  const durationLabel = useMemo(() => {
    const found = DURATIONS.find((item) => item.minutes === minutes);
    return found ? t(found.key) : text(`${minutes}m`);
  }, [minutes]);

  if (!available) {
    return (
      <AdminShell active="/admin/blocks" pageTitle={t('block.title')}>
        <div className="stack">
          <header className="page-head">
            <div className="page-head__text">
              <p className="eyebrow">{t('block.eyebrow')}</p>
              <h1>{t('block.title')}</h1>
              <p className="page-head__copy">{t('block.copy')}</p>
            </div>
          </header>
          <EmptyState title={t('block.unavailable')}>{t('block.unavailableNote')}</EmptyState>
        </div>
      </AdminShell>
    );
  }

  return (
    <AdminShell active="/admin/blocks" pageTitle={t('block.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('block.eyebrow')}</p>
            <h1>{t('block.title')}</h1>
            <p className="page-head__copy">{t('block.copy')}</p>
          </div>
          <div className="page-head__actions">
            <BusyButton busy={loading} className="btn btn--sm" onClick={() => void load()}>
              {t('block.refresh')}
            </BusyButton>
          </div>
        </header>

        {error ? <FormStatus message={error} tone="error" /> : null}

        <div className="block-notes">
          <p className="block-note">
            <span className="block-note__label">{t('block.scopeNoteLabel')}</span>
            {t('block.scopeNote')}
          </p>
          <p className="block-note">
            <span className="block-note__label">{t('block.notAutoNoteLabel')}</span>
            {t('block.notAutoNote')}
          </p>
          <p className="block-note">
            <span className="block-note__label">{t('block.maxNoteLabel')}</span>
            {t('block.maxNote')}
          </p>
        </div>

        <section className="panel" aria-labelledby="block-add-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="block-add-title">
                {t('block.add')}
              </h2>
              <p className="panel__note">{t('block.addMessage')}</p>
            </div>
          </div>
          <div className="panel__body">
            <div className="block-form">
              <div className="field">
                <label className="field__label" htmlFor="block-ip">
                  {t('block.ipLabel')}
                </label>
                <input
                  id="block-ip"
                  className="input u-mono"
                  value={ip}
                  placeholder={t('block.ipPlaceholder')}
                  onChange={(event) => setIp(event.target.value)}
                />
              </div>
              <div className="field">
                <label className="field__label" htmlFor="block-duration">
                  {t('block.durationLabel')}
                </label>
                <select
                  id="block-duration"
                  className="input"
                  value={minutes}
                  onChange={(event) => setMinutes(Number(event.target.value))}
                >
                  {DURATIONS.map((item) => (
                    <option key={item.minutes} value={item.minutes}>
                      {t(item.key)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="field block-form__reason">
                <label className="field__label" htmlFor="block-reason">
                  {t('block.reasonLabel')}
                </label>
                <input
                  id="block-reason"
                  className="input"
                  value={reason}
                  maxLength={200}
                  placeholder={t('block.reasonPlaceholder')}
                  onChange={(event) => setReason(event.target.value)}
                />
                <p className="field__hint">{t('block.reasonHint')}</p>
              </div>
              <div className="block-form__actions">
                <BusyButton busy={busy} className="btn btn--danger" onClick={() => void block()}>
                  {busy ? t('block.blocking') : t('block.add')}
                </BusyButton>
                <span className="cell-sub">{durationLabel}</span>
              </div>
            </div>
            {formError ? <FormStatus message={formError} tone="error" /> : null}
          </div>
        </section>

        <section className="panel" aria-labelledby="block-list-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="block-list-title">
                {t('block.count', { count: formatNumber(items.length) })}
              </h2>
            </div>
          </div>
          <div className="table-wrap">
            <table className="table block-table">
              <thead>
                <tr>
                  <th scope="col">{t('block.colIp')}</th>
                  <th scope="col">{t('block.colExpires')}</th>
                  <th scope="col" className="num">
                    {t('block.colRemaining')}
                  </th>
                  <th scope="col">{t('block.colActions')}</th>
                </tr>
              </thead>
              <tbody aria-busy={loading}>
                {items.length === 0 && !loading ? (
                  <tr>
                    <td colSpan={BLOCK_COLUMNS}>
                      <EmptyState title={t('block.empty')}>{t('block.emptyBody')}</EmptyState>
                    </td>
                  </tr>
                ) : (
                  items.map((entry) => (
                    <tr key={entry.ip}>
                      <td>
                        <span className="block-ip">{entry.ip}</span>
                      </td>
                      <td>
                        <span className="cell-sub u-nowrap">{formatDateTime(entry.expiresAt)}</span>
                      </td>
                      <td className="num">
                        <RemainingBar seconds={entry.remainingSeconds} />
                      </td>
                      <td>
                        <button className="btn btn--sm" onClick={() => void unblock(entry)}>
                          {t('block.unblock')}
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
 * 相對於「最長可選時長」而不是「一年」：兩個長條要能互相比較，而比較的前提
 * 是同一個尺度。管理員同時看到「1 小時」與「30 天」的封鎖時，兩根長條的
 * 滿度必須代表同樣的長度。
 */
function RemainingBar({ seconds }: { seconds: number }) {
  if (seconds < 0) {
    return <span className="muted">{t('session.unknown')}</span>;
  }
  const ratio = Math.min(1, Math.max(0, seconds / (MAX_MINUTES * 60)));
  const bin = REMAINING_BINS.findIndex((edge) => ratio < edge);
  const level = bin < 0 ? REMAINING_BINS.length - 1 : bin;
  return (
    <span className="u-nowrap">
      <span className={`block-remaining block-remaining--${level}`} aria-hidden="true">
        <span className="block-remaining__fill" />
      </span>
      <span className="cell-sub">{text(formatRemaining(seconds))}</span>
    </span>
  );
}

/** 把剩餘秒數寫成「2 天 3 小時」這種形式。理由同 sessions.html 的同款。 */
function formatRemaining(seconds: number): string {
  const days = Math.floor(seconds / 86_400);
  const hours = Math.floor((seconds % 86_400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}
