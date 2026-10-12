import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { socketDescription } from './items';

const WANT = JSON.parse(readFileSync(resolve(import.meta.dirname, '../../../../testdata/oracle/socket_description_expected.json'), 'utf8')) as Record<
  string,
  string
>;

describe('socketDescription', () => {
  it("matches PoB's Item sockets line", () => {
    expect(Object.keys(WANT).length).toBeGreaterThan(5);
    for (const [sockets, line] of Object.entries(WANT)) expect(socketDescription(sockets), sockets).toBe(line);
  });
});
