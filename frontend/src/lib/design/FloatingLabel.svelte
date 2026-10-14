<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue } from 'svelte/elements';

  type Props = { label: string; class?: ClassValue; children: Snippet };
  let { label, class: extra, children }: Props = $props();
</script>

<label class={['relative flex flex-col pt-1.75 [.items-center>&]:-mt-[calc(var(--spacing)*0.875)] [.items-center>&]:mb-[calc(var(--spacing)*0.875)]', extra]}>
  <span data-floating-label class="absolute top-0 right-2.5 z-1 select-none bg-[color:var(--float-bg,var(--color-ground))] px-1 text-11 text-text-3"
    >{label}</span>
  {@render children()}
</label>
