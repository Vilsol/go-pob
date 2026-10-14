import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { chmodSync, copyFileSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { afterAll, expect, test } from 'vitest';

const REAL = resolve(import.meta.dirname, '..');
const SERVE = process.env.SERVE_MJS ?? join(REAL, 'e2e', 'serve.mjs');
const dir = mkdtempSync(join(tmpdir(), 'gopob-serve-test-'));
const sandbox = join(dir, 'frontend');

const serve = (cmd) => {
  const r = spawnSync('node', [join(sandbox, 'e2e', 'serve.mjs'), cmd], { encoding: 'utf8', timeout: 90_000 });
  return { rc: r.status, out: r.stdout + r.stderr };
};
const FAKE_VITE = `#!/usr/bin/env node
import { readdirSync, readFileSync } from 'node:fs';
import { createServer } from 'node:http';
import { join, relative } from 'node:path';
const files = new Map();
const walk = (d) => readdirSync(d, { withFileTypes: true }).forEach((e) =>
  e.isDirectory() ? walk(join(d, e.name)) : files.set('/' + relative('build', join(d, e.name)), readFileSync(join(d, e.name))));
walk('build');
const port = Number(process.argv[process.argv.indexOf('--port') + 1]);
createServer((req, res) => {
  const body = files.get(req.url === '/' ? '/index.html' : req.url);
  res.writeHead(body ? 200 : 404).end(body);
}).listen(port);
`;

const pidOf = (out) => Number(/pid (\d+)/.exec(out)?.[1]);

const build = (name) => {
  rmSync(join(sandbox, 'build'), { recursive: true, force: true });
  mkdirSync(join(sandbox, 'build', '_app', 'immutable', 'entry'), { recursive: true });
  writeFileSync(join(sandbox, 'build', '_app', 'immutable', 'entry', `${name}.js`), `export const build = '${name}';\n`);
  writeFileSync(join(sandbox, 'build', 'index.html'), `<script type="module" src="/_app/immutable/entry/${name}.js"></script>\n`);
};

afterAll(() => {
  serve('stop');
  rmSync(dir, { recursive: true, force: true });
  rmSync(join(tmpdir(), `gopob-e2e-serve-${createHash('sha256').update(sandbox).digest('hex').slice(0, 12)}.log`), { force: true });
});

test('a stale or killed warm server is restarted, a current one reused', { timeout: 300_000 }, () => {
  mkdirSync(join(sandbox, 'e2e'), { recursive: true });
  copyFileSync(SERVE, join(sandbox, 'e2e', 'serve.mjs'));
  mkdirSync(join(sandbox, 'node_modules', '.bin'), { recursive: true });
  writeFileSync(join(sandbox, 'node_modules', '.bin', 'vite'), FAKE_VITE);
  chmodSync(join(sandbox, 'node_modules', '.bin', 'vite'), 0o755);
  writeFileSync(join(sandbox, 'package.json'), '{ "type": "module" }\n');
  build('first');

  const first = serve('start');
  expect(first.rc, first.out).toBe(0);
  expect(serve('port').rc).toBe(0);
  const again = serve('start');
  expect(again.out).toContain('reusing');
  expect(pidOf(again.out)).toBe(pidOf(first.out));

  build('second');
  const stale = serve('port');
  expect(stale.rc, stale.out).toBe(3);
  expect(stale.out).toContain('older build');
  const restarted = serve('start');
  expect(restarted.rc, restarted.out).toBe(0);
  expect(restarted.out).toContain('restarting');
  expect(pidOf(restarted.out.split('\n').slice(1).join('\n'))).not.toBe(pidOf(first.out));
  expect(serve('port').rc).toBe(0);

  process.kill(-pidOf(restarted.out.split('\n').slice(1).join('\n')));
  const afterKill = serve('start');
  expect(afterKill.rc, afterKill.out).toBe(0);
  expect(afterKill.out).not.toContain('reusing');
  expect(serve('port').rc).toBe(0);
});
