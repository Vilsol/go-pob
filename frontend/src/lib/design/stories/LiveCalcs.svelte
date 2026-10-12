<script lang="ts">
  import CalcsPage from '../CalcsPage.svelte';
  import type { BuildFixture } from '../fixtures';

  type Props = { fx: BuildFixture; showAll?: boolean; pin?: string; moddebug?: () => Promise<string> };
  let { fx, showAll = false, pin, moddebug }: Props = $props();

  let s = $state({ group: fx.mainSocketGroup, controls: structuredClone(fx.mainSkill), mode: 'Effective DPS', showAll, q: '' });
</script>

<CalcsPage
  calcs={fx.calcs}
  socketGroups={fx.socketGroups}
  mainSocketGroup={s.group}
  controls={s.controls}
  mode={s.mode}
  {pin}
  {moddebug}
  bind:showAll={s.showAll}
  bind:q={s.q}
  onmaingroup={(i: number) => (s.group = i)}
  oncontrol={(key: 'MainActiveSkill' | 'SkillPart' | 'StageCount' | 'MineCount' | 'Minion' | 'MinionSkill', value: number | undefined) => {
    if (!s.controls) return;
    if (key === 'StageCount' || key === 'MineCount') s.controls[key] = value === undefined ? '' : String(value);
    else if (value !== undefined) s.controls[key] = value;
  }}
  onmode={(m: string) => (s.mode = m)} />
