<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { topbar: Snippet; sidebar: Snippet; children: Snippet; label?: string; mainClass?: string };
  let { topbar, sidebar, children, label, mainClass = '' }: Props = $props();
</script>

<div class="page">
  {@render topbar()}
  <div class="page-body">
    {@render sidebar()}
    <main class={['page-main', mainClass]} aria-label={label}>{@render children()}</main>
  </div>
</div>
