/*
 * 語言切換器（src/i18n/LanguageSwitcher.tsx）
 *
 * 右上角的兩字母代碼（TW / CN / EN / JP），點開後是全部語言的下拉清單。
 * 九個頁面共用這一個元件：公開頁放在導覽列的 .nav-actions，後臺放在 topbar 的
 * .topbar__actions，兩邊的差別只有外觀（variant），互動邏輯完全相同。
 *
 * 選單自己實作而不用 <select> 或 <details> 的三個理由：
 *   1. <select> 的選項由瀏覽器原生繪製，無法使用本站的字體與圓角，行動裝置上
 *      更是會跳成系統選單，與介面完全脫節。
 *   2. <details> 沒有「點外部關閉」的行為 —— 點了頁面其他地方選單還開著，是最
 *      常見的可及性問題之一。
 *   3. 需要 role="menu" / menuitemradio 這組語意才符合「一組互斥的單選」，
 *      而 <details> 只能表達展開／收合，拿不到「目前選哪一個」。
 *
 * 鍵盤行為（WAI-ARIA menu button 模式）：Enter/Space/↓ 開啟並聚焦目前選項，
 * ↑ 開啟並聚焦最後一項、↑↓ 移動、Home/End 跳首尾、Enter 選取、Esc 關閉並把
 * 焦點還給觸發鈕、Tab 直接離開（順便關閉）。
 *
 * 選單會捲動：語言已經有十七種，每項 34px 加起來遠超過小螢幕的高度，style.css
 * 因此給了 max-height + overflow-y。焦點移動時瀏覽器會自動把項目捲進視窗，
 * 所以鍵盤操作在清單底部一樣正常。
 */

import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react';

import { useI18n } from './index';
import type { LocaleCode } from './locales';

export interface LanguageSwitcherProps {
  /**
   * 外觀。'forum' 走公開頁的膠囊按鈕（白底、位於導覽列右側），'admin' 走後臺
   * topbar 的深色連結樣式。樣式都在 style.css，元件本身不判斷自己在哪一頁 ——
   * 由呼叫處決定，比從 DOM 反推可靠。
   */
  variant: 'forum' | 'admin';
}

export function LanguageSwitcher({ variant }: LanguageSwitcherProps) {
  const { locale, meta, locales, setLocale, t } = useI18n();
  const [open, setOpen] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([]);
  /**
   * 開啟時要聚焦哪一項。null 代表「聚焦目前選中的那一項」。
   *
   * 之所以需要這個而不是在事件處理器裡直接 focus()：那一刻觸發鈕還在 DOM 上
   * 而選單還沒被繪製（清單是條件渲染），focus() 會落在不存在的元素上。用 ref
   * 記住意圖、讓 effect 在 commit 之後執行，兩邊的時序就不必互相猜。
   */
  const pendingFocusRef = useRef<number | null>(null);

  const selectedIndex = Math.max(
    0,
    locales.findIndex((item) => item.code === locale),
  );

  const focusItem = useCallback(
    (index: number) => {
      const bounded = (index + locales.length) % locales.length;
      itemRefs.current[bounded]?.focus();
    },
    [locales.length],
  );

  /** 目前焦點落在第幾項；-1 代表焦點不在選單內。 */
  const focusedIndex = () => itemRefs.current.indexOf(document.activeElement as HTMLButtonElement);

  /*
   * 開啟後聚焦 pendingFocus 指定的那一項，預設是「目前選中的那一項」而不是
   * 第一項 —— 選單是「目前的選擇 + 其他選項」，焦點落在沒有選取的項目上，
   * 使用者要再按幾次 ↓ 才碰得到自己正在用的語言，鍵盤操作的成本會明顯變高。
   */
  useEffect(() => {
    if (!open) return;
    const target = pendingFocusRef.current ?? selectedIndex;
    pendingFocusRef.current = null;
    focusItem(target);
  }, [open, focusItem, selectedIndex]);

  /*
   * 點擊容器以外就關閉。
   *
   * 監聽 pointerdown 而不是 click：pointerdown 在指標按下時就觸發，早於隨後的
   * click，也不用擔心「在別處按下、在別處放開」該不該算點到外面。監聽掛在
   * document 上而不是用 focusout —— focusout 在同一頁的兩個元素之間移動焦點時
   * 也會觸發，會讓選單莫名關掉。
   *
   * 刻意不阻止那個點擊穿透：情境是使用者展開選單、看見要繁體中文、點擊頁面
   * 另一處 —— 這個點擊本來就該作用在被點的地方，而不是被選單攔下。
   */
  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      if (containerRef.current?.contains(event.target as Node)) return;
      setOpen(false);
    };
    document.addEventListener('pointerdown', onPointerDown);
    return () => document.removeEventListener('pointerdown', onPointerDown);
  }, [open]);

  /*
   * 捲動或改變視窗大小就關掉。選單是絕對定位在觸發鈕上的，不會跟著內容走，
   * 留著只會讓它指著一個已經不在畫面上的按鈕，還擋住下方的點擊。
   *
   * 這裡有一個容易漏掉的例外：選單本身在語言變多之後會變高（十七種語言 × 每項
   * 34px ≈ 578px，遠超過小螢幕），因此 max-height + overflow-y 讓它自己會捲動。
   * 而 scroll 監聽掛在 window 上且用 capture 階段，選單內部的捲動事件也會被收��
   * —— 沒有這個判斷，使用者想看完清單底部就會發現選單自己關了。
   * 判斷 event.target 是最直接的：自己捲自己不算「頁面在動」。
   */
  useEffect(() => {
    if (!open) return;
    const close = () => setOpen(false);
    const onScroll = (event: Event) => {
      if (containerRef.current?.contains(event.target as Node)) return;
      setOpen(false);
    };
    window.addEventListener('resize', close);
    window.addEventListener('scroll', onScroll, true);
    return () => {
      window.removeEventListener('resize', close);
      window.removeEventListener('scroll', onScroll, true);
    };
  }, [open]);

  const closeMenu = (restoreFocus: boolean) => {
    setOpen(false);
    if (restoreFocus) triggerRef.current?.focus();
  };

  const choose = (code: LocaleCode) => {
    setLocale(code);
    // 關閉後焦點還給觸發鈕：選完就消失的焦點會讓鍵盤使用者失去位置，必須再按
    // 一次 Tab 才回得去。
    closeMenu(true);
  };

  const onTriggerKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    switch (event.key) {
      case 'ArrowDown':
      case 'Enter':
      case ' ':
        event.preventDefault();
        setOpen(true);
        break;
      case 'ArrowUp':
        // ↑ 是「開啟並聚焦最後一項」，與 ↓ 的對稱。
        event.preventDefault();
        pendingFocusRef.current = locales.length - 1;
        setOpen(true);
        break;
      default:
        break;
    }
  };

  const onMenuKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const current = focusedIndex();
    // 焦點不在選單內（理論上不會發生）時以「目前選中項」為基準，方向鍵才不會
    // 從 -1 跳到清單尾端。
    const base = current < 0 ? selectedIndex : current;
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault();
        focusItem(base + 1);
        break;
      case 'ArrowUp':
        event.preventDefault();
        focusItem(base - 1);
        break;
      case 'Home':
        event.preventDefault();
        focusItem(0);
        break;
      case 'End':
        event.preventDefault();
        focusItem(locales.length - 1);
        break;
      case 'Escape':
        event.preventDefault();
        closeMenu(true);
        break;
      case 'Tab':
        // 不 preventDefault：Tab 應該真的離開這個控制項。順便關閉，免得留下一個
        // 開著卻沒有焦點的選單。
        setOpen(false);
        break;
      default:
        break;
    }
  };

  return (
    <div className={`lang-switch lang-switch--${variant}`} ref={containerRef}>
      <button
        className="lang-switch__trigger"
        type="button"
        ref={triggerRef}
        aria-haspopup="menu"
        aria-expanded={open}
        // 標籤說明「這是什麼」加「現在是什麼」：只寫「語言」時，讀螢幕軟體
        // 的使用者聽不出按下去會不會改變介面的語言。
        aria-label={t('i18n.current', { name: meta.nativeName })}
        onClick={() => (open ? closeMenu(false) : setOpen(true))}
        onKeyDown={onTriggerKeyDown}
      >
        <span className="lang-switch__badge">{meta.badge}</span>
        <span className="lang-switch__caret" aria-hidden="true" />
      </button>

      {/*
        關閉時用 hidden 拿掉整個節點，而不是只藏起來：清單裡的按鈕都是可聚焦
        元素，留在 DOM 裡卻 display:none 的話，仍會被「下一個可聚焦元素」之類
        的工具算進去。
      */}
      <div
        className="lang-switch__menu"
        role="menu"
        aria-label={t('i18n.ariaLabel')}
        hidden={!open}
        onKeyDown={onMenuKeyDown}
      >
        {locales.map((item, index) => (
          <button
            key={item.code}
            className="lang-switch__item"
            type="button"
            role="menuitemradio"
            aria-checked={item.code === locale}
            // roving tabindex：整組只有一個可 Tab 到的項目，其餘用方向鍵。
            // 沒有它的話，使用者要按四次 Tab 才出得了選單。
            tabIndex={index === selectedIndex ? 0 : -1}
            ref={(element) => {
              itemRefs.current[index] = element;
            }}
            onClick={() => choose(item.code)}
          >
            <span className="lang-switch__item-badge">{item.badge}</span>
            <span className="lang-switch__item-name">{item.nativeName}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
