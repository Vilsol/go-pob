<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { showWarnings } from '$lib/display/item_options';
  import Sidebar from '$lib/design/Sidebar.svelte';
  import SpectreLibrary from '$lib/design/SpectreLibrary.svelte';
  import type { SpectreRow } from '$lib/design/spectres';
  import { currentEngine } from '$lib/go/engine';
  import { logError } from '$lib/utils';
  import type { MainSkillField } from './main_skill';
  import { statPins } from './stat_pins.svelte';

  let { fill = false }: { fill?: boolean } = $props();

  const engine = currentEngine();
  const sidebar = engine.sidebar;
  const controls = engine.mainSkill;
  const main = engine.mainGroup();

  // Build.lua:1453-1482
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
</script>

{#if $sidebar}
  <Sidebar
    {fill}
    sidebar={$showWarnings ? $sidebar : { ...$sidebar, warnings: [] }}
    socketGroups={main.groups.length ? main.groups : ['<No skills added yet>']}
    mainSocketGroup={Math.max(0, main.index)}
    bind:pinned={statPins.ids}
    controls={$controls}
    onmaingroup={(i: number) => main.select(i)}
    oncontrol={(field: MainSkillField, v: number | undefined) => engine.setMainSkill(field, v)}
    onspectres={engine.spectres ? openSpectres : undefined}
    steps={engine.breakdownSteps}
    onbreakdown={() => void goto(`${base}/calcs`)} />
{/if}
{#if engine.spectres}
  <SpectreLibrary bind:open={spectresOpen} {library} {spectres} onsave={(ids: string[]) => engine.spectres?.save(ids)?.catch(logError)} />
{/if}
