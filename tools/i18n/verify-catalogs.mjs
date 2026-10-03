// 驗證所有翻譯目錄：鍵集、順序、佔位符、字元損毀、未翻譯殘留，以及 README 的語系宣稱。
//
// 目錄有兩種排版 —— 早期那批是一行一筆
//   'common.cancel': '取消',
// 新的（由 merge 腳本產生）是鍵與值分兩行
//   'common.cancel':
//     '取消',
// 因此比對的正則必須兩種都吃，否則會把整個檔案判成「缺 393 鍵」。
//
// 路徑一律以本檔位置（import.meta.url）推導，而不是 process.cwd()：
// 這個腳本同時需要讀 src/i18n/messages.ts（前端底下）與 README.md（倉庫根），
// 用 cwd 的話它只能從 frontend/ 執行，而 README 記的呼叫方式是
// node ../../tools/... —— 兩者對不上，結果是「文件寫的指令跑不起來」。
// 以檔位置為基準讓它從任何目錄執行都得到相同結果，與 vite.config.ts 的取法一致。
import { readFileSync, readdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..');
const messagesPath = resolve(repoRoot, 'frontend/src/i18n/messages.ts');
const translationsDir = resolve(repoRoot, 'frontend/src/i18n/translations');
const readmePath = resolve(repoRoot, 'README.md');

const PIN = [
  'newPost.eyebrow', 'profile.eyebrow', 'publicProfile.eyebrow', 'login.eyebrow',
  'users.eyebrow', 'users.signinEyebrow', 'posts.eyebrow', 'reports.eyebrow',
  'admin.consoleName', 'users.colId', 'common.placeholder',
];
// 明確的 \u 逸出，不用字面 CJK 字元：字面寫進檔案的範圍字元會在編碼往返中
// 悄悄錯位，實際上就因此把韓文誤判成「未翻譯」。
const HAN = /[\u3400-\u4DBF\u4E00-\u9FFF\uF900-\uFAFF]/;
const ph = (s) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',');

/**
 * 哪些語言可以用「還有漢字」來判斷未翻譯。
 *
 * 中文與日文本身就以漢字書寫，這條檢查對它們只會產生噪音 —— 日文介面裡
 * 出現「確認」「削除」是正常的，不是沒翻。韓文用 Hangul，介面裡出現漢字
 * 才是真的可疑，所以保留。
 */
const HAN_IS_FOREIGN = (file) => !/^(zh-|ja)/.test(file);

/*
 * U+FFFD（REPLACEMENT CHARACTER）＝ 字元損毀，與語言無關。
 *
 * 為什麼這條要獨立於「未翻譯殘留」：U+FFFD 不是任何一種語言的字，它出現在
 * 檔案裡只代表某個字元在**某一次編碼往返**中被解碼失敗並換成替代符號 ——
 * 原始位元組當場就丟了，因此它既不能被翻譯修好，也不會因為換語系而消失。
 *
 * 這是唯一一條「看起來像翻譯問題、實際上是資料損毀」的情況，而且現有檢查
 * 完全抓不到：佔位符仍對得上（損毀發生在純文字裡）、鍵集完整、順序正確。
 * 實際抓到過兩處：th.ts 的 export.batchNote 與 block.reasonHint，兩者都
 * 因為「整句還看得出大致意思」而被當成可接受的翻譯擱置了。
 *
 * 因此這一條**不放行**：命中就是 BAD，並在訊息裡指出是哪個鍵 —— 那個鍵的
 * 值已經永久失去，必須照 messages.ts 的原文重寫，不能靠修字元救回。
 */
const REPLACEMENT = /\uFFFD/;

const src = readFileSync(messagesPath, 'utf8');
const body = src.slice(src.indexOf('export const zhTW = {'));
const order = [...body.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':/gm)].map((m) => m[1]);
const srcVals = new Map(
  [...body.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':\s*\n?\s*'([\s\S]*?)',\r?\n/gm)].map((m) => [m[1], m[2]]),
);

const files = readdirSync(translationsDir).filter((f) => f.endsWith('.ts')).sort();
let bad = 0;

for (const file of files) {
  const text = readFileSync(resolve(translationsDir, file), 'utf8');
  const entries = [...text.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':\s*\n?\s*'([\s\S]*?)',\r?\n/gm)];
  const keys = entries.map((m) => m[1]);
  const vals = new Map(entries.map((m) => [m[1], m[2]]));

  const missing = order.filter((k) => !keys.includes(k));
  const extra = keys.filter((k) => !order.includes(k));
  const badPh = [];
  const untranslated = [];
  const damaged = [];
  for (const k of keys) {
    if (REPLACEMENT.test(vals.get(k))) damaged.push(k);
    if (PIN.includes(k)) continue;
    if (ph(srcVals.get(k) ?? '') !== ph(vals.get(k))) badPh.push(k);
    if (HAN_IS_FOREIGN(file) && HAN.test(vals.get(k))) untranslated.push(k);
  }
  const orderOk = JSON.stringify(keys) === JSON.stringify(order);
  const ok =
    !missing.length &&
    !extra.length &&
    !badPh.length &&
    !untranslated.length &&
    !damaged.length &&
    orderOk;
  if (!ok) bad += 1;
  console.log(
    `${ok ? 'OK  ' : 'BAD '}${file.padEnd(12)}${String(keys.length).padStart(4)}` +
      (missing.length ? ` missing:${missing.length}` : '') +
      (extra.length ? ` extra:${extra.length}` : '') +
      (orderOk ? '' : ' order:differs') +
      (badPh.length ? ` placeholder:${badPh.join(',')}` : '') +
      (untranslated.length ? ` untranslated:${untranslated.join(',')}` : '') +
      (damaged.length
        ? ` damaged:${damaged.join(',')}（含 U+FFFD，原始字元已丟失，須照 messages.ts 重寫）`
        : ''),
  );
}

/*
 * README 的語系宣稱必須與目錄實際內容一致。
 *
 * 為什麼要這個檢查：README 宣稱「18 個語系」而目錄裡只有 16 份加一個基準
 * （實際 17 個）這件事，靠人核對是核不出來的 —— 兩處數字分散在 1,100 行的
 * 文件裡，而「少了一個語系」不會讓任何編譯或測試失敗。這是「文件與實作漂移」
 * 最便宜的一次根治：新增或刪除一個語系檔時，這裡會指名 README 的哪一處過期。
 *
 * 檢查三件事，缺一不可：
 *   1. 特色列表的「N 個語系」= translations 檔數 + 1（基準的 zh-TW 不在該目錄）。
 *   2. 內文「共 N 份，連同基準的 zh-TW 共 M 個」的 N 與 M 分別對應檔數與總數。
 *      只檢查 M 會漏掉「份數寫對、總數寫錯」這種誤差方向相反的情況。
 *   3. README 列出的語系清單與實際檔名完全相同（順序不拘）。數字對但清單
 *      少一個，同樣是漂移 —— 而這是刪除語系時最常見的結果。
 */
const readme = readFileSync(readmePath, 'utf8');
const total = files.length + 1;
const declared = files.map((f) => f.replace(/\.ts$/, ''));
const readmeProblems = [];

const headline = readme.match(/\*\*多語系\*\*：(\d+) 個語系/);
if (!headline) {
  readmeProblems.push('README.md 找不到「**多語系**：N 個語系」那一行（格式已變？請更新這個腳本）');
} else if (Number(headline[1]) !== total) {
  readmeProblems.push(`README.md 的「**多語系**：${headline[1]} 個語系」應為 ${total} 個（${declared.length} 份目錄 + 基準的 zh-TW）`);
}

const detail = readme.match(/共 (\d+) 份，連同基準的 `zh-TW`[\s\S]{0,40}?共 (\d+) 個/);
if (!detail) {
  readmeProblems.push('README.md 找不到「共 N 份，連同基準的 `zh-TW` 共 M 個」那一段（格式已變？請更新這個腳本）');
} else {
  if (Number(detail[1]) !== files.length) {
    readmeProblems.push(`README.md 的「共 ${detail[1]} 份」應為 ${files.length} 份（translations/ 下的實際檔數）`);
  }
  if (Number(detail[2]) !== total) {
    readmeProblems.push(`README.md 的「連同基準的 zh-TW 共 ${detail[2]} 個」應為 ${total} 個`);
  }
  /*
   * 語系清單：精確切出「共 M 個：」之後到該句句號為止的那一段。
   *
   * 邊界為什麼必須收這麼緊：這一節之後的內容全是反引號包住的設定鍵名
   * （MEDIA_TOKEN_KEY_PREFIX、http.ListenAndServe、forum:session:…），而那個
   * 段落很長、內容量又大。視窗只要寬一點，就會把那些鍵名誤判成「README 多列了
   * 不存在的語系」—— 一個每次都失敗的檢查等於沒有檢查。
   *
   * 為什麼用「句號」而不是「空行」當結尾：這一段是換行過的長段落，空行要到
   * 好幾個小節之後才會出現。語系清單本身是一個以全形句號結束的句子，因此句號
   * 才是它的真正邊界。
   *
   * 為什麼不對反引號後面的標點做前瞻比對：清單的寫法是 `ar`（RTL）、`de`、`en`…，
   * 而 `ar` 後面跟的是全形括號而不是頓號，任何前瞻寫法都會漏掉它 —— 而漏掉的
   * 恰好是這一項最該抓到的情況（刪掉第一個語系時，總數對不上、清單也少一項，
   * 兩個檢查都應該出聲）。改為「收集切段內所有反引號識別碼，再濾掉明顯不是
   * 語系的」：寬鬆的收集配嚴格的比對，比嚴格的收集更不容易產生假陰性。
   */
  const listTail = readme.slice(detail.index + detail[0].length);
  const colonAt = listTail.indexOf('：');
  const periodAt = colonAt === -1 ? -1 : listTail.indexOf('。', colonAt);
  const listSection = listTail.slice(colonAt + 1, periodAt === -1 ? listTail.length : periodAt);
  const listed = [...listSection.matchAll(/`([^`\n]+)`/g)]
    .map((m) => m[1])
    // 段落裡也會出現 `zh-TW`（基準）與 `frontend/src/i18n/translations/`，
    // 前者不在 translations/ 裡，後者含斜線 —— 兩者都要排除。
    .filter((id) => id !== 'zh-TW' && !id.includes('/'));
  const missingInReadme = declared.filter((id) => !listed.includes(id));
  const extraInReadme = listed.filter((id) => !declared.includes(id));
  if (missingInReadme.length) {
    readmeProblems.push(`README.md 的語系清單少了：${missingInReadme.join('、')}`);
  }
  if (extraInReadme.length) {
    readmeProblems.push(`README.md 的語系清單多了（目錄裡沒有）：${extraInReadme.join('、')}`);
  }
}

for (const p of readmeProblems) console.log(`BAD  README.md: ${p}`);

console.log('---');
console.log(
  `${files.length} catalog(s) checked, ${bad} with problems, ${order.length} keys each, ` +
    `${total} locale(s) total (含基準 zh-TW), ${readmeProblems.length} README mismatch(es)`,
);
process.exit(bad || readmeProblems.length ? 1 : 0);
