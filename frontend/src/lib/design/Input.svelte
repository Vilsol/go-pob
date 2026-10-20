<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  type Props = Omit<HTMLInputAttributes, 'size'> & { size?: 'xs' | 'sm' | 'default' | 'md' | 'lg'; value?: string | number | null; ref?: HTMLInputElement };
  let { size = 'default', value = $bindable(), ref = $bindable(), class: extra = '', ...rest }: Props = $props();

  const sizes = { xs: 'input-xs', sm: 'input-sm', default: '', md: 'input-md', lg: 'input-lg' };
</script>

<input class={['input', sizes[size], extra]} bind:value bind:this={ref} {...rest} />
