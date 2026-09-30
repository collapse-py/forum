/*
 * 可用的語言（src/i18n/locales.ts）
 *
 * 這一檔只放「有哪些語言」與「怎麼把瀏覽器回報的語言標籤對應到其中一種」。
 * 訊息本身在 messages.ts（繁體中文）、translations/*.ts（其他語言）。
 *
 * 為什麼只有四種：本站的文案量約兩百條，語言數每多一種，維護成本就是線性增加
 * —— 新增一個功能就得多寫一份。加語言只需要在這裡加一個項目、加一個翻譯檔，
 * 介面本身不用動（這是下面 type MessageKey 與 Record<MessageKey, string> 組
 * 合起來保證的）。
 *
 * 為什麼 badge 用地區碼而不是語言碼：EN / TW / CN / JP 在右上角一眼可辨，
 * 而且與 <html lang> 的寫法（zh-TW / en / ja）可以互相對照。使用者看到 JP
 * 不會誤以為那是「某個日本的論壇語系」，因為下拉清單裡緊接著就是日本語全文。
 */

/** 站台支援的語言代碼。這個聯集型別是 messages 與 translations 的共同基準。 */
export type LocaleCode =
  | 'zh-TW'
  | 'zh-HK'
  | 'zh-MO'
  | 'zh-CN'
  | 'en'
  | 'ja'
  | 'ko'
  | 'vi'
  | 'th'
  | 'id'
  | 'fr'
  | 'de'
  | 'es'
  | 'pt-BR'
  | 'ru'
  | 'ar'
  | 'hi';

export interface LocaleMeta {
  code: LocaleCode;
  /** 右上角顯示的兩字母代碼。 */
  badge: string;
  /** 下拉清單裡的名稱。用該語言自己書寫，不翻譯。 */
  nativeName: string;
  /** 給 <html lang> 與 Intl.NumberFormat / DateTimeFormat 用的標準標籤。 */
  htmlLang: string;
  /**
   * 書寫方向。給 <html dir> 用。
   *
   * 為什麼在資料裡而不是用一段 CSS 或 Intl 推斷：書寫方向決定的不只是文字對齊，
   * 還有整頁的版面鏡像（導覽列的欄位順序、rail 在左還在右、抽屜滑入的方向），
   * 而 CSS 只認得 dir 屬性。一個明確的欄位比「從 locale 對照表查」可靠，也讓
   * 新增語言時必須宣告方向 —— 不會忘記。
   */
  dir: 'ltr' | 'rtl';
}

/**
 * 支援的語言。清單順序就是下拉選單的順序，因此這裡也決定了閱讀順序：
 * 中文（含三個地區變體）→ 亞洲其他 → 歐洲語系 → 其他。中文放最前面是因為那是
 * 這個站點的原始語言。
 */
export const LOCALES: readonly LocaleMeta[] = [
  { code: 'zh-TW', badge: 'TW', nativeName: '繁體中文', htmlLang: 'zh-Hant-TW', dir: 'ltr' },
  { code: 'zh-HK', badge: 'HK', nativeName: '香港繁體', htmlLang: 'zh-Hant-HK', dir: 'ltr' },
  { code: 'zh-MO', badge: 'MO', nativeName: '澳門繁體', htmlLang: 'zh-Hant-MO', dir: 'ltr' },
  { code: 'zh-CN', badge: 'CN', nativeName: '简体中文', htmlLang: 'zh-Hans-CN', dir: 'ltr' },
  { code: 'en', badge: 'EN', nativeName: 'English', htmlLang: 'en', dir: 'ltr' },
  { code: 'ja', badge: 'JP', nativeName: '日本語', htmlLang: 'ja', dir: 'ltr' },
  { code: 'ko', badge: 'KO', nativeName: '한국어', htmlLang: 'ko', dir: 'ltr' },
  { code: 'vi', badge: 'VN', nativeName: 'Tiếng Việt', htmlLang: 'vi', dir: 'ltr' },
  { code: 'th', badge: 'TH', nativeName: 'ไทย', htmlLang: 'th', dir: 'ltr' },
  { code: 'id', badge: 'ID', nativeName: 'Bahasa Indonesia', htmlLang: 'id', dir: 'ltr' },
  { code: 'fr', badge: 'FR', nativeName: 'Français', htmlLang: 'fr', dir: 'ltr' },
  { code: 'de', badge: 'DE', nativeName: 'Deutsch', htmlLang: 'de', dir: 'ltr' },
  { code: 'es', badge: 'ES', nativeName: 'Español', htmlLang: 'es', dir: 'ltr' },
  { code: 'pt-BR', badge: 'BR', nativeName: 'Português (Brasil)', htmlLang: 'pt-BR', dir: 'ltr' },
  { code: 'ru', badge: 'RU', nativeName: 'Русский', htmlLang: 'ru', dir: 'ltr' },
  { code: 'ar', badge: 'AR', nativeName: 'العربية', htmlLang: 'ar', dir: 'rtl' },
  { code: 'hi', badge: 'HI', nativeName: 'हिन्दी', htmlLang: 'hi', dir: 'ltr' },
];

/**
 * 沒有任何線索（無 cookie、navigator.languages 為空、連 html lang 都沒有）
 * 時的語言。
 *
 * 選繁體中文的理由是這個站台本來就是繁體中文介面：與其猜一個隨機使用者的偏好，
 * 不如讓「沒有偏好」的人看到網站原本的樣子。英文因此不會是預設值 —— 一個把
 * 介面改成英文的站台，沒設 cookie 的訪客理應先看到原文。
 */
export const DEFAULT_LOCALE: LocaleCode = 'zh-TW';

const META_BY_CODE: ReadonlyMap<LocaleCode, LocaleMeta> = new Map(LOCALES.map((item) => [item.code, item]));

export function localeMeta(code: LocaleCode): LocaleMeta {
  const meta = META_BY_CODE.get(code);
  if (!meta) throw new Error(`[FORUM] 未知語言：${code}`);
  return meta;
}

export function isLocaleCode(value: string): value is LocaleCode {
  return META_BY_CODE.has(value as LocaleCode);
}

/**
 * 語言標籤 → 站台語言的比對表。
 *
 * 存在的理由是 BCP 47 允許的標籤遠多於我們翻譯的語言：瀏覽器可能回報
 * zh-Hant-HK、en-AU、pt-PT、ar-EG…，而我們只有十七種語言。沒有這張表，這些
 * 標籤全都會掉到預設值，於是香港的繁體使用者拿不到香港版、澳洲的英文使用者拿不到
 * 英文（掉到 zh-TW）—— 後者是最常見的實際情況。
 *
 * 比對必須「最長的鍵優先」，因此比對前會依鍵長排序。原因很具體：沒有排序時，
 * `zh` 這個泛用前綴會先命中 `zh-hans-cn`，把明確要求簡體的使用者送去繁體。
 * 這是這張表裡唯一一個順序會改變結果的地方。
 *
 * 比對是「完全相同」或「後面接 - 的前綴」。不加邊界的話 'e' 會命中 'en'、
 * 'ja' 會命中 'jpy' 之類的東西。
 */
const ALIASES: readonly (readonly [string, LocaleCode])[] = [
  // 中文：明確的地區與文字系統先於泛用的 zh
  ['zh-hant-hk', 'zh-HK'],
  ['zh-hant-mo', 'zh-MO'],
  ['zh-hant-tw', 'zh-TW'],
  ['zh-hant', 'zh-TW'],
  ['zh-hk', 'zh-HK'],
  ['zh-mo', 'zh-MO'],
  ['zh-tw', 'zh-TW'],
  ['zh-hans-cn', 'zh-CN'],
  ['zh-hans-sg', 'zh-CN'],
  ['zh-hans', 'zh-CN'],
  ['zh-cn', 'zh-CN'],
  ['zh-sg', 'zh-CN'],
  ['zh-my', 'zh-CN'],
  ['zh', 'zh-TW'],
  // 葡語：我們只有巴西版。pt-PT（歐洲）與裸 pt 都送到 pt-BR —— 巴西葡語的
  // 使用者遠多於歐洲版，而兩邊互相看得懂的比例遠高於反過來。
  ['pt-br', 'pt-BR'],
  ['pt', 'pt-BR'],
  // 其餘語言本體
  ['en', 'en'],
  ['ja', 'ja'],
  ['ko', 'ko'],
  ['vi', 'vi'],
  ['th', 'th'],
  ['id', 'id'],
  ['fr', 'fr'],
  ['de', 'de'],
  ['es', 'es'],
  ['ru', 'ru'],
  ['ar', 'ar'],
  ['hi', 'hi'],
];

/** ALIASES 依前綴長度由長到短排序後的快取；第一次呼叫時建立。 */
let aliasByLength: readonly (readonly [string, LocaleCode])[] | null = null;

/**
 * 把單一語言標籤對應到站台支援的語言。
 *
 * 大小寫不敏感（BCP 47 的語言與地區子標籤在常見實作裡是任意大小寫，
 * navigator.languages 也可能給出 'zh-tw'），並且會去掉底層的 Unicode
 * 副語言（ja-JP-u-ca-japanese）—— Intl.getCanonicalLocales 會把它保留成
 * 'ja-JP-u-ca-japanese'，直接比對前綴會多一層比對，因此先切掉 -u- 之後的部分。
 *
 * 對不上就回 null，由呼叫端決定要不要繼續往下一個線索找。
 */
export function matchLocale(tag: string): LocaleCode | null {
  const lower = tag.toLowerCase().trim();
  if (lower === '') return null;

  // -u- 之後是 Unicode 副語言擴充（ca、nu、…），與「顯示哪一種語言」無關。
  const base = lower.split('-u-')[0] ?? lower;

  if (!aliasByLength) {
    aliasByLength = [...ALIASES].sort((a, b) => b[0].length - a[0].length);
  }
  for (const [alias, code] of aliasByLength) {
    // 必須是「完全相同」或「後面接 - 的前綴」。不加邊界的話 'e' 會命中 'en'、
    // 'ja' 會命中 'jpy' 之類的東西。
    if (base === alias || base.startsWith(`${alias}-`)) return code;
  }
  return null;
}

/**
 * 依序試完一組候選語言標籤，回第一個對得上的。
 *
 * 標籤之間的優先順序有意義：navigator.languages[0] 是使用者的第一偏好，
 * [1] 是他排序在後但仍接受的。用者用 3 個人工排序的「次要語言」時，
 * 那是明確的訊息，不該被後面的兜底值蓋掉。
 */
export function matchFirstLocale(tags: readonly string[]): LocaleCode | null {
  for (const tag of tags) {
    const code = matchLocale(tag);
    if (code) return code;
  }
  return null;
}
