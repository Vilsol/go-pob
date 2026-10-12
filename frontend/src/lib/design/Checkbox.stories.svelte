<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Checkbox from './Checkbox.svelte';
  import { expectDarkCheckboxes } from './stories/layout_checks';

  const { Story } = defineMeta({ title: 'Spec/Checkbox', component: Checkbox });
</script>

<Story name="Checked" asChild play={async ({ canvasElement }) => expectDarkCheckboxes(canvasElement as HTMLElement)}>
  <Checkbox checked>Enemy is a Boss</Checkbox>
</Story>

<Story name="Compact" asChild play={async ({ canvasElement }) => expectDarkCheckboxes(canvasElement as HTMLElement)}>
  <Checkbox compact>Tooltips only for affected slots</Checkbox>
</Story>

<Story name="States" asChild play={async ({ canvasElement }) => expectDarkCheckboxes(canvasElement as HTMLElement)}>
  <div class="flex flex-col gap-2">
    <Checkbox>Enemy is a Boss</Checkbox>
    <Checkbox checked>Enemy is a Boss</Checkbox>
    <Checkbox disabled>Enemy is a Boss</Checkbox>
    <Checkbox checked disabled>Enemy is a Boss</Checkbox>
  </div>
</Story>
