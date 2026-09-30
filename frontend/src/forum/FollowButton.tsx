/*
 * 追蹤鈕（src/forum/FollowButton.tsx）
 *
 * 一顆按鈕、三個位置：貼文卡作者列（inline）、他人個人頁（panel）、
 * 追蹤頁的清單列（inline）。因此樣式在 style.css 第 3 節，與貼文卡同層。
 *
 * 為什麼是文字藥丸而不是一顆圖示加文字
 * --------------------------------
 * 貼文卡的作者列已經有暱稱、管理員指派的標籤，旁邊再放一顆只有圖示的按鈕
 * 需要一個新的 SVG 路徑常數（src/icons.tsx 的既有做法是路徑寫死、直接以 JSX
 * 表達），而追蹤的兩種狀態本來就需要文字才說得清楚 —— 「追蹤」與「追蹤中」
 * 對讀螢幕軟體的使用者比兩個長得差不多的鈴鐺有意義得多。
 *
 * 可及性
 * ------
 *   - aria-pressed 讓它是一顆 toggle button，而不是每次都換名字的按鈕：
 *     螢幕閱讀器會念出「已按下 / 未按下」，比念文字更準確地表達狀態。
 *   - disabled 而不是「未登入時不渲染」：未登入者按下去應該得到「去登入」
 *     的結果（由父層的 onRequireLogin 導頁），不是一個按不動的按鈕。
 *   - busy 期間不換文字（只加 .is-busy 淡出），因此按鈕寬度不會跳動 ——
 *     作者列是 flex-wrap，寬度跳動會讓整列換行。
 */

import { t } from '../i18n';

export type FollowVariant = 'inline' | 'panel';

export interface FollowButtonProps {
  /** 對象的 publicForumKey（64 個十六進位字元）。 */
  userKey: string;
  following: boolean;
  /** 送 POST 的期間。同一顆按鈕的 busy 由父層的 useFollow 控管。 */
  busy?: boolean;
  /** 未登入或任何無法操作的情況。 */
  disabled?: boolean;
  variant?: FollowVariant;
  /** 取消追蹤的 aria-label（與視覺文字不同：視覺是「追蹤中」）。 */
  onToggle: () => void;
}

export function FollowButton({ userKey, following, busy = false, disabled = false, variant = 'inline', onToggle }: FollowButtonProps) {
  // 沒有對象就不渲染：空字串會讓 useFollow.toggle 立刻返回 false，畫面上則是
  // 一顆永遠按不動的按鈕。父層在「這個金鑰沒有對應的公開個人資料」時會傳空字串
  // （後端在查無此人時刻意不回 following 欄位），那種情況正確的呈現是沒有按鈕。
  if (!userKey) return null;

  const classes = ['follow-button', `follow-button--${variant}`];
  // 追蹤中的樣態掛成明確的 class 而不是 :active / :focus：那兩者是「使用者
  // 剛碰過它」，重新載入後就消失，按鈕會顯示成未追蹤但實際上已追蹤。
  if (following) classes.push('follow-button--on');
  if (busy) classes.push('is-busy');

  return (
    <button
      className={classes.join(' ')}
      type="button"
      aria-pressed={following}
      aria-label={following ? t('follow.unfollow') : t('follow.label')}
      disabled={disabled || busy}
      onClick={onToggle}
    >
      {following ? t('follow.actionDone') : t('follow.action')}
    </button>
  );
}
