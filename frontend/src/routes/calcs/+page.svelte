<script lang="ts">
  import { backendLoaded, outputs } from '$lib/global';
  import { type CalcDataCol, type CalcDataColProp, type CalcDataRow, type CalcSection, calcSections } from '$lib/calcs/calc_sections';
  import ColoredText from '$lib/components/common/ColoredText.svelte';
  import { FormatStr } from '$lib/calcs/formatting.js';
  import { onMount } from 'svelte';
  import { syncWrap } from '$lib/go/worker';

  const grouped = calcSections.reduce(
    (groups, section) => ({
      ...groups,
      [section[2]]: [...(groups[section[2]] || []), section]
    }),
    {} as Record<number, CalcSection[]>
  );

  const elementsOf = <M,>(obj: { [key: number]: M }): M[] =>
    Object.entries(obj)
      .filter(([k]) => !isNaN(parseInt(k)))
      .map(([, v]) => v);

  const getFormat = (row?: CalcDataRow): string => {
    if (!row) {
      return '';
    }

    if (row.format) {
      return row.format;
    }

    const sub = elementsOf(row).find((e) => e.format);
    if (sub && sub.format) {
      return sub.format;
    }

    return '';
  };

  const widest = (rows: CalcDataRow[]) => {
    let w = 0;
    rows.forEach((row) => {
      w = Math.max(w, elementsOf(row || {}).length);
    });
    return w;
  };

  let tooltipStyle = $state('');
  let tooltipElement = $state<HTMLElement>();

  let hoveredItem: CalcDataCol | undefined = $state(undefined);
  let hoveredBreakdown: string | undefined = $state(undefined);

  const moveEvent = (event: MouseEvent) => {
    if (!hoveredItem) {
      tooltipStyle = 'display: none';
      return;
    }

    const breakdown: CalcDataColProp | undefined = elementsOf(hoveredItem).find((e) => e.breakdown);
    if (!breakdown) {
      tooltipStyle = 'display: none';
      return;
    }

    hoveredBreakdown = breakdown.breakdown;

    let left = event.x;
    if (event.x > window.innerWidth / 2) {
      left -= (tooltipElement?.clientWidth || 200) + 15;
    } else {
      left += 15;
    }

    let top = event.y;
    if (event.y > window.innerHeight / 2) {
      top -= (tooltipElement?.clientHeight || 100) + 15;
    }

    tooltipStyle = `top: ${top}px; left: ${left}px; border-color: #00CB3A`;
  };

  onMount(() => {
    void backendLoaded.then(() => {
      const dbElements: Record<string, CalcDataColProp> = {};
      for (const section of calcSections) {
        for (const [j, block] of section[4].entries()) {
          for (const [k, row] of elementsOf(block.data).entries()) {
            for (const [l, col] of elementsOf(row).entries()) {
              for (const [m, prop] of elementsOf(col).entries()) {
                const id = `${section[1]}:${j}:${k}:${l}:${m}`;
                if (prop.modName && prop.modType && prop.cfg) {
                  dbElements[id] = prop;
                }
              }
            }
          }
        }
      }

      void syncWrap?.setCalcTabElements(dbElements);
    });
  });
</script>

<svelte:window onmousemove={moveEvent} />

<div class="p-2 px-4 h-full flex flex-col flex-wrap gap-4 w-full overflow-x-auto">
  <div class="flex flex-row flex-wrap gap-2">
    {#each Object.values(grouped) as group}
      <div class="flex flex-col flex-wrap gap-2">
        {#each group as section}
          <div class="flex flex-col border-2 w-full" style="border-color: {section[3]}">
            {#each section[4] as block, j}
              {@const width = widest(elementsOf(block.data))}
              <div class="head" style="border-color: {section[3]}">
                <ColoredText text={FormatStr(block.label || '', $outputs)} />:
                {#if block.data.extra}
                  <ColoredText text={FormatStr(block.data.extra || '', $outputs)} />
                {/if}
              </div>
              <div class="w-full">
                <table class="w-full" style="border-collapse: collapse;">
                  <tbody>
                    {#each elementsOf(block.data) as row, k}
                      {@const columns = elementsOf(row)}
                      <tr>
                        <td class="col w-fit text-right" style="border-color: {section[3]}"><ColoredText text={FormatStr(row.label || '', $outputs)} /></td>
                        {#each columns as column, l}
                          <td
                            class="col"
                            style="border-color: {section[3]}"
                            onmouseover={() => (hoveredItem = column)}
                            onfocus={() => (hoveredItem = column)}
                            onmouseleave={() => (hoveredItem = undefined)}>
                            <pre><ColoredText text={FormatStr(getFormat(column), $outputs, column, `${section[1]}:${j}:${k}:${l}`)} /></pre>
                          </td>
                        {/each}
                        {#each Array(width - columns.length).fill(0) as ignored}
                          <td class="col" style="border-color: {section[3]}" data-ignored={ignored}></td>
                        {/each}
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/each}
          </div>
        {/each}
      </div>
    {/each}
  </div>
</div>

<div class="absolute pointer-events-none border-4 bg-black" style={tooltipStyle} bind:this={tooltipElement}>
  {#if hoveredItem !== undefined && hoveredBreakdown !== undefined}
    {@const hoveredData = $outputs?.Breakdown?.[hoveredBreakdown]}
    {#if hoveredData}
      <div class="flex flex-col gap-2">
        {#if (hoveredData.Lines?.length || 0) > 0}
          <div class="p-2 flex flex-col gap-1">
            {#each hoveredData.Lines || [] as line}
              <ColoredText text={line} />
            {/each}
          </div>
        {/if}
        {#if (hoveredData.Columns?.length || 0) > 0}
          <table class="hover-table">
            <thead>
              <tr>
                {#each hoveredData.Columns || [] as col}
                  <th>{col.Label}</th>
                {/each}
              </tr>
            </thead>
            <tbody>
              {#each hoveredData.Rows || [] as row}
                {#if row}
                  <tr>
                    {#each hoveredData.Columns || [] as col}
                      <td><ColoredText text={row?.[col.Key]} /></td>
                    {/each}
                  </tr>
                {/if}
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
    {:else}
      <span>N/A</span>
    {/if}
  {/if}
</div>

<style lang="postcss">
  .head {
    @apply border-b-2 p-1;

    &:not(:first-child) {
      @apply border-t-2;
    }
  }
  .col {
    @apply text-sm p-1;

    &:hover {
      box-shadow: inset 0px 0px 0px 2px #00cb3a;
    }

    &:not(:last-child) {
      @apply border-r-2;
    }
  }

  .hover-table {
    & td,
    & th {
      @apply p-1;
    }

    & tr:not(:last-child),
    & thead tr {
      @apply border-b;
    }
    & td:not(:last-child),
    & th:not(:last-child) {
      @apply border-r;
    }
  }
</style>
