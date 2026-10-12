<script lang="ts">
  type Props = { x: number; y: number; w: number; r?: number; state?: 'on' | 'path' | 'lose' };
  let { x, y, w, r = 0, state }: Props = $props();
</script>

<div class="tree-edge" data-state={state} style:--x="{x}px" style:--y="{y}px" style:--w="{w}px" style:--r="{r}deg"></div>
