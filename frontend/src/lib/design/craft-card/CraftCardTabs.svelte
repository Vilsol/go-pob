<script lang="ts">
  import type { Snippet } from 'svelte';
  import Tabs from '../Tabs.svelte';

  type Props = { base: Snippet; implicits: Snippet; affixes: Snippet; result?: Snippet; implicitCount: number; explicitCount: number };
  let { base, implicits, affixes, result, implicitCount, explicitCount }: Props = $props();
  let selected = $state('affixes');
  const tabs = $derived([
    { id: 'base', label: 'Base' },
    { id: 'implicits', label: 'Implicits', badge: { text: String(implicitCount), tone: 'dim' as const } },
    { id: 'affixes', label: 'Affixes', badge: { text: String(explicitCount), tone: 'dim' as const } }
  ]);
  const uid = $props.id();
</script>

<div aria-label="Crafting card" role="group" class="flex flex-col overflow-hidden rounded-md border border-line bg-surface-1">
  <Tabs {tabs} bind:selected label="Crafting sections" class="border-b border-line px-2" panelClass="p-3 min-h-60">
    {#snippet panel(id: string)}
      {#each [{ id: 'base', body: base }, { id: 'implicits', body: implicits }, { id: 'affixes', body: affixes }] as p (p.id)}
        <div class="flex flex-col gap-2.5" hidden={id !== p.id}>{@render p.body()}</div>
      {/each}
    {/snippet}
  </Tabs>
  {#if result}
    <section aria-labelledby="{uid}-result" class="flex flex-col gap-2.5 border-t-2 border-line-strong bg-surface-2 p-3">
      <h3 id="{uid}-result" class="caps">Result</h3>
      {@render result()}
    </section>
  {/if}
</div>
