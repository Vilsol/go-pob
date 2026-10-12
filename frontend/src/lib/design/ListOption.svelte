<script lang="ts">
  type Props = {
    label: string;
    count?: number | string;
    selected?: boolean;
    enabled?: boolean;
    onselect?: () => void;
    ontoggle?: (enabled: boolean) => void;
  };
  let { label, count, selected = false, enabled = $bindable(true), onselect, ontoggle }: Props = $props();
</script>

<div class="list-option">
  <input type="checkbox" class="check size-3.75" bind:checked={enabled} aria-label={`Enable ${label}`} onchange={() => ontoggle?.(enabled)} />
  <button class={['list-option-btn', !enabled && 'text-text-2 line-through']} aria-current={selected || undefined} onclick={onselect}>{label}</button>
  {#if count != null}<span class="font-mono text-11 text-text-3">{count}</span>{/if}
</div>
