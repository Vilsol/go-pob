<script lang="ts">
  import { tone as toneClass, type Tone } from './tone';

  type Props = { text: string; meta: string; tone: Tone; current?: boolean; onclick?: () => void };
  let { text, meta, tone, current = false, onclick }: Props = $props();
</script>

<button
  class={[
    'flex justify-between rounded border-0 bg-transparent px-1.5 py-1.25 text-left font-sans text-13 text-(--tone) aria-[current=true]:bg-surface-3',
    toneClass(tone)
  ]}
  aria-current={current || undefined}
  {onclick}><span>{text}</span><span class={['note-sm font-mono', current && 'text-text-2']}>{meta}</span></button>
