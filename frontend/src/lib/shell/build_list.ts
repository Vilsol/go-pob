import type { SavedBuild } from '$lib/design/fixtures/app';
import type { storage } from '$lib/types';

export const dirOf = (folder: string) => (folder ? `/${folder}` : '');
export const pathOf = (folder: string, name: string) => `${dirOf(folder)}/${name}`;

export const CLASSES = ['All', 'Witch', 'Shadow', 'Marauder', 'Ranger', 'Duelist', 'Templar', 'Scion'];

export const toSaved = (e: storage.DirEntry, folder: string, version: string, now: number): SavedBuild => ({
  name: e.Name,
  className: e.Class,
  ascendancy: e.Ascendancy ?? '',
  level: e.Level,
  skill: e.MainSkill ?? '',
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
