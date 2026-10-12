<script lang="ts">
  type Line = { time: string; level: 'info' | 'warn' | 'error'; src: string; msg: string };
  type Props = { lines: Line[]; label?: string };
  let { lines, label = 'Application log' }: Props = $props();

  const levelTone = { info: 'tone-better', warn: 'tone-worse', error: 'tone-error' };
</script>

<ol aria-label={label} class="log-list">
  {#each lines as l, i (i)}
    <li class={['log-row', l.level === 'error' && 'bg-error/8']}>
      <span class="text-text-4">{l.time}</span><span class={['toned font-semibold', levelTone[l.level]]}>{l.level.toUpperCase()}</span><span class="text-text-3"
        >{l.src}</span
      ><span class="text-text-soft [overflow-wrap:anywhere]">{l.msg}</span>
    </li>
  {/each}
</ol>
