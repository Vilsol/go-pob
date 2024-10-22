import { syncWrap } from './go/worker';
import type { exposition } from './types';

export type SkillGemCacheItem = Omit<exposition.SkillGem, 'CalculateStuff'>;
let skillGemCache: SkillGemCacheItem[];
let skillGemPromise: Promise<typeof skillGemCache>;
export const GetSkillGems = async (): Promise<typeof skillGemCache> => {
  if (!syncWrap) {
    return [];
  }

  if (skillGemCache !== undefined) {
    return skillGemCache;
  }

  if (skillGemPromise === undefined) {
    skillGemPromise = new Promise(async (resolve) => {
      if (!syncWrap) {
        resolve([]);
        return;
      }

      const gems = await syncWrap.GetSkillGems();
      const len = await gems.length;

      const allGems: Promise<SkillGemCacheItem>[] = [];
      for (let i = 0; i < len; i++) {
        allGems.push(
          new Promise(async (innerResolve) => {
            const gem = gems[i];
            innerResolve({
              Vaal: await gem.Vaal,
              Base: await gem.Base,
              GemType: await gem.GemType,
              ID: await gem.ID,
              MaxLevel: await gem.MaxLevel,
              Support: await gem.Support
            });
          })
        );
      }

      skillGemCache = await Promise.all(allGems);
      resolve(skillGemCache);
    });
  }

  return skillGemPromise;
};
