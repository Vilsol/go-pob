import { parseAst } from 'vite';
import { expect, test } from 'vitest';

import { compact, instrument } from './coverage.mjs';

test('every function marks its file on entry, and the result still parses', () => {
  const src = 'export function f(a){return a}\nconst g = () => ({ a: 1 });\nconst h = (x) => (y) => x + y;\nclass A { m() { return 1 } n = async () => 2 }\n';
  const r = instrument(src, 'frontend/src/a.ts');
  expect(r.functions).toBe(6);
  expect(() => parseAst(r.code)).not.toThrow();
  expect(r.code.match(/__e2eCov\?\.h\("frontend\/src\/a\.ts"\)/g)).toHaveLength(6);
  expect(r.code.startsWith(';(globalThis.__e2eCov||')).toBe(true);
});

test('a file with no function counts as run where it was loaded', () => {
  const part = {
    commit: 'c',
    files: { 'frontend/src/a.ts': 2, 'frontend/src/consts.ts': 0, 'frontend/src/b.ts': 1 },
    specs: { 'e2e/x.spec.ts': { hit: ['frontend/src/a.ts'], loaded: ['frontend/src/a.ts', 'frontend/src/b.ts', 'frontend/src/consts.ts'] } }
  };
  const m = compact([part]);
  expect(m.specs['e2e/x.spec.ts'].map((i) => m.files[i])).toEqual(['frontend/src/a.ts', 'frontend/src/consts.ts']);
});
