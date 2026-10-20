<script lang="ts">
  import Dot from './Dot.svelte';
  import type { Snippet } from 'svelte';
  import type { HTMLLabelAttributes } from 'svelte/elements';

  type Props = HTMLLabelAttributes & { label: string | Snippet; modified?: boolean; inactive?: boolean; control: Snippet };
  let { label, modified = false, inactive = false, control, class: extra = '', ...rest }: Props = $props();
</script>

<label class={['option-row', inactive && 'text-text-3', extra]} {...rest}>
  <span class="flex items-center gap-1.5">
    <Dot class={modified ? 'size-1.5' : 'invisible size-1.5'} />
    {#if typeof label === 'string'}{label}{:else}<span>{@render label()}</span>{/if}
    {#if modified}<span class="sr-only">(changed)</span>{/if}
  </span>
  {@render control()}
</label>
