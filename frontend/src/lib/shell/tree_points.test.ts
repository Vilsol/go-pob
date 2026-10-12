import { expect, describe, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const header = readFileSync(fileURLToPath(new URL('./AppTree.svelte', import.meta.url)), 'utf8');

/* Modules/Build.lua:190-210 */
describe('Tree page point counters', () => {
  it('does not ship a hard-coded passive point counter', () => {
    expect(header).not.toContain('0 / 123');
    expect(header).not.toContain('0 / 8');
  });

  it('does not ship a placeholder TODO in the point counters', () => {
    expect(header).not.toContain('TODO Placeholder');
  });

  it("derives the counters from Go's point counts", () => {
    expect(header).toContain('PointsUsed');
    expect(header).toContain('estimatePlayerProgress');
  });
});
