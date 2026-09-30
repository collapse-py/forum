/*
 * 檢舉（src/forum/useReport.ts）
 *
 * 檢舉目標（文章或留言）與理由輸入的狀態。
 *
 * 舊版在這裡用 window.prompt 收理由。改成 React 後這裡仍然不必是 prompt：
 * 檢舉表單沿用同一頁既有的 .comment-form 版面（forum.html 的 <style> 已經有這組
 * 樣式），就地展開在目標下方，不需要為公開頁另外引入後臺的 <dialog> 樣板。
 * 順帶解決 prompt 沒有的兩件事：理由有 500 字上限的實檔限制，
 * 空白理由不會送出。
 */

import { useCallback, useRef, useState } from 'react';

import { LoginRequiredError, errorText, goToLogin, requestJSON } from '../core';
import { msg, tr, type Message } from '../i18n';

export interface ReportTarget {
  kind: 'post' | 'comment';
  postId: number;
  /** 檢舉留言時是留言自己的 id；檢舉文章時為 0。 */
  commentId: number;
}

export interface ReportController {
  target: ReportTarget | null;
  reason: string;
  sending: boolean;
  /** 判斷某個目標是否正在被檢舉 —— PostCard 據此決定要不要顯示輸入框。 */
  isTarget: (target: ReportTarget) => boolean;
  start: (target: ReportTarget) => void;
  setReason: (value: string) => void;
  cancel: () => void;
  submit: () => void;
}

export function useReport({ onSent, onError }: { onSent: (message: Message | string) => void; onError: (message: Message | string) => void }): ReportController {
  const [target, setTarget] = useState<ReportTarget | null>(null);
  const [reason, setReason] = useState('');
  const [sending, setSending] = useState(false);
  const inFlightRef = useRef(false);

  const start = useCallback((next: ReportTarget) => {
    setTarget(next);
    setReason('');
  }, []);

  const cancel = useCallback(() => {
    setTarget(null);
    setReason('');
  }, []);

  const isTarget = useCallback(
    (candidate: ReportTarget) =>
      target !== null &&
      target.kind === candidate.kind &&
      target.postId === candidate.postId &&
      target.commentId === candidate.commentId,
    [target],
  );

  const submit = useCallback(() => {
    if (!target || inFlightRef.current) return;
    const trimmed = reason.trim();
    if (!trimmed) return;

    inFlightRef.current = true;
    setSending(true);
    const path =
      target.kind === 'comment'
        ? `/api/forum/posts/${target.postId}/comments/report`
        : `/api/forum/posts/${target.postId}/report`;

    void (async () => {
      try {
        await requestJSON(path, { method: 'POST', json: { reason: trimmed }, fallback: tr(msg('error.fallbackReport')) });
        cancel();
        onSent(msg('report.sent'));
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          setSending(false);
          inFlightRef.current = false;
          goToLogin();
          return;
        }
        onError(errorText(error, msg('report.failed')));
        // 失敗時維持輸入框與內容，讓使用者改完再送一次。
      } finally {
        inFlightRef.current = false;
        setSending(false);
      }
    })();
  }, [cancel, onError, onSent, reason, target]);

  return { target, reason, sending, isTarget, start, setReason, cancel, submit };
}
