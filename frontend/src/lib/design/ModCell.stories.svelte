<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import ModCell from './ModCell.svelte';

  const { Story } = defineMeta({
    title: 'Spec/ModCell',
    component: ModCell,
    argTypes: { k: { control: 'inline-radio', options: ['ok', 'bad', 'comment', 'empty'] } },
    args: { text: '80% increased global Accuracy Rating', k: 'ok' }
  });
</script>

<Story name="Parsed" />

<Story name="States" asChild>
  <div class="flex flex-wrap items-center gap-1.5">
    <ModCell text="80% increased global Accuracy Rating" k="ok" hover />
    <ModCell text="Crit Immune" k="bad" />
    <ModCell text="# boss uptime assumptions" k="comment" />
    <ModCell text="Nearby Enemies take 10% increased Physical Damage" k="ok" off />
  </div>
</Story>
