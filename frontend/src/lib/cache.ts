import { syncWrap } from './go/worker';
import type { exposition } from './types';

export type SkillGemCacheItem = exposition.SkillGem;

let skillGemPromise: Promise<SkillGemCacheItem[]> | undefined;

export const GetSkillGems = (): Promise<SkillGemCacheItem[]> => {
  if (!syncWrap) {
    return Promise.resolve([]);
  }

  skillGemPromise ??= syncWrap.GetSkillGems();
  return skillGemPromise;
};
