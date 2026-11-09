<script lang="ts">
  import { activityPanelOpen, jobs, runningJobs } from '$lib/activity/jobs';
  import ActivityPanel from './ActivityPanel.svelte';
  import Dot from './Dot.svelte';
  import Icon from './Icon.svelte';
  import Spinner from './Spinner.svelte';

  // the top bar's spinner with the current job, shown while something runs or a job failed; floating, it sits in the
  // bottom corner of a page without a top bar
  let { floating = false, class: extra = '' }: { floating?: boolean; class?: string } = $props();
  let btn = $state<HTMLButtonElement>();

  const current = $derived($runningJobs.at(-1));
  const failed = $derived($jobs.filter((j) => j.state === 'error').length);
  const phase = $derived(current?.phases.at(-1)?.name);
  const text = $derived(current ? [current.label, phase].filter(Boolean).join(' · ') : failed ? `${failed} failed` : '');
  const percent = $derived(current?.total ? `${Math.round(Math.min(1, (current.done ?? 0) / current.total) * 100)}%` : '');

  const close = () => {
    activityPanelOpen.set(false);
    btn?.focus();
  };
</script>

{#if $runningJobs.length || failed || $activityPanelOpen}
  <div
    class={[
      'flex items-center',
      floating ? 'fixed right-3 bottom-3 z-toast max-phone:bottom-[calc(3.5rem+env(safe-area-inset-bottom)+0.75rem)]' : 'relative',
      extra
    ]}>
    <button
      bind:this={btn}
      class={[
        'flex h-8 max-w-72 min-w-0 items-center gap-1.5 rounded border border-line bg-surface-1 px-2.5 font-sans text-12 text-text-2',
        floating && 'shadow-pop'
      ]}
      aria-haspopup="dialog"
      aria-expanded={$activityPanelOpen}
      aria-label={text ? `Activity: ${text}` : 'Activity'}
      title="Activity: what is loading or running, and what finished"
      onclick={() => activityPanelOpen.update((o) => !o)}>
      {#if current}<Spinner />{:else if failed}<Dot tone="error" />{:else}<Icon name="check" size={14} />{/if}
      {#if text}<span class="min-w-0 truncate max-narrow:hidden">{text}</span>{/if}
      {#if percent}<span class="font-mono text-11 text-text-3">{percent}</span>{/if}
      {#if $runningJobs.length > 1}<span class="text-11 text-text-3">+{$runningJobs.length - 1}</span>{/if}
    </button>
    <span role="status" aria-live="polite" class="sr-only">{text}</span>
  </div>
  {#if $activityPanelOpen}
    <ActivityPanel
      onclose={close}
      class={floating ? 'fixed right-3 bottom-14 max-phone:bottom-[calc(6.5rem+env(safe-area-inset-bottom))]' : 'fixed top-14 right-4'} />
  {/if}
{/if}
