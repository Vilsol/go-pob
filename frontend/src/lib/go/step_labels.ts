// the undo history's labels: what one recorded step did, named from the build it changed

export type StepKind = 'tree' | 'item' | 'gem' | 'config' | 'import' | 'set' | 'party' | 'calcs' | 'notes' | 'other';
export type StepLabel = { action: string; page: string; kind: StepKind };
// one headline stat line the step changed, as the stats strip shows it
export type StepDelta = { label: string; value: string; better: boolean; section?: string };

const ENTITIES: Record<string, string> = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'" };
const unescape = (s: string) =>
  s.replace(/&(#x[\da-f]+|#\d+|\w+);/gi, (m, e: string) =>
    e[0] === '#' ? String.fromCodePoint(e[1].toLowerCase() === 'x' ? parseInt(e.slice(2), 16) : Number(e.slice(1))) : (ENTITIES[e] ?? m)
  );

export const xmlAttrs = (s: string): Record<string, string> => {
  const out: Record<string, string> = {};
  for (const m of s.matchAll(/([\w:-]+)="([^"]*)"/g)) out[m[1]] = unescape(m[2]);
  return out;
};

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`;

// tree

const TREE = /<Tree(?=[\s>])([^>]*)>([\s\S]*?)<\/Tree>/;
const SPEC = /<Spec(?=[\s>/])([^>]*)>/g;

export const activeSpecNodes = (xml: string): number[] => {
  const tree = TREE.exec(xml);
  if (!tree) return [];
  const specs = [...tree[2].matchAll(SPEC)];
  const active = Number(xmlAttrs(tree[1]).activeSpec ?? 1);
  const spec = specs[Math.min(Math.max(active, 1), specs.length) - 1];
  const nodes = spec ? xmlAttrs(spec[1]).nodes : undefined;
  return nodes ? nodes.split(',').filter(Boolean).map(Number) : [];
};

export type TreeNodeInfo = { name: string; rank: number };

export const nodeRank = (n: { isKeystone?: boolean; isNotable?: boolean; isMastery?: boolean; isJewelSocket?: boolean; ascendancyName?: string }) =>
  n.isKeystone ? 5 : n.ascendancyName && n.isNotable ? 4 : n.isNotable ? 3 : n.isMastery ? 2 : n.isJewelSocket ? 1 : 0;

const treePart = (verb: string, ids: number[], rest: string, info: (id: number) => TreeNodeInfo | undefined) => {
  const best = ids
    .map((id, i) => ({ i, info: info(id) }))
    .filter((n): n is { i: number; info: TreeNodeInfo } => !!n.info?.name)
    .sort((x, y) => y.info.rank - x.info.rank || x.i - y.i)[0];
  if (!best) return `${verb} ${plural(ids.length, 'passive')}`;
  return ids.length > 1 ? `${verb} ${best.info.name} (+${plural(ids.length - 1, `${rest} node`)})` : `${verb} ${best.info.name}`;
};

export const treeChangeLabel = (before: number[], after: number[], info: (id: number) => TreeNodeInfo | undefined): string | undefined => {
  const was = new Set(before);
  const now = new Set(after);
  const added = after.filter((id) => !was.has(id));
  const removed = before.filter((id) => !now.has(id));
  if (added.length && removed.length) return `${treePart('Allocated', added, 'path', info)}, ${treePart('deallocated', removed, 'other', info)}`;
  if (added.length) return treePart('Allocated', added, 'path', info);
  if (removed.length) return treePart('Deallocated', removed, 'dependent', info);
  return undefined;
};

// skills

type SkillGroup = { attrs: Record<string, string>; gems: Record<string, string>[] };

const SKILLS = /<Skills(?=[\s>])([^>]*)>([\s\S]*?)<\/Skills>/;
const SKILL_SET = /<SkillSet(?=[\s>])([^>]*)>([\s\S]*?)<\/SkillSet>/g;
const SKILL = /<Skill(?=[\s>/])([^>]*?)(?:\/>|>([\s\S]*?)<\/Skill>)/g;
const GEM = /<Gem(?=[\s>/])([^>]*)>/g;

export const activeSkillGroups = (xml: string): SkillGroup[] => {
  const skills = SKILLS.exec(xml);
  if (!skills) return [];
  const active = xmlAttrs(skills[1]).activeSkillSet;
  const sets = [...skills[2].matchAll(SKILL_SET)];
  const body = sets.length ? (sets.find((s) => xmlAttrs(s[1]).id === active) ?? sets[0])[2] : skills[2];
  return [...body.matchAll(SKILL)].map((s) => ({ attrs: xmlAttrs(s[1]), gems: [...(s[2] ?? '').matchAll(GEM)].map((g) => xmlAttrs(g[1])) }));
};

const isSupport = (g: Record<string, string>) => /SupportGem|\/Support/.test(g.gemId ?? '') || (g.skillId ?? '').startsWith('Support');
const gemName = (g: Record<string, string> | undefined) => g?.nameSpec || g?.skillId || 'a gem';
const groupName = (g: SkillGroup) => g.attrs.label || (g.gems.length ? gemName(g.gems.find((x) => !isSupport(x)) ?? g.gems[0]) : 'an empty socket group');
const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);
const on = (v: string | undefined) => v !== 'false' && v !== 'nil';

const gemChange = (b: Record<string, string>, a: Record<string, string>, group: string): string => {
  const where = group === gemName(a) || group === gemName(b) ? '' : ` in ${group}`;
  if (b.nameSpec !== a.nameSpec || b.skillId !== a.skillId) return `Replaced ${gemName(b)} with ${gemName(a)}${where}`;
  if (b.enabled !== a.enabled) return `${on(a.enabled) ? 'Enabled' : 'Disabled'} ${gemName(a)}${where}`;
  if (b.level !== a.level) return `Set ${gemName(a)} to level ${a.level}`;
  if (b.quality !== a.quality) return `Set ${gemName(a)} to quality ${a.quality}`;
  if (b.count !== a.count) return `Set ${gemName(a)} count to ${a.count}`;
  return `Edited ${gemName(a)}${where}`;
};

const groupChange = (b: SkillGroup, a: SkillGroup): string => {
  const group = groupName(a);
  if (a.gems.length > b.gems.length) {
    const i = a.gems.findIndex((g, j) => !same(g, b.gems[j]));
    return `Added ${gemName(a.gems[i])} to ${groupName(b)}`;
  }
  if (a.gems.length < b.gems.length) {
    const i = b.gems.findIndex((g, j) => !same(g, a.gems[j]));
    return `Removed ${gemName(b.gems[i])} from ${groupName(b)}`;
  }
  const i = a.gems.findIndex((g, j) => !same(g, b.gems[j]));
  if (i >= 0) return gemChange(b.gems[i], a.gems[i], group);
  const ba = b.attrs;
  const aa = a.attrs;
  if (ba.label !== aa.label) return aa.label ? `Renamed ${groupName(b)} to ${aa.label}` : `Cleared the label of ${group}`;
  if (ba.slot !== aa.slot) return aa.slot ? `Socketed ${group} in ${aa.slot}` : `Unsocketed ${group}`;
  if (ba.enabled !== aa.enabled) return `${on(aa.enabled) ? 'Enabled' : 'Disabled'} ${group}`;
  if (ba.includeInFullDPS !== aa.includeInFullDPS) return on(aa.includeInFullDPS) ? `Included ${group} in Full DPS` : `Excluded ${group} from Full DPS`;
  if (ba.groupCount !== aa.groupCount) return `Set ${group} count to ${aa.groupCount}`;
  return `Edited ${group}`;
};

export const skillsChangeLabel = (before: string, after: string): string | undefined => {
  const b = activeSkillGroups(before);
  const a = activeSkillGroups(after);
  if (a.length > b.length) {
    if (a.length - b.length > 1) return `Added ${plural(a.length - b.length, 'socket group')}`;
    const i = a.findIndex((g, j) => !same(g, b[j]));
    return a[i].gems.length || a[i].attrs.label ? `Added socket group ${groupName(a[i])}` : 'Added a socket group';
  }
  if (a.length < b.length) {
    if (b.length - a.length > 1) return `Removed ${plural(b.length - a.length, 'socket group')}`;
    const i = b.findIndex((g, j) => !same(g, a[j]));
    return `Removed ${b[i].gems.length || b[i].attrs.label ? `socket group ${groupName(b[i])}` : 'an empty socket group'}`;
  }
  const changed = a.flatMap((g, i) => (same(g, b[i]) ? [] : [i]));
  if (!changed.length) return undefined;
  if (changed.length > 1 && same([...a].map((g) => JSON.stringify(g)).sort(), [...b].map((g) => JSON.stringify(g)).sort())) {
    return `Moved ${groupName(b[changed[0]])}`;
  }
  return groupChange(b[changed[0]], a[changed[0]]);
};

// items

export const itemTextName = (text: string) =>
  text
    .split(/\r?\n/)
    .map((l) => l.trim())
    .find((l) => l && !/^(Item Class|Rarity):/i.test(l)) ?? 'an item';

// config

type ConfigVar = { label?: string; type: string; list?: { label: string; value: unknown }[] };

export const configLabel = (key: string, opt: ConfigVar | undefined) => (opt?.label ?? key).replace(/\^x[\dA-F]{6}|\^\d/gi, '').replace(/[\s:?]+$/, '');

export const configChangeLabel = (key: string, value: boolean | number | string, remove: boolean, opt: ConfigVar | undefined) => {
  const name = configLabel(key, opt);
  if (opt?.type === 'check' || typeof value === 'boolean') return `${value ? 'Turned on' : 'Turned off'} ${name}`;
  if (opt?.type === 'list') return `Set ${name} to ${opt.list?.find((o) => o.value === value)?.label ?? String(value)}`;
  if (remove) return `Cleared ${name}`;
  return `Set ${name} to ${String(value)}`;
};
