<script lang="ts">
  import Button from '$lib/design/Button.svelte';
  import { poe2Build, type Poe2View } from '$lib/global.js';

  let { view }: { view: Poe2View } = $props();

  const STATS = [
    { key: 'Life', label: 'Life' },
    { key: 'EnergyShield', label: 'Energy Shield' },
    { key: 'Mana', label: 'Mana' },
    { key: 'TotalDPS', label: 'Total DPS' }
  ];
  const shown = (v: number | undefined) => (v === undefined ? '0' : String(Math.round(v * 100) / 100));
</script>

<section class="page gap-3 p-4" aria-label="Path of Exile 2 build">
  <div class="flex items-center gap-3">
    <span class="text-18 font-semibold">Path of Exile 2 build (preview)</span>
    <Button onclick={() => poe2Build.set(undefined)}>Close</Button>
  </div>
  <p class="m-0 text-14">
    Level {view.level}
    {view.ascendClassName && view.ascendClassName !== 'None' ? view.ascendClassName : view.className}
  </p>
  <dl class="grid grid-cols-[max-content_max-content] gap-x-4 gap-y-1 text-14">
    {#each STATS as s (s.key)}
      <dt class="text-text-2">{s.label}</dt>
      <dd class="m-0 tabular-nums" data-stat={s.key}>{shown(view.output[s.key])}</dd>
    {/each}
  </dl>
</section>
