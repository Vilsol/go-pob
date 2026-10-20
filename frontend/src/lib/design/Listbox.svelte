<script lang="ts" generics="T">
  import type { Snippet } from 'svelte';
  import type { ClassValue } from 'svelte/elements';
  import { rovingIndex } from './keys';

  type Props = {
    items: T[];
    key: (t: T, i: number) => string | number;
    label: string;
    /** index of the selected item, -1 for none; the arrow keys move it */
    selected: number;
    busy?: boolean;
    class?: ClassValue;
    optionClass?: ClassValue;
    row: Snippet<[T, number]>;
    /** shown in place of the options when there are none */
    empty?: Snippet;
    onselect: (t: T, i: number) => void;
    /** double click or Enter */
    onactivate?: (t: T, i: number) => void;
    /** the hovered index, -1 when the pointer leaves an option */
    onhover?: (i: number) => void;
  };
  let { items, key, label, selected, busy = false, class: extra = '', optionClass = '', row, empty, onselect, onactivate, onhover }: Props = $props();
  const uid = $props.id();
  let box = $state<HTMLElement>();

  const pick = (i: number) => {
    onselect(items[i], i);
    box?.querySelector(`#${CSS.escape(`${uid}-${i}`)}`)?.scrollIntoView?.({ block: 'nearest' });
  };
  const onkeydown = (e: KeyboardEvent) => {
    if (e.key === 'Enter' && selected >= 0 && selected < items.length) {
      e.preventDefault();
      onactivate?.(items[selected], selected);
      return;
    }
    const to = rovingIndex(e, selected, items.length, { orientation: 'vertical', wrap: false });
    if (to === undefined) return;
    e.preventDefault();
    pick(to);
  };
</script>

<div class={['scroll flex flex-col border border-line', extra]}>
  <div
    bind:this={box}
    role="listbox"
    aria-label={label}
    aria-busy={busy}
    tabindex="0"
    aria-activedescendant={selected >= 0 && selected < items.length ? `${uid}-${selected}` : undefined}
    class="flex flex-col focus-visible:-outline-offset-2"
    {onkeydown}>
    {#each items as t, i (key(t, i))}
      <!-- the listbox takes the keys (aria-activedescendant) -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div
        id="{uid}-{i}"
        role="option"
        tabindex="-1"
        aria-selected={i === selected}
        class={['cursor-default px-2 py-0.5 text-left font-sans text-13 text-text aria-selected:bg-row-selected', optionClass]}
        onclick={() => onselect(t, i)}
        ondblclick={() => onactivate?.(t, i)}
        onmouseenter={() => onhover?.(i)}
        onmouseleave={() => onhover?.(-1)}>
        {@render row(t, i)}
      </div>
    {/each}
  </div>
  {#if !items.length}{@render empty?.()}{/if}
</div>
