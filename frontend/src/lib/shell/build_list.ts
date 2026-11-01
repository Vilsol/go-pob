import type { SavedBuild } from '$lib/design/fixtures/app';
import { GAME_FILTERS, type GameFilter } from '$lib/design/builds';
import type { storage } from '$lib/types';
import { get } from 'svelte/store';
import { syncWrap } from '$lib/go/worker';
import { currentBuildPath } from '$lib/global.js';
import { defaultCharLevel, defaultGemQuality } from '$lib/display/item_options';
import { opening, unsavedName } from './build_name.svelte';
import { resetAccountSession } from './account_session.svelte';

export const dirOf = (folder: string) => (folder ? `/${folder}` : '');
export const pathOf = (folder: string, name: string) => `${dirOf(folder)}/${name}`;

export const CLASSES = ['All', 'Witch', 'Shadow', 'Marauder', 'Ranger', 'Duelist', 'Templar', 'Scion'];

// the class filter's chips for the game the list shows
export const gameClasses = (game: GameFilter, poe2: string[]) =>
  game === 'poe1' ? CLASSES : game === 'poe2' ? ['All', ...poe2.filter((c) => c !== 'All')] : [...CLASSES, ...poe2.filter((c) => !CLASSES.includes(c))];

export const BUILD_GAME_KEY = 'options:buildListGame';
export const loadBuildGame = (): GameFilter => {
  try {
    const v = globalThis.localStorage?.getItem(BUILD_GAME_KEY);
    return GAME_FILTERS.find((g) => g.id === v)?.id ?? 'all';
  } catch {
    return 'all';
  }
};
export const saveBuildGame = (game: GameFilter) => {
  try {
    globalThis.localStorage?.setItem(BUILD_GAME_KEY, game);
  } catch {
    // empty
  }
};

export const toSaved = (e: storage.DirEntry, folder: string, version: string, now: number): SavedBuild => ({
  name: e.Name,
  className: e.Class,
  ascendancy: e.Ascendancy ?? '',
  level: e.Level,
  skill: e.MainSkill ?? '',
  game: e.Game || undefined,
  edited: Math.max(0, Math.round((now - Date.parse(e.LastEdit)) / 60_000)) || 0,
  folder,
  dps: 0,
  ehp: 0,
  life: 0,
  es: 0,
  version
});

export const LIVE_COLUMNS = ['name', 'cls', 'asc', 'level', 'skill', 'edited'] as const;

type List = (dir: string) => Promise<storage.DirEntry[] | undefined>;

export const loadTree = async (list: List, version: string, now = Date.now()) => {
  const builds: SavedBuild[] = [];
  const folders: string[] = [];
  const walk = async (folder: string) => {
    const entries = [...((await list(dirOf(folder))) ?? [])].sort((a, b) => a.Name.localeCompare(b.Name));
    for (const e of entries) {
      if (e.Type === 'dir') {
        const sub = folder ? `${folder}/${e.Name}` : e.Name;
        folders.push(sub);
        await walk(sub);
      } else {
        builds.push(toSaved(e, folder, version, now));
      }
    }
  };
  await walk('');
  return { builds, folders };
};

export const newBuild = () => {
  opening.modFlag = false;
  const made = syncWrap.NewBuild({ level: get(defaultCharLevel), gemQuality: get(defaultGemQuality) });
  unsavedName.imported = false;
  unsavedName.predefined = undefined;
  resetAccountSession();
  currentBuildPath.current = '';
  return made;
};
