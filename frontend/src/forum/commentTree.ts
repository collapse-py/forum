/*
 留言的樹（src/forum/commentTree.ts）

 後端給的留言是**扁平**的一串（依 created_at 遞增），而畫面要的是「哪一則在回覆
 誰」的樹狀結構。組樹在這裡做而不是後端：分頁的 offset／hasMore 語意只在扁平列表
 上單純（見 useComments 的說明），而扁平陣列又剛好帶了組樹需要的一切 —— 子留言的
 created_at 必定晚於父留言（先有父才有子），因此陣列順序同時保證「組樹時父留言一定
 已經在陣列裡」。

 公開頁（PostCard 的 CommentBranch）與後台（ForumAdminPage 的 CommentThread）用
 的是同一個函式：兩邊的節點型別不同（ForumComment / AdminComment），但它們對組樹
 的需求只有 id 與 parentId 兩個欄位。抽成泛型而不是各寫一份的理由不是省那 20 行，
 而是「父留言不在陣列裡時退化為根節點」這個判斷必須兩邊一致 —— 一邊退化、一邊丟
 棄，會讓管理員看到與讀者不同的樹。
*/

/** 組樹用的節點：一則留言與它的回覆。 */
export interface CommentNode<T> {
  comment: T;
  replies: CommentNode<T>[];
}

/**
 * 把扁平的留言陣列組成一棵樹（回傳第一層的節點）。
 *
 * parentId 讀不到或為 0 都當成根節點 —— 那正是後端對第一層的表示（公開 API 省略
 * parentId，後台則是 0）。萬一某則回覆的父留言不在陣列裡（理論上不會發生：刪除是
 * 連子樹一起刪，而分頁由舊到新載入），它會退化成根節點而不是消失；看得見的內容比
 * 正確的層級重要。
 *
 * 順序保持陣列順序（不重新排序）：呼叫端傳進來的已經是 created_at 遞增，而回覆
 * 被子留言依建立時間掛在父留言後面。
 */
export function buildCommentTree<T extends { id: number; parentId?: number }>(items: T[]): CommentNode<T>[] {
  const nodes = new Map<number, CommentNode<T>>();
  for (const comment of items) {
    nodes.set(comment.id, { comment, replies: [] });
  }
  const roots: CommentNode<T>[] = [];
  for (const comment of items) {
    const node = nodes.get(comment.id);
    if (!node) continue;
    const parentId = Number(comment.parentId) || 0;
    const parent = parentId > 0 ? nodes.get(parentId) : undefined;
    if (parent) parent.replies.push(node);
    else roots.push(node);
  }
  return roots;
}
