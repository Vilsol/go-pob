<script lang="ts">
  type Props = { x: number; y: number; z: number; hot?: boolean };
  let { x, y, z, hot = false }: Props = $props();
</script>

<div class="radius-ring" data-hot={hot || undefined} style:--x="{x}px" style:--y="{y}px" style:--z="{z}px"></div>
