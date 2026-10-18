import { spawn } from 'node:child_process';
import { readdirSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { resolve } from 'node:path';

const PORT = process.env.UNLIGHTHOUSE_PORT || '4392';
const SITE = process.env.UNLIGHTHOUSE_SITE;
const root = resolve(import.meta.dirname, '..');
const out = resolve(root, 'e2e/.results/unlighthouse');
const MARK = `--gopob-unlighthouse=${process.pid}`;

const preview = SITE ? undefined : spawn('npx', ['vite', 'preview', '--port', PORT, '--strictPort'], { cwd: root, stdio: 'inherit' });

const up = async () => {
  for (let i = 0; i < 60; i++) {
    try {
      if ((await fetch(`http://localhost:${PORT}/`)).ok) return;
    } catch {
      /* not listening yet */
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`vite preview did not answer on :${PORT}`);
};

const lighthouse = (dir) => {
  const html = readFileSync(resolve(out, 'reports', dir, 'lighthouse.html'), 'utf8');
  const json = /window\.__LIGHTHOUSE_JSON__ = (\{.*?\});\s*<\/script>/s.exec(html)?.[1];
  if (!json) throw new Error(`no Lighthouse result in reports/${dir}/lighthouse.html`);
  return JSON.parse(json);
};

// Unlighthouse leaves its Chromium workers running, and each starts its own session, so they are found by the
// marker argument every one of them was launched with. Chromium rewrites its argv into one space-joined string.
const reap = () => {
  for (const pid of readdirSync('/proc').filter((p) => /^\d+$/.test(p))) {
    let cmd;
    try {
      cmd = readFileSync(`/proc/${pid}/cmdline`, 'utf8').replaceAll('\0', ' ');
    } catch {
      continue;
    }
    if (!` ${cmd} `.includes(` ${MARK} `)) continue;
    const profile = / --user-data-dir=(\S+)/.exec(cmd)?.[1];
    try {
      process.kill(Number(pid), 'SIGKILL');
    } catch {
      /* already gone */
    }
    if (profile?.startsWith(tmpdir())) rmSync(profile, { recursive: true, force: true });
  }
};

// Informative audits weigh 0 and report score 1 whatever they find, so a category score hides them.
const findings = () => {
  const seen = new Map();
  const dirs = [
    '',
    ...readdirSync(resolve(out, 'reports'), { withFileTypes: true })
      .filter((d) => d.isDirectory() && !d.name.startsWith('__'))
      .map((d) => d.name)
  ];
  for (const dir of dirs) {
    const lh = lighthouse(dir);
    const route = new URL(lh.finalDisplayedUrl).pathname;
    for (const cat of ['best-practices', 'accessibility']) {
      for (const ref of lh.categories[cat].auditRefs) {
        const a = lh.audits[ref.id];
        const items = a.details?.items ?? [];
        if (ref.id === 'baseline') continue;
        const failed = a.scoreDisplayMode === 'informative' ? items.length > 0 : a.score !== null && a.score < 1;
        if (!failed) continue;
        const key = `${cat} ${ref.id}`;
        const f = seen.get(key) ?? { cat, id: ref.id, weight: ref.weight, title: a.title, detail: items[0]?.description ?? '', routes: [] };
        f.routes.push(route);
        seen.set(key, f);
      }
    }
  }
  console.log(`\n${seen.size} findings across ${dirs.length} routes (w=0 counts for nothing in the score):`);
  for (const f of [...seen.values()].sort((a, b) => b.weight - a.weight || a.id.localeCompare(b.id))) {
    console.log(`  w=${f.weight} ${f.cat} ${f.id}: ${f.title}${f.detail ? ` (${String(f.detail).slice(0, 80)})` : ''}`);
    console.log(`      ${f.routes.length === dirs.length ? 'every route' : f.routes.sort().join(' ')}`);
  }
};

let status = 1;
try {
  if (!SITE) await up();
  const scan = spawn('pnpm', ['dlx', '@unlighthouse/cli@0.19.1', 'ci', '--config-file', 'e2e/load/unlighthouse.config.ts'], {
    cwd: root,
    stdio: 'inherit',
    env: { ...process.env, UNLIGHTHOUSE_PORT: PORT, UNLIGHTHOUSE_MARK: MARK }
  });
  status = await new Promise((r) => scan.on('exit', (c) => r(c ?? 1)));
  if (status === 0) findings();
} finally {
  preview?.kill();
  reap();
}
process.exit(status);
