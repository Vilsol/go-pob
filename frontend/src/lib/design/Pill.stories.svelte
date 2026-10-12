<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Pill from './Pill.svelte';
  import Tag from './Tag.svelte';

  const { Story } = defineMeta({ title: 'Spec/Pill', component: Pill });
</script>

<Story name="Pills and tags" asChild>
  <div class="flex items-center gap-2">
    <Pill winner="a">A +18 pp</Pill><Pill winner="b">B −26.2%</Pill><Pill>=</Pill>
    <Tag kind="a">A only</Tag><Tag kind="b">B only</Tag><Tag kind="level">lvl/q</Tag><Tag kind="same">same</Tag>
  </div>
</Story>
