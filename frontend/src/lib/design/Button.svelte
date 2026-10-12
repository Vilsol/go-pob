<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  type Props = HTMLButtonAttributes & {
    size?: '2xs' | 'xs' | 'sm' | 'default' | 'md' | 'lg';
    variant?: 'default' | 'primary' | 'ghost' | 'danger' | 'danger-solid' | 'dashed';
    icon?: boolean;
    children: Snippet;
  };

  let { size = 'default', variant = 'default', icon = false, children, class: extra = '', ...rest }: Props = $props();

  const sizes = { '2xs': 'btn-2xs', xs: 'btn-xs', sm: 'btn-sm', default: '', md: 'btn-md', lg: 'btn-lg' };
  const variants = {
    default: '',
    primary: 'btn-primary',
    ghost: 'btn-ghost',
    danger: 'btn-danger',
    'danger-solid': 'border-warn-line bg-warn-bg font-semibold text-worse hover:enabled:border-worse',
    dashed: 'btn-dashed'
  };
</script>

<button class={['btn', sizes[size], variants[variant], icon && 'btn-icon', extra]} {...rest}>
  {@render children()}
</button>
