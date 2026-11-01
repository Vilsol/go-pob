import { isVariable, type AllVarTypes, type ConfigLabel, type ConfigSection } from '$lib/display/configurations';
import {
  configurations2,
  currentValue,
  isEligible2,
  isModified2,
  isVariable2,
  relevanceSets2,
  type Option2,
  type Variable2,
  type RelevanceSets2
} from '$lib/display/configurations2';
import type { calculator2 } from '$lib/types2';
import type { ConfigTable } from './config';

// PoB-PoE2's options table in the shape the Config page renders; the rules read the table's own rows
const toRow = (r: Option2): AllVarTypes | ConfigLabel =>
  isVariable2(r) && r.type === 'list'
    ? ({ ...r, list: (r.list ?? []).map((l) => ({ label: l.label, value: l.val })) } as unknown as AllVarTypes)
    : ({ ...r } as unknown as AllVarTypes | ConfigLabel);

// PoB-PoE2's table repeats a var name, so each row keeps its own source
const source = new Map<AllVarTypes | ConfigLabel, Option2>();
const sections: ConfigSection[] = configurations2.map((s) => {
  const rows = s.rows.map((r) => {
    const row = toRow(r);
    source.set(row, r);
    return row;
  });
  return { name: s.name, col: s.col, rows, variables: rows.filter(isVariable) };
});

const own = (row: AllVarTypes | ConfigLabel) => source.get(row) ?? (row as unknown as Option2);

export const poe2ConfigTable: ConfigTable<RelevanceSets2, calculator2.ConfigRelevance> = {
  sections,
  options: Object.fromEntries(sections.flatMap((s) => s.variables.map((v) => [v.var, v]))),
  relevanceSets: relevanceSets2,
  isRelevant: (row, inputs, rel) => isEligible2(own(row), inputs, rel),
  isModified: (v, inputs) => isModified2(own(v) as Variable2, inputs),
  // ConfigTab.lua:211-219 (PoB-PoE2)
  isShowAllExcluded: (row) => row.ifOption !== undefined || !!row.legacy,
  value: (v, inputs) => currentValue(own(v) as Variable2, inputs),
  isSet: (v, inputs) => currentValue(own(v) as Variable2, inputs) !== undefined,
  // ConfigTab.lua:376-390 (PoB-PoE2)
  tooltip: (v) => v.tooltip || undefined
};
