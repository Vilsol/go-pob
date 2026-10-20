<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  type Props = Omit<HTMLInputAttributes, 'size' | 'type' | 'value'> & {
    label: string;
    value?: string | null;
    size?: 'xs' | 'sm' | 'default' | 'md' | 'lg';
    ref?: HTMLInputElement;
  };
  let { label, value = $bindable(), size = 'default', ref = $bindable(), class: extra = '', ...rest }: Props = $props();
  const sizes = { xs: 'input-xs', sm: 'input-sm', default: '', md: 'input-md', lg: 'input-lg' };
</script>

<input type="search" class={['input', sizes[size], extra]} aria-label={label} bind:value bind:this={ref} {...rest} />
