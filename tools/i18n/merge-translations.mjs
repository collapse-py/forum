// 把子代理寫回來的 TSV 譯文合併成 frontend/src/i18n/translations/<file>.ts。
//
// 完全以「鍵」為準，不在意分段。分段只是代理模型的輸入單位，對輸出格式沒有
// 影響 —— 這讓「某一段重跑」與「2 段改 4 段」都不需要考慮其他段。
//
// 這個腳本同時是流水線的品質關卡。四種失敗是實際發生過的，各自對應一段邏輯：
//
//   1. 少寫了行或整段沒回來      → 報 missing，並輸出 -todo.tsv 讓下一輪補。
//   2. 分隔符被寫成空格或句點    → 對照預期的鍵前綴修復（見 parseLine）。
//   3. 佔位符被丟掉或改名        → 報錯，不修復：猜測語意比壞掉更糟。
//   4. 整段原樣抄回中文          → 報錯（只比對漢字，全形 ｜ 與 （） 是刻意保留的；
//                                  中文與日文排除，因為漢字對它們是正常文字）。
import { readFileSync, writeFileSync, existsSync, readdirSync } from 'node:fs';

const PINNED = new Map([
  ['newPost.eyebrow', 'NEW POST'],
  ['profile.eyebrow', 'YOUR PROFILE'],
  ['publicProfile.eyebrow', 'PUBLIC PROFILE'],
  ['login.eyebrow', 'MEMBER ACCESS'],
  ['users.eyebrow', 'USER MANAGEMENT'],
  ['users.signinEyebrow', 'FORUM ADMIN'],
  ['posts.eyebrow', 'POST MODERATION'],
  ['reports.eyebrow', 'REPORT MODERATION'],
  ['admin.consoleName', 'Admin Console'],
  ['users.colId', 'ID'],
  ['common.placeholder', '—'],
]);

const [wlDir, lang, exportName, fileName] = process.argv.slice(2);

const messages = readFileSync('src/i18n/messages.ts', 'utf8');
const body = messages.slice(messages.indexOf('export const zhTW = {'));
const order = [...body.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':/gm)].map((m) => m[1]);
const srcValues = new Map(
  [...body.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':\s*\n?\s*'([\s\S]*?)',\r?\n/gm)].map((m) => [
    m[1],
    m[2].replace(/'\\\''/g, "'"),
  ]),
);
const ph = (s) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',');

/** 這一批要翻的鍵（不含必須原樣保留的）。 */
const todo = order.filter((key) => !PINNED.has(key));

/**
 * 切一行成 key 與 value。
 *
 * 正常是 tab 分隔。實際發生過的兩種損壞：tab 被換成一個空格（`error.requestStatus
 * Permintaan...`），以及 tab 被換成句點（`admin.railNavLabel.Funciones`）。兩種
 * 都能用「這一行的前綴應該是哪個鍵」推回來 —— 輸入檔的鍵是已知的，而且每段
 * 的行序是固定的，所以拿「第 i 行該有的鍵」去比對前綴就夠了。
 */
function parseLine(line, expectedKey) {
  const tab = line.indexOf('\t');
  if (tab >= 0) return [line.slice(0, tab), line.slice(tab + 1)];
  if (line.startsWith(`${expectedKey} `) || line.startsWith(`${expectedKey}.`)) {
    return [expectedKey, line.slice(expectedKey.length + 1)];
  }
  return null;
}

const translated = new Map();
const problems = [];
const duplicates = [];
const repaired = [];

// `-todo` 段排在最後。它是「上一輪缺什麼就補什麼」的產物，與同一個鍵的完整段
// 重疊時要讓完整段勝出：它來自翻過周遭文案的那個代理，用詞與檔案裡其他部分一致。
// 這不是罕見情況 —— 生成 todo 之後，原本缺的那一段補交了就會重疊。
const parts = readdirSync(wlDir)
  .filter((f) => f.startsWith(`${lang}-`) && f.endsWith('.out.tsv'))
  .sort(
    (a, b) =>
      (a.includes('-todo') ? 1 : 0) - (b.includes('-todo') ? 1 : 0) ||
      a.length - b.length ||
      a.localeCompare(b),
  );

for (const file of parts) {
  // file 本身已經帶著語言前綴（ko-1.out.tsv），不要再加一次 —— 那會變成
  // ko-ko-1.tsv，existsSync 回 false 而 expected 變成空陣列，於是所有鍵
  // 都被誤判成 missing。
  const srcFile = `${wlDir}/${file.replace('.out.tsv', '.tsv')}`;
  const expected = existsSync(srcFile)
    ? readFileSync(srcFile, 'utf8').trim().split('\n').map((l) => l.split('\t')[0])
    : [];
  const lines = readFileSync(`${wlDir}/${file}`, 'utf8')
    .replace(/\r\n/g, '\n')
    .replace(/\n$/, '')
    .split('\n');

  lines.forEach((line, i) => {
    const key = expected[i];
    if (!key) return;
    const parsed = parseLine(line, key);
    if (!parsed) {
      problems.push(`${file} line ${i + 1}: separator lost -> ${JSON.stringify(line.slice(0, 50))}`);
      return;
    }
    const [gotKey, value] = parsed;
    if (gotKey !== key) {
      problems.push(`${file} line ${i + 1}: key ${gotKey} != ${key}`);
      return;
    }
    if (translated.has(key)) {
      // 完整段先到就保留它；後到的 todo 段直接丟掉（見上面的排序說明）。
      duplicates.push(`${key} (${file})`);
      return;
    }
    if (value.trim() === '') {
      problems.push(`${key}: empty value`);
      return;
    }
    const want = ph(srcValues.get(key) ?? '');
    const got = ph(value);
    if (want !== got) {
      problems.push(`${key}: placeholder {${want}} != {${got}}`);
      return;
    }
    if (!line.includes('\t')) repaired.push(key);
    translated.set(key, value);
  });
}

// 未翻譯殘留：非中文語言的譯文裡不該還有漢字。整段原樣抄回去是實際發生過的失敗。
// 只比對漢字：全形直線 ｜ 與全形括號 （） 是這個專案刻意保留的（title.* 的分隔符、
// reports.targetGone 的括號），算進去會讓所有語言都被誤報。
//
// 排除清單必須與 verify-catalogs.mjs 的 HAN_IS_FOREIGN 一致：中文（本身以漢字
// 書寫）與日文（漢字是它的正式文字之一）都不能用「還有漢字」當成未翻的訊號。
// 曾經這裡只排除 zh-*，於是 ja 有 714/839 筆被誤判，合併永遠 exit 1 ——
// 而那時工具給出的指引是「這些鍵還沒翻」，與事實完全相反。
const HAN = /[\u3400-\u4DBF\u4E00-\u9FFF\uF900-\uFAFF]/;
if (!/^(zh-(TW|HK|MO|CN)|ja)$/.test(lang)) {
  for (const [key, value] of translated) {
    if (HAN.test(value)) problems.push(`${key}: still contains Chinese -> ${JSON.stringify(value.slice(0, 40))}`);
  }
}

const missing = todo.filter((key) => !translated.has(key));

// 沒收齊就不寫檔：寧可留著舊的完整目錄，也不要用一份缺了 200 條的目錄讓編譯過。
if (missing.length > 0) {
  writeFileSync(
    `${wlDir}/${lang}-todo.tsv`,
    missing.map((key) => `${key}\t${srcValues.get(key)}`).join('\n') + '\n',
    'utf8',
  );
  console.log(`${lang}: INCOMPLETE — ${missing.length} key(s) still missing, ${problems.length} other problem(s)`);
  console.log(`  wrote ${lang}-todo.tsv for a follow-up round`);
  for (const p of problems.slice(0, 8)) console.log('  ' + p);
  process.exit(1);
}

if (problems.length > 0) {
  console.log(`${lang}: ${problems.length} problem(s) on keys that ARE present:`);
  for (const p of problems.slice(0, 8)) console.log('  ' + p);
  process.exit(1);
}

const escape = (s) => s.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
const lines = order.map((key) => {
  const value = PINNED.has(key) ? PINNED.get(key) : translated.get(key);
  return `  '${key}':\n    '${escape(value)}',`;
});

writeFileSync(
  `src/i18n/translations/${fileName}`,
  `/*
 * ${lang} catalog (src/i18n/translations/${fileName})
 *
 * The Traditional Chinese source lives in ../messages.ts, which is the type
 * baseline; this file holds only strings. See that file's header for the rules.
 *
 * The 11 typographic-accent / product-name keys are intentionally identical to
 * the zh-TW source; everything else is a translation.
 */

import type { MessageKey } from '../messages';

export const ${exportName}: Record<MessageKey, string> = {
${lines.join('\n')}
};
`,
  'utf8',
);
console.log(
  `${lang}: OK -> src/i18n/translations/${fileName} (${order.length} entries` +
    `${repaired.length ? `, ${repaired.length} separator repaired` : ''}` +
    `${duplicates.length ? `, ${duplicates.length} duplicate keys dropped` : ''})`,
);
