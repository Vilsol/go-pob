<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Input from './Input.svelte';

  const { Story } = defineMeta({
    title: 'Spec/Input',
    component: Input,
    argTypes: { size: { control: 'inline-radio', options: ['xs', 'default', 'lg'] } },
    args: { size: 'default', type: 'search', placeholder: 'Search nodes, stats, gems', 'aria-label': 'Search' }
  });
</script>

<Story name="Playground" />

<Story name="Sizes" asChild>
  <div class="flex items-center gap-2">
    <Input size="xs" class="w-16 font-mono" placeholder="50" aria-label="xs" />
    <Input class="w-16 font-mono" value={94} aria-label="default" />
    <Input size="lg" class="w-22 bg-surface-1 font-mono" value={94} aria-label="lg" />
  </div>
</Story>
