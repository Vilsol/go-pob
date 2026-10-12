<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import RadiusRing from './RadiusRing.svelte';
  import TreeNode from './TreeNode.svelte';

  const { Story } = defineMeta({ title: 'Spec/RadiusRing', component: RadiusRing });
</script>

<Story name="Jewel radius" asChild>
  <div class="relative h-24 w-60 overflow-hidden rounded-md border border-line bg-tree-ground">
    <RadiusRing x={20} y={8} z={80} />
    <RadiusRing x={150} y={8} z={80} hot />
    <TreeNode kind="j" x={53} y={41} z={14} alloc label="socketed jewel" />
    <TreeNode kind="j" x={183} y={41} z={14} hover label="hovered socket" />
    <TreeNode kind="n" x={205} y={22} z={14} radius label="notable in radius" />
    <TreeNode kind="s" x={162} y={68} z={8} radius label="small node in radius" />
  </div>
</Story>
