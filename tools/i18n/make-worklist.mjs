// 把 messages.ts 拆成給子代理用的純文字工作單。
//
// 為什麼是純文字而不是 TS：本地代理模型的 context 很小。讓它讀 messages.ts
// （25KB 的 TS）再寫出 393 筆 TS 字串，單次回應就會溢出；改成「讀一份
// key<TAB>來源 的小清單，回一份 key<TAB>譯文 的小清單」之後，代理模型唯一要做的
// 事就是翻譯 —— 不碰引號跳脫、結尾逗號、型別，也不用跑任何驗證。TS 檔案由 merge
// 腳本機械式地生成。
//
// 分成幾份由實測決定：192 行仍然會溢出，因此預設四份（約 96 行）。線性分段的
// 失敗點是行太多，不是內容太複雜。
//
// 必須原樣保留的鍵（拉丁字母排版裝飾與產品名）不放進工作單，改由 merge 直接填入
// 原值 —— 少一類失敗原因，而且這些鍵本來就不該翻。
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';

export const PINNED = [
  'newPost.eyebrow',
  'profile.eyebrow',
  'publicProfile.eyebrow',
  'login.eyebrow',
  'users.eyebrow',
  'users.signinEyebrow',
  'posts.eyebrow',
  'reports.eyebrow',
  'admin.consoleName',
  'users.colId',
  'common.placeholder',
];

const messages = readFileSync('src/i18n/messages.ts', 'utf8');
const body = messages.slice(messages.indexOf('export const zhTW = {'));
const all = [...body.matchAll(/^ {2}'([a-zA-Z0-9_.]+)':\s*\n?\s*'([\s\S]*?)',\r?\n/gm)].map(
  (m) => [m[1], m[2].replace(/'\\\''/g, "'")],
);

if (all.length !== 393) {
  console.error(`FATAL: parsed ${all.length} entries, expected 393`);
  process.exit(1);
}

const entries = all.filter(([key]) => !PINNED.includes(key));
const [outDir, lang, partsArg] = process.argv.slice(2);
const parts = Number(partsArg ?? 4);
mkdirSync(outDir, { recursive: true });

const per = Math.ceil(entries.length / parts);
const sizes = [];
for (let i = 0; i < parts; i += 1) {
  const slice = entries.slice(i * per, (i + 1) * per);
  if (slice.length === 0) continue;
  writeFileSync(
    `${outDir}/${lang}-${i + 1}.tsv`,
    slice.map(([key, value]) => `${key}\t${value}`).join('\n') + '\n',
    'utf8',
  );
  sizes.push(slice.length);
}
writeFileSync(`${outDir}/${lang}.meta.json`, JSON.stringify({ parts, sizes, total: entries.length }), 'utf8');

console.log(`${lang}: ${all.length} total, ${entries.length} to translate, ${parts} parts: ${sizes.join(' + ')}`);
