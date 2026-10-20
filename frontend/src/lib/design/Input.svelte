<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  type Props = Omit<HTMLInputAttributes, 'size'> & { size?: 'xs' | 'sm' | 'default' | 'md' | 'lg'; value?: string | number | null };
  let { size = 'default', value = $bindable(), class: extra = '', ...rest }: Props = $props();

  const sizes = { xs: 'input-xs', sm: 'input-sm', default: '', md: 'input-md', lg: 'input-lg' };
</script>

<input class={['input', sizes[size], extra]} bind:value {...rest} />
