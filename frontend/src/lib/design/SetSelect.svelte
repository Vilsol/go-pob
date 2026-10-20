<script lang="ts">
  import Select from './Select.svelte';

  // A set picker whose last option opens the set manager instead of switching sets.
  type Props = {
    sets: string[];
    value: number;
    manage?: string;
    disabled?: boolean;
    title?: string;
    class?: string;
    onchange?: (i: number) => void;
    onmanage?: () => void;
  };
  let { sets, value, manage, disabled, title, class: extra, onchange, onmanage }: Props = $props();
</script>

<Select
  class={extra}
  {disabled}
  {title}
  options={[...sets.map((s, i) => ({ value: String(i), label: s })), ...(manage && onmanage ? [{ value: '-1', label: manage }] : [])]}
  value={String(value)}
  onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
    const v = Number(e.currentTarget.value);
    if (v === -1) {
      e.currentTarget.value = String(value);
      onmanage?.();
      return;
    }
    onchange?.(v);
  }} />
