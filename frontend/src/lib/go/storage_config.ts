import { configure } from '@zenfs/core';
import { IndexedDB } from '@zenfs/dom';
import { DESKTOP } from '$lib/pwa/desktop';

export const configureStorage = async () => {
  if (DESKTOP) return;
  const [cache, builds, settings] = await Promise.all(['cache', 'builds', 'settings'].map((storeName) => IndexedDB.create({ storeName })));
  await configure({ mounts: { '/cache': cache, '/builds': builds, '/settings': settings } });
};
