<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  type Props = HTMLButtonAttributes & { pressed: boolean; children: Snippet };
  let { pressed, children, class: extra = 'flex-col items-start gap-0.5 px-2.5 py-2', ...rest }: Props = $props();
</script>

<button class={['choice-card', extra]} aria-pressed={pressed} {...rest}>{@render children()}</button>
