import type { calculator } from '$lib/types';
import type { PlayerProgress } from '$lib/display/points';
import type { Sidebar } from '$lib/display/sidebar';
import type { StatCompareSection } from '$lib/skill_tree/stat_compare';
import spectre from './0243-necromancer-raise-spectre.json';
import fireball from './0041-pathfinder-vaal-fireball.json';
import nova from './0029-inquisitor-spark-of-the-nova.json';
import gems from './gems.json';
import mods from './mods.json';
import about from './about.json';
import notes from './notes.json';

export const notesSamples: { source: string; text: string }[] = notes;

export type ModLine = { text: string; kind: string };

export type ItemTip = {
  influence: string[];
  title: string[];
  lines: { text: string; sep: boolean }[];
  /* ItemsTab.lua:4748-4794 */
  children?: { text: string; sep: boolean }[][];
  /* PassiveTreeView.lua:1607-1627 */
  childrenShown?: boolean;
};

export type ItemSummary = {
  name: string;
  base: string;
  rarity: 'normal' | 'magic' | 'rare' | 'unique' | 'relic';
  sockets: string;
  implicits: ModLine[];
  explicits: ModLine[];
  flask: boolean;
  fits: string[];
  tooltip?: ItemTip;
};

export type StatDiff = { header: string; slot?: string; sections: StatCompareSection[] };

export type ItemSet = {
  id: string;
  title: string;
  swap: boolean;
  slots: { slot: string; item: number; active: boolean }[];
  /* ItemsTab.lua:4138-4147 */
  tip?: string[];
};

export type ItemsFixture = {
  activeSet: string;
  items: Record<string, ItemSummary>;
  sets: ItemSet[];
  impact: Record<string, StatDiff>;
  jewels: { node: number; item: number }[];
  /* ItemListControl.lua:242-258 */
  stash: { item: number; impact?: StatDiff; used?: string }[];
  paste: { text: string; item: ItemSummary; impact?: StatDiff };
};

export type GemAttr = 'str' | 'dex' | 'int' | 'white';

/* SkillsTab.lua:1007-1012, SkillsTab.lua:1014-1050 */
export type SkillGemFixture = {
  name: string;
  level: number;
  quality: number;
  enabled: boolean;
  support: boolean;
  attr: GemAttr;
  count: number;
  error?: string;
  globals?: { label: string; on: boolean }[];
  /* CompareTab.lua:4032-4052 */
  imbued?: boolean;
};

export type SocketGroupFixture = {
  label: string;
  slot: string;
  enabled: boolean;
  inFull: boolean;
  count: number;
  sockets: string;
  gems: SkillGemFixture[];
};

export type SkillsFixture = { activeSet: number; sets: { title: string; groups: SocketGroupFixture[] }[] };

export type ConfigValue = boolean | number | string;

export type ModVerdict = { k: 'ok' | 'bad' | 'comment' | 'empty'; parsed: string };
export type CustomCell = { text: string } & ModVerdict;
export type CustomGroup = { title: string; enabled: boolean; cells: CustomCell[] };

export type ConfigFixture = {
  sets: string[];
  activeSet: number;
  inputs: Record<string, ConfigValue>;
  placeholders: Record<string, number | string>;
  relevance: calculator.ConfigRelevance;
  custom: CustomGroup[];
};

export type { CalcCell, CalcRow, CalcBlock, CalcBreakdown, CalcSectionView as CalcSectionFixture, CalcsView as CalcsFixture } from '$lib/display/calcs';
import type { CalcsView } from '$lib/display/calcs';

export type CompareStat = { key: string; label: string; v: number; text: string; lower: boolean; pc: boolean; diff?: string };
export type PowerRow = {
  cat: 'items' | 'tree' | 'config' | 'gems' | 'supports';
  what: string;
  detail: string;
  tone: string;
  points?: number;
  impact: number[];
};
export type Metrics = Record<string, number>;
export type CompareSide = {
  name: string;
  className: string;
  ascendClassName: string;
  level: number;
  points: number;
  metrics: Metrics;
  stats: CompareStat[][];
  configSet: string;
  power: PowerRow[];
};
export type ConfigKind = 'check' | 'number' | 'list';
export type CompareItemSide = { name: string; rarity: ItemSummary['rarity']; line: string; base?: string } | null;
export type CompareFixture = {
  a: CompareSide;
  b: CompareSide & { skills: SkillsFixture };
  items: { slot: string; a: CompareItemSide; b: CompareItemSide; impactA?: StatCompareSection[]; impactB?: StatCompareSection[] }[];
  /* CompareTab.lua:1372-1436 */
  jewels?: { slot: string; label: string; node: number; a: CompareItemSide; b: CompareItemSide; aWarn: boolean; bWarn: boolean }[];
  tree: { nodes: { id: number; name: string; kind: string; owner: 'a' | 'b' | 'both'; x: number | null; y: number | null }[]; edges: [number, number][] };
  /* CompareTab.lua:1128-1160 */
  config: {
    key: string;
    section: string;
    label: string;
    a: string;
    b: string;
    diff?: boolean;
    impactA: number[];
    impactB: number[];
    kind?: ConfigKind;
    options?: { value: string | number; label: string }[];
    aValue?: ConfigValue;
    bValue?: ConfigValue;
  }[];
};

export type TreeNodeFx = {
  id: number;
  k: 's' | 'n' | 'k' | 'm' | 'j' | 'c';
  x: number;
  y: number;
  name: string;
  a?: 1;
  stats?: string[];
  recipe?: string[];
  pts?: number;
  next?: number;
  lose?: number;
  loseIds?: number[];
  effects?: string[][];
  effect?: number;
  picked?: number;
};
export type TreeSocketFx = { id: number; pts: number; notables: string[]; keystones: string[]; smalls: number; jewel?: string };
export type TreeFixture = {
  version: string;
  classes: string[];
  ascendancies: string[];
  specs: string[];
  activeSpec: number;
  aspect: number;
  radius: number;
  nodes: TreeNodeFx[];
  edges: [number, number][];
  tips: Record<string, { header: string; sections: StatCompareSection[] }[]>;
  sockets: TreeSocketFx[];
};

export type BuildFixture = {
  source: string;
  name: string;
  className: string;
  ascendClassName: string;
  level: number;
  mainSocketGroup: number;
  socketGroups: string[];
  mainSkill: calculator.MainSkillControls;
  progress: PlayerProgress;
  sidebar: Sidebar;
  tree: TreeFixture;
  shareCode: string;
  items: ItemsFixture;
  skills: SkillsFixture;
  config: ConfigFixture;
  calcs: CalcsView;
  compare: CompareFixture;
};

const as = (json: unknown) => json as BuildFixture;

export const spectreBuild = as(spectre);
export const fireballBuild = as(fireball);
export const novaBuild = as(nova);

export const gemCatalogue = (gems as [string, GemAttr, number, string, string, number][]).map(([name, attr, support, searchName, tags, levelReq]) => ({
  name,
  attr,
  support: support === 1,
  searchName,
  tags: tags ? tags.split(' ') : [],
  levelReq
}));

export const modVerdicts: Record<string, ModVerdict> = Object.fromEntries(
  (mods as [string, ModVerdict['k'], string][]).map(([text, k, parsed]) => [text, { k, parsed }])
);

export const buildFixtures = { spectre: spectreBuild, fireball: fireballBuild, nova: novaBuild };

export type AboutInfo = {
  version: string;
  go: string;
  gameData: string;
  tree: string;
  settings: [string, string][];
  deps: [string, string][];
};
export const aboutInfo = about as AboutInfo;
