<script lang="ts">
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { phoneMedia } from '$lib/design/mobile/mobile';
  import SkillsPanes from '$lib/design/mobile/SkillsPanes.svelte';
  import SkillsPage from '$lib/design/SkillsPage.svelte';
  import { currentEngine, type SkillsPageModel } from '$lib/go/engine';

  let { make }: { make: () => SkillsPageModel } = $props();

  const model = make();
  const phone = phoneMedia();
  const failed = currentEngine().error;
</script>

{#if $failed}<p class="m-0 text-14" role="alert">{$failed}</p>{/if}

{#if model.ready}
  <SkillsPanes on={phone.current}>
    <SkillsPage
      sets={model.sets}
      set={model.set}
      selected={model.selected}
      mainGroup={model.mainGroup}
      catalogue={model.catalogue}
      deltas={model.deltas}
      slots={model.slots}
      bind:sortOn={model.sortOn}
      bind:sortStat={model.sortStat}
      onset={model.onset}
      onsets={model.onsets}
      onreordergroups={model.onreordergroups}
      oncopygroup={model.oncopygroup}
      grouptip={model.grouptip}
      groupicon={model.groupicon}
      onpastegroup={model.onpastegroup}
      onselect={model.onselect}
      ongroup={model.ongroup}
      ongem={model.ongem}
      onaddgem={model.onaddgem}
      onremovegem={model.onremovegem}
      onnewgroup={model.onnewgroup}
      onreordergems={model.onreordergems}
      setLocked={model.setLocked}
      groupsLocked={model.groupsLocked}
      onmain={model.onmain}
      ondeletegroup={model.ondeletegroup}
      ondeleteall={model.ondeleteall}
      onsort={model.onsort}
      options={model.options}
      onoption={model.onoption}
      optionLists={model.optionLists}
      editor={model.editor}
      revision={model.revision} />
  </SkillsPanes>
{:else}
  <EmptyState busy text="Loading the skills…" />
{/if}
