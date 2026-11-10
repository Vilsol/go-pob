// The TinyGo build against the standard Go build: every build's outputs, compared bit for bit.
//   node harness/engine_diff.mjs <poe1|poe2> [--jobs N] [--sample N] [--keep DIR] <xml or dir>...
// runs each engine in its own processes (`run` below) and diffs what they wrote; exits 1 on any difference, 2 when it
// cannot run. Game data comes from the Go bundle cache, as the e2e model reads it, and from the network past it.
import { spawn } from 'node:child_process';
import { createReadStream, existsSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { createInterface } from 'node:readline';
import { DATA_HOST, cachedAsset } from '../e2e/support/data_cache.ts';

const HERE = import.meta.dirname;
const STATIC = resolve(HERE, '../static');
const GAMES = {
  poe1: { stem: 'go-pob', ns: 'go-pob', types: '../src/lib/types/index.js' },
  poe2: { stem: 'go-pob2', ns: 'go-pob2', types: '../src/lib/types2/index.js' }
};
const ENGINES = {
  go: { wasm: (g) => `${STATIC}/${g.stem}.wasm`, exec: '../src/wasm_exec.js', build: 'mise run build-go / build-go2' },
  tinygo: { wasm: (g) => `${STATIC}/${g.stem}-tiny.wasm`, exec: '../src/wasm_exec_tinygo.js', build: 'mise run build-go-tiny / build-go2-tiny' }
};

const refuse = (msg) => {
  console.error(`REFUSING: ${msg}`);
  process.exit(2);
};

// numbers JSON cannot carry exactly (NaN, ±Inf, -0) keep their identity
const canonical = (v) =>
  JSON.stringify(v, (_k, x) => (typeof x === 'number' && (!Number.isFinite(x) || Object.is(x, -0)) ? `#${Object.is(x, -0) ? '-0' : String(x)}` : x));

const sorted = (o) => (o ? Object.fromEntries(Object.entries(o).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))) : undefined);

const boot = async (gameName, engineName) => {
  const game = GAMES[gameName];
  const engine = ENGINES[engineName];
  const wasmPath = engine.wasm(game);
  if (!existsSync(wasmPath)) refuse(`${wasmPath} is missing — ${engine.build}`);
  await import(resolve(HERE, engine.exec));
  if (!booted) patchIo();
  booted = true;
  // a program that exited (a panic) leaves its namespace behind
  delete globalThis.go?.[game.ns];
  const go = new globalThis.Go();
  const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject);
  void go.run(instance);
  while (globalThis.go?.[game.ns] === undefined) await new Promise((r) => setImmediate(r));
  const mod = await import(resolve(HERE, game.types));
  mod.initializeCrystalline();
  mod.config.InitLogging(false);
  if (gameName === 'poe1') {
    const res = await mod.raw.InitializeAll(mod.raw.GameDataVersion(), () => undefined);
    if (!res.ok) throw new Error(`game data: ${res.error}`);
  } else {
    const res = await mod.exposition.Preload(() => {});
    if (res && res.ok === false) throw new Error(`game data: ${res.error}`);
  }
  return { go, mod };
};

let booted = false;
// the last Go output, so a build that panics says why
let goOutput = '';
const patchIo = () => {
  const decoder = new TextDecoder();
  if (process.env.GO_LOG !== '1')
    globalThis.fs.writeSync = (_fd, buf) => {
      goOutput = (goOutput + decoder.decode(buf)).slice(-20000);
      return buf.length;
    };
  const realFetch = globalThis.fetch;
  globalThis.fetch = async (input, init) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
    if (!url.startsWith(DATA_HOST)) return realFetch(input, init);
    let asset;
    try {
      asset = cachedAsset(url);
    } catch {
      return realFetch(input, init);
    }
    return asset.status === 200 ? new Response(new Uint8Array(asset.body)) : new Response(null, { status: 404 });
  };
};

const outputsOf = async (gameName, mod, xml) => {
  if (gameName === 'poe1') {
    const build = mod.builds.ParseBuildStr(xml).unwrap();
    const calc = mod.calculator.NewCalculator(build);
    const env = await calc.BuildOutput('MAIN');
    const player = env?.Player;
    const minion = env?.Minion?.Actor;
    try {
      return {
        Output: sorted(player?.Output),
        OutputTable: sorted(player?.OutputTable),
        MinionOutput: sorted(minion?.Output),
        MinionOutputTable: sorted(minion?.OutputTable)
      };
    } finally {
      // every Go object a build made, or a long run holds them all and runs out of memory
      for (const o of [minion, env?.Minion, player, env, calc, build]) o?.release?.();
    }
  }
  const calc = mod.exposition.ParseBuild(xml).unwrap();
  try {
    const r = (await calc.Calculate()).unwrap();
    return { Output: sorted(r?.Output), Minion: sorted(r?.Minion) };
  } finally {
    calc.release();
  }
};

// one engine over a list of builds: one JSON line per build, in list order
const run = async (gameName, engineName, out, list) => {
  // an exited program's pending callbacks throw from the event loop; the build that panicked is already recorded
  process.on('uncaughtException', (e) => {
    if (e?.message !== 'Go program has already exited') throw e;
  });
  let { go, mod } = await boot(gameName, engineName);
  const lines = [];
  for (const file of readFileSync(list, 'utf8').split('\n').filter(Boolean)) {
    let line;
    let trapped = false;
    try {
      const outputs = await outputsOf(gameName, mod, readFileSync(file, 'utf8'));
      // the control: ENGINE_DIFF_PERTURB=1 moves one TinyGo value by one part in 1e15, which the diff must report
      if (process.env.ENGINE_DIFF_PERTURB === '1' && engineName === 'tinygo' && lines.length === 0) {
        const k = Object.keys(outputs.Output ?? {}).find((n) => outputs.Output[n] !== 0 && Number.isFinite(outputs.Output[n]));
        if (k) outputs.Output[k] *= 1 + 1e-15;
      }
      line = canonical({ build: file, ...outputs });
    } catch (e) {
      trapped = e instanceof WebAssembly.RuntimeError;
      line = canonical({ build: file, error: String(e?.message ?? e) });
    }
    // a panic ends a standard Go program and traps a TinyGo one: the error is this build's line, and the next build
    // gets a fresh engine
    if (go.exited || trapped) {
      const why = goOutput.split('\n').find((l) => /^(panic|fatal error)/.test(l)) ?? goOutput.trim().split('\n')[0];
      line = canonical({ build: file, error: `the program ended: ${go.exited ? why : JSON.parse(line).error}` });
      goOutput = '';
      ({ go, mod } = await boot(gameName, engineName));
    }
    lines.push(line);
  }
  writeFileSync(out, lines.join('\n') + '\n');
  process.exit(0);
};

const child = (args) =>
  new Promise((res, rej) => {
    const p = spawn(process.execPath, [import.meta.filename, 'run', ...args], { stdio: ['ignore', 'inherit', 'inherit'] });
    p.on('exit', (code) => (code === 0 ? res() : rej(new Error(`${args.join(' ')}: exit ${code}`))));
  });

const lines = (file) => createInterface({ input: createReadStream(file), crlfDelay: Infinity })[Symbol.asyncIterator]();

const diffValues = (a, b, path, out) => {
  if (a === b) return;
  if (typeof a === 'object' && typeof b === 'object' && a && b) {
    for (const k of new Set([...Object.keys(a), ...Object.keys(b)])) diffValues(a[k], b[k], `${path}.${k}`, out);
    return;
  }
  out.push(`${path}: go=${JSON.stringify(a)} tinygo=${JSON.stringify(b)}`);
};

const main = async () => {
  const [gameName, ...rest] = process.argv.slice(2);
  if (!GAMES[gameName]) refuse(`first argument is the game, poe1 or poe2; got ${gameName}`);
  let jobs = 2;
  let sample = 0;
  let keep;
  const inputs = [];
  for (let i = 0; i < rest.length; i++) {
    const a = rest[i];
    if (a === '--jobs') jobs = Number(rest[++i]);
    else if (a === '--sample') sample = Number(rest[++i]);
    else if (a === '--keep') keep = rest[++i];
    else if (a.startsWith('--')) refuse(`unknown flag ${a}`);
    else inputs.push(a);
  }
  if (!(jobs >= 1) || !(sample >= 0)) refuse('--jobs and --sample take a positive number');
  let files = inputs.flatMap((p) =>
    statSync(p).isDirectory()
      ? readdirSync(p)
          .filter((f) => f.endsWith('.xml'))
          .sort()
          .map((f) => resolve(p, f))
      : [resolve(p)]
  );
  if (files.length === 0) refuse('no build XMLs given');
  // an even spread over the sorted list, so a sample is the same builds every run
  if (sample > 0 && sample < files.length) files = Array.from({ length: sample }, (_, i) => files[Math.floor((i * files.length) / sample)]);

  const dir = keep ?? mkdtempSync(join(tmpdir(), 'engine-diff-'));
  const shards = Array.from({ length: Math.min(jobs, files.length) }, (_, s) => files.filter((_f, i) => i % jobs === s));
  shards.forEach((s, i) => writeFileSync(`${dir}/list-${i}.txt`, s.join('\n') + '\n'));
  const start = Date.now();
  for (const engine of ['go', 'tinygo']) {
    const t = Date.now();
    await Promise.all(shards.map((_s, i) => child([gameName, engine, `${dir}/${engine}-${i}.jsonl`, `${dir}/list-${i}.txt`])));
    console.log(`${engine}: ${files.length} builds in ${((Date.now() - t) / 1000).toFixed(0)} s`);
  }

  let builds = 0;
  let values = 0;
  let errors = 0;
  const diffs = [];
  const errorSamples = [];
  for (let i = 0; i < shards.length; i++) {
    const a = lines(`${dir}/go-${i}.jsonl`);
    const b = lines(`${dir}/tinygo-${i}.jsonl`);
    for (;;) {
      const [x, y] = await Promise.all([a.next(), b.next()]);
      if (x.done || y.done) {
        if (!x.done || !y.done) diffs.push(`shard ${i}: the engines wrote a different number of builds`);
        break;
      }
      builds++;
      const p = JSON.parse(x.value);
      const q = JSON.parse(y.value);
      if (p.error || q.error) {
        errors++;
        if (errorSamples.length < 5) errorSamples.push(`${p.build.split('/').pop()}: go=${p.error ?? 'ok'} tinygo=${q.error ?? 'ok'}`);
      }
      for (const k of ['Output', 'OutputTable', 'MinionOutput', 'MinionOutputTable', 'Minion']) values += Object.keys(p[k] ?? {}).length;
      if (x.value !== y.value) {
        const found = [];
        diffValues(p, q, p.build.split('/').pop(), found);
        diffs.push(...found);
      }
    }
  }
  if (!keep) rmSync(dir, { recursive: true, force: true });
  console.log(
    `${gameName}: ${builds} builds, ${values} values, ${errors} builds that failed on an engine, ${diffs.length} differences (${((Date.now() - start) / 1000).toFixed(0)} s)`
  );
  for (const e of errorSamples) console.log(`  failed: ${e}`);
  for (const d of diffs.slice(0, 50)) console.log(`  ${d}`);
  if (diffs.length > 50) console.log(`  ... ${diffs.length - 50} more`);
  if (builds === 0) refuse('compared no builds');
  process.exit(diffs.length > 0 ? 1 : 0);
};

if (process.argv[2] === 'run') {
  const [, gameName, engineName, out, list] = process.argv.slice(2);
  await run(gameName, engineName, out, list);
} else {
  await main();
}
