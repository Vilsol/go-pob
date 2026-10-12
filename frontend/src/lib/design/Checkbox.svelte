<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLInputAttributes } from 'svelte/elements';

  type Props = Omit<HTMLInputAttributes, 'type'> & { checked?: boolean; compact?: boolean; children: Snippet };
  let { checked = $bindable(false), compact = false, children, class: extra = '', ...rest }: Props = $props();
</script>

<label class={[compact ? 'check-label' : 'flex items-center gap-2.5 text-14', extra]}>
  <input class="check" type="checkbox" bind:checked {...rest} />
  {@render children()}
</label>
