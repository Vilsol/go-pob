<script lang="ts">
  interface Props {
    title: string;
    subtitle: string;
    value?: string;
    confirm: string;
    onconfirm: () => void;
    onclose: () => void;
  }

  let { title, subtitle, confirm, onconfirm, onclose }: Props = $props();

  const clickConfirm = () => {
    onconfirm();
    onclose();
  };
</script>

<div class="flex flex-col gap-2">
  <fieldset class="border border-white bg-neutral-900 p-2 mt-4 min-w-[15vw]">
    <legend class="container">{title}</legend>
    <div class="flex flex-row justify-center">
      <p>{subtitle}</p>
    </div>
  </fieldset>

  <div class="flex flex-row items-center justify-center gap-2">
    <button class="container" onclick={clickConfirm}>{confirm}</button>
    <button class="container" onclick={onclose}>Close</button>
  </div>
</div>
