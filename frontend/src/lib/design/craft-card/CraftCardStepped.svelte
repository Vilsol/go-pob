<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { base: Snippet; implicits: Snippet; affixes: Snippet; result?: Snippet };
  let { base, implicits, affixes, result }: Props = $props();
  const steps = $derived(
    [
      { id: 'base', title: 'Base', caption: 'What the item is, and what belongs to its base', body: base },
      { id: 'implicits', title: 'Implicits', caption: 'Enchants and implicit modifiers', body: implicits },
      { id: 'affixes', title: 'Affixes', caption: 'Prefixes and suffixes, and the lines they make', body: affixes },
      { id: 'result', title: 'Result', caption: 'What equipping the item gives', body: result }
    ].filter((s) => s.body)
  );
  const uid = $props.id();
</script>

<div aria-label="Crafting card" role="group" class="@container">
  <div class="grid grid-cols-1 items-start gap-3 @[760px]:grid-cols-2">
    {#each steps as s, i (s.id)}
      <section
        aria-labelledby="{uid}-{s.id}"
        class={[
          'flex min-w-0 flex-col gap-2.5 rounded-md border bg-surface-1 p-3',
          s.id === 'result' ? 'border-l-3 border-line border-l-accent' : 'border-line'
        ]}>
        <header class="flex items-baseline gap-2 border-b border-line-faint pb-2">
          <span class="flex size-5 shrink-0 items-center justify-center rounded-full bg-surface-3 font-mono text-11 text-text-2" aria-hidden="true"
            >{i + 1}</span>
          <h3 id="{uid}-{s.id}" class="caps">{s.title}</h3>
          <span class="note truncate">{s.caption}</span>
        </header>
        {@render s.body?.()}
      </section>
    {/each}
  </div>
</div>
