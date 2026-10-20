<script lang="ts">
  import PobText from './PobText.svelte';
  import TooltipLines from './TooltipLines.svelte';
  import type { NodeTip, NodeTipCost } from './tree';

  type Props = { tip?: NodeTip; cost?: NodeTipCost };
  let { tip, cost }: Props = $props();
</script>

{#if tip}
  {#if tip.pick}<div class="text-12 text-accent" data-power-pick={tip.pick.index}>Power report picks: {tip.pick.text}</div>{/if}
  {#if tip.comparedMastery}
    <!-- PassiveTreeView.lua:1550-1576 -->
    <div class="flex flex-col gap-0.5 text-12" aria-label="Compared mastery">
      {#each tip.comparedMastery.remove as l (l)}<span class="text-code-remove">&lt;- {l}</span>{/each}
      {#each tip.comparedMastery.add as l (l)}<span class="text-code-add">-&gt; {l}</span>{/each}
    </div>
  {/if}
  {#if tip.jewel}<div class="border-t border-line pt-2" aria-label="Socketed jewel"><TooltipLines lines={tip.jewel} /></div>{/if}
  <!-- PassiveTreeView.lua:1490-1497 -->
  {#if tip.comparedEmptySocket}
    <div class="border-t border-line pt-2 text-12" aria-label="Compared socket"><span class="text-code-orange">Compared build:</span> (empty socket)</div>
  {/if}
  {#if tip.distance}<div class="border-t border-line pt-2 text-12" aria-label="Distance to start">Distance to start: {tip.distance}</div>{/if}
  {#if tip.affecting?.length}
    <div class="flex flex-col gap-1.5 border-t border-line pt-2 text-12" aria-label="Jewels affecting this node">
      {#each tip.affecting as a (a.socket)}
        <div class="flex flex-col gap-0.5" data-affecting-socket={a.socket}>
          <span class="text-text-2">In radius of <PobText text={a.title} /></span>
          {#if a.lines.length}<div class="pl-2"><TooltipLines lines={a.lines} /></div>{/if}
        </div>
      {/each}
    </div>
  {/if}
  {#each tip.skills ?? [] as lines, i (i)}
    <div class="border-t border-line pt-2" aria-label="Granted skill"><TooltipLines {lines} center /></div>
  {/each}
  {#if tip.reminder?.length}
    <div class="flex flex-col gap-0.5 border-t border-line pt-2 text-12 text-code-reminder" aria-label="Reminder text">
      {#each tip.reminder as l, i (i)}<span>{l}</span>{/each}
    </div>
  {/if}
  {#if tip.flavour?.length}
    <div class="flex flex-col gap-0.5 border-t border-line pt-2 italic text-unique" aria-label="Flavour text">
      {#each tip.flavour as l, i (i)}<span data-pob-color>{l}</span>{/each}
    </div>
  {/if}
  {#if tip.tattooKind}
    <!-- PassiveTreeView.lua:1646-1656 -->
    <div class="border-t border-line pt-2 text-12 text-code-tip" aria-label="Tattoo tip">Tip: Right click to edit the {tip.tattooKind} for this node</div>
  {/if}
  {#if tip.compareOnly}
    <!-- PassiveTreeView.lua:1804-1805 -->
    <div class="border-t border-line pt-2 text-12 text-code-positive" aria-label="Compared build">This node is only in the compared build</div>
  {/if}
{/if}
{#if cost}
  <div class="flex flex-col gap-0.5 border-t border-line pt-2 text-12" aria-label="Points and gold">
    {#if cost.path > 0 && cost.traced}
      <span>{cost.traced} nodes in trace path</span>
    {:else if cost.path > 0}
      <span
        >{cost.path} points to node{#if cost.leap}<span class="text-text-3"> (Can be allocated without pathing to it)</span>{/if}</span>
      {#if cost.path > 1 && !cost.leap}<span class="text-text-3"
          >Tip: To reach this node by a different path, hold Shift, then trace the path and click this node</span
        >{/if}
    {/if}
    {#if cost.depends > 1}
      <span>{cost.depends} points gained from unallocating these nodes</span>
      <span><span class="text-code-gold">{cost.dependsGold} Gold</span> required to unallocate these nodes</span>
    {:else if cost.depends === 1}
      <span><span class="text-code-gold">{cost.gold} Gold</span> required to unallocate this node</span>
    {/if}
    {#if !cost.socket}<span class="text-text-3">Tip: Press Ctrl+C to copy this node's text.</span>
    {:else if cost.allocated}
      <!-- PassiveTreeView.lua:1499-1501 -->
      <span class="text-text-3">Tip: Right click this socket to choose the jewel for this socket.</span>{/if}
  </div>
{/if}
