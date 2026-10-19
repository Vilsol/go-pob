<script lang="ts">
  import EmptyState from './EmptyState.svelte';
  import Segmented from './Segmented.svelte';
  import { isTypingTarget } from './keys';
  import { tick } from 'svelte';
  import type { BuildWarning } from '$lib/display/sidebar';
  import { issueKinds } from './issue_kinds';
  import Button from './Button.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Icon from './Icon.svelte';
  import IssueRow from './IssueRow.svelte';
  import LogList from './LogList.svelte';
  import Menu from './Menu.svelte';
  import NavTabs from './NavTabs.svelte';
  import SaveButton from './SaveButton.svelte';
  import Select from './Select.svelte';
  import Tabs from './Tabs.svelte';

  type Tab = { label: string; href: string; key?: string };
  type HistoryEntry = { action: string; page: string; when: string };
  type LogLine = { time: string; level: 'info' | 'warn' | 'error'; src: string; msg: string };
  type Panel = 'issues' | 'logs';
  type Props = {
    name: string;
    tabs: Tab[];
    current: string;
    backHref?: string;
    importHref?: string;
    history: HistoryEntry[];
    undone?: number;
    historyMax?: number;
    loadouts: string[];
    loadout?: string;
    issues: BuildWarning[];
    logs: LogLine[];
    dirty: boolean;
    panel?: Panel | null;
    historyOpen?: boolean;
    onnavigate?: (tab: Tab, e: Event) => void;
    onback?: (e: MouseEvent) => void;
    onundo?: () => void;
    onredo?: () => void;
    onhistory?: (i: number) => void;
    onloadout?: (name: string) => void;
    onsave?: () => void;
    /* Build.lua:1160-1162 */
    onimport?: () => void;
    onsaveas?: () => void;
    onsettings?: () => void;
    /* Main.lua:209-211 */
    onabout?: () => void;
    /* Main.lua:230-241 */
    version?: string;
    devMode?: boolean;
  };
  let {
    name,
    tabs,
    current,
    backHref = '#builds',
    importHref = '#import',
    history,
    undone = 0,
    historyMax = 50,
    loadouts,
    loadout,
    issues,
    logs,
    dirty,
    panel = $bindable(null),
    historyOpen = $bindable(false),
    onnavigate,
    onback,
    onundo,
    onredo,
    onhistory,
    onloadout,
    onsave,
    onimport,
    onsaveas,
    onsettings,
    onabout,
    version,
    devMode = false
  }: Props = $props();

  const uid = $props.id();

  const undoLabel = $derived(undone < history.length ? `Undo: ${history[undone].action} (Ctrl+Z)` : 'Nothing to undo');
  const redoLabel = $derived(undone > 0 ? `Redo: ${history[undone - 1].action} (Ctrl+Y / Ctrl+Shift+Z)` : 'Nothing to redo');
  type HistoryItem = { label: string; current?: boolean };
  const historyItems = $derived(
    history.map((h, i) => ({ label: h.action, current: i === undone, tone: i < undone ? ('muted' as const) : undefined, onselect: () => onhistory?.(i) }))
  );

  let pressTimer: ReturnType<typeof setTimeout> | undefined;
  let longPressed = false;
  const pressStart = (e: PointerEvent) => {
    if (e.button) return;
    longPressed = false;
    clearTimeout(pressTimer);
    pressTimer = setTimeout(() => {
      longPressed = true;
      historyOpen = true;
    }, 500);
  };
  const pressEnd = () => clearTimeout(pressTimer);
  const undoClick = () => {
    if (longPressed) {
      longPressed = false;
      return;
    }
    onundo?.();
  };

  function onkeydown(e: KeyboardEvent) {
    if (!(e.ctrlKey || e.metaKey) || e.altKey) return;
    const typing = isTypingTarget(e.target);
    const k = e.key.toLowerCase();
    if (k === 's') {
      e.preventDefault();
      onsave?.();
    } else if (typing) return;
    else if (k === 'i' && onimport && !e.shiftKey) {
      e.preventDefault();
      onimport();
    } else if (k === 'z' && !e.shiftKey && undone < history.length) {
      e.preventDefault();
      onundo?.();
    } else if ((k === 'y' || (k === 'z' && e.shiftKey)) && undone > 0) {
      e.preventDefault();
      onredo?.();
    }
  }

  const errors = $derived(logs.filter((l) => l.level === 'error').length);
  const panelTabs = $derived([
    { id: 'issues', label: 'Build issues', badge: { text: String(issues.length), tone: 'warn' as const } },
    { id: 'logs', label: 'Logs', badge: errors ? { text: `${errors} error`, tone: 'error' as const } : { text: String(logs.length), tone: 'muted' as const } }
  ]);
  let level = $state<'all' | LogLine['level']>('all');
  const levels = $derived([
    { id: 'all' as const, label: `All ${logs.length}` },
    { id: 'info' as const, label: 'Info' },
    { id: 'warn' as const, label: 'Warn' },
    { id: 'error' as const, label: 'Error' }
  ]);

  let issuesBtn: HTMLButtonElement;
  let logsBtn: HTMLButtonElement;
  let closeBtn = $state<HTMLButtonElement>();
  async function toggle(lane: Panel) {
    panel = panel === lane ? null : lane;
    if (panel) {
      await tick();
      closeBtn?.focus();
    }
  }
  function closePanel() {
    const back = panel === 'logs' ? logsBtn : issuesBtn;
    panel = null;
    back?.focus();
  }
</script>

{#snippet lanePanel(lane: string)}
  {#if lane === 'issues'}
    <div class="note border-b border-surface-3 px-4 py-2">Problems with this build found by the calculation. They clear themselves when fixed.</div>
    {#if issues.length}
      <ul aria-label="Build issues" class="scroll m-0 max-h-130 list-none p-0">
        {#each issues as it, i (i)}
          <li><IssueRow text={it.text} kind={issueKinds[it.kind][0]} where={issueKinds[it.kind][1]} /></li>
        {/each}
      </ul>
    {:else}
      <EmptyState inset="row" text="No build issues." />
    {/if}
  {:else}
    <Segmented
      chip
      label="Level"
      options={levels}
      value={level}
      onchange={(id) => (level = id as typeof level)}
      class="border-b border-surface-3 px-3 py-2"
      itemClass="h-6.5 px-2.5" />
    <div class="scroll max-h-130">
      <LogList lines={logs.filter((l) => level === 'all' || l.level === level)} />
    </div>
  {/if}
{/snippet}

<svelte:window {onkeydown} />

<header
  class="relative box-border flex h-13 w-full min-w-0 items-stretch gap-3 border-b border-line bg-surface-1 px-5 font-sans text-text max-wide:gap-2.5 max-wide:px-3 max-narrow:gap-2">
  <div class="flex min-w-0 shrink items-center gap-3 max-wide:gap-2.5">
    <a href={backHref} onclick={onback} class="shrink-0 text-14 whitespace-nowrap text-text-2 no-underline" aria-label="Back to builds"
      >←<span class="max-wide:hidden">&nbsp;Builds</span></a>
    <div class="min-w-0 truncate text-16 font-semibold max-wide:text-15" title={name}>{name}</div>
    <Menu
      label="Undo history"
      items={historyItems}
      bind:open={historyOpen}
      class="top-10 right-auto left-0 w-90"
      itemClass="h-auto min-h-7.5 justify-between gap-3 py-1">
      {#snippet trigger(attrs)}
        <div class="flex shrink-0 gap-1">
          <Button
            icon
            onclick={undoClick}
            oncontextmenu={(e: MouseEvent) => {
              e.preventDefault();
              historyOpen = true;
            }}
            onpointerdown={pressStart}
            onpointerup={pressEnd}
            onpointerleave={pressEnd}
            disabled={undone >= history.length}
            aria-label={undoLabel}
            title={undoLabel}>
            <svg
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"><path d="M5 3 2 6l3 3"></path><path d="M2 6h8a4 4 0 0 1 0 8H7"></path></svg>
          </Button>
          <Button icon size="xs" class="h-8 w-4 border-0" {...attrs} aria-label="Undo history (right-click or long-press Undo)" title="Undo history">▾</Button>
          <Button icon onclick={onredo} disabled={undone === 0} aria-label={redoLabel} title={redoLabel}>
            <svg
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"><path d="m11 3 3 3-3 3"></path><path d="M14 6H6a4 4 0 0 0 0 8h3"></path></svg>
          </Button>
        </div>
      {/snippet}
      {#snippet head()}
        <div class="flex justify-between px-2 pt-0.5 pb-1 text-11 text-text-3">
          <span>Undo history · last {historyMax} actions, all pages</span><span>click = go back to</span>
        </div>
      {/snippet}
      {#snippet row(it: HistoryItem, i: number)}
        <span class="flex min-w-0 items-center gap-1.5"
          ><span class="w-3 shrink-0 text-accent" aria-hidden="true">{it.current ? '●' : ''}</span><span class={['truncate', i < undone && 'line-through']}
            >{it.label}</span
          ></span
        ><span class="shrink-0 text-11 text-text-3">{history[i].page} · {history[i].when}</span>
      {/snippet}
    </Menu>
  </div>

  <NavTabs
    {tabs}
    {current}
    {onnavigate}
    label="Build pages"
    class="shrink-0 [&_.tab]:px-3 max-wide:[&_.tab]:px-2.25 max-wide:[&_.tab]:text-13 max-wide:[&_.tab-key]:hidden max-narrow:[&_.tab]:px-1.25 max-narrow:[&_.tab]:text-12" />

  <div class="grow"></div>

  <div class="flex shrink-0 items-center gap-3 max-wide:gap-2 max-narrow:gap-1.5">
    {#if loadouts.length}
      <FloatingLabel label="Loadout">
        <Select
          options={loadouts}
          value={loadout}
          class="max-wide:max-w-32 max-narrow:max-w-20"
          title="Switches tree, skill, item and config sets together"
          onchange={(e: Event) => onloadout?.((e.currentTarget as HTMLSelectElement).value)} />
      </FloatingLabel>
    {/if}
    <button
      bind:this={issuesBtn}
      class={[
        'flex h-8 items-center gap-1.5 rounded border px-2.5 font-sans text-13 font-semibold',
        issues.length ? 'border-warn-line bg-warn-bg-2 text-worse' : 'border-line bg-transparent text-text-2'
      ]}
      onclick={() => toggle('issues')}
      aria-haspopup="dialog"
      aria-expanded={panel === 'issues'}
      aria-label="{issues.length} build issues"
      title="Build issues: problems the calculation found in this build"><span aria-hidden="true">⚠</span>{issues.length}</button>
    <button
      bind:this={logsBtn}
      class="btn px-3 max-narrow:px-2"
      onclick={() => toggle('logs')}
      aria-haspopup="dialog"
      aria-expanded={panel === 'logs'}
      aria-label={errors ? `Logs, ${errors} error${errors > 1 ? 's' : ''}` : 'Logs'}
      title="Application logs: data, worker, save and import"
      >Logs{#if errors}<span aria-hidden="true" class="dot tone-error size-1.75"></span>{/if}</button>
    <a
      href={importHref}
      class="btn max-wide:size-8 max-wide:p-0"
      aria-current={current === importHref ? 'page' : undefined}
      aria-label="Import / Export"
      title="Import / Export"><span class="hidden max-wide:flex"><Icon name="import-export" /></span><span class="max-wide:hidden">Import / Export</span></a>
    <SaveButton {dirty} {onsave} class="max-wide:size-8 max-wide:p-0 max-wide:[&_.save-icon]:flex max-wide:[&_.save-text]:hidden" />
    {#if onsaveas}
      <Button
        onclick={onsaveas}
        aria-haspopup="dialog"
        title="Save a copy of this build under a new name"
        class="max-wide:px-2.5 max-narrow:px-1.5 max-narrow:text-12">Save As</Button>
    {/if}
    {#if devMode}<span class="text-11 font-semibold whitespace-nowrap text-worse">Dev Mode</span>{/if}
    {#if onabout}<Button
        icon
        onclick={onabout}
        aria-haspopup="dialog"
        aria-label="About go-pob"
        title={['About go-pob', version && `Version: ${version}`, version && devMode && 'Dev Mode'].filter(Boolean).join(' · ')}><Icon name="info" /></Button
      >{/if}
    <Button icon onclick={onsettings} aria-haspopup="dialog" aria-label="Settings" title="Settings"><Icon name="gear" /></Button>
  </div>

  {#if panel}
    <div
      role="dialog"
      aria-labelledby="{uid}-panel-title"
      tabindex="-1"
      class="fixed top-14 right-4 z-dropdown box-border flex h-160 max-h-[calc(100vh-72px)] w-120 flex-col overflow-hidden rounded-ovl border border-line-strong bg-surface-2 text-text shadow-panel"
      onkeydown={(e) => {
        if (e.key !== 'Escape') return;
        e.preventDefault();
        closePanel();
      }}>
      <div class="flex items-center gap-2 pt-2.5 pr-3 pl-4">
        <h2 id="{uid}-panel-title" class="grow text-15">{panel === 'issues' ? 'Build issues' : 'Logs'}</h2>
        <button bind:this={closeBtn} class="btn-close" onclick={closePanel} aria-label="Close panel (Esc)">×</button>
      </div>
      <Tabs tabs={panelTabs} bind:selected={() => panel ?? 'issues', (v) => (panel = v as Panel)} label="Panels" panel={lanePanel} />
    </div>
  {/if}
</header>
