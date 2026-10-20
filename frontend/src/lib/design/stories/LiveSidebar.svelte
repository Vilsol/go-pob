<script lang="ts">
  import Sidebar from '../Sidebar.svelte';
  import type { BuildFixture } from '../fixtures';
  import { breakdowns } from '../fixtures/shell';
  import type { StatLine } from '$lib/display/stat_list';

  type Control = 'MainActiveSkill' | 'SkillPart' | 'StageCount' | 'MineCount' | 'Minion' | 'MinionSkill';

  type Props = { fx: BuildFixture; collapsed?: boolean; fill?: boolean; pinned?: string[] };
  let { fx, collapsed = false, fill = false, pinned = $bindable([]) }: Props = $props();

  let s = $state({ collapsed, group: fx.mainSocketGroup, controls: { ...fx.mainSkill } });
</script>

<Sidebar
  {fill}
  sidebar={fx.sidebar}
  socketGroups={fx.socketGroups}
  mainSocketGroup={s.group}
  controls={s.controls}
  bind:collapsed={s.collapsed}
  bind:pinned
  breakdown={(l: StatLine) => breakdowns[l.label]}
  onmaingroup={(i: number) => (s.group = i)}
  oncontrol={(field: Control, v: number | undefined) => {
    if (field === 'StageCount' || field === 'MineCount') s.controls[field] = v === undefined ? '' : String(v);
    else s.controls[field] = v ?? 0;
  }} />
