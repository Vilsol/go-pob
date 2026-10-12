<script lang="ts">
  type Option = { id: string; label: string; count?: string };
  type Props = { options: Option[]; value?: string; label: string; tabs?: boolean; chip?: boolean; onchange?: (id: string) => void };
  let { options, value = $bindable(options[0]?.id), label, tabs = false, chip = false, onchange }: Props = $props();

  const btns = $state<HTMLButtonElement[]>([]);
  const pick = (id: string) => {
    value = id;
    onchange?.(id);
  };

  function onkeydown(e: KeyboardEvent, i: number) {
    if (!tabs) return;
    const n = options.length;
    const to = { ArrowRight: (i + 1) % n, ArrowLeft: (i - 1 + n) % n, Home: 0, End: n - 1 }[e.key];
    if (to === undefined) return;
    e.preventDefault();
    pick(options[to].id);
    btns[to]?.focus();
  }
</script>

<div role={tabs ? 'tablist' : 'group'} aria-label={label} class={chip ? 'flex gap-1' : 'seg'}>
  {#each options as o, i (o.id)}
    <button
      bind:this={btns[i]}
      role={tabs ? 'tab' : undefined}
      class={chip ? 'chip' : 'seg-btn'}
      aria-selected={tabs ? o.id === value : undefined}
      aria-pressed={tabs ? undefined : o.id === value}
      tabindex={tabs && o.id !== value ? -1 : undefined}
      onclick={() => pick(o.id)}
      onkeydown={(e) => onkeydown(e, i)}
      >{o.label}{#if o.count}<span class="ml-1 font-mono text-11 text-text-2">{o.count}</span>{/if}</button>
  {/each}
</div>
