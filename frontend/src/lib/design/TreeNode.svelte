<script lang="ts">
  type Props = {
    kind: 's' | 'n' | 'k' | 'm' | 'j' | 'c';
    x: number;
    y: number;
    z: number;
    label?: string;
    alloc?: boolean;
    state?: 'path' | 'lose';
    hover?: boolean;
    match?: boolean;
    conq?: boolean;
    radius?: boolean;
  };
  let { kind, x, y, z, label, alloc = false, state, hover = false, match = false, conq = false, radius = false }: Props = $props();
</script>

<span
  class="tree-node"
  data-kind={kind}
  data-alloc={alloc}
  data-state={state}
  data-hover={hover}
  data-match={match}
  data-conq={conq}
  data-radius={radius || undefined}
  title={label}
  style:--x="{x}px"
  style:--y="{y}px"
  style:--z="{z}px"></span>
