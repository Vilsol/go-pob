<script lang="ts">
  import ComparePage from '../ComparePage.svelte';
  import type { BuildFixture } from '../fixtures';
  import type { CompareView, MetricId } from '../compare';
  import type { CompareCalcSection } from '$lib/display/compare_calcs';
  import type { PowerRow } from '../fixtures';
  import { nodeWhatIfLines } from '$lib/shell/compare_node_tip';

  type Props = { fx: BuildFixture; view?: CompareView; swapped?: boolean; calcsB?: BuildFixture };
  let { fx, view = 'summary', swapped = false, calcsB }: Props = $props();

  const calcs = $derived.by((): CompareCalcSection[] | undefined => {
    if (!calcsB) return undefined;
    const bRow = (id: string, block: string, label: string) =>
      calcsB.calcs.sections
        .find((x) => x.id === id)
        ?.blocks?.find((b) => b.label === block)
        ?.rows.find((r) => r.label === label)?.cells[0]?.text ?? '';
    return fx.calcs.sections
      .filter((x) => x.applicable && !x.wide)
      .map((x) => ({
        id: x.id,
        color: x.color,
        blocks: (x.blocks ?? []).map((b) => ({
          label: b.label,
          rows: b.rows.filter((r) => r.label).map((r) => ({ label: r.label, a: r.cells[0]?.text ?? '', b: bRow(x.id, b.label, r.label) }))
        }))
      }));
  });

  let s = $state({ view, swapped, metric: 'dps' as MetricId });

  const METRIC_LABELS = ['Full DPS', 'Effective Hit Pool', 'Life'];
  const powertip = (row: PowerRow) =>
    Promise.resolve(
      row.cat !== 'tree'
        ? undefined
        : {
            influence: [],
            title: [row.what.replace(/^Take /, '')],
            lines: nodeWhatIfLines(
              [
                {
                  header: '^7Allocating this node and all nodes leading to it will give you:',
                  sections: [
                    {
                      lines: row.impact
                        .map((v, i) => ({ label: METRIC_LABELS[i] ?? '', value: `${v > 0 ? '+' : ''}${Math.round(v)}`, better: v > 0 }))
                        .filter((l) => l.value !== '0')
                    }
                  ]
                }
              ],
              row.points ?? 1
            )
          }
    );
</script>

<ComparePage
  compare={fx.compare}
  aSkills={fx.skills}
  swapped={s.swapped}
  bind:view={s.view}
  bind:metric={s.metric}
  {calcs}
  onswap={() => (s.swapped = !s.swapped)}
  {powertip}
  onapply={() => {}}
  onnode={() => {}}
  onuseconfig={() => {}}
  oncopyconfig={() => {}} />
