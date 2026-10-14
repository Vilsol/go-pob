import { describe, expect, it } from 'vitest';
import { nodeWhatIfLines } from './compare_node_tip';

const dps = { label: 'Total DPS', value: '+1,234.5', percent: '(+3.2%)', better: true };
const life = { label: 'Life', value: '-12', better: false };

// PassiveTreeView.lua:1695-1724, Build.lua:1889-1936
describe('nodeWhatIfLines', () => {
  it('writes each what-if under its header, coloured by direction, then the points to the node', () => {
    const lines = nodeWhatIfLines(
      [
        { header: '^7Allocating this node will give you:', sections: [{ lines: [dps, life] }] },
        { header: '^7Allocating this node and all nodes leading to it will give you:', sections: [{ lines: [{ ...dps, perPoint: '[+411.5 pp]' }] }] }
      ],
      3
    );
    expect(lines.map((l) => (l.sep ? '---' : l.text))).toEqual([
      '---',
      '^7Allocating this node will give you:',
      '^x33FF77+1,234.5 Total DPS (+3.2%)',
      '^xDD0022-12 Life',
      '^7Allocating this node and all nodes leading to it will give you:',
      '^x33FF77+1,234.5 Total DPS (+3.2%) ^8[+411.5 pp]',
      '^x80A080Tip: Press Ctrl+D to disable the display of stat differences.',
      '---',
      '^73 points to node',
      '^x80A080Tip: To reach this node by a different path, hold Shift, then trace the path and click this node'
    ]);
  });

  it('splits a minion build into Minion and Player under the header (Build.lua:1926-1933)', () => {
    const lines = nodeWhatIfLines(
      [
        {
          header: '^7Allocating this node will give you:',
          sections: [
            { actor: 'Minion', lines: [dps] },
            { actor: 'Player', lines: [life] }
          ]
        }
      ],
      1
    );
    expect(lines.map((l) => (l.sep ? '---' : l.text)).slice(1, 6)).toEqual([
      '^7Allocating this node will give you:',
      '^7Minion:',
      '^x33FF77+1,234.5 Total DPS (+3.2%)',
      '^7Player:',
      '^xDD0022-12 Life'
    ]);
  });

  it("says PoB's no-change line when nothing moves (PassiveTreeView.lua:1703)", () => {
    for (const n of [1, 2]) expect(nodeWhatIfLines([], n).map((l) => l.text)).toContain('^7No changes from allocating this node.');
  });

  it('shows only the Ctrl+D tip while the stat differences are off (PassiveTreeView.lua:1707-1710)', () => {
    expect(
      nodeWhatIfLines([{ header: '^7Allocating this node will give you:', sections: [{ lines: [dps] }] }], 2, false).map((l) => (l.sep ? '---' : l.text))
    ).toEqual([
      '---',
      '^x80A080Tip: Press Ctrl+D to enable the display of stat differences.',
      '---',
      '^72 points to node',
      '^x80A080Tip: To reach this node by a different path, hold Shift, then trace the path and click this node'
    ]);
  });
});
