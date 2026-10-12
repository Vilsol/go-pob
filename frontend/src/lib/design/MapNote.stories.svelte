<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import MapNote from './MapNote.svelte';

  const { Story } = defineMeta({ title: 'Spec/MapNote', component: MapNote });
</script>

<Story name="Over the tree" asChild>
  <div class="relative h-19 w-80 rounded-md border border-line bg-tree-ground">
    <MapNote>Scroll to zoom · drag to pan</MapNote>
    <MapNote class="bottom-1.5 left-1.5">97 / 123 passives</MapNote>
  </div>
</Story>
