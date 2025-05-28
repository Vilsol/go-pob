/* eslint-disable */
// @ts-nocheck
const wrap = (fn) => {
  return (...args) => {
    const result = fn.call(undefined, ...args);
    if (globalThis.goInternalError) {
      const error = new Error(globalThis.goInternalError);
      globalThis.goInternalError = undefined;
      throw error;
    }
    return result;
  }
};

export let builds;
export let calculator;
export let config;
export let exposition;
export let pob;
export let raw;
export let storage;

export const initializeCrystalline = () => {
  builds = {
    EmptyBuild: wrap(globalThis['go']['go-pob']['builds']['EmptyBuild']),
    ParseBuild: wrap(globalThis['go']['go-pob']['builds']['ParseBuild']),
    ParseBuildStr: wrap(globalThis['go']['go-pob']['builds']['ParseBuildStr']),
    SerializeBuild: wrap(globalThis['go']['go-pob']['builds']['SerializeBuild'])
  };
  calculator = {
    NewCalculator: wrap(globalThis['go']['go-pob']['calculator']['NewCalculator'])
  };
  config = {
    InitLogging: wrap(globalThis['go']['go-pob']['config']['InitLogging'])
  };
  exposition = {
    CalculateAllocationPaths: wrap(globalThis['go']['go-pob']['exposition']['CalculateAllocationPaths']),
    CalculatePrunableNodes: wrap(globalThis['go']['go-pob']['exposition']['CalculatePrunableNodes']),
    GetRawTree: wrap(globalThis['go']['go-pob']['exposition']['GetRawTree']),
    GetSkillGems: wrap(globalThis['go']['go-pob']['exposition']['GetSkillGems']),
    GetStatByIndex: wrap(globalThis['go']['go-pob']['exposition']['GetStatByIndex']),
    SetCalcTabElements: wrap(globalThis['go']['go-pob']['exposition']['SetCalcTabElements'])
  };
  pob = {
    BuildInfo: wrap(globalThis['go']['go-pob']['pob']['BuildInfo']),
    CompressEncode: wrap(globalThis['go']['go-pob']['pob']['CompressEncode']),
    DecodeDecompress: wrap(globalThis['go']['go-pob']['pob']['DecodeDecompress'])
  };
  raw = {
    InitializeAll: wrap(globalThis['go']['go-pob']['raw']['InitializeAll'])
  };
  storage = {
    DeleteBuild: wrap(globalThis['go']['go-pob']['storage']['DeleteBuild']),
    GetBuild: wrap(globalThis['go']['go-pob']['storage']['GetBuild']),
    InitializeStorage: wrap(globalThis['go']['go-pob']['storage']['InitializeStorage']),
    ListBuilds: wrap(globalThis['go']['go-pob']['storage']['ListBuilds']),
    NewFolder: wrap(globalThis['go']['go-pob']['storage']['NewFolder']),
    SetBuild: wrap(globalThis['go']['go-pob']['storage']['SetBuild'])
  };
};