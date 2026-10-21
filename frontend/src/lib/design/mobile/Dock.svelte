<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from '../Icon.svelte';

  // non-modal: the page stays usable, so this is not a Sheet (a modal dialog)
  type Props = {
    /** shown on a phone; when false the dock hides there and only lays out on wider screens */
    active: boolean;
    folded?: boolean;
    title: string;
    controls: string;
    children: Snippet;
  };
  let { active, folded = $bindable(false), title, controls, children }: Props = $props();
</script>

<!-- on wider screens the dock is layout-transparent and its children sit in the page's row -->
<div
  class={[
    'contents',
    active
      ? [
          'max-phone:fixed max-phone:inset-x-0 max-phone:z-dock max-phone:flex max-phone:flex-col max-phone:border-t max-phone:border-line-strong max-phone:bg-surface-1 max-phone:shadow-sheet max-phone:transition-[height,bottom]',
          folded ? 'max-phone:bottom-[calc(3.5rem+env(safe-area-inset-bottom))] max-phone:h-11' : 'max-phone:bottom-0 max-phone:h-[55dvh]'
        ]
      : 'max-phone:hidden'
  ]}
  data-breakdown-dock
  data-docked={active && folded ? '' : undefined}>
  {#if active}
    <button
      type="button"
      class="flex h-11 w-full shrink-0 items-center gap-2 border-0 border-b border-line bg-bar px-3 text-left font-sans text-13 text-text phone:hidden"
      aria-expanded={!folded}
      aria-controls={controls}
      onclick={() => (folded = !folded)}
      ><span class="min-w-0 flex-1 truncate">{title}</span><span aria-hidden="true" class="inline-flex text-text-3"
        ><Icon name={folded ? 'chevron-up' : 'chevron-down'} size={12} /></span
      ><span class="sr-only">{folded ? 'Show the breakdown' : 'Fold the breakdown down'}</span></button>
  {/if}{@render children()}
</div>
