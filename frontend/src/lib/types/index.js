/* eslint-disable */
// @ts-nocheck
export let builds;
export let cache;
export let calculator;
export let config;
export let exposition;
export let pob;
export let raw;

export const initializeCrystalline = () => {
  builds = {
    ParseBuild: globalThis['go']['go-pob']['builds']['ParseBuild'],
    ParseBuildStr: globalThis['go']['go-pob']['builds']['ParseBuildStr']
  };
  cache = {
    InitializeDiskCache: globalThis['go']['go-pob']['cache']['InitializeDiskCache']
  };
  calculator = {
    NewCalculator: globalThis['go']['go-pob']['calculator']['NewCalculator']
  };
  config = {
    InitLogging: globalThis['go']['go-pob']['config']['InitLogging']
  };
  exposition = {
    CalculateTreePath: globalThis['go']['go-pob']['exposition']['CalculateTreePath'],
    GetRawTree: globalThis['go']['go-pob']['exposition']['GetRawTree'],
    GetSkillGems: globalThis['go']['go-pob']['exposition']['GetSkillGems'],
    GetStatByIndex: globalThis['go']['go-pob']['exposition']['GetStatByIndex']
  };
  pob = {
    BuildInfo: globalThis['go']['go-pob']['pob']['BuildInfo'],
    CompressEncode: globalThis['go']['go-pob']['pob']['CompressEncode'],
    DecodeDecompress: globalThis['go']['go-pob']['pob']['DecodeDecompress']
  };
  raw = {
    InitializeAll: globalThis['go']['go-pob']['raw']['InitializeAll']
  };
};