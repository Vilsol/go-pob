<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Tag from './Tag.svelte';

  const { Story } = defineMeta({ title: 'Spec/Tag', component: Tag });
</script>

<Story name="Kinds" asChild>
  <div class="flex items-center gap-2"><Tag kind="a">A only</Tag><Tag kind="b">B only</Tag><Tag kind="level">lvl/q</Tag><Tag kind="same">same</Tag></div>
</Story>
