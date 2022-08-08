<script lang='ts'>
  export let prefix: string | undefined = undefined;
  export let min: number | undefined = undefined;
  export let max: number | undefined = undefined;
  export let value: number | undefined = undefined;
  export let placeholder = '';

  let inputElement: HTMLInputElement;

  const change = (n: number) => {
    if (!value) {
      return;
    }

    value = value + n || min || 0;
  }

  $: plusDisabled = (typeof value === 'undefined' || (max && value >= max)) as boolean;
  $: minusDisabled = (typeof value === 'undefined' || (min && value <= min)) as boolean;
</script>

<div class='flex flex-row'>
  <div class='input-wrapper flex flex-row items-center' on:click={() => inputElement.focus()}>
    {#if prefix}
      <span class='mx-1 select-none'>{prefix}</span>
    {/if}
    <input bind:this={inputElement} type='number' {min} {max} bind:value class='input' {placeholder} />
  </div>
  <button class='container font-bold' on:click={() => change(1)} disabled={plusDisabled}>
    &plus;
  </button>
  <button class='container font-bold' on:click={() => change(-1)} disabled={minusDisabled}>
    &minus;
  </button>
</div>
