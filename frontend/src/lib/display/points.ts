// Build.lua:880-923

// src/Modules/Build.lua:711-746
const acts = [
  { level: 1, questPoints: 0 },
  { level: 12, questPoints: 2 },
  { level: 22, questPoints: 4 },
  { level: 32, questPoints: 6 },
  { level: 40, questPoints: 7 },
  { level: 44, questPoints: 9 },
  { level: 50, questPoints: 12 },
  { level: 54, questPoints: 15 },
  { level: 60, questPoints: 18 },
  { level: 64, questPoints: 20 },
  { level: 67, questPoints: 23 }
];

// src/Modules/Build.lua:748-751
const actExtra = (act: number, extra: number) => (act > 2 ? extra : 0);

export interface PlayerProgress {
  used: number;
  usedMax: number;
  ascUsed: number;
  ascMax: number;
  secondaryAscUsed: number;
  secondaryAscMax: number;
  level: number;
  act: number | 'Endgame';
  questPoints: number;
  extraSkillPoints: number;
  labSuggest: string;
  warnings: string[];
}

// src/Modules/Build.lua:880-923
export const estimatePlayerProgress = (pointsUsed: [number, number, number, number], extraPoints: number | undefined): PlayerProgress => {
  const [used, ascUsed, secondaryAscUsed] = pointsUsed;
  const extra = extraPoints ?? 0;
  const usedMax = 99 + 23 + extra;
  const ascMax = 8;
  const secondaryAscMax = 8;

  let act = 0;
  let level = 1;
  do {
    act++;
    level = Math.min(Math.max(used + 1 - acts[act - 1].questPoints - actExtra(act, extra), acts[act - 1].level), 100);
  } while (!(act === 11 || level <= acts[act].level));

  const labSuggest =
    level < 33
      ? ''
      : level < 55
        ? 'Labyrinth: Normal Lab'
        : level < 68
          ? 'Labyrinth: Cruel Lab'
          : level < 75
            ? 'Labyrinth: Merciless Lab'
            : level < 90
              ? 'Labyrinth: Uber Lab'
              : '';

  const warnings: string[] = [];
  if (used > usedMax) warnings.push('You have too many passive points allocated');
  if (ascUsed > ascMax) warnings.push('You have too many ascendancy points allocated');
  if (secondaryAscUsed > secondaryAscMax) warnings.push('You have too many secondary ascendancy points allocated');

  return {
    used,
    usedMax,
    ascUsed,
    ascMax,
    secondaryAscUsed,
    secondaryAscMax,
    level,
    act: level < 90 && act <= 10 ? act : 'Endgame',
    questPoints: acts[act - 1].questPoints,
    extraSkillPoints: actExtra(act, extra),
    labSuggest,
    warnings
  };
};

// src/Modules/Build.lua:918-921
export const progressTooltip = (p: PlayerProgress): string =>
  `Required Level: ${p.level}\nEstimated Progress:\nAct: ${p.act}\nQuestpoints: ${p.questPoints}\nExtra Skillpoints: ${p.extraSkillPoints}${p.labSuggest ? `\n${p.labSuggest}` : ''}`;

/* Build.lua:892-896 */
// src/Modules/Build.lua:892-896
export const autoLevel = (p: PlayerProgress, autoMode: boolean, characterLevel: number): number | undefined =>
  autoMode && characterLevel !== p.level ? p.level : undefined;
