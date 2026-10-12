<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import TreeEdge from './TreeEdge.svelte';

  const { Story } = defineMeta({ title: 'Spec/TreeEdge', component: TreeEdge });
</script>

<Story name="States and angles" asChild>
  <div class="relative h-24 w-80 rounded-md border border-line bg-tree-ground">
    <TreeEdge x={16} y={20} w={64} state="on" />
    <TreeEdge x={96} y={20} w={64} state="path" />
    <TreeEdge x={176} y={20} w={64} state="lose" />
    <TreeEdge x={256} y={20} w={48} />
    <TreeEdge x={40} y={50} w={60} r={30} state="on" />
    <TreeEdge x={160} y={80} w={60} r={-30} state="path" />
  </div>
</Story>
