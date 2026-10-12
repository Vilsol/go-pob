import { configure } from '@zenfs/core';
import { IndexedDB } from '@zenfs/dom';

export const configureStorage = async () => {
  const [cache, builds, settings] = await Promise.all(['cache', 'builds', 'settings'].map((storeName) => IndexedDB.create({ storeName })));
  await configure({ mounts: { '/cache': cache, '/builds': builds, '/settings': settings } });
};
