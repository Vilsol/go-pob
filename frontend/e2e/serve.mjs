import { spawn } from 'node:child_process';
import { createHash } from 'node:crypto';
import { closeSync, existsSync, openSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const FRONTEND = resolve(import.meta.dirname, '..');
const KEY = createHash('sha256').update(FRONTEND).digest('hex').slice(0, 12);
const STATE = join(tmpdir(), `gopob-e2e-serve-${KEY}.json`);
const LOG = join(tmpdir(), `gopob-e2e-serve-${KEY}.log`);
const REFUSED = 3;

const alive = (pid) => {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
};

const responds = async (port) => {
  try {
    const res = await fetch(`http://localhost:${port}/`, { signal: AbortSignal.timeout(2000) });
    return res.ok;
  } catch {
    return false;
  }
};

const servesStale = async (port) => {
  const index = join(FRONTEND, 'build', 'index.html');
  if (!existsSync(index)) return 'there is no build/index.html';
  const asset = readFileSync(index, 'utf8').match(/["'](?:\.)?(\/_app\/immutable\/[^"']+\.js)["']/)?.[1];
  if (!asset) return 'build/index.html names no /_app/immutable asset';
  try {
    const res = await fetch(`http://localhost:${port}${asset}`, { signal: AbortSignal.timeout(2000) });
    if (!res.ok) return `${asset} answers ${res.status}`;
    const body = Buffer.from(await res.arrayBuffer());
    if (!body.equals(readFileSync(join(FRONTEND, 'build', asset)))) return `${asset} differs from the build on disk`;
    return '';
  } catch (e) {
    return `${asset}: ${e.message}`;
  }
};

const readState = async () => {
  if (!existsSync(STATE)) return { state: null, why: 'no warm server was started for this worktree' };
  const state = JSON.parse(readFileSync(STATE, 'utf8'));
  if (!alive(state.pid)) return { state: null, why: `the warm server (pid ${state.pid}) has exited; its log is ${LOG}` };
  const live = { state: null, pid: state.pid };
  if (!(await responds(state.port))) return { ...live, why: `pid ${state.pid} is alive but port ${state.port} does not answer` };
  const stale = await servesStale(state.port);
  if (stale) return { ...live, why: `pid ${state.pid} serves an older build (${stale})` };
  return { state };
};

const kill = (pid) => {
  try {
    process.kill(-pid);
    return true;
  } catch {
    return false;
  }
};

const freePort = () =>
  new Promise((ok, fail) => {
    const s = createServer();
    s.once('error', fail);
    s.listen(0, '127.0.0.1', () => {
      const { port } = s.address();
      s.close(() => ok(port));
    });
  });

const start = async () => {
  const { state, pid, why } = await readState();
  if (state) {
    console.log(`e2e-serve: reusing pid ${state.pid}; it serves the build just made`);
    console.log(`E2E_PORT=${state.port}`);
    return 0;
  }
  if (pid) {
    console.log(`e2e-serve: restarting: ${why}`);
    kill(pid);
    rmSync(STATE, { force: true });
  }
  const port = await freePort();
  const log = openSync(LOG, 'w');
  const child = spawn('npx', ['vite', 'preview', '--port', String(port), '--strictPort'], {
    cwd: FRONTEND,
    detached: true,
    stdio: ['ignore', log, log]
  });
  child.unref();
  closeSync(log);
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    if (!alive(child.pid)) break;
    if (await responds(port)) {
      writeFileSync(STATE, JSON.stringify({ pid: child.pid, port, frontend: FRONTEND }));
      console.log(`e2e-serve: vite preview pid ${child.pid}, log ${LOG}`);
      console.log(`E2E_PORT=${port}`);
      return 0;
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  console.error(`REFUSING: e2e-serve: vite preview did not answer on ${port} within 60s; log ${LOG}`);
  kill(child.pid);
  return REFUSED;
};

const port = async () => {
  const { state, why } = await readState();
  if (!state) {
    console.error(`REFUSING: REUSE=1 needs a warm server: ${why}. Run \`task e2e-serve\` first.`);
    return REFUSED;
  }
  console.log(state.port);
  return 0;
};

const stop = async () => {
  if (!existsSync(STATE)) {
    console.log('e2e-serve: nothing to stop');
    return 0;
  }
  const { pid } = JSON.parse(readFileSync(STATE, 'utf8'));
  console.log(kill(pid) ? `e2e-serve: stopped pid ${pid}` : `e2e-serve: pid ${pid} had already exited`);
  rmSync(STATE, { force: true });
  return 0;
};

const cmds = { start, port, stop };
const cmd = cmds[process.argv[2]];
if (!cmd) {
  console.error(`REFUSING: usage: node e2e/serve.mjs ${Object.keys(cmds).join('|')}`);
  process.exit(64);
}
process.exit(await cmd());
