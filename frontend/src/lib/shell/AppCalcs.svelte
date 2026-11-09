<script lang="ts">
  import { onMount } from 'svelte';
  import { readable } from 'svelte/store';
  import { devMode } from '$lib/global.js';
  import { currentEngine } from '$lib/go/engine';
  import MiniTreeView from '$lib/components/skill-tree/MiniTreeView.svelte';
  import CalcsPage from '$lib/design/CalcsPage.svelte';
  import type { MainSkillField } from '$lib/shell/main_skill';
  import { logError } from '$lib/utils';
  import SpectreLibrary from '$lib/design/SpectreLibrary.svelte';
  import type { SpectreRow } from '$lib/design/spectres';

  const engine = currentEngine();
  const calc = engine.calcs;
  const { tab } = calc;
  const { calculated, busy, error: failed } = engine;
  const collapseRevision = calc.collapse?.revision ?? readable(undefined);
  const main = engine.mainGroup();
  // CalcsTab.lua:452-455, CalcsTab.lua:40-119
  const calcs = $derived($tab?.sections);

  // CalcsTab.lua:12-17
  const MODES: Record<string, string> = { Unbuffed: 'UNBUFFED', Buffed: 'BUFFED', 'In Combat': 'COMBAT', 'Effective DPS': 'EFFECTIVE' };
  let inputs = $state<{ buffMode: string; showMinion: boolean; skillNumber: number }>({ buffMode: 'EFFECTIVE', showMinion: false, skillNumber: 1 });
  $effect(() => {
    void $calculated;
    void Promise.resolve(calc.inputs?.read())
      .then((i) => {
        if (i) inputs = i;
      })
      .catch(logError);
  });
  // CalcsTab.lua:135-137
  const lists = $derived.by(() => {
    if (!calc.inputs && !$tab?.lists) return undefined;
    const s = $tab?.lists ?? {};
    const rank = ['UNBUFFED', 'BUFFED', 'COMBAT', 'EFFECTIVE'].indexOf(inputs.buffMode);
    return [
      { label: 'Aura and Buff Skills', text: s.BuffList ?? '', from: 1, bd: 'SkillBuffs' },
      { label: 'Combat Buffs', text: s.CombatList ?? '', from: 2 },
      { label: 'Curses and Debuffs', text: s.CurseList ?? '', from: 3, bd: 'SkillDebuffs' }
    ].filter((l) => rank >= l.from);
  });

  // CalcsTab.lua:174-189, CalcSectionControl.lua:23-39
  let collapsed = $state<Record<string, boolean>>();
  $effect(() => {
    void $collapseRevision;
    void Promise.resolve(calc.collapse?.read())
      .then((saved) => {
        if (saved) collapsed = saved;
      })
      .catch(logError);
  });

  const modeLabel = $derived(Object.entries(MODES).find(([, v]) => v === inputs.buffMode)?.[0] ?? 'Effective DPS');
  // without its own skill number the tab shows the sidebar's main group
  const skillNumber = $derived(calc.inputs ? inputs.skillNumber : main.index + 1);

  // CalcsTab.lua:106-110, Build.lua:1453-1482
  let spectresOpen = $state(false);
  let library = $state.raw<SpectreRow[]>([]);
  let spectres = $state.raw<SpectreRow[]>([]);
  const openSpectres = () => {
    engine.spectres
      ?.read()
      ?.then((l) => {
        library = l.library;
        spectres = l.build;
        spectresOpen = true;
      })
      .catch(logError);
  };

  onMount(() => calc.attach?.());
</script>

{#snippet nodeview(id: number)}<MiniTreeView nodeId={id} />{/snippet}
{#snippet page()}
  {#if calcs}
    <CalcsPage
      {calcs}
      socketGroups={main.groups.length ? main.groups : ['<No skills added yet>']}
      mainSocketGroup={Math.max(0, Math.min(skillNumber, main.groups.length) - 1)}
      controls={$tab?.controls}
      mode={calc.inputs && modeLabel}
      {lists}
      {collapsed}
      oncollapse={(id: string, value: boolean) => calc.collapse?.set(id, value)}
      showMinion={calc.inputs && $tab?.haveMinion ? inputs.showMinion : undefined}
      onshowminion={calc.inputs &&
        ((show: boolean) => {
          inputs.showMinion = show;
          calc.inputs?.showMinion(show);
        })}
      onmode={calc.inputs &&
        ((label: string) => {
          const buffMode = MODES[label];
          if (!buffMode || buffMode === inputs.buffMode) return;
          inputs.buffMode = buffMode;
          calc.inputs?.buffMode(buffMode);
        })}
      onmaingroup={(i: number) => {
        if (!calc.inputs) return main.select(i);
        inputs.skillNumber = i + 1;
        calc.inputs.skillNumber(i + 1);
      }}
      oncontrol={(field: MainSkillField, v: number | undefined) => calc.setMainSkill(field, v)}
      onspectres={engine.spectres && openSpectres}
      grouptip={calc.grouptip && (async (i: number) => calc.grouptip?.(i))}
      modrows={calc.modrows && (async (sd: unknown) => calc.modrows?.(sd))}
      listrows={calc.listrows && (async (bd: string) => calc.listrows?.(bd))}
      moddebug={$devMode && calc.moddebug ? async () => (await calc.moddebug?.()) ?? '' : undefined}
      itemtip={calc.itemtip && (async (name: string) => calc.itemtip?.(name))}
      nodeview={engine.game === 'poe1' ? nodeview : undefined} />
  {:else}
    <p class="note p-5" role="status">Calculating…</p>
  {/if}
{/snippet}

{#if $failed}<p class="m-0 px-4 py-2 text-14" role="alert">{$failed}</p>{/if}
<!-- an engine whose edits the page waits on dims it meanwhile -->
{#if busy}
  <div class="flex min-h-0 min-w-0 flex-1 flex-col" class:opacity-60={$busy} aria-label="Calcs">{@render page()}</div>
{:else}
  {@render page()}
{/if}
{#if engine.spectres}
  <SpectreLibrary bind:open={spectresOpen} {library} {spectres} onsave={(ids: string[]) => engine.spectres?.save(ids)?.catch(logError)} />
{/if}
