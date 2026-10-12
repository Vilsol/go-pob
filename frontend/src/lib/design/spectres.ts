// MinionSearchListControl.lua:36-67

export type SpectreRow = { id: string; name: string; skills: string[]; tooltip: string[] };

export function spectreMatches(row: SpectreRow, search: string, mode: number): boolean {
  const s = search.toLowerCase();
  if (!/\S/.test(s)) return true;
  const alternatives = s.split('|').filter((a) => a !== '');
  const match = (text: string) => alternatives.some((a) => text.toLowerCase().includes(a));
  if ((mode === 1 || mode === 3) && match(row.name)) return true;
  if ((mode === 2 || mode === 3) && row.skills.some(match)) return true;
  return false;
}
