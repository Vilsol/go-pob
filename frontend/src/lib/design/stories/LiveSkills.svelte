<script lang="ts">
  import SkillsPage from '../SkillsPage.svelte';
  import { gemCatalogue, type BuildFixture, type SocketGroupFixture, type SkillGemFixture } from '../fixtures';
  import { gemDeltas } from '../fixtures/items';
  import type { GemOptions, SkillSetOp } from '../skills';

  type Props = { fx: BuildFixture; selected?: number };
  let { fx, selected = fx.mainSocketGroup }: Props = $props();

  let s = $state({
    sets: structuredClone(fx.skills.sets),
    set: fx.skills.activeSet,
    selected,
    main: fx.mainSocketGroup,
    sortOn: true,
    sortStat: 'Combined DPS',
    options: { defaultLevel: 'normalMaximum', defaultQuality: 0, showSupport: 'ALL', showAlt: false, showLegacy: false } as GemOptions
  });
  const moveItem = <T,>(list: T[], from: number, to: number) => list.splice(to, 0, ...list.splice(from, 1));
  const follow = (i: number, from: number, to: number) => (i === from ? to : from < i && i <= to ? i - 1 : to <= i && i < from ? i + 1 : i);
  const groups = () => s.sets[s.set].groups;
  const slots = [
    'None',
    'Weapon 1',
    'Weapon 2',
    'Weapon 1 Swap',
    'Weapon 2 Swap',
    'Helmet',
    'Body Armour',
    'Gloves',
    'Boots',
    'Amulet',
    'Ring 1',
    'Ring 2',
    'Belt'
  ];
</script>

<SkillsPage
  sets={s.sets}
  set={s.set}
  selected={s.selected}
  mainGroup={s.main}
  catalogue={gemCatalogue}
  deltas={gemDeltas}
  {slots}
  bind:sortOn={s.sortOn}
  bind:sortStat={s.sortStat}
  onset={(i: number) => {
    s.set = i;
    s.selected = 0;
  }}
  onsets={(op: SkillSetOp) => {
    if (op.op === 'new') s.sets.push({ title: op.title, groups: [] });
    else if (op.op === 'copy') s.sets.push({ title: op.title, groups: structuredClone($state.snapshot(s.sets[op.index].groups)) });
    else if (op.op === 'rename') s.sets[op.index].title = op.title;
    else if (s.sets.length > 1) {
      s.sets.splice(op.index, 1);
      if (op.index === s.set) s.set = Math.max(0, op.index - 1);
      else if (op.index < s.set) s.set--;
    }
  }}
  onselect={(i: number) => {
    s.selected = i;
  }}
  ongroup={(i: number, patch: Partial<SocketGroupFixture>) => {
    Object.assign(groups()[i], patch);
  }}
  ongem={(i: number, g: number, patch: Partial<SkillGemFixture>) => {
    Object.assign(groups()[i].gems[g], patch);
  }}
  onaddgem={(i: number, gem: SkillGemFixture) => {
    groups()[i].gems.push(gem);
  }}
  onremovegem={(i: number, g: number) => {
    groups()[i].gems.splice(g, 1);
  }}
  onreordergroups={(from: number, to: number) => {
    moveItem(groups(), from, to);
    s.selected = follow(s.selected, from, to);
    s.main = follow(s.main, from, to);
  }}
  onreordergems={(from: number, to: number) => {
    moveItem(groups()[s.selected].gems, from, to);
  }}
  onnewgroup={() => {
    groups().push({ label: '', slot: '', enabled: true, inFull: true, count: 1, sockets: '', gems: [] });
    s.selected = groups().length - 1;
  }}
  onmain={(i: number) => (s.main = i)}
  ondeletegroup={(i: number) => {
    groups().splice(i, 1);
    s.main = s.main === i ? -1 : s.main > i ? s.main - 1 : s.main;
    s.selected = 0;
  }}
  ondeleteall={() => {
    s.sets[s.set].groups = [{ label: '', slot: '', enabled: true, inFull: true, count: 1, sockets: '', gems: [] }];
    s.main = 0;
    s.selected = 0;
  }}
  options={s.options}
  onoption={(patch: Partial<GemOptions>) => Object.assign(s.options, patch)}
  editor={{
    ongemcount: (i: number, g: number, count: number) => {
      groups()[i].gems[g].count = count;
    }
  }} />
