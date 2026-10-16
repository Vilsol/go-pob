import { displayStats, minionDisplayStats } from '$lib/display/stats';
import { pobPlain } from '$lib/design/pob_text';

export type Change = 'better' | 'worse' | 'changed';

const LOWER_IS_BETTER = new Map([...displayStats, ...minionDisplayStats].filter((s) => s.stat).map((s) => [s.stat!, !!s.lowerIsBetter]));

const NUMBER = /-?\d[\d,]*(?:\.\d+)?/;
const numberOf = (text: string) => {
  const m = NUMBER.exec(pobPlain(text));
  return m ? parseFloat(m[0].replace(/,/g, '')) : undefined;
};

export const changeOf = (prev: string, next: string, stat?: string): Change | undefined => {
  if (prev === next) return undefined;
  const [a, b] = [numberOf(prev), numberOf(next)];
  const lower = stat === undefined ? undefined : LOWER_IS_BETTER.get(stat);
  if (a === undefined || b === undefined || a === b || lower === undefined) return 'changed';
  return b > a !== lower ? 'better' : 'worse';
};
