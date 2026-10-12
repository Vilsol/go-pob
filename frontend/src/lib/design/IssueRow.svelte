<script lang="ts">
  type Props = { text: string; kind: string; where: string };
  let { text, kind, where }: Props = $props();
</script>

<div class="issue-row">
  <span aria-hidden="true" class="issue-badge">⚠</span>
  <span class="flex min-w-0 grow flex-col gap-0.75"><span class="text-13">{text}</span><span class="note">{kind} · {where}</span></span>
</div>
