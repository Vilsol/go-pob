<script lang="ts">
  import PobText from '$lib/design/PobText.svelte';
  import type { Poe2Items } from '$lib/go/sync_worker2';

  let { items, busy = false }: { items: Poe2Items | undefined; busy?: boolean } = $props();

  // ItemsTab.lua:240: a swap slot is labelled with its weapon slot's name
  const slotLabel = (slot: string) => slot.replace(/ Swap$/, '');
</script>

{#if items}
  <div class="flex min-h-0 min-w-0 flex-1 flex-col gap-3 overflow-y-auto" class:opacity-60={busy} aria-label="Items" data-poe2-page>
    <span class="text-14" aria-label="Weapon set">Weapon Set {items.useSecondWeaponSet ? 2 : 1}</span>
    {#if items.items.length === 0}<span class="note text-14">No items equipped</span>{/if}
    <div class="grid grid-cols-[repeat(auto-fill,minmax(min(100%,280px),1fr))] gap-3">
      {#each items.items as item (item.slot)}
        <section class="flex flex-col gap-1 border border-line bg-surface-1 p-2 text-14" aria-label={slotLabel(item.slot)} data-slot={item.slot}>
          <span class="text-12 text-text-3">{slotLabel(item.slot)}</span>
          <div class="flex flex-col items-center text-center font-semibold">
            {#each item.name as line, i (i)}<PobText text={line} />{/each}
          </div>
          {#each item.lines as group, gi (gi)}
            <div class="flex flex-col items-center border-t border-line-faint pt-1 text-center" data-lines>
              {#each group as line, li (li)}<span><PobText text={line} /></span>{/each}
            </div>
          {/each}
        </section>
      {/each}
    </div>
  </div>
{:else}
  <p class="note p-5" role="status">Loading items…</p>
{/if}
