import type { AffixControls, AffixSeries, CraftApi } from '../crafting';
import type { DisplayInfo, DisplayMod, ImpactBlock, TipProps } from '../items';
import type { ItemSummary } from '../fixtures';

const mod = (section: string, line: string, kind = 'magic', range?: number): DisplayMod => ({
  section,
  line,
  kind,
  hasRange: range !== undefined,
  range: range ?? 0,
  disabled: false,
  removable: kind === 'crafted',
  removal: kind === 'crafted' ? 'Crafted' : ''
});

const series = (label: string, tiers: number, tooltip: string[] = [], mid: string[] = []) => ({ label, tiers, haveRange: tiers > 0, tooltip, mid });

const slot = (type: string, sel: number, entries: AffixSeries[], sliderVal = 0.5) => ({
  type,
  series: [series('None', 0), ...entries],
  sel,
  sliderShown: sel > 0,
  sliderVal
});
const life = series('+# to maximum Life', 13, ['^713 Tiers', '+(3-129) to maximum Life', 'Level: 1 to 86', 'Tags: resource, life'], ['+70 to maximum Life']);
const evasionEs = series(
  '+# to Evasion Rating/+# to maximum Energy Shield',
  7,
  ['^77 Tiers', '+(6-120) to Evasion Rating', '+(3-48) to maximum Energy Shield'],
  ['+60 to Evasion Rating', '+25 to maximum Energy Shield']
);
const stunRecovery = series(
  '#% increased Evasion and Energy Shield/#% increased Stun and Block Recovery',
  6,
  ['^76 Tiers'],
  ['30% increased Evasion and Energy Shield', '12% increased Stun and Block Recovery']
);
const cold = series('+#% to Cold Resistance', 8, ['^78 Tiers', '+(6-48)% to Cold Resistance'], ['+30% to Cold Resistance']);
const ailments = series(
  '#% chance to Avoid being Ignited, Chilled, Frozen or Shocked while you have at least one Endurance Charge',
  3,
  ['^73 Tiers'],
  ['30% chance to Avoid being Ignited, Chilled, Frozen or Shocked while you have at least one Endurance Charge']
);
const dex = series('+# to Dexterity', 9, ['^79 Tiers', '+(8-55) to Dexterity'], ['+30 to Dexterity']);

const affixes: AffixControls = {
  sorting: true,
  base: 'Carnal Armour',
  bases: ['Carnal Armour', 'Sadist Garb', 'Assassin’s Garb'],
  controls: [
    slot('Prefix', 1, [life, evasionEs, stunRecovery], 0.72),
    slot('Prefix', 2, [life, evasionEs, stunRecovery], 0.6),
    slot('Prefix', 3, [life, evasionEs, stunRecovery], 0.4),
    slot('Suffix', 1, [cold, ailments, dex], 0.86),
    slot('Suffix', 2, [cold, ailments, dex], 0.3),
    slot('Suffix', 0, [cold, ailments, dex])
  ]
};

const rolled = (label: string, v: number) => label.split('/').map((l) => l.replaceAll('#', String(Math.round(5 + 95 * v))));

const blocks: ImpactBlock[] = [
  {
    header: 'Equipping this item in Body Armour will give you:\n(replacing Blunderbore, Astral Plate)',
    slot: 'Body Armour',
    sections: [
      {
        lines: [
          { label: 'Total Life', value: '+94', better: true, percent: '(+1.6%)' },
          { label: 'Energy Shield', value: '+113', better: true, percent: '(+21.4%)' },
          { label: 'Cold Resistance', value: '+42%', better: true },
          { label: 'Armour', value: '-1,322', better: false, percent: '(-58.0%)' },
          { label: 'Hit DPS', value: '-6,982,957', better: false, percent: '(-23.7%)' }
        ]
      }
    ]
  }
];

const item: ItemSummary = {
  name: 'Dusk Shroud',
  base: 'Carnal Armour',
  rarity: 'rare',
  sockets: 'B-B-B-B-B-B',
  implicits: [{ text: '+18 to maximum Mana', kind: 'magic' }],
  explicits: [
    { text: '+94 to maximum Life', kind: 'magic' },
    { text: '+42% to Cold Resistance', kind: 'magic' },
    { text: '+15% to Lightning Resistance', kind: 'crafted' }
  ],
  flask: false,
  fits: ['Body Armour']
};

export const craftCardSample: { item: ItemSummary; info: DisplayInfo; tip: TipProps; blocks: ImpactBlock[]; craft: CraftApi } = {
  item,
  info: {
    quality: 20,
    hasQuality: true,
    variants: [],
    variant: 0,
    altVariants: [],
    sockets: 'B-B-B-B-B-B',
    socketLimit: 6,
    defaultColor: 'G',
    canInfluence: true,
    influence: ['Hunter'],
    showsCatalyst: false,
    catalyst: 0,
    catalystQuality: 0,
    enchants: 1,
    anoints: 0,
    corruptible: true,
    addImplicit: true,
    addCustom: true,
    affixes,
    mods: [
      mod('enchant', '10% increased maximum Life', 'enchant'),
      mod('implicit', '+18 to maximum Mana', 'magic', 0.4),
      mod('explicit', '+94 to maximum Life', 'magic', 0.72),
      mod('explicit', '+98 to Evasion Rating', 'magic', 0.6),
      mod('explicit', '+31 to maximum Energy Shield', 'magic', 0.6),
      mod('explicit', '24% increased Evasion and Energy Shield', 'magic', 0.4),
      mod('explicit', '11% increased Stun and Block Recovery', 'magic', 0.4),
      mod('explicit', '+42% to Cold Resistance', 'magic', 0.86),
      mod('explicit', '20% chance to Avoid being Ignited, Chilled, Frozen or Shocked while you have at least one Endurance Charge', 'magic', 0.3),
      mod('explicit', '+15% to Lightning Resistance', 'crafted', 1)
    ]
  },
  tip: {
    item: 'Dusk Shroud',
    base: 'Carnal Armour',
    rarity: 'rare',
    mods: ['10% increased maximum Life', '+18 to maximum Mana', '+94 to maximum Life', '+42% to Cold Resistance', '+15% to Lightning Resistance'],
    impactTitle: '',
    impact: []
  },
  blocks,
  craft: {
    enchantChoices: () => Promise.resolve(undefined),
    notables: () => Promise.resolve([]),
    sortStats: () =>
      Promise.resolve([
        { stat: 'Life', label: 'Life' },
        { stat: 'EnergyShield', label: 'Energy Shield' }
      ]),
    preview: () => Promise.resolve(undefined),
    enchantSort: () => Promise.resolve([]),
    anointPowers: () => Promise.resolve([]),
    anointCompare: () => Promise.resolve(undefined),
    modCompare: () => Promise.resolve(blocks.map((b) => ({ ...b, header: '\nAdding this mod will give: ', slot: undefined }))),
    affixEdit: () => Promise.resolve(undefined),
    affixRollTooltip: (k, v) => {
      const c = affixes.controls[k];
      return Promise.resolve([...rolled(c.series[c.sel].label, v), '', '^7Affix: Tier 1 (Prime)']);
    },
    affixPowers: () => Promise.resolve([0, 2, 1]),
    rebase: () => Promise.resolve([]),
    rebaseTip: (base) => Promise.resolve({ ...craftCardSample.tip, base })
  }
};
