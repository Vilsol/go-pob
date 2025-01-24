import type { Outputs } from '$lib/custom_types';
import type { CalcDataCol } from '$lib/calcs/calc_sections';

const formatNumber = (numValue: number | string): string => {
  if (typeof numValue === 'string') {
    return '';
  }

  if (isNaN(numValue)) {
    return 'ERROR: ' + numValue.toString();
  }

  const roundedNum = Number(numValue.toFixed(2));
  const [integerPart, decimalPart] = roundedNum.toString().split('.');
  const formattedInteger = integerPart.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return decimalPart ? `${formattedInteger}.${decimalPart}` : formattedInteger;
};

const round = (value: number, decimalPlaces: number): number => {
  const multiplier = Math.pow(10, decimalPlaces);
  return Math.round(value * multiplier) / multiplier;
};

export const FormatStr = (str: string, outputs?: Outputs, column?: CalcDataCol, id?: string): string => {
  str = str.replace(/\{output:([a-zA-Z.:]+)\}/g, (_, c: string) => {
    const match = c.match(/^([a-zA-Z]+)\.([a-zA-Z]+)$/);
    if (match) {
      const [, ns, var_] = match;
      return (outputs?.OutputTable?.[ns]?.[var_] ?? '').toString();
    }
    return (outputs?.Output?.[c] ?? '').toString();
  });

  str = str.replace(/\{(\d+):output:([a-zA-Z.:]+)\}/g, (_, p: string, c: string) => {
    const match = c.match(/^([a-zA-Z]+)\.([a-zA-Z]+)$/);
    const pNum = parseInt(p);
    if (match) {
      const [, ns, var_] = match;
      return formatNumber(round(outputs?.OutputTable?.[ns]?.[var_] ?? 0, pNum));
    }
    return formatNumber(round(outputs?.Output?.[c] ?? 0, pNum));
  });

  str = str.replace(/\{(\d+):mod:([\d,]+)\}/g, (_, p: string, c: string) => {
    const match = c.match(/(\d+)/g);
    if (!match) {
      return 'ERR: ' + str;
    }

    const modType = column?.[parseInt(match[0]) - 1].modType;
    let modTotal = modType === 'MORE' ? 1 : 0;
    for (const n of [...match]) {
      const propNum = parseInt(n) - 1;
      const fullID = `${id}:${propNum}`;
      const value = outputs?.Calcs?.[fullID] || 0;
      if (modType === 'MORE') {
        modTotal = modTotal * value;
      } else {
        modTotal = modTotal + value;
      }
    }

    if (modType === 'MORE') {
      modTotal = (modTotal - 1) * 100;
    }

    return formatNumber(round(modTotal, parseInt(p)));
  });

  return str;
};
