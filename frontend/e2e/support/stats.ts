import { buildSidebar } from '../../src/lib/display/sidebar';
import { colorCodes } from '../../src/lib/display/colors';
import type { StatLine as SidebarLine } from '../../src/lib/display/stat_list';
import type { ModelOutputs } from './go_model';

export type StatLine = { label: string; value: string };

const isSkillGroup = (g: SidebarLine[]) => g.length > 0 && g.every((l) => l.detail || l.labelColor === colorCodes.CUSTOM);

export const modelLines = (out: ModelOutputs): StatLine[] =>
  buildSidebar(out)
    .sections.flatMap((s) => s.groups.flat())
    .map((l) => ({ label: l.label + (l.labelNote ?? ''), value: l.value + (l.note ?? '') }));

/* Build.lua:1849-1872 */
export const expectedLines = (out: ModelOutputs): StatLine[] =>
  buildSidebar(out).sections.flatMap((s) =>
    s.groups.flatMap((g, gi) => {
      if (gi > 0 && isSkillGroup(g) && s.groups[gi - 1].some((l) => l.label === 'Full DPS')) return [];
      const next = s.groups[gi + 1];
      return g.flatMap((l) => {
        const line = { label: l.label + (l.labelNote ?? ''), value: l.value + (l.note ?? '') };
        if (l.label !== 'Full DPS' || !next || !isSkillGroup(next)) return [line];
        const first = next.findIndex((k) => !k.detail);
        const skills = first < 0 ? [] : next.slice(first);
        return [line, ...skills.map((k) => (k.detail ? { label: '', value: k.value } : { label: k.label + (k.labelNote ?? ''), value: k.value }))];
      });
    })
  );

export const diffLines = (want: StatLine[], got: StatLine[]): string[] => {
  const diffs: string[] = [];
  const n = Math.max(want.length, got.length);
  for (let i = 0; i < n; i++) {
    const w = want[i];
    const g = got[i];
    if (!w || !g || w.label !== g.label || w.value !== g.value) {
      diffs.push(`#${i}: want ${w ? `${w.label}=${w.value}` : '<none>'}, shown ${g ? `${g.label}=${g.value}` : '<none>'}`);
    }
  }
  return diffs;
};
