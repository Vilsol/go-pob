<script lang="ts">
  interface Props {
    errors: unknown;
    data: unknown;
  }

  let { errors, data }: Props = $props();

  console.error(errors, data);
</script>

<h1 class="text-3xl font-bold">How did you get here?</h1>
