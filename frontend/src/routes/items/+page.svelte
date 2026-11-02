<script lang="ts">
  import GamePage from '$lib/shell/GamePage.svelte';
  import AppItems from '$lib/shell/AppItems.svelte';
  import AppItemsPoe2 from '$lib/shell/AppItemsPoe2.svelte';
  import { game } from '$lib/go/engine';
</script>

<GamePage route="items"
  >{#if $game === 'poe2'}<AppItemsPoe2 />{:else}<AppItems />{/if}</GamePage>
