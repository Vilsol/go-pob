<script lang="ts">
  import type { HTMLTextareaAttributes } from 'svelte/elements';

  type Props = Omit<HTMLTextareaAttributes, 'value'> & { value?: string | null; ref?: HTMLTextAreaElement };
  let { value = $bindable(), ref = $bindable(), class: extra = '', ...rest }: Props = $props();
</script>

<textarea class={['input', extra]} bind:value bind:this={ref} {...rest}></textarea>
