export type PartyCategory = 'PartyMemberStats' | 'Aura' | 'Curse' | 'Warcry' | 'Link' | 'EnemyConditions' | 'EnemyMods';

/* PartyTab.lua:382-480 */
export const PARTY_CATEGORIES: { id: PartyCategory; label: string }[] = [
  { id: 'PartyMemberStats', label: 'Party Member Stats' },
  { id: 'Aura', label: 'Auras' },
  { id: 'Curse', label: 'Curses' },
  { id: 'Warcry', label: 'Warcry Skills' },
  { id: 'Link', label: 'Link Skills' },
  { id: 'EnemyConditions', label: 'Enemy Conditions' },
  { id: 'EnemyMods', label: 'Enemy Modifiers' }
];

/* PartyTab.lua:35 */
export const PARTY_DESTINATIONS = ['All', 'Party Member Stats', 'Aura', 'Curse', 'Warcry Skills', 'Link Skills', 'EnemyConditions', 'EnemyMods'];
const DEST_CATEGORY: Record<string, PartyCategory> = {
  'Party Member Stats': 'PartyMemberStats',
  Aura: 'Aura',
  Curse: 'Curse',
  'Warcry Skills': 'Warcry',
  'Link Skills': 'Link',
  EnemyConditions: 'EnemyConditions',
  EnemyMods: 'EnemyMods'
};
export const destinationCategories = (dest: string): PartyCategory[] => (dest === 'All' ? PARTY_CATEGORIES.map((c) => c.id) : [DEST_CATEGORY[dest]]);

export type PartyMember = { id: string; name: string; className: string; ascendancy: string; level: number; active: boolean; gives?: string };

export type PartyEntry = { name: string; value: string };

// PartyTab.lua:713
function read(cat: PartyCategory, buf: string): PartyEntry[] {
  const lines = buf.split('\n').map((l) => l.trim());
  const out: PartyEntry[] = [];
  if (cat === 'EnemyConditions') {
    for (const l of lines) if (l) out.push({ name: l.replace(/^Condition:/, ''), value: 'yes' });
    return out;
  }
  if (cat === 'EnemyMods') {
    let cur: PartyEntry | undefined;
    for (const l of lines) {
      if (!l.includes('|')) {
        cur = l ? { name: l, value: '' } : undefined;
        if (cur) out.push(cur);
      } else if (cur) {
        const [value, , , type] = l.split('|');
        cur.value = [cur.value, `${value} ${type === 'FLAG' ? '' : type}`.trim()].filter(Boolean).join(', ');
      }
    }
    return out;
  }
  if (cat === 'PartyMemberStats') {
    for (const l of lines) {
      const m = /^([\w ]+?)(?:\|[^=]*)?=(.+)$/.exec(l);
      if (m) out.push({ name: m[1], value: m[2] });
    }
    return out;
  }
  let mode: 'limit' | 'name' | 'effect' | 'mark' | 'stats' = cat === 'Curse' ? 'limit' : 'name';
  let cur: PartyEntry | undefined;
  for (const l of lines) {
    if (l !== '---' && l.includes('---')) continue;
    if (mode === 'limit' && l) mode = 'name';
    else if (mode === 'name' && l) {
      cur = { name: l.replace('_Debuff', ''), value: '' };
      out.push(cur);
      mode = l === 'extraAura' || l === 'otherEffects' ? 'stats' : 'effect';
    } else if (mode === 'effect') {
      if (cur) cur.value = cat === 'Curse' ? `${l}% effect` : `+${l}% effect`;
      mode = cat === 'Curse' ? 'mark' : 'stats';
    } else if (mode === 'mark') mode = 'stats';
    else if (l === '---') mode = 'name';
  }
  return out.filter((e) => e.name !== 'extraAura' && e.name !== 'otherEffects');
}

export const summarize = (cat: PartyCategory, buf: string): PartyEntry[] => [...new Map(read(cat, buf).map((e) => [e.name, e])).values()];
