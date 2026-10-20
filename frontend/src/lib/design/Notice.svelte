<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue } from 'svelte/elements';

  type Props = {
    kind?: 'neutral' | 'info' | 'warn' | 'error';
    title?: string;
    size?: 'sm' | 'md';
    float?: boolean;
    row?: boolean;
    role?: 'status' | 'alert';
    actions?: Snippet;
    class?: ClassValue;
    children: Snippet;
  };
  let { kind = 'neutral', title, size = 'md', float = false, row = false, role, actions, class: extra = '', children }: Props = $props();
  const kinds = {
    neutral: 'border-line bg-surface-1',
    info: 'border-info-line bg-info-bg',
    warn: 'border-warn-line bg-warn-bg-2',
    error: 'border-danger-line bg-danger-bg'
  };
  const titles = { neutral: 'text-text-2', info: 'text-better', warn: 'text-worse', error: 'text-error' };
  const sizes = { sm: 'px-2.5 py-1.5 text-12 leading-4', md: 'px-3 py-2 text-13' };
  const grid = $derived(!!(title && actions));
</script>

<div
  role={role ?? (kind === 'error' ? 'alert' : 'status')}
  class={[
    'rounded border',
    (row || title || actions) && 'flex items-center gap-2.5',
    float ? 'rounded-md border-line bg-surface-2 px-3 py-1 text-13 text-text-2 shadow-pop' : [kinds[kind], sizes[size]],
    grid && 'max-phone:grid max-phone:grid-cols-[minmax(0,1fr)_auto] max-phone:gap-y-1',
    extra
  ]}>
  {#if title}<span class={['eyebrow whitespace-nowrap', titles[kind]]}>{title}</span>{/if}{#if actions}<span
      class={[grid && 'max-phone:col-span-2 max-phone:row-start-2']}>{@render children()}</span
    >
    <div class="grow max-phone:hidden"></div>
    <div class={['flex items-center gap-2.5', grid && 'max-phone:col-start-2 max-phone:row-start-1']}>
      {@render actions()}
    </div>{:else}{@render children()}{/if}
</div>
