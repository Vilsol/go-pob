<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import LogList from './LogList.svelte';

  const { Story } = defineMeta({
    title: 'Spec/LogList',
    component: LogList,
    args: {
      lines: [
        { time: '12:04:01', level: 'info', src: 'worker', msg: 'Calculation worker started (wasm 21.3 MB)' },
        {
          time: '12:06:41',
          level: 'warn',
          src: 'import',
          msg: 'Item “Doom Visor”: unknown mod line kept as custom: “+1 to Maximum Frenzy Charges (fractured)”'
        },
        { time: '12:11:15', level: 'error', src: 'save', msg: 'Save failed: browser storage quota exceeded (retrying after cleanup)' },
        { time: '12:11:16', level: 'info', src: 'save', msg: 'Saved after removing 2 old autosaves' }
      ]
    }
  });
</script>

<Story name="Levels" />
