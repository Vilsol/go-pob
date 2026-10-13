export type BreakdownStep = { op?: string; value: string; note?: string } | { prose: string };

const STEP = /^(?:([x×*/+=-])\s+)?(-?[\d.,]+(?:%|m|s|x)?(?:\s+to\s+-?[\d.,]+)?)(?:\s+(.*))?$/;
const OPS: Record<string, string> = { x: '×', '*': '×', '/': '÷', '-': '−' };

export const breakdownStep = (line: string): BreakdownStep => {
  const m = STEP.exec(line.trim());
  if (!m) return { prose: line };
  const [, op, value, note] = m;
  return { ...(op && { op: OPS[op] ?? op }), value, ...(note && { note }) };
};
