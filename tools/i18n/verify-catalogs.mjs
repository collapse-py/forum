// 驗證所有翻譯目錄：鍵集、順序、佔位符、未翻譯殘留。
//
// 目錄有兩種排版 —— 早期那批是一行一筆
//   'common.cancel': '取消',
// 新的（由 merge 腳本產生）是鍵與值分兩行
//   'common.cancel':
//     '取消',
// 因此比對的正則必須兩種都吃，否則會把整個檔案判成「缺 393 鍵」。
import { readFileSync, readdirSync } from 'node:fs';

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

const src = readFileSync('src/i18n/messages.ts', 'utf8');
const body = src.slice(src.indexOf('export const zhTW = {'));
const order = [...body.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':/gm)].map((m) => m[1]);
const srcVals = new Map(
  [...body.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':\s*\n?\s*'([\s\S]*?)',\r?\n/gm)].map((m) => [m[1], m[2]]),
);

const files = readdirSync('src/i18n/translations').filter((f) => f.endsWith('.ts')).sort();
let bad = 0;

for (const file of files) {
  const text = readFileSync(`src/i18n/translations/${file}`, 'utf8');
  const entries = [...text.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':\s*\n?\s*'([\s\S]*?)',\r?\n/gm)];
  const keys = entries.map((m) => m[1]);
  const vals = new Map(entries.map((m) => [m[1], m[2]]));

  const missing = order.filter((k) => !keys.includes(k));
  const extra = keys.filter((k) => !order.includes(k));
  const badPh = [];
  const untranslated = [];
  for (const k of keys) {
    if (PIN.includes(k)) continue;
    if (ph(srcVals.get(k) ?? '') !== ph(vals.get(k))) badPh.push(k);
    if (HAN_IS_FOREIGN(file) && HAN.test(vals.get(k))) untranslated.push(k);
  }
  const orderOk = JSON.stringify(keys) === JSON.stringify(order);
  const ok = !missing.length && !extra.length && !badPh.length && !untranslated.length && orderOk;
  if (!ok) bad += 1;
  console.log(
    `${ok ? 'OK  ' : 'BAD '}${file.padEnd(12)}${String(keys.length).padStart(4)}` +
      (missing.length ? ` missing:${missing.length}` : '') +
      (extra.length ? ` extra:${extra.length}` : '') +
      (orderOk ? '' : ' order:differs') +
      (badPh.length ? ` placeholder:${badPh.join(',')}` : '') +
      (untranslated.length ? ` untranslated:${untranslated.join(',')}` : ''),
  );
}

console.log('---');
console.log(`${files.length} catalog(s) checked, ${bad} with problems, ${order.length} keys each`);
process.exit(bad ? 1 : 0);
