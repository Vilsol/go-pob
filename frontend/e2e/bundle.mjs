// The bundle the e2e tests serve. The TinyGo wasm takes minutes to build after a Go change, so it is built only when
// asked for or when a selected test loads it:
//
//   node e2e/bundle.mjs with | without          the standard bundle, with or without the TinyGo wasm
//   node e2e/bundle.mjs for [selection]...      with it only when the Playwright selection lists a TinyGo spec
import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';

const FRONTEND = resolve(import.meta.dirname, '..');
// the specs that boot a TinyGo engine
const TINYGO_SPECS = ['tinygo-engine.spec.ts'];

const run = (cmd, args) => {
  const r = spawnSync(cmd, args, { cwd: FRONTEND, stdio: 'inherit' });
  if (r.status !== 0) process.exit(r.status ?? 1);
};

const selected = (selection) => {
  const listed = spawnSync('npx', ['playwright', 'test', '-c', 'e2e/playwright.config.ts', '--list', ...selection], {
    cwd: FRONTEND,
    encoding: 'utf8',
    maxBuffer: 64 << 20
  });
  if (listed.status !== 0) {
    process.stderr.write(listed.stdout + listed.stderr);
    console.error('REFUSING: Playwright could not list the selection above');
    process.exit(listed.status ?? 1);
  }
  const total = listed.stdout.match(/^Total: (.*)$/m)?.[1] ?? 'an unknown count';
  const tiny = TINYGO_SPECS.filter((s) => listed.stdout.includes(s));
  console.log(`e2e selection: ${total}; ${tiny.length ? `TinyGo for ${tiny.join(', ')}` : 'no TinyGo spec'}`);
  return tiny.length > 0;
};

const [mode, ...rest] = process.argv.slice(2);
let tinygo;
if (mode === 'with' && rest.length === 0) tinygo = true;
else if (mode === 'without' && rest.length === 0) tinygo = false;
else if (mode === 'for') tinygo = selected(rest);
else {
  console.error('REFUSING: usage: node e2e/bundle.mjs with | without | for [playwright selection]...');
  process.exit(64);
}

if (tinygo) {
  run('mise', ['run', 'build-go-tiny', ':::', 'build-go2-tiny']);
  run('go', ['run', '../scripts/gopob', 'nice', 'gzip', '-kf9', 'static/go-pob-tiny.wasm', 'static/go-pob2-tiny.wasm']);
}
run('mise', ['run', 'e2e-bundle-build']);
