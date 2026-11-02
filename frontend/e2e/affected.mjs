// Picks the e2e specs a change can reach, for `mise run e2e-affected`. A spec is affected when it ran a function of the
// changed file in the recorded coverage map (coverage-map.json, `mise run e2e-coverage`), or, for a spec or file the
// map does not know, when the file is in the import closure of the spec itself, of the support files it uses, of a
// route it visits, or of the app shell (the root layout, which every test renders). A Go file reaches the specs that
// boot the wasm whose packages include it. A change to the e2e harness or the build reaches every spec. A selection
// over E2E_AFFECTED_MAX specs is refused loudly and narrowed to the change's own specs, since the operator runs the
// whole suite at integration; a change nothing maps to selects only the @smoke tests and is listed as unmapped.
import { execFileSync } from 'node:child_process';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';

const FRONTEND = resolve(import.meta.dirname, '..');
const REPO = resolve(FRONTEND, '..');
const E2E = join(FRONTEND, 'e2e');
const SRC = join(FRONTEND, 'src');

// the navigation link a spec clicks → the route directory it renders (src/lib/design/TopBar.svelte's tabs)
export const LINK_ROUTES = {
  Tree: 'tree',
  Skills: 'skills',
  Items: 'items',
  Calcs: 'calcs',
  Config: 'configuration',
  Party: 'party',
  Notes: 'notes',
  'Import / Export': 'import',
  Compare: 'compare'
};

// changes that reach every spec: the harness and the build
const EVERYTHING = [
  /^frontend\/e2e\/(support|fixtures)\//,
  /^frontend\/e2e\/[^/]+\.(ts|json|go)$/,
  /^frontend\/e2e\/(playwright\.config\.ts|serve\.mjs|shard\.mjs|vite\.coverage\.config\.js)$/,
  /^frontend\/(package\.json|pnpm-lock\.yaml|vite\.config\.(ts|js)|svelte\.config\.js|tsconfig\.json)$/,
  /^frontend\/static\//,
  /^frontend\/src\/(app\.html|app\.d\.ts|hooks\.server\.ts|service-worker\.ts|wasm_exec\.js)$/
];
const SPEC = /^frontend\/e2e\/[^/]+\.spec\.ts$/;
// not part of what a browser test runs
const INERT = [
  /\.test\.(ts|mjs)$/,
  /_test\.go$/,
  /\.stories\.svelte$/,
  /\.md$/,
  /^frontend\/e2e\/(affected|coverage)\.mjs$/,
  /^frontend\/e2e\/coverage-map\.json$/,
  /^frontend\/e2e\/durations\.json$/,
  /^frontend\/harness\//,
  /^frontend\/\.storybook\//,
  /^\.github\//,
  /^docs\//
];
// each wasm build's generated bindings: a spec that ran one ran that wasm's Go
export const WASM_BOOT = { wasm: 'frontend/src/lib/types/index.js', wasm2: 'frontend/src/lib/types2/index.js' };
export const MAX = Number(process.env.E2E_AFFECTED_MAX || 12);
const MAP = join(E2E, 'coverage-map.json');

const walk = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    if (name === 'node_modules' || name.startsWith('.')) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (/\.(ts|js|mjs|svelte)$/.test(name)) out.push(p);
  }
  return out;
};

const IMPORT =
  /(?:import|export)\s+(?:type\s+)?(?:[^'"`;]*?\sfrom\s+)?['"]([^'"]+)['"]|import\(\s*['"]([^'"]+)['"]\s*\)|new URL\(\s*['"]([^'"]+)['"],\s*import\.meta\.url\s*\)/g;

const resolveImport = (from, spec) => {
  let base;
  if (spec.startsWith('$lib/')) base = join(SRC, 'lib', spec.slice(5));
  else if (spec === '$lib') base = join(SRC, 'lib');
  else if (spec.startsWith('.')) base = resolve(dirname(from), spec);
  else return undefined;
  base = base.replace(/\?.*$/, '');
  for (const c of [base, base.replace(/\.js$/, '.ts'), `${base}.ts`, `${base}.js`, `${base}.svelte`, join(base, 'index.ts'), join(base, 'index.js')]) {
    if (existsSync(c) && statSync(c).isFile()) return c;
  }
  return undefined;
};

const graph = new Map();
const importsOf = (file) => {
  if (!graph.has(file)) {
    const text = readFileSync(file, 'utf8');
    const out = new Set();
    for (const m of text.matchAll(IMPORT)) {
      const r = resolveImport(file, m[1] ?? m[2] ?? m[3]);
      if (r) out.add(r);
    }
    graph.set(file, out);
  }
  return graph.get(file);
};

const closure = (roots) => {
  const seen = new Set();
  const stack = [...roots];
  while (stack.length) {
    const f = stack.pop();
    if (seen.has(f)) continue;
    seen.add(f);
    stack.push(...importsOf(f));
  }
  return seen;
};

const rel = (f) => relative(REPO, f);

// a spec reaches a page through its navigation link, whose name it may hold in a list it loops over, so any string
// literal naming a link counts; page.goto reaches one by path
const NAV = /'([^'\n]+)'|"([^"\n]+)"|page\.goto\(\s*[`'][^`']*?\/([a-z]+)[`'?]/g;
const routesIn = (text) => [...text.matchAll(NAV)].map((m) => (m[3] ? m[3] : LINK_ROUTES[m[1] ?? m[2]])).filter(Boolean);

const APP = join(E2E, 'support', 'app.ts');
// App method → the routes its body navigates to; `open` lands on the root page, which redirects to the tree
const appRoutes = () => {
  const out = Object.create(null);
  let method;
  for (const line of readFileSync(APP, 'utf8').split('\n')) {
    const m = /^ {2}(?:static |private )*(?:async )?(\w+)\(/.exec(line);
    if (m) method = m[1];
    if (method) out[method] = [...(out[method] ?? []), ...routesIn(line)];
  }
  out.open = [...(out.open ?? []), 'tree'];
  return out;
};

export const model = () => {
  const appMethods = appRoutes();
  const shell = closure([join(SRC, 'routes', '+layout.svelte'), join(SRC, 'routes', '+page.svelte'), join(SRC, 'routes', '+error.svelte')]);
  const routes = {};
  for (const dir of readdirSync(join(SRC, 'routes'))) {
    const p = join(SRC, 'routes', dir);
    if (!statSync(p).isDirectory()) continue;
    routes[dir] = closure(walk(p).filter((f) => !/\.test\.ts$/.test(f)));
  }
  const specs = {};
  for (const name of readdirSync(E2E).filter((n) => n.endsWith('.spec.ts'))) {
    const file = join(E2E, name);
    const own = closure([file]);
    const visited = new Set();
    for (const f of own) {
      if (!f.startsWith(E2E) || f === APP) continue;
      const text = readFileSync(f, 'utf8');
      for (const route of routesIn(text)) if (routes[route]) visited.add(route);
      // App's helpers navigate on the spec's behalf; count only the ones it calls
      for (const [method, via] of Object.entries(appMethods)) if (text.includes(`.${method}(`)) for (const r of via) visited.add(r);
    }
    specs[`e2e/${name}`] = { own, routes: [...visited].sort() };
  }
  return { shell, routes, specs, coverage: loadCoverage() };
};

// spec → the repo paths it ran, from coverage-map.json; a spec that goes offline posts nothing reliable, so it is mapped
// by imports
export const loadCoverage = (file = MAP) => {
  if (!existsSync(file)) return undefined;
  const raw = JSON.parse(readFileSync(file, 'utf8'));
  const specs = {};
  for (const [s, idx] of Object.entries(raw.specs)) {
    const path = join(FRONTEND, s);
    if (!existsSync(path) || readFileSync(path, 'utf8').includes('setOffline(')) continue;
    specs[s] = new Set(idx.map((i) => raw.files[i]));
  }
  return { commit: raw.commit, files: new Set(raw.files), specs };
};

// Go package directory (repo-relative) → the wasm builds that compile it
const goPackages = () => {
  const out = new Map();
  for (const w of Object.keys(WASM_BOOT)) {
    const dirs = execFileSync('go', ['list', '-deps', '-f', '{{if not .Standard}}{{.Dir}}{{end}}', `./${w}`], {
      cwd: REPO,
      encoding: 'utf8',
      env: { ...process.env, GOOS: 'js', GOARCH: 'wasm' }
    });
    for (const d of dirs.split('\n').filter(Boolean)) out.set(relative(REPO, d), [...(out.get(relative(REPO, d)) ?? []), w]);
  }
  return out;
};

// the specs a frontend file reaches: by coverage for the specs the map recorded, by imports for the rest
const reaching = (f, m, all) => {
  const abs = resolve(REPO, f);
  const cov = m.coverage;
  const known = cov?.files.has(f);
  const inShell = m.shell.has(abs);
  return all.filter((s) => {
    if (known && cov.specs[s]) return cov.specs[s].has(f);
    return inShell || m.specs[s].own.has(abs) || m.specs[s].routes.some((r) => m.routes[r].has(abs));
  });
};

export const select = (changed, m = model(), { max = MAX, go = goPackages } = {}) => {
  const all = Object.keys(m.specs).sort();
  const own = new Set();
  const picked = new Set();
  const everything = [];
  const unmapped = [];
  let pkgs;
  for (const f of changed) {
    if (INERT.some((r) => r.test(f))) continue;
    if (SPEC.test(f)) {
      const key = f.replace(/^frontend\//, '');
      if (m.specs[key]) own.add(key);
      continue;
    }
    if (EVERYTHING.some((r) => r.test(f))) {
      everything.push(f);
      continue;
    }
    let hit;
    if (/\.go$|^go\.(mod|sum)$/.test(f)) {
      pkgs ??= go();
      const builds = /^go\.(mod|sum)$/.test(f) ? Object.keys(WASM_BOOT) : (pkgs.get(dirname(f)) ?? []);
      hit = [...new Set(builds.flatMap((w) => reaching(WASM_BOOT[w], m, all)))];
    } else hit = reaching(f, m, all);
    if (hit.length === 0) unmapped.push(f);
    for (const s of hit) picked.add(s);
  }
  const reached = everything.length ? all : [...new Set([...own, ...picked])].sort();
  const capped = reached.length > max ? { reached: reached.length, max } : undefined;
  return { specs: capped ? [...own].sort() : reached, everything, unmapped, capped };
};

const changedSince = (base) => {
  const git = (...a) => execFileSync('git', ['-C', REPO, ...a], { encoding: 'utf8' });
  const files = new Set([
    ...git('diff', '--name-only', '--no-renames', `${base}...HEAD`).split('\n'),
    ...git('diff', '--name-only', '--no-renames', 'HEAD').split('\n')
  ]);
  for (const f of git('ls-files', '--others', '--exclude-standard').split('\n')) files.add(f);
  files.delete('');
  return [...files].sort();
};

if (import.meta.url === `file://${process.argv[1]}`) {
  const [cmd, ...args] = process.argv.slice(2);
  if (cmd !== 'grep' && cmd !== 'explain') {
    console.error('REFUSING: usage: node e2e/affected.mjs grep|explain <base> | -- <file>...');
    process.exit(64);
  }
  const changed = args[0] === '--' ? args.slice(1) : changedSince(args[0] ?? 'dev');
  const m = model();
  const r = { ...select(changed, m), coverage: m.coverage };
  const out = cmd === 'grep' ? process.stderr : process.stdout;
  out.write(`e2e-affected: ${changed.length} changed files\n`);
  const cov = r.coverage;
  out.write(
    cov
      ? `  coverage map recorded at ${cov.commit.slice(0, 8)} over ${Object.keys(cov.specs).length} specs\n`
      : '  no coverage map (mise run e2e-coverage): every spec is mapped by imports\n'
  );
  if (r.everything.length) out.write(`  every spec, because of: ${r.everything.join(', ')}\n`);
  if (r.capped)
    out.write(
      `  CAPPED: the change reaches ${r.capped.reached} specs, over E2E_AFFECTED_MAX=${r.capped.max}; running only the change's own specs (the full suite runs at integration)\n`
    );
  out.write(`  specs: ${r.specs.join(' ') || '(none)'}\n`);
  if (r.unmapped.length) out.write(`  UNMAPPED (only @smoke covers them): ${r.unmapped.join(', ')}\n`);
  out.write('  plus every @smoke test\n');
  if (cmd === 'grep') {
    const esc = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    // Playwright matches --grep against "<file> <title> <tags>"
    process.stdout.write([...r.specs.map((s) => `(?:^|[ /])${esc(s.replace(/^e2e\//, ''))} `), '@smoke'].join('|'));
  }
}
