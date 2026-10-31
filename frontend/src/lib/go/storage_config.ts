import { configure, mount, umount } from '@zenfs/core';
import { IndexedDB } from '@zenfs/dom';
import { DESKTOP } from '$lib/pwa/desktop';

export const configureStorage = async () => {
  if (DESKTOP) return;
  const [cache, builds, settings] = await Promise.all(['cache', 'builds', 'settings'].map((storeName) => IndexedDB.create({ storeName })));
  await configure({ mounts: { '/cache': cache, '/builds': builds, '/settings': settings } });
};

// zenfs reads a database's file table once, at mount; the new mount is ready before the old one goes, so no read
// finds the bucket missing
export const remountStore = async (point: string, storeName: string) => {
  const fresh = await IndexedDB.create({ storeName });
  await fresh.ready();
  umount(point);
  mount(point, fresh);
};
