<script lang="ts" module>
  /* ItemsTab.lua:605-625, 776-854 (PoB-PoE2): the socket count, and each socket's rune from the runes that fit */
  export type RuneSocketsInfo = { canHaveSockets: boolean; sockets: string[]; choices: { name: string; lines: string[] }[] };
  export type RuneSocketsEdit = { op: 'sockets'; count: number } | { op: 'rune'; socket: number; rune: string };
</script>

<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import NumberInput from './NumberInput.svelte';
  import Select from './Select.svelte';

  let { info, onedit }: { info: RuneSocketsInfo; onedit: (edit: RuneSocketsEdit) => void } = $props();

  const options = $derived(info.choices.map((c) => ({ value: c.name, label: c.name, title: c.lines.join('\n') })));
</script>

{#if info.canHaveSockets}
  <section aria-label="Rune sockets" class="flex flex-col gap-2 border-t border-line pt-2.5">
    <NumberInput
      label="Sockets"
      size="sm"
      class="w-30"
      min={0}
      max={6}
      value={info.sockets.length}
      onchange={(count: number) => onedit({ op: 'sockets', count })} />
    {#each info.sockets as rune, i (i)}
      <FloatingLabel label="Rune #{i + 1}" class="[--float-bg:var(--color-surface-1)]"
        ><Select
          class="bg-surface-1"
          aria-label="Rune #{i + 1}"
          {options}
          value={rune}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onedit({ op: 'rune', socket: i + 1, rune: e.currentTarget.value })} /></FloatingLabel>
    {/each}
  </section>
{/if}
