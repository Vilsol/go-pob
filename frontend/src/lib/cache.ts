import { syncWrap } from './go/worker';
import type { exposition } from './types';

let skillGemCache: exposition.SkillGem[];
export const GetSkillGems = async (): Promise<exposition.SkillGem[]> => {
  if (!syncWrap) {
    return [];
  }

  if (skillGemCache !== undefined) {
    return skillGemCache;
  }

  const gems = await syncWrap.GetSkillGems();
  const len = await gems.length;

  const allGems = [];
  for (let i = 0; i < len; i++) {
    allGems.push(
      new Promise(async (resolve) => {
        const gem = gems[i] as unknown as exposition.SkillGem;
        resolve({
          Vaal: await gem.Vaal,
          Base: await gem.Base,
          GemType: await gem.GemType,
          ID: await gem.ID,
          MaxLevel: await gem.MaxLevel
        });
      })
    );
  }

  skillGemCache = (await Promise.all(allGems)) as exposition.SkillGem[];
  return skillGemCache;
};
