import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const CORPUS = resolve(import.meta.dirname, '../../../testdata/corpus-3.29');

export const BUILD_SET: { file: string; why: string }[] = [
  { file: '0008-necromancer-vaal-summon-skeletons', why: 'minion main skill (env.Minion present)' },
  { file: '0516-reliquarian-kinetic-fusillade', why: 'totem (ballista) main skill' },
  { file: '0037-ascendant-fire-trap-of-blasting-2', why: 'trap main skill' },
  { file: '4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne', why: 'triggered main skill (skillFlags.triggered)' },
  { file: '13276-ladder-cuba_5717.cubamrc', why: 'dual wield (DualWielding condition), 11 cluster jewels, a tincture' },
  { file: '0339-reliquarian-shield-crush-of-the-chieftain-2', why: 'large tree (142 nodes) with cluster + timeless jewels, 2 item sets / 4 skill sets' },
  { file: '0667-guardian-summon-raging-spirit', why: 'many item sets / skill sets (24 / 24)' },
  { file: '0977-luminary-purity-of-ice-2', why: 'party tab: the largest imported party buffs' },
  { file: '0009-trickster-cobra-lash', why: 'config-heavy: 52 config inputs, 19 item sets' },
  { file: '0280-elementalist-fireball', why: 'small character: level 61, 38 allocated nodes' },
  { file: '0444-reliquarian-static-strike-2', why: 'flask + tincture setup: 5 active flasks, 1 tincture' },
  { file: '1591-necromancer-summon-holy-relic-of-conviction-15', why: 'slowest main-skill calculation in the corpus' },
  { file: '0001-necromancer-unknown', why: 'largest build (2.8 MB XML, 38 item sets / 34 skill sets); among the slowest to calculate' }
];

export const buildXml = (file: string) => readFileSync(resolve(CORPUS, `${file}.xml`), 'utf8');
