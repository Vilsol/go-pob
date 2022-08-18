<script lang="ts">
  export let prefix: string | undefined = undefined;
  export let min: number | undefined = undefined;
  export let max: number | undefined = undefined;
  export let value: number | undefined = undefined;
  export let placeholder = '';
  export let fullWidth = false;
  export let id: string | null = null;

  let inputElement: HTMLInputElement;

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

  $: plusDisabled = (typeof value !== 'number' || (max !== undefined && value >= max)) as boolean;
  $: minusDisabled = (typeof value !== 'number' || (min !== undefined && value <= min)) as boolean;
</script>

<div class="flex flex-row" class:min-w-full={fullWidth}>
  <div class="input-wrapper flex flex-row items-center flex-1" on:click={() => inputElement.focus()}>
    {#if prefix}
      <span class="mx-1 select-none">{prefix}</span>
    {/if}
    <input bind:this={inputElement} type="number" {min} {max} bind:value class="input w-full" {placeholder} {id} />
  </div>
  <button class="container font-bold" on:click={() => change(1)} disabled={plusDisabled}>&plus;</button>
  <button class="container font-bold" on:click={() => change(-1)} disabled={minusDisabled}>&minus;</button>
</div>
