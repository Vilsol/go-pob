<script lang="ts">
  import TopBar from '../TopBar.svelte';
  import type { BuildFixture } from '../fixtures';
  import { pageTabs, undoHistory, loadouts, appLog } from '../fixtures/shell';

  type Props = {
    fx: BuildFixture;
    dirty?: boolean;
    panel?: 'issues' | 'logs' | null;
    historyOpen?: boolean;
    current?: string;
    onsaveas?: () => void;
  };
  let { fx, dirty = true, panel = null, historyOpen = false, current = '#tree', onsaveas }: Props = $props();

  let s = $state({ current, undone: 0, dirty, panel, historyOpen, loadout: loadouts[0] });
</script>

<TopBar
  name={fx.name}
  tabs={pageTabs}
  current={s.current}
  history={undoHistory}
  undone={s.undone}
  {loadouts}
  loadout={s.loadout}
  issues={fx.sidebar.warnings}
  logs={appLog}
  oncopylogs={() => {}}
  dirty={s.dirty}
  bind:panel={s.panel}
  bind:historyOpen={s.historyOpen}
  onnavigate={(t: { href: string }, e: Event) => {
    e.preventDefault();
    s.current = t.href;
  }}
  onundo={() => {
    s.undone++;
    s.dirty = true;
  }}
  onredo={() => {
    s.undone--;
    s.dirty = true;
  }}
  onhistory={(i: number) => {
    s.undone = i;
    s.dirty = true;
  }}
  onloadout={(l: string) => (s.loadout = l)}
  onsave={() => (s.dirty = false)}
  {onsaveas} />
