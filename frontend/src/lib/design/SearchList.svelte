<script lang="ts" generics="T">
  import type { Snippet } from 'svelte';
  import type { ClassValue } from 'svelte/elements';
  import EmptyState from './EmptyState.svelte';
  import Listbox from './Listbox.svelte';
  import SearchField from './SearchField.svelte';

  type Props = {
    query: string;
    searchLabel: string;
    placeholder?: string;
    /** controls beside the search field (a search-in select) */
    filters?: Snippet;
    items: T[];
    key: (t: T, i: number) => string | number;
    label: string;
    selected: number;
    busy?: boolean;
    /** replaces the options while busy (a progress note) */
    busyNote?: Snippet;
    emptyText: string;
    listClass?: ClassValue;
    row: Snippet<[T, number]>;
    onselect: (t: T, i: number) => void;
    onactivate?: (t: T, i: number) => void;
    onhover?: (i: number) => void;
  };
  let {
    query = $bindable(),
    searchLabel,
    placeholder = 'Search',
    filters,
    items,
    busy = false,
    busyNote,
    emptyText,
    listClass = '',
    ...list
  }: Props = $props();
</script>

<div class="flex gap-1">
  <SearchField size="sm" class="grow" {placeholder} label={searchLabel} bind:value={query} />
  {@render filters?.()}
</div>
<Listbox {...list} items={busy ? [] : items} {busy} class={listClass}>
  {#snippet empty()}
    {#if busy}{@render busyNote?.()}{:else}<EmptyState class="px-2 py-0.5" text={emptyText} />{/if}
  {/snippet}
</Listbox>
