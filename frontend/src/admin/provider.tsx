/*
 * 後臺共用元件的狀態層（src/admin/provider.tsx）
 *
 * 三個後臺頁面（/admin、/admin/forum、/admin/forum-report）共用的三件事
 * 集中在這裡：toast、<dialog> 互動視窗、rail 上的用戶數徽章。
 *
 * 舊版把這三者做成 admin-core.ts 的模組級單例（toastHost() 自己找 #toasts、
 * dialog 只有一個、setNavCount() 直接改 DOM）。改成 React 後這些都變成
 * provider 內的 state：單例的「只有一份」語意由 provider 保證，而 DOM 的
 * 建立與銷毀交給 React，頁面卸載時不會留下孤兒元素。
 *
 * 為什麼不是三個獨立 provider：dialog 與 toast 的生命週期完全一致（都掛在
 * 單一後臺殼層下），拆開只會讓 admin.tsx 多包兩層。
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
  type RefObject,
} from 'react';

import { Icon } from '../icons';
import { t } from '../i18n';

/* ==========================================================================
   Toast
   ========================================================================== */

export type ToastTone = 'info' | 'ok' | 'error';

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
  open: boolean;
}

interface ToastHandle {
  frame: number;
  timer: number;
}

/* ==========================================================================
   Dialog
   ========================================================================== */

export type DialogTone = 'primary' | 'danger' | 'ghost' | 'success';

export interface ConfirmActionOptions {
  title: string;
  message?: string | undefined;
  confirmLabel?: string | undefined;
  tone?: DialogTone | undefined;
}

export interface PromptTextOptions {
  title: string;
  message?: string | undefined;
  label: string;
  value?: string | undefined;
  placeholder?: string | undefined;
  maxLength?: number | undefined;
  rows?: number | undefined;
  confirmLabel?: string | undefined;
}

export interface PickTagOption {
  id: number;
  name: string;
}

export interface PickTagsOptions {
  title: string;
  message?: string | undefined;
  tags: PickTagOption[];
  selectedIds?: number[] | undefined;
}

type DialogRequest =
  | {
      kind: 'confirm';
      title: string;
      message: string;
      confirmLabel: string;
      tone: DialogTone;
    }
  | {
      kind: 'prompt';
      title: string;
      message: string;
      confirmLabel: string;
      label: string;
      placeholder: string;
      maxLength: number;
      rows: number;
    }
  | { kind: 'tags'; title: string; message: string; confirmLabel: string; tags: PickTagOption[] };

interface DialogState {
  id: number;
  request: DialogRequest;
  /** prompt 的輸入內容。 */
  text: string;
  /** tags 的勾選結果。 */
  selected: number[];
}

export interface DialogApi {
  /** 破壞性動作的確認框。回傳 boolean。 */
  confirm: (options: ConfirmActionOptions) => Promise<boolean>;
  /** 單行／多行文字輸入框。回傳字串，或取消時的 null。 */
  promptText: (options: PromptTextOptions) => Promise<string | null>;
  /** 標籤多選框。回傳被勾選的 id 陣列，或取消時的 null。 */
  pickTags: (options: PickTagsOptions) => Promise<number[] | null>;
}

export interface AdminContextValue {
  toast: (message: string, tone?: ToastTone, timeout?: number) => void;
  dialog: DialogApi;
  /** rail 上的用戶數徽章。null 代表不顯示。 */
  userCount: number | null;
  /** 更新用戶數徽章。由 /admin 頁面在載入用戶清單後呼叫。 */
  setUserCount: (value: number | null) => void;
}

const AdminContext = createContext<AdminContextValue | null>(null);

/** 取用後臺共用能力。必須在 AdminProvider 內使用。 */
export function useAdmin(): AdminContextValue {
  const value = useContext(AdminContext);
  if (!value) throw new Error('[FORUM] useAdmin 必須在 AdminProvider 內使用。');
  return value;
}

/* ==========================================================================
   Provider
   ========================================================================== */

export interface AdminProviderProps {
  children: ReactNode;
}

export function AdminProvider({ children }: AdminProviderProps) {
  /* --- toast ------------------------------------------------------------- */

  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [userCount, setUserCount] = useState<number | null>(null);

  const toastIdRef = useRef(0);
  const toastTimersRef = useRef(new Map<number, ToastHandle>());

  const clearToastHandles = useCallback((id: number) => {
    const handles = toastTimersRef.current.get(id);
    if (!handles) return;
    cancelAnimationFrame(handles.frame);
    window.clearTimeout(handles.timer);
    toastTimersRef.current.delete(id);
  }, []);

  const dismissToast = useCallback(
    (id: number) => {
      clearToastHandles(id);
      setToasts((list) => list.map((item) => (item.id === id ? { ...item, open: false } : item)));
      // transitionend 在元素被分頁切換或 prefers-reduced-motion 影響時可能不觸發，
      // 因此移除用計時器保底 —— 與舊版同一個理由。
      const timer = window.setTimeout(() => {
        toastTimersRef.current.delete(id);
        setToasts((list) => list.filter((item) => item.id !== id));
      }, 400);
      toastTimersRef.current.set(id, { frame: 0, timer });
    },
    [clearToastHandles],
  );

  const toast = useCallback(
    (message: string, tone: ToastTone = 'info', timeout = 3800) => {
      const id = (toastIdRef.current += 1);
      setToasts((list) => [...list, { id, message, tone, open: false }]);
      // 次一幀才補 is-open：元素是在這次 render 才插進 DOM，若一開始就帶著
      // is-open，瀏覽器沒有任何一幀看得到 opacity:0，過場就不會播放。
      const frame = requestAnimationFrame(() => {
        setToasts((list) => list.map((item) => (item.id === id ? { ...item, open: true } : item)));
        toastTimersRef.current.set(id, { frame: 0, timer: window.setTimeout(() => dismissToast(id), timeout) });
      });
      toastTimersRef.current.set(id, { frame, timer: 0 });
    },
    [dismissToast],
  );

  useEffect(() => {
    const timers = toastTimersRef.current;
    return () => {
      for (const [, handles] of timers) {
        cancelAnimationFrame(handles.frame);
        window.clearTimeout(handles.timer);
      }
      timers.clear();
    };
  }, []);

  /* --- dialog ------------------------------------------------------------ */

  const dialogRef = useRef<HTMLDialogElement | null>(null);
  const pendingRef = useRef<((value: unknown) => void) | null>(null);
  const resultRef = useRef<unknown>(null);
  const dialogIdRef = useRef(0);
  const [dialogState, setDialogState] = useState<DialogState | null>(null);

  const settleDialog = useCallback(() => {
    const resolve = pendingRef.current;
    if (!resolve) return;
    pendingRef.current = null;
    resolve(resultRef.current);
  }, []);

  const openDialog = useCallback(
    (next: Omit<DialogState, 'id'>): Promise<unknown> =>
      new Promise((resolve) => {
        // 前一個視窗被新請求取代時等同取消。showModal() 在已開啟的 <dialog> 上
        // 會丟 InvalidStateError，而「快速連按兩顆列按鈕」在列表頁是常見情境。
        settleDialog();
        resultRef.current = null;
        pendingRef.current = resolve;
        dialogIdRef.current += 1;
        setDialogState({ ...next, id: dialogIdRef.current });
      }),
    [settleDialog],
  );

  // 顯示與關閉都放在 effect：showModal() 必須在 React 把這個請求的內容畫進
  // <dialog> 之後才呼叫，否則會看到一幀舊內容。
  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog || !dialogState) return;
    if (dialog.open) return;
    dialog.showModal();
    // 焦點要自己補：表單上的 autoFocus 在掛載那一刻就執行了，而那時 <dialog>
    // 還是關閉的，focus() 對 display:none 的元素不生效；showModal() 反而會
    // 把焦點放在 <dialog> 內第一個可聚焦元素（關閉鈕）。這裡再對一次輸入欄，
    // 與舊版 .dlg__form input, textarea, select { focus() } 的行為一致。
    const field =
      dialog.querySelector<HTMLElement>('[name="value"]') ??
      dialog.querySelector<HTMLElement>('.input, .textarea, .select');
    field?.focus();
  }, [dialogState]);

  const handleDialogClose = useCallback(() => {
    // 被取代的舊視窗，其 close 事件會排在新視窗 showModal 之後才送達。
    // 此時 open 為 true，代表這是上一個視窗的收尾事件，不該結清目前 pending
    // 的那一筆 —— 否則剛開的新視窗會立刻被 resolve 成 null。
    if (dialogRef.current?.open) return;
    settleDialog();
  }, [settleDialog]);

  useEffect(
    () => () => {
      if (dialogRef.current?.open) dialogRef.current.close();
      settleDialog();
    },
    [settleDialog],
  );

  const dialog = useMemo<DialogApi>(
    () => ({
      confirm: (options) =>
        openDialog({
          request: {
            kind: 'confirm',
            title: options.title,
            message: options.message ?? '',
            confirmLabel: options.confirmLabel ?? t('admin.dlgConfirm'),
            tone: options.tone ?? 'danger',
          },
          text: '',
          selected: [],
        }).then((value) => value === true),

      promptText: (options) =>
        openDialog({
          request: {
            kind: 'prompt',
            title: options.title,
            message: options.message ?? '',
            confirmLabel: options.confirmLabel ?? t('admin.dlgSave'),
            label: options.label,
            placeholder: options.placeholder ?? '',
            maxLength: options.maxLength ?? 0,
            rows: options.rows ?? 0,
          },
          text: options.value ?? '',
          selected: [],
        }).then((value) => (typeof value === 'string' ? value : null)),

      pickTags: (options) =>
        openDialog({
          request: {
            kind: 'tags',
            title: options.title,
            message: options.message ?? '',
            confirmLabel: t('admin.dlgApplyTags'),
            tags: options.tags,
          },
          text: '',
          selected: options.selectedIds ?? [],
        }).then((value) => (Array.isArray(value) ? (value as number[]) : null)),
    }),
    [openDialog],
  );

  const value = useMemo<AdminContextValue>(
    () => ({ toast, dialog, userCount, setUserCount }),
    [toast, dialog, userCount],
  );

  return (
    <AdminContext.Provider value={value}>
      {children}
      <Toasts items={toasts} onDismiss={dismissToast} />
      <Dialog
        state={dialogState}
        dialogRef={dialogRef}
        onClose={handleDialogClose}
        onTextChange={(text) => setDialogState((current) => (current ? { ...current, text } : current))}
        onToggleTag={(id) =>
          setDialogState((current) => {
            if (!current) return current;
            const selected = current.selected.includes(id)
              ? current.selected.filter((value) => value !== id)
              : [...current.selected, id];
            return { ...current, selected };
          })
        }
        onSubmit={(event) => {
          // 瀏覽器的 interactive validation 已經擋下未填 required 的提交；
          // 這裡再確認一次是為了讓「驗證失敗」時瀏覽器不關閉視窗的行為明確。
          if (!event.currentTarget.reportValidity()) {
            event.preventDefault();
            return;
          }
          const current = dialogState;
          if (!current) return;
          if (current.request.kind === 'confirm') {
            resultRef.current = true;
          } else if (current.request.kind === 'prompt') {
            resultRef.current = current.text.trim();
          } else {
            resultRef.current = [...current.selected];
          }
          // 刻意不 preventDefault：交給 method="dialog" 關閉視窗，
          // close 事件再呼叫 settleDialog 收尾。
        }}
      />
    </AdminContext.Provider>
  );
}

/* ==========================================================================
   呈現
   ========================================================================== */

interface ToastsProps {
  items: ToastItem[];
  onDismiss: (id: number) => void;
}

function Toasts({ items, onDismiss }: ToastsProps) {
  return (
    <div className="toasts" id="toasts" role="status" aria-live="polite">
      {items.map((item) => (
        <div
          key={item.id}
          className={`toast toast--${item.tone}${item.open ? ' is-open' : ''}`}
          onClick={() => onDismiss(item.id)}
        >
          {item.message}
        </div>
      ))}
    </div>
  );
}

interface DialogProps {
  state: DialogState | null;
  dialogRef: RefObject<HTMLDialogElement | null>;
  onClose: () => void;
  onTextChange: (value: string) => void;
  onToggleTag: (id: number) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

function Dialog({ state, dialogRef, onClose, onTextChange, onToggleTag, onSubmit }: DialogProps) {
  if (!state) return null;
  const { request } = state;
  const dismiss = () => dialogRef.current?.close();
  // tone 只在確認框有語意（破壞性動作是紅的）；文字輸入與標籤挑選都是
  // 中性動作，一律用主要色的按鈕。
  const tone = request.kind === 'confirm' ? request.tone : 'primary';

  return (
    <dialog className="dlg" ref={dialogRef} onClose={onClose}>
      {/*
        key 換掉整個 form：每次開新視窗都是一次全新掛載，輸入框不會沿用上一次
        殘留的自動填入內容。焦點則由 provider 掛載後的 effect 統一處理
        （見該處註解），因此這裡不需要 autoFocus。
      */}
      <form className="dlg__form" method="dialog" key={state.id} onSubmit={onSubmit}>
        <header className="dlg__head">
          <h2 className="dlg__title">{request.title}</h2>
          <button className="dlg__close" type="button" onClick={dismiss} aria-label={t('admin.dlgClose')}>
            <Icon name="close" />
          </button>
        </header>
        <div className="dlg__body">
          {request.message ? <p className="dlg__message">{request.message}</p> : null}
          {request.kind === 'prompt' ? (
            <label className="field">
              <span className="field__label">{request.label}</span>
              {request.rows > 0 ? (
                <textarea
                  className="textarea"
                  name="value"
                  rows={request.rows}
                  placeholder={request.placeholder}
                  maxLength={request.maxLength || undefined}
                  required
                  value={state.text}
                  onChange={(event) => onTextChange(event.target.value)}
                />
              ) : (
                <input
                  className="input"
                  name="value"
                  type="text"
                  value={state.text}
                  placeholder={request.placeholder}
                  maxLength={request.maxLength || undefined}
                  required
                  onChange={(event) => onTextChange(event.target.value)}
                />
              )}
            </label>
          ) : null}
          {request.kind === 'tags' ? (
            <div className="check-list">
              {request.tags.length === 0 ? (
                <p className="muted">{t('admin.dlgNoTags')}</p>
              ) : (
                request.tags.map((tag) => (
                  <label className="check-list__item" key={tag.id}>
                    <input
                      type="checkbox"
                      checked={state.selected.includes(tag.id)}
                      onChange={() => onToggleTag(tag.id)}
                    />
                    <span>{tag.name}</span>
                    <span className="check-list__id">#{tag.id}</span>
                  </label>
                ))
              )}
            </div>
          ) : null}
        </div>
        <footer className="dlg__foot">
          {/* 取消路徑（✕、取消鈕、Esc）一律是 type="button"，因此不會觸發
              method="dialog" 的提交，也不會被驗證邏輯攔下。 */}
          <button className="btn btn--ghost" type="button" onClick={dismiss}>
            {t('common.cancel')}
          </button>
          <button className={`btn btn--${tone}`} type="submit">
            {request.confirmLabel}
          </button>
        </footer>
      </form>
    </dialog>
  );
}
