<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue } from 'svelte/elements';

  type Props = {
    level?: 2 | 3 | 4;
    size?: 'sm' | 'md' | 'lg';
    count?: string;
    actions?: Snippet;
    class?: ClassValue;
    children: Snippet;
  };
  let { level = 2, size = 'sm', count, actions, class: extra = '', children }: Props = $props();
  const sizes = { sm: '', md: 'text-12', lg: 'text-13' };
</script>

{#snippet heading(cls: ClassValue)}
  <svelte:element this={`h${level}`} class={['subhead', sizes[size], cls]}
    >{@render children()}
    {#if count}<span class="font-normal tracking-normal text-text-3 normal-case">{count}</span>{/if}</svelte:element>
{/snippet}

{#if actions}
  <div class={['flex items-baseline justify-between gap-2', extra]}>
    {@render heading('')}
    {@render actions()}
  </div>
{:else}
  {@render heading(extra)}
{/if}
