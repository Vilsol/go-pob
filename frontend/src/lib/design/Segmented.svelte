<script lang="ts">
  type Option = { id: string; label: string; count?: string; class?: string };
  type Props = { options: Option[]; value?: string; label: string; chip?: boolean; class?: string; itemClass?: string; onchange?: (id: string) => void };
  let { options, value = $bindable(options[0]?.id), label, chip = false, class: extra = '', itemClass = '', onchange }: Props = $props();

  const pick = (id: string) => {
    value = id;
    onchange?.(id);
  };
</script>

<div role="group" aria-label={label} class={[chip ? 'flex gap-1' : 'seg', extra]}>
  {#each options as o (o.id)}
    <button class={[chip ? 'chip' : 'seg-btn', itemClass, o.class]} aria-pressed={o.id === value} onclick={() => pick(o.id)}
      >{o.label}{#if o.count}<span class="ml-1 font-mono text-11 text-text-2">{o.count}</span>{/if}</button>
  {/each}
</div>
