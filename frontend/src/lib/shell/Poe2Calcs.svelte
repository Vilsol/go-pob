<script lang="ts">
  import CalcsPage from '$lib/design/CalcsPage.svelte';
  import { calcSections2 } from '$lib/calcs/calc_sections2';
  import { calcsView2, type CalcsActor2 } from '$lib/display/calcs2';
  import type { Poe2Calc, Poe2CalcsPage } from '$lib/go/sync_worker2';
  import type { calculator } from '$lib/types';

  // CalcsTab.lua (PoB-PoE2) over the main calculation: calculator2 runs no CALCS pass, so the tab's own skill
  // number and buff mode are the sidebar's
  let {
    calc,
    page,
    busy = false,
    onmaingroup,
    onmainskill
  }: {
    calc: Partial<Poe2Calc>;
    page: Poe2CalcsPage | undefined;
    busy?: boolean;
    onmaingroup: (index: number) => void;
    onmainskill: (index: number) => void;
  } = $props();

  const groups = $derived(calc.socketGroups ?? []);
  const mainGroup = $derived(groups[(calc.mainSocketGroup ?? 1) - 1]);
  const view = $derived.by(() => {
    if (!page) return undefined;
    const output: CalcsActor2['output'] = { ...calc.output, ...page.flags };
    if (page.mainHand) output.MainHand = page.mainHand;
    if (page.offHand) output.OffHand = page.offHand;
    return calcsView2(calcSections2, { output, skillFlags: calc.skillFlags ?? {}, skillData: calc.skillData ?? {}, mods: page.mods });
  });
  const controls = $derived(
    mainGroup && mainGroup.skills.length > 0
      ? ({ Skills: mainGroup.skills, MainActiveSkill: calc.mainActiveSkill || 1 } as calculator.MainSkillControls)
      : undefined
  );
</script>

{#if view}
  <div class="flex min-h-0 min-w-0 flex-1 flex-col" class:opacity-60={busy} aria-label="Calcs">
    <CalcsPage
      calcs={view}
      socketGroups={groups.length ? groups.map((g) => g.label) : ['<No skills added yet>']}
      mainSocketGroup={Math.max(0, Math.min(calc.mainSocketGroup ?? 1, Math.max(groups.length, 1)) - 1)}
      {controls}
      onmaingroup={(i: number) => onmaingroup(i + 1)}
      oncontrol={(field: string, v: number | undefined) => {
        if (field === 'MainActiveSkill' && v !== undefined) onmainskill(v);
      }} />
  </div>
{:else}
  <p class="note p-5" role="status">Calculating…</p>
{/if}
