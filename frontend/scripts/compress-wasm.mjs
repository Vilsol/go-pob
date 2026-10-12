import { createHash } from 'node:crypto';
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { brotliCompressSync, constants } from 'node:zlib';

const src = resolve(import.meta.dirname, '../static/go-pob.wasm');
const wasm = readFileSync(src);
if (wasm.readUInt32BE(0) !== 0x0061736d) {
  throw new Error(`REFUSING: ${src} is not a wasm module`);
}
const hash = createHash('sha256').update(wasm).digest('hex');
if (existsSync(`${src}.br`) && existsSync(`${src}.br.sha256`) && readFileSync(`${src}.br.sha256`, 'utf8') === hash) {
  console.log('go-pob.wasm.br is up to date');
  process.exit(0);
}
const br = brotliCompressSync(wasm, {
  params: {
    [constants.BROTLI_PARAM_QUALITY]: 11,
    [constants.BROTLI_PARAM_LGWIN]: 24,
    [constants.BROTLI_PARAM_SIZE_HINT]: wasm.length
  }
});
writeFileSync(`${src}.br`, br);
writeFileSync(`${src}.br.sha256`, hash);
console.log(`go-pob.wasm ${wasm.length} -> go-pob.wasm.br ${br.length}`);
