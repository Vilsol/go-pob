<script lang="ts">
  import { tone as toneClass, type Tone } from './tone';

  type Props = { tone?: Tone; hollow?: boolean; class?: string };
  let { tone, hollow = false, class: extra = '' }: Props = $props();
</script>

<span class={['dot', toneClass(tone), hollow && 'box-border border border-text-2 bg-transparent', extra]} aria-hidden="true"></span>
