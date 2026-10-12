<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import TreeNode from './TreeNode.svelte';
  import TreeEdge from './TreeEdge.svelte';
  import MapNote from './MapNote.svelte';

  const { Story } = defineMeta({
    title: 'Spec/TreeNode',
    component: TreeNode,
    argTypes: { kind: { control: 'inline-radio', options: ['s', 'n', 'k', 'm', 'j', 'c'] } },
    args: { kind: 'n', x: 8, y: 8, z: 14, alloc: true, label: 'allocated' }
  });

  type Kind = 's' | 'n' | 'k' | 'm' | 'j';
  const size = { s: 8, n: 14, k: 19, m: 12, j: 14 };
  const strip: [Kind, string, { alloc?: boolean; state?: 'path' | 'lose'; hover?: boolean; match?: boolean; conq?: boolean }][] = [
    ['n', 'allocated', { alloc: true }],
    ['s', 'path', { state: 'path' }],
    ['n', 'would lose', { alloc: true, state: 'lose' }],
    ['n', 'search hit', { match: true }],
    ['n', 'conquered', { conq: true }],
    ['k', 'hovered', { hover: true }],
    ['m', 'mastery', {}],
    ['j', 'jewel', {}]
  ];
  const nodes = strip.map(([kind, label, f], i) => ({ kind, label, ...f, cx: 24 + i * 64, z: size[kind] }));
</script>

<Story name="Playground" asChild>
  <div class="relative h-8 w-8 bg-tree-ground"><TreeNode kind="n" x={9} y={9} z={14} alloc label="allocated" /></div>
</Story>

<Story name="States" asChild>
  <div class="relative h-19 w-150 overflow-hidden rounded-md border border-line bg-tree-ground text-11 text-text-2">
    <TreeEdge x={24} y={20} w={64} state="on" />
    <TreeEdge x={88} y={20} w={64} state="path" />
    <TreeEdge x={152} y={20} w={64} state="lose" />
    <TreeEdge x={216} y={20} w={64} />
    {#each nodes as { cx, ...n } (n.label)}
      <TreeNode {...n} x={cx - n.z / 2} y={21 - n.z / 2} />
      <span class="absolute top-9 left-(--x) w-16 -translate-x-1/2 text-center" style:--x="{cx}px">{n.label}</span>
    {/each}
    <MapNote>map note</MapNote>
  </div>
</Story>
