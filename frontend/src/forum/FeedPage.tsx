/*
 * 論壇首頁（/forum）
 *
 * 職責：文章動態（向下捲動無限載入）、留言層展開與分頁、按讚、檢舉，
 * 以及貼文搜尋（關鍵字 → 以全文檢索取代動態列表）。
 *
 * 這裡沒有 fetch：資料狀態在 useFeed / useSearch / useComments / useReport，
 * 本檔只負責把狀態組裝成頁面。舊版同一支檔案同時是狀態機、事件代理與模板，
 * 一篇貼文的生命週期要跨四處閱讀才看得完。
 *
 * 動態與搜尋是同一頁的兩種「內容來源」，而不是兩個頁面：使用者搜尋完想
 * 回到動態只要清掉關鍵字，不必重新導向（也不必讓瀏覽器多存一份歷史狀態）。
 * 兩者的卡片同為 PostCard，因為後端保證搜尋結果與列表的欄位形狀一致
 * （見 httpapi/search.go 的說明）。
 *
 * 順帶修掉一個舊版的靜默失敗：舊版在這裡有 status 元素，卻只在
 * forum-new 與 forum-profile 的 HTML 裡存在；forum.html 沒有，所以
 * setStatus() 在首頁是 no-op —— 按讚、留言、檢舉的失敗訊息全被丟掉，
 * 使用者只會看到按鈕沒反應。現在首頁有一行真正的狀態列。
 */

import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';

import { LoginRequiredError, errorMessage, goToLogin, requestJSON } from '../core';
import { msg, tr, t, usePageTitle, type Message } from '../i18n';
import type { ForumPost } from '../types';
import { SearchGlyph } from '../icons';
import { AnnouncementBanner } from './AnnouncementBanner';
import { PostCard } from './PostCard';
import { BottomNav, ForumNav, ForumShell, InstallHint, useAuth, usePwaInstall } from './shell';
import { useComments } from './useComments';
import { useFeed, toggleLike } from './useFeed';
import { useFollow } from './useFollow';
import { useMediaTokenRelease } from './useMediaTokenRelease';
import { useReport } from './useReport';
import { useSearch, searchStatusText } from './useSearch';

const PAGE_SIZE = 25;
const PRELOAD_DISTANCE = '800px';

/*
 * 導覽列的登入狀態文字是函式而不是模組層常數：模組層的物件會在 import 時把
 * 當下的語言固定住，使用者切換語言後整個 session 都不會跟著換。
 * 這一頁與新增貼文頁、登入閘門各有一組不同的三句，因此是頁面自己的資料。
 */
function authLabels(): { loggedIn: string; loggedOut: string; error: string } {
  return {
    loggedIn: t('auth.feedLoggedIn'),
    loggedOut: t('auth.feedLoggedOut'),
    error: t('auth.statusUnknown'),
  };
}

export function FeedPage() {
  usePageTitle('title.forum');

  const auth = useAuth();
  const install = usePwaInstall();

  // message 存的是 Message（key+參數）或後端回的字串，render 時才用 tr() 解析。
  // 存字串的話切換語言後這一行會停在舊語言（見 runtime.ts 的「延後翻譯的訊息」）。
  const [status, setStatus] = useState<{ message: Message | string | null; isError: boolean }>({
    message: null,
    isError: false,
  });
  const setOk = useCallback((message: Message | string) => setStatus({ message, isError: false }), []);
  const setFail = useCallback((message: Message | string) => setStatus({ message, isError: true }), []);

  const { state: feed, load: loadFeed, patch: patchFeedPost, remove: removeFeedPost, items: feedItems } = useFeed({ pageSize: PAGE_SIZE });
  const search = useSearch();
  const canInteract = auth.phase === 'ready' && auth.loggedIn;

  /*
   * 追蹤清單在這一頁只讀一次，動態與搜尋結果共用同一個 controller。
   *
   * autoLoad 綁在 canInteract 上是必要的，不是最佳化：GET /api/forum/follows
   * 掛在 requireLogin 後面，未登入時回 303 轉址到登入頁，requestJSON 會把它
   * 判成 LoginRequiredError 而把整頁導走 —— 匿名訪客會連首頁都開不了。
   * auth 從 checking 翻成 ready+loggedIn 時這個 effect 會重跑並補上載入。
   */
  const follow = useFollow({ autoLoad: canInteract });

  /*
   * 搜尋結果與動態各有一份 patch。兩個都呼叫、各自略過不屬於自己的 ID，
   * 比「判斷目前在哪個模式再選一個」少一個會忘記更新分支的地方。
   */
  const patchPost = useCallback(
    (postId: number, update: Partial<ForumPost>) => {
      patchFeedPost(postId, update);
      search.patch(postId, update);
    },
    [patchFeedPost, search],
  );
  const removePost = useCallback(
    (postId: number) => {
      removeFeedPost(postId);
      search.remove(postId);
    },
    [removeFeedPost, search],
  );
  // 留言數要從「目前看得到的那份列表」找出該篇貼文；兩種模式都找一次。
  const findItem = useCallback(
    (postId: number) => feedItems.find((item) => item.id === postId) ?? search.state.items.find((item) => item.id === postId),
    [feedItems, search.state.items],
  );

  const comments = useComments({
    onPosted: (postId) => {
      const post = findItem(postId);
      patchPost(postId, { commentCount: (post?.commentCount || 0) + 1 });
    },
    onError: setFail,
  });

  const report = useReport({ onSent: setOk, onError: setFail });
  const { cancel: cancelReport } = report;

  /* --- 搜尋 --------------------------------------------------------------- */

  /*
   * 輸入框的值與「已送出的關鍵字」是兩個狀態：前者隨打隨變，後者決定畫面
   * 顯示哪一份結果。刻意不綁在一起 —— 綁在一起會讓使用者每打一個字就送出
   * 一次請求（每個字都產生一次全文檢索），而且中途打錯的字也會立刻影響畫面。
   */
  const [keyword, setKeyword] = useState('');
  const searching = search.state.phase !== 'idle';

  // 網址帶 ?q= 進站：讓搜尋結果可以被分享，也讓重新整理不會掉回動態。
  // 取代而不是新增歷史紀錄，否則按返回會被一堆搜尋步驟塞滿。
  useEffect(() => {
    const initial = new URLSearchParams(window.location.search).get('q');
    if (initial && initial.trim() !== '') {
      setKeyword(initial);
      search.submit(initial);
    }
    // 僅在掛載時執行一次：search.submit 依賴穩定，但把它放進依賴會讓這段
    // 在每次 useSearch 內部回圈時重跑，造成無限請求。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSearch = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const next = keyword.trim();
      search.submit(next);
      const url = new URL(window.location.href);
      if (next === '') url.searchParams.delete('q');
      else url.searchParams.set('q', next);
      window.history.replaceState(null, '', url);
    },
    [keyword, search],
  );

  const clearSearch = useCallback(() => {
    setKeyword('');
    search.submit('');
    const url = new URL(window.location.href);
    url.searchParams.delete('q');
    window.history.replaceState(null, '', url);
  }, [search]);

  /* --- 圖片 token 釋放 ---------------------------------------------------- */

  /*
   * 動態與搜尋結果的圖片都要納入：兩者是同一頁的內容，離開時一併釋放。
   * 實作見 useMediaTokenRelease（為什麼走 sendBeacon、為什麼只送一次）。
   */
  useMediaTokenRelease(feedItems, search.state.items);

  /* --- 動作 --------------------------------------------------------------- */

  const handleLike = useCallback(
    (post: ForumPost) => {
      void (async () => {
        try {
          const result = await toggleLike(post.id);
          patchPost(post.id, { liked: result.liked, likeCount: result.count });
        } catch (error) {
          if (error instanceof LoginRequiredError) {
            goToLogin();
            return;
          }
          setFail(errorMessage(error, tr(msg('feed.likeFailed'))));
        }
      })();
    },
    [patchPost, setFail],
  );

  const handleDelete = useCallback(
    async (post: ForumPost) => {
      try {
        await requestJSON(`/api/forum/posts/${post.id}`, {
          method: 'DELETE',
          fallback: tr(msg('posts.deleteFailed')),
        });
        removePost(post.id);
        setOk(msg('posts.deleted'));
      } catch (error) {
        if (error instanceof LoginRequiredError) {
          goToLogin();
          return;
        }
        setFail(errorMessage(error, tr(msg('posts.deleteFailed'))));
      }
    },
    [removePost, setFail, setOk],
  );

  /* --- 預載 --------------------------------------------------------------- */

  /*
   * 同一個 sentinel 服務動態與搜尋兩種結果：IntersectionObserver 只認得
   * 「畫面底部」，不該關心底部是動態的第 3 頁還是搜尋的第 2 頁。要觸發哪一個
   * 由 searching 決定。
   *
   * 依賴刻意不放 searching 本身以外的狀態值：IntersectionObserver 重建一次
   * 成本很低，但每個關鍵字都重建會讓舊的 observer 在 disconnect 前觸發一次，
   * 導致送出兩個請求。
   */
  const sentinelRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const node = sentinelRef.current;
    if (!node) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries[0]?.isIntersecting) return;
        if (searching) search.loadMore();
        else loadFeed(false);
      },
      { rootMargin: PRELOAD_DISTANCE },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [loadFeed, search, searching]);

  const handleHome = useCallback(() => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
    cancelReport();
    // 首頁鈕的語意是「回到動態的最新一頁」，因此連搜尋狀態一起清掉；
    // 否則按首頁會停在同一份搜尋結果上，看起來像沒反應。
    if (searching) clearSearch();
    else loadFeed(true);
  }, [cancelReport, clearSearch, loadFeed, searching]);

  const renderPost = (post: ForumPost) => (
    <PostCard
      key={post.id}
      post={post}
      canInteract={canInteract}
      comments={comments}
      report={report}
      follow={follow}
      onLike={handleLike}
      onDelete={handleDelete}
      onRequireLogin={goToLogin}
    />
  );

  /*
   * 搜尋欄傳給 ForumNav 渲染，因此它位於導覽列內、整頁的最上方：
   * 使用者找東西時不必先捲到內容區，搜尋中想放棄也不用往回捲。
   *
   * 這裡只送出 form 本身；結果的狀態列（找到幾筆、降級模式）留在 <main> 內，
   * 那是「關於結果」的訊息，擺在結果上方才對，而塞進導覽列會讓那裡同時
   * 帶著輸入框與說明、在手機上換行成三行。
   */
  const searchForm = (
    <form className="search-bar" role="search" onSubmit={handleSearch}>
      {/*
        放大鏡、輸入框與兩個動作鈕都在同一顆藥丸內（.search-field），因此置中
        的是整顆搜尋框而不是「框 + 框外的按鈕」那一組 —— 後者的輸入框會因為
        右側的按鈕而偏離導覽列正中心約 32px。
        圖示不包 <label>：輸入框已有 aria-label，再包一層 label 會讓螢幕閱讀器
        讀出兩次名稱。
      */}
      <div className="search-field">
        <SearchGlyph className="search-field__icon" />
        <input
          className="search-input"
          type="search"
          name="q"
          value={keyword}
          maxLength={100}
          placeholder={t('feed.searchPlaceholder')}
          aria-label={t('feed.searchAriaLabel')}
          onChange={(event) => setKeyword(event.target.value)}
        />
        {searching ? (
          // type="button"：它與表單的送出鈕在同一個 <form> 裡，預設的 submit
          // 會讓「取消」變成「送出並取消」。
          <button className="search-go search-go--quiet" type="button" onClick={clearSearch}>
            {t('common.cancel')}
          </button>
        ) : null}
        <button className="search-go" type="submit">
          {t('common.search')}
        </button>
      </div>
    </form>
  );

  return (
    <ForumShell>
      <ForumNav auth={auth} labels={authLabels()} loginReturn="/forum" install={install} search={searchForm} />

      <main className="forum-main">
        {/*
         * 公告橫幅放在狀態列**之前**。
         *
         * 順序的理由：橫幅是「站方現在要說的事」，狀態列是「你剛才的操作結果」。
         * 反過來的話，一次失敗的操作訊息會被推到橫幅下方，而那則站方公告會
         * 被一個只與該使用者有關的訊息擋住 —— 而公告是給所有人看的。
         */}
        <AnnouncementBanner />

        {searching ? (
          <p className="status" role="status" aria-live="polite">
            {searchStatusText(search.state)}
          </p>
        ) : follow.state.error !== null ? (
          /*
           * 追蹤的錯誤優先於這一頁自己的狀態列：兩者不會同時發生（追蹤失敗
           * 不影響按讚或留言），但畫面只有一行，因此讓較新的那一則佔用它。
           * 用 tr() 而非 errorMessage()：這是存進 state 的訊息，切換語言
           * 後必須重新翻譯。
           */
          <p className="status error" role="status" aria-live="polite">
            {tr(follow.state.error)}
          </p>
        ) : (
          <p className={`status${status.isError ? ' error' : ''}`} role="status" aria-live="polite">
            {status.message === null ? null : tr(status.message)}
          </p>
        )}

        {searching ? (
          <section className="feed" aria-label={t('feed.searchResultsLabel')}>
            {search.state.phase === 'failed' ? <div className="empty">{t('feed.searchFailed')}</div> : null}
            {search.state.phase === 'ready' && search.state.items.length === 0 ? (
              <div className="empty">
                {t('feed.searchNoResults', { query: search.state.query })}
              </div>
            ) : null}
            {search.state.items.map(renderPost)}
            <div className="feed-status" role="status" aria-live="polite">
              {tr(search.state.status)}
            </div>
          </section>
        ) : (
          <section className="feed" aria-label={t('feed.postsLabel')}>
            {feed.phase === 'loading' ? <div className="loading">{t('feed.loadingPosts')}</div> : null}
            {feed.phase === 'failed' ? <div className="empty">{t('feed.postsFailed')}</div> : null}
            {feed.phase === 'ready' && feedItems.length === 0 ? (
              <div className="empty">{t('feed.noPosts')}</div>
            ) : null}
            {feedItems.map(renderPost)}
            <div className="feed-status" ref={sentinelRef} role="status" aria-live="polite">
              {tr(feed.status)}
            </div>
          </section>
        )}
      </main>

      <BottomNav active="home" onHome={handleHome} />

      <footer>
        <a href="/forum">{t('common.backToForumHome')}</a>
      </footer>
      <InstallHint hint={install.hint} />
    </ForumShell>
  );
}
