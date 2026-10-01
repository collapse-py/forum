/*
 * 內容趨勢統計頁（/admin/stats）
 *
 * 職責：把 /api/admin/stats 的日別序列與三個排行榜畫成看得懂的樣子。
 *
 * 這個頁面與另外五個後臺頁的定位差異，是它所有設計決定的來源：
 *
 *   監控頁（monitor）回答「現在正在怎樣」—— 它會自己動，每十秒更新。
 *   稽核頁（log）回答「誰做過什麼」—— 它是一份依時間倒序的清單。
 *   這一頁回答「變成這樣多久了」—— 它是一張圖加三份排名。
 *
 * 三個刻意的取捨：
 *
 *  1. 三條序列分開顯示，不疊在一起。各自的量級差一個數量級（新使用者通常
 *     遠少於新留言），疊在一張圖上只有最小的那條還看得見。代價是無法直接
 *     比較「哪一條在上升」—— 換來的是三條都看得見。
 *
 *  2. 缺口補成 0，並且畫成貼地的線而不是斷開。後端刻意把沒有資料的日子也
 *     放進陣列（見 types.ts 的說明）；若這裡把它當成「沒有資料」而斷開線段，
 *     使用者會以為是統計漏了，而不是那天真的沒有活動。
 *
 *  3. 視窗切換用按鈕列而不是 select。可選值只有五個，而「換區間」是這頁
 *     最常做的操作 —— select 要兩次點擊，按鈕列只要一次。請求過大時後端會
 *     收斂到 90 天，介面另外明寫出來（否則使用者會以為自己看到的就是他選的
 *     90 天，而那其實是他選 365 天被收斂的結果）。
 */

import { useCallback, useEffect, useState } from 'react';

import { adminApi } from '../api/admin';
import { errorMessage, formatDateTime, formatNumber, text } from '../core';
import { t, usePageTitle } from '../i18n';
import type { ContentStatsResponse } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import { BusyButton, EmptyState, FormStatus } from './ui';

/** 可選的視窗長度（天）。最後一個必須等於後端的 maxStatsDays。 */
const WINDOW_OPTIONS = [7, 14, 30, 60, 90] as const;

/** 預設視窗。30 天是「看得出趨勢」與「一季以內」的中點。 */
const DEFAULT_WINDOW = 30;

/** 圖的 viewBox 寬高。用固定座標系 + preserveAspectRatio="none"，讓圖能
 *  撐滿容器寬度；高度則由 CSS 固定，因此窄螢幕上圖會變矮而不會變形。
 *  刻度文字用 userSpaceOnUse（預設）而不是縮放座標，因此在 non-scaling 的
 *  viewBox 下字級仍然由 CSS 的 font-size 決定，不會被拉成巨大或扁掉。 */
const CHART_WIDTH = 900;
const CHART_HEIGHT = 200;
const CHART_PADDING = { top: 14, right: 46, bottom: 22, left: 34 };

export function ContentStatsPage() {
  usePageTitle('title.adminStats');

  const { toast } = useAdmin();
  const [data, setData] = useState<ContentStatsResponse | null>(null);
  const [days, setDays] = useState<number>(DEFAULT_WINDOW);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (window: number) => {
      setLoading(true);
      try {
        const response = await adminApi<ContentStatsResponse>(`/api/admin/stats?days=${window}`);
        setData(response);
        setError(null);
      } catch (thrown) {
        const message = errorMessage(thrown, t('stats.loadFailed'));
        setError(message);
        toast(message, 'error');
      } finally {
        setLoading(false);
      }
    },
    [toast],
  );

  useEffect(() => {
    void load(days);
  }, [days, load]);

  const series = data?.series ?? null;
  const totals = data?.totals ?? null;
  const topPosts = data?.topPosts ?? [];
  const topTags = data?.topTags ?? [];
  const topAuthors = data?.topAuthors ?? [];
  // 「這個視窗完全沒有任何活動」與「有圖但都是零」在畫面上必須能區分：
  // 前者顯示空狀態，後者顯示貼地的三條線。否則使用者會以為圖壞了。
  const isEmpty = series !== null && series.dates.length === 0;

  return (
    <AdminShell active="/admin/stats" pageTitle={t('stats.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('stats.eyebrow')}</p>
            <h1>{t('stats.title')}</h1>
            <p className="page-head__copy">{t('stats.copy')}</p>
          </div>
          <div className="page-head__actions">
            <BusyButton busy={loading} className="btn btn--sm" onClick={() => void load(days)}>
              {t('stats.refresh')}
            </BusyButton>
          </div>
        </header>

        {error ? <FormStatus message={error} tone="error" /> : null}

        {/* --- 每日新增 ---------------------------------------------------- */}
        <section className="panel" aria-labelledby="stats-series-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="stats-series-title">
                {t('stats.seriesTitle')}
              </h2>
              <p className="panel__note">{t('stats.seriesNote')}</p>
            </div>
          </div>
          <div className="panel__body">
            {series && series.dates.length > 0 && !isEmpty ? (
              <>
                <SeriesChart dates={series.dates} users={series.users} posts={series.posts} comments={series.comments} />
                <ul className="stats-legend">
                  <li className="stats-legend__item">
                    <i className="stats-legend__swatch stats-legend__swatch--users" aria-hidden="true" />
                    {t('stats.seriesUsers')}
                  </li>
                  <li className="stats-legend__item">
                    <i className="stats-legend__swatch stats-legend__swatch--posts" aria-hidden="true" />
                    {t('stats.seriesPosts')}
                  </li>
                  <li className="stats-legend__item">
                    <i className="stats-legend__swatch stats-legend__swatch--comments" aria-hidden="true" />
                    {t('stats.seriesComments')}
                  </li>
                </ul>
              </>
            ) : (
              <div className="stats-chart__empty" aria-busy={loading}>
                {loading ? t('common.loading') : t('stats.seriesEmpty')}
              </div>
            )}
          </div>
          {/* 視窗控制與時區說明都在 panel__foot：它們是「這張圖怎麼讀」的
              附註，不是圖的一部分。 */}
          <div className="panel__foot">
            <p className="stats-window">
              <span>{t('stats.window')}</span>
              <span className="stats-window__options" role="group" aria-label={t('stats.window')}>
                {WINDOW_OPTIONS.map((option) => (
                  <button
                    key={option}
                    className={`stats-window__option${option === days ? ' is-active' : ''}`}
                    aria-pressed={option === days}
                    onClick={() => setDays(option)}
                  >
                    {t('stats.windowDays', { days: option })}
                  </button>
                ))}
              </span>
              {days > 90 ? <span>{t('stats.windowClamped')}</span> : null}
              <span className="muted">{t('stats.windowNote')}</span>
            </p>
            {data?.now ? (
              <p className="monitor-hint">{t('stats.generatedAt', { time: formatDateTime(data.now) })}</p>
            ) : null}
          </div>
        </section>

        {/* --- 視窗內合計 --------------------------------------------------
         * 放在圖的正下方而不是最上面：它的用途是「給這條曲線一個數字」，
         * 因此必須緊接著曲線，而不是讓使用者先讀完三個排行榜再回頭找。
         * ------------------------------------------------------------------ */}
        <section className="panel" aria-labelledby="stats-totals-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="stats-totals-title">
                {t('stats.totalsTitle')}
              </h2>
              <p className="panel__note">{t('stats.totalsNote')}</p>
            </div>
          </div>
          <div className="panel__body">
            <div className="stats">
              <StatCard label={t('stats.totalUsers')} value={formatNumber(totals?.users ?? 0)} />
              <StatCard label={t('stats.totalPosts')} value={formatNumber(totals?.posts ?? 0)} />
              <StatCard label={t('stats.totalComments')} value={formatNumber(totals?.comments ?? 0)} />
              <StatCard label={t('stats.totalLikes')} value={formatNumber(totals?.likes ?? 0)} />
            </div>
          </div>
        </section>

        {/* --- 三個排行榜 -------------------------------------------------- */}
        <div className="stats-boards">
          <section className="panel stats-board" aria-labelledby="stats-top-posts-title">
            <div className="panel__head">
              <div className="panel__titles">
                <h2 className="panel__title" id="stats-top-posts-title">
                  {t('stats.topPostsTitle')}
                </h2>
                <p className="panel__note">{t('stats.topPostsNote')}</p>
              </div>
            </div>
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th scope="col" className="num">
                      #
                    </th>
                    <th scope="col">{t('stats.colExcerpt')}</th>
                    <th scope="col" className="num">
                      {t('stats.colEngagement')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {topPosts.length === 0 ? (
                    <tr>
                      <td colSpan={3}>
                        <EmptyState title={t('stats.empty')}>{t('stats.emptyBody')}</EmptyState>
                      </td>
                    </tr>
                  ) : (
                    topPosts.map((post, index) => (
                      <tr key={post.id}>
                        <td className="num">
                          <Rank value={index + 1} />
                        </td>
                        <td>
                          <span className="stats-excerpt">{post.excerpt}</span>
                          <span className="cell-sub u-nowrap">{post.authorEmail}</span>
                        </td>
                        <td className="num">
                          {formatNumber(post.comments + post.likes)}
                          <span className="cell-sub">
                            {t('stats.engagement', {
                              comments: formatNumber(post.comments),
                              likes: formatNumber(post.likes),
                            })}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </section>

          <section className="panel stats-board" aria-labelledby="stats-top-tags-title">
            <div className="panel__head">
              <div className="panel__titles">
                <h2 className="panel__title" id="stats-top-tags-title">
                  {t('stats.topTagsTitle')}
                </h2>
                <p className="panel__note">{t('stats.topTagsNote')}</p>
              </div>
            </div>
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th scope="col" className="num">
                      #
                    </th>
                    <th scope="col">{t('log.colTarget')}</th>
                    <th scope="col" className="num">
                      {t('stats.colUsers')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {topTags.length === 0 ? (
                    <tr>
                      <td colSpan={3}>
                        <EmptyState title={t('stats.empty')}>{t('stats.emptyBody')}</EmptyState>
                      </td>
                    </tr>
                  ) : (
                    topTags.map((tag, index) => (
                      <tr key={tag.id}>
                        <td className="num">
                          <Rank value={index + 1} />
                        </td>
                        <td>
                          <span className="cell-primary">{tag.name}</span>
                        </td>
                        <td className="num">{formatNumber(tag.users)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </section>

          <section className="panel stats-board" aria-labelledby="stats-top-authors-title">
            <div className="panel__head">
              <div className="panel__titles">
                <h2 className="panel__title" id="stats-top-authors-title">
                  {t('stats.topAuthorsTitle')}
                </h2>
                <p className="panel__note">{t('stats.topAuthorsNote')}</p>
              </div>
            </div>
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th scope="col" className="num">
                      #
                    </th>
                    <th scope="col">{t('stats.colAuthor')}</th>
                    <th scope="col" className="num">
                      {t('stats.colPosts')}
                    </th>
                    <th scope="col" className="num">
                      {t('stats.colComments')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {topAuthors.length === 0 ? (
                    <tr>
                      <td colSpan={4}>
                        <EmptyState title={t('stats.empty')}>{t('stats.emptyBody')}</EmptyState>
                      </td>
                    </tr>
                  ) : (
                    topAuthors.map((author, index) => (
                      <tr key={author.email}>
                        <td className="num">
                          <Rank value={index + 1} />
                        </td>
                        <td>
                          <span className="cell-primary">{author.email}</span>
                        </td>
                        <td className="num">{formatNumber(author.posts)}</td>
                        <td className="num">{formatNumber(author.comments)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </section>
        </div>
      </div>
    </AdminShell>
  );
}

/* ==========================================================================
   子元件
   ========================================================================== */

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="stat">
      <span className="stat__label">{label}</span>
      <span className="stat__value">{value}</span>
    </div>
  );
}

/** 名次徽章。前三名用強調色 —— 排行榜的用途是找出值得看的那幾個。 */
function Rank({ value }: { value: number }) {
  return <span className={`stats-rank${value <= 3 ? ' stats-rank--top' : ''}`}>{value}</span>;
}

interface SeriesChartProps {
  dates: string[];
  users: number[];
  posts: number[];
  comments: number[];
}

/**
 * 三條折線的 SVG 圖。
 *
 * 三條線**共用同一個縱軸刻度**（取三者最大值），不是各自正規化。
 *
 * 這個決定是這一頁最重要的一個，而且很容易做錯。各自正規化看起來比較
 * 「好看」—— 三條線都會有明顯的起伏 —— 但那會讓讀者誤以為三者的量級相當。
 * 實情是新使用者通常比新留言少一到兩個數量級，各自正規化會讓那條幾乎沒有
 * 事件的線看起來最活躍。共用量級之後，新使用者的線會貼著底部，而那正是
 * 事實；它的起伏仍然看得見（因為線寬與顏色不變），只是不會被放大到與
 * 「每天 150 則留言」同一個高度。
 *
 * 刻度文字用 userSpaceOnUse（SVG 的預設）而不是 non-scaling：在
 * preserveAspectRatio="none" 之下，non-scaling 的文字會被水平拉伸，
 * 變成一堆看不出是幾的線條。因此只有線與格線用 non-scaling-stroke
 * （讓線寬不隨 viewBox 縮放而變粗或變細），文字維持縮放 ——
 * 這個組合是「座標系統縮放、視覺樣式不縮放」的分界。
 */
function SeriesChart({ dates, users, posts, comments }: SeriesChartProps) {
  const plotWidth = CHART_WIDTH - CHART_PADDING.left - CHART_PADDING.right;
  const plotHeight = CHART_HEIGHT - CHART_PADDING.top - CHART_PADDING.bottom;

  // 每一格的水平間距。三條線共用同一組 x 座標，因此只需要算一次 ——
  // 讓 scaleFor 與 pathFor 各自算一次會讓兩者有機會在「只有一天」時算出
  // 不同的 0 與除零結果。
  const step = dates.length > 1 ? plotWidth / (dates.length - 1) : 0;

  // 三條線共用一個最大值，因此「線的高度」在三條線之間是可比的。
  const peak = Math.max(1, ...users, ...posts, ...comments);
  const y = (value: number) => CHART_PADDING.top + plotHeight - (value / peak) * plotHeight;

  const pathFor = (values: number[]) => {
    if (values.length === 0) return '';
    return values
      .map((value, index) => `${index === 0 ? 'M' : 'L'}${(CHART_PADDING.left + index * step).toFixed(1)},${y(value).toFixed(1)}`)
      .join(' ');
  };

  // 刻度：只標起始與結束兩個日期，以及縱軸的最大值。標每一個 x 刻度在
  // 90 天時會讓軸上全是重疊的文字。
  const endX = CHART_PADDING.left + step * (dates.length - 1);

  return (
    <svg
      className="stats-chart"
      viewBox={`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`}
      preserveAspectRatio="none"
      role="img"
      aria-label={t('stats.seriesTitle')}
    >
      {/* 基線與兩條中間格線。刻意只有三條：更多格線在這個高度下會變成雜訊。 */}
      <line
        className="stats-chart__grid"
        x1={CHART_PADDING.left}
        y1={CHART_PADDING.top + plotHeight}
        x2={CHART_PADDING.left + plotWidth}
        y2={CHART_PADDING.top + plotHeight}
      />
      <line
        className="stats-chart__grid"
        x1={CHART_PADDING.left}
        y1={CHART_PADDING.top + plotHeight / 2}
        x2={CHART_PADDING.left + plotWidth}
        y2={CHART_PADDING.top + plotHeight / 2}
      />

      {/* 縱軸刻度只在頂端標一次最大值。這是共用量級的關鍵：沒有它，
          讀者無法知道「那條貼地的線」是真的接近零，還是圖被拉大了。 */}
      <text
        className="stats-chart__axis"
        x={CHART_PADDING.left - 6}
        y={CHART_PADDING.top + 4}
        textAnchor="end"
      >
        {formatNumber(peak)}
      </text>

      <text className="stats-chart__axis" x={CHART_PADDING.left} y={CHART_HEIGHT - 6} textAnchor="start">
        {formatAxisDate(dates[0] ?? '')}
      </text>
      <text
        className="stats-chart__axis"
        x={CHART_PADDING.left + plotWidth}
        y={CHART_HEIGHT - 6}
        textAnchor="end"
      >
        {formatAxisDate(dates.at(-1) ?? '')}
      </text>

      {/* 峰值標籤放在圖的右側留白區。共用量級之後三個峰值的高度各不相同，
          因此不會互相重疊 —— 這是「各自正規化」另一個看得見的缺陷（那時
          三個峰值都會落在頂端而疊成一團）。 */}
      <PeakLabel values={users} y={y} />
      <PeakLabel values={posts} y={y} />
      <PeakLabel values={comments} y={y} />

      <path className="stats-chart__line stats-chart__line--comments" d={pathFor(comments)} />
      <path className="stats-chart__line stats-chart__line--posts" d={pathFor(posts)} />
      <path className="stats-chart__line stats-chart__line--users" d={pathFor(users)} />

      {/* 起點與終點標記：一條線在圖上只有「今天的值」是不夠的，使用者需要
          兩個對應到 x 軸的錨點才知道起訖各是哪一天。
          series 的四個陣列由後端保證等長（見 types.ts），但 ?? 0 仍然留著：
          圖寬與高度在資料到達之前就已經算好了，那一格若渲染成 NaN，整條
          <path> 的 d 屬性會變成「NaN」而讓整個 SVG 消失。 */}
      <circle cx={CHART_PADDING.left} cy={y(users.at(0) ?? 0)} r="3" fill="var(--accent)" />
      <circle cx={endX} cy={y(users.at(-1) ?? 0)} r="3" fill="var(--accent)" />
    </svg>
  );
}

/**
 * 在圖的右側留白區標出這條線的峰值。
 *
 * 追蹤「最大值」而不是「最大值的索引」：這個元件只需要那個數字，而取值
 * 需要一次索引；追蹤索引會讓 TypeScript 在 noUncheckedIndexedAccess 下
 * 一直提醒索引可能不存在，而那個可能性在這裡並不存在（reduce 從 0 起算）。
 */
function PeakLabel({ values, y }: { values: number[]; y: (value: number) => number }) {
  const peak = values.reduce((max, value) => (value > max ? value : max), 0);
  return (
    <text
      className="stats-chart__axis"
      x={CHART_WIDTH - CHART_PADDING.right + 6}
      y={y(peak) + 3}
      textAnchor="start"
    >
      {formatNumber(peak)}
    </text>
  );
}

/**
 * x 軸刻度的日期格式化。
 *
 * 只取月與日（「9/30」），不取年份：視窗最長 90 天，跨年時使用者看的是
 * 「年底那一段」，年份在這裡沒有判讀價值。刻意不用 Intl：它會依照語言
 * 產生不同的格式，而軸標籤的位置與長度是固定的，手寫的短格式在所有語系
 * 長度都差不多。
 */
function formatAxisDate(value: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if (!match) return text(value);
  return `${Number(match[2])}/${Number(match[3])}`;
}
