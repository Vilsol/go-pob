// Records which source files each e2e spec runs, for `mise run e2e-affected`. A coverage build (vite.coverage.config.js)
// gives every function in frontend/src a call that marks its file as run; every realm (the page and each worker) posts
// what it ran to the preview server, which files it under the spec playing at the time. One preview server per runner
// and one spec at a time per runner is what makes the attribution exact: no spec runs beside another on its server.
//
//   node e2e/coverage.mjs record [--servers K] [--workers W] [--part i/n] [spec ...]  → e2e/.coverage/part-<i>.json
//   node e2e/coverage.mjs merge [part.json ...]                                     → e2e/coverage-map.json
//   node e2e/coverage.mjs specs [spec ...]: refuses a flag or a missing spec before the coverage build starts
import { execFileSync, spawn } from 'node:child_process';
import { appendFileSync, existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { createServer } from 'node:net';
import { join, relative, resolve } from 'node:path';
import { parseAst } from 'vite';

const FRONTEND = resolve(import.meta.dirname, '..');
const REPO = resolve(FRONTEND, '..');
const SRC = join(FRONTEND, 'src');
export const OUT = join(FRONTEND, 'e2e', '.coverage');
export const FILES = join(OUT, 'files.json');
export const MAP = join(FRONTEND, 'e2e', 'coverage-map.json');
const ENDPOINT = '/__e2e_cov';

// one shared recorder per realm, created by whichever instrumented module evaluates first; it posts the whole set
// whenever it grew, so a lost post costs nothing the next one does not carry
const RUNTIME = `;(globalThis.__e2eCov||(globalThis.__e2eCov=(()=>{const h=new Set(),l=new Set();let n=-1;const u=new URL(${JSON.stringify(ENDPOINT)},location.origin).href;const f=()=>{const k=h.size*100003+l.size;if(k===n)return;n=k;try{fetch(u,{method:'POST',body:JSON.stringify({h:[...h],l:[...l]}),keepalive:true}).catch(()=>{})}catch{}};setInterval(f,500);if(typeof document!=='undefined')addEventListener('pagehide',f);return{h:(p)=>h.add(p),l:(p)=>l.add(p)}})())).l(%P%);`;

const FUNCTION = new Set(['FunctionDeclaration', 'FunctionExpression', 'ArrowFunctionExpression']);

// the offsets to insert at, for every function in `ast`
const functionsIn = (ast) => {
  const out = [];
  const visit = (node) => {
    if (!node || typeof node.type !== 'string') return;
    if (FUNCTION.has(node.type)) out.push(node);
    for (const k in node) {
      const v = node[k];
      if (Array.isArray(v)) for (const c of v) visit(c);
      else if (v && typeof v === 'object' && k !== 'loc') visit(v);
    }
  };
  visit(ast);
  return out;
};

// `code` with a hit on entry to each function and a load mark at the top, and how many functions it has
export const instrument = (code, path) => {
  const p = JSON.stringify(path);
  const hit = `globalThis.__e2eCov?.h(${p});`;
  const fns = functionsIn(parseAst(code));
  const edits = [{ at: 0, text: RUNTIME.replace('%P%', p) }];
  for (const fn of fns) {
    if (fn.body.type === 'BlockStatement') edits.push({ at: fn.body.start + 1, text: hit });
    else edits.push({ at: fn.body.start, text: `(${hit.slice(0, -1)},` }, { at: fn.body.end, text: ')' });
  }
  edits.sort((a, b) => b.at - a.at);
  let out = code;
  for (const e of edits) out = out.slice(0, e.at) + e.text + out.slice(e.at);
  return { code: out, functions: fns.length };
};

const files = new Map();

// the Vite plugin: instruments the browser build (never SSR) and, in preview, files each post under the current spec
export const coveragePlugin = () => ({
  name: 'e2e-coverage',
  enforce: 'post',
  transform(code, id, options) {
    if (options?.ssr) return;
    const file = id.replace(/\?.*$/, '');
    if (!file.startsWith(SRC) || id.includes('?') || !/\.(ts|js|mjs|svelte)$/.test(file) || /\.test\.ts$/.test(file)) return;
    const path = relative(REPO, file);
    const r = instrument(code, path);
    files.set(path, r.functions);
    return { code: r.code, map: null };
  },
  buildEnd() {
    if (files.size === 0) return;
    mkdirSync(OUT, { recursive: true });
    const all = existsSync(FILES) ? JSON.parse(readFileSync(FILES, 'utf8')) : {};
    for (const [k, v] of files) all[k] = v;
    writeFileSync(FILES, JSON.stringify(all, Object.keys(all).sort(), 1));
  },
  configurePreviewServer(server) {
    const sink = process.env.E2E_COVERAGE_OUT;
    if (!sink) return;
    let spec = '';
    server.middlewares.use((req, res, next) => {
      if (!req.url?.startsWith(ENDPOINT)) return next();
      const url = new URL(req.url, 'http://x');
      if (url.pathname === `${ENDPOINT}/spec`) {
        spec = url.searchParams.get('name') ?? '';
        res.statusCode = 204;
        return res.end();
      }
      let body = '';
      req.on('data', (c) => (body += c));
      req.on('end', () => {
        if (spec) {
          try {
            const { h = [], l = [] } = JSON.parse(body);
            appendFileSync(sink, `${JSON.stringify({ spec, h, l })}\n`);
          } catch {
            // a truncated keepalive post; the next one carries the same set
          }
        }
        res.statusCode = 204;
        res.end();
      });
    });
  }
});

const freePort = () =>
  new Promise((ok, fail) => {
    const s = createServer();
    s.once('error', fail);
    s.listen(0, '127.0.0.1', () => {
      const { port } = s.address();
      s.close(() => ok(port));
    });
  });

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const run = (cmd, args, env) =>
  new Promise((ok) => {
    const child = spawn(cmd, args, { cwd: FRONTEND, env: { ...process.env, ...env }, stdio: ['ignore', 'ignore', 'inherit'] });
    child.on('exit', (code) => ok(code ?? 1));
  });

const startServer = async (sink) => {
  const port = await freePort();
  const child = spawn('npx', ['vite', 'preview', '-c', 'e2e/vite.coverage.config.js', '--port', String(port), '--strictPort'], {
    cwd: FRONTEND,
    env: { ...process.env, E2E_COVERAGE_OUT: sink },
    stdio: 'ignore',
    detached: true
  });
  for (const deadline = Date.now() + 60_000; Date.now() < deadline; await sleep(250)) {
    try {
      if ((await fetch(`http://localhost:${port}/`)).ok) return { port, child };
    } catch {
      // not up yet
    }
  }
  process.kill(-child.pid);
  throw new Error(`REFUSING: the coverage preview server did not answer on ${port} within 60s`);
};

const allSpecs = () =>
  readdirSync(join(FRONTEND, 'e2e'))
    .filter((n) => n.endsWith('.spec.ts'))
    .sort()
    .map((n) => `e2e/${n}`);

const record = async (argv) => {
  let servers = 2;
  let workers = 2;
  let part = '1/1';
  const specs = [];
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--servers') servers = Number(argv[++i]);
    else if (a === '--workers') workers = Number(argv[++i]);
    else if (a === '--part') part = argv[++i];
    else if (a.startsWith('--')) throw new Error(`REFUSING: unknown flag ${a}`);
    else specs.push(a.replace(/^frontend\//, ''));
  }
  const [pi, pn] = part.split('/').map(Number);
  if (!(pi >= 1 && pi <= pn) || !(servers >= 1) || !(workers >= 1)) throw new Error(`REFUSING: bad --part/--servers/--workers`);
  if (!existsSync(FILES))
    throw new Error(`REFUSING: ${relative(FRONTEND, FILES)} is missing; build with \`npx vite build -c e2e/vite.coverage.config.js\` first`);
  const todo = (specs.length ? specs : allSpecs()).filter((_, i) => i % pn === pi - 1);
  mkdirSync(OUT, { recursive: true });
  const queue = [...todo];
  const failed = [];
  const serverRun = async (k) => {
    const sink = join(OUT, `server-${pi}-${k}.jsonl`);
    rmSync(sink, { force: true });
    const { port, child } = await startServer(sink);
    try {
      for (let spec; (spec = queue.shift()); ) {
        await fetch(`http://localhost:${port}${ENDPOINT}/spec?name=${encodeURIComponent(spec)}`);
        const t = Date.now();
        const code = await run('npx', ['playwright', 'test', '-c', 'e2e/playwright.config.ts', '--reporter=dot', spec], {
          E2E_REUSE: '1',
          E2E_PORT: String(port),
          E2E_WORKERS: String(workers)
        });
        // the last posts of a closing page land after the run exits
        await sleep(1500);
        console.error(`coverage: ${spec} ${code === 0 ? 'passed' : `exited ${code}`} in ${((Date.now() - t) / 1000).toFixed(0)}s`);
        if (code !== 0) failed.push(spec);
      }
      await fetch(`http://localhost:${port}${ENDPOINT}/spec?name=`);
    } finally {
      process.kill(-child.pid);
    }
    return sink;
  };
  const sinks = await Promise.all(Array.from({ length: Math.min(servers, todo.length) }, (_, k) => serverRun(k)));
  const reached = {};
  for (const sink of sinks) {
    if (!existsSync(sink)) continue;
    for (const line of readFileSync(sink, 'utf8').split('\n').filter(Boolean)) {
      const { spec, h, l } = JSON.parse(line);
      const r = (reached[spec] ??= { h: new Set(), l: new Set() });
      for (const f of h) r.h.add(f);
      for (const f of l) r.l.add(f);
    }
  }
  const silent = todo.filter((s) => !reached[s]);
  const out = {
    commit: execFileSync('git', ['-C', REPO, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(),
    files: JSON.parse(readFileSync(FILES, 'utf8')),
    failed,
    specs: Object.fromEntries(Object.entries(reached).map(([s, r]) => [s, { hit: [...r.h].sort(), loaded: [...r.l].sort() }]))
  };
  const dest = join(OUT, `part-${pi}.json`);
  writeFileSync(dest, JSON.stringify(out));
  console.error(`coverage: ${Object.keys(out.specs).length} of ${todo.length} specs recorded into ${relative(FRONTEND, dest)}`);
  if (silent.length) console.error(`coverage: NOTHING ARRIVED from ${silent.join(' ')}; e2e-affected maps them by imports`);
  if (failed.length) console.error(`coverage: failing specs (recorded anyway): ${failed.join(' ')}`);
  return 0;
};

// the committed map: per spec, the files it ran a function of, plus the function-less files it loaded
export const compact = (parts) => {
  const files = {};
  const specs = {};
  let commit = '';
  for (const p of parts) {
    commit ||= p.commit;
    Object.assign(files, p.files);
    for (const [s, r] of Object.entries(p.specs)) specs[s] = [...new Set([...r.hit, ...r.loaded.filter((f) => p.files[f] === 0)])].sort();
  }
  const names = Object.keys(files).sort();
  const index = new Map(names.map((n, i) => [n, i]));
  return {
    note: 'Which frontend/src files each e2e spec runs, recorded by `mise run e2e-coverage`; read by frontend/e2e/affected.mjs. Regenerate, never edit.',
    commit,
    files: names,
    specs: Object.fromEntries(
      Object.keys(specs)
        .sort()
        .map((s) => [s, specs[s].filter((f) => index.has(f)).map((f) => index.get(f))])
    )
  };
};

const merge = (argv) => {
  const paths = argv.length
    ? argv
    : readdirSync(OUT)
        .filter((n) => /^part-\d+\.json$/.test(n))
        .map((n) => join(OUT, n));
  if (paths.length === 0) throw new Error(`REFUSING: no part-*.json in ${relative(FRONTEND, OUT)}; run \`node e2e/coverage.mjs record\` first`);
  const map = compact(paths.map((p) => JSON.parse(readFileSync(p, 'utf8'))));
  writeFileSync(MAP, `${JSON.stringify(map)}\n`);
  console.error(`coverage: ${Object.keys(map.specs).length} specs over ${map.files.length} files into ${relative(FRONTEND, MAP)}`);
  return 0;
};

const specs = (argv) => {
  for (const a of argv) {
    if (a.startsWith('-')) throw new Error(`REFUSING: ${a} is a flag, not a spec; mise run e2e-coverage takes --servers and --workers`);
    if (!existsSync(join(FRONTEND, a.replace(/^frontend\//, '')))) throw new Error(`REFUSING: no spec file ${a}`);
  }
  return 0;
};

if (import.meta.url === `file://${process.argv[1]}`) {
  const [cmd, ...args] = process.argv.slice(2);
  const cmds = { record, merge, specs };
  if (!cmds[cmd]) {
    console.error(
      'REFUSING: usage: node e2e/coverage.mjs record [--servers K] [--workers W] [--part i/n] [spec ...] | merge [part.json ...] | specs [spec ...]'
    );
    process.exit(64);
  }
  try {
    process.exit(await cmds[cmd](args));
  } catch (e) {
    console.error(e.message);
    process.exit(3);
  }
}
