<script lang="ts">
  import AboutDialog from '../AboutDialog.svelte';
  import { aboutInfo } from '../fixtures';
  import { parseHelp } from '../help';
  import helpText from '../../../../static/help.txt?raw';

  type Props = { section?: string };
  let { section }: Props = $props();
  let open = $state(false);
  let tab = $state('help');
  const help = parseHelp(helpText, false);
</script>

<button
  class="btn m-4"
  onclick={() => {
    tab = 'help';
    open = true;
  }}>Help</button>
<AboutDialog bind:open bind:tab info={aboutInfo} {help} helpSection={section} />
