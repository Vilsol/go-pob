const baseColorCodes = {
  NORMAL: '#C8C8C8',
  MAGIC: '#8888FF',
  RARE: '#FFFF77',
  UNIQUE: '#AF6025',
  RELIC: '#60C060',
  GEM: '#1AA29B',
  PROPHECY: '#B54BFF',
  CURRENCY: '#AA9E82',
  CRAFTED: '#B8DAF1',
  CUSTOM: '#5CF0BB',
  SOURCE: '#88FFFF',
  UNSUPPORTED: '#F05050',
  WARNING: '#FF9922',
  TIP: '#80A080',
  FIRE: '#B97123',
  COLD: '#3F6DB3',
  LIGHTNING: '#ADAA47',
  CHAOS: '#D02090',
  POSITIVE: '#33FF77',
  NEGATIVE: '#DD0022',
  OFFENCE: '#E07030',
  DEFENCE: '#8080E0',
  SCION: '#FFF0F0',
  MARAUDER: '#E05030',
  RANGER: '#70FF70',
  WITCH: '#7070FF',
  DUELIST: '#E0E070',
  TEMPLAR: '#C040FF',
  SHADOW: '#30C0D0',
  MAINHAND: '#50FF50',
  MAINHANDBG: '#071907',
  OFFHAND: '#B7B7FF',
  OFFHANDBG: '#070719',
  SHAPER: '#55BBFF',
  ELDER: '#AA77CC',
  FRACTURED: '#A29160',
  ADJUDICATOR: '#E9F831',
  BASILISK: '#00CB3A',
  CRUSADER: '#2946FC',
  EYRIE: '#AAB7B8',
  CLEANSING: '#F24141',
  TANGLE: '#038C8C',
  CHILLBG: '#151e26',
  FREEZEBG: '#0c262b',
  SHOCKBG: '#191732',
  SCORCHBG: '#270b00',
  BRITTLEBG: '#00122b',
  SAPBG: '#261500',
  SCOURGE: '#FF6E25',
  GRAY: '#9F9F9F',
  WHITE: '#FFFFFF'
};

const withStatsColorCodes = {
  ...baseColorCodes,
  STRENGTH: baseColorCodes.MARAUDER,
  DEXTERITY: baseColorCodes.RANGER,
  INTELLIGENCE: baseColorCodes.WITCH
};

export const colorCodes = {
  ...withStatsColorCodes,
  LIFE: withStatsColorCodes.MARAUDER,
  MANA: withStatsColorCodes.WITCH,
  ES: withStatsColorCodes.SOURCE,
  WARD: withStatsColorCodes.RARE,
  EVASION: withStatsColorCodes.POSITIVE,
  RAGE: withStatsColorCodes.WARNING,
  PHYS: withStatsColorCodes.NORMAL
};

const colorRegex = new RegExp(/\^#([0-9A-F]{6})?/g);

export const formatColors = (s: string): string => {
  let result = '';

  let openCount = 0;
  let lastIndex = 0;
  const matches = s.matchAll(colorRegex);
  for (const match of matches) {
    result += s.substring(lastIndex, match.index);

    if (match[1]) {
      openCount++;
      result += `<span style='color: #${match[1]}'>`;
    } else {
      if (openCount > 0) {
        openCount--;
        result += '</span>';
      }
    }

    lastIndex = (match.index || 0) + match[0].length;
  }

  result += s.substring(lastIndex, s.length);

  result += '</span>'.repeat(openCount);

  return '<span>' + result + '</span>';
};
