import { expose, proxy } from 'comlink';
import '../../wasm_exec.js';
import { initializeCrystalline, cache, raw, config, pob, builds, calculator } from '../types';
import type { Outputs } from '../custom_types';
import localforage from 'localforage';
import type { currentBuild } from '../global';
import type { DeepPromise } from '../type_utils';
import { reverseConfigOptions } from '../display/configurations';

class PoBWorker {
  private _currentBuild?: pob.PathOfBuilding;

  get currentBuild(): pob.PathOfBuilding | undefined {
    return this._currentBuild;
  }

  set currentBuild(value: pob.PathOfBuilding | undefined) {
    this._currentBuild = value;
    this.updateStore();
  }

  booted = false;

  callback?: (out: Outputs) => void;
  currentBuildStore?: typeof currentBuild;

  private updateStore() {
    if (this.currentBuildStore) {
      // Re-cast so we can force the correct type
      this.currentBuildStore.set(proxy(this._currentBuild) as unknown as DeepPromise<pob.PathOfBuilding>);
    }
  }

  boot(wasm: ArrayBuffer, callback: (out: Outputs) => void, currentBuildStore: typeof currentBuild) {
    this.callback = callback;
    this.booted = true;
    this.currentBuildStore = currentBuildStore;

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

  async loadData(updates: (data: string) => Promise<void>) {
    const start = Date.now();
    const err = await raw.InitializeAll('3.18', updates);
    console.log('Initialization took', Date.now() - start, 'ms');
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

    console.log('TICK');
    const out = calc.BuildOutput('MAIN');
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

  async SetConfigOption(key: string, value: boolean | number | string) {
    if (!this.currentBuild || !this.currentBuild.Config.Inputs) {
      return;
    }

    let remove;
    const v = reverseConfigOptions[key];
    switch (v.type) {
      case 'list':
        remove = value === v.list[0].value;
        break;
      case 'check':
        if (v.defaultState !== undefined) {
          remove = value === v.defaultState;
        } else {
          remove = value === false;
        }
        break;
      default:
        remove = value === null;
        break;
    }

    if (remove) {
      this.currentBuild.RemoveConfigOption(key);
      this.updateStore();
      this.Tick();
      return;
    }

    const newValue: pob.Input = {
      Name: key
    };

    switch (typeof value) {
      case 'boolean':
        newValue.Boolean = value;
        break;
      case 'string':
        newValue.String = value;
        break;
      case 'number':
        newValue.Number = value;
        break;
    }

    this.currentBuild.SetConfigOption(newValue);
    this.updateStore();
    this.Tick();
  }

  GetConfigOption(name: string): boolean | number | string | undefined {
    if (!this.currentBuild || !this.currentBuild.Config.Inputs) {
      return undefined;
    }

    const input = this.currentBuild.Config.Inputs.find((i) => i.Name === name);
    if (!input) {
      return undefined;
    }

    if (input.String !== undefined) {
      return input.String;
    }

    if (input.Number !== undefined) {
      return input.Number;
    }

    if (input.Boolean !== undefined) {
      return input.Boolean;
    }

    return undefined;
  }
}

expose(new PoBWorker());

export type WorkerType = PoBWorker;
