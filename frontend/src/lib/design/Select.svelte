<script lang="ts">
  import type { HTMLSelectAttributes } from 'svelte/elements';

  type Option = string | { value: string; label: string; title?: string };
  type Props = Omit<HTMLSelectAttributes, 'size'> & { options: Option[]; size?: 'xs' | 'default' | 'lg'; value?: string };
  let { options, size = 'default', value = $bindable(), class: extra = '', ...rest }: Props = $props();

  const sizes = { xs: 'input-xs', default: '', lg: 'input-lg' };
  const norm = $derived(options.map((o): { value: string; label: string; title?: string } => (typeof o === 'string' ? { value: o, label: o } : o)));
</script>

<select class={['select', sizes[size], extra]} bind:value {...rest}>
  <!-- Build.lua:849 -->
  {#each norm as o, i (i)}<option value={o.value} title={o.title}>{o.label}</option>{/each}
</select>
