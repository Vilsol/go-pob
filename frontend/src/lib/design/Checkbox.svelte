<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLInputAttributes } from 'svelte/elements';

  type Props = Omit<HTMLInputAttributes, 'type'> & { checked?: boolean; compact?: boolean; children?: Snippet };
  let { checked = $bindable(false), compact = false, children, class: extra = '', ...rest }: Props = $props();
</script>

<!-- without children it is the bare box, for a label or cell the caller lays out; class then styles the box -->
{#if children}
  <label class={[compact ? 'check-label' : 'flex items-center gap-2.5 text-14', extra]}>
    <input class="check" type="checkbox" bind:checked {...rest} />
    {@render children()}
  </label>
{:else}
  <input class={['check', extra]} type="checkbox" bind:checked {...rest} />
{/if}
