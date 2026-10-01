/*
 * 站內公告橫幅（src/forum/AnnouncementBanner.tsx）
 *
 * 顯示全站唯一生效中的公告。沒有公告時回 null（不渲染任何 DOM），而那是
 * 壓倒性的常見情況 —— 因此這個元件的「常態」是它沒有輸出。
 *
 * 什麼翻譯、什麼不翻譯
 *
 *   不翻譯 —— 公告的內文。它是管理員寫的純文字；翻譯它需要一套翻譯資料庫，
 *             而本站的多語系是純介面層的（見 src/i18n/index.tsx）。所以內文
 *             以原文顯示，換行以 \n 保留（本站沒有富文字編輯器，textarea
 *             送出的換行就是唯一的排版資訊）。
 *
 *   翻譯   —— 圍著它的每一個字：標題、關閉鈕的 aria-label、發佈時間的格式。
 *             這也是「橫幅文字要走 i18n 而非硬編碼」的實際含意：把標題寫死
 *             成「公告」會讓英文介面裡出現兩個中文字。
 *
 * 關閉（dismiss）的行為
 *
 * 關閉只影響「這個瀏覽器、這個 session」，不影響其他人 —— 公告仍然對每一個
 * 訪客顯示。關閉狀態記在 sessionStorage 而非 localStorage：
 *  · localStorage 會讓「我關掉了」永久生效，之後連新的公告都看不見，而
 *    使用者沒有任何地方可以把它找回來。
 *  · sessionStorage 在分頁關閉後消失，因此回到首頁時公告會再次出現 ——
 *    對一個站方公告來說這是正確的預設（它預設該被看到）。
 *  記錄裡含公告的 id，因此更新同一則公告（body 或到期時間變了）會讓它
 *  重新出現 —— 那正是使用者想看到的：內容變了，等於是新的資訊。
 */

import { useEffect, useState } from 'react';

import { requestJSON } from '../core';
import { t } from '../i18n';
import type { ForumAnnouncement, ForumAnnouncementResponse } from '../types';

const DISMISS_STORAGE_KEY = 'forum:announcement-dismissed';

function readDismissed(): string {
  try {
    return window.sessionStorage.getItem(DISMISS_STORAGE_KEY) ?? '';
  } catch {
    // 隱私模式或使用者明確停用了儲存空間。回空字串代表「沒關過」，
    // 那是安全的預設：橫幅會一直顯示，而它本來就該被看到。
    return '';
  }
}

function writeDismissed(id: string): void {
  try {
    window.sessionStorage.setItem(DISMISS_STORAGE_KEY, id);
  } catch {
    // 寫不進去就只是「這次不記得關過」，下次載入會再顯示一次。
    // 不值得為了讓它記得起而把整個橫幅也一併跳過 —— 那會讓「儲存空間不可用
    // 的使用者永遠看不到公告」。
  }
}

export function AnnouncementBanner() {
  const [announcement, setAnnouncement] = useState<ForumAnnouncement | null>(null);
  const [dismissed, setDismissed] = useState(() => readDismissed());

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const response = await requestJSON<ForumAnnouncementResponse>('/api/forum/announcement');
        if (!cancelled) setAnnouncement(response.announcement ?? null);
      } catch {
        // 公告讀不到不是使用者該知道的問題：它是「站方可能忘了公告」而不是
        // 「你看不到內容」。靜默忽略讓橫幅不出現，而頁面照常運作。
        if (!cancelled) setAnnouncement(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  if (announcement === null || dismissed === String(announcement.publishedAt)) return null;

  return (
    // role="status" 而非 role="alert"：這不是一個緊急訊息，而是一段陳述。
    // alert 會讓螢幕閱讀器打斷使用者正在讀的內容。
    <aside className="announce" role="status" aria-label={t('announce.label')}>
      <div className="announce__body">
        <p className="announce__label">
          <span className="announce__tag">{t('announce.publicNote')}</span>
        </p>
        {/*
          不用 white-space: pre-line 之外的任何方式處理換行：後端送來的是
          純文字加 \n，而 pre-line 正好是「保留換行但不保留連續空行」——
          管理員在 textarea 裡多按幾次 Enter 不該在橫幅上留下空行。
        */}
        <p className="announce__text">{announcement.body}</p>
        <p className="announce__meta">
          {t('announce.publishedOn', { date: announcement.publishedAt })}
          {' · '}
          {announcement.expiresAt
            ? t('announce.expiresOn', { date: announcement.expiresAt })
            : t('announce.neverExpires')}
        </p>
      </div>
      <button
        className="announce__close"
        type="button"
        aria-label={t('announce.closeAria')}
        onClick={() => {
          writeDismissed(announcement.publishedAt);
          setDismissed(announcement.publishedAt);
        }}
      >
        ×
      </button>
    </aside>
  );
}
