<script lang="ts">
  import { tone as toneClass, type Tone } from './tone';

  type Props = { socket: Tone; gem?: Tone };
  let { socket, gem }: Props = $props();
</script>

<span class={['socket', toneClass(socket)]}>
  {#if gem}<span aria-hidden="true" class={['socket-gem', toneClass(gem)]}></span>{/if}
</span>
