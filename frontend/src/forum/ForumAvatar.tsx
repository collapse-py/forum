/*
 * 使用者頭像（src/forum/ForumAvatar.tsx）
 *
 * 四個頁面都要畫「這個人是誰」的那一顆圓形：貼文卡的作者列（還要能點進公開
 * 個人頁）、自己的個人資料頁、他人的公開個人頁，以及編輯表單裡的預覽。它們的
 * 外框各不相同（.avatar 40px、.profile-avatar 64px、表單裡又是另一個尺寸），
 * 但「有頭像就顯示圖片、沒有就退回首字」這個判斷只有一份。
 *
 * 為什麼不各自在 JSX 裡寫條件渲染：那個條件會出現在四個地方，而其中一處漏掉
 * 時的症狀是「某個頁面的作者永遠沒有頭像」—— 沒有錯誤、沒有 console，只有一顆
 * 永遠是文字圓圈。更糟的是「首字」的計算法（姓名取第一個字、空名退匿名）也會
 * 被複製四份，而它必須與 PostCard 既有的 `post.authorAnonymous` 行為一致。
 *
 * 兩個刻意的設計：
 *
 *  1. **圖片用 alt=""**。頭像旁邊一定有作者名稱（或頁面標題），那才是這個人的
 *     可及名稱；頭像本身是裝飾。給它一個「使用者的頭像」之類的 alt 只會讓螢幕
 *     閱讀器在同一個區塊裡念兩次同樣的資訊。（表單裡的預覽是例外，呼叫端可以用
 *     label 給它一個真正的 alt。）
 *  2. **loading="lazy"**。一篇貼文的頭像與附圖會一起出現在列表裡，而列表是
 *     無限捲動的：25 顆頭像全部 eager 會讓第一屏多 25 個請求，其中多數永遠不
 *     會被捲到。
 */

import { t } from '../i18n';

export interface ForumAvatarProps {
  /**
   * 頭像網址。省略或空字串代表「沒有頭像」—— 此時渲染首字圓圈。
   *
   * 語意與 ForumPost.imageUrl 相同：省略即沒有，因此呼叫端不必分辨
   * 「沒有頭像」與「頭像壞了」。
   */
  url?: string | undefined;
  /** 顯示名稱。取它的第一個字作為首字；空白時退回匿名使用者。 */
  name: string;
  /** 外層元素的 class。尺寸與圓形由呼叫端的樣式決定（.avatar / .profile-avatar…）。 */
  className: string;
  /**
   * 給定時外層渲染成 <a>（頭像可點擊前往這位使用者的公開個人頁）。
   *
   * 存在的理由只有 PostCard：那一顆頭像與作者名是兩個相鄰的連結，指向同一個
   * 地方。沒有 href 的場合（個人資料卡的標題列）渲染成 <div>。
   */
  href?: string | undefined;
  /**
   * 圖片的 alt。預設空字串（裝飾性，理由見檔頭）。
   *
   * 只有「這張圖是使用者正要確認的內容」時才該給值 —— 例如編輯表單裡的新頭像
   * 預覽（profile.avatarPreviewAlt）。
   */
  label?: string | undefined;
}

export function ForumAvatar({ url, name, className, href, label }: ForumAvatarProps) {
  const initial = (name.trim() || t('post.authorAnonymous')).charAt(0);
  const image = url ? (
    <img className="avatar-image" src={url} alt={label ?? ''} loading="lazy" decoding="async" />
  ) : null;

  if (href) {
    return (
      <a className={className} href={href}>
        {image ?? initial}
      </a>
    );
  }
  return <div className={className}>{image ?? initial}</div>;
}
