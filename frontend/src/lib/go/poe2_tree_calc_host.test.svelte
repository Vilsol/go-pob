<script lang="ts">
  import { Poe2TreeCalc } from './engine_poe2_tree_calc.svelte';

  // the model needs a component to own its effects, as the tree page owns it
  let { onmodel }: { onmodel: (m: Poe2TreeCalc) => void } = $props();
  onmodel(new Poe2TreeCalc());
</script>
