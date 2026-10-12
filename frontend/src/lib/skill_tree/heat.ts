import type { PowerRow } from '$lib/design/tree';

/* PassiveTreeView.lua:906-919 */
export const heatChannel = (power: number, max: number) => (max > 0 ? Math.min(1, Math.sqrt((Math.max(power, 0) / max) * 1.5)) : 0);

/* Main.lua:957-961 */
export type NodePowerTheme = 'RED/BLUE' | 'RED/GREEN' | 'GREEN/BLUE';
export const NODE_POWER_THEMES: NodePowerTheme[] = ['RED/BLUE', 'RED/GREEN', 'GREEN/BLUE'];

/* PassiveTreeView.lua:913-918 */
export const heatSingleColour = (power: number, max: number, theme: NodePowerTheme = 'RED/BLUE'): [number, number, number] => {
  const c = heatChannel(power, max);
  return theme === 'RED/GREEN' ? [0, c, 0] : theme === 'GREEN/BLUE' ? [0, 0, c] : [c, 0, 0];
};

export const heatValues = (rows: PowerRow[], ascendancy: Set<number> = new Set()): { values: Map<number, number>; max: number } => {
  const values = new Map<number, number>();
  let max = 0;
  for (const r of rows) {
    if (r.Allocated || r.Cluster) continue;
    values.set(r.ID, Math.max(values.get(r.ID) ?? -Infinity, r.Power));
    // CalcsTab.lua:533-534
    if (r.PathDist < 1000 && !ascendancy.has(r.ID)) max = Math.max(max, r.Power);
  }
  return { values, max };
};

/* PassiveTreeView.lua:920-933, CalcsTab.lua:543-547 */
export const heatMixValues = (
  rows: PowerRow[],
  ascendancy: Set<number> = new Set()
): { mix: Map<number, [number, number]>; maxOffence: number; maxDefence: number } => {
  const mix = new Map<number, [number, number]>();
  let maxOffence = 0;
  let maxDefence = 0;
  for (const r of rows) {
    if (r.Allocated || r.Cluster) continue;
    const off = r.Offence ?? 0;
    const def = r.Defence ?? 0;
    mix.set(r.ID, [off, def]);
    if (r.PathDist < 1000 && !ascendancy.has(r.ID)) {
      maxOffence = Math.max(maxOffence, off);
      maxDefence = Math.max(maxDefence, def);
    }
  }
  return { mix, maxOffence, maxDefence };
};

/* PassiveTreeView.lua:927-932 */
export const heatMixColour = (
  off: number,
  def: number,
  maxOffence: number,
  maxDefence: number,
  theme: NodePowerTheme = 'RED/BLUE'
): [number, number, number] => {
  const dps = maxOffence > 0 ? Math.sqrt((Math.max(off, 0) / maxOffence) * 1.5) : 0;
  const defence = maxDefence > 0 ? Math.sqrt((Math.max(def, 0) / maxDefence) * 1.5) : 0;
  const mixed = (Math.max(dps - 0.5, 0) + Math.max(defence - 0.5, 0)) / 2;
  const [d, m, f] = [Math.min(1, dps), Math.min(1, mixed), Math.min(1, defence)];
  return theme === 'RED/GREEN' ? [d, f, m] : theme === 'GREEN/BLUE' ? [m, d, f] : [d, m, f];
};
