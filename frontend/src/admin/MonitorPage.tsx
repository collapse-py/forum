/*
 * 系統監控頁（/admin/monitor）
 *
 * 職責：把後端 /api/admin/monitor 回來的監控資料畫成五個區塊 —— 服務狀態、
 * 請求總覽、近 N 分鐘流量、依路由統計、限流器。
 *
 * 這個頁面與其他三個後臺頁最大的不同是它「會自己動」。用戶管理與檢舉管理是
 * 一次載入、使用者操作後再重取；監控頁如果同樣只在載入時讀一次，那它在三分
 * 鐘之後就變成一張過期的快照，而它存在的唯一理由就是回答「現在正在怎樣」。
 * 因此這裡用兩種機制讓它保持即時，且都刻意可被使用者關掉：
 *
 *   1. 固定週期輪詢（REFRESH_INTERVAL_MS）。可切換、可暫停。
 *   2. visibilitychange：分頁切到背景就暫停。理由是背景分頁繼續每十秒打一次
 *      請求，對一個已經有人在用的後臺頁面來說是純粹的浪費；而且管理員切回
 *      分頁時看到的應該是「剛讀的」而不是「三十秒前讀的」。
 *
 * 刻意不做的事：
 *
 *   - 不畫「即時」以外的假曲線。不為了讓圖好看而對沒有資料的分鐘補零：圖上
 *     沒有柱子的分鐘代表服務當時沒有流量，那和「有流量但都是零」是兩回事。
 *   - 不在頁面上顯示任何 email、路徑參數或 cookie。路由已經由後端正規化
 *     （數字與 email 換成 :id），所以這裡不需要（也不該）再做任何過濾。
 *   - 不顯示 p50/p95 之外的精確延遲。後端回來的分位數本來就是直方圖的桶上界
 *     （見 metrics 套件說明），把它當成精確值呈現會是誤導；這一頁因此照實
 *     標示為分位數，並在時間軸的說明文字裡寫清楚。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { adminApi } from '../api/admin';
import { errorMessage, formatDateTime, formatNumber, text } from '../core';
import { Icon } from '../icons';
import { t, usePageTitle } from '../i18n';
import type { MonitorDependency, MonitorResponse } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import { BusyButton, EmptyState, FormStatus } from './ui';

/** 自動更新週期。10 秒是「足以看出流量變化」與「不把後端打成負載」之間的取捨。 */
const REFRESH_INTERVAL_MS = 10_000;

/** 時間軸的長度（分鐘）。144 = 24 小時，與後端的 maxTimelinePoints 一致。 */
const TIMELINE_MINUTES = 144;

/** 路由表格最多顯示幾列。更多列在 .table-wrap 內捲動，見 .monitor-table。 */
const MAX_ROUTE_ROWS = 40;

const ROUTE_COLUMNS = 7;
const LIMIT_COLUMNS = 6;

/* ==========================================================================
   格式化
   ========================================================================== */

/** 位元組 → 人看得懂的字串。1024 進位，與作業系統的慣例一致。 */
function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  // 對數決定單位；Math.log10(0) 已在上面被 <= 0 擋掉。
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / 1024 ** exponent;
  // 小於 10 時保留一位小數：1.0 GiB 與 1 GiB 的差別在診斷上不重要，
  // 但 0.1 MiB 與 0.2 MiB 的差別在診斷上非常關鍵。
  const digits = exponent === 0 || value >= 10 ? 0 : 1;
  return `${value.toFixed(digits)} ${units[exponent] ?? 'B'}`;
}

/** 毫秒 → 短字串。刻意不帶單位尾巴（欄標題已經寫了「延遲」）。 */
function formatMs(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0';
  if (value >= 1000) return `${(value / 1000).toFixed(1)}s`;
  if (value >= 10) return String(Math.round(value));
  return value.toFixed(1);
}

/** 秒數 → 1 天 3 小時 / 5 分鐘 12 秒。用於運行時間。 */
function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '0s';
  const total = Math.floor(seconds);
  const days = Math.floor(total / 86_400);
  const hours = Math.floor((total % 86_400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const rest = total % 60;
  // 只顯示非零的最高兩級：天數已經很大時再加小時是雜訊，分鐘已經很大時再加
  // 秒數也是雜訊。這個頁面顯示的是「運行多久」，不是計時競賽。
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${rest}s`;
  return `${rest}s`;
}

/**
 * 時間軸刻度用的「時:分」。
 *
 * 刻意不用 Intl.DateTimeFormat 的完整日期時間：那會在 144 個刻度上佔掉大量
 * 寬度，而時間軸的橫軸只需要分鐘 —— 分鐘索引連續，因此刻度只標整點與半点。
 */
function formatClock(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return text(value);
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  return `${hours}:${minutes}`;
}

/* ==========================================================================
   型別窄化
   ========================================================================== */

/** Detail 是後端放該依賴特有數值的地方，形狀因依賴而異。 */
function detailNumber(detail: Record<string, unknown> | undefined, key: string): number | null {
  const value = detail?.[key];
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

/* ==========================================================================
   頁面
   ========================================================================== */

export function MonitorPage() {
  usePageTitle('title.adminMonitor');

  const { toast } = useAdmin();
  const [data, setData] = useState<MonitorResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);
  // 分頁是否可見。false 期間不輪詢，理由見檔頭。
  const [visible, setVisible] = useState(() => !document.hidden);
  // 最後一次成功讀取的時刻。顯示它而不是「現在」，因為頁面卡住時這兩個值
  // 必須看得出差別 —— 否則「10 秒前更新」會在請求卡死時仍然繼續走字。
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [countdown, setCountdown] = useState(REFRESH_INTERVAL_MS / 1000);

  // 請求進行中不排下一輪。用 ref 而不是 state：它是給 effect 讀的，不參與
  // 渲染；放進 state 會讓「開始讀取」觸發一次重繪，而那個重繪沒有任何畫面
  // 變化。
  const inFlightRef = useRef(false);
  // AbortController 讓「暫停自動更新」能真正中止進行中的請求，而不只是不再
  // 排下一輪。不中止的話，按下暫停之後還會有半秒才停下來。
  const abortRef = useRef<AbortController | null>(null);

  const load = useCallback(
    async (options: { quiet: boolean }) => {
      if (inFlightRef.current) return;
      inFlightRef.current = true;
      if (!options.quiet) setLoading(true);

      const controller = new AbortController();
      abortRef.current = controller;
      try {
        const next = await adminApi<MonitorResponse>('/api/admin/monitor', { signal: controller.signal });
        setData(next);
        setError(null);
        setLastUpdated(new Date());
        setCountdown(REFRESH_INTERVAL_MS / 1000);
      } catch (thrown) {
        // 401 由 adminApi 導回登入閘門，那條路徑會整頁跳轉，因此在這裡
        // 不該出現（除非 session 剛好過期）。其他失敗保留上一次成功的資料：
        // 監控頁在故障時最有價值的狀態就是「最後幾秒是正常的，現在連不上」。
        if (!controller.signal.aborted) {
          const message = errorMessage(thrown, t('monitor.unreachable'));
          setError(message);
          if (options.quiet) toast(message, 'error');
        }
      } finally {
        if (abortRef.current === controller) abortRef.current = null;
        inFlightRef.current = false;
        setLoading(false);
      }
    },
    [toast],
  );

  // 首次載入，以及自動更新的開關改變時立刻重讀一次。
  // 依賴陣列刻意不含 load：它是 useCallback 穩定的，而把它放進去會讓
  // 「自動更新開關」變成每次重繪都重新建立 interval。
  useEffect(() => {
    void load({ quiet: false });
  }, [load, autoRefresh]);

  useEffect(() => {
    if (!autoRefresh || !visible) return;
    const timer = window.setInterval(() => {
      void load({ quiet: true });
    }, REFRESH_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [autoRefresh, visible, load]);

  // 倒數字只在自動更新且分頁可見時前進；否則停在最後的值，讓「暫停」在畫面上
  // 是可見的（否則使用者只會看到一個不動的數字，不知道它已經停了）。
  useEffect(() => {
    if (!autoRefresh || !visible) return;
    const timer = window.setInterval(() => {
      setCountdown((value) => (value <= 1 ? REFRESH_INTERVAL_MS / 1000 : value - 1));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [autoRefresh, visible]);

  useEffect(() => {
    const onVisibilityChange = () => setVisible(!document.hidden);
    document.addEventListener('visibilitychange', onVisibilityChange);
    // 中止進行中的請求，讓切到背景時不留一個還在等的 fetch。
    return () => {
      document.removeEventListener('visibilitychange', onVisibilityChange);
      abortRef.current?.abort();
    };
  }, []);

  const stats = data?.stats ?? null;
  const dependencies = data?.dependencies ?? null;
  const routes = useMemo(() => (stats?.requests.routes ?? []).slice(0, MAX_ROUTE_ROWS), [stats]);
  const limits = stats?.rateLimits ?? [];
  const errorRate = stats && stats.requests.total > 0
    ? ((stats.requests.clientErrors + stats.requests.serverErrors) / stats.requests.total) * 100
    : 0;
  const blockedTotal = limits.reduce((sum, limit) => sum + limit.blocked, 0);
  const dbDetail = dependencies?.mysql?.detail;

  return (
    <AdminShell active="/admin/monitor" pageTitle={t('monitor.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('monitor.eyebrow')}</p>
            <h1>{t('monitor.title')}</h1>
            <p className="page-head__copy">{t('monitor.copy')}</p>
          </div>
          <div className="page-head__actions">
            <label className="switch">
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(event) => setAutoRefresh(event.target.checked)}
              />
              <span>{t('monitor.autoRefresh')}</span>
            </label>
            <BusyButton
              busy={loading}
              className="btn btn--sm"
              onClick={() => void load({ quiet: false })}
              title={autoRefresh ? t('monitor.autoRefreshOn', { seconds: countdown }) : t('monitor.autoRefreshOff')}
            >
              {loading ? t('monitor.refreshing') : t('monitor.refresh')}
            </BusyButton>
          </div>
        </header>

        {/* 狀態列：這三件事決定「畫面上的數字還值不值得看」。 */}
        <p className="monitor-status" role="status" aria-live="polite">
          {error ? (
            <span className="monitor-status__error">{error}</span>
          ) : null}
          {data ? (
            <span>
              {t('monitor.lastUpdated', { time: lastUpdated ? formatDateTime(lastUpdated) : t('common.placeholder') })}
              {' · '}
              {t('monitor.probeTook', { ms: formatNumber(Math.round(data.probesMs ?? 0)) })}
            </span>
          ) : null}
          {autoRefresh && visible ? (
            <span>{t('monitor.nextUpdate', { seconds: countdown })}</span>
          ) : autoRefresh ? (
            <span>{t('monitor.visibilityPaused')}</span>
          ) : (
            <span>{t('monitor.pausedHint')}</span>
          )}
        </p>

        {error && !data ? (
          <section className="panel">
            <EmptyState title={t('monitor.loadFailed')}>
              {t('monitor.loadFailedHint')}
            </EmptyState>
          </section>
        ) : null}

        {/* --- 服務狀態 -------------------------------------------------- */}
        <section className="panel" aria-labelledby="monitor-deps-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="monitor-deps-title">
                {t('monitor.depsTitle')}
              </h2>
              <p className="panel__note">{t('monitor.depsNote')}</p>
            </div>
          </div>
          <div className="panel__body">
            <div className="dep-grid">
              <DependencyCard
                name={t('monitor.depMysql')}
                status={dependencies?.mysql}
                lines={[
                  dbDetail === undefined
                    ? null
                    : t('monitor.depPoolUsage', {
                        inUse: formatNumber(detailNumber(dbDetail, 'inUse') ?? 0),
                        open: formatNumber(detailNumber(dbDetail, 'openConnections') ?? 0),
                        max: formatNumber(detailNumber(dbDetail, 'maxOpenConnections') ?? 0),
                      }),
                  dbDetail === undefined
                    ? null
                    : t('monitor.depPoolWait', {
                        count: formatNumber(detailNumber(dbDetail, 'waitCount') ?? 0),
                        ms: formatNumber(Math.round(detailNumber(dbDetail, 'waitDurationMs') ?? 0)),
                      }),
                ]}
              />
              <DependencyCard
                name={t('monitor.depRedis')}
                status={dependencies?.redis}
                lines={[
                  dependencies?.redis?.detail
                    ? t('monitor.depKeys', { count: formatNumber(detailNumber(dependencies.redis.detail, 'keys') ?? 0) })
                    : null,
                  dependencies?.redis?.detail
                    ? t('monitor.depMemory', { size: formatBytes(detailNumber(dependencies.redis.detail, 'usedMemoryBytes') ?? 0) })
                    : null,
                  dependencies?.redis?.detail
                    ? t('monitor.depRedisPool', {
                        hits: formatNumber(detailNumber(dependencies.redis.detail, 'poolHits') ?? 0),
                        misses: formatNumber(detailNumber(dependencies.redis.detail, 'poolMisses') ?? 0),
                      })
                    : null,
                ]}
              />
              <DependencyCard
                name={t('monitor.depSearch')}
                status={dependencies?.search}
                lines={[
                  dependencies?.search?.state === 'disabled'
                    ? t('monitor.depSearchFallback')
                    : dependencies?.search?.state === 'down'
                      ? null
                      : t('monitor.depEngineEs'),
                ]}
              />
            </div>
          </div>
        </section>

        {/* --- 請求總覽 -------------------------------------------------- */}
        <section className="panel" aria-labelledby="monitor-stats-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="monitor-stats-title">
                {t('monitor.statsTitle')}
              </h2>
              <p className="panel__note">
                {stats?.runtime
                  ? t('monitor.goVersion', {
                      version: stats.runtime.version,
                      cpu: formatNumber(stats.runtime.numCpu),
                      gc: formatNumber(stats.runtime.gcCycles),
                    })
                  : t('monitor.noData')}
              </p>
            </div>
          </div>
          <div className="panel__body">
            <div className="stats">
              <StatCard
                label={t('monitor.statUptime')}
                value={formatDuration(stats?.uptimeSeconds ?? 0)}
              />
              <StatCard
                label={t('monitor.statRequests')}
                value={formatNumber(stats?.requests.total ?? 0)}
                note={t('monitor.statCountWithPeak', { peak: formatNumber(stats?.requests.maxInFlight ?? 0) })}
              />
              <StatCard
                label={t('monitor.statErrorRate')}
                value={`${errorRate.toFixed(1)}%`}
                tone={stats && stats.requests.serverErrors > 0 ? 'warn' : stats && errorRate > 5 ? 'warn' : 'ok'}
                note={
                  stats
                    ? t('monitor.statBlockedSplit', {
                        client: formatNumber(stats.requests.clientErrors),
                        server: formatNumber(stats.requests.serverErrors),
                      })
                    : undefined
                }
              />
              <StatCard label={t('monitor.statP95')} value={formatMs(stats?.requests.p95Ms ?? 0)} />
              <StatCard
                label={t('monitor.statInFlight')}
                value={formatNumber(stats?.requests.inFlight ?? 0)}
              />
              <StatCard
                label={t('monitor.statGoroutines')}
                value={formatNumber(stats?.runtime.goroutines ?? 0)}
              />
              <StatCard
                label={t('monitor.statHeap')}
                value={formatBytes(stats?.runtime.heapInUseBytes ?? 0)}
              />
              <StatCard
                label={t('monitor.statDbPool')}
                value={formatNumber(detailNumber(dbDetail, 'inUse') ?? 0)}
                note={t('monitor.statCountWithInUse', {
                  inUse: formatNumber(detailNumber(dbDetail, 'inUse') ?? 0),
                  idle: formatNumber(detailNumber(dbDetail, 'idle') ?? 0),
                })}
              />
              <StatCard
                label={t('monitor.statRateLimited')}
                value={formatNumber(blockedTotal)}
                tone={blockedTotal > 0 ? 'warn' : undefined}
              />
            </div>
          </div>
        </section>

        {/* --- 時間軸 ---------------------------------------------------- */}
        <section className="panel" aria-labelledby="monitor-timeline-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="monitor-timeline-title">
                {t('monitor.timelineTitle', { minutes: TIMELINE_MINUTES })}
              </h2>
              <p className="panel__note">{t('monitor.timelineNote')}</p>
            </div>
            <div className="panel__actions">
              <p className="monitor-legend">
                <span className="monitor-legend__item">
                  <i className="monitor-legend__swatch monitor-legend__swatch--volume" aria-hidden="true" />
                  {t('monitor.timelineLegendVolume')}
                </span>
                <span className="monitor-legend__item">
                  <i className="monitor-legend__swatch monitor-legend__swatch--error" aria-hidden="true" />
                  {t('monitor.timelineLegendError')}
                </span>
                <span className="monitor-legend__item">
                  <i className="monitor-legend__swatch monitor-legend__swatch--history" aria-hidden="true" />
                  {t('monitor.timelineHistory')}
                </span>
              </p>
            </div>
          </div>
          <div className="panel__body">
            <Timeline points={stats?.timeline ?? []} loading={loading && stats === null} />
            {!stats?.historyLoaded ? (
              <p className="monitor-hint">{t('monitor.timelineNoHistory')}</p>
            ) : null}
          </div>
        </section>

        {/* --- 依路由統計 ------------------------------------------------ */}
        <section className="panel" aria-labelledby="monitor-routes-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="monitor-routes-title">
                {t('monitor.routesTitle')}
              </h2>
              <p className="panel__note">{t('monitor.routesNote')}</p>
            </div>
          </div>
          <div className="table-wrap">
            <table className="table monitor-routes">
              <thead>
                <tr>
                  <th scope="col">{t('monitor.colRoute')}</th>
                  <th scope="col" className="num">
                    {t('monitor.colCount')}
                  </th>
                  <th scope="col" className="num">
                    {t('monitor.colAvg')}
                  </th>
                  <th scope="col" className="num">
                    {t('monitor.colP50')}
                  </th>
                  <th scope="col" className="num">
                    {t('monitor.colP95')}
                  </th>
                  <th scope="col" className="num">
                    {t('monitor.colMax')}
                  </th>
                  <th scope="col" className="num">
                    {t('monitor.colErrors')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {routes.length === 0 ? (
                  <tr>
                    <td colSpan={ROUTE_COLUMNS}>
                      <div className="state">
                        <Icon name="inbox" />
                        <strong>{t('monitor.noData')}</strong>
                        <p>{t('monitor.noDataBody')}</p>
                      </div>
                    </td>
                  </tr>
                ) : (
                  routes.map((route) => (
                    <tr key={`${route.method} ${route.route}`}>
                      <td>
                        <span className="method">{route.method}</span>
                        <span className="cell-primary">
                          {route.route === '__other__' ? t('monitor.routeOther') : route.route}
                        </span>
                      </td>
                      <td className="num">{formatNumber(route.total)}</td>
                      <td className="num">{formatMs(route.avgMs)}</td>
                      <td className="num">{formatMs(route.p50Ms)}</td>
                      <td className="num">{formatMs(route.p95Ms)}</td>
                      <td className="num">{formatMs(route.maxMs)}</td>
                      <td className="num">
                        {route.clientErrors + route.serverErrors > 0 ? (
                          <span className="badge badge--warn">
                            {formatNumber(route.clientErrors + route.serverErrors)}
                          </span>
                        ) : (
                          <span className="muted">0</span>
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* --- 限流器 ---------------------------------------------------- */}
        <section className="panel" aria-labelledby="monitor-limits-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="monitor-limits-title">
                {t('monitor.limitsTitle')}
              </h2>
              <p className="panel__note">{t('monitor.limitsNote')}</p>
            </div>
          </div>
          <div className="table-wrap">
            <table className="table monitor-limits">
              <thead>
                <tr>
                  <th scope="col">{t('monitor.colLimiter')}</th>
                  <th scope="col">{t('monitor.colBudget')}</th>
                  <th scope="col" className="num">
                    {t('monitor.colAllowed')}
                  </th>
                  <th scope="col" className="num">
                    {t('monitor.colBlocked')}
                  </th>
                  <th scope="col" className="num">
                    {t('monitor.colTracked')}
                  </th>
                  <th scope="col" className="num">
                    {t('monitor.colBlockedRate')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {limits.length === 0 ? (
                  <tr>
                    <td colSpan={LIMIT_COLUMNS}>
                      <EmptyState title={t('monitor.noLimits')}>{t('monitor.noLimitsBody')}</EmptyState>
                    </td>
                  </tr>
                ) : (
                  limits.map((limit) => (
                    <tr key={limit.name}>
                      <td>
                        <span className="cell-primary">{limitName(limit.name)}</span>
                      </td>
                      <td className="cell-sub u-nowrap">{limitBudget(limit)}</td>
                      <td className="num">{formatNumber(limit.allowed)}</td>
                      <td className="num">
                        {limit.blocked > 0 ? (
                          <span className="badge badge--warn">{formatNumber(limit.blocked)}</span>
                        ) : (
                          <span className="muted">0</span>
                        )}
                      </td>
                      <td className="num">{formatNumber(limit.trackedKeys)}</td>
                      <td className="num">
                        {blockedRatio(limit) === null ? (
                          <span className="muted">—</span>
                        ) : (
                          <span className="num">{`${(blockedRatio(limit) ?? 0).toFixed(1)}%`}</span>
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* 錯誤訊息也放在底部：頁面很長，只在頂端顯示一個紅字會讓「資料是舊的」
            這件事被忽略。 */}
        {error && data ? <FormStatus message={error} tone="error" /> : null}
      </div>
    </AdminShell>
  );
}

/* ==========================================================================
   子元件
   ========================================================================== */

interface StatCardProps {
  label: string;
  value: string;
  note?: string | undefined;
  tone?: 'warn' | 'ok' | undefined;
}

/**
 * 統計卡。
 *
 * tone 只在「值得注意」時才給：正常狀態不給綠色，因為一整排綠色等於沒有強調，
 * 真正要看的（5xx 出現、限流有阻擋）才會跳出來。
 */
function StatCard({ label, value, note, tone }: StatCardProps) {
  return (
    <div className={`stat${tone ? ` stat--${tone}` : ''}`}>
      <span className="stat__label">{label}</span>
      <span className="stat__value">{value}</span>
      {note ? <span className="stat__note">{note}</span> : null}
    </div>
  );
}

interface DependencyCardProps {
  name: string;
  status: MonitorDependency | undefined;
  lines: (string | null)[];
}

const STATE_BADGE: Record<string, string> = {
  ok: 'badge--ok',
  down: 'badge--danger',
  disabled: 'badge--info',
};

function stateLabel(state: string | undefined): string {
  if (state === 'ok') return t('monitor.stateOk');
  if (state === 'down') return t('monitor.stateDown');
  if (state === 'disabled') return t('monitor.stateDisabled');
  return t('common.placeholder');
}

/**
 * 單一依賴的卡片。
 *
 * error 直接顯示原始訊息而不是翻譯：那是後端回來的診斷文字，翻譯它只會讓
 * 管理員看不懂真正的原因（與 core.ts 的 errorText 是同一個理由）。
 */
function DependencyCard({ name, status, lines }: DependencyCardProps) {
  const state = status?.state;
  const detail = lines.filter((line): line is string => line !== null && line !== '');

  return (
    <div className="dep">
      <div className="dep__head">
        <strong className="dep__name">{name}</strong>
        <span className={`badge ${STATE_BADGE[state ?? ''] ?? 'badge--info'}`}>{stateLabel(state)}</span>
      </div>
      {status?.latencyMs ? (
        <p className="dep__line">{t('monitor.depLatency', { ms: formatNumber(Math.round(status.latencyMs)) })}</p>
      ) : null}
      {detail.map((line) => (
        <p className="dep__line" key={line}>
          {line}
        </p>
      ))}
      {state === 'disabled' && name === t('monitor.depRedis') ? (
        <p className="dep__line">{t('monitor.depDisabled')}</p>
      ) : null}
      {status?.error ? <p className="dep__error">{status.error}</p> : null}
    </div>
  );
}

interface TimelineProps {
  points: MonitorResponse['stats']['timeline'];
  loading: boolean;
}

/**
 * 分鐘級流量圖。
 *
 * 用 flexbox 的長條而不是 SVG 或 canvas。三個理由：
 *
 *  1. 144 根柱子的 <div> 對瀏覽器來說只是 144 個盒子，規模小到不需要
 *     虛擬化，而且每一根都帶 <span class="sr-only"> 讓讀屏器能逐格念出
 *     「幾點、幾次、多慢」—— 一張對讀屏器而言完全空白的圖等於沒有圖。
 *  2. 不能用 inline style 設定高度。CSP 的 style-src 沒有 'unsafe-inline'
 *     （見 backend/forum/httpapi/securityheaders.go），而 style 屬性同樣受
 *     style-src 管轄。因此高度改由八個 class（--1 到 --8）在頁面的 <style>
 *     區塊裡定義 —— 那個區塊由後端以 SHA-256 授權，所以是合法的。
 *  3. 不引入圖表函式庫。為了兩種顏色的長條圖引入 d3 或 chart.js，與這個
 *     專案「後端只用標準函式庫」的取向不符。
 *
 * 八階是刻意的取捨：真實流量的分鐘之間差異通常在一個數量級以內，八階對
 * 「有沒有流量波動」這個判讀目標足夠，而階梯越少，柱子之間的高度差就越不
 * 會被誤讀成精確數值 —— 這與頁面上其他數字「照實標示為分位數上界」是同一
 * 個原則。滿刻度用 95 百分位而非最大值，理由見 referenceVolume。
 */
function Timeline({ points, loading }: TimelineProps) {
  // 以 95 百分位而不是最大值當滿刻度的參考點。
  //
  // 這一條是整張圖最關鍵的一個決定。用 max 的話，一次爬蟲掃過或一個重啟就會
  // 把滿刻度拉到平常流量的十倍，於是平常的每分鐘請求全部壓在最低那一階，整
  // 張圖看起來像一片平原 —— 而「平原上有沒有波動」正是這張圖要回答的問題。
  // 用 p95 的話，刻度落在「通常的樣子」，超出 p95 的少數分鐘頂到天花板並另外
  // 標成 is-over，讓「這裡有一個遠超平常的分鐘」變成圖上看得見的事實。
  //
  // 資料點少於 20 個時 p95 幾乎等於 max，此時退回用 max —— 樣本太少時談
  // 「百分位」是沒有意義的。
  const reference = useMemo(() => referenceVolume(points.map((point) => point.total)), [points]);

  if (points.length === 0) {
    return <div className="monitor-timeline monitor-timeline--empty" aria-busy={loading} />;
  }

  return (
    <div className="monitor-timeline" role="img" aria-label={t('monitor.timelineTitle', { minutes: points.length })}>
      {points.map((point) => {
        const level = barLevel(point.total, reference);
        const over = point.total > reference;
        // 只有 5xx 才把整格標成紅色。
        //
        // 4xx 刻意不算：公開入口每天都在被爬蟲打，404 與 403 是背景雜訊，
        // 把「這一格有 4xx」畫成紅色會讓圖上大部分分鐘都是紅的，於是真正
        // 需要看的 5xx 反而看不見 —— 一個永遠在告警的圖等於沒有告警。4xx
        // 並沒有被丟掉：表格的「錯誤」欄與提示文字裡都還在。
        const broken = point.serverErrors > 0;
        return (
          <div
            className={[
              'monitor-bar',
              broken ? 'has-error' : '',
              point.source === 'history' ? 'is-history' : '',
              over ? 'is-over' : '',
            ]
              .filter(Boolean)
              .join(' ')}
            // key 用分鐘字串：同一分鐘的資料會隨重讀更新，key 必須是能辨識
            // 「同一根柱子」的值。用 index 會讓每一輪輪詢都重建整個 DOM。
            key={point.minute}
          >
            <span className={`monitor-bar__fill monitor-bar__fill--${level}`} />
            <span className="sr-only">
              {formatClock(point.minute)} · {formatNumber(point.total)} · {formatMs(point.avgDurationMs)}
            </span>
            <span className="monitor-bar__tip" aria-hidden="true">
              {formatClock(point.minute)} · {t('monitor.timelinePeak', { count: formatNumber(point.total) })}
            </span>
          </div>
        );
      })}
    </div>
  );
}

/** 取總量的 95 百分位作為滿刻度參考。樣本不足時退回最大值。 */
function referenceVolume(values: number[]): number {
  const positive = values.filter((value) => value > 0).sort((a, b) => a - b);
  if (positive.length === 0) return 0;
  if (positive.length < 20) return positive[positive.length - 1] ?? 0;
  const index = Math.min(positive.length - 1, Math.ceil(positive.length * 0.95) - 1);
  return positive[index] ?? 0;
}

/**
 * 把請求量對映到 1..8 的階梯。
 *
 * 線性而非開根號：有了 p95 當參考點，參考值以下的值域本來就被分配得滿滿的
 * （0.5×reference → 第 4 階），開根號反而會把它們全部壓到前三階。
 *
 * reference 為 0（完全沒有流量）時全部回 1，讓圖上出現一條基線而不是空白 ——
 * 全空很容易被當成「圖還沒載入」。
 */
function barLevel(value: number, reference: number): number {
  if (value <= 0 || reference <= 0) return 1;
  return Math.min(8, Math.max(1, Math.ceil((value / reference) * 8)));
}

/* ==========================================================================
   限流器列
   ========================================================================== */

/** 限流器名稱 → 顯示用標籤。未知名稱原樣輸出（後端只會回這三個）。 */
function limitName(name: string): string {
  if (name === 'content') return t('monitor.limitContent');
  if (name === 'upload') return t('monitor.limitUpload');
  if (name === 'auth') return t('monitor.limitAuth');
  return name;
}

function limitBudget(limit: MonitorResponse['stats']['rateLimits'][number]): string {
  if (limit.limit <= 0 || limit.windowSeconds <= 0) return t('monitor.limitUnknown');
  return t('monitor.limitBudget', {
    limit: formatNumber(limit.limit),
    window: formatNumber(Math.round(limit.windowSeconds)),
  });
}

/** 阻擋率（百分比）。分母為 0 時回 null，呼叫端顯示破折號而不是 0%。 */
function blockedRatio(limit: MonitorResponse['stats']['rateLimits'][number]): number | null {
  const total = limit.allowed + limit.blocked;
  if (total <= 0) return null;
  return (limit.blocked / total) * 100;
}
