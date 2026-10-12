<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  type Props = HTMLAttributes<HTMLDivElement> & {
    text: string;
    k: 'ok' | 'bad' | 'comment' | 'empty';
    off?: boolean;
    hover?: boolean;
    edit?: { label: string; oninput?: (text: string) => void; oncommit?: (text: string) => void; field?: (el: HTMLInputElement) => void };
    title?: string;
    before?: Snippet;
    after?: Snippet;
  };
  let { text, k, off = false, hover = false, edit, title, before, after, ...rest }: Props = $props();

  const status = { ok: '✓', bad: '✗ no match', comment: '#', empty: '' };
  let field = $state<HTMLInputElement>();
  $effect(() => {
    if (field) edit?.field?.(field);
  });
</script>

<div class="mod-cell" data-k={k} data-off={off} data-hover={hover} {...rest}>
  {@render before?.()}
  {#if edit}
    <input
      bind:this={field}
      class="mod-cell-text min-w-0 border-0 bg-transparent p-0 font-mono text-12 outline-none"
      size={Math.max(12, text.length + 1)}
      spellcheck="false"
      aria-label={edit.label}
      value={text}
      oninput={(e) => edit.oninput?.(e.currentTarget.value)}
      onchange={(e) => edit.oncommit?.(e.currentTarget.value)}
      onkeydown={(e) => {
        if (e.key === 'Enter') edit.oncommit?.(e.currentTarget.value);
      }} />
  {:else}
    <span class="mod-cell-text">{text}</span>
  {/if}
  {#if status[k]}<span class="mod-cell-badge" {title}>{status[k]}</span>{/if}
  {@render after?.()}
</div>
