import { syncWrap } from './go/worker';
import type { exposition } from './types';

let skillGemCache: exposition.SkillGem[];
let skillGemPromise: Promise<exposition.SkillGem[]>;
export const GetSkillGems = async (): Promise<exposition.SkillGem[]> => {
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

      const allGems = [];
      for (let i = 0; i < len; i++) {
        allGems.push(
          new Promise(async (innerResolve) => {
            const gem = gems[i] as unknown as exposition.SkillGem;
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

      skillGemCache = (await Promise.all(allGems)) as exposition.SkillGem[];
      resolve(skillGemCache);
    });
  }

  return skillGemPromise;
};
