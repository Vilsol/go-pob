import { describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { brotliDecompressSync } from 'node:zlib';
import { readFileSync } from 'node:fs';

import { cachedAsset } from '../../../e2e/support/data_cache';

const rawJson = (v: string) => {
  const a = cachedAsset(`https://data.pob.gg/data/${v.replace('_', '.')}/tree/data.json.br`);
  if (a.status !== 200) throw new Error(`REFUSING: no ${v} tree in the bundle cache`);
  return brotliDecompressSync(a.body).toString('utf8');
};

vi.mock('$lib/go/worker', () => ({ syncWrap: { GetTree: (v: string) => Promise.resolve(rawJson(v)) }, syncWorker: undefined }));

const st = await import('$lib/skill_tree');
const { groupCentre, secondaryAscendancy } = await import('$lib/skill_tree/geometry');
const pobGroups = JSON.parse(readFileSync(new URL('./testdata/pob-3_29-ascendancy-groups.json', import.meta.url), 'utf8')) as Record<
  string,
  { ascendancy: string; x: number; y: number }
>;

type Tree = import('$lib/skill_tree/types').Tree;
const rawTree = (v: string) => JSON.parse(rawJson(v)) as Tree;

describe('switching the tree version', () => {
  it('replaces every drawn node, group and offset with the new version’s', async () => {
    await st.loadSkillTree('3_18');
    expect(get(st.skillTreeVersion)).toBe('3_18');
    expect(st.ascendancyGroupPositionOffsets.Juggernaut, 'the 3.18 tree has no Juggernaut offset').toBeDefined();

    await st.loadSkillTree('3_29');
    expect(get(st.skillTreeVersion)).toBe('3_29');
    const t = rawTree('3_29');
    const legacy = new Set(['Warden', 'Warlock', 'Primalist']);
    const want = Object.values(t.groups)
      .flatMap((g) => g.nodes ?? [])
      .filter((id) => !legacy.has(t.nodes[id].ascendancyName ?? ''))
      .map(Number)
      .sort((a, b) => a - b);
    expect([...st.drawnNodes.keys()].sort((a, b) => a - b)).toEqual(want);
    expect(st.drawnGroups.size).toBe(Object.values(t.groups).filter((g) => !(g.nodes ?? []).some((id) => legacy.has(t.nodes[id].ascendancyName ?? ''))).length);
    expect(Object.keys(st.ascendancyGroupPositionOffsets).sort()).toEqual(
      [
        ...new Set(Object.values(t.nodes).flatMap((n) => (n.isAscendancyStart && n.ascendancyName && !legacy.has(n.ascendancyName) ? [n.ascendancyName] : [])))
      ].sort()
    );
  });

  it('puts every 3.29 ascendancy group where PoB’s tree data has it', () => {
    const t = get(st.skillTree)!;
    const starts = Object.values(t.nodes).filter((n) => n.isAscendancyStart && n.ascendancyName && st.drawnNodes.has(n.skill!));
    expect(starts.length, 'the 3.29 tree has classic, Scion, and bloodline starts').toBe(34);
    let compared = 0;
    const absent: string[] = [];
    for (const [id, want] of Object.entries(pobGroups)) {
      if (!st.drawnGroups.has(Number(id))) {
        if (!['Warden', 'Warlock', 'Primalist'].includes(want.ascendancy)) absent.push(id);
        continue;
      }
      const got = groupCentre(t, st.ascendancyGroupPositionOffsets, Number(id), want.ascendancy);
      expect(got.x, `${want.ascendancy} group ${id}`).toBeCloseTo(want.x, 6);
      expect(got.y, `${want.ascendancy} group ${id}`).toBeCloseTo(want.y, 6);
      compared++;
    }
    expect(compared).toBeGreaterThan(100);
    expect(absent.sort()).toEqual(['25519', '37841', '44472', '50933', '56600', '60495', '63033']);
  });

  it('draws a bloodline’s frames from its own sheet, and PoB’s other ascendancies from the ascendancy sheet', () => {
    expect(st.bloodlineSprites.Trialmaster?.AscendancyMiddle?.filename).toMatch(/bloodline-3\.webp$/);
    expect(st.bloodlineSprites.Abyssal, 'Abyssal is not in PoB’s bloodline sprite table').toBeUndefined();
    expect(st.inverseSpritesOther.AscendancyMiddle?.filename).toMatch(/ascendancy-3\.webp$/);
    expect(st.inverseSpritesOther.ClassesAbyssal?.filename, 'bloodline class art is found on the bloodline sheet').toMatch(/bloodline-3\.webp$/);
    expect(st.inverseSpritesOther.ClassesWarden, 'legacy class art is not drawn').toBeUndefined();
  });

  it('reads the build’s bloodline from the tree’s 1-based list, skipping the legacy ones', () => {
    const t = get(st.skillTree)!;
    expect(secondaryAscendancy(t, 0)).toBeUndefined();
    expect(secondaryAscendancy(t, 1), 'Warden is removed on load').toBeUndefined();
    expect(secondaryAscendancy(t, 4)).toBe('Trialmaster');
    expect(secondaryAscendancy(t, 16)).toBe('Abyssal');
  });
});
