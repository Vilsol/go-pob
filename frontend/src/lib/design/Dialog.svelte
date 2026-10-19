<script lang="ts">
  import type { Snippet } from 'svelte';

  const DEFAULT_CLASS = 'w-95 p-4';
  type Props = {
    open?: boolean;
    title: string;
    sheet?: boolean;
    note?: string;
    class?: string;
    onclose?: () => void;
    children: Snippet;
    actions?: Snippet;
    head?: Snippet;
    backdrop?: boolean;
    /* a short confirm or prompt: stays a centred dialog on a phone instead of filling the screen */
    small?: boolean;
  };
  let {
    open = $bindable(false),
    title,
    sheet = false,
    note,
    class: extra = sheet ? 'w-100' : DEFAULT_CLASS,
    onclose,
    children,
    actions,
    head,
    backdrop = false,
    small
  }: Props = $props();
  const isSmall = $derived(small ?? (!sheet && extra === DEFAULT_CLASS));

  const uid = $props.id();
  let el: HTMLDialogElement;
  let opener: HTMLElement | null = null;

  $effect(() => {
    if (open && !el.open) {
      opener = document.activeElement as HTMLElement | null;
      el.showModal();
    } else if (!open && el.open) el.close();
  });

  function wrapTab(e: KeyboardEvent) {
    const all = [...el.querySelectorAll<HTMLElement>('button, a[href], input, select, textarea, [tabindex="0"]')];
    const f = all.filter((x) => !x.matches(':disabled') && x.offsetParent !== null && !x.closest('dialog:not([open])'));
    if (!f.length) return;
    const i = f.indexOf(document.activeElement as HTMLElement);
    if (e.shiftKey && i <= 0) {
      e.preventDefault();
      f[f.length - 1].focus();
    } else if (!e.shiftKey && i === f.length - 1) {
      e.preventDefault();
      f[0].focus();
    }
  }

  function clicked(e: MouseEvent) {
    if (!backdrop || e.target !== el) return;
    const r = el.getBoundingClientRect();
    if (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom) open = false;
  }

  function closed() {
    if (el.open) return;
    open = false;
    opener?.focus();
    onclose?.();
  }
</script>

<dialog
  bind:this={el}
  aria-labelledby="{uid}-title"
  class={['dialog m-auto text-text not-open:hidden backdrop:bg-[rgb(8_9_10/0.72)]', sheet && 'dialog-sheet', isSmall && 'dialog-small', extra]}
  onclose={closed}
  onclick={clicked}
  onkeydown={(e) => {
    if (e.key === 'Tab') {
      e.stopPropagation();
      wrapTab(e);
    }
    if (e.key !== 'Escape') return;
    e.preventDefault();
    e.stopPropagation();
    open = false;
  }}>
  {#if sheet}
    <div class="sheet-head">
      <h3 id="{uid}-title" class="text-16 font-semibold">{title}</h3>
      {#if note}<span class="note">{note}</span>{/if}
      {@render head?.()}
      <button class="btn-close ml-auto" aria-label="Close" onclick={() => (open = false)}>×</button>
    </div>
    {@render children()}
  {:else}
    <div class="flex items-center justify-between">
      <h3 id="{uid}-title" class="text-15">{title}</h3>
      <button class="btn-close" aria-label="Close" onclick={() => (open = false)}>×</button>
    </div>
    {@render children()}
    {#if actions}<div class="flex justify-end gap-2">{@render actions()}</div>{/if}
  {/if}
</dialog>
