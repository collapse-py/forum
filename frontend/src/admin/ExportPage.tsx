/*
 * 匯出與批次操作頁（/admin/export）
 *
 * 職責：提供三份 CSV 的下載入口，並說明批次操作在哪裡進行。
 *
 * 這一頁刻意「很薄」—— 三個下載連結、兩段說明。原因是它實際上只有三件事可做，
 * 而任何一個說明文字比一個按鈕長的情況下，額外的版面都是在稀釋那三個按鈕的
 * 重要性。它存在的價值不是功能，是**說明**：
 *
 *   1. 匯出檔裡的單引號前綴是什麼、為什麼不能移除（export.safetyPrefix）
 *   2. 批次操作在用戶管理頁做，而不在這裡（因為選取狀態只存在那裡）
 *
 * 第二點尤其重要：若這一頁也放批次按鈕（而沒有選取狀態），那個按鈕會永遠是
 * 壞的，而且使用者不會知道為什麼。因此這裡連一個「去用戶管理」的可點按鈕都
 * 不放 —— 只放一句指向該頁的說明，避免出現一個「在這裡按也沒用」的誘餌。
 *
 * 為什麼用 <a href download> 而不是 fetch + blob
 *
 *   1. 匯出是取得一份檔案，不是執行一個動作。fetch + blob 會讓瀏覽器把它
 *      當成「XHR 產生的一個下載」，而真實的 <a download> 是瀏覽器自己的
 *      下載機制 —— 進度列、另存新檔、取消下載都因此是原生的。
 *   2. 三份檔案可能不小（每份上限 5 萬列）。blob 會讓整份檔案先存在
 *      記憶體裡；<a> 是串流，見 backend 的 csv_export.go。
 *   3. 這個方式**不可能忘記帶 session cookie**。fetch 必須顯式設定
 *      credentials，而漏設時症狀是「下載下來是一個 401 的 JSON」——
 *      使用者會以為匯出壞了。瀏覽器的 <a> 一定會帶 cookie。
 *
 *   403（來源檢查）在這個設計下不會發生：匯出是 GET，而 GET 不做來源檢查
 *   （它是唯讀的，跨站請求即使通過也讀不到回應內容 —— 見 server.go 的說明）。
 *   若日後有人把匯出改成 POST，這一段就必須跟著改。
 */

import { useEffect, useState } from 'react';

import { adminApi } from '../api/admin';
import { t, usePageTitle, type MessageKey } from '../i18n';
import type { ExportKind } from '../types';
import { useAdmin } from './provider';
import { AdminShell } from './shell';
import { FormStatus } from './ui';

/**
 * 三份匯出的描述。
 *
 * `columns` 放在前端是為了讓頁面顯示「這份匯出有哪些欄位」，而不必把三個 CSV
 * 標頭各打一次。打出三次就有三次不一致的機會，而那種不一致的症狀是「管理員
 * 拿匯出檔對帳而得到錯誤結論」—— 它不會出現在任何測試失敗裡。
 */
const EXPORTS: { kind: ExportKind; name: MessageKey; note: MessageKey; columns: string }[] = [
  {
    kind: 'users',
    name: 'export.exportUsers',
    note: 'export.exportUsersNote',
    columns: 'email, status, created_at, updated_at, posts, comments',
  },
  {
    kind: 'posts',
    name: 'export.exportPosts',
    note: 'export.exportPostsNote',
    columns: 'id, author_email, content, created_at, comments, likes',
  },
  {
    kind: 'reports',
    name: 'export.exportReports',
    note: 'export.exportReportsNote',
    columns: 'id, target_type, target_id, reporter_email, reason, status, created_at, reviewed_at, reviewed_by',
  },
];

export function ExportPage() {
  usePageTitle('title.adminExport');

  const { toast } = useAdmin();
  /*
   * 後端可達性。
   *
   * 這一頁沒有任何資料要抓，因此沒有 loading 狀態。唯一能讓它變得有用的
   * 狀態是「後端不可達」：沒有它的話，使用者按下三個下載連結都沒有反應，
   * 症狀是「網站壞了」而不是「請稍後再試」。這個檢查用最便宜的方式做 ——
   * 打一支 stats（最輕的後臺端點），結果不影響畫面，只影響那個提示列。
   */
  const [reachable, setReachable] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        // days=7 是最短的視窗，因此這是最便宜的一支。
        await adminApi('/api/admin/stats?days=7');
        if (!cancelled) setReachable(true);
      } catch {
        if (!cancelled) setReachable(false);
      }
    })();
    return () => {
      // 元件卸載後不要 setState：那會在 React 18 之下產生一個無害但會被
      // StrictMode 的雙重呼叫放大的警告。
      cancelled = true;
    };
  }, []);

  /*
   * 下載開始的提示。
   *
   * <a download> 的點擊由瀏覽器處理，不會經過任何非同步流程，因此這個
   * state 記的只是「使用者剛剛觸發了下載」這個事實。它唯一的作用是讓使用
   * 者確認按鈕有反應 —— 沒有它，一個下載失敗的按鈕看起來和一個還在下載的
   * 按鈕一模一樣。
   */
  const announce = () => {
    toast(t('export.downloading'), 'ok');
  };

  return (
    <AdminShell active="/admin/export" pageTitle={t('export.title')}>
      <div className="stack">
        <header className="page-head">
          <div className="page-head__text">
            <p className="eyebrow">{t('export.eyebrow')}</p>
            <h1>{t('export.title')}</h1>
            <p className="page-head__copy">{t('export.copy')}</p>
          </div>
        </header>

        {/* 後端不可達時的提示。它刻意是一行 FormStatus 而不是蓋住整頁的
            modal：使用者仍然可能想讀下面那些說明文字（尤其是安全提示）。 */}
        {reachable === false ? <FormStatus message={t('stats.loadFailed')} tone="error" /> : null}

        <section className="panel" aria-labelledby="export-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="export-title">
                {t('export.exportTitle')}
              </h2>
              <p className="panel__note">{t('export.exportNote')}</p>
            </div>
          </div>
          <div className="panel__body">
            <div className="export-grid">
              {EXPORTS.map((item) => (
                <div className="export-card" key={item.kind}>
                  <span className="export-card__name">{t(item.name)}</span>
                  <p className="export-card__note">{t(item.note)}</p>
                  {/* 欄位清單用等寬字體：那是資料結構而不是文案，而且它必須
                      與實際下載到的 CSV 標頭逐字相符。 */}
                  <p className="export-card__columns">{item.columns}</p>
                  <a
                    className="btn btn--primary btn--sm export-card__action"
                    href={`/api/admin/export/${item.kind}.csv`}
                    download
                    onClick={announce}
                  >
                    {t('export.download')}
                  </a>
                </div>
              ))}
            </div>

            <p className="export-safety">
              <span>{t('export.safety')}</span>
              <span>{t('export.safetyPrefix')}</span>
            </p>
          </div>
        </section>

        <section className="panel" aria-labelledby="export-batch-title">
          <div className="panel__head">
            <div className="panel__titles">
              <h2 className="panel__title" id="export-batch-title">
                {t('export.batchTitle')}
              </h2>
              <p className="panel__note">{t('export.batchNote')}</p>
            </div>
          </div>
          <div className="panel__body">
            <ul className="export-batch-list">
              {/*
                停權與恢復刻意不帶副標：它們在名稱裡已經說明了，而一個
                只寫「停權」的重複標籤看起來像缺了說明。真正需要說明的只有
                標籤，因為它的「覆寫」語意是唯一一個不從名稱就看得出來的部分。
              */}
              <li className="export-batch-item">
                <span className="export-batch-item__name">{t('export.batchSuspend')}</span>
              </li>
              <li className="export-batch-item">
                <span className="export-batch-item__name">{t('export.batchReinstate')}</span>
              </li>
              <li className="export-batch-item">
                <span className="export-batch-item__name">{t('export.batchTags')}</span>
                <p className="export-batch-item__note">{t('export.batchTagsNote')}</p>
              </li>
            </ul>
            <p className="export-batch-note export-batch-note--after">
              {t('export.noSelection')}（<code>/admin</code>）
            </p>
            <p className="export-batch-note">{t('export.batchMax')}</p>
          </div>
        </section>
      </div>
    </AdminShell>
  );
}
