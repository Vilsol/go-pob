import { execFileSync } from 'node:child_process';

const DECLARATIONS = 'src/lib/types/index.d.ts';

const ALLOWED = [
  {
    match: /Namespace '.*\.mod' has no exported member 'Mod'/,
    why:
      'crystalline emits moddb.ModResult.Mod as mod.Mod, but namespace mod exports only Tag, ' +
      'ModValueMulti, Type, Source, MFlag and KeywordFlag — the interface for Mod is never declared. ' +
      'Upstream: the type walk and the emitters disagree about what is in scope.',
    retires: 'a crystalline release that declares mod.Mod, or drops the reference'
  }
];

let raw = '';
try {
  execFileSync('npx', ['tsc', '--noEmit', '--strict', '--skipLibCheck', 'false', DECLARATIONS], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe']
  });
} catch (e) {
  raw = `${e.stdout ?? ''}${e.stderr ?? ''}`;
}

const errors = raw
  .split('\n')
  .map((l) => l.trim())
  .filter((l) => /error TS\d+:/.test(l));

if (raw !== '' && errors.length === 0) {
  console.error('REFUSING: tsc produced output but no parsable "error TSnnnn:" lines — the');
  console.error('  invocation or the parser is broken, and 0 errors here would be a false green.');
  console.error(raw.slice(0, 600));
  process.exit(2);
}

const unexpected = errors.filter((e) => !ALLOWED.some((a) => a.match.test(e)));

console.log(`${DECLARATIONS}: ${errors.length} error(s), ${errors.length - unexpected.length} allowed`);

for (const a of ALLOWED) {
  const hit = errors.some((e) => a.match.test(e));
  console.log(`  ${hit ? 'allowed  ' : 'RETIRED? '} ${a.match.source.slice(0, 60)}`);
  if (!hit) {
    console.log(`    -> no longer present. REMOVE this allowance. It retires on: ${a.retires}`);
  }
}

if (unexpected.length > 0) {
  console.error(`\n${unexpected.length} UNEXPECTED error(s) in generated declarations:`);
  for (const e of unexpected) console.error(`  ${e}`);
  console.error('\nThese are generator output, not hand-written code. Either upstream regressed or');
  console.error('the manifest now exposes something that cannot be declared.');
  process.exit(1);
}

console.log('OK — no unexpected errors in the generated declarations.');
