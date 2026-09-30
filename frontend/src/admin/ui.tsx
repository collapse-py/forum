/*
 * 後臺共用的小元件（src/admin/ui.tsx）
 *
 * 三個頁面重複出現的東西：表格的四種狀態（載入中 / 有資料 / 沒資料 / 載入失敗）、
 * 忙碌中的按鈕、表單狀態列。這些在舊版是 admin-core.ts 的 createList() 與
 * setBusy() 兩個函式；React 版把它們變成看得見的元件，樣板的三種措辭
 * 不可能再各寫各的。
 */

import type { ReactNode } from 'react';

import { Icon } from '../icons';
import { t } from '../i18n';

/* ==========================================================================
   表格狀態
   ========================================================================== */

/**
 * 列表的四種狀態。
 *
 * 舊版用三個函式（start / render / fail）表達同一件事，呼叫端要自己記住
 * 現在是第幾種。收斂成一個型別後，「忘了切回 ready」會變成編譯期錯誤。
 */
export type ListPhase = 'loading' | 'ready' | 'error';

export interface ListState<T> {
  phase: ListPhase;
  items: T[];
  /** phase 為 error 時要顯示的訊息。 */
  message: string;
}

export function loadingList<T>(): ListState<T> {
  return { phase: 'loading', items: [], message: '' };
}

export function readyList<T>(items: T[]): ListState<T> {
  return { phase: 'ready', items, message: '' };
}

export function failedList<T>(message: string): ListState<T> {
  return { phase: 'error', items: [], message };
}

/** 載入中的灰色骨架。欄數要給對，否則欄寬會在資料到達時跳動。 */
export function SkeletonRows({ rows = 5, columns = 6 }: { rows?: number; columns?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, rowIndex) => (
        <tr key={rowIndex}>
          {Array.from({ length: columns }, (_, cellIndex) => (
            <td key={cellIndex}>
              <span className="skeleton" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

/** 載入失敗：整列合併成一個帶訊息的狀態格。 */
export function ErrorRow({ columns, message }: { columns: number; message: string }) {
  return (
    <tr>
      <td colSpan={columns}>
        <div className="state">
          <Icon name="flag" />
          <strong>{t('admin.listLoadFailed')}</strong>
          <p>{message}</p>
        </div>
      </td>
    </tr>
  );
}

/**
 * 沒有資料時的狀態區塊。
 *
 * 放在表格外面（不是 <tbody> 內），這是 style.css 的 .state 版面契約，
 * 也讓它在窄螢幕時不會被表格的 min-width 撐開。
 */
export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="state">
      <span className="state__icon">
        <Icon name="inbox" />
      </span>
      <strong>{title}</strong>
      <p>{children}</p>
    </div>
  );
}

/**
 * 依 phase 決定要畫什麼，並順帶回報「該不該顯示空狀態」。
 *
 * 回傳 null 代表現在不是「成功但沒資料」，呼叫端據此決定 EmptyState 的顯示。
 */
export function ListBody<T>({
  state,
  columns,
  skeletonRows,
  children,
}: {
  state: ListState<T>;
  columns: number;
  skeletonRows: number;
  children: (items: T[]) => ReactNode;
}) {
  if (state.phase === 'loading') return <SkeletonRows rows={skeletonRows} columns={columns} />;
  if (state.phase === 'error') return <ErrorRow columns={columns} message={state.message} />;
  return <>{children(state.items)}</>;
}

/** 是否該顯示「沒有資料」區塊。 */
export function isEmpty<T>(state: ListState<T>): boolean {
  return state.phase === 'ready' && state.items.length === 0;
}

/* ==========================================================================
   按鈕
   ========================================================================== */

export interface BusyButtonProps {
  busy: boolean;
  className?: string | undefined;
  children: ReactNode;
  onClick?: (() => void) | undefined;
  title?: string | undefined;
  'aria-label'?: string | undefined;
}

/**
 * 處理中的按鈕。
 *
 * 用 color: transparent + ::after spinner（.btn.is-busy）而不是改文字，目的是
 * 讓按鈕寬度不跳動 —— 舊版在「儲存中…」與原文之間來回撐開版面。
 */
export function BusyButton({ busy, className, children, onClick, title, ...rest }: BusyButtonProps) {
  return (
    <button
      type="button"
      className={className}
      disabled={busy}
      onClick={onClick}
      title={title}
      aria-busy={busy || undefined}
      {...rest}
    >
      {children}
    </button>
  );
}

/* ==========================================================================
   表單狀態列
   ========================================================================== */

export type FormTone = '' | 'error' | 'ok';

export function FormStatus({ message, tone = '' }: { message: string; tone?: FormTone }) {
  return (
    <p className="form-status" role="status" aria-live="polite" data-tone={tone || undefined}>
      {message}
    </p>
  );
}
