<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLSelectAttributes } from 'svelte/elements';

  type Option = string | { value: string; label: string; title?: string };
  type Props = Omit<HTMLSelectAttributes, 'size' | 'value'> & {
    options?: Option[];
    size?: 'xs' | 'sm' | 'default' | 'md' | 'lg';
    value?: string | number | null;
    ref?: HTMLSelectElement;
    children?: Snippet;
  };
  let { options = [], size = 'default', value = $bindable(), ref = $bindable(), class: extra = '', onchange, children, ...rest }: Props = $props();

  const sizes = { xs: 'input-xs', sm: 'input-sm', default: '', md: 'input-md', lg: 'input-lg' };
  const norm = $derived(options.map((o): { value: string; label: string; title?: string } => (typeof o === 'string' ? { value: o, label: o } : o)));

  // a handler that puts the previous option back cancels the change, as on a plain <select>
  function change(e: Event & { currentTarget: EventTarget & HTMLSelectElement }) {
    onchange?.(e);
    const shown = e.currentTarget.value;
    if (String(value ?? '') !== shown) value = typeof value === 'number' ? Number(shown) : shown;
  }
</script>

<select class={['select', sizes[size], extra]} bind:value bind:this={ref} onchange={change} {...rest}>
  {#if children}{@render children()}{:else}
    <!-- Build.lua:849 -->
    {#each norm as o, i (i)}<option value={o.value} title={o.title}>{o.label}</option>{/each}
  {/if}
</select>
