<script lang="ts">
  import { onMount } from 'svelte';
  import { clearFinished, elapsed, jobs, remaining, usually, type Job } from '$lib/activity/jobs';
  import { formatAbout, formatProgress, formatSeconds } from '$lib/activity/format';
  import Button from './Button.svelte';
  import Icon from './Icon.svelte';
  import Spinner from './Spinner.svelte';

  // every running job and the recently finished ones, with their phases, counts or bytes and times
  let { onclose, class: extra = '' }: { onclose: () => void; class?: string } = $props();
  const uid = $props.id();
  let el = $state<HTMLElement>();
  let now = $state(Date.now());
  onMount(() => {
    el?.focus();
    const t = setInterval(() => (now = Date.now()), 500);
    return () => clearInterval(t);
  });
  const finished = $derived($jobs.filter((j) => j.state !== 'running'));

  const status = (j: Job) => {
    const parts: string[] = [];
    const phase = j.phases.at(-1);
    if (j.state === 'running' && phase) parts.push(phase.name);
    if (j.state === 'running' && j.done !== undefined) parts.push(formatProgress(j.done, j.total, j.unit));
    if (j.state === 'running') {
      const left = remaining(j, now);
      if (left !== undefined) parts.push(`${formatAbout(left)} left`);
      const typical = usually(j);
      if (typical !== undefined) parts.push(`usually ${formatAbout(typical)}`);
    } else if (j.state === 'ok') parts.push('Done');
    return parts.join(' · ');
  };
</script>

<div
  bind:this={el}
  role="dialog"
  aria-labelledby="{uid}-title"
  tabindex="-1"
  class={[
    'z-dropdown box-border flex max-h-[calc(100vh-72px)] w-100 max-w-[calc(100vw-1rem)] flex-col overflow-hidden rounded-ovl border border-line-strong bg-surface-2 text-text shadow-panel',
    extra
  ]}
  onkeydown={(e) => {
    if (e.key !== 'Escape') return;
    e.preventDefault();
    onclose();
  }}>
  <div class="flex items-center gap-2 border-b border-surface-3 py-2 pr-3 pl-4">
    <h2 id="{uid}-title" class="grow text-15">Activity</h2>
    {#if finished.length}<Button size="xs" variant="ghost" onclick={clearFinished}>Clear finished</Button>{/if}
    <Button variant="close" onclick={onclose} aria-label="Close activity (Esc)"><Icon name="close" size={14} /></Button>
  </div>
  {#if !$jobs.length}
    <p class="note m-0 p-4">Nothing is running.</p>
  {:else}
    <ul aria-label="Jobs" class="scroll m-0 list-none p-0">
      {#each $jobs as j (j.id)}
        <li class="flex flex-col gap-1 border-b border-surface-3 px-4 py-2.5 text-12 last:border-b-0" aria-busy={j.state === 'running'}>
          <div class="flex items-center gap-2">
            {#if j.state === 'running'}<Spinner />{:else if j.state === 'error'}<span class="text-error"><Icon name="warn" size={14} /></span>{:else}<span
                class="text-better"><Icon name="check" size={14} /></span
              >{/if}
            <span class="min-w-0 grow truncate text-13 font-semibold">{j.label}</span>
            <span class="font-mono text-11 text-text-3" title="Elapsed">{formatSeconds(elapsed(j, now))}</span>
          </div>
          {#if status(j)}<span class="text-text-2">{status(j)}</span>{/if}
          {#if j.state === 'running' && j.total}
            <div class="h-1 overflow-hidden rounded-full bg-surface-3" aria-hidden="true">
              <div class="h-full bg-accent" style:width="{Math.min(1, (j.done ?? 0) / j.total) * 100}%"></div>
            </div>
          {/if}
          {#if j.error}<span class="break-words text-error" role="alert">{j.error}</span>{/if}
          {#if j.phases.length > 1 || (j.state !== 'running' && j.phases.length)}
            <ol aria-label="Phases of {j.label}" class="m-0 flex list-none flex-col gap-0.5 p-0 pl-5.5 text-11 text-text-3">
              {#each j.phases as p, i (i)}
                <li class="flex justify-between gap-2">
                  <span class="truncate"
                    >{p.name}{#if p.done !== undefined}&nbsp;· {formatProgress(p.done, p.total, p.unit)}{/if}</span>
                  <span class="font-mono">{p.end === undefined ? '…' : formatSeconds(p.end - p.start)}</span>
                </li>
              {/each}
            </ol>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>
