<script lang="ts">
  import type { Snippet } from 'svelte';
  import Socket from './Socket.svelte';
  import NumberInput from './NumberInput.svelte';
  import { tone as toneClass, type Tone } from './tone';
  import PobText from './PobText.svelte';
  import { gemWikiName } from './wiki';

  // SkillsTab.lua:1007-1012, SkillsTab.lua:1014-1050
  type Props = {
    n: number;
    name: string;
    level: number;
    quality: number;
    tone: Tone;
    socket: Tone;
    support?: boolean;
    from?: string;
    linked?: boolean;
    onlinkchange?: (linked: boolean) => void;
    enabled?: boolean;
    onenabled?: (enabled: boolean) => void;
    onlevel?: (level: number) => void;
    onquality?: (quality: number) => void;
    onremove?: () => void;
    count?: number;
    oncount?: (count: number) => void;
    maxLevel?: number;
    linkLocked?: string;
    error?: string;
    globals?: { label: string; on: boolean }[];
    onglobal?: (index: number, on: boolean) => void;
    grip?: Snippet;
    /* SkillsTab.lua:861-920 */
    onqualityhover?: (on: boolean) => void;
    /* GemSelectControl.lua:629-646 */
    onnamehover?: (on: boolean) => void;
  };
  let {
    n,
    name,
    level,
    quality,
    tone,
    socket,
    support = false,
    from,
    linked = $bindable(),
    onlinkchange,
    enabled,
    onenabled,
    onlevel,
    onquality,
    onremove,
    count = 1,
    oncount,
    maxLevel = 40,
    linkLocked,
    error,
    globals,
    onglobal,
    grip,
    onqualityhover,
    onnamehover
  }: Props = $props();
  const shownGlobals = $derived((globals ?? []).map((g, i) => ({ ...g, i })).filter((g) => g.label));

  const linkLabel = $derived(`Sockets ${n - 1}–${n}: ${linked ? 'linked' : 'not linked'}`);
</script>

<div class={['gem-row', from && 'bg-info-bg', (error || shownGlobals.length) && 'h-auto py-1']} data-wiki={name ? gemWikiName(name, support) : undefined}>
  <span class="flex">{@render grip?.()}</span>
  <div class="relative flex h-full items-center justify-center">
    {#if linked !== undefined}
      <button
        class="socket-link"
        aria-pressed={linked}
        aria-label={linkLabel}
        title={linkLocked ? `${linkLabel} (${linkLocked})` : linkLabel}
        disabled={!!linkLocked}
        onclick={() => {
          linked = !linked;
          onlinkchange?.(linked);
        }}></button>
    {/if}
    <Socket {socket} gem={from ? 'flask' : tone} />
  </div>
  <span
    >{#if enabled !== undefined}<input
        type="checkbox"
        class="check size-3.75"
        checked={enabled}
        aria-label="{name} enabled"
        onchange={(e) => onenabled?.(e.currentTarget.checked)} />{/if}</span>
  {#if onlevel}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <span
      class={['toned truncate', oncount ? 'col-span-10' : 'col-span-13', toneClass(tone), !support && 'font-semibold', enabled === false && 'line-through']}
      onmouseenter={() => onnamehover?.(true)}
      onmouseleave={() => onnamehover?.(false)}>{name}</span>
    <span class="col-span-4"><NumberInput size="xs" value={level} min={1} max={maxLevel} label="{name} level" class="w-full" onchange={onlevel} /></span>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <span
      class="col-span-4"
      onmouseenter={() => onqualityhover?.(true)}
      onmouseleave={() => onqualityhover?.(false)}
      onfocusin={() => onqualityhover?.(true)}
      onfocusout={() => onqualityhover?.(false)}
      ><NumberInput size="xs" value={quality} min={0} max={23} label="{name} quality" class="w-full" onchange={onquality} /></span>
    {#if oncount}<span class="col-span-3"><NumberInput size="xs" value={count} min={1} max={99} label="{name} count" class="w-full" onchange={oncount} /></span
      >{/if}
    <span class="col-span-2 flex justify-end">
      {#if onremove}<button class="btn-bare size-5.5" aria-label="Remove {name}" title="Remove this gem" onclick={onremove}>×</button>{/if}
    </span>
  {:else}
    <span class={['toned col-span-14', toneClass(tone), !support && 'font-semibold']}
      >{name}{#if from}<span class="note-sm font-normal">{'  from ' + from}</span>{/if}</span>
    <span class="col-span-5 font-mono">{level} / {quality}</span>
    <span class="col-span-4"></span>
  {/if}
  {#if shownGlobals.length}
    <span class="col-span-full flex flex-wrap gap-x-4 pl-8 text-12">
      {#each shownGlobals as g (g.i)}
        <label class="flex items-center gap-1.5"
          >Enable {g.label}:<input
            type="checkbox"
            class="check size-3.5"
            checked={g.on}
            aria-label="Enable {g.label}"
            onchange={(e) => onglobal?.(g.i, e.currentTarget.checked)} /></label>
      {/each}
    </span>
  {/if}
  {#if error}<span class="col-span-full pl-8 text-12" data-gem-error><PobText text={`^1${error}`} /></span>{/if}
</div>
