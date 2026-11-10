// rerun_failed.mjs --log <file> [--before <n>] [--times <n>]: re-run each test a full e2e run failed, together with the
// spec files of the tests that started just before it, in one worker so pages carry over as they did, several times.
// A lone failure in a full run is usually a page the previous test left behind; this reproduces it in minutes.
import { readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';

const opts = { log: '', before: 12, times: 3 };
const argv = process.argv.slice(2);
for (let i = 0; i < argv.length; i++) {
  const [k, v] = [argv[i], argv[i + 1]];
  if (k === '--log') opts.log = v;
  else if (k === '--before') opts.before = Number(v);
  else if (k === '--times') opts.times = Number(v);
  else {
    console.error(`rerun_failed: unknown argument ${k}`);
    process.exit(2);
  }
  i++;
}
if (!opts.log || !(opts.before >= 0) || !(opts.times >= 1)) {
  console.error('usage: rerun_failed.mjs --log <file> [--before <n>] [--times <n>]');
  process.exit(2);
}

// "  ✘  73 e2e/layout-phone.spec.ts:96:5 › 390×769 layout › Items (2ms)"
const LINE = /^\s+(✓|✘|-)\s+(\d+)\s+(e2e\/[^:\s]+):(\d+):\d+ › (.*?)(?: \([\d.]+m?s\))?$/;
const started = new Map();
const failed = [];
for (const line of readFileSync(opts.log, 'utf8').split('\n')) {
  const m = LINE.exec(line);
  if (!m) continue;
  const t = { n: Number(m[2]), spec: m[3], line: m[4], title: m[5] };
  if (!started.has(t.n)) started.set(t.n, t);
  if (m[1] === '✘') failed.push(t);
}
if (started.size === 0) {
  console.error(`rerun_failed: ${opts.log} has no Playwright list-reporter lines (is it a full e2e run's output?)`);
  process.exit(3);
}
if (failed.length === 0) {
  console.error(`rerun_failed: ${opts.log} lists ${started.size} tests and no failures; nothing to re-run`);
  process.exit(3);
}

const seen = new Set();
let bad = 0;
for (const f of failed) {
  const id = `${f.spec}:${f.line}`;
  if (seen.has(id)) continue;
  seen.add(id);
  const specs = [];
  for (let n = f.n - opts.before; n < f.n; n++) {
    const t = started.get(n);
    if (t && !specs.includes(t.spec)) specs.push(t.spec);
  }
  if (!specs.includes(f.spec)) specs.push(f.spec);
  console.log(`\n${id} › ${f.title}\n  with ${specs.join(' ')}\n  ${opts.times} run(s), one worker`);
  let reproduced = 0;
  for (let i = 1; i <= opts.times; i++) {
    const r = spawnSync('mise', ['run', 'e2e', '--slow', '--', ...specs], {
      env: { ...process.env, E2E_WORKERS: '1' },
      encoding: 'utf8',
      maxBuffer: 256 * 2 ** 20
    });
    const out = `${r.stdout}${r.stderr}`;
    const fails = out.split('\n').filter((l) => /^\s+✘/.test(l));
    const self = fails.some((l) => l.includes(id));
    if (self) reproduced++;
    const summary = out.match(/\d+ passed[^\n]*/)?.[0] ?? `exit ${r.status}`;
    console.log(`  run ${i}: ${self ? 'FAILED' : 'passed'} · ${summary}${fails.length ? `\n    ${fails.map((l) => l.trim()).join('\n    ')}` : ''}`);
  }
  console.log(`  ${id}: failed ${reproduced} of ${opts.times}`);
  if (reproduced > 0) bad++;
}
process.exit(bad > 0 ? 1 : 0);
