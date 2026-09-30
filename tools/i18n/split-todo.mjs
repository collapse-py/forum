// 把 merge 產生的 <lang>-todo.tsv 再切成幾份，讓代理模型的單次回應不會太大。
// 已知的失敗點是「行數太多」，不是內容太複雜，所以這裡只做等分。
import { readFileSync, writeFileSync } from 'node:fs';

const [dir, lang, partsArg] = process.argv.slice(2);
const src = readFileSync(`${dir}/${lang}-todo.tsv`, 'utf8').trim().split('\n');
const parts = Number(partsArg ?? 4);
const per = Math.ceil(src.length / parts);
const sizes = [];
for (let i = 0; i < parts; i += 1) {
  const slice = src.slice(i * per, (i + 1) * per);
  if (slice.length === 0) continue;
  writeFileSync(`${dir}/${lang}-fix${i + 1}.tsv`, slice.join('\n') + '\n', 'utf8');
  sizes.push(slice.length);
}
console.log(`${lang}-todo (${src.length} keys) -> ${parts} fix files: ${sizes.join(' + ')}`);
