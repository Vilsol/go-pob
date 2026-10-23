<script lang="ts">
  import type { ClassValue } from 'svelte/elements';
  import { tone, type Tone } from './tone';
  import { artPath, artUrl, spriteOf, type ArtKind, type Sprite } from './item_art';

  type Props = {
    name: string;
    base?: string;
    rarity?: string;
    gem?: boolean;
    /* box edge in px; the box never changes size, whatever loads */
    size?: number;
    /* box height when it is not square; size is then its width */
    height?: number;
    /* rem: the size is read at the default font size and scales with the root font, for icons inside rem-sized boxes */
    unit?: 'px' | 'rem';
    /* the name is already next to the icon: no alt text, hidden from assistive tech */
    decorative?: boolean;
    /* an image URL that skips the art lookup */
    src?: string;
    class?: ClassValue;
  };
  let { name, base = '', rarity = '', gem = false, size = 32, height, unit = 'px', decorative = false, src, class: extra = '' }: Props = $props();

  const unique = $derived(rarity === 'unique' || rarity === 'relic');
  let path = $state<string | undefined>();
  let loaded = $state(false);
  let failed = $state(false);
  let sprite = $state<{ sheet: Sprite; aspect: number } | undefined>();

  $effect(() => {
    const query = { Name: name, Base: base, Unique: unique, Gem: gem };
    loaded = false;
    failed = false;
    sprite = undefined;
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
  const kind: ArtKind = $derived(gem ? 'gem' : /\/Flasks\//.test(url ?? '') || /\bFlask$/.test(base) ? 'flask' : undefined);
  const onload = (img: HTMLImageElement) => {
    const sheet = spriteOf(kind, img.naturalWidth, img.naturalHeight);
    sprite = sheet && { sheet, aspect: img.naturalWidth / sheet.frames / img.naturalHeight };
    loaded = true;
  };
  const len = (px: number) => (unit === 'rem' ? `${px / 16}rem` : `${px}px`);
  const box = $derived(size / (height ?? size));
</script>

<span
  class={['relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-tight', !loaded && 'bg-surface-2', extra]}
  style:width={len(size)}
  style:height={len(height ?? size)}
  aria-hidden={decorative ? 'true' : undefined}
  data-item-icon={loaded ? 'loaded' : showImg ? 'loading' : 'fallback'}>
  {#if !loaded}
    <span
      data-initials={initials}
      class={[
        'font-mono leading-none font-semibold text-(--tone,var(--color-text-3)) select-none before:content-[attr(data-initials)]',
        tone((gem ? 'gem' : rarity || 'normal') as Tone)
      ]}
      style:font-size={len(Math.max(8, Math.round(Math.min(size, height ?? size) * 0.34)))}
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
      class={['absolute inset-0 size-full object-contain', (!loaded || sprite) && 'opacity-0']}
      onload={(e) => onload(e.currentTarget as HTMLImageElement)}
      onerror={() => (failed = true)} />
    {#if loaded && sprite}
      {#each sprite.sheet.layers as frame (frame)}
        <span
          data-sprite-frame={frame}
          aria-hidden="true"
          class="absolute inset-0 m-auto overflow-hidden"
          style:width="{Math.min(1, sprite.aspect / box) * 100}%"
          style:height="{Math.min(1, box / sprite.aspect) * 100}%">
          <img
            src={url}
            alt=""
            draggable="false"
            class="absolute top-0 h-full max-w-none"
            style:width="{sprite.sheet.frames * 100}%"
            style:left="{-frame * 100}%" />
        </span>
      {/each}
    {/if}
  {/if}
</span>
