<script lang="ts">
  import { poe2TreeModel } from './engine_poe2_tree.svelte';

  // the model needs a component to own its effects, as the tree page owns it
  let { onmodel }: { onmodel: (m: ReturnType<typeof poe2TreeModel>) => void } = $props();
  onmodel(poe2TreeModel());
</script>
