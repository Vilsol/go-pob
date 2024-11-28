<script lang="ts">
  let {
    prefix = undefined,
    min = undefined,
    max = undefined,
    value = $bindable(undefined),
    placeholder = '',
    fullWidth = false,
    id = null
  }: {
    prefix?: string | undefined;
    min?: number | undefined;
    max?: number | undefined;
    value?: number | undefined;
    placeholder?: string;
    fullWidth?: boolean;
    id?: string | null;
  } = $props();

  let inputElement = $state<HTMLInputElement>();

  const change = (n: number) => {
    if (value === undefined) {
      return;
    }

    value = (typeof value === 'string' ? parseInt(value) || 0 : value) + (n || min || 0);
    if (max !== undefined && value > max) {
      value = max;
    } else if (min !== undefined && value < min) {
      value = min;
    }
  };

  let plusDisabled = $derived(typeof value !== 'number' || (max !== undefined && value >= max));
  let minusDisabled = $derived(typeof value !== 'number' || (min !== undefined && value <= min));
</script>

<div class="flex flex-row" class:min-w-full={fullWidth}>
  <button class="input-wrapper flex flex-row items-center flex-1" onclick={() => inputElement?.focus()}>
    {#if prefix}
      <span class="mx-1 select-none">{prefix}</span>
    {/if}
    <input bind:this={inputElement} type="number" {min} {max} bind:value class="input w-full" {placeholder} {id} />
  </button>
  <button class="container font-bold" onclick={() => change(1)} disabled={plusDisabled}>&plus;</button>
  <button class="container font-bold" onclick={() => change(-1)} disabled={minusDisabled}>&minus;</button>
</div>
