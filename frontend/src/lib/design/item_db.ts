export type ItemDBKind = 'UNIQUE' | 'RARE';

export type ItemDBLists = {
  slots: string[];
  types: string[];
  leagues: string[];
  requirements: string[];
  obtainable: string[];
  searchModes: string[];
  sortStats: { stat: string; label: string }[];
};

export type ItemDBFilterInput = {
  slot: number;
  type: number;
  league: number;
  requirement: number;
  obtainable: number;
  search: string;
  searchMode: number;
};

export const defaultItemDBFilter = (): ItemDBFilterInput => ({ slot: 1, type: 1, league: 1, requirement: 1, obtainable: 1, search: '', searchMode: 1 });

export type ItemDBRowPlain = {
  name: string;
  title: string;
  base: string;
  rarity: string;
  type: string;
  slot: string;
  level: number;
  league: string;
  source: string;
  power: number;
};
