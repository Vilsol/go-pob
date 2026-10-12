// CalcBreakdownControl.lua:626-641

export const RADIUS_BOX = { width: 480, height: 270 };

const cos45 = Math.cos(Math.PI / 4);
const cos35 = Math.cos(Math.PI * 0.195);
const sin35 = Math.sin(Math.PI * 0.195);
const round = (v: number) => Math.floor(v + 0.5);

/* Main.lua:1490-1499 */
export const worldToScreen = (x: number, y: number, z: number, width: number, height: number): [number, number] => {
  const cx = (x - y) * cos45;
  const cy = -5.33 - (y + x) * cos45 * cos35 - z * sin35;
  const cz = 122 + (y + x) * cos45 * sin35 - z * cos35;
  return [round(width * 0.5 + (cx / cz) * 1.27 * height), round(height * 0.5 + (cy / cz) * 1.27 * height)];
};

/* Main.lua:1502-1523 */
export const circleRows = (radius: number, width = RADIUS_BOX.width, height = RADIUS_BOX.height): [number, number, number][] => {
  const minX = new Map<number, number>();
  const maxX = new Map<number, number>();
  let minY = height;
  let maxY = 0;
  for (let d = 0; d <= 360; d += 0.15) {
    const r = (d / 180) * Math.PI;
    const [sx, py] = worldToScreen(Math.sin(r) * radius, Math.cos(r) * radius, 0, width, height);
    if (py >= 0 && py < height) {
      const px = Math.min(width, Math.max(0, sx));
      minY = Math.min(minY, py);
      maxY = Math.max(maxY, py);
      minX.set(py, Math.min(minX.get(py) ?? px, px));
      maxX.set(py, Math.max(maxX.get(py) ?? px, px));
    }
  }
  const rows: [number, number, number][] = [];
  for (let y = minY; y <= maxY; y++) {
    const lo = minX.get(y);
    if (lo !== undefined) rows.push([y, lo, maxX.get(y)! - lo + 1]);
  }
  return rows;
};
