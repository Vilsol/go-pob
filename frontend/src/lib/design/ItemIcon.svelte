<script lang="ts">
  import type { ClassValue } from 'svelte/elements';
  import { tone, type Tone } from './tone';
  import { artPath, artUrl } from './item_art';

  type Props = {
    name: string;
    base?: string;
    rarity?: string;
    gem?: boolean;
    /* box edge in px; the box never changes size, whatever loads */
    size?: number;
    /* the name is already next to the icon: no alt text, hidden from assistive tech */
    decorative?: boolean;
    /* an image URL that skips the art lookup */
    src?: string;
    class?: ClassValue;
  };
  let { name, base = '', rarity = '', gem = false, size = 32, decorative = false, src, class: extra = '' }: Props = $props();

  const unique = $derived(rarity === 'unique' || rarity === 'relic');
  let path = $state<string | undefined>();
  let loaded = $state(false);
  let failed = $state(false);

  $effect(() => {
    const query = { Name: name, Base: base, Unique: unique, Gem: gem };
    loaded = false;
    failed = false;
    path = undefined;
    if (src !== undefined) return;
    let live = true;
    void artPath(query).then((p) => {
      if (live) path = p;
    });
    return () => {
      live = false;
    };
  });

  const url = $derived(src ?? (path ? artUrl(path) : undefined));
  const initials = $derived(
    (unique || gem ? name : base || name)
      .split(/[\s-]+/)
      .filter((w) => /^[A-Za-z]/.test(w))
      .slice(0, 2)
      .map((w) => w[0].toUpperCase())
      .join('')
  );
  const showImg = $derived(!!url && !failed);
</script>

<span
  class={['relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-tight', !loaded && 'bg-surface-2', extra]}
  style:width="{size}px"
  style:height="{size}px"
  aria-hidden={decorative ? 'true' : undefined}
  data-item-icon={loaded ? 'loaded' : showImg ? 'loading' : 'fallback'}>
  {#if !loaded}
    <span
      data-initials={initials}
      class={[
        'font-mono leading-none font-semibold text-(--tone,var(--color-text-3)) select-none before:content-[attr(data-initials)]',
        tone((gem ? 'gem' : rarity || 'normal') as Tone)
      ]}
      style:font-size="{Math.max(8, Math.round(size * 0.34))}px"
      role={showImg || decorative ? undefined : 'img'}
      aria-label={showImg || decorative ? undefined : name}
      aria-hidden={showImg || decorative ? 'true' : undefined}></span>
  {/if}
  {#if showImg}
    <img
      src={url}
      alt={decorative ? '' : name}
      loading="lazy"
      decoding="async"
      draggable="false"
      class={['absolute inset-0 size-full object-contain', !loaded && 'opacity-0']}
      onload={() => (loaded = true)}
      onerror={() => (failed = true)} />
  {/if}
</span>
