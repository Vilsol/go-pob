<script lang="ts">
  import type { Snippet } from 'svelte';
  import MobileFrame from '../mobile/MobileFrame.svelte';
  import StatsStrip from '../mobile/StatsStrip.svelte';
  import LiveSidebar from './LiveSidebar.svelte';
  import type { BuildFixture } from '../fixtures';
  import { undoHistory, loadouts, appLog } from '../fixtures/shell';
  import type { StatCompareLine } from '$lib/skill_tree/stat_compare';

  type Props = {
    fx: BuildFixture;
    current: string;
    label: string;
    delta?: StatCompareLine[];
    pinned?: string[];
    statsOpen?: boolean;
    moreOpen?: boolean;
    mainClass?: string;
    children: Snippet;
    onnavigate?: (href: string) => void;
  };
  let { fx, current, label, delta = [], pinned = [], statsOpen = false, moreOpen = false, mainClass, children, onnavigate }: Props = $props();

  let s = $state({ current, undone: 0, dirty: true, statsOpen, moreOpen, loadout: loadouts[0], log: '' });
  const actions = $derived([
    { key: 'history', label: 'Undo history', note: `${undoHistory.length - s.undone} steps` },
    { key: 'loadout', label: 'Loadout', note: s.loadout },
    { key: 'saveas', label: 'Save as…' },
    { key: 'issues', label: 'Build issues', note: String(fx.sidebar.warnings.length) },
    { key: 'logs', label: 'Logs', note: String(appLog.length) },
    { key: 'settings', label: 'Settings' },
    { key: 'about', label: 'About' }
  ]);
  // the strip and the sheet's sidebar share the pins
  let pins = $state([...pinned]);
</script>

<MobileFrame
  name={fx.name}
  current={s.current}
  {label}
  {mainClass}
  dirty={s.dirty}
  canUndo={s.undone < undoHistory.length}
  canRedo={s.undone > 0}
  {actions}
  bind:statsOpen={s.statsOpen}
  bind:moreOpen={s.moreOpen}
  onnavigate={(href: string) => {
    s.current = href;
    s.log = `navigate ${href}`;
    onnavigate?.(href);
  }}
  onback={() => (s.log = 'back')}
  onundo={() => {
    s.undone++;
    s.dirty = true;
  }}
  onredo={() => {
    s.undone--;
    s.dirty = true;
  }}
  onsave={() => (s.dirty = false)}
  onaction={(k: string) => (s.log = `action ${k}`)}>
  {#snippet strip(open: () => void)}<StatsStrip sidebar={fx.sidebar} pinned={pins} {delta} onopen={open} />{/snippet}
  {#snippet stats()}<LiveSidebar {fx} bind:pinned={pins} fill />{/snippet}
  {@render children()}
</MobileFrame>
<span class="sr-only" data-log>{s.log}</span>
