import { createHash } from 'node:crypto';
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { brotliCompressSync, constants } from 'node:zlib';

// node compress-wasm.mjs [name.wasm ...]; go-pob.wasm when none is named
const names = process.argv.slice(2);
for (const name of names.length > 0 ? names : ['go-pob.wasm']) {
  if (!/^[\w-]+\.wasm$/.test(name)) {
    throw new Error(`REFUSING: ${name} is not a static/*.wasm file name`);
  }
  const src = resolve(import.meta.dirname, '../static', name);
  const wasm = readFileSync(src);
  if (wasm.readUInt32BE(0) !== 0x0061736d) {
    throw new Error(`REFUSING: ${src} is not a wasm module`);
  }
  const hash = createHash('sha256').update(wasm).digest('hex');
  if (existsSync(`${src}.br`) && existsSync(`${src}.br.sha256`) && readFileSync(`${src}.br.sha256`, 'utf8') === hash) {
    console.log(`${name}.br is up to date`);
    continue;
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
  console.log(`${name} ${wasm.length} -> ${name}.br ${br.length}`);
}
