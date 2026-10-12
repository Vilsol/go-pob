<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Socket from './Socket.svelte';

  const { Story } = defineMeta({ title: 'Spec/Socket', component: Socket, args: { socket: 'B', gem: 'int' } });
</script>

<Story name="Matching gem" />

<Story name="States" asChild>
  <div class="flex items-center gap-3">
    <Socket socket="R" gem="str" />
    <Socket socket="G" gem="dex" />
    <Socket socket="B" gem="int" />
    <Socket socket="W" gem="white" />
    <Socket socket="warn" gem="dex" />
    <Socket socket="B" gem="flask" />
    <Socket socket="B" />
  </div>
</Story>
