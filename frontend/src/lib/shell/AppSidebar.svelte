<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { outputs } from '$lib/global.js';
  import { showWarnings } from '$lib/display/item_options';
  import { sidebar } from '$lib/build_warnings';
  import Sidebar from '$lib/design/Sidebar.svelte';
  import { MainGroup } from './main_group.svelte';
  import { type MainSkillField, writeMainSkill } from './main_skill';
  import SpectreLibrary from '$lib/design/SpectreLibrary.svelte';
  import type { SpectreRow } from '$lib/design/spectres';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import type { calculator } from '$lib/types';

  const main = new MainGroup();

  // Build.lua:1453-1482
  let spectresOpen = $state(false);
  let library = $state.raw<SpectreRow[]>([]);
  let spectres = $state.raw<SpectreRow[]>([]);
  const row = (r: calculator.SpectreRow): SpectreRow => ({ id: r.ID, name: r.Name, skills: [...(r.Skills ?? [])], tooltip: [...(r.Tooltip ?? [])] });
  const openSpectres = () => {
    syncWrap
      ?.GetSpectreLibrary()
      .then((l) => {
        library = l.library.map(row);
        spectres = l.build.map(row);
        spectresOpen = true;
      })
      .catch(logError);
  };
</script>

{#if $sidebar}
  <Sidebar
    sidebar={$showWarnings ? $sidebar : { ...$sidebar, warnings: [] }}
    socketGroups={main.groups.length ? main.groups : ['<No skills added yet>']}
    mainSocketGroup={Math.max(0, main.index)}
    controls={$outputs?.MainSkill}
    onmaingroup={(i: number) => main.select(i)}
    oncontrol={(field: MainSkillField, v: number | undefined) => writeMainSkill($outputs?.MainSkill, field, v)}
    onspectres={openSpectres}
    onbreakdown={() => void goto(`${base}/calcs`)} />
{/if}
<SpectreLibrary bind:open={spectresOpen} {library} {spectres} onsave={(ids: string[]) => syncWrap?.SetSpectres(ids).catch(logError)} />
