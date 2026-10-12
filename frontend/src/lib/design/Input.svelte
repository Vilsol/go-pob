<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  type Props = Omit<HTMLInputAttributes, 'size'> & { size?: 'xs' | 'default' | 'lg'; value?: string | number | null };
  let { size = 'default', value = $bindable(), class: extra = '', ...rest }: Props = $props();

  const sizes = { xs: 'input-xs', default: '', lg: 'input-lg' };
</script>

<input class={['input', sizes[size], extra]} bind:value {...rest} />
