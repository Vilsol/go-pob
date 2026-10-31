import { expose } from 'comlink';
import '$lib/console_hook';
import '../../wasm_exec.js';
import { boot, config, exposition } from '../types2';
import type { Poe2View } from '../global';

class PoB2Worker {
  private booted?: Promise<unknown>;

  boot(wasm: ArrayBuffer) {
    this.booted ??= boot(wasm).then(() => config.InitLogging(false));
    return this.booted;
  }

  async Open(xml: string): Promise<Poe2View> {
    await this.booted;
    const calc = exposition.ParseBuild(xml).unwrap();
    if (!calc) throw new Error('the PoE2 build did not parse');
    try {
      const out = (await calc.Calculate()).unwrap();
      if (!out) throw new Error('the PoE2 build did not calculate');
      return { className: out.ClassName, ascendClassName: out.AscendClassName, level: out.Level, output: { ...out.Output } };
    } finally {
      calc.release();
    }
  }
}

export const worker2Instance = new PoB2Worker();

expose(worker2Instance);

export type Worker2Type = PoB2Worker;
