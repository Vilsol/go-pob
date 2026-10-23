// Splits the e2e tests over CI's shards by how long each took, not by count. `plan <i>/<n> <file>` writes shard i's
// test list for `playwright test --test-list`; `record` reads a run's log (`gh run view <id> --log`) on stdin and
// updates durations.json. A test with no recorded duration counts as the median.
import { execFileSync } from 'node:child_process';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';

const FRONTEND = resolve(import.meta.dirname, '..');
const DURATIONS = join(FRONTEND, 'e2e', 'durations.json');
const SEP = ' › ';

// every test the config selects (E2E_SLOW decides @slow), in list order
export function listTests(env = process.env) {
  const out = execFileSync('npx', ['playwright', 'test', '-c', 'e2e/playwright.config.ts', '--list', '--reporter=json'], {
    cwd: FRONTEND,
    env,
    encoding: 'utf8',
    maxBuffer: 64 << 20
  });
  const tests = [];
  const walk = (suite, titles) => {
    for (const spec of suite.specs ?? []) for (const _ of spec.tests) tests.push({ file: spec.file, titles: [...titles, spec.title] });
    for (const s of suite.suites ?? []) walk(s, [...titles, s.title]);
  };
  for (const file of JSON.parse(out).suites) walk(file, []);
  return tests;
}

// "<file> › <describe…> › <title>", as durations.json and the list reporter name a test
export const testKey = (t) => [t.file, ...t.titles].join(SEP);

// the lines a shard's --test-list can hold, each with the tests it selects. A line matches every test whose title path
// starts with it, and it splits on '›', so a test with '›' in a title goes under the titles before that one, and a
// line that is a prefix of another takes that one's tests with it.
export function units(tests) {
  const lines = new Map();
  for (const t of tests) {
    const cut = t.titles.findIndex((x) => x.includes('›'));
    const line = [t.file, ...(cut < 0 ? t.titles : t.titles.slice(0, cut))].join(SEP);
    lines.set(line, [...(lines.get(line) ?? []), testKey(t)]);
  }
  const out = new Map();
  for (const line of [...lines.keys()].sort((a, b) => a.length - b.length)) {
    const wider = [...out.keys()].find((o) => line.startsWith(o + SEP));
    if (wider) out.get(wider).push(...lines.get(line));
    else out.set(line, [...lines.get(line)]);
  }
  return out;
}

// longest first onto the least-loaded shard; the same inputs always give the same split
export function plan(lines, durations, total) {
  const known = Object.values(durations)
    .filter((s) => s > 0)
    .sort((a, b) => a - b);
  const median = known.length ? known[Math.floor(known.length / 2)] : 1;
  const weight = [...lines].map(([line, keys]) => [line, keys.reduce((sum, k) => sum + (durations[k] ?? median), 0), keys.length]);
  const shards = Array.from({ length: total }, () => ({ seconds: 0, tests: 0, lines: [] }));
  for (const [line, w, n] of weight.sort((a, b) => b[1] - a[1] || (a[0] < b[0] ? -1 : 1))) {
    const s = shards.reduce((min, s) => (s.seconds < min.seconds ? s : min));
    s.seconds += w;
    s.tests += n;
    s.lines.push(line);
  }
  return { shards, unknown: [...lines.values()].flat().filter((k) => durations[k] === undefined) };
}

const UNIT = { ms: 0.001, s: 1, m: 60, h: 3600 };
// the list reporter's result lines: "✓  12 e2e/file.spec.ts:9:3 › describe › title @smoke (1.5m)"; it prints a
// test's tags after its title, where the title path --test-list matches does not have them
const RESULT = /[✓✘]\s+\d+\s+(?:e2e\/)?(\S+?\.spec\.ts):\d+:\d+ › (.*?)(?: @[\w-]+)* \((\d+(?:\.\d+)?)(ms|s|m|h)\)\s*$/;
// a skipped test: "-  29 e2e/file.spec.ts:159:1 › title", no duration
const SKIPPED = /\s-\s+\d+\s+(?:e2e\/)?(\S+?\.spec\.ts):\d+:\d+ › (.*?)(?: @[\w-]+)*\s*$/;

export function parseLog(text) {
  const seen = {};
  for (const raw of text.split('\n')) {
    const line = raw.replace(/\x1b\[[0-9;]*m/g, '');
    const m = RESULT.exec(line);
    if (m) seen[[m[1], m[2]].join(SEP)] = Number(m[3]) * UNIT[m[4]];
    const s = !m && SKIPPED.exec(line);
    if (s) seen[[s[1], s[2]].join(SEP)] = 0;
  }
  return seen;
}

const readDurations = () => JSON.parse(readFileSync(DURATIONS, 'utf8'));

if (import.meta.url === `file://${process.argv[1]}`) {
  const [cmd, arg, file] = process.argv.slice(2);
  if (cmd === 'plan') {
    const [i, n] = (arg ?? '').split('/').map(Number);
    if (!(Number.isInteger(i) && Number.isInteger(n) && i >= 1 && i <= n) || !file) throw new Error('usage: shard.mjs plan <i>/<n> <file>');
    const { shards, unknown } = plan(units(listTests()), readDurations(), n);
    shards.forEach((s, j) => console.error(`shard ${j + 1}/${n}: ${s.tests} tests, ~${Math.round(s.seconds / 60)} min of test time`));
    if (unknown.length) console.error(`${unknown.length} tests have no recorded duration and count as the median (task e2e-durations records them)`);
    mkdirSync(dirname(resolve(file)), { recursive: true });
    writeFileSync(file, shards[i - 1].lines.join('\n') + '\n');
  } else if (cmd === 'record') {
    const seen = parseLog(readFileSync(0, 'utf8'));
    if (!Object.keys(seen).length) throw new Error('no test results in the log on stdin');
    // only tests that still exist, with @slow listed too
    const live = new Set(listTests({ ...process.env, E2E_SLOW: '1' }).map(testKey));
    const merged = { ...readDurations(), ...seen };
    const out = Object.fromEntries(
      Object.keys(merged)
        .filter((k) => live.has(k))
        .sort()
        .map((k) => [k, Math.round(merged[k] * 10) / 10])
    );
    writeFileSync(DURATIONS, JSON.stringify(out, null, 2) + '\n');
    const missing = [...live].filter((k) => out[k] === undefined).length;
    console.error(`recorded ${Object.keys(seen).length} results; ${Object.keys(out).length} tests have a duration, ${missing} do not`);
  } else {
    throw new Error('usage: shard.mjs plan <i>/<n> <file> | record < log');
  }
}
