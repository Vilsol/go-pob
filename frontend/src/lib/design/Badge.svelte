<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue, HTMLAttributes } from 'svelte/elements';
  import { tone as toneClass, type Tone } from './tone';

  type Props = Omit<HTMLAttributes<HTMLSpanElement>, 'class'> & {
    tone?: Tone;
    variant?: 'solid' | 'soft' | 'outline';
    shape?: 'pill' | 'tag';
    size?: 'xs' | 'sm';
    mono?: boolean;
    class?: ClassValue;
    children: Snippet;
  };
  let { tone, variant = 'soft', shape = 'pill', size = 'xs', mono = false, class: extra = '', children, ...rest }: Props = $props();
  const variants = {
    solid: 'bg-(--tone,var(--color-accent)) text-accent-ink',
    soft: 'bg-(--tone,var(--color-text-3))/14 text-(--tone,var(--color-text-3))',
    outline: 'border border-line text-(--tone,var(--color-text-2))'
  };
  const sizes = { xs: 'h-4 px-1.5 text-10', sm: 'h-4.5 min-w-4.5 px-1.25 text-11' };
</script>

<span
  class={[
    'box-border inline-flex shrink-0 items-center justify-center whitespace-nowrap',
    shape === 'pill' ? 'rounded-pill' : 'rounded-tight',
    mono ? 'font-mono' : 'font-sans',
    variant !== 'outline' && 'font-semibold',
    variants[variant],
    sizes[size],
    tone && toneClass(tone),
    extra
  ]}
  {...rest}>{@render children()}</span>
