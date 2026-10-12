import { describe, expect, it } from 'vitest';
import { crc32, zipStore } from './zip';

const enc = new TextEncoder();

describe('zipStore', () => {
  it('CRC-32 is the zip one (check value of "123456789")', () => {
    expect(crc32(enc.encode('123456789'))).toBe(0xcbf43926);
    expect(crc32(new Uint8Array())).toBe(0);
  });

  it('writes each file stored, a central directory pointing at each, and the end record', () => {
    const files = [
      { path: 'a.xml', data: enc.encode('<A/>') },
      { path: 'Leagues/Ünïcode.xml', data: enc.encode('<B>two</B>') }
    ];
    const z = zipStore(files, new Date(2026, 9, 11, 12, 30, 10));
    const v = new DataView(z.buffer);
    const end = z.length - 22;
    expect(v.getUint32(end, true)).toBe(0x06054b50);
    expect(v.getUint16(end + 10, true)).toBe(2);
    let at = v.getUint32(end + 16, true);
    expect(at + v.getUint32(end + 12, true)).toBe(end);
    const read: [string, string][] = [];
    for (let i = 0; i < 2; i++) {
      expect(v.getUint32(at, true)).toBe(0x02014b50);
      const n = v.getUint16(at + 28, true);
      const name = new TextDecoder().decode(z.slice(at + 46, at + 46 + n));
      const off = v.getUint32(at + 42, true);
      expect(v.getUint32(off, true)).toBe(0x04034b50);
      expect(v.getUint16(off + 8, true), 'stored').toBe(0);
      const size = v.getUint32(off + 18, true);
      const data = z.slice(off + 30 + v.getUint16(off + 26, true), off + 30 + v.getUint16(off + 26, true) + size);
      expect(crc32(data)).toBe(v.getUint32(off + 14, true));
      read.push([name, new TextDecoder().decode(data)]);
      at += 46 + n;
    }
    expect(read).toEqual([
      ['a.xml', '<A/>'],
      ['Leagues/Ünïcode.xml', '<B>two</B>']
    ]);
    expect(v.getUint16(10, true)).toBe((12 << 11) | (30 << 5) | 5);
    expect(v.getUint16(12, true)).toBe((46 << 9) | (10 << 5) | 11);
  });
});
