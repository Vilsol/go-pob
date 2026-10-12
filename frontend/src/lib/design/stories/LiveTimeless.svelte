<script lang="ts">
  import TimelessPage from '../TimelessPage.svelte';
  import type { BuildFixture } from '../fixtures';
  import { timelessSample } from '../fixtures/app';
  import { socketLabel, type TimelessForm, type TimelessRow, type TimelessStat } from '../timeless';

  type Props = { fx: BuildFixture; step?: number; jewel?: number };
  let { fx, step = 1, jewel = 2 }: Props = $props();

  const jewels = timelessSample.jewels.map((j) => ({ ...j, conquerors: ['Any', ...j.conquerors] }));
  const id = (label: string) => label.replace(/[^a-z]+/gi, '_').toLowerCase();
  const stats: TimelessStat[] = timelessSample.stats.map((label) => ({ id: id(label), label, descriptions: [] }));
  const rows = new Map<number, TimelessRow>(
    timelessSample.results.map(([seed, i, weight], n) => {
      const socket = fx.tree.sockets[i % fx.tree.sockets.length];
      return [
        n,
        {
          label: `${socketLabel(socket)} | ${seed}:`,
          seed,
          total: weight,
          socket: socket.id,
          nodes:
            n === 0
              ? [{ name: 'Flesh of the Kaom', targets: ['Herbalism'] }]
              : timelessSample.requirements.map(([name], j) => ({ name, targets: socket.notables.slice(j, j + ((seed + j) % 4)) }))
        }
      ];
    })
  );
  let form = $state<TimelessForm>({
    jewelType: jewel,
    conqueror: 2,
    devotion1: 1,
    devotion2: 1,
    socket: fx.tree.sockets[0].id,
    filter: false,
    distance: 0,
    protect: [],
    searchList: timelessSample.requirements.map(([label, w, min]) => `${id(label)}, ${Math.min(w, 10)}, 0, ${min * 10}`).join('\n'),
    searchListFallback: '',
    fallbackMode: 1,
    totalMinimum: '',
    realm: 'PC',
    league: 'Standard',
    tradeType: 1,
    searchMore: false
  });
  let s = $state({ step, sel: 1, highlight: 0, added: new Set<number>(), log: '' });
</script>

<TimelessPage
  tree={fx.tree}
  sockets={fx.tree.sockets}
  {jewels}
  devotion={timelessSample.devotion}
  {stats}
  fallbackModes={['Sort by Full DPS', 'Sort by Effective Hit Pool', 'Sort by Life']}
  realms={['PC', 'Sony', 'Xbox']}
  leagues={['Standard', 'Hardcore']}
  tradeTypes={['Instant buyout', 'Any (includes offline)']}
  bind:form
  bind:step={s.step}
  count={rows.size}
  {rows}
  bind:sel={s.sel}
  bind:highlight={s.highlight}
  added={s.added}
  onsearch={() => (s.log = 'searched')}
  ongenerate={() => (s.log = 'generated')}
  onadd={(i: number) => {
    s.added = new Set([...s.added, i]);
    s.log = `added ${rows.get(i)?.seed}`;
  }}
  ontrade={() => (s.log = 'trade')}
  onclose={() => (s.log = 'closed')} />
<span class="sr-only" data-log>{s.log}</span>
