<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAnchorAttributes, HTMLButtonAttributes } from 'svelte/elements';

  type Props = HTMLButtonAttributes &
    Pick<HTMLAnchorAttributes, 'href' | 'target' | 'rel' | 'download'> & {
      size?: '2xs' | 'xs' | 'sm' | 'default' | 'md' | 'lg';
      variant?: 'default' | 'primary' | 'ghost' | 'danger' | 'danger-solid' | 'dashed' | 'close' | 'bare';
      icon?: boolean;
      ref?: HTMLElement;
      children: Snippet;
    };

  let { size = 'default', variant = 'default', icon = false, href, ref = $bindable(), children, class: extra = '', ...rest }: Props = $props();

  const sizes = { '2xs': 'btn-2xs', xs: 'btn-xs', sm: 'btn-sm', default: '', md: 'btn-md', lg: 'btn-lg' };
  const variants = {
    default: '',
    primary: 'btn-primary',
    ghost: 'btn-ghost',
    danger: 'btn-danger',
    'danger-solid': 'border-warn-line bg-warn-bg font-semibold text-worse hover:enabled:border-worse',
    dashed: 'btn-dashed',
    close: 'btn-close',
    bare: 'btn-bare'
  };
  // close and bare stand alone: .btn's height, padding and hover would show through them
  const base = $derived(variant === 'close' || variant === 'bare' ? '' : 'btn');
</script>

{#if href != null}
  <a class={[base, sizes[size], variants[variant], icon && 'btn-icon', extra]} {href} bind:this={ref} {...rest as HTMLAnchorAttributes}>{@render children()}</a>
{:else}
  <button class={[base, sizes[size], variants[variant], icon && 'btn-icon', extra]} bind:this={ref} {...rest}>
    {@render children()}
  </button>
{/if}
