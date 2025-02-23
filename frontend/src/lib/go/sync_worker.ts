import { expose, proxy } from 'comlink';
import '$lib/console_hook';
import '../../wasm_exec.js';
import { initializeCrystalline, storage, raw, config, pob, builds, calculator, exposition } from '../types';
import type { Outputs } from '../custom_types';
import type { currentBuild } from '../global.js';
import { dump, type ProxiedRemote } from '../type_utils';
import { reverseConfigOptions } from '../display/configurations';
import type { CalcDataColProp } from '$lib/calcs/calc_sections';
import { configure, fs } from '@zenfs/core';
import { IndexedDB } from '@zenfs/dom';

const storageConfigurationPromise = configure({
  mounts: {
    '/cache': {
      backend: IndexedDB,
      storeName: 'cache'
    },
    '/builds': {
      backend: IndexedDB,
      storeName: 'builds'
    }
  }
});

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
      if (this._currentBuild) {
        // Re-cast so we can force the correct type
        this.currentBuildStore.set(proxy(this._currentBuild) as unknown as ProxiedRemote<pob.PathOfBuilding>);
      } else {
        this.currentBuildStore.set(undefined);
      }
    }
  }

  boot(wasm: ArrayBuffer, callback: (out: Outputs) => void, currentBuildStore: typeof currentBuild) {
    this.callback = callback;
    this.booted = true;
    this.currentBuildStore = currentBuildStore;

    return new Promise((resolve) => {
      const go = new Go();
      void WebAssembly.instantiate(wasm, go.importObject).then(async (result) => {
        void go.run(result.instance);

        initializeCrystalline();

        config.InitLogging(false);

        await storageConfigurationPromise;

        await storage.InitializeStorage(
          // List
          async (bucket: string, dir: string) =>
            new Promise((res, rej) => {
              fs.readdir(`/${bucket}/${dir}`, (err, files) => {
                if (err) {
                  console.error(err);
                  return rej(err);
                }

                res(
                  files?.map((file) => {
                    const fullPath = `/${bucket}/${dir}/${file}`;
                    const stats = fs.statSync(fullPath);

                    let clazz = '';
                    let level = 0;

                    if (stats.isFile()) {
                      const buildFile = fs.readFileSync(fullPath);
                      if (buildFile) {
                        const [build] = builds.ParseBuildStr(buildFile.toString());
                        clazz = build?.Build?.ClassName || '';
                        level = build?.Build?.Level || 0;
                      }
                    }

                    return {
                      Name: file,
                      Type: stats.isDirectory() ? 'dir' : 'file',
                      Class: clazz,
                      LastEdit: stats.mtime.toISOString(),
                      Level: level
                    };
                  })
                );
              });
            }),
          // Get
          async (bucket: string, key: string) =>
            new Promise((res, rej) => {
              fs.readFile(`/${bucket}/${key}`, (err, file) => {
                if (err) {
                  console.error(err);
                  return rej(err);
                }

                if (file) {
                  return res(new Uint8Array(file));
                }

                return res(new Uint8Array(0));
              });
            }),
          // Write
          async (bucket: string, key: string, value: Uint8Array | undefined) => {
            if (!value || !key) {
              return;
            }

            return new Promise((res, rej) => {
              console.log('Writing to storage', `/${bucket}/${key}`);
              fs.writeFile(`/${bucket}/${key}`, value, (err) => {
                if (err) {
                  return rej(err);
                }
                res();
              });
            });
          },
          // Exists
          async (bucket: string, key: string) =>
            new Promise((res) => {
              fs.access(`/${bucket}/${key}`, (err) => {
                if (err) {
                  console.error(err);
                  return res(false);
                }

                return res(true);
              });
            }),
          // New Folder
          async (bucket: string, dir: string) =>
            new Promise((res) => {
              fs.mkdir(`/${bucket}/${dir}`, 0o777, (err) => {
                if (err) {
                  console.error(err);
                  return res();
                }

                return res();
              });
            }),
          // Delete
          async (bucket: string, path: string) =>
            new Promise((res) => {
              fs.unlink(`/${bucket}/${path}`, (err) => {
                if (err) {
                  console.error(err);
                  return res();
                }

                return res();
              });
            })
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

  ImportCode(code: string) {
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

  async Tick(source: string) {
    if (!this.currentBuild) {
      return;
    }

    const calc = calculator.NewCalculator(this.currentBuild);
    if (!calc) {
      return;
    }

    console.log('TICK from', source);
    const out = await calc.BuildOutput('MAIN');
    if (!out || !out.Player || !out.Player.MainSkill) {
      return;
    }

    if (this.callback) {
      this.callback({
        Output: out.Player.Output,
        OutputTable: out.Player.OutputTable,
        SkillFlags: out.Player.MainSkill.SkillFlags,
        Breakdown: dump(out.Player.Breakdown?.GetData() || {}),
        Calcs: dump(out.CalcProps || {})
      });

      if (out.DebugErrors?.length) {
        console.log('Errors from last tick:');
        console.log(out.DebugErrors);
      }
    }
  }

  SetConfigOption(key: string, value: boolean | number | string) {
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
      void this.Tick('SetConfigOption: remove');
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
    void this.Tick('SetConfigOption: change');
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

  GetAllConfigOptions(): Record<string, boolean | number | string> {
    if (!this.currentBuild || !this.currentBuild.Config.Inputs) {
      return {};
    }

    return this.currentBuild.Config.Inputs.reduce((out, input) => {
      let val: unknown;

      if (input.String !== undefined) {
        val = input.String;
      }

      if (input.Number !== undefined) {
        val = input.Number;
      }

      if (input.Boolean !== undefined) {
        val = input.Boolean;
      }

      return {
        ...out,
        [input.Name]: val
      };
    }, {});
  }

  SetMainSocketGroup(mainSocketGroup: number) {
    this.currentBuild?.SetMainSocketGroup(mainSocketGroup);
    void this.Tick('SetMainSocketGroup');
  }

  GetSkillGems(): ProxiedRemote<exposition.SkillGem[]> {
    return proxy(exposition.GetSkillGems()!) as unknown as ProxiedRemote<exposition.SkillGem[]>;
  }

  async GetTree(version: string): Promise<string> {
    const rawData = await exposition.GetRawTree(version);
    if (!rawData) {
      throw new Error('Failed loading tree');
    }
    return new TextDecoder().decode(rawData);
  }

  SetClass(value: string) {
    this.currentBuild?.SetClass(value);
    void this.Tick('SetClass');
  }

  SetAscendancy(value: string) {
    this.currentBuild?.SetAscendancy(value);
    void this.Tick('SetAscendancy');
  }

  SetLevel(value: number) {
    this.currentBuild?.SetLevel(value);
    void this.Tick('SetLevel');
  }

  AllocateNodes(nodeIds: number[]) {
    this.currentBuild?.AllocateNodes(nodeIds);
    void this.Tick('AllocateNode');
  }

  DeallocateNodes(nodeIds: number[]) {
    this.currentBuild?.DeallocateNodes(nodeIds);
    void this.Tick('DeallocateNodes');
  }

  CalculateAllocationPaths(version: string, activeNodes: number[], rootNodes: number[]) {
    return exposition.CalculateAllocationPaths(version, activeNodes, rootNodes);
  }

  CalculatePrunableNodes(version: string, activeNodes: number[], rootNodes: number[]) {
    return exposition.CalculatePrunableNodes(version, activeNodes, rootNodes);
  }

  BuildInfo() {
    return dump(pob.BuildInfo);
  }

  setCalcTabElements(elements: Record<string, CalcDataColProp>) {
    exposition.SetCalcTabElements({
      Elements: Object.fromEntries(
        Object.entries(elements).map(([k, v]) => [
          k,
          {
            Cfg: v.cfg!,
            ModSource: v.modSource,
            Enemy: v.enemy,
            ModName: typeof v.modName === 'string' ? [v.modName] : v.modName!,
            ModType: v.modType!
          } as calculator.ColProps
        ])
      )
    });
    void this.Tick('setCalcTabElements');
  }

  async ListBuilds(dir: string) {
    const [b] = await storage.ListBuilds(dir);
    return dump(b);
  }

  async NewFolder(name: string) {
    return await storage.NewFolder(name);
  }

  async Copy(from: string, to: string) {
    return await storage.GetBuild(from).then(([build]) => {
      if (!build) {
        return;
      }

      return storage.SetBuild(to, build);
    });
  }

  async Rename(from: string, to: string) {
    return await this.Copy(from, to).then(() => storage.DeleteBuild(from));
  }

  async Delete(path: string) {
    return await storage.DeleteBuild(path);
  }

  NewBuild() {
    this.currentBuild = builds.EmptyBuild();
  }

  async SaveBuildAs(path: string) {
    const [serialized, err] = builds.SerializeBuild(this.currentBuild);

    if (err) {
      console.error(dump(err));
      return;
    }

    if (!serialized) {
      console.error('failed serializing build');
      return;
    }

    const build = new TextDecoder().decode(serialized);

    return storage.SetBuild(path, build);
  }

  async OpenBuild(path: string) {
    const [xml, err] = await storage.GetBuild(path);

    if (err) {
      console.error(dump(err));
      return;
    }

    const [build, parseError] = builds.ParseBuildStr(xml);
    if (parseError) {
      throw parseError;
    }

    this.currentBuild = build;
  }

  ClearBuild() {
    this.currentBuild = undefined;
  }
}

expose(new PoBWorker());

export type WorkerType = PoBWorker;
