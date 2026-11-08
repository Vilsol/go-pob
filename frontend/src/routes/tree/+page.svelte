<script lang="ts">
  import GamePage from '$lib/shell/GamePage.svelte';
  import AppTree from '$lib/shell/AppTree.svelte';
  import AppTree2 from '$lib/shell/AppTree2.svelte';
  import { game } from '$lib/go/engine';
</script>

<GamePage route="tree"
  >{#if $game === 'poe2'}<AppTree2 />{:else}<AppTree />{/if}</GamePage>
