<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Dot from './Dot.svelte';
  import SaveButton from './SaveButton.svelte';

  const { Story } = defineMeta({ title: 'Spec/Dot', component: Dot });
</script>

<Story name="Status" asChild>
  <div class="flex flex-col gap-2">
    <div class="flex items-center gap-2 text-13 text-text-2">
      <SaveButton dirty />unsaved changes: the Save button fills with the accent<SaveButton dirty={false} />saved: plain button
    </div>
    <div class="flex items-center gap-2 text-13 text-text-2"><Dot hollow />Calculating… (values dim after 150 ms, never blank)</div>
  </div>
</Story>

<Story name="Tones" asChild>
  <div class="flex items-center gap-2"><Dot /><Dot tone="up" /><Dot tone="warn" /><Dot tone="error" /><Dot hollow /></div>
</Story>
