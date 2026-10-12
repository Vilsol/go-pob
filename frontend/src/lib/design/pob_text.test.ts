import { describe, expect, it } from 'vitest';
import { pobSegments } from './pob_text';

// Main.lua:117, 1127-1131
describe('pobSegments', () => {
  const line = '^x8888FF+10 to Foo ^8(Not supported in PoB yet)';
  it('keeps the unsupported note by default and drops it when the option is off', () => {
    expect(pobSegments(line).map((s) => s.text)).toEqual(['+10 to Foo ', '(Not supported in PoB yet)']);
    expect(pobSegments(line, false)).toEqual([{ text: '+10 to Foo', color: '#8888FF' }]);
  });
});
