import { describe, expect, it } from 'vitest';
import { showPowerRows, type PowerRow } from './tree';
import { formatNumSep, luaFormat } from '$lib/display/format_stat';

const fmt = (f: string, v: number) => formatNumSep(luaFormat(f, v));
const row = (ID: number, Power: number, extra: Partial<PowerRow> = {}): PowerRow => ({
  ID,
  Name: `n${ID}`,
  Type: 'Normal',
  Power,
  PathPower: Power / 2,
  PathDist: 2,
  Allocated: false,
  ...extra
});

// PowerReportListControl.lua:98-135, TreeTab.lua:1080-1089
describe('the power report as the list shows it', () => {
  const rows = [row(1, 5), row(2, 0), row(3, 1200), row(4, -3, { Allocated: true, PathDist: 4 }), row(5, 7, { Type: 'Mastery', PathDist: 1000 })];

  it('unallocated: rows with power, highest first, in the stat’s format', () => {
    const shown = showPowerRows(rows, { fmt: 'd' }, 'unallocated', true, fmt);
    expect(shown.map((r) => [r.ID, r.power, r.perPoint, r.points, r.dir])).toEqual([
      [3, '1,200', '600', '2', 'up'],
      [5, '7', '3', 'Anoint', 'up'],
      [1, '5', '2', '2', 'up']
    ]);
  });

  it('allocated: the allocated rows, lowest first; masteries can be hidden', () => {
    expect(showPowerRows(rows, { fmt: 'd' }, 'allocated', true, fmt).map((r) => r.ID)).toEqual([4]);
    expect(showPowerRows(rows, { fmt: 'd' }, 'unallocated', false, fmt).map((r) => r.ID)).toEqual([3, 1]);
  });

  it('a percent stat is x100, a stat without a display entry is .1f, lower-is-better flips the colour', () => {
    expect(showPowerRows([row(1, 0.25)], { fmt: '.1f', pc: true }, 'unallocated', true, fmt)[0].power).toBe('25.0');
    expect(showPowerRows([row(1, 2)], undefined, 'unallocated', true, fmt)[0].power).toBe('2.0');
    expect(showPowerRows([row(1, 2)], { fmt: 'd', lowerIsBetter: true }, 'unallocated', true, fmt)[0].dir).toBe('down');
  });
});

// PowerReportListControl.lua:30-43
describe('the power report filters and sorts as PoB’s list does', () => {
  const rows = [
    row(1, 5, { Type: 'Notable', PathDist: 3, PathPower: 2 }),
    row(2, 9, { Type: 'Normal', PathDist: 1, PathPower: 9 }),
    row(3, 9, { Type: 'Keystone', PathDist: 1000, PathPower: 9 }),
    row(4, 4, { Type: 'Notable', Cluster: true, PathDist: 0, PathPower: 0 }),
    row(5, 2, { Type: 'Normal', PathDist: 2, PathPower: 1 }),
    row(6, -3, { Allocated: true, PathDist: 2, PathPower: -1 }),
    row(7, -8, { Allocated: true, PathDist: 1, PathPower: -8 })
  ];
  const ids = (filter: Parameters<typeof showPowerRows>[2], sort?: Parameters<typeof showPowerRows>[5]) =>
    showPowerRows(rows, { fmt: 'd' }, filter, true, fmt, sort).map((r) => r.ID);

  it('"Show Unallocated" leaves the cluster notables out; "& Clusters" lists them with Cluster and --', () => {
    expect(ids('unallocated')).toEqual([2, 3, 1, 5]);
    expect(ids('clusters')).toEqual([2, 3, 1, 4, 5]);
    const cluster = showPowerRows(rows, { fmt: 'd' }, 'clusters', true, fmt).find((r) => r.ID === 4)!;
    expect([cluster.points, cluster.perPoint]).toEqual(['Cluster', '--']);
  });

  it('sorts by type, then power; by points (Anoint and Cluster last), then power; by per point, then points', () => {
    expect(ids('clusters', 'type')).toEqual([3, 2, 5, 1, 4]);
    expect(ids('clusters', 'points')).toEqual([2, 5, 1, 3, 4]);
    expect(ids('clusters', 'perPoint')).toEqual([2, 3, 1, 5, 4]);
    expect(ids('clusters', 'power')).toEqual([2, 3, 1, 4, 5]);
  });

  it('the allocated list sorts the other way (its numbers are negative)', () => {
    expect(ids('allocated')).toEqual([7, 6]);
    expect(ids('allocated', 'perPoint')).toEqual([7, 6]);
    expect(ids('allocated', 'points')).toEqual([7, 6]);
  });
});
