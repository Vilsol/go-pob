import { expose } from "comlink";
import "../../wasm_exec.js";
import { initializeCrystalline, cache, raw, config, pob, builds, calculator } from "../types";
import type { Outputs } from "../custom_types";
import localforage from "localforage";

class PoBWorker {

  currentBuild?: pob.PathOfBuilding;
  callback?: (out: Outputs) => void;

  boot(wasm: ArrayBuffer, callback: (out: Outputs) => void) {
    this.callback = callback;

    return new Promise((resolve) => {
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      const go = new Go();
      WebAssembly.instantiate(wasm, go.importObject).then(async (result) => {
        go.run(result.instance);

        initializeCrystalline();

        config.InitLogging(false);

        await cache.InitializeDiskCache(
          async (key: string) => {
            const item = await localforage.getItem(key);
            if (item) {
              return item as Uint8Array;
            }
            return new Uint8Array(0);
          },
          async (key: string, value: Uint8Array | undefined) => {
            await localforage.setItem(key, value);
          },
          async (key: string) => (await localforage.getItem(key)) instanceof Uint8Array
        );

        resolve(undefined);
      });
    });
  }

  async loadData() {
    const err = await raw.InitializeAll("3.18");
    if (err) {
      console.error(err);
    }
  }

  async ImportCode(code: string) {
    const [xml, decodeError] = pob.DecodeDecompress(code);
    if (decodeError) {
      throw decodeError;
    }

    const [build, parseError] = builds.ParseBuildStr(xml);
    if (parseError) {
      throw parseError;
    }

    this.currentBuild = build;
  }

  async Tick() {
    if (!this.currentBuild) {
      return;
    }

    const calc = calculator.NewCalculator(this.currentBuild);
    if (!calc) {
      return;
    }

    const out = calc.BuildOutput("MAIN");
    if (!out || !out.Player || !out.Player.MainSkill) {
      return;
    }

    if (this.callback) {
      this.callback({
        Output: out.Player.Output,
        OutputTable: out.Player.OutputTable,
        SkillFlags: out.Player.MainSkill.SkillFlags
      });
    }
  }
}

expose(new PoBWorker());

export type WorkerType = PoBWorker;
