import type { TreeNodeFx } from '$lib/design/fixtures';

/* TreeTab.lua:870-874 */
export type TattooChoiceFx = { dn: string; label: string; descriptions: string[]; legacy: boolean };

const ATTRIBUTES = ['Strength', 'Dexterity', 'Intelligence'];

/* PassiveTreeView.lua:1646-1656 */
export function tattooEditKind(n: Pick<TreeNodeFx, 'k' | 'name' | 'stats'>, tattooed: boolean): '' | 'tattoo' | 'runegraft' {
  const first = n.stats?.[0] ?? '';
  const editable =
    tattooed ||
    (n.k === 's' && ATTRIBUTES.includes(n.name)) ||
    (n.k === 'n' && ATTRIBUTES.some((a) => first.includes(`+30 to ${a}`))) ||
    n.k === 'k' ||
    n.k === 'm';
  if (!editable) return '';
  return n.k === 'm' ? 'runegraft' : 'tattoo';
}
