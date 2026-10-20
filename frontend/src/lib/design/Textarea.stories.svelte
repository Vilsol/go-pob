<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect } from 'storybook/test';
  import Textarea from './Textarea.svelte';

  const { Story } = defineMeta({ title: 'Spec/Textarea', component: Textarea });
</script>

<Story
  name="Code"
  asChild
  play={async ({ canvasElement }) => {
    const area = (canvasElement as HTMLElement).querySelector('textarea')!;
    await expect(area.classList.contains('input')).toBe(true);
    await expect(area.value).toBe('eNrtfQl0VNeZ');
  }}>
  <Textarea class="h-27 w-80 p-2 font-mono text-11 break-all text-text-3" aria-label="Build code" readonly value="eNrtfQl0VNeZ" />
</Story>
