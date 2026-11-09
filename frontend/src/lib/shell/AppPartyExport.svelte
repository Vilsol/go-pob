<script lang="ts">
  import { resource } from 'runed';
  import { poe2Build } from '$lib/global.js';
  import { poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import Checkbox from '$lib/design/Checkbox.svelte';
  import Textarea from '$lib/design/Textarea.svelte';
  import SectionHeading from '$lib/design/SectionHeading.svelte';

  // ImportTab.lua:138-141 (PoB-PoE2): Export Support; PartyTab.lua:612-653: the parts Save writes into the build
  const exported = resource(
    () => $poe2Build,
    async () => (await poe2Worker()).GetPartyExport()
  );
  const on = $derived(exported.current?.on ?? false);
  const setOn = (next: boolean) => {
    if (next !== on) void poe2Run((w) => w.SetExportParty(next));
  };
</script>

{#if exported.current}
  <section aria-label="Party export" class="scroll flex w-80 shrink-0 flex-col gap-2.5 max-phone:w-full">
    <SectionHeading size="md">EXPORT</SectionHeading>
    <Checkbox
      compact
      class="text-13"
      title="This is for party play, to export support character, it enables the exporting of auras, curses and modifiers to the enemy"
      bind:checked={() => on, setOn}>Export Support</Checkbox>
    {#if on}
      <p class="note m-0">Party members import this build’s code or link from their own Party page; this is what they receive.</p>
      {#each exported.current.parts as p (p.name)}
        <span class="text-12 text-text-2">{p.name}</span>
        <Textarea class="h-24 shrink-0 font-mono text-11" aria-label="{p.name} export" readonly spellcheck="false" value={p.text} />
      {:else}
        <span class="note">This build exports nothing yet.</span>
      {/each}
    {/if}
  </section>
{/if}
