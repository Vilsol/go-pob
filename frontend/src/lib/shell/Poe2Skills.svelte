<script lang="ts">
  import Button from '$lib/design/Button.svelte';
  import PobText from '$lib/design/PobText.svelte';
  import type { Poe2SkillGroup } from '$lib/go/sync_worker2';

  let {
    groups,
    mainGroup,
    busy = false,
    onmain
  }: {
    groups: Poe2SkillGroup[] | undefined;
    mainGroup: number;
    busy?: boolean;
    onmain: (index: number) => void;
  } = $props();

  let selected = $state(0);
  const group = $derived(groups?.[Math.min(selected, (groups?.length ?? 1) - 1)]);

  // SkillListControl.lua:78-96 (PoB-PoE2); Global.lua: RELIC, CUSTOM
  const listLabel = (g: Poe2SkillGroup, index: number) => {
    let label = g.label || '?';
    const current = mainGroup === index + 1;
    if (!g.enabled) label = `${current ? '' : '^x7F7F7F'}${label} (Disabled)`;
    if (current) label += `^x60C060${g.enabled ? ' (Active)' : ' (Forced Active)'}`;
    if (g.includeInFullDPS) label += '^x5CF0BB (FullDPS)';
    return label;
  };
</script>

{#if groups}
  <div class="flex min-h-0 min-w-0 flex-1 flex-wrap gap-4 overflow-y-auto" class:opacity-60={busy} aria-label="Skills" data-poe2-page>
    <div class="flex w-full max-w-[320px] flex-col gap-1" role="listbox" aria-label="Socket groups">
      {#if groups.length === 0}<span class="note text-14">No skills added yet</span>{/if}
      {#each groups as g, i (i)}
        <button
          type="button"
          role="option"
          aria-selected={i === selected}
          class={['border border-line px-2 py-1 text-left text-14', i === selected ? 'bg-surface-3' : 'bg-surface-1']}
          onclick={() => (selected = i)}><PobText text={listLabel(g, i)} /></button>
      {/each}
    </div>
    {#if group}
      <section class="flex min-w-0 flex-1 basis-[280px] flex-col gap-2" aria-label="Socket group">
        <div class="flex flex-wrap items-center gap-3 text-14">
          <span class="font-semibold"><PobText text={group.label || '?'} /></span>
          {#if group.slot}<span>Socketed in: {group.slot}</span>{/if}
          {#if group.source}<span class="note">Source: {group.source}</span>{/if}
          <span>{group.enabled ? 'Enabled' : 'Disabled'}</span>
          {#if group.includeInFullDPS}<span>Included in Full DPS</span>{/if}
          <Button disabled={busy || mainGroup === selected + 1} onclick={() => onmain(selected + 1)}>Set as main skill</Button>
        </div>
        <div class="max-w-full overflow-x-auto">
          <table class="text-14">
            <thead>
              <tr class="text-left">
                <th class="pr-3 font-normal">Gem</th>
                <th class="pr-3 font-normal">Level</th>
                <th class="pr-3 font-normal">Quality</th>
                <th class="pr-3 font-normal">Count</th>
                <th class="font-normal">Enabled</th>
              </tr>
            </thead>
            <tbody>
              {#each group.gems as gem, gi (gi)}
                <tr class:opacity-60={!gem.enabled} data-gem={gem.name}>
                  <td class="pr-3"><PobText text={gem.color + gem.name} /></td>
                  <td class="pr-3 tabular-nums">{gem.level}</td>
                  <td class="pr-3 tabular-nums">{gem.quality}</td>
                  <td class="pr-3 tabular-nums">{gem.count}</td>
                  <td>{gem.enabled ? 'Yes' : 'No'}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
    {/if}
  </div>
{:else}
  <p class="note p-5" role="status">Loading skills…</p>
{/if}
